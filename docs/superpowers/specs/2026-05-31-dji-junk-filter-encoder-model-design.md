# DJI Junk Filter and Encoder Camera-Model Fallback

## Problem

DJI Osmo Pocket exports produce two failure modes for `imv import`:

1. **Junk files contaminate the vault.** Each captured clip ships with
   auxiliary files the camera writes alongside the real media:
   - `.thm` — DJI thumbnail
   - `.lrf` — low-resolution preview track
   - `.scr` — screen/aux data

   `enumerateFiles` (`internal/importer/importer.go:187`) skips OS junk
   via `defaults.IsIgnoredFile` but accepts every other file. These DJI
   auxiliaries have no usable EXIF, so they land in
   `0001/sources/Unknown (image)/0001-01-01/` (no date) or
   `<year>/sources/Unknown (video)/<date>/` (date from filesystem
   mtime, no model). A live vault sample at `/Volumes/T5/vault`
   contained 66 such files in `0001/` and dozens more mixed into
   `2000/` and `2026/`.

2. **DJI Osmo Pocket MP4s lack `Make`/`Model` EXIF tags.** The current
   model fallback chain in `metadata.BuildFileMetadata`
   (`internal/metadata/metadata.go:120`) is
   `Model → DeviceModelName → ""`. DJI MP4s leave both empty and
   instead populate `Encoder` with the camera identity, e.g.
   `Encoder: DJI OsmoPocket4`. All these MP4s currently route to
   `<year>/sources/Unknown (video)/...` — losing the camera attribution.

## Goal

- Stop importing the three DJI junk extensions, and stop flagging them
  during verify when they already exist in a vault.
- When `Encoder` looks like a DJI camera string and EXIF Make/Model
  are both empty, parse `Make` and `Model` from `Encoder` so the
  resulting device directory is `DJI <Model> (video)`.

## Non-Goals

- **No cleanup of existing files** already imported to
  `Unknown (image|video)` or `0001-01-01`. They stay where they are.
  Re-importing the original source after this change would route
  them correctly; that is a manual choice.
- **No `imv verify --fix` enhancement** to relocate previously-
  imported files based on the new Encoder-derived model.
- **No additional camera-brand encoder patterns** (GoPro, Insta360,
  etc.). DJI-only for this change. A future spec can generalize the
  pattern set.
- **No configurable denylist** via flag or env var. Hard-coded to
  match the existing convention of `defaults.IgnoredFiles`.
- **No CLI surface changes.** No new commands, flags, or config.
- **No filtering applied to `imv tools scan` / `tools ext-count`.**
  Those commands intentionally use raw `filepath.Walk` to give the
  user a complete inventory; that stays untouched.

## Design

### Convention

The denylist mechanism in `internal/defaults/defaults.go` already
encodes OS-generated junk filenames via `IgnoredFiles` and the
`IsIgnoredFile` lookup. The same mechanism extends naturally to
junk extensions.

DJI's `Encoder` tag carries the camera identity for MP4 outputs in
the form `<Brand> <Model>` (e.g. `DJI OsmoPocket4`,
`DJI Osmo Action 5 Pro`). When neither `Make` nor `Model` resolved
from EXIF, this string is the only available camera attribution and
should be parsed by splitting on the first whitespace run.

### Junk extension denylist

**Package additions** (`internal/defaults/defaults.go`):

```go
var IgnoredExtensions = []string{".thm", ".lrf", ".scr"}

func IsIgnoredExtension(ext string) bool   // case-insensitive

func IsIgnored(name string) bool {
    return IsIgnoredFile(name) || IsIgnoredExtension(filepath.Ext(name))
}
```

`IgnoredExtensions` is initialized into a lookup set at package init
identically to `IgnoredFiles` (`defaults.go:38-43`). `IsIgnoredFile`
and `IsIgnoredExtension` remain exported so tests can assert each
axis independently.

**Call-site migration.** All four current call sites of
`IsIgnoredFile` switch to `IsIgnored`:

| File | Line | Context |
|---|---|---|
| `internal/importer/importer.go` | 199 | `enumerateFiles` — skip during source walk |
| `internal/library/library.go` | 212 | library introspection |
| `internal/extnormalize/extnormalize.go` | 72 | normalize-ext walk |
| `internal/verifier/cache.go` | 51 | verify cache loader |

Result:
- `imv import <src>` — junk extensions are dropped before hashing
  or EXIF extraction.
- `imv verify` — pre-existing junk files in the vault are treated
  exactly like `.DS_Store`: ignored, not flagged.
- `imv lib-tools normalize-ext` — junk extensions are skipped
  (consistent with how it already skips OS-junk filenames).

### Encoder fallback for camera model

**Where:** `internal/metadata/metadata.go`, inside
`BuildFileMetadata`, after the existing Make and Model resolution
blocks.

**Implementation ordering.** The existing code assigns
`make_ = "Unknown"` immediately after the Make/Manufacturer
fallback. To let the Encoder branch decide on its own, that
`"Unknown"` assignment moves to AFTER the new Encoder block — so
the branch can test `make_ == ""` directly. Model resolution stays
in place; it never assigns a default string.

**Trigger condition** — all must hold:

1. `Model` (after `Model → DeviceModelName` fallback and
   `NormalizeModel`) is empty.
2. `make_` is empty at the point of the check (i.e. neither `Make`
   nor `DeviceManufacturer` resolved to a non-empty value after
   `NormalizeMake`). The `"Unknown"` assignment has been deferred
   per the ordering note above.
