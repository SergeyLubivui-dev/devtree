package render

import (
	"strings"
	"testing"

	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

func filtered(t *testing.T) *tree.Node {
	t.Helper()
	return &tree.Node{
		ID: "auth", Title: "Authentication", Status: tree.InProgress,
		Owner: "Ann", Tags: []string{"Backend", "money"},
	}
}

func TestEmptyFilterLetsEverythingThrough(t *testing.T) {
	var f Filter
	if !f.Empty() {
		t.Error("a zero Filter should be empty")
	}
	if !f.Match(filtered(t)) {
		t.Error("an empty filter should match anything")
	}
}

func TestFilterConditionsCombineWithAnd(t *testing.T) {
	n := filtered(t)

	if !(Filter{Status: tree.InProgress, Owner: "ann"}).Match(n) {
		t.Error("both conditions hold; the node should match")
	}
	if (Filter{Status: tree.Blocked, Owner: "ann"}).Match(n) {
		t.Error("one condition failing should reject the node")
	}
}

func TestFilterIgnoresCaseAndDecoration(t *testing.T) {
	n := filtered(t)

	// People type @ann, ANN, and ann, and mean the same person.
	for _, owner := range []string{"ann", "Ann", "ANN"} {
		if !(Filter{Owner: owner}).Match(n) {
			t.Errorf("owner %q should have matched", owner)
		}
	}
	if !(Filter{Tags: []string{"backend"}}).Match(n) {
		t.Error("tags should match without regard to case")
	}
}

func TestTagsCombineWithOr(t *testing.T) {
	n := filtered(t)

	// "Show me anything touching billing or payments" is the question people
	// actually ask, so any tag matching is enough.
	if !(Filter{Tags: []string{"nothing", "money"}}).Match(n) {
		t.Error("one matching tag should be enough")
	}
	if (Filter{Tags: []string{"nothing", "at-all"}}).Match(n) {
		t.Error("no matching tag should reject the node")
	}
}

func TestFilterDescribesItself(t *testing.T) {
	f := Filter{Status: tree.Blocked, Owner: "ann", Tags: []string{"backend"}}
	got := f.Describe()

	for _, want := range []string{"blocked", "@ann", "#backend"} {
		if !strings.Contains(got, want) {
			t.Errorf("Describe() = %q, missing %q", got, want)
		}
	}
}

func TestASCIIFiltersByOwnerAndTag(t *testing.T) {
	tr := tree.New("x")
	for _, n := range []*tree.Node{
		{ID: "mvp", Title: "MVP"},
		{ID: "auth", Title: "Authentication", Parent: "mvp", Owner: "ann", Tags: []string{"backend"}},
		{ID: "docs", Title: "Docs", Parent: "mvp", Owner: "bob", Tags: []string{"writing"}},
	} {
		if err := tr.Add(n); err != nil {
			t.Fatal(err)
		}
	}

	got := ASCII(tr, Filter{Owner: "ann"})
	if !strings.Contains(got, "Authentication") || strings.Contains(got, "Docs") {
		t.Errorf("owner filter did not apply:\n%s", got)
	}

	got = ASCII(tr, Filter{Tags: []string{"writing"}})
	if !strings.Contains(got, "Docs") || strings.Contains(got, "Authentication") {
		t.Errorf("tag filter did not apply:\n%s", got)
	}
}

func TestURLPrefersTheMostSpecificThing(t *testing.T) {
	tr := tree.New("x")
	tr.Repo = "https://github.com/acme/pay/"
	n := &tree.Node{ID: "auth", Title: "Authentication", Branch: "feat/auth", Issue: "12", PR: "44"}
	if err := tr.Add(n); err != nil {
		t.Fatal(err)
	}

	cases := map[string]string{
		"":         "https://github.com/acme/pay/pull/44",
		KindPR:     "https://github.com/acme/pay/pull/44",
		KindIssue:  "https://github.com/acme/pay/issues/12",
		KindBranch: "https://github.com/acme/pay/tree/feat/auth",
	}
	for kind, want := range cases {
		if got, _ := URL(tr, n, kind); got != want {
			t.Errorf("URL(kind=%q) = %q, want %q", kind, got, want)
		}
	}
}

func TestURLNeedsARepository(t *testing.T) {
	tr := tree.New("x") // no repo configured
	n := &tree.Node{ID: "a", Title: "A", Issue: "12"}
	if err := tr.Add(n); err != nil {
		t.Fatal(err)
	}

	// A relative link works inside a Markdown file in the repository root, but
	// there is nothing to hand a browser.
	if got, _ := URL(tr, n, ""); got != "" {
		t.Errorf("got %q, want nothing openable", got)
	}
}

func TestURLFallsThroughToWhatExists(t *testing.T) {
	tr := tree.New("x")
	tr.Repo = "https://github.com/acme/pay"
	n := &tree.Node{ID: "a", Title: "A", Branch: "feat/x"}
	if err := tr.Add(n); err != nil {
		t.Fatal(err)
	}

	got, kind := URL(tr, n, "")
	if got != "https://github.com/acme/pay/tree/feat/x" || kind != KindBranch {
		t.Fatalf("got %q (%s)", got, kind)
	}
	if url, _ := URL(tr, n, KindPR); url != "" {
		t.Error("asking for a pull request that does not exist should return nothing")
	}
}
