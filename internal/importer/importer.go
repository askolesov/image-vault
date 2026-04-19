package importer

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/askolesov/image-vault/internal/defaults"
	"github.com/askolesov/image-vault/internal/logging"
	"github.com/askolesov/image-vault/internal/metadata"
	"github.com/askolesov/image-vault/internal/pathbuilder"
	"github.com/askolesov/image-vault/internal/transfer"
)

// MetadataExtractor extracts metadata from a file.
type MetadataExtractor interface {
	Extract(path string, hasher *defaults.Hasher) (*metadata.FileMetadata, error)
}

// Config holds configuration for the importer.
type Config struct {
	LibraryPath   string
	SeparateVideo bool
	HashAlgo      string
	KeepAll       bool
	FailFast      bool
	Move          bool
	DryRun        bool
	SkipCompare   bool
	Randomize     bool
	YearFilter    string
}

// Result holds the outcome counts of an import operation.
type Result struct {
	Imported       int
	Skipped        int
	Replaced       int
	Dropped        int
	Errors         int
	ProcessedBytes int64
}

// fileWithSidecars groups a primary file with its sidecar files.
type fileWithSidecars struct {
	Path     string
	Sidecars []string
}

// Importer orchestrates the per-file import pipeline.
type Importer struct {
	cfg    Config
	ext    MetadataExtractor
	logger *logging.Logger
	hasher *defaults.Hasher
}

// New creates a new Importer, initializing the hasher from cfg.HashAlgo.
// Returns an error if cfg.HashAlgo is unsupported so callers can surface
// the misconfiguration instead of silently substituting the default.
func New(cfg Config, ext MetadataExtractor, logger *logging.Logger) (*Importer, error) {
	hasher, err := defaults.NewHasher(cfg.HashAlgo)
	if err != nil {
		return nil, fmt.Errorf("importer: %w", err)
	}
	return &Importer{
		cfg:    cfg,
		ext:    ext,
		logger: logger,
		hasher: hasher,
	}, nil
}

// ImportDir imports all files from sourceDir into the library.
func (imp *Importer) ImportDir(sourceDir string) (*Result, error) {
	files, err := enumerateFiles(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("enumerate files: %w", err)
	}

	groups := linkSidecars(files)

	if imp.cfg.Randomize {
		rand.Shuffle(len(groups), func(i, j int) {
			groups[i], groups[j] = groups[j], groups[i]
		})
	}

	result := &Result{}
	total := len(groups)

	for i, g := range groups {
		stats := fmt.Sprintf("new:%d skipped:%d dropped:%d %s",
			result.Imported, result.Skipped, result.Dropped, logging.FormatBytes(result.ProcessedBytes))
		imp.logger.ProgressWithStats(i+1, total, "", stats, g.Path)

		if err := imp.importFile(g, result); err != nil {
			result.Errors++
			imp.logger.Error("import %s: %v", g.Path, err)
			if imp.cfg.FailFast {
				return result, err
			}
		}
	}

	return result, nil
}

func (imp *Importer) importFile(g fileWithSidecars, result *Result) error {
	md, err := imp.ext.Extract(g.Path, imp.hasher)
	if err != nil {
		return fmt.Errorf("extract metadata: %w", err)
	}

	// Drop non-media files unless KeepAll
	if md.MediaType == defaults.MediaTypeOther && !imp.cfg.KeepAll {
		result.Dropped++
		return nil
	}

	// Year filter
	if imp.cfg.YearFilter != "" {
		year := md.DateTime.Format("2006")
		if year != imp.cfg.YearFilter {
			result.Skipped++
			return nil
		}
	}

	// Build destination path
	pbOpts := pathbuilder.Options{SeparateVideo: imp.cfg.SeparateVideo}
	relPath := pathbuilder.BuildSourcePath(md, pbOpts)
	destPath := filepath.Join(imp.cfg.LibraryPath, relPath)

	// Transfer — pass hasher and pre-computed source hash to avoid re-reading the file
	tOpts := transfer.Options{
		Move:        imp.cfg.Move,
		DryRun:      imp.cfg.DryRun,
		NewHash:     imp.hasher.New,
		SourceHash:  md.FullHash,
		SkipCompare: imp.cfg.SkipCompare,
	}

	// Stat before transfer — if --move succeeds the source file is gone.
	var sourceSize int64
	if info, statErr := os.Stat(g.Path); statErr == nil {
		sourceSize = info.Size()
	}

	action, err := transfer.TransferFile(g.Path, destPath, tOpts)
	if err != nil {
		return fmt.Errorf("transfer file: %w", err)
	}

	// Map action to result counts. ProcessedBytes counts only bytes that
	// actually moved to (or would move to) the library — skipped dupes
	// and non-media drops are excluded, so the number matches what users
	// expect from a "Processed" label.
	switch action {
	case transfer.ActionCopied, transfer.ActionMoved, transfer.ActionWouldCopy, transfer.ActionWouldMove:
		result.Imported++
		result.ProcessedBytes += sourceSize
	case transfer.ActionReplaced, transfer.ActionWouldReplace:
		result.Replaced++
		result.ProcessedBytes += sourceSize
	case transfer.ActionSkipped:
		result.Skipped++
	}

	// Transfer sidecars
	for _, sidecar := range g.Sidecars {
		sidecarExt := filepath.Ext(sidecar)
		sidecarDest := pathbuilder.BuildSidecarPath(destPath, sidecarExt)
		if _, err := transfer.TransferFile(sidecar, sidecarDest, tOpts); err != nil {
			return fmt.Errorf("transfer sidecar %s: %w", sidecar, err)
		}
	}

	return nil
}

// enumerateFiles walks sourceDir recursively, returning all files (skipping
// directories, permission errors, and OS junk files).
func enumerateFiles(sourceDir string) ([]string, error) {
	var files []string
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				return nil
			}
			return err
		}
		if info.IsDir() {
			return nil
		}
		if defaults.IsIgnoredFile(info.Name()) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, err
}

// linkSidecars groups files by base name without extension.
// Primary = non-sidecar extension, sidecar = sidecar extension.
// If no primary exists, sidecars become primaries (orphan sidecars).
func linkSidecars(files []string) []fileWithSidecars {
	type group struct {
		primaries []string
		sidecars  []string
	}

	groups := make(map[string]*group)
	// Track insertion order
	var keys []string

	for _, f := range files {
		dir := filepath.Dir(f)
		base := filepath.Base(f)
		ext := filepath.Ext(base)
		nameNoExt := strings.TrimSuffix(base, ext)
		key := filepath.Join(dir, nameNoExt)

		g, ok := groups[key]
		if !ok {
			g = &group{}
			groups[key] = g
			keys = append(keys, key)
		}

		if defaults.IsSidecarExtension(ext) {
			g.sidecars = append(g.sidecars, f)
		} else {
			g.primaries = append(g.primaries, f)
		}
	}

	var result []fileWithSidecars
	// Sort keys for deterministic order
	sort.Strings(keys)

	for _, key := range keys {
		g := groups[key]
		if len(g.primaries) == 0 {
			// Orphan sidecars become primaries
			for _, s := range g.sidecars {
				result = append(result, fileWithSidecars{Path: s})
			}
		} else {
			for _, p := range g.primaries {
				result = append(result, fileWithSidecars{
					Path:     p,
					Sidecars: g.sidecars,
				})
			}
		}
	}

	return result
}
