package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

const (
	// DirName and FileName sit next to the code they describe, so the plan is
	// versioned, reviewed in the pull request, and merged line by line.
	DirName  = ".devtree"
	FileName = "tree.yaml"
)

// ErrNotFound is returned when no .devtree directory exists above the working
// directory. Callers match on it to print the "run devtree init" hint.
var ErrNotFound = errors.New("no " + DirName + "/" + FileName + " found in this directory or any parent")

// Path is the location of the plan file for a given project root.
func Path(root string) string { return filepath.Join(root, DirName, FileName) }

// FindRoot walks up from start looking for .devtree/tree.yaml, the same way
// git finds .git. Running devtree from a subdirectory therefore just works.
func FindRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(Path(dir)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir { // hit the volume root
			return "", ErrNotFound
		}
		dir = parent
	}
}

// Exists reports whether a project root already holds a plan.
func Exists(root string) bool {
	_, err := os.Stat(Path(root))
	return err == nil
}

// Load reads, parses, and validates the plan. Errors are prefixed with the
// file path so a message pasted into chat still says where to look.
func Load(root string) (*tree.Tree, error) {
	data, err := os.ReadFile(Path(root))
	if err != nil {
		return nil, err
	}
	t, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", Path(root), err)
	}
	if err := t.Build(); err != nil {
		return nil, fmt.Errorf("%s: %w", Path(root), err)
	}
	return t, nil
}

// Save writes the plan atomically: a temp file in the same directory, then a
// rename. A crash mid-write can leave the old plan or the new one, never a
// half-written file that no longer parses.
func Save(root string, t *tree.Tree) error {
	dir := filepath.Join(root, DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	final := Path(root)
	tmp, err := os.CreateTemp(dir, FileName+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds

	if _, err := tmp.Write(Marshal(t)); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpName, final)
}
