// Package serve is the local editor: a two-pane page on 127.0.0.1 where the
// plan is edited on the left and drawn, live, on the right.
//
// It is the first thing in devtree that opens a port, so the boundaries are
// worth stating. The server binds to the loopback interface and nothing else.
// It is started by an explicit command and dies with it. It stores nothing:
// every change goes through the same domain rules and lands in the same
// .devtree/tree.yaml a person could have typed by hand, which means the editor
// and the command line can be used in the same minute without either one
// losing an edit.
//
// There is no build step and no framework. The page is three files embedded in
// the binary, and the whole API is a handful of JSON endpoints.
package serve

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/SergeyLubivui-dev/devtree/internal/icons"
	"github.com/SergeyLubivui-dev/devtree/internal/render"
	"github.com/SergeyLubivui-dev/devtree/internal/render/html"
	"github.com/SergeyLubivui-dev/devtree/internal/render/svg"
	"github.com/SergeyLubivui-dev/devtree/internal/store"
	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

//go:embed ui
var ui embed.FS

// DefaultPort is where the editor listens unless told otherwise.
const DefaultPort = 9312

// Server edits one project.
type Server struct {
	Root string    // the project root, the one holding .devtree
	Log  io.Writer // where requests worth mentioning are reported

	// mu serializes writes. Two browser tabs and a terminal can all be pointed
	// at one plan; a read-modify-write race would lose whichever edit finished
	// second, silently.
	mu sync.Mutex
}

// Handler wires the routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	pages, err := fs.Sub(ui, "ui")
	if err != nil {
		panic(err) // the embedded directory is compiled in; it cannot be missing
	}
	// The page is compiled into the binary, and embedded files all carry the
	// same zero timestamp — so a browser that caches them will happily serve a
	// page from the devtree you replaced this morning against the API of the
	// one running now. Revalidating costs nothing over loopback.
	mux.Handle("/", noStore(http.FileServer(http.FS(pages))))

	mux.HandleFunc("/api/glyphs", s.handleGlyphs)
	mux.HandleFunc("/api/plan", s.handlePlan)
	mux.HandleFunc("/api/task", s.handleTask)
	mux.HandleFunc("/api/task/", s.handleTaskByID)
	mux.HandleFunc("/api/document", s.handleDocument)
	mux.HandleFunc("/api/document/", s.handleDocumentByPath)
	mux.HandleFunc("/api/render", s.handleWriteOutputs)
	mux.HandleFunc("/api/view/", s.handleView)
	mux.HandleFunc("/api/events", s.handleEvents)

	return mux
}

// DefaultHost is the only interface an editor with no authentication has any
// business listening on.
const DefaultHost = "127.0.0.1"

// ListenAndServe starts the editor.
//
// The host is a knob rather than a constant for exactly one reason: inside a
// container, loopback means the container's own loopback, and a published port
// would reach nothing. Anywhere else, changing it hands an unauthenticated
// editor to the network, which is why the CLI says so out loud.
func (s *Server) ListenAndServe(host string, port int) error {
	if host == "" {
		host = DefaultHost
	}
	addr := net.JoinHostPort(host, fmt.Sprint(port))

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("cannot listen on %s: %w", addr, err)
	}

	server := &http.Server{
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return server.Serve(listener)
}

// URL is where a browser should be pointed. A server bound to every interface
// is still reached from this machine over loopback, so that is what is shown.
func URL(port int) string { return fmt.Sprintf("http://127.0.0.1:%d/", port) }

// ---------------------------------------------------------------------------
// the plan, as the page sees it
// ---------------------------------------------------------------------------

// jsonNode is one task, flattened. Depth rather than nesting: the page draws a
// list, and a list is easier to keep in sync with a filter than a tree of
// elements is.
type jsonNode struct {
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Status string   `json:"status"`
	Parent string   `json:"parent"`
	Branch string   `json:"branch,omitempty"`
	Issue  string   `json:"issue,omitempty"`
	PR     string   `json:"pr,omitempty"`
	Owner  string   `json:"owner,omitempty"`
	Tags   []string `json:"tags,omitempty"`
	Note   string   `json:"note,omitempty"`

	Depth int    `json:"depth"`
	Done  int    `json:"done"`
	Total int    `json:"total"`
	URL   string `json:"url,omitempty"`
}

type jsonPlan struct {
	Project  string      `json:"project"`
	Repo     string      `json:"repo"`
	Outputs  []string    `json:"outputs"`
	Docs     []jsonDoc   `json:"docs"`
	Statuses []jsonStat  `json:"statuses"`
	Nodes    []jsonNode  `json:"nodes"`
	Totals   [2]int      `json:"totals"`
	Warnings []string    `json:"warnings"`
	Saved    interface{} `json:"saved,omitempty"`
}

type jsonStat struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Icon  string `json:"icon"`
}

