package serve

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/SergeyLubivui-dev/devtree/internal/icons"
	"github.com/SergeyLubivui-dev/devtree/internal/store"
	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

// project writes a real plan into a temporary directory and returns a server
// pointed at it. Every test then talks HTTP, the way the page does.
func project(t *testing.T) (*Server, http.Handler) {
	t.Helper()

	root := t.TempDir()
	plan := tree.New("Payment Gateway")
	plan.Repo = "https://github.com/acme/pay"
	plan.Outputs = []string{"TREE.md"}

	for _, n := range []*tree.Node{
		{ID: "mvp", Title: "MVP", Status: tree.InProgress},
		{ID: "auth", Title: "Authentication", Status: tree.InProgress, Parent: "mvp",
			Branch: "feat/auth", Issue: "12", Owner: "ann"},
		{ID: "stripe", Title: "Stripe", Status: tree.Done, Parent: "mvp", PR: "44"},
	} {
		if err := plan.Add(n); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Save(root, plan); err != nil {
		t.Fatal(err)
	}

	server := &Server{Root: root, Log: io.Discard}
	return server, server.Handler()
}

func request(t *testing.T, handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func decodePlan(t *testing.T, rec *httptest.ResponseRecorder) jsonPlan {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var plan jsonPlan
	if err := json.Unmarshal(rec.Body.Bytes(), &plan); err != nil {
		t.Fatalf("%v: %s", err, rec.Body.String())
	}
	return plan
}

func TestThePageIsServed(t *testing.T) {
	_, handler := project(t)

	rec := request(t, handler, http.MethodGet, "/", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	for _, want := range []string{"<!doctype html>", "app.css", "app.js", `data-theme="dark"`} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("the page is missing %q", want)
		}
	}

	// The stylesheet and the script are embedded in the binary, not read from
	// disk, so a devtree copied anywhere still serves a working editor.
	for _, asset := range []string{"/app.css", "/app.js"} {
		if rec := request(t, handler, http.MethodGet, asset, ""); rec.Code != http.StatusOK {
			t.Errorf("%s: status %d", asset, rec.Code)
		}
	}
}

func TestPlanIsDescribedInTreeOrder(t *testing.T) {
	_, handler := project(t)
	plan := decodePlan(t, request(t, handler, http.MethodGet, "/api/plan", ""))

	if plan.Project != "Payment Gateway" {
		t.Errorf("project is %q", plan.Project)
	}
	if len(plan.Nodes) != 3 {
		t.Fatalf("%d nodes", len(plan.Nodes))
	}
	if plan.Nodes[0].ID != "mvp" || plan.Nodes[1].ID != "auth" {
		t.Errorf("order is %s, %s", plan.Nodes[0].ID, plan.Nodes[1].ID)
	}
	if plan.Nodes[1].Depth != 1 {
		t.Errorf("a child should carry its depth, got %d", plan.Nodes[1].Depth)
	}
	if plan.Nodes[0].Total != 2 || plan.Nodes[0].Done != 1 {
		t.Errorf("rollup is %d/%d", plan.Nodes[0].Done, plan.Nodes[0].Total)
	}
	// The page turns a task into a link with this.
	if plan.Nodes[2].URL != "https://github.com/acme/pay/pull/44" {
		t.Errorf("url is %q", plan.Nodes[2].URL)
	}
	if len(plan.Statuses) != len(tree.Statuses) {
		t.Error("the status vocabulary should come from the domain, not the page")
	}
}

func TestEditingWritesThePlanFile(t *testing.T) {
	server, handler := project(t)

	rec := request(t, handler, http.MethodPatch, "/api/task/auth",
		`{"status":"blocked","note":"waiting on SMTP","owner":"@bob","tags":["backend"," "]}`)
	plan := decodePlan(t, rec)

	var auth jsonNode
	for _, n := range plan.Nodes {
		if n.ID == "auth" {
			auth = n
		}
	}
	if auth.Status != "blocked" || auth.Note != "waiting on SMTP" {
		t.Errorf("the edit did not take: %+v", auth)
	}
	if auth.Owner != "bob" {
		t.Errorf("the @ should be trimmed, got %q", auth.Owner)
	}
	if len(auth.Tags) != 1 {
		t.Errorf("blank tags should be dropped, got %v", auth.Tags)
	}

	// And it is on disk, in the same file the command line reads.
	saved, err := store.Load(server.Root)
	if err != nil {
		t.Fatal(err)
	}
	if n, _ := saved.Get("auth"); n.Status != tree.Blocked {
		t.Error("the plan file was not written")
	}
}

func TestAddingAndRemoving(t *testing.T) {
	_, handler := project(t)

	plan := decodePlan(t, request(t, handler, http.MethodPost, "/api/task",
		`{"title":"OAuth providers","parent":"auth"}`))
	if plan.Saved != "oauth-providers" {
		t.Errorf("the new id came back as %v", plan.Saved)
	}
	if len(plan.Nodes) != 4 {
		t.Fatalf("%d nodes after adding", len(plan.Nodes))
	}

	// Without cascade a child moves up rather than disappearing.
	plan = decodePlan(t, request(t, handler, http.MethodDelete, "/api/task/auth", ""))
	ids := map[string]string{}
	for _, n := range plan.Nodes {
		ids[n.ID] = n.Parent
	}
	if _, gone := ids["auth"]; gone {
		t.Error("the task was not removed")
	}
	if ids["oauth-providers"] != "mvp" {
		t.Errorf("the child should have moved up, its parent is %q", ids["oauth-providers"])
	}
}

func TestRejectedEditsChangeNothing(t *testing.T) {
	server, handler := project(t)

	for _, bad := range []struct{ path, body string }{
		{"/api/task/auth", `{"status":"nearly"}`},
		{"/api/task/mvp", `{"parent":"auth"}`}, // a parent under its own child
		{"/api/task/nope", `{"title":"ghost"}`},
	} {
		rec := request(t, handler, http.MethodPatch, bad.path, bad.body)
		if rec.Code < 400 {
			t.Errorf("%s %s was accepted", bad.path, bad.body)
		}
		if !strings.Contains(rec.Body.String(), "error") {
			t.Errorf("the page should get something to show: %s", rec.Body.String())
		}
	}

	// A task with no title is not a task.
	if rec := request(t, handler, http.MethodPost, "/api/task", `{"title":"   "}`); rec.Code < 400 {
		t.Error("an empty title was accepted")
	}

	saved, err := store.Load(server.Root)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.Nodes) != 3 {
		t.Fatalf("the plan changed after rejected edits: %d nodes", len(saved.Nodes))
	}
	if n, _ := saved.Get("mvp"); n.Parent != "" {
		t.Error("a rejected move was written anyway")
	}
}

