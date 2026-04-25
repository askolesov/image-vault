# Verify Scan Progress Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Surface a `[scan] <year>: X dirs, Y files` line on stderr during the otherwise-silent `library.ListSourceFiles` walk that runs at the start of every year in `verify`.

**Architecture:** Add a `ProgressCallback` to `library.ListSourceFiles` (mirroring the `RemoveEmptyDirsProgress` convention already in the same package), throttled every 100 dirs plus a final emit. Add a TTY-aware `Logger.Scan(label, dirs, files)` method that overwrites in place on a TTY and prints periodic lines off-TTY. Wire the callback in `verifier.walkAndStatYear`. Stat and cache-prep phases stay silent.

**Tech Stack:** Go, existing `internal/library` and `internal/logging` packages, testify for tests.

**Reference spec:** `docs/superpowers/specs/2026-04-26-verify-scan-progress-design.md`

---

### Task 1: Add `Logger.Scan` method

**Files:**
- Modify: `internal/logging/logging.go` (add method after `ProgressWithStats`)
- Test: `internal/logging/logging_test.go`

- [ ] **Step 1: Add the failing tests**

Append to `internal/logging/logging_test.go`:

```go
func TestLoggerScanNonTTY(t *testing.T) {
	var stdout, stderr bytes.Buffer
	l := New(&stdout, &stderr, false)

	l.Scan("2024", 1_234, 56_789)

	out := stderr.String()
	assert.Equal(t, "[scan] 2024: 1,234 dirs, 56,789 files\n", out)
	assert.Empty(t, stdout.String())
}

func TestLoggerScanTTY(t *testing.T) {
	var stdout, stderr bytes.Buffer
	l := New(&stdout, &stderr, true)

	l.Scan("2024", 100, 4_123)

	out := stderr.String()
	assert.Contains(t, out, "\r\033[K")
	assert.Contains(t, out, "[scan] 2024: 100 dirs, 4,123 files")
	assert.NotContains(t, out, "\n")
	assert.Empty(t, stdout.String())
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/logging/ -run TestLoggerScan -v`

Expected: build error — `l.Scan undefined`.

- [ ] **Step 3: Implement `Logger.Scan`**

Insert after the `ProgressWithStats` method (around `internal/logging/logging.go:92`):

```go
// Scan reports an unbounded scan with running counts. On a TTY it
// overwrites in place; off-TTY it prints a single line each call (the
// caller is expected to throttle off-TTY callers to avoid log spam).
func (l *Logger) Scan(label string, dirs, files int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.isTTY {
		_, _ = fmt.Fprintf(l.stderr, "\r\033[K[scan] %s: %s dirs, %s files",
			label, FormatNumber(dirs), FormatNumber(files))
	} else {
		_, _ = fmt.Fprintf(l.stderr, "[scan] %s: %s dirs, %s files\n",
			label, FormatNumber(dirs), FormatNumber(files))
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/logging/ -v`

Expected: all tests PASS, including the two new ones.

- [ ] **Step 5: Commit**

```bash
git add internal/logging/logging.go internal/logging/logging_test.go
git commit -m "feat(logging): add Scan method for unbounded progress"
```

---

### Task 2: Add progress callback to `library.ListSourceFiles`

**Files:**
- Modify: `internal/library/library.go:56-90`
- Modify: `internal/library/library_test.go:77,152,162,227,340` (call sites)
- Modify: `internal/integration_test.go:195,225` (call sites)
- Modify: `internal/verifier/verifier.go:142` (call site — temporary `{}` until Task 3)

**Note on ordering:** The signature change forces every caller to compile-update at once. This task does the signature change, the implementation, the new test, and the mechanical caller updates in one commit. Task 3 adds the real callback wiring in the verifier.

- [ ] **Step 1: Add the failing test**

Append to `internal/library/library_test.go`:

