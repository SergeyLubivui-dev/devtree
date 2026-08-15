package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergeyLubivui-dev/devtree/internal/store"
)

// finishedPlan is a plan with one finished milestone, one finished leaf under
// an active milestone, and one task still open.
func finishedPlan(t *testing.T) *harness {
	t.Helper()
	h := newHarness(t)
	h.run("init", "--empty", "--outputs", "TREE.md")
	h.run("add", "v1", "-s", "done")
	h.run("add", "Login", "-p", "v1", "-s", "done")
	h.run("add", "Signup", "-p", "v1", "-s", "dropped")
	h.run("add", "v2", "-s", "wip")
	h.run("add", "Search", "-p", "v2", "-s", "done")
	h.run("add", "Filters", "-p", "v2")
	return h
}

func TestArchiveSaysWhatItWouldDoBeforeDoingIt(t *testing.T) {
	h := finishedPlan(t)

	out := h.run("archive")
	if !strings.Contains(out, "v1") || !strings.Contains(out, "Search") {
		t.Errorf("both finished branches should be listed:\n%s", out)
	}
	if strings.Contains(out, "Filters") {
		t.Errorf("unfinished work must not be offered for archiving:\n%s", out)
	}
	if !strings.Contains(out, "--all") {
		t.Error("the dry run should say how to go through with it")
	}

	// Nothing moved.
	if _, err := os.Stat(store.ArchivePath(h.dir)); !os.IsNotExist(err) {
		t.Fatal("the dry run wrote an archive file")
	}
	if plan := h.read(filepath.Join(store.DirName, store.FileName)); !strings.Contains(plan, `id: "v1"`) {
		t.Fatal("the dry run removed something from the plan")
	}
}

func TestArchiveAllMovesFinishedWorkOnly(t *testing.T) {
	h := finishedPlan(t)
	h.run("archive", "--all")

	plan := h.read(filepath.Join(store.DirName, store.FileName))
	for _, gone := range []string{`id: "v1"`, `id: "login"`, `id: "signup"`, `id: "search"`} {
		if strings.Contains(plan, gone) {
			t.Errorf("%s should have left the plan:\n%s", gone, plan)
		}
	}
	for _, kept := range []string{`id: "v2"`, `id: "filters"`} {
		if !strings.Contains(plan, kept) {
			t.Errorf("%s must stay in the plan:\n%s", kept, plan)
		}
	}

	archive := h.read(filepath.Join(store.DirName, store.ArchiveFileName))
	for _, want := range []string{`id: "v1"`, `id: "signup"`, `id: "search"`, `parent: "v2"`} {
		if !strings.Contains(archive, want) {
			t.Errorf("%s is missing from the archive:\n%s", want, archive)
		}
	}

	// The record is still readable, and the diagram no longer carries the tail.
	if out := h.run("archive", "--list"); !strings.Contains(out, "4 task(s) archived") {
		t.Errorf("got %q", out)
	}
	if diagram := h.read("TREE.md"); strings.Contains(diagram, "Login") {
		t.Error("archived work should be gone from the rendered diagram")
	}
}

func TestArchiveRefusesToTakeLiveWorkWithIt(t *testing.T) {
	h := finishedPlan(t)

	msg := h.runExpectingFailure("archive", "v2")
	if !strings.Contains(msg, "unfinished") {
		t.Fatalf("got %q", msg)
	}
	if _, err := os.Stat(store.ArchivePath(h.dir)); !os.IsNotExist(err) {
		t.Fatal("a refused archive should write nothing")
	}
}

func TestArchiveOneBranchByName(t *testing.T) {
	h := finishedPlan(t)
	h.run("archive", "v1")

	plan := h.read(filepath.Join(store.DirName, store.FileName))
	if strings.Contains(plan, `id: "v1"`) {
		t.Error("the named branch should have moved")
	}
	if !strings.Contains(plan, `id: "search"`) {
		t.Error("only the named branch should have moved")
	}
}

func TestRestoreBringsTheSubtreeBack(t *testing.T) {
	h := finishedPlan(t)
	h.run("archive", "--all")
	h.run("restore", "v1")

	plan := h.read(filepath.Join(store.DirName, store.FileName))
	for _, want := range []string{`id: "v1"`, `id: "login"`, `id: "signup"`} {
		if !strings.Contains(plan, want) {
			t.Errorf("%s did not come back:\n%s", want, plan)
		}
	}
	if out := h.run("archive", "--list"); !strings.Contains(out, "1 task(s) archived") {
		t.Errorf("only Search should be left in the archive, got %q", out)
	}
}

func TestRestoreRootsAnOrphan(t *testing.T) {
	h := finishedPlan(t)
	h.run("archive", "--all")

	// Search's parent is v2, which is still in the plan... so remove it first
	// and check the task comes back rather than being refused.
	h.run("rm", "v2", "--cascade")
	h.run("restore", "search")

	if !strings.Contains(h.err.String(), "came back as a root") {
		t.Errorf("the user should be told the parent is gone, got %q", h.err.String())
	}
	plan := h.read(filepath.Join(store.DirName, store.FileName))
	if !strings.Contains(plan, `id: "search"`) {
		t.Error("the task should be back in the plan")
	}
	if strings.Contains(plan, `parent: "v2"`) {
		t.Error("it should not point at a parent that no longer exists")
	}
}

func TestRestoreRejectsAnUnknownID(t *testing.T) {
	h := finishedPlan(t)
	if msg := h.runExpectingFailure("restore", "nope"); !strings.Contains(msg, "archive --list") {
		t.Fatalf("got %q", msg)
	}
}

func TestSyncNeedsARepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	h := newHarness(t)
	h.run("init", "--empty")

	if msg := h.runExpectingFailure("sync"); !strings.Contains(msg, "git repository") {
		t.Fatalf("got %q", msg)
	}
}

func TestSyncProposesThenApplies(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	h := newHarness(t)
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = h.dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	git("init", "-q", "-b", "main")
	git("config", "user.email", "test@example.com")
	git("config", "user.name", "Test")
	git("config", "commit.gpgsign", "false")
	git("commit", "-q", "--allow-empty", "-m", "init")
	git("switch", "-qc", "feat/auth")
	git("commit", "-q", "--allow-empty", "-m", "auth")
	git("switch", "-q", "main")
	git("merge", "-q", "--no-ff", "feat/auth", "-m", "merge")
	git("switch", "-qc", "feat/pay")
	git("commit", "-q", "--allow-empty", "-m", "pay")
	git("switch", "-q", "main")

	h.run("init", "--empty", "--outputs", "TREE.md")
	h.run("add", "Authentication", "-b", "feat/auth", "-s", "wip")
	h.run("add", "Payments", "-b", "feat/pay", "-s", "wip")

	out := h.run("sync")
	if !strings.Contains(out, "feat/auth") || strings.Contains(out, "feat/pay") {
		t.Errorf("only the merged branch should be proposed:\n%s", out)
	}
	if !strings.Contains(out, "--apply") {
		t.Error("the dry run should say how to go through with it")
	}
	if plan := h.read(filepath.Join(store.DirName, store.FileName)); strings.Contains(plan, `status: "done"`) {
		t.Error("a dry run must not change anything")
	}

	h.run("sync", "--apply")
	plan := h.read(filepath.Join(store.DirName, store.FileName))
	if strings.Count(plan, `status: "done"`) != 1 {
		t.Errorf("exactly the merged task should be done:\n%s", plan)
	}
}
