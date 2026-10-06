# Image Vault

[![Lint](https://github.com/askolesov/image-vault/actions/workflows/lint.yaml/badge.svg)](https://github.com/askolesov/image-vault/actions/workflows/lint.yaml)
[![Test](https://github.com/askolesov/image-vault/actions/workflows/test.yaml/badge.svg)](https://github.com/askolesov/image-vault/actions/workflows/test.yaml)
[![Version](https://img.shields.io/github/v/release/askolesov/image-vault?include_prereleases)](https://github.com/askolesov/image-vault/releases)

CLI tool for organizing photo libraries by convention. Deterministic paths from EXIF metadata + content hash — same file always lands in the same place, no duplicates.

## Quick Start

```bash
cd ~/Photos
imv import /path/to/photos
imv verify --fix
```

## Install

```bash
go install github.com/askolesov/image-vault/cmd/imv@latest
```

Requires `exiftool`:

```bash
brew install exiftool        # macOS
sudo apt install libimage-exiftool-perl  # Linux
```

## Library Structure

No config files — the library is defined by its directory layout, and holds only
what can be filed from EXIF. Everything else (exports, screenshots, files with no
date) belongs outside the vault.

```
~/Photos/
  2024/
    .imv/verify.cache
    Apple iPhone 15 Pro (image)/
      2024-08-20/
        2024-08-20_18-45-03_a1b2c3d4.jpg
        2024-08-20_18-45-03_a1b2c3d4.xmp   # sidecar
    Apple iPhone 15 Pro (video)/
      2024-12-25/
        2024-12-25_10-00-15_e5f6g7h8.mp4
  2025/
    ...
```

Naming conventions:

- **Year dirs** — `YYYY`
- **Device dirs** — `<Make> <Model> (<type>)` where type is `image`, `video`, or `audio`;
  the name is canonical (see below)
- **Date dirs** — `YYYY-MM-DD`
- **Filenames** — `YYYY-MM-DD_HH-MM-SS_<hash>.<ext>`
- **Sidecars** (`.xmp`, `.yaml`, `.json`) — placed next to their primary file
- **Extensions** — always lowercase

The root holds only year dirs; a year holds only device dirs and `.imv/`. Anything else
is reported by `verify`.

Device names are canonical and a pure function of EXIF make + model: the make's spelling
is normalized (`SONY` → `Sony`), a make repeated in the model is dropped
(`Canon Canon EOS 5D` → `Canon EOS 5D`), a missing make is taken from the model
(`Canon EOS 550D` video with no Make), and model codes get market names
(`ILCE-6300` → `a6300`, `FC2103` → `Mavic Air`). The tables live in
`internal/defaults/defaults.go`. Files with no make at all go to `Unknown (<type>)/`.
Videos get separate device dirs by default.

Files with no EXIF date are not imported — they are counted as `No date` and left in place.

## Commands

### import

```bash
imv import <source-path> [flags]
```

| Flag | Description |
|------|-------------|
| `--move` | Move files instead of copying |
| `--dry-run` | Show what would be done |
| `--keep-all` | Keep non-media files (dropped by default) |
| `--year YYYY` | Only import files from this year |
| `--no-fail-fast` | Continue on errors |
| `--no-separate-video` | Put videos in same device dir as photos |
| `--no-verify` | Skip hash verification of existing files |
| `--no-randomize` | Import in directory order |
| `--hash-algo` | `md5` (default) or `sha256` |

Files with no EXIF date are skipped (`No date` in the summary) and stay where they are.

### verify

```bash
imv verify [flags]
```

| Flag | Description |
|------|-------------|
| `--fix` | Move misplaced files to correct location |
| `--fast` | Validate filenames/structure only, skip hashing |
| `--year YYYY` | Only verify files from this year |
| `--no-fail-fast` | Continue on errors |
| `--no-randomize` | Verify in directory order |
| `--no-cache` | Skip the per-year verification cache (don't read or write it) |
| `--hash-algo` | `md5` (default) or `sha256` |

Verify keeps a per-year cache at `<year>/.imv/verify.cache` so repeated runs can skip files whose size and mtime are unchanged since the last successful verification. The cache survives crashes and power loss (appends are fsynced every 10 s; compaction is atomic). It is invalidated when files are copied without preserving mtime — use `rsync -a` or `cp -p` when migrating a library. To force a full re-verify of a year, delete its cache file or pass `--no-cache`.

### tools

```bash
imv tools info <file>               # Show file metadata as JSON
imv tools scan <dir> -o scan.json   # Produce directory manifest
imv tools diff a.json b.json        # Compare two manifests
imv tools remove-empty-dirs         # Clean up empty directories
imv tools ext-count <dir>           # Tree of per-dir file extension counts
```

### lib-tools

Library-aware maintenance commands. Run from the library root.

```bash
imv lib-tools normalize-ext         # Lowercase extensions inside device dirs
imv lib-tools migrate-layout        # One-off: old sources/ layout → current layout
```

#### normalize-ext


| Flag | Description |
|------|-------------|
| `--year YYYY` | Only normalize files from this year |
| `--dry-run` | Show what would be renamed without modifying files |
| `--no-fail-fast` | Continue on errors |

Walks `<year>/<device>/` subtrees only. Skips hidden dirs (`.imv/`) and loose
files outside device dirs. Pure case fixup — no hashing, no exiftool. After
running, the next `verify` will re-verify renamed files once (cache miss) and
re-cache them.

#### migrate-layout

| Flag | Description |
|------|-------------|
| `--year YYYY` | Only migrate this year |
| `--dry-run` | Print the plan without changing anything |

For vaults in the old layout (`<year>/sources/…`, `sources-manual/`, `processed/`,
root `undated/`). Lifts device dirs out of `sources/`, renames them to canonical
names (merging dirs that map to the same name; a file already present at the
target is a conflict and stops that year), and rewrites each year's verify cache
so the next `verify` is still served from cache. Refuses to run while
`sources-manual/`, `processed/` or `undated/` exist — move them out of the vault
first. No hashing, no exiftool; run `imv verify --no-cache` afterwards to confirm
EXIF agrees with every new path.

### version

```bash
imv version
```

## Building from Source

```bash
git clone https://github.com/askolesov/image-vault.git
cd image-vault
make build     # binary in build/imv
make install   # install to $GOPATH/bin
make test      # run tests
make lint      # run linter
```
