# EXIF-only Layout and Market Device Names

## Problem

The vault grew non-EXIF areas: `<year>/sources-manual/`, `<year>/processed/` and a root
`undated/`. They were an attempt to hold everything in one tree. In practice the vault is
only trustworthy for what can be rebuilt from EXIF; everything else (exports, screenshots,
undated clips) now lives outside it, in a hand-curated tree. The extra areas only add a
`sources/` level to every path and rules nobody validates.

Device directories also carry raw EXIF spellings: `Canon Canon EOS 5D (image)` (Canon repeats
the make inside the model), `SONY ILCE-6300` next to `Sony ILCE-6300`, `Unknown Canon EOS 550D
(video)` (a video with a model but no make), and codes like `DJI FC2103`.

## Goal

A vault holds only EXIF-filed media:

```
<vault>/
  2024/
    .imv/verify.cache
    Canon EOS 5D (image)/
      2024-08-20/
        2024-08-20_18-45-03_a1b2c3d4.jpg
```

- Root: year directories only.
- Year: device directories and `.imv/` only.
- Device names are canonical and use market names.
- Per-year verification with the cache stays exactly as it is.

## Non-Goals

- No config file for device names — the mapping is code, so a path stays a pure function of
  EXIF (and the vault stays rebuildable from EXIF alone).
- No attempt to date files without a date. They are not imported.

## Design

### Layout

`BuildSourcePath` drops the `sources/` segment: `<year>/<device>/<date>/<file>`.
`sources/`, `sources-manual/`, `processed/` and `undated/` are gone from the verifier's allowed
lists; anything but year dirs at the root, or device dirs and `.imv/` in a year, is
inconsistent. `ListSourceFiles` walks the year dir itself, skipping hidden dirs (`.imv/`).
`ListProcessedDirs` is removed. Cache keys stay relative to the year dir, so they simply lose
the `sources/` prefix.

### Import without a date

A file whose EXIF gives no date (zero time) is not imported: counted as `No date`, warned, and
left in place. Before, it was filed under year `0001`.

### Device names

`defaults.DeviceName(make, model)`:

1. Canonical make spelling from a case-insensitive alias table (`SONY` → `Sony`,
   `NIKON CORPORATION` → `Nikon`, …); unknown makes are kept as written.
2. No make (or `Unknown`) but the model starts with a known make → the make is taken from the
   model (`Canon EOS 550D` → make `Canon`).
3. A model that starts with its own make loses the repeat (`Canon` + `Canon EOS 5D` → `EOS 5D`).
4. Market names for cryptic model codes, keyed by canonical make + model (`Sony ILCE-6300` →
   `a6300`, `DJI FC2103` → `Mavic Air`).
5. Still no make → `Unknown`.

`DeviceDir` = `DeviceName(make, model) + " (" + type + ")"`. `NormalizeMake`/`NormalizeModel`
and their empty maps are replaced by this.

### Migration: `imv lib-tools migrate-layout`

One-off, for vaults in the old layout, no hashing and no exiftool:

1. Refuse a year that still has `sources-manual/` or `processed/`, and a root `undated/` —
   those must be moved out by hand first.
2. Lift `<year>/sources/<device>/` to `<year>/<device>/`.
3. Rename every device dir to its canonical name. The old name is split as first word = make,
   rest = model, then passed through `DeviceName` (old names were built from raw make + model,
   and all makes seen so far are one word). If the target exists, date dirs are merged; a file
   that already exists at the target is a conflict and stops the year.
4. Rewrite the year's cache keys the same way, so the next `verify` is still served from cache.

`--dry-run` prints the plan. A full `verify --no-cache` afterwards confirms EXIF agrees with every
new path.
