package render

import (
	"strings"
	"testing"

	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

func board(t *testing.T) *tree.Tree {
	t.Helper()

	tr := tree.New("Payment Gateway")
	add := func(n *tree.Node) {
		t.Helper()
		if err := tr.Add(n); err != nil {
			t.Fatalf("add %q: %v", n.ID, err)
		}
	}
	add(&tree.Node{ID: "mvp", Title: "MVP", Status: tree.InProgress})
	add(&tree.Node{ID: "auth", Title: "Authentication", Status: tree.InProgress, Parent: "mvp",
		Branch: "feat/auth", Issue: "12", Owner: "ann"})
	add(&tree.Node{ID: "reset", Title: "Password reset", Status: tree.Blocked, Parent: "mvp",
		Note: "waiting on SMTP"})
	add(&tree.Node{ID: "stripe", Title: "Stripe", Status: tree.Done, Parent: "mvp", PR: "44"})
	return tr
}

func TestBoardGroupsByStatus(t *testing.T) {
	out := Board(board(t), Filter{})

	for _, want := range []string{
		"in progress · 1",
		"blocked · 1",
		"done · 1",
		"Authentication",
		"feat/auth",
		"#12",
		"@ann",
		"— waiting on SMTP",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestBoardLeavesContainersOut(t *testing.T) {
	out := Board(board(t), Filter{})

	// A milestone may appear as a breadcrumb on its children, but never as a
	// card of its own — so look at what each listed line is *titled*.
	var titles []string
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "  ") {
			titles = append(titles, strings.TrimSpace(strings.SplitN(strings.TrimSpace(line), "  ", 2)[0]))
		}
	}

	if len(titles) != 3 {
		t.Fatalf("expected the three leaves, got %v:\n%s", titles, out)
	}
	for _, title := range titles {
		if title == "MVP" {
			t.Errorf("the milestone was listed as a card:\n%s", out)
		}
	}
	if !strings.Contains(out, "MVP") {
		t.Errorf("its children should still say where they belong:\n%s", out)
	}
}

func TestBoardShowsTheParentAsContext(t *testing.T) {
	tr := tree.New("x")
	for _, n := range []*tree.Node{
		{ID: "mvp", Title: "MVP"},
		{ID: "task", Title: "Plain task", Parent: "mvp"},
	} {
		if err := tr.Add(n); err != nil {
			t.Fatal(err)
		}
	}
	if !strings.Contains(Board(tr, Filter{}), "MVP") {
		t.Error("a task should carry the milestone it belongs to")
	}
}

func TestBoardFilter(t *testing.T) {
	out := Board(board(t), Filter{Status: tree.Blocked})
	if !strings.Contains(out, "Password reset") {
		t.Errorf("the blocked column should be shown:\n%s", out)
	}
	if strings.Contains(out, "Stripe") {
		t.Errorf("other columns should be filtered out:\n%s", out)
	}
}

func TestBoardAlignsMultiByteTitles(t *testing.T) {
	tr := tree.New("x")
	for _, n := range []*tree.Node{
		{ID: "a", Title: "Аутентификация", Owner: "ann"},
		{ID: "b", Title: "Short", Owner: "bob"},
	} {
		if err := tr.Add(n); err != nil {
			t.Fatal(err)
		}
	}

	out := Board(tr, Filter{})
	var columns []int
	for _, line := range strings.Split(out, "\n") {
		if i := strings.Index(line, "@"); i >= 0 {
			columns = append(columns, len([]rune(line[:i])))
		}
	}
	if len(columns) != 2 {
		t.Fatalf("expected two task lines, got %d:\n%s", len(columns), out)
	}
	// Padding counts runes, not bytes, or a Cyrillic title throws the column
	// out by the number of two-byte characters in it.
	if columns[0] != columns[1] {
		t.Errorf("columns do not line up: %v\n%s", columns, out)
	}
}

func TestBoardOfAnEmptyPlan(t *testing.T) {
	if out := Board(tree.New("x"), Filter{}); !strings.Contains(out, "nothing") {
		t.Errorf("got %q", out)
	}
}
