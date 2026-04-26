# `undated/` Root Directory

## Problem

The vault layout requires every file to live under a year directory (`YYYY/`).
Some files have no usable year — no EXIF date, no reliable filesystem date — or
the user simply doesn't want to file them by year. Today there is no convention
for these, and `verify` warns on any non-year directory at the vault root.

## Goal

Allow a single freeform directory `undated/` at the vault root for media that
has no year, or that the user deliberately wants to keep outside the
year-based hierarchy.

## Non-Goals

- No per-year `undated/` (the year-level layout is unchanged).
- No config file or allowlist mechanism. The name is fixed.
- No `import` integration. `undated/` is populated manually by the user.
- No centralization of well-known directory names. The existing inline-string
  pattern in the verifier stays as-is.

## Design

### Convention

```
~/Photos/
  2024/
    sources/
    sources-manual/
    processed/
  2025/
    ...
  undated/        # NEW — freeform, vault root only
    ...           # anything; not validated
```

Rules:

- Located at the vault root, sibling to year directories.
- Contents are freeform: any file or directory layout is allowed inside.
- `verify` does not recurse into `undated/` and does not warn about it.
- The directory is optional. If absent, behavior is unchanged.

### Behavior changes

| Command | Behavior |
|---|---|
| `verify` | Treats `undated/` at vault root as allowed. Does not recurse. Other non-year root entries still warn as today. |
| `verify --fix` | Same as `verify`. No fixes are applied to `undated/`. |
| `verify --year YYYY` | Unaffected — `undated/` is not a year and is not selected by year filtering. |
| `import` | Unchanged. Import always writes into year directories; it never reads from or writes to `undated/`. |
| `tools remove-empty-dirs` | Unchanged. Operates on whatever path is given; if `undated/` happens to be empty it is removed (consistent with current behavior for any empty dir). |
| `tools scan` / `diff` / `ext-count` | Unchanged. These already operate on arbitrary paths. |

### Implementation

One change in `internal/verifier/verifier.go::verifyLibraryRoot`: extend the
existing root-level check so that, for directory entries that are not year
directories, the literal name `"undated"` is allowed without warning. Files at
the root and other unknown directories still warn / fail-fast as today.

This mirrors the inline allowlist pattern already used in `verifyYearLevel`
(`sources`, `processed`, `sources-manual`, `.imv`).

No other source files change.

### Documentation

`README.md`:

- Add `undated/` to the library structure tree at the vault root, sibling to
  the year directories.
- Add one short note to the structure section explaining the purpose:
  "Use `undated/` at vault root for files with no year or that you don't want
  filed by year. Freeform inside, not validated."

## Testing

Add to `internal/verifier/verifier_test.go`:

1. Root contains `undated/` with arbitrary contents → verify passes with no
   "unexpected directory" warning.
2. Root contains `undated/` plus another non-year dir (e.g. `random/`) → only
   `random/` warns; `undated/` is silent.
3. `undated/` with `--fail-fast`: arbitrary contents inside do not cause
   fail-fast to trip.

Existing tests for unknown-name root dirs remain unchanged and must still pass.

## Risks

- **Name collision.** A user with an existing `undated/` directory containing
  unrelated content would suddenly have it accepted by `verify`. This is the
  intent; no data is moved or modified.
- **Scope creep.** Future requests may push for additional well-known root
  names or per-year `undated/`. Out of scope for this change; revisit if and
  when needed.
