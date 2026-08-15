package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergeyLubivui-dev/devtree/internal/store"
)

// committed builds a project whose plan has a past: three commits, each one
// further along than the last.
func committed(t *testing.T) *harness {
	t.Helper()
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

	h.run("init", "--project", "Payment Gateway", "--empty", "--outputs", "TREE.md")
	h.run("add", "MVP", "-s", "wip")
	h.run("add", "Authentication", "-p", "mvp")
	git("add", ".")
	git("commit", "-q", "-m", "plan: start")

	h.run("add", "Payments", "-p", "mvp")
	h.run("done", "authentication")
	git("add", ".")
	git("commit", "-q", "-m", "plan: authentication done")

	h.run("add", "Stripe", "-p", "payments", "-s", "done")
	git("add", ".")
	git("commit", "-q", "-m", "plan: stripe done")

	return h
}

func TestHistoryReadsThePlansOwnPast(t *testing.T) {
	h := committed(t)

	out := h.run("history")
	if !strings.Contains(out, "3 revision(s)") {
		t.Errorf("expected three revisions:\n%s", out)
	}
	for _, want := range []string{"0/2", "1/3", "2/4", "+1 done", "+1 planned"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "up 2 since") {
		t.Errorf("the summary should say how much moved:\n%s", out)
	}
}

func TestHistoryRespectsTheLimit(t *testing.T) {
	h := committed(t)
	if out := h.run("history", "--limit", "1"); !strings.Contains(out, "1 revision(s)") {
		t.Errorf("--limit was ignored:\n%s", out)
	}
}

func TestHistorySkipsRevisionsItCannotRead(t *testing.T) {
	h := committed(t)

	// A revision written by an older devtree may carry a field this one does
	// not know. One unreadable point is not a reason to refuse the history.
	broken := "version: 1\nproject: \"x\"\nnodes:\n  - id: \"a\"\n    assignee: \"ann\"\n"
	if err := os.WriteFile(store.Path(h.dir), []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "."}, {"commit", "-q", "-m", "plan: from the future"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = h.dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git: %v\n%s", err, out)
		}
	}
	// Put a readable plan back so the command can load the current one.
	h.app.Execute([]string{"init"}) // no-op, the plan file already exists
	if err := os.WriteFile(store.Path(h.dir),
		[]byte("version: 1\nproject: \"x\"\nnodes:\n  - id: \"a\"\n    title: \"A\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := h.run("history")
	if !strings.Contains(h.err.String(), "skipped") {
		t.Errorf("the reader should say it skipped something, got %q", h.err.String())
	}
	if !strings.Contains(out, "revision(s) of the plan") {
		t.Errorf("the rest of the history should still be shown:\n%s", out)
	}
}

func TestHistoryNeedsARepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	h := newHarness(t)
	h.run("init", "--empty")

	if msg := h.runExpectingFailure("history"); !strings.Contains(msg, "git repository") {
		t.Fatalf("got %q", msg)
	}
}

func TestHistoryOfAnUncommittedPlan(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	h := newHarness(t)
	cmd := exec.Command("git", "init", "-q", "-b", "main")
	cmd.Dir = h.dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	h.run("init", "--empty", "--outputs", "TREE.md")

	msg := h.runExpectingFailure("history")
	if !strings.Contains(msg, "never been committed") {
		t.Fatalf("got %q", msg)
	}
	if !strings.Contains(msg, filepath.ToSlash(filepath.Join(store.DirName, store.FileName))) {
		t.Errorf("the message should name the file, got %q", msg)
	}
}
