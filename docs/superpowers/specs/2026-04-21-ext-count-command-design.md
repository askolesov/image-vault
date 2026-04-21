# `tools ext-count` — Extension Counter with Tree Output

**Date:** 2026-04-21
**Status:** Approved, ready for planning
**Scope:** New `imv tools ext-count <dir>` subcommand that walks a directory and prints a tree of per-directory file-extension counts, rolled up recursively. New package `internal/extcounter` for the walk/aggregate/render logic; new command file `internal/command/tools_ext_count.go` wiring it to cobra and registering it under the existing `tools` group.

## Motivation

A user inspecting a photo library (or any unfamiliar directory tree) wants a quick answer to "what kinds of files are in here, and how are they distributed across the tree?" — e.g., confirming that RAW files only live under `sources/`, spotting stray `.mov` files among images, or verifying sidecar coverage. Existing tools (`tools scan`, `tools info`) either produce JSON manifests or summarize a single directory; neither gives a compact, navigable tree view of extension distribution.

This command fills that gap with a read-only, fast (`stat`-only, no hashing) inspection tool.

## Goals

- One command, one directory argument, human-readable tree output on stdout.
- Counts at every visible node are **recursive rollups** — every file beneath that node, regardless of depth.
- `--depth` controls **tree drawing only**, never what gets counted. A deeper file never silently vanishes from the totals.
- Fast on large libraries (no file reads, no exif, no hashing).
- Clean enough package boundaries that adding `--json` later is trivial.

## Non-goals

- JSON / machine-readable output in v1.
- Filtering by extension, size, or mtime.
- Counting bytes, only files.
- Following symlinks specially — we use whatever `filepath.WalkDir` presents.
- Progress output — the walk is fast enough that it's unnecessary.
- Parallelism — directory walking is I/O-bound but a single walker is fine for the scale we target.

## Command surface

```
imv tools ext-count <dir> [flags]
```

Positional argument is required. Exactly 1 directory.

| Flag | Default | Meaning |
|---|---|---|
| `-d`, `--depth N` | `1` | Max tree depth to draw. `0` = root only. Counts always roll up fully regardless of depth. |
| `--include-hidden` | `false` | Include dotfiles and dot-prefixed directories (e.g. `.DS_Store`, `.config/`). |

Negative `--depth` is rejected by cobra validation.

Registered in `internal/command/tools.go` alongside `scan`, `diff`, `info`, `remove-empty-dirs`.

## Output

- Goes to **stdout**. Warnings (unreadable subdirs) go to **stderr**.
- Line shape per directory:
  - Root: `[<label>] — ext1: N, ext2: N, …`
  - Descendants: `<connectors>[<dirname>] — ext1: N, ext2: N, …`
- Root label is whatever the user passed in (after `filepath.Clean`) — `~/Photos`, `.`, `../foo`. No abs conversion.
- Sibling directories sorted alphabetically by basename.
- Extensions within a line sorted by **count desc**, ties broken alphabetically.
- Empty directories (no files in their subtree) render as `[dirname] — (empty)`.
- Files with no extension go into bucket `(none)`. `(none)` sorts by count like any other ext.

### Example

`imv tools ext-count ~/Photos --depth 2`:

```
[~/Photos] — jpg: 1502, xmp: 400, mp4: 23, mov: 12
├── [2024] — jpg: 802, xmp: 200, mp4: 15
│   ├── [processed] — jpg: 22
│   └── [sources] — jpg: 780, xmp: 200, mp4: 15
└── [2025] — jpg: 700, xmp: 200, mp4: 8, mov: 12
    ├── [processed] — jpg: 10
    └── [sources] — jpg: 690, xmp: 200, mp4: 8, mov: 12
```

Depth semantics:
- `--depth 0` → one line, just the root with its full rollup.
- `--depth 1` (default) → root + immediate children.
- `--depth N` → tree to N levels deep, counts at the deepest shown level still include everything below.

## Design

### Package layout

```
internal/extcounter/
  extcounter.go       # Walk + Aggregate
  render.go           # Tree rendering
  extcounter_test.go
  render_test.go
internal/command/
  tools_ext_count.go  # Cobra wiring, flag parsing, stdout/stderr glue
  tools.go            # Add newToolsExtCountCmd() to AddCommand(...)
```

### Core types

```go
package extcounter

type DirNode struct {
    Name     string         // basename; for root, the user-supplied label
    Children []*DirNode     // sorted alphabetically
    Direct   map[string]int // ext -> count of files directly in this dir
    Total    map[string]int // ext -> count including all descendants (populated by Aggregate)
}

type Options struct {
    IncludeHidden bool
}
```

### Public API

```go
// Walk builds the tree with Direct counts filled in. Total is nil until Aggregate runs.
// Errors reading subdirectories are written to stderrW and skipped; only failures on the
// root itself are returned as an error.
func Walk(root string, opts Options, stderrW io.Writer) (*DirNode, error)

// Aggregate walks the tree bottom-up and fills Total on every node.
func Aggregate(n *DirNode)

// Render prints the tree to w, drawing at most depth levels below the root
// (depth 0 = root only, depth 1 = root + immediate children, etc.).
func Render(root *DirNode, depth int, w io.Writer)
```

### Flow inside the cobra `RunE`

1. Parse flags (`depth`, `includeHidden`).
2. `tree, err := extcounter.Walk(dir, Options{IncludeHidden: includeHidden}, os.Stderr)`; return err if non-nil.
3. `extcounter.Aggregate(tree)`.
4. `extcounter.Render(tree, depth, os.Stdout)`.
5. Return nil.