```go
func TestListSourceFilesProgress(t *testing.T) {
	dir := t.TempDir()
	makeFile(t, dir, "sources/iphone/2024-01-01/img1.jpg", "data")
	makeFile(t, dir, "sources/iphone/2024-01-01/img2.jpg", "data")
	makeFile(t, dir, "sources/canon/2024-02-15/img3.cr2", "data")

	var calls []struct{ dirs, files int }
	files, err := ListSourceFiles(dir, ListSourceFilesProgress{
		OnScan: func(d, f int) {
			calls = append(calls, struct{ dirs, files int }{d, f})
		},
	})
	require.NoError(t, err)
	assert.Len(t, files, 3)
	require.NotEmpty(t, calls, "OnScan should be called at least once")

	final := calls[len(calls)-1]
	assert.Equal(t, 3, final.files, "final tally should report all 3 files")
	assert.GreaterOrEqual(t, final.dirs, 1, "final tally should count at least the sources dir")
}

func TestListSourceFilesNilProgress(t *testing.T) {
	dir := t.TempDir()
	makeFile(t, dir, "sources/iphone/2024-01-01/img1.jpg", "data")

	files, err := ListSourceFiles(dir, ListSourceFilesProgress{})
	require.NoError(t, err)
	assert.Len(t, files, 1)
}
```

- [ ] **Step 2: Run the new tests to verify they fail**

Run: `go test ./internal/library/ -run TestListSourceFilesProgress -v` and `go test ./internal/library/ -run TestListSourceFilesNilProgress -v`

Expected: build error — `ListSourceFilesProgress undefined` and signature mismatch.

- [ ] **Step 3: Update `ListSourceFiles` signature and add progress struct**

Replace `internal/library/library.go:56-90` with:

```go
// ListSourceFilesProgress reports progress during ListSourceFiles.
type ListSourceFilesProgress struct {
	// OnScan is called periodically during the walk with running totals,
	// and once more at the end with the final tally. May be nil.
	OnScan func(dirs, files int)
}

// listSourceFilesProgressInterval is how often (in dirs) OnScan fires
// during the walk.
const listSourceFilesProgressInterval = 100

// ListSourceFiles walks <yearDir>/sources/ recursively and returns all file paths (not dirs).
// Skips permission errors. Returns nil if sources/ doesn't exist.
//
// If p.OnScan is non-nil it is invoked every listSourceFilesProgressInterval
// directories with running totals, and once more at the end with the final
// tally. Both counts are zero when sources/ doesn't exist.
func ListSourceFiles(yearDir string, p ListSourceFilesProgress) ([]string, error) {
	sourcesDir := filepath.Join(yearDir, "sources")

	info, err := os.Stat(sourcesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat sources dir: %w", err)
	}
	if !info.IsDir() {
		return nil, nil
	}

	var (
		files      []string
		dirs       int
		fileCount  int
	)
	err = filepath.WalkDir(sourcesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			dirs++
			if p.OnScan != nil && dirs%listSourceFilesProgressInterval == 0 {
				p.OnScan(dirs, fileCount)
			}
			return nil
		}
		files = append(files, path)
		fileCount++
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking sources: %w", err)
	}

	if p.OnScan != nil {
		p.OnScan(dirs, fileCount)
	}

	return files, nil
}
```

- [ ] **Step 4: Update existing test call sites**

Update each existing call in `internal/library/library_test.go` to pass `ListSourceFilesProgress{}`:

- Line 77 (in `TestListSourceFiles`): `files, err := ListSourceFiles(dir)` → `files, err := ListSourceFiles(dir, ListSourceFilesProgress{})`
- Line 152 (in `TestListSourceFilesNoSourcesDir`): same pattern
- Line 162 (in `TestListSourceFilesSourcesIsFile`): same pattern
- Line 227 (in `TestListSourceFilesPermissionError`): same pattern
- Line 340 (in `TestListSourceFilesStatError`): `_, err := ListSourceFiles(dir)` → `_, err := ListSourceFiles(dir, ListSourceFilesProgress{})`

- [ ] **Step 5: Update integration test call sites**

In `internal/integration_test.go`:

- Line 195: `files, err := library.ListSourceFiles(filepath.Join(libDir, "2024"))` → `files, err := library.ListSourceFiles(filepath.Join(libDir, "2024"), library.ListSourceFilesProgress{})`
- Line 225: same pattern

- [ ] **Step 6: Update verifier call site (temporary, no callback yet)**

In `internal/verifier/verifier.go:142`, change:

```go
paths, err := library.ListSourceFiles(yearDir)
```

to:

```go
paths, err := library.ListSourceFiles(yearDir, library.ListSourceFilesProgress{})
```

(Task 3 will replace this with the real callback.)

- [ ] **Step 7: Run all tests to verify the build is green**

Run: `go test ./...`

Expected: all tests pass, including the two new `TestListSourceFilesProgress*` tests.

- [ ] **Step 8: Commit**

