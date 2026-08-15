package store

import (
	"os"
	"strings"
	"testing"

	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

func TestArchiveRoundTrip(t *testing.T) {
	root := t.TempDir()

	if archive, err := LoadArchive(root); err != nil || len(archive.Nodes) != 0 {
		t.Fatalf("a project with no archive should load empty, got %v / %v", archive, err)
	}

	archive := &tree.Tree{Version: 1, Project: "Payment Gateway"}
	archive.Nodes = []*tree.Node{
		{ID: "stripe", Title: "Stripe", Status: tree.Done, Parent: "pay", PR: "44"},
		{ID: "old", Title: "Old idea", Status: tree.Dropped},
	}
	if err := SaveArchive(root, archive); err != nil {
		t.Fatal(err)
	}

	back, err := LoadArchive(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(back.Nodes) != 2 || back.Nodes[0].PR != "44" {
		t.Fatalf("the archive came back as %+v", back.Nodes)
	}
}

func TestArchiveKeepsDanglingParents(t *testing.T) {
	root := t.TempDir()

	// An archived task normally keeps a parent that is still in the active
	// plan — that is how it remembers where it came from. Validating it as a
	// tree would reject a perfectly good record.
	archive := &tree.Tree{Version: 1}
	archive.Nodes = []*tree.Node{{ID: "stripe", Title: "Stripe", Status: tree.Done, Parent: "still-in-the-plan"}}
	if err := SaveArchive(root, archive); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadArchive(root); err != nil {
		t.Fatalf("an archived task with a live parent should load fine: %v", err)
	}
}

func TestArchiveStillRejectsNonsense(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(ArchivePath(root)[:strings.LastIndex(ArchivePath(root), string(os.PathSeparator))], 0o755); err != nil {
		t.Fatal(err)
	}

	body := "version: 1\nnodes:\n  - id: \"a\"\n  - id: \"a\"\n"
	if err := os.WriteFile(ArchivePath(root), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadArchive(root); err == nil {
		t.Fatal("a duplicate id in the archive should still be an error")
	}
}

func TestEmptyArchiveIsRemoved(t *testing.T) {
	root := t.TempDir()

	archive := &tree.Tree{Version: 1}
	archive.Nodes = []*tree.Node{{ID: "a", Title: "A", Status: tree.Done}}
	if err := SaveArchive(root, archive); err != nil {
		t.Fatal(err)
	}

	archive.Nodes = nil
	if err := SaveArchive(root, archive); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ArchivePath(root)); !os.IsNotExist(err) {
		t.Error("an emptied archive should leave no file behind")
	}
}
