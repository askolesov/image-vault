# Image Vault Rewrite — Design Spec

## Overview

Complete rewrite of `image-vault` (imv) to support year-based sharding for fast validation of large (3TB+) photo libraries. Replaces the config-driven template system with a fixed, opinionated directory structure. Clean rewrite in the same repo (Approach A).

## Library Structure

The library is convention-based — no config files, no init command. The tool recognizes a library by its shape.

```
/my-library/
  2024/
    sources/
      Apple iPhone 15 Pro (image)/
        2024-08-20/
          2024-08-20_18-45-03_a1b2c3.jpg
          2024-08-20_18-45-03_a1b2c3.xmp
        2024-12-25/
          ...
      Apple iPhone 15 Pro (video)/
        2024-12-25/
          2024-12-25_10-00-15_d4e5f6.mp4
      Sony ILCE-7M3 (image)/
        ...
      Unknown (image)/
        ...
    processed/
      2024-08-20 Summer vacation/
        (freeform contents)
      2024-12-25 Christmas dinner/
        ...
  2025/
    sources/
      ...
    processed/
      ...
```

### Naming Rules

- **Year directories:** `YYYY`
- **Device directories:** `<Make> <Model> (<media-type>)` where media-type is `image`, `video`, or `audio` (matching MIME type prefixes)
- **Date directories:** `YYYY-MM-DD`
- **Source filenames:** `YYYY-MM-DD_HH-MM-SS_<hash>.<ext>`
- **Processed directories:** `YYYY-MM-DD <event name>` — strictly one space between date and name
- **Sidecars:** same base name as primary file, sit next to it
- **Unknown device:** files with no EXIF make/model go to `Unknown (<media-type>)/`
- **Media filter:** only photo, video, audio imported by default. Others dropped unless `--keep-all`.
- **Video separation:** videos are separated into their own device directory by default (e.g., `Apple iPhone 15 Pro (video)/`). Use `--no-separate-video` on import to merge videos into the same device dir as images.
- **Missing EXIF datetime:** falls back to zero time (1970-01-01) for determinism. The file is still imported, not dropped.
- **Missing EXIF make/model:** file goes to `Unknown (<media-type>)/`.

### Embedded Configuration

A `defaults.go` file centralizes all opinionated defaults in code:

- **Make/model normalization maps** — fix inconsistent EXIF values across devices (e.g., "Canon" vs "CANON"). Maps start empty, infrastructure ready.
- **Ignored OS files** — `.DS_Store`, `Thumbs.db`, `desktop.ini`, etc.
- **Sidecar extensions** — `.xmp`, `.yaml`, `.json`
- **Supported media types** — photo, video, audio MIME type prefixes
- **Hash algorithm** — MD5 by default, configurable via `--hash-algo` flag on import (e.g., `sha256`). Verify auto-detects hash format from filename length.

## CLI Interface

```
imv import <path> [flags]       # import photos into library
imv verify [flags]              # verify library integrity
imv version                     # display version info

imv tools remove-empty-dirs     # remove empty directories
imv tools scan <dir>            # recursive directory scan
imv tools diff <scan1> <scan2>  # compare two scans
imv tools info <file>           # show file metadata
```

### Global Behavior

- **Fail-fast by default** — stops on first error. Use `--no-fail-fast` to collect all errors.

## Package Architecture

```
cmd/
  imv/
    main.go

internal/
  defaults/
    defaults.go                 # all embedded config

  metadata/
    metadata.go                 # EXIF extraction, filesystem info, hashing
    metadata_test.go

  pathbuilder/
    pathbuilder.go              # deterministic path computation from metadata
    pathbuilder_test.go

  transfer/
    transfer.go                 # copy/move with paranoid hash verification
    transfer_test.go

  library/
    library.go                  # library detection, year enumeration,
                                # structure validation
    library_test.go

  importer/
    importer.go                 # orchestrates: metadata -> pathbuilder -> transfer
    importer_test.go

  verifier/
    verifier.go                 # integrity checks, --fix mode, year filtering
    verifier_test.go

  logging/
    logging.go                  # TTY-aware output
    logging_test.go

  command/
    root.go
    import.go
    verify.go
    version.go
    tools.go
    tools_remove_empty_dirs.go
    tools_scan.go
    tools_diff.go
    tools_info.go
```

### Design Principles

- `internal/` not `pkg/` — not a library for external consumption
- Each package has one clear responsibility
- `defaults` is the single source for all embedded configuration
- `pathbuilder` is the core — deterministic path from metadata, heavily tested
- `command/` is a thin CLI wiring layer, delegates everything to domain packages

