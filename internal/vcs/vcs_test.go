package vcs

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

// repo builds a throwaway repository with one merged branch and one that is
// still open, which is the only shape these tests care about.
func repo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}

	run("init", "-q", "-b", "main")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	run("config", "commit.gpgsign", "false")
	run("commit", "-q", "--allow-empty", "-m", "init")

	run("switch", "-qc", "feat/merged")
	run("commit", "-q", "--allow-empty", "-m", "work")
	run("switch", "-q", "main")
	run("merge", "-q", "--no-ff", "feat/merged", "-m", "merge")

	run("switch", "-qc", "feat/open")
	run("commit", "-q", "--allow-empty", "-m", "more work")
	run("switch", "-q", "main")

	return dir
}

func TestMergedBranches(t *testing.T) {
	branches, err := MergedBranches(repo(t))
	if err != nil {
		t.Fatal(err)
	}

	got := strings.Join(branches, ",")
	if !strings.Contains(got, "feat/merged") {
		t.Errorf("the merged branch is missing from %q", got)
	}
	if strings.Contains(got, "feat/open") {
		t.Errorf("an unmerged branch was reported as merged: %q", got)
	}
	// HEAD's own branch is always "merged" into itself and would close every
	// task pointing at the trunk.
	if strings.Contains(got, "main") {
		t.Errorf("the current branch should be excluded, got %q", got)
	}
}

func TestCurrentBranch(t *testing.T) {
	name, err := CurrentBranch(repo(t))
	if err != nil {
		t.Fatal(err)
	}
	if name != "main" {
		t.Errorf("got %q, want main", name)
	}
}

func TestOutsideARepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	_, err := MergedBranches(t.TempDir())
	if !errors.Is(err, ErrNotARepository) {
		t.Fatalf("got %v, want ErrNotARepository so the CLI can explain itself", err)
	}
}