func TestPreviewsRenderWithoutWriting(t *testing.T) {
	server, handler := project(t)

	cases := map[string]string{
		"tree":     "<svg",
		"board":    "<svg",
		"page":     "<!doctype html>",
		"mermaid":  "```mermaid",
		"terminal": "MVP",
		"yaml":     "project:",
	}
	for view, want := range cases {
		rec := request(t, handler, http.MethodGet, "/api/view/"+view, "")
		if rec.Code != http.StatusOK {
			t.Errorf("%s: status %d", view, rec.Code)
			continue
		}
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("%s: missing %q", view, want)
		}
	}

	if rec := request(t, handler, http.MethodGet, "/api/view/nonsense", ""); rec.Code != http.StatusNotFound {
		t.Errorf("an unknown view returned %d", rec.Code)
	}

	// A preview on every keystroke must not touch the working tree.
	if _, err := store.Load(server.Root); err != nil {
		t.Fatal(err)
	}
	if entries, _ := readDir(server.Root); len(entries) != 1 {
		t.Errorf("previewing wrote something: %v", entries)
	}
}

func TestPreviewFollowsThemeAndRoot(t *testing.T) {
	_, handler := project(t)

	dark := request(t, handler, http.MethodGet, "/api/view/tree?theme=dark", "").Body.String()
	if !strings.Contains(dark, "#0d1117") {
		t.Error("the dark preview is not dark")
	}

	sliced := request(t, handler, http.MethodGet, "/api/view/tree?root=auth", "").Body.String()
	if strings.Contains(sliced, "Stripe") {
		t.Error("--root should narrow the preview to one branch")
	}
}