// handleGlyphs hands over the vendored icon set.
//
// The page could have drawn its own little circles, and the first version did.
// Then the editor's idea of "blocked" and the drawing's idea of "blocked" were
// two different pictures maintained in two places. Now there is one set, and
// it is the one compiled into the binary.
func (s *Server) handleGlyphs(w http.ResponseWriter, r *http.Request) {
	out := map[string]string{}
	for _, name := range icons.Names() {
		if body, ok := icons.Get(name); ok {
			out[name] = body
		}
	}
	writeJSON(w, out)
}

func (s *Server) handlePlan(w http.ResponseWriter, r *http.Request) {
	t, err := s.load()
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, s.describe(t))
}

// describe turns a plan into what the page needs, in tree order.
func (s *Server) describe(t *tree.Tree) jsonPlan {
	out := jsonPlan{
		Project: t.Project,
		Repo:    t.Repo,
		Outputs: t.Outputs,
		Docs:    s.documents(t),
	}
	for _, status := range tree.Statuses {
		out.Statuses = append(out.Statuses, jsonStat{
			Name:  string(status),
			Label: status.Label(),
			Icon:  status.Icon(),
		})
	}

	var walk func(nodes []*tree.Node, depth int)
	walk = func(nodes []*tree.Node, depth int) {
		for _, n := range nodes {
			done, total := t.Progress(n)
			url, _ := render.URL(t, n, "")
			out.Nodes = append(out.Nodes, jsonNode{
				ID: n.ID, Title: n.Title, Status: string(n.Status), Parent: n.Parent,
				Branch: n.Branch, Issue: n.Issue, PR: n.PR, Owner: n.Owner,
				Tags: n.Tags, Note: n.Note,
				Depth: depth, Done: done, Total: total, URL: url,
			})
			walk(n.Children(), depth+1)
		}
	}
	walk(t.Roots(), 0)

	done, total := t.Totals()
	out.Totals = [2]int{done, total}
	out.Warnings = warnings(t)
	return out
}

// warnings mirrors what `devtree check` would say, so the editor can show the
// same advice without the user having to run anything.
func warnings(t *tree.Tree) []string {
	var out []string
	for _, n := range t.Nodes {
		if strings.TrimSpace(n.Title) == "" {
			out = append(out, n.ID+" has no title")
		}
		if n.Status == tree.Blocked && strings.TrimSpace(n.Note) == "" {
			out = append(out, n.ID+" is blocked with no note saying why")
		}
		if n.Status == tree.Done {
			for _, c := range n.Children() {
				if c.Status != tree.Done && c.Status != tree.Dropped {
					out = append(out, fmt.Sprintf("%s is done, but %s is still %s",
						n.ID, c.ID, c.Status.Label()))
				}
			}
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// mutations
// ---------------------------------------------------------------------------

type taskRequest struct {
	Title  *string   `json:"title"`
	Status *string   `json:"status"`
	Parent *string   `json:"parent"`
	Branch *string   `json:"branch"`
	Issue  *string   `json:"issue"`
	PR     *string   `json:"pr"`
	Owner  *string   `json:"owner"`
	Tags   *[]string `json:"tags"`
	Note   *string   `json:"note"`
}

// handleTask adds a task.
func (s *Server) handleTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, http.StatusMethodNotAllowed, fmt.Errorf("use POST to add a task"))
		return
	}

	var req taskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	t, err := s.load()
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}

	title := strings.TrimSpace(deref(req.Title))
	if title == "" {
		fail(w, http.StatusBadRequest, fmt.Errorf("a task needs a title"))
		return
	}

	n := &tree.Node{ID: t.NextID(title), Title: title, Status: tree.Todo}
	if err := apply(n, req); err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	if err := t.Add(n); err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	s.commit(w, t, n.ID)
}

// handleTaskByID edits or removes one task.
func (s *Server) handleTaskByID(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/task/"), "/")
	if id == "" {
		fail(w, http.StatusBadRequest, fmt.Errorf("name a task"))
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	t, err := s.load()
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	n, ok := t.Get(id)
	if !ok {
		fail(w, http.StatusNotFound, fmt.Errorf("no such task: %q", id))
		return
	}

	switch r.Method {
	case http.MethodPatch, http.MethodPost:
		var req taskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			fail(w, http.StatusBadRequest, err)
			return
		}
		if err := apply(n, req); err != nil {
			fail(w, http.StatusBadRequest, err)
			return
		}
		// Re-parenting is validated by Move, which rolls itself back, so a
		// rejected drop leaves the plan exactly as it was.
		if req.Parent != nil {
			if err := t.Move(n.ID, *req.Parent); err != nil {
				fail(w, http.StatusBadRequest, err)
				return
			}
		}
		if err := t.Build(); err != nil {
			fail(w, http.StatusBadRequest, err)
			return
		}
		s.commit(w, t, n.ID)

	case http.MethodDelete:
		cascade := r.URL.Query().Get("cascade") == "true"
		if _, err := t.Remove(id, cascade); err != nil {
			fail(w, http.StatusBadRequest, err)
			return
		}
		s.commit(w, t, "")

	default:
		fail(w, http.StatusMethodNotAllowed, fmt.Errorf("use PATCH or DELETE"))
	}
}

