package tree

import (
	"fmt"
	"regexp"
)

// Node is one piece of work: a milestone, a feature, a chore. Every field
// except ID and Title is optional, and the optional ones exist to tie the plan
// back to the things a repository already has — a branch, an issue, a pull
// request, a person.
type Node struct {
	ID     string
	Title  string
	Status Status
	Parent string
	Branch string
	Issue  string
	PR     string
	Owner  string
	Tags   []string
	Note   string

	// children is derived, never stored. Build fills it from Parent so the
	// file format stays a flat list (see the Tree doc comment) while callers
	// still get to walk a real tree.
	children []*Node
}

// Children returns the node's direct children in file order. The slice is
// owned by the tree; treat it as read-only.
func (n *Node) Children() []*Node { return n.children }

// Tree is a whole plan: project metadata plus a flat list of nodes.
//
// The list is flat on purpose. Nesting would mean that adding a task rewrites
// an existing block, which is exactly the shape that makes two feature
// branches conflict. A flat list turns "add a task" into "append lines", so
// parallel branches merge cleanly — and `devtree init` writes a
// `merge=union` rule into .gitattributes to finish the job.
type Tree struct {
	Version int
	Project string
	Repo    string   // e.g. https://github.com/acme/pay — optional
	Outputs []string // files the diagram is written into

	Nodes []*Node

	index map[string]*Node
	roots []*Node
}

// DefaultOutput is where the diagram goes when nobody says otherwise.
const DefaultOutput = "TREE.md"

// New returns an empty, already-built tree for the given project.
func New(project string) *Tree {
	t := &Tree{Version: 1, Project: project, Outputs: []string{DefaultOutput}}
	_ = t.Build() // an empty tree cannot fail validation
	return t
}

// idPattern keeps IDs safe for every place they end up: Mermaid node names,
// shell arguments, anchors in Markdown. ASCII only, and never leading with a
// separator.
var idPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// Build validates the plan and derives the parent/child links.
//
// It is the single gate every mutation goes through, which is why Add, Move,
// and Remove all end with a call to it: if a change would produce a duplicate
// ID, a dangling parent, or a cycle, it is rejected before it can be saved.
func (t *Tree) Build() error {
	t.index = make(map[string]*Node, len(t.Nodes))
	t.roots = nil

	for _, n := range t.Nodes {
		n.children = nil
		if n.ID == "" {
			return fmt.Errorf("node without an id (title %q)", n.Title)
		}
		if !idPattern.MatchString(n.ID) {
			return fmt.Errorf("invalid id %q: use letters, digits, dot, dash, underscore, starting with a letter or digit", n.ID)
		}
		if _, dup := t.index[n.ID]; dup {
			return fmt.Errorf("duplicate id %q", n.ID)
		}
		if n.Status == "" {
			n.Status = Todo
		}
		if !n.Status.Valid() {
			return fmt.Errorf("node %q: unknown status %q (valid: %s)", n.ID, n.Status, StatusList())
		}
		t.index[n.ID] = n
	}

	for _, n := range t.Nodes {
		if n.Parent == "" {
			t.roots = append(t.roots, n)
			continue
		}
		if n.Parent == n.ID {
			return fmt.Errorf("node %q is its own parent", n.ID)
		}
		p, ok := t.index[n.Parent]
		if !ok {
			return fmt.Errorf("node %q points at a parent that does not exist: %q", n.ID, n.Parent)
		}
		p.children = append(p.children, n)
	}

	// Cycle check: walk up from every node. A cycle has no root to reach, so
	// the walk would loop forever without the seen set.
	for _, n := range t.Nodes {
		seen := map[string]bool{n.ID: true}
		for cur := n; cur.Parent != ""; cur = t.index[cur.Parent] {
			if seen[cur.Parent] {
				return fmt.Errorf("cycle in the tree: %q eventually points back at itself", n.ID)
			}
			seen[cur.Parent] = true
		}
	}
	return nil
}

// Get looks a node up by ID.
func (t *Tree) Get(id string) (*Node, bool) {
	n, ok := t.index[id]
	return n, ok
}

// Roots returns the top-level nodes in file order.
func (t *Tree) Roots() []*Node { return t.roots }