func TestWritingOutputsIsExplicit(t *testing.T) {
	server, handler := project(t)

	rec := request(t, handler, http.MethodPost, "/api/render", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "TREE.md") {
		t.Errorf("the answer should name what was written: %s", rec.Body.String())
	}

	entries, err := readDir(server.Root)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, name := range entries {
		if name == "TREE.md" {
			found = true
		}
	}
	if !found {
		t.Errorf("TREE.md was not written, found %v", entries)
	}
}

func TestMethodsAreChecked(t *testing.T) {
	_, handler := project(t)

	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/api/task"},
		{http.MethodGet, "/api/render"},
		{http.MethodPut, "/api/task/auth"},
	} {
		if rec := request(t, handler, c.method, c.path, ""); rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s %s returned %d", c.method, c.path, rec.Code)
		}
	}
}

// readDir lists the entries in a directory, for tests that check nothing was
// written when nothing should have been.
func readDir(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}

func TestDocumentsAreDescribedByName(t *testing.T) {
	// The name is the only place the drawing is recorded, so the editor has to
	// read it back exactly the way the writer reads it.
	server, handler := project(t)

	saved, err := store.Load(server.Root)
	if err != nil {
		t.Fatal(err)
	}
	saved.Outputs = []string{"TREE.md", "docs/tree.svg", "docs/board-dark.svg", "docs/plan.html"}
	if err := store.Save(server.Root, saved); err != nil {
		t.Fatal(err)
	}

	plan := decodePlan(t, request(t, handler, http.MethodGet, "/api/plan", ""))
	want := []struct{ path, kind, theme string }{
		{"TREE.md", "markdown", ""},
		{"docs/tree.svg", "tree", "light"},
		{"docs/board-dark.svg", "board", "dark"},
		{"docs/plan.html", "page", ""},
	}
	if len(plan.Docs) != len(want) {
		t.Fatalf("%d documents", len(plan.Docs))
	}
	for i, w := range want {
		got := plan.Docs[i]
		if got.Path != w.path || got.Kind != w.kind || got.Theme != w.theme {
			t.Errorf("%s came back as %+v", w.path, got)
		}
	}
}

func TestAddingAndDroppingADocument(t *testing.T) {
	server, handler := project(t)

	plan := decodePlan(t, request(t, handler, http.MethodPost, "/api/document", `{"path":"docs/board.svg"}`))
	if len(plan.Outputs) != 2 || plan.Outputs[1] != "docs/board.svg" {
		t.Fatalf("outputs are %v", plan.Outputs)
	}

	// Naming a destination must not write it: that is what the render button
	// is for, and an editor that writes on every keystroke is a diff machine.
	if _, err := os.Stat(server.Root + "/docs/board.svg"); err == nil {
		t.Error("adding a document wrote it")
	}

	if rec := request(t, handler, http.MethodPost, "/api/document", `{"path":"docs/board.svg"}`); rec.Code != http.StatusConflict {
		t.Errorf("a duplicate returned %d", rec.Code)
	}

	plan = decodePlan(t, request(t, handler, http.MethodDelete, "/api/document/docs/board.svg", ""))
	if len(plan.Outputs) != 1 {
		t.Errorf("outputs are %v after dropping one", plan.Outputs)
	}
	if rec := request(t, handler, http.MethodDelete, "/api/document/nope.svg", ""); rec.Code != http.StatusNotFound {
		t.Errorf("dropping something that is not an output returned %d", rec.Code)
	}
}

