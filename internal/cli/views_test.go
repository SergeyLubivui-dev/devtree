package cli

import (
	"strings"
	"testing"
)

// crew is a plan with owners and tags, so the filters have something to bite on.
func crew(t *testing.T) *harness {
	t.Helper()
	h := newHarness(t)
	h.run("init", "--project", "Payment Gateway", "--repo", "https://github.com/acme/pay",
		"--empty", "--outputs", "TREE.md")
	h.run("add", "MVP", "-s", "wip")
	h.run("add", "Authentication", "-p", "mvp", "-s", "wip", "-b", "feat/auth", "-i", "12", "-o", "ann", "--tags", "backend")
	h.run("add", "Password reset", "-p", "authentication", "-s", "blocked", "-n", "waiting on SMTP", "-o", "ann")
	h.run("add", "Stripe", "-p", "mvp", "-s", "done", "--pr", "44", "-o", "bob", "--tags", "backend,money")
	h.run("add", "Docs", "-o", "carol", "--tags", "writing")
	return h
}

func TestFiltersNarrowBothViews(t *testing.T) {
	h := crew(t)

	byOwner := h.run("board", "-o", "ann")
	if !strings.Contains(byOwner, "Password reset") || strings.Contains(byOwner, "Stripe") {
		t.Errorf("owner filter did not apply:\n%s", byOwner)
	}

	byTag := h.run("ls", "--tag", "writing")
	if !strings.Contains(byTag, "Docs") || strings.Contains(byTag, "Stripe") {
		t.Errorf("tag filter did not apply:\n%s", byTag)
	}

	// Filters combine: ann owns two tasks, only one of them is blocked.
	both := h.run("ls", "-o", "ann", "-s", "blocked")
	if !strings.Contains(both, "Password reset") || strings.Contains(both, "Authentication  ") {
		t.Errorf("combined filters did not apply:\n%s", both)
	}
}

func TestRootNarrowsToOneBranch(t *testing.T) {
	h := crew(t)

	out := h.run("ls", "--root", "authentication")
	if !strings.Contains(out, "Password reset") {
		t.Errorf("the subtree is missing:\n%s", out)
	}
	if strings.Contains(out, "Stripe") || strings.Contains(out, "Docs") {
		t.Errorf("the rest of the plan should be out of view:\n%s", out)
	}
	// The heading says where you are, without losing the project name.
	if !strings.Contains(out, "Payment Gateway · Authentication") {
		t.Errorf("heading was %q", strings.SplitN(out, "\n", 2)[0])
	}
}

func TestRootRejectsAnUnknownTask(t *testing.T) {
	h := crew(t)
	if msg := h.runExpectingFailure("board", "--root", "nope"); !strings.Contains(msg, "no such task") {
		t.Fatalf("got %q", msg)
	}
}

func TestRenderRootDrawsOneBranch(t *testing.T) {
	h := crew(t)
	h.run("render", "--root", "authentication", "--file", "docs/auth.svg")

	drawing := h.read("docs/auth.svg")
	if !strings.Contains(drawing, "Password reset") {
		t.Error("the subtree is missing from the drawing")
	}
	if strings.Contains(drawing, "Docs") {
		t.Error("the rest of the plan leaked into the drawing")
	}

	// A one-off override must not change what the plan says it writes.
	if out := h.run("outputs"); strings.TrimSpace(out) != "TREE.md" {
		t.Fatalf("outputs became %q", out)
	}
}

func TestOpenPrintsTheMostSpecificLink(t *testing.T) {
	h := crew(t)

	if got := strings.TrimSpace(h.run("open", "stripe", "--print")); got != "https://github.com/acme/pay/pull/44" {
		t.Errorf("got %q", got)
	}
	if got := strings.TrimSpace(h.run("open", "authentication", "--print")); got != "https://github.com/acme/pay/issues/12" {
		t.Errorf("got %q", got)
	}
	if got := strings.TrimSpace(h.run("open", "authentication", "--branch", "--print")); got != "https://github.com/acme/pay/tree/feat/auth" {
		t.Errorf("got %q", got)
	}
}

func TestOpenExplainsWhenThereIsNothingToOpen(t *testing.T) {
	h := crew(t)

	if msg := h.runExpectingFailure("open", "mvp", "--print"); !strings.Contains(msg, "points at nothing") {
		t.Errorf("got %q", msg)
	}
	if msg := h.runExpectingFailure("open", "stripe", "--issue", "--print"); !strings.Contains(msg, "no issue") {
		t.Errorf("got %q", msg)
	}
	if msg := h.runExpectingFailure("open", "nope", "--print"); !strings.Contains(msg, "no such task") {
		t.Errorf("got %q", msg)
	}
}

func TestOpenNeedsARepositoryURL(t *testing.T) {
	h := newHarness(t)
	h.run("init", "--empty", "--outputs", "TREE.md") // no --repo
	h.run("add", "Authentication", "-i", "12")

	msg := h.runExpectingFailure("open", "authentication", "--print")
	if !strings.Contains(msg, "repo URL") {
		t.Fatalf("got %q", msg)
	}
}
