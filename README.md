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

No config files — the library is defined by its directory layout:

```
~/Photos/
  2024/
    sources/
      Apple iPhone 15 Pro (image)/
        2024-08-20/
          2024-08-20_18-45-03_a1b2c3d4.jpg
          2024-08-20_18-45-03_a1b2c3d4.xmp   # sidecar
      Apple iPhone 15 Pro (video)/
        2024-12-25/
          2024-12-25_10-00-15_e5f6g7h8.mp4
    sources-manual/   # freeform, not validated
    processed/        # freeform, not validated
  2025/
    ...
```

Naming conventions:

- **Year dirs** — `YYYY`
- **Device dirs** — `<Make> <Model> (<type>)` where type is `image`, `video`, or `audio`
- **Date dirs** — `YYYY-MM-DD`
- **Filenames** — `YYYY-MM-DD_HH-MM-SS_<hash>.<ext>`
- **Sidecars** (`.xmp`, `.yaml`, `.json`) — placed next to their primary file

Files with no EXIF make/model go to `Unknown (<type>)/`. Videos get separate device dirs by default.

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
```

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
