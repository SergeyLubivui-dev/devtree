package parts

import (
	"fmt"
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/draw"
)

// ---------------------------------------------------------------------------
// 1. Card
// ---------------------------------------------------------------------------

// CardStyle is what a card is told about itself.
type CardStyle struct {
	Accent  string // the edge colour; empty draws no edge
	Title   string
	Meta    string // one line under the title, for context
	Glyph   string // an icon before the title
	Motion  string // an animation class for the glyph
	Muted   bool   // finished or abandoned work, stated quietly
	Struck  bool   // drawn through: this was dropped
	Trailer func(b *strings.Builder, r draw.Rect)
}

// Card is a raised panel with a full-height accent edge.
//
// The edge rather than a tinted fill, because five tinted fills in one diagram
// compete with the text they are supposed to be qualifying. An edge is legible
// from across the picture and takes no contrast away from the label.
func Card(b *strings.Builder, r draw.Rect, p Palette, s CardStyle) {
	draw.RoundRect(b, r.X, r.Y, r.W, r.H, RadiusCard, p.Card, p.Border)

	if s.Accent != "" {
		// Clipped to the card so the edge follows the rounded corners instead
		// of standing proud of them.
		id := fmt.Sprintf("dt-clip-%.0f-%.0f", r.X, r.Y)
		fmt.Fprintf(b, `<defs><clipPath id="%s"><rect x="0" y="0" width="%.1f" height="%.1f" rx="%.1f"/></clipPath></defs>`,
			id, r.W, r.H, RadiusCard)
		fmt.Fprintf(b, `<g transform="translate(%.1f,%.1f)" clip-path="url(#%s)">`, r.X, r.Y, id)
		draw.RoundRect(b, 0, 0, 3, r.H, 0, s.Accent, "")
		b.WriteString(`</g>`)
	}

	body := r.Pad(0, 12, 0, 13)
	text := body
	if s.Trailer != nil {
		var trailer draw.Rect
		trailer, text = body.SplitRight(54)
		s.Trailer(b, trailer)
	}

	if s.Glyph != "" {
		var mark draw.Rect
		mark, text = text.SplitLeft(15 + 8)
		spot := draw.Rect{X: mark.X, Y: mark.Y, W: 15, H: mark.H}.Fit(15, 15)
		colour := s.Accent
		if colour == "" {
			colour = p.Muted
		}
		if s.Motion == "" {
			draw.Icon(b, s.Glyph, spot.X, spot.Y, 15, colour)
		} else {
			draw.IconClass(b, s.Glyph, s.Motion, spot.X, spot.Y, 15, colour)
		}
	}

	// With a meta line the title takes the upper half and the meta the lower;
	// alone, the title has the whole card and sits centred in it.
	title, meta := text, draw.Rect{}
	if s.Meta != "" {
		title, meta = text.SplitTop(text.H * 0.54)
	}

	colour := p.Text
	if s.Muted {
		colour = p.Muted
	}
	label := draw.Clip(s.Title, title.W, 12)
	draw.Text(b, label, title.X, baseline(title.MidY(), 12), 12, colour, "500", "")

	if s.Struck {
		// A line through the label rather than a lighter colour: dropped work
		// is not less important, it is not happening.
		width := draw.TextWidth(label, 12)
		y := baseline(title.MidY(), 12) - 4
		fmt.Fprintf(b, `<path d="M%.1f %.1fH%.1f" stroke="%s" stroke-width="1"/>`,
			title.X, y, title.X+width, colour)
	}

	if s.Meta != "" {
		draw.Text(b, draw.Clip(s.Meta, meta.W, 10), meta.X, baseline(meta.MidY(), 10), 10, p.Faint, "", "")
	}
}

// ---------------------------------------------------------------------------
// 2. Cluster
// ---------------------------------------------------------------------------

