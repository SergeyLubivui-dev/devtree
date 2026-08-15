package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergeyLubivui-dev/devtree/internal/store"
)

// harness runs commands against a temporary project and hands back whatever
// they printed. Because App takes its directory and writers as fields, a whole
// command can be exercised without a subprocess.
type harness struct {
	t    *testing.T
	dir  string
	out  bytes.Buffer
	err  bytes.Buffer
	app  *App
	code int
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{t: t, dir: t.TempDir()}
	h.app = &App{Dir: h.dir, Out: &h.out, Err: &h.err}
	return h
}

// run executes one command and returns stdout. Failures are reported here so
// the tests read as a sequence of steps rather than a wall of error checks.
func (h *harness) run(args ...string) string {
	h.t.Helper()
	h.out.Reset()
	h.err.Reset()
	h.code = h.app.Execute(args)
	if h.code != 0 {
		h.t.Fatalf("devtree %s exited %d: %s", strings.Join(args, " "), h.code, h.err.String())
	}
	return h.out.String()
}

// runExpectingFailure is the mirror image: the command must fail.
func (h *harness) runExpectingFailure(args ...string) string {
	h.t.Helper()
	h.out.Reset()
	h.err.Reset()
	h.code = h.app.Execute(args)
	if h.code == 0 {
		h.t.Fatalf("devtree %s was expected to fail, but it succeeded: %s", strings.Join(args, " "), h.out.String())
	}
	return h.err.String()
}

func (h *harness) read(name string) string {
	h.t.Helper()
	data, err := os.ReadFile(filepath.Join(h.dir, name))
	if err != nil {
		h.t.Fatalf("read %s: %v", name, err)
	}
	return string(data)
}

func TestInitCreatesTheProject(t *testing.T) {
	h := newHarness(t)
	h.run("init", "--project", "Payment Gateway", "--repo", "https://github.com/acme/pay/")

	plan := h.read(filepath.Join(store.DirName, store.FileName))
	if !strings.Contains(plan, `project: "Payment Gateway"`) {
		t.Errorf("project name missing from the plan:\n%s", plan)
	}
	if !strings.Contains(plan, `repo: "https://github.com/acme/pay"`) {
		t.Errorf("the trailing slash should be trimmed from the repo URL:\n%s", plan)
	}

	if attrs := h.read(".gitattributes"); !strings.Contains(attrs, "merge=union") {
		t.Errorf("the merge rule is missing:\n%s", attrs)
	}
	if diagram := h.read("TREE.md"); !strings.Contains(diagram, "```mermaid") {
		t.Errorf("the first diagram was not rendered:\n%s", diagram)
	}
}

func TestInitRefusesToOverwrite(t *testing.T) {
	h := newHarness(t)
	h.run("init")

	if msg := h.runExpectingFailure("init"); !strings.Contains(msg, "already exists") {
		t.Fatalf("got %q", msg)
	}
}

func TestInitEmptySkipsTheSamples(t *testing.T) {
	h := newHarness(t)
	h.run("init", "--empty")

	if plan := h.read(filepath.Join(store.DirName, store.FileName)); strings.Contains(plan, "mvp") {
		t.Fatalf("--empty should not seed anything:\n%s", plan)
	}
}

func TestAddAcceptsFlagsAfterTheTitle(t *testing.T) {
	h := newHarness(t)
	h.run("init", "--empty")
	h.run("add", "Authentication", "-s", "wip")
	h.run("add", "OAuth providers", "-p", "authentication", "-b", "feat/oauth", "-i", "#12", "-o", "@ann")

	plan := h.read(filepath.Join(store.DirName, store.FileName))
	for _, want := range []string{
		`id: "authentication"`,
		`status: "in_progress"`,
		`id: "oauth-providers"`,
		`parent: "authentication"`,
		`branch: "feat/oauth"`,
		`issue: "12"`,  // "#" trimmed
		`owner: "ann"`, // "@" trimmed
	} {
		if !strings.Contains(plan, want) {
			t.Errorf("missing %s in:\n%s", want, plan)
		}
	}
}

func TestAddRejectsAMissingParent(t *testing.T) {
	h := newHarness(t)
	h.run("init", "--empty")

	if msg := h.runExpectingFailure("add", "Orphan", "-p", "nowhere"); !strings.Contains(msg, "no such parent") {
		t.Fatalf("got %q", msg)
	}
}

func TestAddNeedsATitle(t *testing.T) {
	h := newHarness(t)
	h.run("init", "--empty")

	if msg := h.runExpectingFailure("add"); !strings.Contains(msg, "title") {
		t.Fatalf("got %q", msg)
	}
}