## Import Flow

Per-file pipeline — no batching, keeps memory flat for 3TB+ libraries:

1. **Enumerate** source directory recursively, skip OS junk files (from `defaults`)
2. **Extract metadata** for each file: EXIF (make, model, datetime, mime type), filesystem info, hash
3. **Classify** media type from MIME: photo, video, audio, or other. Drop "other" unless `--keep-all`
4. **Normalize** make/model through normalization maps in `defaults`
5. **Build destination path** deterministically: `<library>/<year>/sources/<Make Model (type)>/<date>/<datetime_hash.ext>`
6. **Check destination:**
   - Doesn't exist: copy/move the file
   - Exists, same hash: skip (already imported)
   - Exists, different hash: **replace** destination (source is truth), log warning
7. **Handle sidecars:** find matching sidecar files (same base name, extensions from `defaults`), place next to primary file
8. **Report:** summary at end — imported, skipped, replaced, dropped, errors

### Import Flags

- `--move` — move instead of copy (default: copy)
- `--dry-run` — show what would happen
- `--keep-all` — don't drop non-media files
- `--year 2025` — only import files from this year (based on EXIF datetime)
- `--no-fail-fast` — collect all errors instead of stopping at first
- `--no-separate-video` — put videos in the same device dir as photos
- `--hash-algo <algo>` — hash algorithm (default: `md5`)

## Verify Flow

Per-file pipeline, same as import:

1. **Select scope:** if `--year 2025`, only scan `<library>/2025/`. Otherwise all years.
2. **Verify sources:** for each file in `sources/`:
   - Extract metadata, compute expected path
   - Compare actual path vs expected path — mismatch is an inconsistency
   - Re-hash file, confirm hash in filename matches actual content
   - With `--fix`: move file to correct location
3. **Verify processed:** for each directory in `processed/`:
   - Validate naming: must match `YYYY-MM-DD <event name>` (strict single space)
   - Validate year: directory date must match parent year
   - Contents are freeform — no validation inside
4. **Report:** summary — verified, inconsistent, fixed, errors

### Verify Flags

- `--year 2025` — scope to one year
- `--fix` — repair inconsistencies (move files to correct paths)
- `--no-fail-fast` — collect all errors

## Logging & Progress

### TTY detected (interactive terminal)

- Interactive progress bar: file count, percentage, current file
- Warnings printed inline above the progress bar — never buried
- Summary at end with totals

### Non-TTY (piped, redirected)

- Periodic progress lines to stderr every 10 seconds: `[progress] 4,521/12,340 (36%)`
- Warnings/errors to stderr with prefixes: `[warn]`, `[error]`
- Summary to stdout at end

### Key Principle

Warnings and errors are always visible — never lost in progress noise. Progress goes to stderr, actionable output goes to stdout.

## Testing Strategy

Target: near 100% coverage.

- **`pathbuilder`** — table-driven tests. Every combination of: normal EXIF, missing make/model, missing datetime, photo/video/audio, normalization maps, unknown device. Most critical package.
- **`metadata`** — test with real sample files in `testdata/`. EXIF extraction, hash computation, missing EXIF.
- **`transfer`** — all scenarios: new file, identical exists, different content exists (replace), sidecars, dry-run, move vs copy, permission errors.
- **`library`** — structure detection, year enumeration, processed dir name validation.
- **`importer`** / **`verifier`** — integration tests using temp directories with known structures. Per-file pipeline, fail-fast vs collect-all, year filtering.
- **`logging`** — TTY vs non-TTY output, warning collection, summary formatting.
- **`defaults`** — normalization maps produce expected results.
- **`command`** — thin layer, test flag parsing and wiring only.

A `testdata/` directory at repo root with small sample files (real JPEGs, ARWs, MP4s with known EXIF).

## What Gets Dropped

- `image-vault.yaml` config file
- `init` command
- Template system (`vault/template.go`, Sprig dependency)
- `.gitignore`-style filter system (`vault/filter.go`, `go-gitignore` dependency)
- `go-pretty` table formatting
- `pkg/` directory structure
- `samber/lo` utility dependency

## What Gets Kept/Ported

- `go-exiftool` integration
- Paranoid hash-verify-on-destination logic from `transfer.go`
- `cobra` for CLI
- Docker + CI/CD setup (updated for new structure)
- `scan` and `diff` logic (moved under `tools`)