// apply copies the fields a request actually carried. A nil pointer means "not
// mentioned"; an empty string means "clear it" — the same distinction the CLI
// makes between an absent flag and an empty one.
func apply(n *tree.Node, req taskRequest) error {
	if req.Title != nil {
		n.Title = strings.TrimSpace(*req.Title)
	}
	if req.Status != nil {
		status, err := tree.ParseStatus(*req.Status)
		if err != nil {
			return err
		}
		n.Status = status
	}
	if req.Branch != nil {
		n.Branch = strings.TrimSpace(*req.Branch)
	}
	if req.Issue != nil {
		n.Issue = strings.TrimPrefix(strings.TrimSpace(*req.Issue), "#")
	}
	if req.PR != nil {
		n.PR = strings.TrimPrefix(strings.TrimSpace(*req.PR), "#")
	}
	if req.Owner != nil {
		n.Owner = strings.TrimPrefix(strings.TrimSpace(*req.Owner), "@")
	}
	if req.Note != nil {
		n.Note = strings.TrimSpace(*req.Note)
	}
	if req.Tags != nil {
		var tags []string
		for _, tag := range *req.Tags {
			if tag = strings.TrimSpace(tag); tag != "" {
				tags = append(tags, tag)
			}
		}
		n.Tags = tags
	}
	if req.Parent != nil && n.Parent == "" {
		n.Parent = strings.TrimSpace(*req.Parent)
	}
	return nil
}

// commit saves the plan and answers with the new state, so the page never has
// to guess what the server made of its request.
func (s *Server) commit(w http.ResponseWriter, t *tree.Tree, saved string) {
	if err := store.Save(s.Root, t); err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	plan := s.describe(t)
	if saved != "" {
		plan.Saved = saved
	}
	writeJSON(w, plan)
}

// handleWriteOutputs runs what `devtree render` runs.
func (s *Server) handleWriteOutputs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, http.StatusMethodNotAllowed, fmt.Errorf("use POST"))
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	t, err := s.load()
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}

	written := []string{}
	for _, output := range t.Outputs {
		full := path.Join(s.Root, output)
		var content string
		switch {
		case strings.HasSuffix(strings.ToLower(output), ".svg"):
			content = svg.RenderFor(t, output)
		case strings.HasSuffix(strings.ToLower(output), ".html"),
			strings.HasSuffix(strings.ToLower(output), ".htm"):
			content = html.Render(t)
		default:
			existing, _ := os.ReadFile(full)
			updated, _ := render.Inject(string(existing), render.Block(t))
			content = updated
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			fail(w, http.StatusInternalServerError, err)
			return
		}
		written = append(written, output)
	}

	writeJSON(w, map[string]any{"written": written})
}

// ---------------------------------------------------------------------------
// previews
// ---------------------------------------------------------------------------

// handleView renders without writing anything. The page asks for these on
// every change, and a preview that touched the working tree would turn every
// keystroke into a diff.
func (s *Server) handleView(w http.ResponseWriter, r *http.Request) {
	kind := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/view/"), "/")
	theme := svg.ThemeFor(r.URL.Query().Get("theme"))

	t, err := s.load()
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	if root := r.URL.Query().Get("root"); root != "" {
		if sliced, err := t.Slice(root); err == nil {
			t = sliced
		}
	}

	w.Header().Set("Cache-Control", "no-store")
	switch kind {
	case "tree":
		w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
		io.WriteString(w, svg.Render(t, theme))
	case "board":
		w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
		io.WriteString(w, svg.Board(t, theme))
	case "page":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, html.Render(t))
	case "mermaid":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(w, render.Block(t))
	case "terminal":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(w, render.ASCII(t, render.Filter{}))
	case "yaml":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write(store.Marshal(t))
	default:
		fail(w, http.StatusNotFound, fmt.Errorf("no such view: %q", kind))
	}
}

// handleEvents tells the page when the plan changed underneath it.
//
// Polling the file's modification time rather than watching it: a watcher would
// mean a dependency, and this is a local editor refreshing a preview, not a
// build system. The page listens, so an edit made in a terminal shows up in the
// browser within a second.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		fail(w, http.StatusInternalServerError, fmt.Errorf("streaming is not supported here"))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Connection", "keep-alive")

	var last time.Time
	if info, err := os.Stat(store.Path(s.Root)); err == nil {
		last = info.ModTime()
	}

	ticker := time.NewTicker(700 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			info, err := os.Stat(store.Path(s.Root))
			if err != nil {
				continue
			}
			if info.ModTime().After(last) {
				last = info.ModTime()
				fmt.Fprint(w, "event: changed\ndata: {}\n\n")
				flusher.Flush()
			}
		}
	}
}

// ---------------------------------------------------------------------------
// plumbing
// ---------------------------------------------------------------------------

func (s *Server) load() (*tree.Tree, error) { return store.Load(s.Root) }

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// fail answers with a message the page can show as it is. Every error a user
// can cause here is one they can fix, so it is worth spelling out rather than
// reducing to a status code.
func fail(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// noStore keeps a browser from holding on to the embedded page.
func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")
		next.ServeHTTP(w, r)
	})
}
