package render

import (
	"strings"
	"testing"

	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

func sample(t *testing.T) *tree.Tree {
	t.Helper()

	tr := tree.New("Payment Gateway")
	tr.Repo = "https://github.com/acme/pay"
	add := func(n *tree.Node) {
		t.Helper()
		if err := tr.Add(n); err != nil {
			t.Fatalf("add %q: %v", n.ID, err)
		}
	}
	add(&tree.Node{ID: "mvp", Title: "MVP", Status: tree.InProgress})
	add(&tree.Node{ID: "auth", Title: `Sign in "safely" | v1 <beta>`, Status: tree.Done, Parent: "mvp", Issue: "12", Owner: "ann"})
	add(&tree.Node{ID: "pay", Title: "Payments", Parent: "mvp", Branch: "feat/pay"})
	add(&tree.Node{ID: "sbp", Title: "Instant transfers", Status: tree.Blocked, Parent: "pay", PR: "44"})
	return tr
}

// diagram pulls the fenced Mermaid block out of a rendered document.
func diagram(t *testing.T, block string) string {
	t.Helper()
	start := strings.Index(block, "```mermaid")
	if start < 0 {
		t.Fatal("no mermaid block in the output")
	}
	end := strings.Index(block[start+len("```mermaid"):], "```")
	if end < 0 {
		t.Fatal("the mermaid block is not closed")
	}
	return block[start : start+len("```mermaid")+end]
}

func TestMermaidEscapesLabels(t *testing.T) {
	got := diagram(t, Block(sample(t)))

	// A raw quote, angle bracket, or pipe inside a label ends it early and
	// produces a diagram that will not parse.
	for _, raw := range []string{`"Sign in "safely"`, "| v1", "<beta>"} {
		if strings.Contains(got, raw) {
			t.Errorf("unescaped %q reached the diagram:\n%s", raw, got)
		}
	}
	for _, want := range []string{"#quot;", "#124;", "#lt;", "#35;12"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected the escape %q in:\n%s", want, got)
		}
	}
}

