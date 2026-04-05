package transfer

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Action describes what happened (or would happen) during a transfer.
type Action string

const (
	ActionCopied   Action = "copied"
	ActionMoved    Action = "moved"
	ActionSkipped  Action = "skipped"
	ActionReplaced Action = "replaced"

	ActionWouldCopy    Action = "would_copy"
	ActionWouldMove    Action = "would_move"
	ActionWouldReplace Action = "would_replace"
)

// Options configures the transfer behaviour.
type Options struct {
	Move   bool
	DryRun bool
}

// TransferFile copies or moves source to target with paranoid hash verification.
func TransferFile(source, target string, opts Options) (Action, error) {
	absSrc, err := filepath.Abs(source)
	if err != nil {
		return "", fmt.Errorf("resolve source path: %w", err)
	}

	absDst, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("resolve target path: %w", err)
	}

	// Same path → skip
	if absSrc == absDst {
		return ActionSkipped, nil
	}

	// Validate source
	srcInfo, err := os.Stat(source)
	if err != nil {
		return "", fmt.Errorf("stat source: %w", err)
	}
	if srcInfo.IsDir() {
		return "", errors.New("source is a directory")
	}

	// Check if target exists
	_, statErr := os.Stat(target)
	targetExists := statErr == nil

	if targetExists {
		identical, err := CompareFiles(source, target)
		if err != nil {
			return "", fmt.Errorf("compare files: %w", err)
		}

		if identical {
			if opts.Move {
				if opts.DryRun {
					return ActionWouldMove, nil
				}
				if err := os.Remove(source); err != nil {
					return "", fmt.Errorf("remove source: %w", err)
				}
				return ActionMoved, nil
			}
			return ActionSkipped, nil
		}

		// Different content → replace
		if opts.DryRun {
			return ActionWouldReplace, nil
		}

		if err := os.Remove(target); err != nil {
			return "", fmt.Errorf("remove target: %w", err)
		}

		if err := copyFile(source, target); err != nil {
			return "", err
		}

		if opts.Move {
			if err := os.Remove(source); err != nil {
				return "", fmt.Errorf("remove source: %w", err)
			}
		}

		return ActionReplaced, nil
	}

	// Target doesn't exist → copy/move
	if opts.DryRun {
		if opts.Move {
			return ActionWouldMove, nil
		}
		return ActionWouldCopy, nil
	}

	if err := copyFile(source, target); err != nil {
		return "", err
	}

	if opts.Move {
		if err := os.Remove(source); err != nil {
			return "", fmt.Errorf("remove source: %w", err)
		}
		return ActionMoved, nil
	}

	return ActionCopied, nil
}

// CompareFiles returns true if both files have identical content.
// It checks sizes first for an early exit, then compares SHA-256 hashes.
func CompareFiles(a, b string) (bool, error) {
	infoA, err := os.Stat(a)
	if err != nil {
		return false, fmt.Errorf("stat %s: %w", a, err)
	}

	infoB, err := os.Stat(b)
	if err != nil {
		return false, fmt.Errorf("stat %s: %w", b, err)
	}

	if infoA.Size() != infoB.Size() {
		return false, nil
	}

	hashA, err := fileHash(a)
	if err != nil {
		return false, err
	}

	hashB, err := fileHash(b)
	if err != nil {
		return false, err
	}

	return hashA == hashB, nil
}

// fileHash returns the hex-encoded SHA-256 hash of the file at path.
func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// copyFile copies source to target, creating parent directories as needed.
func copyFile(source, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create target dir: %w", err)
	}

	src, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("create target: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy data: %w", err)
	}

	return dst.Close()
}