```bash
git add internal/library/library.go internal/library/library_test.go internal/integration_test.go internal/verifier/verifier.go
git commit -m "feat(library): add ListSourceFiles progress callback"
```

---

### Task 3: Wire scan progress into the verifier

**Files:**
- Modify: `internal/verifier/verifier.go:141-163` (the `walkAndStatYear` function)

- [ ] **Step 1: Replace the placeholder progress with the real callback**

In `internal/verifier/verifier.go`, replace the body of `walkAndStatYear`'s `ListSourceFiles` call. Change:

```go
func (v *Verifier) walkAndStatYear(yearDir, year string) ([]FileEntry, error) {
	paths, err := library.ListSourceFiles(yearDir, library.ListSourceFilesProgress{})
	if err != nil {
		return nil, fmt.Errorf("list source files for %s: %w", year, err)
	}
```

to:

```go
func (v *Verifier) walkAndStatYear(yearDir, year string) ([]FileEntry, error) {
	paths, err := library.ListSourceFiles(yearDir, library.ListSourceFilesProgress{
		OnScan: func(dirs, files int) {
			v.logger.Scan(year, dirs, files)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("list source files for %s: %w", year, err)
	}
```

- [ ] **Step 2: Build and run all tests**

Run: `go build ./...` then `go test ./...`

Expected: build succeeds, all tests pass. The wiring is exercised by the existing integration tests in `internal/integration_test.go`.

- [ ] **Step 3: Manual smoke test against a real library**

Run a short scan against a real year (use `--fast` to skip the slow per-file work and just exercise the scan phase, then exit early):

```bash
go run ./cmd/imv verify --year 2024 --fast 2>&1 | head -5
```

Expected: at least one `[scan] 2024: ...` line on stderr (visible because piped to a non-TTY here), followed by the per-file `[progress]` line(s).

(Skip this step if no real library is available locally — the integration tests already cover correctness.)

- [ ] **Step 4: Commit**

```bash
git add internal/verifier/verifier.go
git commit -m "feat(verify): show scan progress during per-year walk"
```

---

### Task 4: Open the PR

- [ ] **Step 1: Push the branch**

The current branch is `fix/import-move-duplicate-skipped`. Decide whether these commits belong on a new branch. If the verify-scan-progress work should not piggyback on the import fix, create a fresh branch off `main`:

```bash
git log --oneline main..HEAD
```

If the only commits ahead of `main` are the three from this plan plus the spec commit, the current branch may be reused — or rename/rebase per project convention. If unsure, ask before pushing.

- [ ] **Step 2: Push to remote**

```bash
git push -u origin <branch-name>
```

- [ ] **Step 3: Open the PR via `gh`**

```bash
gh pr create --title "feat(verify): show scan progress during per-year walk" --body "$(cat <<'EOF'
## Summary
- Adds a `[scan] <year>: X dirs, Y files` line on stderr during the otherwise-silent `library.ListSourceFiles` walk that runs at the start of every year in `verify`.
- New TTY-aware `Logger.Scan` method (overwrites in place on TTY, periodic lines off-TTY).
- `library.ListSourceFiles` now takes a `ListSourceFilesProgress{OnScan}` struct, throttled every 100 dirs plus a final tally. Stat and cache-prep phases stay silent.

Spec: `docs/superpowers/specs/2026-04-26-verify-scan-progress-design.md`

## Test plan
- [ ] `go test ./...` green
- [ ] Run `imv verify --year <year>` against a real library, confirm scan line appears on stderr before per-file progress takes over
- [ ] Pipe stderr to a file (`imv verify 2> verify.log`) and confirm scan emits one line per ~100 dirs (no carriage returns)
EOF
)"
```

Expected: PR URL printed; report it back to the user.

---

## Self-review

**Spec coverage:**
- §1 (`ListSourceFiles` signature) — Task 2 steps 3, 7
- §2 (`Logger.Scan` method) — Task 1 steps 3, 4
- §3 (verifier wiring) — Task 3 step 1
- §4 test plan — Task 1 step 1, Task 2 step 1
- §5 sample output — Task 3 step 3 (manual smoke test)

**Placeholders:** none — every step shows actual code or actual commands.

**Type consistency:** `ListSourceFilesProgress`, `OnScan(dirs, files int)`, `Logger.Scan(label, dirs, files)` are spelled identically wherever they appear (Task 1 step 3, Task 2 step 3, Task 3 step 1).

No fixes needed.
