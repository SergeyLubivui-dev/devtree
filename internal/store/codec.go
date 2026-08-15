// Package store reads and writes .devtree/tree.yaml.
//
// The format is a deliberately strict subset of YAML: a flat list of nodes
// whose fields are all scalars. That subset is parsed by hand here instead of
// pulling in a YAML library, and the trade is a good one — devtree stays a
// single dependency-free binary, and the parser can report the exact line a
// mistake is on, which a general-purpose loader rarely does as clearly.
//
// What is *not* supported, and never will be: anchors, aliases, multi-line
// block scalars, nested mappings. If you need those, you are describing
// something more complicated than a plan.
package store

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

// Parse reads the stored format into a tree. The result is not validated yet;
// call Build (Load does it for you) to check IDs, parents, and cycles.
func Parse(data []byte) (*tree.Tree, error) {
	t := &tree.Tree{Version: 1}
	var current *tree.Node
	inNodes := false

	for i, raw := range strings.Split(normalize(data), "\n") {
		line := strings.TrimRight(raw, " \t")
		trimmed := strings.TrimSpace(line)
		lineNo := i + 1

		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Indentation is the only structure this format has: unindented lines
		// are document keys, indented lines belong to the current node.
		if line == strings.TrimLeft(line, " \t") {
			key, value, err := splitKeyValue(trimmed, lineNo)
			if err != nil {
				return nil, err
			}
			current = nil
			if err := applyDocumentKey(t, key, value, lineNo); err != nil {
				return nil, err
			}
			if key == "nodes" {
				inNodes = true
			}
			continue
		}

		if !inNodes {
			return nil, fmt.Errorf("line %d: indented field outside the nodes list", lineNo)
		}

		if strings.HasPrefix(trimmed, "-") {
			current = &tree.Node{}
			t.Nodes = append(t.Nodes, current)
			rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
			if rest == "" {
				continue // "- " on its own line, fields follow
			}
			trimmed = rest
		}
		if current == nil {
			return nil, fmt.Errorf("line %d: node field outside a list item (missing %q prefix)", lineNo, "- ")
		}

		key, value, err := splitKeyValue(trimmed, lineNo)
		if err != nil {
			return nil, err
		}
		if err := assign(current, key, unquote(value), lineNo); err != nil {
			return nil, err
		}
	}

	if len(t.Outputs) == 0 {
		t.Outputs = []string{tree.DefaultOutput}
	}
	return t, nil
}

// applyDocumentKey applies a top-level key.
func applyDocumentKey(t *tree.Tree, key, value string, lineNo int) error {
	switch key {
	case "nodes":
		return nil
	case "version":
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return fmt.Errorf("line %d: version must be a number, got %q", lineNo, value)
		}
		t.Version = n
	case "project":
		t.Project = unquote(value)
	case "repo":
		t.Repo = strings.TrimRight(unquote(value), "/")
	case "outputs":
		t.Outputs = SplitList(unquote(value))
	default:
		return fmt.Errorf("line %d: unknown key %q", lineNo, key)
	}
	return nil
}

// assign applies a node field.
func assign(n *tree.Node, key, value string, lineNo int) error {
	switch key {
	case "id":
		n.ID = value
	case "title":
		n.Title = value
	case "status":
		s, err := tree.ParseStatus(value)
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNo, err)
		}
		n.Status = s
	case "parent":
		n.Parent = value
	case "branch":
		n.Branch = value
	case "issue":
		// Stored bare so links can be built without guessing; "#12" and "12"
		// are both accepted from humans editing the file.
		n.Issue = strings.TrimPrefix(value, "#")
	case "pr":
		n.PR = strings.TrimPrefix(value, "#")
	case "owner":
		n.Owner = strings.TrimPrefix(value, "@")
	case "tags":
		n.Tags = SplitList(value)
	case "note":
		n.Note = value
	default:
		return fmt.Errorf("line %d: unknown node field %q", lineNo, key)
	}
	return nil
}

// Marshal renders a tree back into the stored format.
//
// Field order is fixed and empty fields are omitted, so a round trip through
// Parse and Marshal is byte-stable. That is what keeps `devtree render` out of
// your diffs when nothing actually changed.
func Marshal(t *tree.Tree) []byte {
	var b strings.Builder

	b.WriteString("# Development tree. Edit by hand or through the `devtree` command.\n")
	b.WriteString("# After editing, run `devtree render` to refresh the diagram.\n")
	fmt.Fprintf(&b, "version: %d\n", t.Version)
	fmt.Fprintf(&b, "project: %s\n", quote(t.Project))
	if t.Repo != "" {
		fmt.Fprintf(&b, "repo: %s\n", quote(t.Repo))
	}
	outputs := t.Outputs
	if len(outputs) == 0 {
		outputs = []string{tree.DefaultOutput}
	}
	fmt.Fprintf(&b, "outputs: %s\n", quote(strings.Join(outputs, ", ")))
	b.WriteString("nodes:\n")

	optional := func(key, value string) {
		if value != "" {
			fmt.Fprintf(&b, "    %s: %s\n", key, quote(value))
		}
	}
	for _, n := range t.Nodes {
		status := n.Status
		if status == "" {
			status = tree.Todo
		}
		fmt.Fprintf(&b, "  - id: %s\n", quote(n.ID))
		fmt.Fprintf(&b, "    title: %s\n", quote(n.Title))
		fmt.Fprintf(&b, "    status: %s\n", quote(string(status)))
		optional("parent", n.Parent)
		optional("branch", n.Branch)
		optional("issue", n.Issue)
		optional("pr", n.PR)
		optional("owner", n.Owner)
		optional("tags", strings.Join(n.Tags, ", "))
		optional("note", n.Note)
	}
	return []byte(b.String())
}

// normalize strips a UTF-8 BOM and CRLF line endings, both of which Windows
// editors add and neither of which should reach the parser.
func normalize(data []byte) string {
	bom := []byte{0xEF, 0xBB, 0xBF}
	return strings.ReplaceAll(string(bytes.TrimPrefix(data, bom)), "\r\n", "\n")
}

func splitKeyValue(s string, lineNo int) (key, value string, err error) {
	i := strings.Index(s, ":")
	if i < 0 {
		return "", "", fmt.Errorf("line %d: expected \"key: value\", got %q", lineNo, s)
	}
	return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:]), nil
}

// unquote accepts both quoted and bare scalars. Quoted values go through Go's
// unquoter so escapes behave the way the writer produced them.
func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		if v, err := strconv.Unquote(s); err == nil {
			return v
		}
		return strings.ReplaceAll(s[1:len(s)-1], `\"`, `"`)
	}
	return s
}

// quote always writes values quoted. Bare scalars would need the caller to
// know which characters YAML treats specially; quoting sidesteps the whole
// question.
func quote(s string) string { return strconv.Quote(s) }

// SplitList parses a comma-separated field, dropping blanks. Shared with the
// CLI so `--tags a, b` and `tags: "a, b"` behave identically.
func SplitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
