package svg

import (
	"fmt"
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/draw"
	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

// statusIcon maps a status to a glyph.
//
// The set is a family on purpose: five circles, outlined while the work is
// still moving and filled once it has settled. A reader learns the convention
// from the legend once and then reads the diagram at a glance.
var statusIcon = map[tree.Status]string{
	tree.Todo:       "clock-circle",
	tree.InProgress: "circle-half-dotted-check",
	tree.Blocked:    "lock-circle",
	tree.Done:       "check-circle",
	tree.Dropped:    "close-circle",
}

// Render draws the whole plan. The result is a complete SVG document: no
// external references, no script, nothing to load at display time.
func Render(t *tree.Tree, th Theme) string {
	places, rows, columns := layout(t)

	width := pad*2 + colGap*float64(max(len(columns)-1, 0))
	for _, w := range columns {
		width += w
	}
	if floor := pad*2 + legendWidth(); width < floor {
		width = floor
	}
	if floor := pad*2 + headerWidth(t); width < floor {
		width = floor
	}

	height := headerH + legendH + pad
	if rows > 0 {
		height += rows*(cardH+rowGap) - rowGap
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%.0f" `+
		`viewBox="0 0 %.0f %.0f" font-family="%s" role="img" aria-label="%s">`,
		width, height, width, height, draw.FontStack, draw.Escape(ariaLabel(t)))
	b.WriteString(draw.Stylesheet)
	defs(&b, columns)

	// The panel is what keeps the diagram legible outside GitHub too: a bare
	// transparent background would leave the dark rendering as pale text on a
	// white page in any other viewer.
	fmt.Fprintf(&b, `<rect x="0.5" y="0.5" width="%.0f" height="%.0f" rx="12" fill="%s" stroke="%s"/>`,
		width-1, height-1, th.Canvas, th.Border)

	header(&b, t, th, width)

	if len(places) == 0 {
		empty(&b, th, width, height)
	} else {
		// Edges first so the cards sit on top of the line ends.
		edges(&b, places, columns, th)
		for i, p := range places {
			card(&b, p, i, columns[p.depth], columnX(columns, p.depth), rowY(p.row), clipID(p.depth), th)
		}
	}

	legend(&b, th, height)
	b.WriteString("</svg>\n")
	return b.String()
}

// defs declares one clip path per column, so an accent stripe can run the full
// height of a card and still follow its rounded corners. Clipping is the only
// way to get that: a rectangle would poke out of the corners, and a rounded
// rectangle narrower than its own radius collapses into a lens.
func defs(b *strings.Builder, columns []float64) {
	if len(columns) == 0 {
		return
	}
	b.WriteString(`<defs>`)
	for i, w := range columns {
		fmt.Fprintf(b, `<clipPath id="%s"><rect x="0" y="0" width="%.1f" height="%.1f" rx="9"/></clipPath>`,
			clipID(i), w, cardH)
	}
	b.WriteString(`</defs>`)
}

func clipID(depth int) string { return fmt.Sprintf("dt-card-%d", depth) }

func columnX(columns []float64, depth int) float64 {
	x := pad
	for i := 0; i < depth && i < len(columns); i++ {
		x += columns[i] + colGap
	}
	return x
}

func rowY(row float64) float64 { return headerH + row*(cardH+rowGap) }

func ariaLabel(t *tree.Tree) string {
	name := t.Project
	if name == "" {
		name = "Development"
	}
	done, total := t.Totals()
	return fmt.Sprintf("%s development tree, %d of %d tasks done", name, done, total)
}

func headerWidth(t *tree.Tree) float64 {
	return 32 + draw.TextWidth(projectName(t), 17)
}

func projectName(t *tree.Tree) string {
	if t.Project == "" {
		return "Development tree"
	}
	return t.Project
}

// header draws the wordmark, the project name, and the overall progress.
func header(b *strings.Builder, t *tree.Tree, th Theme, width float64) {
	draw.Icon(b, "tree", pad, pad-2, 22, th.Done)
	fmt.Fprintf(b, `<text x="%.1f" y="%.1f" font-size="17" font-weight="600" fill="%s">%s</text>`,
		pad+30, pad+15, th.Text, draw.Escape(draw.Clip(projectName(t), width-pad*2-40, 17)))

	done, total := t.Totals()
	const barW = 208.0
	y := pad + 34

	draw.RoundRect(b, pad, y, barW, 6, 3, th.Track, "")
	if total > 0 && done > 0 {
		// The fill grows in once on load. It is the one place a reader's eye
		// should be pulled first, and it settles immediately after.
		draw.RoundRectClass(b, draw.ClassGrow, pad, y, barW*float64(done)/float64(total), 6, 3, th.Done, "")
	}
	fmt.Fprintf(b, `<text x="%.1f" y="%.1f" font-size="11" fill="%s">%d / %d tasks done</text>`,
		pad+barW+10, y+6, th.Muted, done, total)
}

// card draws one node. The index only feeds the entrance stagger, so the
// diagram assembles itself instead of arriving all at once.
func card(b *strings.Builder, p placement, index int, w, x, y float64, clip string, th Theme) {
	status := p.node.Status
	color := th.color(status)

	draw.OpenRise(b, index)
	defer draw.CloseGroup(b)

	draw.RoundRect(b, x, y, w, cardH, 9, th.Card, th.Border)

	// A full-height accent rather than a colored fill: the status is legible
	// from across the diagram without five loud rectangles competing with the
	// text. Clipping to the card shape is what lets it run corner to corner.
	fmt.Fprintf(b, `<g transform="translate(%.1f,%.1f)" clip-path="url(#%s)">`, x, y, clip)
	draw.RoundRect(b, 0, 0, accentW, cardH, 0, color, "")
	b.WriteString(`</g>`)

	draw.IconClass(b, statusIcon[status], glyphMotion(status), x+13, y+(cardH-iconSize)/2, iconSize, color)

	right := rightPad
	if p.ratio != "" {
		right = rightWide
	}
	available := w - textLeft - right

	titleY := y + 29
	if len(p.meta) > 0 {
		titleY = y + 21
	}
	strike := ""
	if status == tree.Dropped {
		strike = ` text-decoration="line-through"`
	}
	fmt.Fprintf(b, `<text x="%.1f" y="%.1f" font-size="%.1f" font-weight="500" fill="%s"%s>%s</text>`,
		x+textLeft, titleY, titleSize, th.Text, strike, draw.Escape(draw.Clip(p.title, available, titleSize)))

	if len(p.meta) > 0 {
		meta(b, p.meta, x+textLeft, y+35, available, th)
	}

	if p.ratio != "" {
		fmt.Fprintf(b, `<text x="%.1f" y="%.1f" font-size="10" fill="%s" text-anchor="end">%s</text>`,
			x+w-12, y+21, th.Muted, p.ratio)
		const barW = 34.0
		bx, by := x+w-12-barW, y+27.0
		draw.RoundRect(b, bx, by, barW, 4, 2, th.Track, "")
		if p.done > 0 {
			draw.RoundRectClass(b, draw.ClassGrow, bx, by, barW*float64(p.done)/float64(p.total), 4, 2, th.Done, "")
		}
	}
}

// glyphMotion decides whether a status glyph moves, and how.
//
// Two states earn motion and the other three do not: work in flight turns,
// because it is turning; blocked work breathes, because it is the one state a
// reader should not scroll past. Settled work holds still.
func glyphMotion(s tree.Status) string {
	switch s {
	case tree.InProgress:
		return draw.ClassSpin
	case tree.Blocked:
		return draw.ClassPulse
	default:
		return ""
	}
}

// meta draws the context line, dropping whole items that will not fit rather
// than letting a truncated branch name run into the card border.
func meta(b *strings.Builder, items []metaItem, x, baseline, available float64, th Theme) {
	cursor := x
	for i, m := range items {
		w := draw.TextWidth(m.text, metaSize)
		if m.icon != "" {
			w += metaIcon + 3
		}
		gap := 0.0
		if i > 0 {
			gap = 10
		}
		if cursor+gap+w > x+available {
			return
		}
		cursor += gap
		if m.icon != "" {
			draw.Icon(b, m.icon, cursor, baseline-metaIcon+1.5, metaIcon, th.Muted)
			cursor += metaIcon + 3
		}
		fmt.Fprintf(b, `<text x="%.1f" y="%.1f" font-size="%.1f" fill="%s">%s</text>`,
			cursor, baseline, metaSize, th.Muted, draw.Escape(m.text))
		cursor += draw.TextWidth(m.text, metaSize)
	}
}

// edges connects parents to children with rounded elbows. Curves would cross
// each other in a wide tree; right angles stay readable however many children
// a milestone grows.
func edges(b *strings.Builder, places []placement, columns []float64, th Theme) {
	index := make(map[string]placement, len(places))
	for _, p := range places {
		index[p.node.ID] = p
	}

	for _, p := range places {
		px := columnX(columns, p.depth) + columns[p.depth]
		py := rowY(p.row) + cardH/2

		for _, child := range p.node.Children() {
			c, ok := index[child.ID]
			if !ok {
				continue
			}
			cx := columnX(columns, c.depth)
			cy := rowY(c.row) + cardH/2

			d := elbowPath(px, py, cx, cy)
			fmt.Fprintf(b, `<path d="%s" fill="none" stroke="%s" stroke-width="1.5"/>`, d, th.Edge)

			// A dash travels down the edges that lead into work in flight, and
			// only those. Motion here is information, not decoration: it marks
			// the live path through the plan at a glance.
			if child.Status == tree.InProgress {
				draw.FlowPath(b, d, th.InProgress)
			}
		}
	}
}

// elbowPath is the connector geometry: out of the parent, across, down or up,
// into the child, with the corners rounded. Returning the path data rather
// than drawing it lets the flow animation reuse the exact same line.
func elbowPath(px, py, cx, cy float64) string {
	if diff := cy - py; diff > -1 && diff < 1 {
		return fmt.Sprintf("M%.1f %.1f H%.1f", px, py, cx)
	}

	elbow := cx - colGap/2
	r := 8.0
	if half := absF(cy-py) / 2; r > half {
		r = half
	}
	dir := 1.0
	if cy < py {
		dir = -1
	}
	return fmt.Sprintf("M%.1f %.1f H%.1f Q%.1f %.1f %.1f %.1f V%.1f Q%.1f %.1f %.1f %.1f H%.1f",
		px, py,
		elbow-r,
		elbow, py, elbow, py+r*dir,
		cy-r*dir,
		elbow, cy, elbow+r, cy,
		cx)
}

// legend spells out the five statuses, generated from the status list so a new
// status can never be added without appearing here.
func legend(b *strings.Builder, th Theme, height float64) {
	y := height - pad - 8
	cursor := pad

	for _, s := range tree.Statuses {
		draw.Icon(b, statusIcon[s], cursor, y-11, 13, th.color(s))
		cursor += 13 + 4
		fmt.Fprintf(b, `<text x="%.1f" y="%.1f" font-size="10.5" fill="%s">%s</text>`,
			cursor, y, th.Muted, draw.Escape(s.Label()))
		cursor += draw.TextWidth(s.Label(), 10.5) + 16
	}
}

func legendWidth() float64 {
	var w float64
	for _, s := range tree.Statuses {
		w += 13 + 4 + draw.TextWidth(s.Label(), 10.5) + 16
	}
	return w - 16
}

// empty says what to do instead of drawing a blank panel.
func empty(b *strings.Builder, th Theme, width, height float64) {
	const message = "Nothing planned yet — add a task with: devtree add \"Task title\""
	x := width/2 - draw.TextWidth(message, 13)/2
	fmt.Fprintf(b, `<text x="%.1f" y="%.1f" font-size="13" fill="%s">%s</text>`,
		x, (height+headerH-legendH)/2, th.Muted, draw.Escape(message))
}

func absF(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