func TestMermaidStructure(t *testing.T) {
	got := diagram(t, Block(sample(t)))

	for _, want := range []string{
		"flowchart TD",
		"classDef in_progress",
		"n_mvp --> n_auth",
		"n_pay --> n_sbp",
		":::blocked",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestMermaidNamesStayUnique(t *testing.T) {
	tr := tree.New("collisions")
	// "a.b" and "a-b" both sanitize to "n_a_b"; without a suffix the two
	// would silently merge into one box.
	for _, id := range []string{"a.b", "a-b"} {
		if err := tr.Add(&tree.Node{ID: id, Title: id}); err != nil {
			t.Fatal(err)
		}
	}

	names := mermaidNames(tr)
	if names["a.b"] == names["a-b"] {
		t.Fatalf("both ids mapped to %q", names["a.b"])
	}
}

func TestEmptyTreeSaysWhatToDo(t *testing.T) {
	got := Block(tree.New("Nothing yet"))
	if !strings.Contains(got, "devtree add") {
		t.Fatalf("an empty diagram should suggest the next step:\n%s", got)
	}
}

func TestLinkTablePrefersPRThenIssueThenBranch(t *testing.T) {
	got := LinkTable(sample(t))

	for _, want := range []string{
		"[#44](https://github.com/acme/pay/pull/44)",
		"[#12](https://github.com/acme/pay/issues/12)",
		"`feat/pay`",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "| `mvp` |") {
		t.Error("a node with nothing to link to should stay out of the table")
	}
}

func TestLinkTableFallsBackToRelativeLinks(t *testing.T) {
	tr := sample(t)
	tr.Repo = "" // a plan that never configured a repository URL

	got := LinkTable(tr)
	if !strings.Contains(got, "[#12](../../issues/12)") {
		t.Fatalf("expected a relative issue link:\n%s", got)
	}
}

func TestLinkTableEscapesPipes(t *testing.T) {
	tr := tree.New("x")
	if err := tr.Add(&tree.Node{ID: "a", Title: "One | Two", Issue: "3"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(LinkTable(tr), `One \| Two`) {
		t.Fatal("a pipe in a title must be escaped or it splits the table row")
	}
}

func TestProgressBar(t *testing.T) {
	cases := []struct {
		done, total int
		want        string
	}{
		{0, 0, strings.Repeat("░", barWidth)},
		{0, 4, strings.Repeat("░", barWidth)},
		{4, 4, strings.Repeat("█", barWidth)},
		{1, 4, strings.Repeat("█", 5) + strings.Repeat("░", 15)},
	}
	for _, c := range cases {
		if got := ProgressBar(c.done, c.total); got != c.want {
			t.Errorf("ProgressBar(%d, %d) = %q, want %q", c.done, c.total, got, c.want)
		}
	}
}

func TestInjectIsIdempotent(t *testing.T) {
	const readme = "# My project\n\nSome prose.\n"
	block := Block(sample(t))

	first, changed := Inject(readme, block)
	if !changed {
		t.Fatal("the first injection should change the file")
	}

	current := first
	for i := 0; i < 3; i++ {
		next, changed := Inject(current, block)
		if changed {
			t.Fatalf("injection %d changed the file again", i+2)
		}
		if next != current {
			t.Fatalf("injection %d rewrote the file: %d bytes → %d", i+2, len(current), len(next))
		}
		current = next
	}

	if !strings.HasPrefix(current, "# My project") {
		t.Error("the author's own content was overwritten")
	}
	if strings.Count(current, MarkerBegin) != 1 {
		t.Error("the block was appended twice")
	}
}

func TestInjectReplacesInPlaceAndKeepsTheTail(t *testing.T) {
	before := "# Title\n\n" + MarkerBegin + "\nstale\n" + MarkerEnd + "\n\n## License\n\nMIT\n"

	got, changed := Inject(before, Block(sample(t)))
	if !changed {
		t.Fatal("replacing a stale block should count as a change")
	}
	if strings.Contains(got, "stale") {
		t.Error("the stale block survived")
	}
	if !strings.Contains(got, "## License") {
		t.Error("content after the block was dropped")
	}
	if !strings.HasPrefix(got, "# Title") {
		t.Error("content before the block was dropped")
	}
}

func TestInjectIgnoresMarkersQuotedInProse(t *testing.T) {
	// A README that explains devtree mentions the markers inside a sentence.
	// Only the pair on their own lines is the real region.
	before := "# Docs\n\n" +
		"The block goes between `" + MarkerBegin + "` and `" + MarkerEnd + "` in your file.\n\n" +
		MarkerBegin + "\n" + MarkerEnd + "\n\n" +
		"## License\n"

	got, changed := Inject(before, Block(sample(t)))
	if !changed {
		t.Fatal("the real region should have been filled in")
	}
	if !strings.Contains(got, "The block goes between `"+MarkerBegin+"` and `"+MarkerEnd+"` in your file.") {
		t.Errorf("the sentence describing the markers was eaten:\n%s", got)
	}
	if !strings.Contains(got, "## License") {
		t.Errorf("content after the region was dropped:\n%s", got)
	}
	if !strings.Contains(got, "```mermaid") {
		t.Errorf("the diagram never landed:\n%s", got)
	}

	// And it stays stable: a second pass must find the same region.
	again, changed := Inject(got, Block(sample(t)))
	if changed || again != got {
		t.Errorf("a second injection moved the region: %d bytes → %d", len(got), len(again))
	}
}

func TestInjectCreatesTheBodyOfAnEmptyFile(t *testing.T) {
	got, changed := Inject("", "BLOCK")
	if !changed || got != "BLOCK\n" {
		t.Fatalf("got %q, changed=%v", got, changed)
	}
}

func TestASCIIDrawsTheTree(t *testing.T) {
	got := ASCII(sample(t), "")

	for _, want := range []string{
		"◐ MVP  (mvp)",
		"├─ ",
		"└─ ",
		"[1/3]",
		"feat/pay",
		"#12",
		"@ann",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestASCIIFiltersByStatus(t *testing.T) {
	got := ASCII(sample(t), tree.Blocked)

	if !strings.Contains(got, "Instant transfers") {
		t.Errorf("the blocked task should be listed:\n%s", got)
	}
	if strings.Contains(got, "MVP") {
		t.Errorf("everything else should be filtered out:\n%s", got)
	}
}

func TestASCIISaysWhenNothingMatches(t *testing.T) {
	if got := ASCII(sample(t), tree.Dropped); !strings.Contains(got, "nothing") {
		t.Fatalf("got %q", got)
	}
}

func TestBlockCountsProgress(t *testing.T) {
	got := Block(sample(t))
	if !strings.Contains(got, "**1 / 4** tasks done") {
		t.Fatalf("progress line missing or wrong:\n%s", got)
	}
	if !strings.Contains(got, "## 🌳 Payment Gateway") {
		t.Fatal("the project heading is missing")
	}
}