### Walk details

- Uses `filepath.WalkDir` — lazy, no extra `stat` beyond what the dirent provides.
- For each dirent:
  - If basename starts with `.` and `opts.IncludeHidden` is false, skip. For directories, return `filepath.SkipDir`.
  - If `d.IsDir()`, create or locate the corresponding `DirNode`.
  - Else, compute extension and increment `parent.Direct[ext]`.
- No special symlink handling: `d.IsDir()` decides dir-vs-file; whatever the kernel says, we take.
- Unreadable subdirectory during walk: `filepath.WalkDir` invokes the walk fn with a non-nil err. We log `warning: cannot read <path>: <err>` to stderrW, return `filepath.SkipDir` (or `nil` for file-level errors), continue.

### Extension extraction

```go
ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
if ext == "" {
    ext = "(none)"
}
```

- `.JPG` and `.jpg` collapse to `jpg`.
- `jpg` and `jpeg` remain distinct (no extension aliasing).
- `.DS_Store` under `--include-hidden`: `filepath.Ext` returns `.DS_Store`; we special-case leading-dot-no-inner-dot filenames to bucket `(none)` instead of `ds_store`. Rule: if the basename starts with `.` and contains no other `.`, the extension is `(none)`.
- Files like `archive.tar.gz` yield `gz`, consistent with `filepath.Ext`.

### Aggregate

Post-order traversal. For each node:
```go
n.Total = make(map[string]int, len(n.Direct))
for k, v := range n.Direct { n.Total[k] = v }
for _, c := range n.Children {
    Aggregate(c)
    for k, v := range c.Total { n.Total[k] += v }
}
```

### Render

Standard tree-drawing algorithm:
- Root line: `[<name>] — <ext-list>` with no connector.
- For each child:
  - If last sibling: prefix `└── `; children's vertical bar uses `    ` (4 spaces).
  - Else: prefix `├── `; children's vertical bar uses `│   `.
- `ext-list` formatting: sort `Total` by count desc (alpha tiebreak), format as `k: v` joined by `, `. If empty map, emit `(empty)`.
- Recurse up to `depth` levels below the root (root is level 0; its children are level 1; etc.).

### Children sort

Sibling `DirNode`s are sorted alphabetically by `Name` once per parent. Done either in `Walk` (when the node is finalized) or lazily in `Render`; design doesn't mandate which, as long as the output is deterministic.

## Error handling

| Condition | Behavior |
|---|---|
| Root path missing / not a directory / permission denied on root | Return error from `Walk`; command exits non-zero; nothing on stdout |
| Unreadable subdirectory during walk | Warning to stderr: `warning: cannot read <path>: <err>`; skip subtree; continue; exit 0 |
| `--depth` negative | Rejected by cobra flag validation |
| Zero files found (legitimately empty tree) | Root renders as `[<label>] — (empty)`, exit 0 |

## Testing strategy

Unit tests build a fixture tree per test with `t.TempDir()` + `os.MkdirAll` / `os.WriteFile` with zero-byte contents. No golden files — assert on the tree shape or on a rendered string via `bytes.Buffer`.

### `extcounter_test.go`

| Test | What it proves |
|---|---|
| `Walk_FlatDir_MixedCase` | `.JPG`, `.jpg`, `.Jpeg`, `.jpeg` — `jpg` and `jpeg` are distinct buckets, both lowercase |
| `Walk_Nested_RollupMatchesSumOfDirect` | After `Aggregate`, each node's `Total` equals sum of its subtree's `Direct` |
| `Walk_NoExtFile` | `README` → `(none)` |
| `Walk_HiddenSkippedByDefault` | `.DS_Store`, `.hidden_dir/` not counted |
| `Walk_IncludeHidden` | With flag on, `.DS_Store` counted as `(none)`; `.hidden_dir/` descended |
| `Walk_EmptyDir` | Directory with no files in subtree produces a node with empty `Direct`/`Total` |
| `Walk_UnreadableSubdir` | Chmod 0 on a subdir: warning written to captured stderr buffer, walk completes, parent counts reflect only readable siblings, no error returned |
| `Walk_RootMissing` | Returns non-nil error |
| `Aggregate_Idempotent` | Calling `Aggregate` twice yields the same `Total` |

### `render_test.go`

| Test | What it proves |
|---|---|
| `Render_Depth0` | Only the root line is emitted |
| `Render_Depth1_Default` | Root + immediate children, no grandchildren |
| `Render_DepthDeeperThanTree` | No crash, renders full actual tree |
| `Render_ConnectorsLastVsMiddle` | `├── ` for middle siblings, `└── ` for last; vertical bars `│   ` vs. `    ` line up correctly |
| `Render_ExtOrderCountDescAlphaTiebreak` | Given `{a:1, b:1, c:3}` we get `c: 3, a: 1, b: 1` |
| `Render_EmptyDir` | `[dirname] — (empty)` |
| `Render_SiblingsAlphabetical` | Children listed in alphabetical order regardless of creation order |

No integration test — the cobra wiring is a thin translation layer and is exercised indirectly when manually running the binary. If an integration is desired later, it fits the pattern of `tools_scan` + a tmpdir.

## Open questions

None. All semantic questions resolved during brainstorming.

## Future extensions (out of scope for this change)

- `--json` / `-o <file>` output mode — the `DirNode` tree already serializes cleanly; a second `Render` function would do it.
- Filter flags (`--ext jpg,xmp`, `--only-non-zero`) — trivial on top of `Total`.
- Byte totals in addition to file counts — requires lstat'ing each dirent (cheap but slightly slower).
