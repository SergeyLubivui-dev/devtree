package render

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

// Board prints the plan grouped by status, for the terminal.
//
// The tree view answers "how is this organized"; the board answers "what is
// everyone on right now". Columns side by side would wrap into porridge on a
// narrow terminal, so the statuses stack instead — the same information, in
// the shape a terminal can actually hold.
//
// Only leaves are listed. A milestone is a container, and a board that mixes
// containers in with the work makes both harder to scan. Each task shows the
// milestone it belongs to instead.
func Board(t *tree.Tree, filter Filter) string {
	groups := map[tree.Status][]*tree.Node{}
	for _, n := range t.Nodes {
		if len(n.Children()) > 0 || !filter.Match(n) {
			continue
		}
		groups[n.Status] = append(groups[n.Status], n)
	}

	// One pass to find the widest title, so the second column of every group
	// lines up down the whole board rather than per group.
	titleWidth := 0
	for _, nodes := range groups {
		for _, n := range nodes {
			if w := utf8.RuneCountInString(title(n)); w > titleWidth {
				titleWidth = w
			}
		}
	}
	if titleWidth > 44 {
		titleWidth = 44
	}

	var b strings.Builder
	for _, s := range tree.Statuses {
		nodes := groups[s]
		if len(nodes) == 0 {
			continue
		}

		fmt.Fprintf(&b, "\n%s %s · %d\n", s.Icon(), s.Label(), len(nodes))
		for _, n := range nodes {
			fmt.Fprintf(&b, "  %s%s\n", padRunes(title(n), titleWidth), context(t, n))
		}
	}

	if b.Len() == 0 {
		return "(nothing on the board)\n"
	}
	return strings.TrimLeft(b.String(), "\n")
}

func title(n *tree.Node) string {
	if n.Title == "" {
		return n.ID
	}
	return n.Title
}

// context is everything to the right of the title: where the task sits, and
// what it is attached to.
func context(t *tree.Tree, n *tree.Node) string {
	var parts []string
	if n.Parent != "" {
		if p, ok := t.Get(n.Parent); ok {
			parts = append(parts, title(p))
		}
	}
	if n.Branch != "" {
		parts = append(parts, n.Branch)
	}
	if n.Issue != "" {
		parts = append(parts, "#"+n.Issue)
	}
	if n.PR != "" {
		parts = append(parts, "!"+n.PR)
	}
	if n.Owner != "" {
		parts = append(parts, "@"+n.Owner)
	}
	if n.Note != "" {
		parts = append(parts, "— "+n.Note)
	}
	return strings.Join(parts, "  ")
}

// padRunes pads to a rune count, not a byte count: a Cyrillic or CJK title
// would otherwise throw the whole column out of alignment.
func padRunes(s string, width int) string {
	if n := utf8.RuneCountInString(s); n < width {
		return s + strings.Repeat(" ", width-n+2)
	}
	return truncateRunes(s, width) + "  "
}

func truncateRunes(s string, width int) string {
	if utf8.RuneCountInString(s) <= width {
		return s
	}
	runes := []rune(s)
	return string(runes[:width-1]) + "…"
}
