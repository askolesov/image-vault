package verifier

import (
	"bufio"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/askolesov/image-vault/internal/defaults"
)

const (
	cacheFileName      = "verify.cache"
	cacheDirName       = ".imv"
	cacheFormatVersion = "v1"
	cacheFieldSep      = "\t"
	persistInterval    = 30 * time.Second
)

// Entry is a single cached verification record.
type Entry struct {
	RelPath    string
	Size       int64
	MtimeNs    int64
	HashAlgo   string
	VerifiedAt int64
}

// Cache holds per-year verification cache state. It is an in-memory
// map[string]Entry plus a Persist method that atomically rewrites the
// whole file. No file handle is held open between persists — this is
// the key property that keeps the cache stable across cross-filesystem
// scenarios (SMB/CIFS, FUSE, permission drift between machines).
//
// A nil *Cache is a valid no-op receiver for every method.
type Cache struct {
	path        string
	entries     map[string]Entry
	lastPersist time.Time
	dirty       bool
}

// isSkippableInLibrary reports whether a filename should be silently skipped
// during structural validation — OS junk files plus any .cache file (state
// reserved for imv, not user content).
func isSkippableInLibrary(name string) bool {
	if defaults.IsIgnoredFile(name) {
		return true
	}
	return strings.EqualFold(filepath.Ext(name), ".cache")
}

// CacheFilePath returns the canonical cache file path for a year directory.
func CacheFilePath(yearDir string) string {
	return filepath.Join(yearDir, cacheDirName, cacheFileName)
}

// CacheDirPath returns the .imv directory path for a year directory.
func CacheDirPath(yearDir string) string {
	return filepath.Join(yearDir, cacheDirName)
}

// NewEntry builds an Entry from a relative path, file info, and hash algo.
// VerifiedAt is set to now.
func NewEntry(relPath string, fi os.FileInfo, algo string) Entry {
	return Entry{
		RelPath:    relPath,
		Size:       fi.Size(),
		MtimeNs:    fi.ModTime().UnixNano(),
		HashAlgo:   algo,
		VerifiedAt: time.Now().Unix(),
	}
}

// Load parses the cache file at path (if present) and returns a populated
// Cache. A missing file is not an error. Malformed lines are silently
// skipped. The returned Cache has lastPersist zero-valued, so the first
// Record after Load will trigger a persist; callers that want to control
// initial persist timing (e.g. openYearCache after intersection) should
// call Persist explicitly.
func Load(path string) (*Cache, error) {
	c := &Cache{
		path:    path,
		entries: make(map[string]Entry),
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return nil, fmt.Errorf("open cache: %w", err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		e, ok := parseCacheLine(line)
		if !ok {
			continue
		}
		c.entries[e.RelPath] = e
	}
	if err := scanner.Err(); err != nil {
		return c, fmt.Errorf("scan cache: %w", err)
	}

	return c, nil
}

// Lookup returns the cached entry for relPath, if any.
func (c *Cache) Lookup(relPath string) (Entry, bool) {
	if c == nil {
		return Entry{}, false
	}
	e, ok := c.entries[relPath]
	return e, ok
}

// Matches reports whether e is still valid for the given FileInfo and algo.
// Mtime is compared at whole-second precision: SMB/CIFS, NFSv3, FAT, and
// several FUSE mounts quantize mtime to seconds, so a cache populated on
// one host (native FS, nanosecond precision) and read on another (SMB,
// second precision) against the same underlying file would otherwise
// mismatch every entry. Second granularity is the common denominator
// supported by every filesystem we care about.
func (c *Cache) Matches(e Entry, fi os.FileInfo, algo string) bool {
	if c == nil || fi == nil {
		return false
	}
	const nsPerSec = int64(time.Second)
	return e.Size == fi.Size() &&
		e.MtimeNs/nsPerSec == fi.ModTime().Unix() &&
		e.HashAlgo == algo
}

// Entries returns a snapshot of current entries (read-only; for observability).
func (c *Cache) Entries() map[string]Entry {
	if c == nil {
		return nil
	}
	out := make(map[string]Entry, len(c.entries))
	maps.Copy(out, c.entries)
	return out
}

// Persist atomically rewrites the on-disk cache file from the current
// in-memory entries. Uses tmp + fsync + rename (with remove+rename
// fallback for filesystems that refuse overwrite). On success: clears
// dirty and updates lastPersist. On failure: leaves the original file
// untouched and the in-memory map unchanged.
func (c *Cache) Persist() error {
	if c == nil {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return fmt.Errorf("mkdir cache dir: %w", err)
	}

	tmpPath := c.path + ".tmp"
	tmp, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("create tmp cache: %w", err)
	}

	writer := bufio.NewWriter(tmp)
	if err := writeCacheHeader(writer); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write cache header: %w", err)
	}
	for _, e := range c.entries {
		if _, err := fmt.Fprintln(writer, formatCacheLine(e)); err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmpPath)
			return fmt.Errorf("write cache entry: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("flush tmp cache: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("fsync tmp cache: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close tmp cache: %w", err)
	}

	if err := renameOverwrite(tmpPath, c.path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename cache: %w", err)
	}

	c.lastPersist = time.Now()
	c.dirty = false
	return nil
}

