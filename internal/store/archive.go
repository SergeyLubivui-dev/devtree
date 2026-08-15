package store

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

// ArchiveFileName holds finished work that has left the active plan.
//
// It uses the very same format as tree.yaml, so there is no second parser, no
// second serializer, and no second thing to keep in sync. The only difference
// is in how strictly it is read: see LoadArchive.
const ArchiveFileName = "archive.yaml"

// ArchivePath is the location of the archive for a project root.
func ArchivePath(root string) string { return filepath.Join(root, DirName, ArchiveFileName) }

// LoadArchive reads the archive without demanding that it be a valid tree.
//
// An archived task usually keeps a parent that is still in the active plan —
// that is the whole point, it remembers where it came from — so running the
// tree validation over it would reject a perfectly good record. The archive is
// history, not a plan: it is parsed, not built.
func LoadArchive(root string) (*tree.Tree, error) {
	data, err := os.ReadFile(ArchivePath(root))
	if os.IsNotExist(err) {
		return &tree.Tree{Version: 1}, nil // no archive yet is not an error
	}
	if err != nil {
		return nil, err
	}

	t, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ArchivePath(root), err)
	}

	seen := make(map[string]bool, len(t.Nodes))
	for _, n := range t.Nodes {
		if n.ID == "" {
			return nil, fmt.Errorf("%s: archived node without an id", ArchivePath(root))
		}
		if seen[n.ID] {
			return nil, fmt.Errorf("%s: duplicate id %q", ArchivePath(root), n.ID)
		}
		seen[n.ID] = true
	}
	return t, nil
}

// SaveArchive writes the archive, atomically, the same way the plan is written.
// An archive with nothing in it is removed rather than left as an empty file.
func SaveArchive(root string, t *tree.Tree) error {
	if len(t.Nodes) == 0 {
		err := os.Remove(ArchivePath(root))
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	return writeAtomic(ArchivePath(root), Marshal(t))
}
