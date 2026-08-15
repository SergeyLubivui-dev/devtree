package serve

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/render/svg"
	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

// A document is one destination in `outputs`, described the way the editor
// needs to show it.
//
// Nothing here is stored: the kind and the palette are read back out of the
// file name every time, by the same rules that decide what gets written. That
// is the whole reason `outputs` is a plain list of paths — name a file
// `board-dark.svg` and you have said which drawing and which palette, with no
// second place to keep in sync.
type jsonDoc struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`  // tree | board | page | markdown
	Theme  string `json:"theme"` // light | dark, for the drawings
	Exists bool   `json:"exists"`
	Bytes  int64  `json:"bytes,omitempty"`
}

// documentKind reads the drawing out of the name, exactly as the writer does.
func documentKind(name string) (kind, theme string) {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".svg"):
		if svg.LayoutForFilename(name) == svg.LayoutBoard {
			kind = "board"
		} else {
			kind = "tree"
		}
		return kind, svg.ThemeForFilename(name).Name
	case strings.HasSuffix(lower, ".html"), strings.HasSuffix(lower, ".htm"):
		return "page", ""
	default:
		return "markdown", ""
	}
}

// documents describes every destination the plan names.
func (s *Server) documents(t *tree.Tree) []jsonDoc {
	out := []jsonDoc{}
	for _, name := range t.Outputs {
		kind, theme := documentKind(name)
		doc := jsonDoc{Path: name, Kind: kind, Theme: theme}
		if info, err := os.Stat(path.Join(s.Root, name)); err == nil {
			doc.Exists, doc.Bytes = true, info.Size()
		}
		out = append(out, doc)
	}
	return out
}

// handleDocument adds a destination to the plan.
//
// Adding a document does not write it — `render` does that, and the editor
// says so. The two are kept apart for the same reason previews write nothing:
// naming a file you might want is not the same as putting it in the diff.
func (s *Server) handleDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, http.StatusMethodNotAllowed, fmt.Errorf("use POST"))
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}

	name, err := cleanDocumentPath(req.Path)
	if err != nil {
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
	for _, existing := range t.Outputs {
		if strings.EqualFold(existing, name) {
			fail(w, http.StatusConflict, fmt.Errorf("%s is already an output", name))
			return
		}
	}
	t.Outputs = append(t.Outputs, name)
	s.commit(w, t, name)
}

// handleDocumentByPath removes a destination.
//
// The file itself is left alone. Dropping a name from `outputs` means devtree
// stops writing it; deleting what is already committed is the repository's
// business, and quietly removing a file somebody may have linked to would be a
// surprising thing for an editor to do.
func (s *Server) handleDocumentByPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		fail(w, http.StatusMethodNotAllowed, fmt.Errorf("use DELETE"))
		return
	}

	name := strings.TrimPrefix(r.URL.Path, "/api/document/")
	if name == "" {
		fail(w, http.StatusBadRequest, fmt.Errorf("which document?"))
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	t, err := s.load()
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}

	kept := make([]string, 0, len(t.Outputs))
	found := false
	for _, existing := range t.Outputs {
		if strings.EqualFold(existing, name) {
			found = true
			continue
		}
		kept = append(kept, existing)
	}
	if !found {
		fail(w, http.StatusNotFound, fmt.Errorf("%s is not an output", name))
		return
	}
	t.Outputs = kept
	s.commit(w, t, "")
}

// cleanDocumentPath keeps a destination inside the repository.
//
// The editor is loopback-only, but "the only caller is trusted" is not a thing
// worth relying on when the cost of not relying on it is one function: an
// output is a path in this repository, so anything absolute, anything with a
// drive letter, and anything that climbs out with .. is not one.
func cleanDocumentPath(raw string) (string, error) {
	name := strings.TrimSpace(strings.ReplaceAll(raw, `\`, "/"))
	if name == "" {
		return "", fmt.Errorf("a document needs a file name")
	}
	if strings.HasPrefix(name, "/") || strings.Contains(name, ":") {
		return "", fmt.Errorf("outputs are paths inside the repository")
	}

	cleaned := path.Clean(name)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("outputs are paths inside the repository")
	}
	if strings.HasPrefix(path.Base(cleaned), ".") {
		return "", fmt.Errorf("a document needs a file name")
	}
	if path.Ext(cleaned) == "" {
		return "", fmt.Errorf("the extension decides the drawing: .svg, .html or .md")
	}
	return cleaned, nil
}