func TestSetChangesOnlyTheFlagsGiven(t *testing.T) {
	h := newHarness(t)
	h.run("init", "--empty")
	h.run("add", "Authentication", "-b", "feat/auth", "-o", "ann")
	h.run("set", "authentication", "-s", "blocked", "-n", "waiting on SMTP")

	plan := h.read(filepath.Join(store.DirName, store.FileName))
	if !strings.Contains(plan, `status: "blocked"`) || !strings.Contains(plan, `note: "waiting on SMTP"`) {
		t.Errorf("the change was not applied:\n%s", plan)
	}
	if !strings.Contains(plan, `branch: "feat/auth"`) || !strings.Contains(plan, `owner: "ann"`) {
		t.Errorf("untouched fields were cleared:\n%s", plan)
	}
}

func TestSetCanClearAField(t *testing.T) {
	h := newHarness(t)
	h.run("init", "--empty")
	h.run("add", "Authentication", "-b", "feat/auth")
	h.run("set", "authentication", "-b", "")

	if plan := h.read(filepath.Join(store.DirName, store.FileName)); strings.Contains(plan, "branch:") {
		t.Fatalf("an explicitly empty flag should clear the field:\n%s", plan)
	}
}

func TestDoneMarksSeveralTasks(t *testing.T) {
	h := newHarness(t)
	h.run("init", "--empty")
	h.run("add", "One")
	h.run("add", "Two")
	h.run("done", "one", "two")

	plan := h.read(filepath.Join(store.DirName, store.FileName))
	if got := strings.Count(plan, `status: "done"`); got != 2 {
		t.Fatalf("%d tasks are done, want 2:\n%s", got, plan)
	}
}

func TestMoveAndRemove(t *testing.T) {
	h := newHarness(t)
	h.run("init", "--empty")
	h.run("add", "Milestone")
	h.run("add", "Feature", "-p", "milestone")
	h.run("add", "Detail", "-p", "feature")

	h.run("mv", "detail", "milestone")
	if plan := h.read(filepath.Join(store.DirName, store.FileName)); !strings.Contains(plan, `parent: "milestone"`) {
		t.Errorf("the move was not saved:\n%s", plan)
	}

	// Without --cascade the children move up rather than disappearing.
	h.run("add", "Sub", "-p", "feature")
	h.run("rm", "feature")
	plan := h.read(filepath.Join(store.DirName, store.FileName))
	if strings.Contains(plan, `id: "feature"`) {
		t.Errorf("the node was not removed:\n%s", plan)
	}
	if !strings.Contains(plan, `id: "sub"`) {
		t.Errorf("the child should have been re-parented, not deleted:\n%s", plan)
	}

	h.run("rm", "milestone", "--cascade")
	if plan := h.read(filepath.Join(store.DirName, store.FileName)); strings.Contains(plan, `id: "sub"`) {
		t.Errorf("--cascade should take the whole subtree:\n%s", plan)
	}
}

func TestRenderIsIdempotentOnDisk(t *testing.T) {
	h := newHarness(t)
	h.run("init", "--project", "Demo")

	first := h.read("TREE.md")
	out := h.run("render")
	if !strings.Contains(out, "unchanged") {
		t.Errorf("a second render should report no change, got %q", out)
	}
	if again := h.read("TREE.md"); again != first {
		t.Errorf("rendering twice rewrote the file: %d bytes → %d", len(first), len(again))
	}
}

func TestRenderIntoAnExistingReadme(t *testing.T) {
	h := newHarness(t)
	h.run("init", "--project", "Demo", "--outputs", "README.md")

	// The plan renders into a README that already has content; the author's
	// prose has to survive, above and below.
	readme := "# Demo\n\nA short description.\n"
	if err := os.WriteFile(filepath.Join(h.dir, "README.md"), []byte(readme), 0o644); err != nil {
		t.Fatal(err)
	}

	h.run("render")
	h.run("add", "Another task", "-p", "mvp")

	got := h.read("README.md")
	if !strings.HasPrefix(got, "# Demo") {
		t.Errorf("the original README was overwritten:\n%s", got)
	}
	if n := strings.Count(got, "<!-- devtree:begin -->"); n != 1 {
		t.Errorf("the block was written %d times, want 1:\n%s", n, got)
	}
	if !strings.Contains(got, "Another task") {
		t.Errorf("the new task is missing from the diagram:\n%s", got)
	}
}

