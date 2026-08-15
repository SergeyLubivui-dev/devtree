package svg

import (
	"encoding/xml"
	"io"
	"strings"
	"testing"

	"github.com/SergeyLubivui-dev/devtree/internal/icons"
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
	add(&tree.Node{ID: "auth", Title: "Authentication", Status: tree.Done, Parent: "mvp",
		Branch: "feat/auth", Issue: "12", Owner: "ann"})
	add(&tree.Node{ID: "pay", Title: "Payments", Parent: "mvp", Tags: []string{"backend", "money"}})
	add(&tree.Node{ID: "sbp", Title: "Instant transfers", Status: tree.Blocked, Parent: "pay", PR: "44"})
	add(&tree.Node{ID: "legacy", Title: "Old idea", Status: tree.Dropped})
	return tr
}

// wellFormed walks every token, which fails on an unescaped angle bracket or
// quote anywhere in the document — the failure mode that turns a diagram into
// a browser error page.
func wellFormed(t *testing.T, doc string) {
	t.Helper()
	decoder := xml.NewDecoder(strings.NewReader(doc))
	for {
		_, err := decoder.Token()
		if err == io.EOF {
			return
		}
		if err != nil {
			t.Fatalf("not well-formed XML: %v", err)
		}
	}
}

func TestRenderIsWellFormed(t *testing.T) {
	for _, th := range []Theme{Light, Dark} {
		t.Run(th.Name, func(t *testing.T) {
			wellFormed(t, Render(sample(t), th))
		})
	}
}

func TestRenderSurvivesHostileTitles(t *testing.T) {
	tr := tree.New(`Project <script>alert("x")</script>`)
	if err := tr.Add(&tree.Node{ID: "a", Title: `Sign in "safely" & <fast>`, Branch: "feat/<b>"}); err != nil {
		t.Fatal(err)
	}

	doc := Render(tr, Light)
	wellFormed(t, doc)

	if strings.Contains(doc, "<script>") {
		t.Error("a raw script tag reached the document")
	}
	if !strings.Contains(doc, "&amp;") || !strings.Contains(doc, "&lt;fast&gt;") {
		t.Errorf("expected escaped text in:\n%s", doc)
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	// Two renders of the same plan must be byte-identical, or every `devtree
	// render` would show up as a diff and the CI staleness check would cry
	// wolf on every commit.
	first := Render(sample(t), Light)
	for i := 0; i < 3; i++ {
		if again := Render(sample(t), Light); again != first {
			t.Fatalf("render %d differs from the first", i+2)
		}
	}
}

func TestRenderIncludesEveryNode(t *testing.T) {
	doc := Render(sample(t), Light)
	for _, want := range []string{"MVP", "Authentication", "Payments", "Instant transfers", "Old idea"} {
		if !strings.Contains(doc, want) {
			t.Errorf("node %q is missing from the diagram", want)
		}
	}
	for _, want := range []string{"feat/auth", "12", "ann", "44", "backend, money"} {
		if !strings.Contains(doc, want) {
			t.Errorf("metadata %q is missing from the diagram", want)
		}
	}
}

func TestRenderShowsProgressAndStrikesDroppedWork(t *testing.T) {
	doc := Render(sample(t), Light)
	if !strings.Contains(doc, ">1/3<") {
		t.Error("the MVP rollup 1/3 is missing")
	}
	if !strings.Contains(doc, "1 / 4 tasks done") {
		t.Error("the overall total is missing or counts dropped work")
	}
	if !strings.Contains(doc, "line-through") {
		t.Error("dropped work should be struck through")
	}
}

func TestThemesActuallyDiffer(t *testing.T) {
	light, dark := Render(sample(t), Light), Render(sample(t), Dark)
	if light == dark {
		t.Fatal("both themes rendered the same bytes")
	}
	if !strings.Contains(light, Light.Canvas) || strings.Contains(light, Dark.Canvas) {
		t.Error("the light rendering does not use the light canvas")
	}
	if !strings.Contains(dark, Dark.Canvas) || strings.Contains(dark, Light.Canvas) {
		t.Error("the dark rendering does not use the dark canvas")
	}
}

func TestThemeForFilename(t *testing.T) {
	cases := map[string]string{
		"docs/tree.svg":      "light",
		"tree-dark.svg":      "dark",
		"docs/TREE-DARK.SVG": "dark",
		"plan_dark.svg":      "dark",
		"darkroom.svg":       "light", // only a suffix counts
	}
	for name, want := range cases {
		if got := ThemeForFilename(name); got.Name != want {
			t.Errorf("ThemeForFilename(%q) = %q, want %q", name, got.Name, want)
		}
	}
	if ThemeFor("DARK").Name != "dark" || ThemeFor("nonsense").Name != "light" {
		t.Error("ThemeFor should be case-insensitive and default to light")
	}
}

func TestEveryStatusHasAGlyph(t *testing.T) {
	for _, s := range tree.Statuses {
		name, ok := statusIcon[s]
		if !ok {
			t.Fatalf("status %q has no icon", s)
		}
		if _, ok := icons.Get(name); !ok {
			t.Fatalf("status %q maps to unknown icon %q", s, name)
		}
	}
}

func TestEmptyTreeSaysWhatToDo(t *testing.T) {
	doc := Render(tree.New("Nothing yet"), Light)
	wellFormed(t, doc)
	if !strings.Contains(doc, "devtree add") {
		t.Errorf("an empty plan should suggest the next step:\n%s", doc)
	}
}

func TestCanvasGrowsWithTheTree(t *testing.T) {
	small := tree.New("x")
	if err := small.Add(&tree.Node{ID: "a", Title: "A"}); err != nil {
		t.Fatal(err)
	}

	shorter, taller := Render(small, Light), Render(sample(t), Light)
	if height(t, shorter) >= height(t, taller) {
		t.Fatalf("a one-node plan (%d) should be shorter than a five-node plan (%d)",
			height(t, shorter), height(t, taller))
	}
}

func height(t *testing.T, doc string) int {
	t.Helper()
	var svg struct {
		Height int `xml:"height,attr"`
	}
	if err := xml.Unmarshal([]byte(doc), &svg); err != nil {
		t.Fatal(err)
	}
	return svg.Height
}

func TestLongTitlesAreClipped(t *testing.T) {
	tr := tree.New("x")
	long := strings.Repeat("a very long milestone title ", 6)
	if err := tr.Add(&tree.Node{ID: "a", Title: long}); err != nil {
		t.Fatal(err)
	}

	doc := Render(tr, Light)
	if !strings.Contains(doc, "…") {
		t.Error("an over-long title should be clipped with an ellipsis")
	}
	if strings.Contains(doc, long) {
		t.Error("the full over-long title was drawn past the card edge")
	}
}
