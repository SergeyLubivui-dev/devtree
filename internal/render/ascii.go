package render

import (
	"fmt"
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

// ASCII draws the plan for a terminal. An empty filter shows everything;
// otherwise only matching nodes are printed, while the tree is still walked in
// full so nothing below a hidden node disappears with it.
func ASCII(t *tree.Tree, filter Filter) string {
	var b strings.Builder

	var walk func(n *tree.Node, prefix string, last bool, depth int)
	walk = func(n *tree.Node, prefix string, last bool, depth int) {
		branch, childPrefix := "├─ ", prefix+"│  "
		if last {
			branch, childPrefix = "└─ ", prefix+"   "
		}
		if depth == 0 {
			// Roots carry no connector: nothing sits above them to connect to.
			branch, childPrefix = "", ""
		}

		if filter.Match(n) {
			var meta strings.Builder
			if n.Branch != "" {
				fmt.Fprintf(&meta, "  %s", n.Branch)
			}
			if n.Issue != "" {
				fmt.Fprintf(&meta, "  #%s", n.Issue)
			}
			if n.Owner != "" {
				fmt.Fprintf(&meta, "  @%s", n.Owner)
			}
			if len(n.Children()) > 0 {
				done, total := t.Progress(n)
				fmt.Fprintf(&meta, "  [%d/%d]", done, total)
			}
			fmt.Fprintf(&b, "%s%s%s %s  (%s)%s\n",
				prefix, branch, n.Status.Icon(), n.Title, n.ID, meta.String())
		}

		children := n.Children()
		for i, c := range children {
			walk(c, childPrefix, i == len(children)-1, depth+1)
		}
	}

	roots := t.Roots()
	for i, root := range roots {
		walk(root, "", i == len(roots)-1, 0)
	}

	if b.Len() == 0 {
		return "(nothing to show)\n"
	}
	return b.String()
}
