package tree

import "testing"

// sample builds the small plan the tests share: one milestone with two tasks
// under it, one of them finished.
func sample(t *testing.T) *Tree {
	t.Helper()

	tr := New("Test project")
	add := func(n *Node) {
		t.Helper()
		if err := tr.Add(n); err != nil {
			t.Fatalf("add %q: %v", n.ID, err)
		}
	}
	add(&Node{ID: "mvp", Title: "MVP", Status: InProgress})
	add(&Node{ID: "auth", Title: "Authentication", Status: Done, Parent: "mvp", Issue: "12"})
	add(&Node{ID: "pay", Title: "Payments", Parent: "mvp", Branch: "feat/pay"})
	return tr
}

func TestBuildRejectsBrokenTrees(t *testing.T) {
	cases := map[string]func(*Tree){
		"missing parent": func(tr *Tree) {
			tr.Nodes = append(tr.Nodes, &Node{ID: "a", Parent: "nowhere"})
		},
		"duplicate id": func(tr *Tree) {
			tr.Nodes = append(tr.Nodes, &Node{ID: "a"}, &Node{ID: "a"})
		},
		"self parent": func(tr *Tree) {
			tr.Nodes = append(tr.Nodes, &Node{ID: "a", Parent: "a"})
		},
		"cycle": func(tr *Tree) {
			tr.Nodes = append(tr.Nodes, &Node{ID: "a", Parent: "b"}, &Node{ID: "b", Parent: "a"})
		},
		"empty id": func(tr *Tree) {
			tr.Nodes = append(tr.Nodes, &Node{Title: "nameless"})
		},
		"invalid id": func(tr *Tree) {
			tr.Nodes = append(tr.Nodes, &Node{ID: "-leading-dash"})
		},
		"unknown status": func(tr *Tree) {
			tr.Nodes = append(tr.Nodes, &Node{ID: "a", Status: Status("made up")})
		},
	}

	for name, breakIt := range cases {
		t.Run(name, func(t *testing.T) {
			tr := New("x")
			breakIt(tr)
			if err := tr.Build(); err == nil {
				t.Fatal("expected an error, got none")
			}
		})
	}
}

func TestAddRollsBackOnFailure(t *testing.T) {
	tr := sample(t)
	before := len(tr.Nodes)

	if err := tr.Add(&Node{ID: "auth", Title: "Duplicate"}); err == nil {
		t.Fatal("expected a duplicate id to be rejected")
	}
	if len(tr.Nodes) != before {
		t.Fatalf("rejected node stayed in the tree: %d nodes, want %d", len(tr.Nodes), before)
	}
	if n, ok := tr.Get("auth"); !ok || n.Title != "Authentication" {
		t.Fatalf("the original node was overwritten: %+v", n)
	}
}

func TestProgressSkipsSelfAndDropped(t *testing.T) {
	tr := sample(t)
	if err := tr.Add(&Node{ID: "legacy", Title: "Old idea", Status: Dropped, Parent: "mvp"}); err != nil {
		t.Fatal(err)
	}

	mvp, _ := tr.Get("mvp")
	done, total := tr.Progress(mvp)
	if done != 1 || total != 2 {
		t.Fatalf("got %d/%d, want 1/2 (the node itself and dropped work do not count)", done, total)
	}
}

func TestTotalsSkipDropped(t *testing.T) {
	tr := sample(t)
	if err := tr.Add(&Node{ID: "legacy", Status: Dropped}); err != nil {
		t.Fatal(err)
	}
	done, total := tr.Totals()
	if done != 1 || total != 3 {
		t.Fatalf("got %d/%d, want 1/3", done, total)
	}
}

func TestRemoveReparentsChildren(t *testing.T) {
	tr := sample(t)
	if err := tr.Add(&Node{ID: "cards", Title: "Card payments", Parent: "pay"}); err != nil {
		t.Fatal(err)
	}

	removed, err := tr.Remove("pay", false)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("removed %d nodes, want 1", removed)
	}

	cards, ok := tr.Get("cards")
	if !ok {
		t.Fatal("the child was deleted along with its parent")
	}
	if cards.Parent != "mvp" {
		t.Fatalf("child re-parented to %q, want %q", cards.Parent, "mvp")
	}
}

func TestRemoveCascadeTakesTheSubtree(t *testing.T) {
	tr := sample(t)
	removed, err := tr.Remove("mvp", true)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 3 {
		t.Fatalf("removed %d nodes, want 3", removed)
	}
	if len(tr.Nodes) != 0 {
		t.Fatalf("%d nodes left, want an empty tree", len(tr.Nodes))
	}
}

func TestMoveRejectsCycleAndRollsBack(t *testing.T) {
	tr := sample(t)
	if err := tr.Move("mvp", "auth"); err == nil {
		t.Fatal("moving a parent under its own child should be rejected")
	}

	mvp, _ := tr.Get("mvp")
	if mvp.Parent != "" {
		t.Fatalf("after the rollback the parent is %q, want it empty", mvp.Parent)
	}
	if got := len(tr.Roots()); got != 1 {
		t.Fatalf("%d roots after the rollback, want 1", got)
	}
}

func TestMoveToRoot(t *testing.T) {
	tr := sample(t)
	if err := tr.Move("pay", "root"); err != nil {
		t.Fatal(err)
	}
	pay, _ := tr.Get("pay")
	if pay.Parent != "" {
		t.Fatalf("parent is %q, want it empty", pay.Parent)
	}
	if got := len(tr.Roots()); got != 2 {
		t.Fatalf("%d roots, want 2", got)
	}
}

func TestNextIDAvoidsCollisions(t *testing.T) {
	tr := sample(t)
	if got := tr.NextID("Authentication"); got != "authentication" {
		t.Fatalf("got %q, want %q", got, "authentication")
	}
	if err := tr.Add(&Node{ID: "authentication", Title: "Authentication"}); err != nil {
		t.Fatal(err)
	}
	if got := tr.NextID("Authentication"); got != "authentication-2" {
		t.Fatalf("got %q, want %q", got, "authentication-2")
	}
}

func TestParseStatusAcceptsShorthand(t *testing.T) {
	for input, want := range map[string]Status{
		"wip":         InProgress,
		"WIP":         InProgress,
		" done ":      Done,
		"x":           Dropped,
		"in-progress": InProgress,
	} {
		got, err := ParseStatus(input)
		if err != nil {
			t.Fatalf("%q: %v", input, err)
		}
		if got != want {
			t.Fatalf("%q resolved to %q, want %q", input, got, want)
		}
	}
	if _, err := ParseStatus("almost done"); err == nil {
		t.Fatal("an unknown status should be rejected")
	}
}

func TestEveryStatusHasAnIconAndLabel(t *testing.T) {
	for _, s := range Statuses {
		if s.Icon() == "" || s.Label() == "" {
			t.Fatalf("status %q is missing an icon or a label", s)
		}
		if !s.Valid() {
			t.Fatalf("status %q should be valid", s)
		}
	}
}
