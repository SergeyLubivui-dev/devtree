package render

import (
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

// Filter narrows what a view shows.
//
// It lives here rather than in each command because the tree and the board ask
// the same questions of a plan — whose is it, what is it tagged, what state is
// it in — and answering them twice is how two views start disagreeing.
//
// An empty Filter matches everything, which is what makes it safe to pass
// unconditionally instead of branching at every call site.
type Filter struct {
	Status tree.Status
	Owner  string
	Tags   []string
}

// Empty reports whether the filter would let everything through.
func (f Filter) Empty() bool {
	return f.Status == "" && f.Owner == "" && len(f.Tags) == 0
}

// Match reports whether a node survives the filter. Conditions combine with
// and; tags combine with or, because "show me anything touching billing or
// payments" is the question people actually ask.
func (f Filter) Match(n *tree.Node) bool {
	if f.Status != "" && n.Status != f.Status {
		return false
	}
	if f.Owner != "" && !strings.EqualFold(strings.TrimPrefix(n.Owner, "@"), f.Owner) {
		return false
	}
	if len(f.Tags) > 0 && !hasAnyTag(n, f.Tags) {
		return false
	}
	return true
}

func hasAnyTag(n *tree.Node, wanted []string) bool {
	for _, want := range wanted {
		for _, tag := range n.Tags {
			if strings.EqualFold(tag, want) {
				return true
			}
		}
	}
	return false
}

// Describe names the filter in one line, for a view that wants to say why it
// is showing so little.
func (f Filter) Describe() string {
	var parts []string
	if f.Status != "" {
		parts = append(parts, f.Status.Label())
	}
	if f.Owner != "" {
		parts = append(parts, "@"+f.Owner)
	}
	for _, tag := range f.Tags {
		parts = append(parts, "#"+tag)
	}
	return strings.Join(parts, " · ")
}