3. `Encoder` field starts with `"DJI"` (case-insensitive prefix
   match against either `"DJI"` alone or `"DJI "` followed by more
   text).

Condition 2 is the conservative guard: if EXIF already identified a
legitimate Make (e.g. `Sony`), the Encoder branch does not fire
and the existing Model-empty-string behavior is preserved.

The prefix check (`"DJI"` start) rather than substring (`"DJI"`
anywhere) prevents misparses of unusual strings like
`"Encoded by Foo (DJI Studio)"`. The exact rule:

```go
upper := strings.ToUpper(strings.TrimSpace(encoder))
isDJI := upper == "DJI" || strings.HasPrefix(upper, "DJI ")
```

**Parse rule.** Split `Encoder` on its first whitespace run:

| Encoder | Make | Model |
|---|---|---|
| `DJI OsmoPocket4` | `DJI` | `OsmoPocket4` |
| `DJI Osmo Action 5 Pro` | `DJI` | `Osmo Action 5 Pro` |
| `DJI` | `DJI` | `""` |
| `dji osmopocket4` | `DJI` (via `NormalizeMake`) | `osmopocket4` |

When `Model` is empty after parsing, `DeviceDir`
(`internal/pathbuilder/pathbuilder.go:47`) produces
`DJI (video)`, which is the same shape used today when only Make
is known.

**Make/Model assignment.** When the trigger condition holds, the
Encoder branch sets `make_` to the parsed brand (`"DJI"`, after
running through `defaults.NormalizeMake`) and sets `model` to the
parsed remainder. Control then falls through to the deferred
`if make_ == "" { make_ = "Unknown" }` guard — which is a no-op
in this case because `make_` is now `"DJI"`. If the trigger
condition does not hold, `make_` remains empty and that same
guard assigns `"Unknown"` as it does today.

### Resulting path comparison

Same DJI Osmo Pocket MP4 file:

```
Before: 2026/sources/Unknown (video)/2026-05-22/2026-05-22_15-56-07_<hash>.mp4
After:  2026/sources/DJI OsmoPocket4 (video)/2026-05-22/2026-05-22_15-56-07_<hash>.mp4
```

`.thm` / `.lrf` / `.scr` companions are not imported at all.

### Error handling

Both features are silent and infallible. Skipping a file during
enumeration is the success path. Failing the Encoder trigger
condition leaves Make/Model resolution at the existing
`"Unknown"` / `""` defaults. No new error returns.

### Backward compatibility

- `IgnoredFiles` and `IsIgnoredFile` remain exported and unchanged.
- The Encoder branch is purely additive — it can only fire in a
  state where the existing code would have produced
  `Make="Unknown"` and `Model=""`. Any file that previously
  resolved Make or Model from EXIF is unaffected.
- Files already imported under the old behavior stay at their
  current paths; the verify cache treats them as expected via the
  same path-derivation logic that placed them there.

## Tests

### `internal/defaults/defaults_test.go`

New cases:

- `IsIgnoredExtension(".thm")`, `.lrf`, `.scr` → true.
- Case-insensitive: `.THM`, `.LrF` → true.
- `IsIgnoredExtension(".jpg")`, `.mp4` → false.
- `IsIgnoredExtension("")` → false.
- `IsIgnored(".DS_Store")` → true (filename axis).
- `IsIgnored("clip.thm")` → true (extension axis).
- `IsIgnored("clip.MP4")` → false.
- `IsIgnored("clip.jpg")` → false.

### `internal/metadata/metadata_test.go`

New cases for the Encoder branch in `BuildFileMetadata`:

- `Encoder="DJI OsmoPocket4"`, Make/Model/Manufacturer/
  DeviceModelName empty → Make=`DJI`, Model=`OsmoPocket4`.
- `Encoder="DJI Osmo Action 5 Pro"` → Make=`DJI`,
  Model=`Osmo Action 5 Pro`.
- `Encoder="DJI"` alone → Make=`DJI`, Model=`""`.
- `Encoder="dji osmopocket4"` → Make=`DJI`, Model=`osmopocket4`.
- `Encoder="Lavf61.7.100"` → branch does NOT fire;
  Make=`Unknown`, Model=`""`.
- `Encoder="DJI OsmoPocket4"` but `Make="Sony"` already set →
  branch does NOT fire; Make=`Sony` preserved, Model=`""`.
- No `Encoder` field present → existing behavior unchanged.
- `Encoder=" DJI OsmoPocket4 "` (whitespace padding) → still
  triggers, Make=`DJI`, Model=`OsmoPocket4`.

### `internal/integration_test.go`

Extend the existing import → verify integration test with a
DJI-shaped fixture:
- A synthetic source MP4 whose stubbed EXIF carries only
  `Encoder=DJI OsmoPocket4` and a `MediaCreateDate`.
- A companion `.thm` file at the same basename.

Assertions:
- The MP4 lands at
  `<year>/sources/DJI OsmoPocket4 (video)/<date>/...mp4`.
- The `.thm` file is not present in the vault.
- A subsequent `verify` run on the vault reports no errors.

## Out of scope summary

Restating for clarity:

- No cleanup of existing files in `0001/`, `2000/`, or any
  `Unknown (video|image)/` directory.
- No `verify --fix` relocation of files based on the new
  Encoder-derived model.
- No support for non-DJI Encoder patterns.
- No user-configurable denylist or Encoder pattern list.
- No CLI surface changes.
- No filtering in `tools scan` / `tools ext-count`.