// Record inserts e into the in-memory cache. If persistInterval has
// elapsed since lastPersist, Record also calls Persist as a side effect.
// Paths containing tab or newline are rejected (they would corrupt the
// TSV format) — the map is not mutated and no persist happens in that
// case.
func (c *Cache) Record(e Entry) error {
	if c == nil {
		return nil
	}
	if strings.ContainsAny(e.RelPath, "\t\n") {
		return fmt.Errorf("cache: path contains tab or newline: %q", e.RelPath)
	}
	c.entries[e.RelPath] = e
	c.dirty = true

	if time.Since(c.lastPersist) > persistInterval {
		return c.Persist()
	}
	return nil
}

// renameOverwrite renames src over dst. POSIX rename(2) overwrites on the
// same filesystem, but SMB/CIFS and several FUSE mounts refuse to overwrite
// with various errnos. On any rename failure, remove dst and retry. If the
// retry fails the cache is lost — acceptable since this is a regenerable
// cache, not durable state.
func renameOverwrite(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	_ = os.Remove(dst)
	return os.Rename(src, dst)
}

func writeCacheHeader(w *bufio.Writer) error {
	if _, err := fmt.Fprintf(w, "# imv verify-cache %s \u2014 fields: path\\tsize\\tmtime_ns\\thash_algo\\tverified_at_unix\n", cacheFormatVersion); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# Invalidated when files are copied without preserving mtime. Use rsync -a or cp -p."); err != nil {
		return err
	}
	return nil
}

func formatCacheLine(e Entry) string {
	var b strings.Builder
	b.WriteString(e.RelPath)
	b.WriteString(cacheFieldSep)
	b.WriteString(strconv.FormatInt(e.Size, 10))
	b.WriteString(cacheFieldSep)
	b.WriteString(strconv.FormatInt(e.MtimeNs, 10))
	b.WriteString(cacheFieldSep)
	b.WriteString(e.HashAlgo)
	b.WriteString(cacheFieldSep)
	b.WriteString(strconv.FormatInt(e.VerifiedAt, 10))
	return b.String()
}

func parseCacheLine(line string) (Entry, bool) {
	parts := strings.Split(line, cacheFieldSep)
	if len(parts) != 5 {
		return Entry{}, false
	}
	if parts[0] == "" || parts[3] == "" {
		return Entry{}, false
	}
	size, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || size < 0 {
		return Entry{}, false
	}
	mtimeNs, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return Entry{}, false
	}
	verifiedAt, err := strconv.ParseInt(parts[4], 10, 64)
	if err != nil {
		return Entry{}, false
	}
	return Entry{
		RelPath:    parts[0],
		Size:       size,
		MtimeNs:    mtimeNs,
		HashAlgo:   parts[3],
		VerifiedAt: verifiedAt,
	}, true
}