func TestRenderFileOverrideIsNotSaved(t *testing.T) {
	h := newHarness(t)
	h.run("init", "--project", "Demo")
	h.run("render", "--file", "PLAN.md")

	if _, err := os.Stat(filepath.Join(h.dir, "PLAN.md")); err != nil {
		t.Fatalf("the override file was not written: %v", err)
	}
	if out := h.run("outputs"); strings.TrimSpace(out) != "TREE.md" {
		t.Fatalf("the override leaked into the saved plan: %q", out)
	}
}

func TestCheckWarnsWithoutFailing(t *testing.T) {
	h := newHarness(t)
	h.run("init", "--empty")
	h.run("add", "Parent", "-s", "done")
	h.run("add", "Child", "-p", "parent", "-s", "todo")

	h.run("check")
	if !strings.Contains(h.err.String(), "still not started") {
		t.Errorf("expected a warning about the unfinished child, got %q", h.err.String())
	}

	if msg := h.runExpectingFailure("check", "--strict"); !strings.Contains(msg, "warning") {
		t.Fatalf("--strict should turn warnings into a failure, got %q", msg)
	}
}

func TestCheckReportsAHandEditedPlanWithTheLineNumber(t *testing.T) {
	h := newHarness(t)
	h.run("init", "--empty")

	broken := "version: 1\nproject: \"x\"\nnodes:\n  - id: \"a\"\n    status: \"nearly\"\n"
	if err := os.WriteFile(store.Path(h.dir), []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}

	msg := h.runExpectingFailure("check")
	if !strings.Contains(msg, "line 5") {
		t.Fatalf("the error should point at the line, got %q", msg)
	}
}

func TestCommandsWithoutAPlan(t *testing.T) {
	h := newHarness(t)
	if msg := h.runExpectingFailure("ls"); !strings.Contains(msg, "devtree init") {
		t.Fatalf("the error should suggest init, got %q", msg)
	}
}

func TestListShowsTheTree(t *testing.T) {
	h := newHarness(t)
	h.run("init", "--project", "Demo")
	h.run("add", "Authentication", "-p", "mvp", "-s", "wip")

	out := h.run("ls")
	for _, want := range []string{"Demo", "MVP", "Authentication", "0/3"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	filtered := h.run("ls", "-s", "in_progress")
	if strings.Contains(filtered, "First task") {
		t.Errorf("the filter did not apply:\n%s", filtered)
	}
}

func TestUnknownCommandAndFlagAreUsageErrors(t *testing.T) {
	h := newHarness(t)

	if code := h.app.Execute([]string{"frobnicate"}); code != 2 {
		t.Errorf("unknown command exited %d, want 2", code)
	}

	h.run("init", "--empty")
	if code := h.app.Execute([]string{"add", "Task", "--nope"}); code != 1 {
		t.Errorf("unknown flag exited %d, want 1", code)
	}
}

func TestHelpAndVersion(t *testing.T) {
	h := newHarness(t)

	if out := h.run("version"); !strings.Contains(out, Version) {
		t.Errorf("version output was %q", out)
	}
	if out := h.run("help"); !strings.Contains(out, "USAGE") {
		t.Errorf("help output was %q", out)
	}
	if code := h.app.Execute(nil); code != 2 {
		t.Errorf("no arguments exited %d, want 2", code)
	}
}

func TestInstallHookAndAction(t *testing.T) {
	h := newHarness(t)
	h.run("init", "--empty")

	// No .git directory yet: installing the hook has to say so plainly.
	if msg := h.runExpectingFailure("install", "hook"); !strings.Contains(msg, "git init") {
		t.Fatalf("got %q", msg)
	}

	if err := os.MkdirAll(filepath.Join(h.dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	h.run("install", "all")

	if _, err := os.Stat(filepath.Join(h.dir, ".git", "hooks", "pre-commit")); err != nil {
		t.Errorf("the hook was not installed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(h.dir, ".github", "workflows", "devtree.yml")); err != nil {
		t.Errorf("the workflow was not installed: %v", err)
	}
}

func TestRunFromASubdirectory(t *testing.T) {
	h := newHarness(t)
	h.run("init", "--empty")
	h.run("add", "Task")

	// Same project, but invoked from deeper in the tree, the way a developer
	// actually works.
	sub := filepath.Join(h.dir, "internal", "server")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	deep := &App{Dir: sub, Out: &h.out, Err: &h.err}

	h.out.Reset()
	if code := deep.Execute([]string{"ls"}); code != 0 {
		t.Fatalf("ls from a subdirectory exited %d: %s", code, h.err.String())
	}
	if !strings.Contains(h.out.String(), "Task") {
		t.Fatalf("the plan was not found from a subdirectory:\n%s", h.out.String())
	}
}