func TestDocumentsStayInsideTheRepository(t *testing.T) {
	server, handler := project(t)

	for _, bad := range []string{
		`{"path":"/etc/passwd"}`,
		`{"path":"../escape.svg"}`,
		`{"path":"C:/Windows/system.svg"}`,
		`{"path":"   "}`,
		`{"path":"docs/plan"}`, // no extension: nothing decides the drawing
	} {
		if rec := request(t, handler, http.MethodPost, "/api/document", bad); rec.Code < 400 {
			t.Errorf("%s was accepted", bad)
		}
	}

	saved, err := store.Load(server.Root)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.Outputs) != 1 {
		t.Errorf("a rejected document changed the plan: %v", saved.Outputs)
	}
}

func TestThePageAsksInItsOwnPanel(t *testing.T) {
	// A browser dialog blocks the page, cannot be styled, and lands wherever the
	// browser decides. Everything the editor asks, it asks in the drawer — so a
	// call to one of these is a regression, not a shortcut.
	_, handler := project(t)
	script := request(t, handler, http.MethodGet, "/app.js", "").Body.String()

	for _, banned := range []string{"confirm(", "alert(", "prompt(", "window.confirm"} {
		if strings.Contains(script, banned) {
			t.Errorf("the page calls %s instead of asking in the drawer", banned)
		}
	}
	for _, want := range []string{"openDrawer", "closeDrawer", "function ask("} {
		if !strings.Contains(script, want) {
			t.Errorf("the page is missing %s", want)
		}
	}
}

func TestTheEmbeddedPageIsNotCached(t *testing.T) {
	// Embedded files all carry the same zero timestamp, so a browser left to its
	// own devices will serve this morning's page against tonight's API.
	_, handler := project(t)
	for _, asset := range []string{"/", "/app.js", "/app.css"} {
		got := request(t, handler, http.MethodGet, asset, "").Header().Get("Cache-Control")
		if !strings.Contains(got, "no-cache") {
			t.Errorf("%s is served with Cache-Control: %q", asset, got)
		}
	}
}

func TestTheEditorTurnsTheSameGlyphTheDrawingDoes(t *testing.T) {
	// One icon set, one behaviour. The glyph names its own moving part, and
	// both the drawing and the page put the motion there.
	_, handler := project(t)

	glyphs := request(t, handler, http.MethodGet, "/api/glyphs", "").Body.String()
	if !strings.Contains(glyphs, icons.MovingPart) {
		t.Error("the page is served glyphs with no moving part named")
	}

	style := request(t, handler, http.MethodGet, "/app.css", "").Body.String()
	if !strings.Contains(style, "."+icons.MovingPart) {
		t.Error("the page never animates the part the glyph nominated")
	}
	if !strings.Contains(style, "prefers-reduced-motion") {
		t.Error("the page's motion is not guarded")
	}
}

func TestTheViewIsChosenFromAList(t *testing.T) {
	// Five tabs spent the header saying what you were not looking at, and gave
	// each view a word where it could have had a sentence.
	_, handler := project(t)

	page := request(t, handler, http.MethodGet, "/", "").Body.String()
	if strings.Contains(page, `role="tablist"`) && strings.Contains(page, `data-view=`) {
		t.Error("the view is still a row of tabs in the markup")
	}
	for _, want := range []string{`id="view-btn"`, `id="view-menu"`, `aria-haspopup="menu"`} {
		if !strings.Contains(page, want) {
			t.Errorf("the view picker is missing %q", want)
		}
	}

	script := request(t, handler, http.MethodGet, "/app.js", "").Body.String()
	for _, view := range []string{"tree", "board", "page", "mermaid", "yaml"} {
		if !strings.Contains(script, `id: '`+view+`'`) {
			t.Errorf("the picker does not offer %q", view)
		}
		// Every view devtree can render must also be reachable from the API,
		// or the list is offering something the server will not draw.
		if rec := request(t, handler, http.MethodGet, "/api/view/"+view, ""); rec.Code != http.StatusOK {
			t.Errorf("the picker offers %q but the server answers %d", view, rec.Code)
		}
	}
}