// Cluster is a dashed box around a region, with its name on the border.
//
// Dashed rather than solid: a cluster is a reading aid, not another object in
// the diagram, and a solid box the same weight as the cards inside it reads as
// one more card.
func Cluster(b *strings.Builder, r draw.Rect, p Palette, label string) {
	fmt.Fprintf(b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="%.1f" fill="none" stroke="%s" stroke-width="1" stroke-dasharray="5 4"/>`,
		r.X, r.Y, r.W, r.H, RadiusCard+3, p.Border)

	if label == "" {
		return
	}
	// The label sits on the top edge, with a patch of canvas behind it so the
	// dashes do not run through the letters.
	size := 10.0
	text := draw.Clip(label, r.W-24, size)
	width := draw.TextWidth(text, size) + 10
	draw.RoundRect(b, r.X+14, r.Y-size/2-2, width, size+4, 3, p.Canvas, "")
	draw.Text(b, text, r.X+19, r.Y+size/2-2, size, p.Muted, "600", "")
}

// ---------------------------------------------------------------------------
// 3. Callout
// ---------------------------------------------------------------------------

// Callout is a bordered note with an icon and a tinted edge: the one place a
// diagram is allowed to address the reader in a sentence.
func Callout(b *strings.Builder, r draw.Rect, p Palette, icon, text, accent string) {
	if accent == "" {
		accent = p.Accent
	}
	draw.RoundRect(b, r.X, r.Y, r.W, r.H, RadiusCard, p.Card, p.Border)
	draw.RoundRect(b, r.X, r.Y+6, 2, r.H-12, 1, accent, "")

	body := r.Pad(10, 12, 10, 14)
	if used := glyph(b, icon, "", draw.Rect{X: body.X, Y: body.Y, W: 14, H: 14}, 14, accent); used > 0 {
		_, body = body.SplitLeft(used)
	}

	lines := draw.Lines(text, body.W, 11, int(body.H/15))
	for i, line := range lines {
		draw.Text(b, line, body.X, body.Y+11+float64(i)*15, 11, p.Muted, "", "")
	}
}

// ---------------------------------------------------------------------------
// 4. Swimlane
// ---------------------------------------------------------------------------

// Swimlane is a labelled horizontal band, for diagrams organised by who or by
// stage. The label rides in a gutter on the left so the lane's contents all
// start at the same x whatever the label says.
func Swimlane(b *strings.Builder, r draw.Rect, p Palette, label string, gutter float64) draw.Rect {
	draw.RoundRect(b, r.X, r.Y, r.W, r.H, RadiusCard, p.Card, "")

	head, body := r.SplitLeft(gutter)
	fmt.Fprintf(b, `<path d="M%.1f %.1fV%.1f" stroke="%s" stroke-width="1"/>`,
		head.Right(), r.Y+8, r.Bottom()-8, p.Border)

	draw.Text(b, draw.Clip(label, head.W-20, 11), head.X+12, baseline(head.MidY(), 11), 11, p.Muted, "600", "")
	return body.InsetXY(12, 8)
}

// ---------------------------------------------------------------------------
// 5. ColumnHead
// ---------------------------------------------------------------------------

// ColumnHead is the heading of a board column: a glyph, a name, a count, and a
// hairline in the column's colour tying it together without boxing it in.
func ColumnHead(b *strings.Builder, r draw.Rect, p Palette, icon, label string, count int, colour, motion string) {
	if colour == "" {
		colour = p.Accent
	}
	line, _ := r.SplitTop(r.H - 4)

	used := glyph(b, icon, motion, draw.Rect{X: line.X, Y: line.Y, W: 15, H: line.H}, 15, colour)
	name := draw.Clip(label, line.W-used-34, 11.5)
	draw.Text(b, name, line.X+used, baseline(line.MidY(), 11.5), 11.5, p.Text, "600", "")
	draw.Text(b, fmt.Sprintf("%d", count), line.Right(), baseline(line.MidY(), 11.5), 11.5, p.Muted, "", "end")

	draw.RoundRect(b, r.X, r.Bottom()-2, r.W, 2, 1, colour, "")
}

// ---------------------------------------------------------------------------
// 6. Stat
// ---------------------------------------------------------------------------

// Stat is one number said loudly, with a caption under it.
//
// The number is fitted rather than clipped: a stat tile exists to be read at a
// glance from across a room, and "1…" is not a number.
func Stat(b *strings.Builder, r draw.Rect, p Palette, value, caption, colour string) {
	if colour == "" {
		colour = p.Text
	}
	draw.RoundRect(b, r.X, r.Y, r.W, r.H, RadiusCard, p.Card, p.Border)

	// The caption gets a row of its own before the number is sized. Fitting the
	// number to the whole box and then writing the caption over it is how a
	// stat tile ends up with "23 / 25" and "tasks done" sharing a line.
	box := r.InsetXY(14, 9)
	value_, caption_ := box, draw.Rect{}
	if caption != "" {
		value_, caption_ = box.SplitTop(box.H - 13)
	}

	size := draw.FitSize(value, value_.W, minOf(26, value_.H), 12)
	draw.Text(b, value, value_.X, baseline(value_.MidY(), size), size, colour, "600", "")

	if caption != "" {
		draw.Text(b, draw.Clip(caption, caption_.W, 10), caption_.X, caption_.Bottom()-1, 10, p.Faint, "", "")
	}
}

func minOf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
