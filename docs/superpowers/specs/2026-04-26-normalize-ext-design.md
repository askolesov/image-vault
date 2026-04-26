# Lowercase Extension Normalization

## Problem

File extensions written into the library are inconsistently cased.

- **Primary files** are case-normalized to lowercase. `metadata.BuildFileMetadata`
  (`internal/metadata/metadata.go:131`) does
  `strings.ToLower(filepath.Ext(path))` and that value flows through the
  importer into the built path.
- **Sidecar files** are NOT case-normalized. `importer.importFile`
  (`internal/importer/importer.go:175`) does
  `sidecarExt := filepath.Ext(sidecar)` (raw) and passes it to
  `pathbuilder.BuildSidecarPath`, which preserves the input case. So
  `IMG.JPG` lands at `..._<hash>.jpg` while its companion `IMG.XMP` lands
  at `..._<hash>.XMP`.
- **`verify`** detects neither case mismatch directly. The filename regex
  (`pathbuilder.go:90`) uses `\.\w+`, which accepts uppercase. In full
  mode with `--fix`, uppercase primaries are caught indirectly via the
  path-rebuild flow (rebuilt expected path is lowercase → string
  mismatch → `TransferFile` rename). Sidecars in `verifySourceFiles`
  are skipped entirely (line 239) and are never inspected for case.

This produces mixed-case extensions in the library that are cosmetic on
case-insensitive filesystems (macOS APFS) but become real divergence on
case-sensitive filesystems (Linux ext4) and create import-time and
cross-mount collision hazards.

## Goal

Make every extension written into the controlled `sources/` area
lowercase, and provide a fast tool to repair libraries that already
have uppercase extensions.

## Non-Goals

- **No alias detection** (jpg/jpeg, tif/tiff, htm/html, mpeg/mpg, etc.).
  These are different extensions semantically; renaming silently is
  wrong. A future spec can address alias detection / warning.
- **No automatic case-fixup of freeform dirs** (`sources-manual/`,
  `processed/`, `undated/`). The new tool deliberately scopes to
  `sources/` only.
- **No migration of existing `tools` subcommands** into `lib-tools`.
  Adding `lib-tools` as a new umbrella; existing `tools` stays.
- **No new flag on `verify`.** Detection is added in-place; no command
  surface change.

## Design

### Convention

All file extensions inside `<year>/sources/...` are lowercase. This
applies to primaries (`.jpg`, `.mp4`, `.arw`, …) and sidecars (`.xmp`,
`.yaml`, `.json`).

### Behavior changes

| Command | Behavior |
|---|---|
| `import` | Lowercases sidecar extensions at write time (primaries already lowercased). |
| `verify` (fast or full) | Counts any non-lowercase extension under `sources/` as `Inconsistent` — both primaries and sidecars. |
| `verify --fix` (full mode) | Unchanged. Uppercase primaries continue to be moved via the existing path-mismatch flow (rebuilt path is lowercase → string mismatch → `TransferFile` rename). Sidecars are not auto-fixed by `verify`; the warning points users to `lib-tools normalize-ext`. |
| `lib-tools normalize-ext` | NEW. Walks `<year>/sources/` per year and renames any file whose extension differs from its lowercase form. Pure case fixup — no hashing, no exiftool. |

### Implementation

#### 1. Importer — close the leak

`internal/importer/importer.go:175`:

```go
sidecarExt := strings.ToLower(filepath.Ext(sidecar))
```

#### 2. Pathbuilder — defensive contract

`internal/pathbuilder/pathbuilder.go::BuildSidecarPath` lowercases the
input as part of the contract, so future callers cannot reintroduce the
bug:

```go
func BuildSidecarPath(primaryPath string, sidecarExt string) string {
    sidecarExt = strings.ToLower(sidecarExt)
    ext := filepath.Ext(primaryPath)
    return primaryPath[:len(primaryPath)-len(ext)] + sidecarExt
}
```

#### 3. Verifier — symmetric case detection

`internal/verifier/verifier.go::verifySourceFiles` — inserted just after
`isSkippableInLibrary` and before the sidecar-skip:

```go
ext := filepath.Ext(baseName)
if ext != strings.ToLower(ext) {
    result.Inconsistent++
    v.logger.Warn("non-lowercase extension: %s (run 'imv lib-tools normalize-ext' to fix)", filePath)
    if v.cfg.FailFast {
        return fmt.Errorf("non-lowercase extension in %s", filePath)
    }
    continue
}
if defaults.IsSidecarExtension(ext) {
    continue
}
```

This applies in both fast and full mode. Detection is symmetric across
primaries and sidecars; fix-up is split (full+fix moves uppercase
primaries via the existing path-rebuild flow; sidecars require
`lib-tools normalize-ext`).

#### 4. New package: `internal/extnormalize`

Walks `<library>/<year>/sources/` and renames any file whose extension
is not already lowercase. Uses existing `library.ListYearsFiltered` and
`library.ListSourceFiles`.

```go
package extnormalize

type Options struct {
    LibraryPath string
    YearFilter  string  // "" = all years
    DryRun      bool
    FailFast    bool
}

type Result struct {
    Renamed   int  // ext was non-lowercase, rename succeeded (or would in dry-run)
    AlreadyOK int  // already lowercase
    Conflicts int  // would clobber a different file at lowercase target
    Errors    int
}

func Run(opts Options, logger *logging.Logger) (*Result, error)
```

