package store

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

func TestSaveThenLoad(t *testing.T) {
	root := t.TempDir()

	if Exists(root) {
		t.Fatal("a fresh directory should not hold a plan")
	}
	if err := Save(root, sample(t)); err != nil {
		t.Fatalf("save: %v", err)
	}
	if !Exists(root) {
		t.Fatal("the plan should exist after Save")
	}

	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Project != "Payment Gateway" || len(loaded.Nodes) != 3 {
		t.Fatalf("loaded %q with %d nodes", loaded.Project, len(loaded.Nodes))
	}
	if _, ok := loaded.Get("auth"); !ok {
		t.Fatal("the tree was not rebuilt after loading")
	}
}

func TestSaveLeavesNoTempFiles(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 3; i++ {
		if err := Save(root, sample(t)); err != nil {
			t.Fatal(err)
		}
	}

	entries, err := os.ReadDir(filepath.Join(root, DirName))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != FileName {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("%s holds %v, want just %s", DirName, names, FileName)
	}
}

func TestFindRootWalksUp(t *testing.T) {
	root := t.TempDir()
	if err := Save(root, tree.New("x")); err != nil {
		t.Fatal(err)
	}

	deep := filepath.Join(root, "cmd", "server", "internal")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	found, err := FindRoot(deep)
	if err != nil {
		t.Fatalf("find root: %v", err)
	}
	// Compare the resolved paths: on macOS a temp dir is a symlink.
	want, _ := filepath.EvalSymlinks(root)
	got, _ := filepath.EvalSymlinks(found)
	if got != want {
		t.Fatalf("found %q, want %q", got, want)
	}
}

func TestFindRootReportsMissingPlan(t *testing.T) {
	_, err := FindRoot(t.TempDir())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound so the CLI can add its hint", err)
	}
}

func TestLoadRejectsAnInvalidPlan(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, DirName), 0o755); err != nil {
		t.Fatal(err)
	}
	// Parses fine, but the parent does not exist: Load must run Build too.
	body := "version: 1\nnodes:\n  - id: \"a\"\n    parent: \"ghost\"\n"
	if err := os.WriteFile(Path(root), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(root); err == nil {
		t.Fatal("expected the dangling parent to be rejected")
	}
}
