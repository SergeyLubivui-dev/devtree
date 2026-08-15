package svg

import (
	"fmt"

	"github.com/SergeyLubivui-dev/devtree/internal/draw"
	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

// Geometry. Everything the renderer measures is derived from these, so the
// diagram can be retuned in one place instead of by hunting magic numbers.
const (
	pad       = 24.0 // canvas margin
	headerH   = 96.0 // wordmark, project name, overall progress
	legendH   = 44.0 // status key along the bottom
	cardH     = 48.0
	rowGap    = 12.0
	colGap    = 46.0 // horizontal space between a parent and its children
	minCardW  = 190.0
	maxCardW  = 300.0
	textLeft  = 40.0 // where a title starts inside a card
	rightPad  = 14.0 // padding after the title when there is no progress block
	rightWide = 56.0 // padding when a progress ratio and bar sit on the right
	groupGap  = 0.7  // extra rows inserted between top-level groups

	iconSize  = 18.0
	metaIcon  = 11.0
	titleSize = 12.5
	metaSize  = 10.5
	accentW   = 4.0 // the status stripe down the left edge of a card
)

// metaItem is one piece of context under a title: a branch, an issue, an owner.
type metaItem struct {
	icon string // name in internal/icons, empty for text-only
	text string
}

// placement is a node with everything the emitter needs, measured once.
type placement struct {
	node  *tree.Node
	depth int
	row   float64 // in card rows; parents land on a half row between children

	title string
	meta  []metaItem
	ratio string // "3/7" for a node with children, empty otherwise
	done  int
	total int

	natural float64 // width this card would like
}

// layout walks the tree and decides where everything goes.
//
// The row assignment is Reingold-Tilford reduced to what a plan needs: leaves
// take the next free row, a parent centers on the span of its children. Depth
// becomes the column. That produces the shape people draw on a whiteboard —
// milestones on the left, work fanning out to the right.
func layout(t *tree.Tree) (places []placement, rows float64, columns []float64) {
	var next float64
	var walk func(n *tree.Node, depth int)

	walk = func(n *tree.Node, depth int) {
		p := placement{node: n, depth: depth, title: n.Title}
		if p.title == "" {
			p.title = n.ID
		}

		children := n.Children()
		if len(children) == 0 {
			p.row = next
			next++
		} else {
			first := next
			index := len(places)
			places = append(places, p) // reserve the slot, fill the row after
			for _, c := range children {
				walk(c, depth+1)
			}
			places[index].row = (first + next - 1) / 2
			places[index].done, places[index].total = t.Progress(n)
			places[index].ratio = fmt.Sprintf("%d/%d", places[index].done, places[index].total)
			places[index].meta = metaFor(n)
			places[index].natural = naturalWidth(places[index])
			return
		}

		p.meta = metaFor(n)
		p.natural = naturalWidth(p)
		places = append(places, p)
	}

	roots := t.Roots()
	for i, r := range roots {
		if i > 0 {
			next += groupGap // let top-level groups breathe
		}
		walk(r, 0)
	}

	for _, p := range places {
		for len(columns) <= p.depth {
			columns = append(columns, minCardW)
		}
		if p.natural > columns[p.depth] {
			columns[p.depth] = p.natural
		}
	}
	for i, w := range columns {
		if w > maxCardW {
			columns[i] = maxCardW
		}
	}
	return places, next, columns
}

// metaFor picks the context line for a node. Order is fixed — branch, issue,
// pull request, owner, tags — so two renders of the same plan produce the same
// bytes and `devtree render` stays out of the diff.
func metaFor(n *tree.Node) []metaItem {
	var out []metaItem
	if n.Branch != "" {
		out = append(out, metaItem{"nodes", n.Branch})
	}
	if n.Issue != "" {
		out = append(out, metaItem{"hashtag", n.Issue})
	}
	if n.PR != "" {
		out = append(out, metaItem{"code-circle", n.PR})
	}
	if n.Owner != "" {
		out = append(out, metaItem{"user", n.Owner})
	}
	if len(n.Tags) > 0 {
		out = append(out, metaItem{"bookmark", joinTags(n.Tags)})
	}
	return out
}

func joinTags(tags []string) string {
	out := tags[0]
	for _, t := range tags[1:] {
		out += ", " + t
	}
	return out
}

// metaWidth measures a context line, including the gaps between items.
func metaWidth(items []metaItem) float64 {
	var w float64
	for i, m := range items {
		if i > 0 {
			w += 10
		}
		if m.icon != "" {
			w += metaIcon + 3
		}
		w += draw.TextWidth(m.text, metaSize)
	}
	return w
}

// naturalWidth is the width a card wants before its column votes on it.
//
// The slack matters: the widest card in a column is the one that sets the
// column width, and without a pixel or two of margin it would clip its own
// title against the estimate that produced it.
const widthSlack = 3.0

func naturalWidth(p placement) float64 {
	right := rightPad
	if p.ratio != "" {
		right = rightWide
	}
	content := draw.TextWidth(p.title, titleSize)
	if m := metaWidth(p.meta); m > content {
		content = m
	}
	w := textLeft + content + right + widthSlack
	if w < minCardW {
		w = minCardW
	}
	return w
}