// Subtree returns n followed by every descendant, depth first.
func (t *Tree) Subtree(n *Node) []*Node {
	out := []*Node{n}
	for _, c := range n.children {
		out = append(out, t.Subtree(c)...)
	}
	return out
}

// Progress counts finished work below n.
//
// The node itself is excluded — a milestone is a container, and counting it
// would let a parent show progress it did not earn. Dropped work is excluded
// too, so cancelling a task raises the percentage instead of freezing it.
func (t *Tree) Progress(n *Node) (done, total int) {
	for _, x := range t.Subtree(n) {
		if x == n || x.Status == Dropped {
			continue
		}
		total++
		if x.Status == Done {
			done++
		}
	}
	return done, total
}

// Finished reports whether a node and everything under it has been settled —
// every task either done or dropped.
//
// It is the rule that decides what may leave the active plan for the archive.
// Anything looser would let live work disappear from the board along with the
// milestone above it.
func (t *Tree) Finished(n *Node) bool {
	for _, x := range t.Subtree(n) {
		if x.Status != Done && x.Status != Dropped {
			return false
		}
	}
	return true
}

// Archivable returns the topmost nodes whose whole subtree is finished.
//
// Only the topmost: if a milestone is finished then its children are finished
// too, and listing them separately would offer to archive the same work twice.
func (t *Tree) Archivable() []*Node {
	var out []*Node
	var walk func(n *Node)
	walk = func(n *Node) {
		if t.Finished(n) {
			out = append(out, n)
			return
		}
		for _, c := range n.children {
			walk(c)
		}
	}
	for _, r := range t.roots {
		walk(r)
	}
	return out
}

// Totals counts the whole plan, using the same rule about dropped work.
func (t *Tree) Totals() (done, total int) {
	for _, n := range t.Nodes {
		if n.Status == Dropped {
			continue
		}
		total++
		if n.Status == Done {
			done++
		}
	}
	return done, total
}

// Add appends a node and revalidates. On failure the tree is left as it was.
func (t *Tree) Add(n *Node) error {
	if n.Status == "" {
		n.Status = Todo
	}
	t.Nodes = append(t.Nodes, n)
	if err := t.Build(); err != nil {
		t.Nodes = t.Nodes[:len(t.Nodes)-1]
		_ = t.Build()
		return err
	}
	return nil
}

// Remove deletes a node and reports how many nodes went away.
//
// Without cascade the children are re-parented to the deleted node's parent,
// because dropping a milestone should not silently delete the work under it.
// With cascade the whole subtree goes.
func (t *Tree) Remove(id string, cascade bool) (int, error) {
	n, ok := t.index[id]
	if !ok {
		return 0, fmt.Errorf("no such node: %q", id)
	}

	doomed := map[string]bool{}
	if cascade {
		for _, x := range t.Subtree(n) {
			doomed[x.ID] = true
		}
	} else {
		doomed[n.ID] = true
		for _, c := range n.children {
			c.Parent = n.Parent
		}
	}

	kept := t.Nodes[:0]
	for _, x := range t.Nodes {
		if !doomed[x.ID] {
			kept = append(kept, x)
		}
	}
	t.Nodes = kept
	return len(doomed), t.Build()
}

// Move re-parents a node. An empty parent (or "root") sends it to the top.
//
// The change is applied, validated, and rolled back on failure, which is what
// stops a node from being moved underneath its own descendant.
func (t *Tree) Move(id, parent string) error {
	n, ok := t.index[id]
	if !ok {
		return fmt.Errorf("no such node: %q", id)
	}
	if parent == "root" || parent == "-" {
		parent = ""
	}
	if parent != "" {
		if _, ok := t.index[parent]; !ok {
			return fmt.Errorf("no such parent: %q", parent)
		}
	}

	previous := n.Parent
	n.Parent = parent
	if err := t.Build(); err != nil {
		n.Parent = previous
		_ = t.Build()
		return err
	}
	return nil
}

// NextID picks a free, human-readable ID derived from a title. Collisions get
// a numeric suffix rather than a hash, because `devtree done oauth-2` is
// something a person can type from memory.
func (t *Tree) NextID(title string) string {
	base := Slugify(title)
	if len(base) < 2 {
		base = "task"
	}
	if _, taken := t.index[base]; !taken {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if _, taken := t.index[candidate]; !taken {
			return candidate
		}
	}
}
