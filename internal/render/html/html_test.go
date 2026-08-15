package html

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
	add(&tree.Node{ID: "auth", Title: "Authentication", Status: tree.InProgress, Parent: "mvp",
		Branch: "feat/auth", Issue: "12", Owner: "ann", Tags: []string{"backend"}})
	add(&tree.Node{ID: "reset", Title: "Password reset", Status: tree.Blocked, Parent: "auth",
		Note: "waiting on the SMTP contract", Owner: "bob"})
	add(&tree.Node{ID: "stripe", Title: "Stripe", Status: tree.Done, Parent: "mvp", PR: "44",
		Owner: "bob", Tags: []string{"money"}})
	add(&tree.Node{ID: "old", Title: "Old idea", Status: tree.Dropped})
	return tr
}

// balanced checks that a container element opens and closes the same number of
// times. It is not a parser, but it catches the mistake a hand-written emitter
// actually makes: a branch that forgets to close.
func balanced(t *testing.T, doc, element string) {
	t.Helper()
	// Both spellings of an opening tag, and neither of them a longer name that
	// merely starts the same way — <header> is not a <head>.
	open := strings.Count(doc, "<"+element+">") + strings.Count(doc, "<"+element+" ")
	closed := strings.Count(doc, "</"+element+">")
	if open != closed {
		t.Errorf("%s: %d opened, %d closed", element, open, closed)
	}
}

func TestPageStructureCloses(t *testing.T) {
	doc := Render(sample(t))

	if !strings.HasPrefix(doc, "<!doctype html>") {
		t.Error("a document should say what it is")
	}
	if !strings.HasSuffix(strings.TrimSpace(doc), "</html>") {
		t.Error("the document does not close")
	}
	for _, element := range []string{"html", "head", "body", "section", "ul", "li", "fieldset", "footer"} {
		balanced(t, doc, element)
	}
}

func TestPageIsSelfContained(t *testing.T) {
	doc := Render(sample(t))

	// The point of this export is that it works from a file:// URL, from a zip,
	// and from GitHub Pages without anything else being served alongside it.
	for _, forbidden := range []string{"<script", "<link ", "@import", "url(http", `src="http`} {
		if strings.Contains(doc, forbidden) {
			t.Errorf("the page pulls in something external: %q", forbidden)
		}
	}
}

func TestEveryTaskAppears(t *testing.T) {
	doc := Render(sample(t))
	for _, title := range []string{"MVP", "Authentication", "Password reset", "Stripe", "Old idea"} {
		if !strings.Contains(doc, title) {
			t.Errorf("%q is missing from the page", title)
		}
	}
}

func TestCardsLinkToWhatTheyPointAt(t *testing.T) {
	doc := Render(sample(t))

	// This is the whole reason the backend exists: Mermaid ignores click
	// directives and an SVG served as an image is sandboxed, so neither can do
	// this.
	for _, href := range []string{
		`href="https://github.com/acme/pay/pull/44"`,
		`href="https://github.com/acme/pay/issues/12"`,
	} {
		if !strings.Contains(doc, href) {
			t.Errorf("missing %s", href)
		}
	}
	// A task pointing at nothing must not become an empty link.
	if strings.Contains(doc, `href=""`) {
		t.Error("a card with nothing to point at became a link anyway")
	}
}

func TestWithoutARepositoryNothingIsALink(t *testing.T) {
	tr := sample(t)
	tr.Repo = ""
	if strings.Contains(Render(tr), "<a href") {
		t.Error("relative links make no sense in an exported page")
	}
}

func TestHostileTextIsEscaped(t *testing.T) {
	tr := tree.New(`Project <script>alert("x")</script>`)
	if err := tr.Add(&tree.Node{ID: "a", Title: `Sign in "safely" & <fast>`, Note: "<b>note</b>"}); err != nil {
		t.Fatal(err)
	}

	doc := Render(tr)
	if strings.Contains(doc, "<script>alert") || strings.Contains(doc, "<b>note</b>") {
		t.Error("markup from the plan reached the page unescaped")
	}
	if !strings.Contains(doc, "&lt;fast&gt;") || !strings.Contains(doc, "&amp;") {
		t.Errorf("expected escaped text:\n%s", doc)
	}
}

func TestBothViewsAreOnThePage(t *testing.T) {
	doc := Render(sample(t))

	if !strings.Contains(doc, `class="tree"`) || !strings.Contains(doc, `class="board"`) {
		t.Error("the page should carry both views and switch between them in CSS")
	}
	if !strings.Contains(doc, `id="view-board"`) {
		t.Error("the view switch is missing")
	}
	// Only leaves belong on a board: a milestone is a container, not a card.
	board := doc[strings.Index(doc, `class="board"`):]
	if strings.Contains(board, ">MVP<") {
		t.Error("a milestone was drawn as a board card")
	}
}

func TestFilterControlsComeFromThePlan(t *testing.T) {
	doc := Render(sample(t))

	for _, id := range []string{`id="owner-ann"`, `id="owner-bob"`, `id="tag-backend"`, `id="tag-money"`} {
		if !strings.Contains(doc, id) {
			t.Errorf("missing filter control %s", id)
		}
	}
	// And a rule that actually does something with each of them.
	if !strings.Contains(doc, `body:has(#owner-ann:checked) .card:not([data-owner="ann"]){display:none}`) {
		t.Error("the owner filter has a control but no rule")
	}
	if !strings.Contains(doc, `body:has(#tag-money:checked) .card:not([data-tags~="money"]){display:none}`) {
		t.Error("the tag filter has a control but no rule")
	}
}

func TestNotesBecomeTooltips(t *testing.T) {
	doc := Render(sample(t))

	if !strings.Contains(doc, `role="tooltip"`) || !strings.Contains(doc, "waiting on the SMTP contract") {
		t.Error("a note should be reachable without taking a line of its own")
	}
	// The delay belongs only to the hover rule, so leaving snaps it to zero and
	// the bubble goes at once.
	if !strings.Contains(doc, "transition-delay:80ms") {
		t.Error("the tooltip should ease in on a delay")
	}
	if strings.Contains(doc, ".tip{transition:opacity 50ms ease-out,transform 50ms ease-out;transition-delay") {
		t.Error("the delay leaked into the leave transition")
	}
}

func TestMotionAndPrintAreConsidered(t *testing.T) {
	doc := Render(sample(t))
	for _, want := range []string{"prefers-reduced-motion", "@media print"} {
		if !strings.Contains(doc, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	first := Render(sample(t))
	for i := 0; i < 3; i++ {
		if again := Render(sample(t)); again != first {
			t.Fatalf("render %d differs from the first", i+2)
		}
	}
}

func TestEmptyPlanSaysWhatToDo(t *testing.T) {
	doc := Render(tree.New("Nothing yet"))
	if !strings.Contains(doc, "devtree add") {
		t.Errorf("an empty page should suggest the next step:\n%s", doc)
	}
}
