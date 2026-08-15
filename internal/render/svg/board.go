package svg

import (
	"fmt"
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/draw"
	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

// Board geometry.
const (
	colGutter  = 14.0
	colHead    = 34.0
	boardCardH = 44.0
	boardGap   = 8.0
	minColW    = 150.0
)

// Board draws the plan as columns of work, one per status.
//
// It answers a different question than the tree does. The tree says how the
// work is organized; the board says what state it is in this morning. Both are
// the same file — that is the point of keeping the plan as data.
//
// Only leaves appear. A milestone is a container, not a task, and a board that
// lists containers next to the work inside them stops being a board. Each card
// carries its parent as a breadcrumb instead, so nothing loses its place.
func Board(t *tree.Tree, th Theme) string {
	columns := boardColumns(t)

	width := 800.0
	if n := float64(len(columns)); n > 0 {
		if needed := pad*2 + n*minColW + (n-1)*colGutter; needed > width {
			width = needed
		}
	}

	colW := minColW
	if n := float64(len(columns)); n > 0 {
		colW = (width - pad*2 - (n-1)*colGutter) / n
	}

	tallest := 0
	for _, c := range columns {
		if len(c.cards) > tallest {
			tallest = len(c.cards)
		}
	}
	height := headerH + colHead + float64(tallest)*(boardCardH+boardGap) + pad
	if tallest == 0 {
		height = headerH + colHead + boardCardH + pad
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%.0f" `+
		`viewBox="0 0 %.0f %.0f" font-family="%s" role="img" aria-label="%s">`,
		width, height, width, height, draw.FontStack, draw.Escape(boardLabel(t)))
	b.WriteString(draw.Stylesheet)

	// One clip path per column, shared by every card in it.
	b.WriteString(`<defs>`)
	for i := range columns {
		fmt.Fprintf(&b, `<clipPath id="dt-col-%d"><rect x="0" y="0" width="%.1f" height="%.1f" rx="9"/></clipPath>`,
			i, colW, boardCardH)
	}
	b.WriteString(`</defs>`)

	draw.RoundRect(&b, 0.5, 0.5, width-1, height-1, 12, th.Canvas, th.Border)
	header(&b, t, th, width)

	if len(columns) == 0 {
		empty(&b, th, width, height)
		b.WriteString("</svg>\n")
		return b.String()
	}

	for i, c := range columns {
		x := pad + float64(i)*(colW+colGutter)
		boardColumn(&b, c, i, x, colW, th)
	}

	b.WriteString("</svg>\n")
	return b.String()
}

// column is one status and the work sitting in it.
type column struct {
	status tree.Status
	cards  []boardCard
}

// boardCard is a leaf task with the context a board needs: where it came from
// and what it is attached to.
type boardCard struct {
	title  string
	parent string
	meta   []metaItem
}

// boardColumns groups the leaves by status, keeping the status order from the
// domain so the columns always read todo, in progress, blocked, done, dropped.
// Empty columns are dropped: a board with nothing blocked should not spend a
// fifth of its width saying so.
func boardColumns(t *tree.Tree) []column {
	byStatus := map[tree.Status][]boardCard{}

	for _, n := range t.Nodes {
		if len(n.Children()) > 0 {
			continue // a milestone is a container, not a card
		}
		card := boardCard{title: n.Title, meta: metaFor(n)}
		if card.title == "" {
			card.title = n.ID
		}
		if n.Parent != "" {
			if p, ok := t.Get(n.Parent); ok {
				card.parent = p.Title
				if card.parent == "" {
					card.parent = p.ID
				}
			}
		}
		byStatus[n.Status] = append(byStatus[n.Status], card)
	}

	var out []column
	for _, s := range tree.Statuses {
		if cards := byStatus[s]; len(cards) > 0 {
			out = append(out, column{status: s, cards: cards})
		}
	}
	return out
}

func boardLabel(t *tree.Tree) string {
	name := t.Project
	if name == "" {
		name = "Development"
	}
	return name + " board"
}

// boardColumn draws a heading and the cards under it.
func boardColumn(b *strings.Builder, c column, index int, x, w float64, th Theme) {
	color := th.color(c.status)
	y := headerH

	glyphClass := ""
	if c.status == tree.InProgress {
		glyphClass = draw.ClassSpin
	}
	draw.IconClass(b, statusIcon[c.status], glyphClass, x, y-2, 15, color)
	draw.Text(b, draw.Clip(c.status.Label(), w-60, 11.5), x+20, y+10, 11.5, th.Text, "600", "")
	draw.Text(b, fmt.Sprintf("%d", len(c.cards)), x+w, y+10, 11.5, th.Muted, "", "end")

	// A hairline under the heading, in the status color, ties the column
	// together without boxing it in.
	draw.RoundRect(b, x, y+18, w, 2, 1, color, "")

	for i, card := range c.cards {
		cardY := headerH + colHead + float64(i)*(boardCardH+boardGap)
		boardCardAt(b, card, index, x, cardY, w, color, th)
	}
}

func boardCardAt(b *strings.Builder, c boardCard, column int, x, y, w float64, color string, th Theme) {
	draw.RoundRect(b, x, y, w, boardCardH, 9, th.Card, th.Border)
	fmt.Fprintf(b, `<g transform="translate(%.1f,%.1f)" clip-path="url(#dt-col-%d)">`, x, y, column)
	draw.RoundRect(b, 0, 0, accentW, boardCardH, 0, color, "")
	b.WriteString(`</g>`)

	available := w - 26
	draw.Text(b, draw.Clip(c.title, available, 12), x+14, y+19, 12, th.Text, "500", "")

	// The second line is the breadcrumb, unless the task carries metadata
	// worth more: a branch and an issue say more than a parent name does.
	if len(c.meta) > 0 {
		meta(b, c.meta, x+14, y+34, available, th)
		return
	}
	if c.parent != "" {
		draw.Text(b, draw.Clip(c.parent, available, 10), x+14, y+34, 10, th.Muted, "", "")
	}
}
