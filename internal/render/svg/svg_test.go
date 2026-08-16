package svg

import (
	"encoding/xml"
	"fmt"
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

func TestAnimationIsDeclaredOnceAndCanBeTurnedOff(t *testing.T) {
	doc := Render(sample(t), Light)

	if n := strings.Count(doc, "<style>"); n != 1 {
		t.Errorf("expected exactly one stylesheet, found %d", n)
	}
	if !strings.Contains(doc, "prefers-reduced-motion") {
		t.Error("motion must be switchable off by the reader's own setting")
	}
	if strings.Contains(doc, "<script") {
		t.Error("a script would be blocked by GitHub's CSP and must never be emitted")
	}
}

func TestMotionMarksTheLivePathOnly(t *testing.T) {
	// One branch is moving and one is not. The dash belongs on the moving one
	// and nowhere else, or the animation stops meaning anything.
	tr := tree.New("x")
	add := func(n *tree.Node) {
		t.Helper()
		if err := tr.Add(n); err != nil {
			t.Fatal(err)
		}
	}
	add(&tree.Node{ID: "root", Title: "Root", Status: tree.InProgress})
	add(&tree.Node{ID: "live", Title: "Live", Status: tree.InProgress, Parent: "root"})
	add(&tree.Node{ID: "idle", Title: "Idle", Status: tree.Todo, Parent: "root"})
	add(&tree.Node{ID: "stuck", Title: "Stuck", Status: tree.Blocked, Parent: "root"})

	doc := Render(tr, Light)

	if n := strings.Count(doc, `class="`+"dt-flow"+`"`); n != 1 {
		t.Errorf("%d edges are flowing, want exactly the one leading into in-progress work", n)
	}
	// Two in-progress nodes, so two turning glyphs and no more.
	if n := strings.Count(doc, `class="`+"dt-spin"+`"`); n != 2 {
		t.Errorf("%d glyphs are turning, want 2", n)
	}
}

func TestBlockedWorkBreathesAndSettledWorkDoesNot(t *testing.T) {
	doc := Render(sample(t), Light)

	// One blocked node in the sample, and nothing else may pulse: the point of
	// spending motion on it is that it is rare.
	if n := strings.Count(doc, `class="`+"dt-pulse"+`"`); n != 1 {
		t.Errorf("%d glyphs are pulsing, want exactly the blocked one", n)
	}
	if glyphMotion(tree.Done) != "" || glyphMotion(tree.Todo) != "" || glyphMotion(tree.Dropped) != "" {
		t.Error("settled work should hold still")
	}
}

func TestCardsArriveInSequence(t *testing.T) {
	doc := Render(sample(t), Light)

	if n := strings.Count(doc, `class="dt-rise"`); n != 5 {
		t.Errorf("%d cards fade in, want one per node", n)
	}
	if !strings.Contains(doc, "animation-delay:0.00s") || !strings.Contains(doc, "animation-delay:0.04s") {
		t.Errorf("the stagger is missing:\n%s", doc)
	}

	// The delay must stop growing, or a plan with sixty nodes would spend
	// three seconds assembling itself.
	big := tree.New("big")
	for i := 0; i < 60; i++ {
		if err := big.Add(&tree.Node{ID: fmt.Sprintf("n%d", i), Title: "Task"}); err != nil {
			t.Fatal(err)
		}
	}
	if strings.Contains(Render(big, Light), "animation-delay:1.") {
		t.Error("the entrance stagger should be capped well under a second")
	}
}

func TestProgressBarsGrowIn(t *testing.T) {
	doc := Render(sample(t), Light)
	if !strings.Contains(doc, `class="dt-grow"`) {
		t.Error("the filled part of a progress bar should grow in on load")
	}
	if strings.Count(doc, `class="dt-grow"`) < 2 {
		t.Error("both the header bar and the per-node bars should grow")
	}
}

func TestAccentRunsTheFullHeightOfACard(t *testing.T) {
	doc := Render(sample(t), Light)

	// The stripe is clipped to the card outline, which is the only way it can
	// reach the rounded corners without poking out of them.
	if !strings.Contains(doc, "<clipPath") || !strings.Contains(doc, "clip-path=\"url(#dt-card-0)\"") {
		t.Errorf("expected the accent to be clipped to the card shape:\n%s", doc)
	}
	if !strings.Contains(doc, fmt.Sprintf(`width="%.1f" height="%.1f"`, accentW, cardH)) {
		t.Error("the accent should be as tall as the card itself")
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

func TestEveryCardCarriesItsID(t *testing.T) {
	// The editor hangs controls off these, so a card without an identity is a
	// card nothing can be attached to.
	plan := sample(t)
	doc := Render(plan, Light)
	for _, n := range plan.Nodes {
		if !strings.Contains(doc, `data-node="`+n.ID+`"`) {
			t.Errorf("the tree drew %q with no identity", n.ID)
		}
	}

	board := Board(plan, Light)
	for _, n := range plan.Nodes {
		if len(n.Children()) > 0 {
			continue // containers are not cards on a board
		}
		if !strings.Contains(board, `data-node="`+n.ID+`"`) {
			t.Errorf("the board drew %q with no identity", n.ID)
		}
	}
}

func TestNoDrawingClipsItsOwnSummary(t *testing.T) {
	// The header is the one part that does not shrink with the work: a plan
	// with two tasks left still says how many there were in total.
	for _, plan := range []*tree.Tree{sample(t), tree.New("A project with a rather long name indeed")} {
		for _, doc := range []string{Render(plan, Light), Board(plan, Light)} {
			if got, need := widthOf(t, doc), headerNeeds(plan); got < need-1 {
				t.Errorf("drawn %.0f wide, header needs %.0f", got, need)
			}
		}
	}
}
