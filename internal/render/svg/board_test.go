package svg

import (
	"strings"
	"testing"

	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

func TestBoardIsWellFormed(t *testing.T) {
	for _, th := range []Theme{Light, Dark} {
		t.Run(th.Name, func(t *testing.T) {
			wellFormed(t, Board(sample(t), th))
		})
	}
}

func TestBoardShowsWorkNotContainers(t *testing.T) {
	doc := Board(sample(t), Light)

	// mvp and pay have children: they are milestones, and a board that lists
	// milestones next to the tasks inside them stops being a board.
	for _, container := range []string{">MVP<", ">Payments<"} {
		if strings.Contains(doc, container) {
			t.Errorf("container %q was drawn as a card", container)
		}
	}
	for _, task := range []string{"Authentication", "Instant transfers", "Old idea"} {
		if !strings.Contains(doc, task) {
			t.Errorf("task %q is missing from the board", task)
		}
	}
}

func TestBoardKeepsTheParentAsABreadcrumb(t *testing.T) {
	tr := tree.New("x")
	add := func(n *tree.Node) {
		t.Helper()
		if err := tr.Add(n); err != nil {
			t.Fatal(err)
		}
	}
	add(&tree.Node{ID: "mvp", Title: "MVP"})
	add(&tree.Node{ID: "plain", Title: "Plain task", Parent: "mvp"})
	add(&tree.Node{ID: "rich", Title: "Rich task", Parent: "mvp", Branch: "feat/x"})

	doc := Board(tr, Light)
	if !strings.Contains(doc, ">MVP<") {
		t.Error("a task with no metadata should show the milestone it belongs to")
	}
	if !strings.Contains(doc, "feat/x") {
		t.Error("a task with metadata should show it instead")
	}
}

func TestBoardOnlyDrawsColumnsThatHaveWork(t *testing.T) {
	tr := tree.New("x")
	for _, n := range []*tree.Node{
		{ID: "a", Title: "A", Status: tree.Todo},
		{ID: "b", Title: "B", Status: tree.Done},
	} {
		if err := tr.Add(n); err != nil {
			t.Fatal(err)
		}
	}

	doc := Board(tr, Light)
	if !strings.Contains(doc, "not started") || !strings.Contains(doc, ">done<") {
		t.Error("the two occupied columns should both be drawn")
	}
	for _, absent := range []string{"in progress", "blocked", "dropped"} {
		if strings.Contains(doc, ">"+absent+"<") {
			t.Errorf("empty column %q should not take up width", absent)
		}
	}
}

func TestBoardCountsEachColumn(t *testing.T) {
	tr := tree.New("x")
	for _, id := range []string{"a", "b", "c"} {
		if err := tr.Add(&tree.Node{ID: id, Title: strings.ToUpper(id), Status: tree.Todo}); err != nil {
			t.Fatal(err)
		}
	}
	if !strings.Contains(Board(tr, Light), ">3<") {
		t.Error("the column heading should carry the number of cards")
	}
}

func TestBoardIsDeterministic(t *testing.T) {
	first := Board(sample(t), Light)
	for i := 0; i < 3; i++ {
		if again := Board(sample(t), Light); again != first {
			t.Fatalf("board render %d differs from the first", i+2)
		}
	}
}

func TestBoardOfAnEmptyPlan(t *testing.T) {
	doc := Board(tree.New("Nothing yet"), Light)
	wellFormed(t, doc)
	if !strings.Contains(doc, "devtree add") {
		t.Errorf("an empty board should say what to do:\n%s", doc)
	}
}

func TestLayoutForFilename(t *testing.T) {
	cases := map[string]Layout{
		"docs/board.svg":      LayoutBoard,
		"docs/board-dark.svg": LayoutBoard,
		"BOARD.SVG":           LayoutBoard,
		"docs/plan.board.svg": LayoutBoard,
		`docs\board.svg`:      LayoutBoard, // a plan written on Windows
		"docs/tree.svg":       LayoutTree,
		"docs/keyboard.svg":   LayoutTree, // a name that merely ends in "board"
		"docs/onboarding.svg": LayoutTree,
	}
	for name, want := range cases {
		if got := LayoutForFilename(name); got != want {
			t.Errorf("LayoutForFilename(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestRenderForPicksBothLayoutAndPalette(t *testing.T) {
	plan := sample(t)

	// The sample has no unstarted leaves, so check a column it does have.
	board := RenderFor(plan, "docs/board.svg")
	if !strings.Contains(board, "dt-col-") || strings.Contains(board, "dt-card-") {
		t.Error("a board file should get the board, not the tree")
	}
	if !strings.Contains(board, ">blocked<") {
		t.Errorf("the board is missing its columns:\n%s", board)
	}

	treeDoc := RenderFor(plan, "docs/tree-dark.svg")
	if !strings.Contains(treeDoc, Dark.Canvas) {
		t.Error("a -dark file should get the dark palette")
	}
	if !strings.Contains(treeDoc, "dt-card-") {
		t.Error("a plain .svg file should get the tree")
	}
}