Per-file algorithm:

```
ext   := filepath.Ext(name)
lower := strings.ToLower(ext)
if ext == lower → AlreadyOK; continue
src   := <abs path>
dst   := dir/<name with lowercase ext>
srcInfo := os.Lstat(src)
dstInfo, err := os.Lstat(dst)
case err is os.IsNotExist:
    safe rename → os.Rename(src, dst); Renamed++
case err == nil and os.SameFile(srcInfo, dstInfo):
    case-insensitive FS, same inode → os.Rename(src, dst); Renamed++
case err == nil and different file:
    Conflicts++; warn; FailFast or continue
case other err:
    Errors++; warn; FailFast or continue
```

The `os.SameFile` path is what makes case-only renames safe on macOS
APFS / Windows: there `.JPG` and `.jpg` resolve to the same inode, but
`os.Rename` correctly updates the directory-entry case.

OS junk filtering uses `defaults.IsIgnoredFile`.

#### 5. New umbrella: `imv lib-tools`

`internal/command/lib_tools.go`:

```go
func newLibToolsCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "lib-tools",
        Short: "Library-aware maintenance commands",
    }
    cmd.AddCommand(newLibToolsNormalizeExtCmd())
    return cmd
}
```

Wired into `internal/command/root.go` alongside existing top-level
commands.

#### 6. New subcommand: `imv lib-tools normalize-ext`

`internal/command/lib_tools_normalize_ext.go` — cobra wrapper that:

- Takes no positional args. Library root is CWD (matches `verify`).
- Flags: `--year YYYY`, `--dry-run`, `--no-fail-fast`.
- Calls `extnormalize.Run`, prints summary counts via
  `logger.PrintSummary`.

### Cache impact (informational)

The verify cache (`internal/verifier/cache.go`) keys on `RelToYear`
byte-exactly, so it is not case-aware. After `normalize-ext` renames
`..._abcd.JPG` → `..._abcd.jpg`:

- Next verify run loads cache (still keyed `...JPG`).
- `openYearCache` (verifier.go:184-197) intersection drops the orphan
  key (no file at `...JPG` on disk).
- New file at `...jpg` has no cache entry → cache miss → full
  re-verify of that file → records new entry under `...jpg`.
- Compaction (already part of `openYearCache`) writes the trimmed map.

So renamed files pay a one-time cache-miss cost; the cache self-heals
on the next run. No special invalidation logic in `normalize-ext`.

### Documentation

`README.md`:

- Add `lib-tools` section listing `normalize-ext` with a one-line
  description and flag table.
- Mention in the structure / convention text that all extensions in
  `sources/` are lowercase.

## Testing

### `internal/extnormalize` (new tests)

1. File with uppercase extension and no conflict → renamed; `Renamed=1`.
2. File with already-lowercase extension → no-op; `AlreadyOK=1`.
3. Two files on a case-sensitive FS at uppercase + lowercase variants
   with different content → `Conflicts=1`, no rename. (Use
   `t.Skip` if filesystem reports as case-insensitive — check by
   creating two distinct casings and comparing.)
4. `--dry-run`: file would be renamed but is not; `Renamed=1`,
   on-disk file unchanged.
5. `--year YYYY`: only that year is walked; other years are untouched.
6. Multiple year dirs each with uppercase exts → all years processed in
   sorted order.
7. OS junk file (e.g., `.DS_Store` with uppercase variant) → ignored
   via `defaults.IsIgnoredFile`.
8. `sources-manual/`, `processed/`, `undated/` siblings of `sources/`
   contain uppercase exts → untouched.
9. Sidecar `.XMP` next to primary `.jpg` → both end up lowercase
   (sidecar renamed; primary already OK).

### `internal/pathbuilder` (existing test file)

10. `BuildSidecarPath("…/foo.jpg", ".XMP")` → ends in `.xmp`.

### `internal/importer/importer_test.go`

11. Importing a file with sidecar `.XMP` produces a target with
    extension `.xmp`.

### `internal/verifier/verifier_test.go`

12. Fast mode + uppercase primary in `sources/` → `Inconsistent=1`.
13. Fast mode + uppercase sidecar in `sources/` → `Inconsistent=1`.
14. Full mode + uppercase sidecar (no `--fix`) → `Inconsistent=1`,
    no rename.
15. Full mode + `--fix` + uppercase primary → still moved via existing
    path-mismatch flow (regression check; behavior unchanged).
16. Full mode + `--fix` + uppercase sidecar → `Inconsistent=1`, NOT
    renamed (verify never moves sidecars).

## Risks

- **Case-insensitive FS rename semantics.** On macOS APFS / Windows,
  `os.Rename` of `.JPG` → `.jpg` is a directory-entry-case update and
  works atomically. The `os.SameFile` check disambiguates this case
  from a real conflict on case-sensitive FS. Tested in (1)–(3) above.
- **Cache miss after normalize.** One-time cost per renamed file on
  the next verify run. Acceptable; documented above.
- **User running `normalize-ext` on a partially-imported library.**
  Tool only renames files whose extension is non-lowercase; it does
  not modify content, hashes, or paths. Safe to rerun.
- **Future migration of `tools` subcommands.** `lib-tools` and `tools`
  coexist for now. If we later move library-aware ones into
  `lib-tools`, that's a separate spec.
