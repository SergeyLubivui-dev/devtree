package store

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
	add(&tree.Node{ID: "auth", Title: `Sign in "with a password" | v1`, Status: tree.Done, Parent: "mvp", Issue: "12", Owner: "ann"})
	add(&tree.Node{ID: "pay", Title: "Payments", Parent: "mvp", Branch: "feat/pay", Tags: []string{"backend", "money"}})
	return tr
}

func TestRoundTripIsByteStable(t *testing.T) {
	original := Marshal(sample(t))

	parsed, err := Parse(original)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := parsed.Build(); err != nil {
		t.Fatalf("build: %v", err)
	}

	again := Marshal(parsed)
	if string(again) != string(original) {
		t.Fatalf("a round trip changed the file:\n--- before ---\n%s\n--- after ---\n%s", original, again)
	}
}

func TestRoundTripKeepsAwkwardTitles(t *testing.T) {
	parsed, err := Parse(Marshal(sample(t)))
	if err != nil {
		t.Fatal(err)
	}
	if err := parsed.Build(); err != nil {
		t.Fatal(err)
	}

	n, ok := parsed.Get("auth")
	if !ok {
		t.Fatal("the node disappeared")
	}
	if want := `Sign in "with a password" | v1`; n.Title != want {
		t.Fatalf("title came back as %q, want %q", n.Title, want)
	}
	if n.Owner != "ann" || n.Issue != "12" {
		t.Fatalf("metadata came back as owner=%q issue=%q", n.Owner, n.Issue)
	}

	pay, _ := parsed.Get("pay")
	if len(pay.Tags) != 2 || pay.Tags[0] != "backend" {
		t.Fatalf("tags came back as %v", pay.Tags)
	}
}

func TestParseRejectsBadInputWithLineNumbers(t *testing.T) {
	cases := map[string]string{
		"unknown top-level key": "version: 1\nwhatever: 2\n",
		"unknown node field":    "version: 1\nnodes:\n  - id: \"a\"\n    color: \"red\"\n",
		"unknown status":        "version: 1\nnodes:\n  - id: \"a\"\n    status: \"nearly\"\n",
		"version not a number":  "version: one\n",
		"missing colon":         "version: 1\nnodes:\n  - id \"a\"\n",
		"field before item":     "version: 1\nnodes:\n    id: \"a\"\n",
		"indent outside nodes":  "version: 1\n    id: \"a\"\n",
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Parse([]byte(src))
			if err == nil {
				t.Fatal("expected an error, got none")
			}
			if !strings.Contains(err.Error(), "line ") {
				t.Fatalf("error should name the line, got: %v", err)
			}
		})
	}
}

func TestParseAcceptsHandWrittenInput(t *testing.T) {
	// What a person types: bare scalars, a "#" on an issue, comments, blank
	// lines, CRLF endings from a Windows editor, and a leading BOM.
	src := string([]byte{0xEF, 0xBB, 0xBF}) + // UTF-8 byte order mark
		"# my plan\r\n" +
		"version: 1\r\n" +
		"project: Payment Gateway\r\n" +
		"\r\n" +
		"nodes:\r\n" +
		"  - id: mvp\r\n" +
		"    title: MVP\r\n" +
		"    status: wip\r\n" +
		"  - id: auth\r\n" +
		"    title: Authentication\r\n" +
		"    parent: mvp\r\n" +
		"    issue: \"#12\"\r\n" +
		"    owner: \"@ann\"\r\n"

	tr, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := tr.Build(); err != nil {
		t.Fatalf("build: %v", err)
	}

	if tr.Project != "Payment Gateway" {
		t.Fatalf("project = %q", tr.Project)
	}
	mvp, _ := tr.Get("mvp")
	if mvp.Status != tree.InProgress {
		t.Fatalf("shorthand status resolved to %q", mvp.Status)
	}
	auth, _ := tr.Get("auth")
	if auth.Issue != "12" || auth.Owner != "ann" {
		t.Fatalf("decoration was not trimmed: issue=%q owner=%q", auth.Issue, auth.Owner)
	}
	if len(tr.Outputs) != 1 || tr.Outputs[0] != tree.DefaultOutput {
		t.Fatalf("outputs defaulted to %v, want [%s]", tr.Outputs, tree.DefaultOutput)
	}
}

func TestParseInlineFirstField(t *testing.T) {
	// `- id: "a"` on one line is what Marshal writes; both that and a bare
	// `-` followed by indented fields have to parse.
	tr, err := Parse([]byte("version: 1\nnodes:\n  -\n    id: \"a\"\n    title: \"A\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.Nodes) != 1 || tr.Nodes[0].ID != "a" {
		t.Fatalf("parsed %d nodes: %+v", len(tr.Nodes), tr.Nodes)
	}
}

func TestMarshalOmitsEmptyFields(t *testing.T) {
	tr := tree.New("Minimal")
	if err := tr.Add(&tree.Node{ID: "a", Title: "A"}); err != nil {
		t.Fatal(err)
	}

	out := string(Marshal(tr))
	for _, field := range []string{"parent:", "branch:", "issue:", "pr:", "owner:", "tags:", "note:"} {
		if strings.Contains(out, field) {
			t.Errorf("empty field %q should not be written:\n%s", field, out)
		}
	}
	if !strings.Contains(out, `status: "todo"`) {
		t.Errorf("status should always be written:\n%s", out)
	}
}

func TestSplitList(t *testing.T) {
	got := SplitList(" TREE.md ,, README.md , ")
	if len(got) != 2 || got[0] != "TREE.md" || got[1] != "README.md" {
		t.Fatalf("got %#v", got)
	}
	if SplitList("   ") != nil {
		t.Fatal("a blank list should come back empty")
	}
}
