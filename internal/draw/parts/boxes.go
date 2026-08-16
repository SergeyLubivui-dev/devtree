package parts

import (
	"fmt"
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/draw"
)

// ---------------------------------------------------------------------------
// 1. Card
// ---------------------------------------------------------------------------

// Bullet is one line of description under a card's title.
//
// A card says what the work is; a bullet says something about it that a reader
// would otherwise have to open an issue to find out — what is left, what it is
// waiting on, what was decided. It is optional on purpose: a plan diagram where
// every card carries three lines is a document, not a diagram.
type Bullet struct {
	Text   string
	Icon   string // a glyph instead of the dot, when the line has a kind
	Colour string // the mark's colour; the text stays quiet either way
}

// CardStyle is what a card is told about itself.
type CardStyle struct {
	Accent  string // the edge colour; empty draws no edge
	Title   string
	Meta    string // one line under the title, for context
	Bullets []Bullet
	Glyph   string // an icon before the title
	Motion  string // an animation class for the glyph
	Muted   bool   // finished or abandoned work, stated quietly
	Struck  bool   // drawn through: this was dropped
	Trailer func(b *strings.Builder, r draw.Rect)
}

// Card geometry. The lines are fixed rather than a share of the box, because a
// card that grows a bullet should grow by exactly one line — not restyle every
// line it already had.
const (
	cardTitleH  = 17.0
	cardMetaH   = 13.0
	cardBulletH = 15.0
	cardPadY    = 11.0
)

// CardHeight is the room a card needs for what it has been given.
//
// A component that can grow has to be able to say by how much, or every caller
// works it out again and one of them gets it wrong. Give the returned height to
// the box and nothing clips; give it more and the content sits centred in it.
func CardHeight(s CardStyle) float64 {
	h := cardTitleH + cardPadY*2
	if s.Meta != "" {
		h += cardMetaH
	}
	h += float64(len(s.Bullets)) * cardBulletH
	return h
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

	// The block is laid out from a fixed line height and then centred in
	// whatever room it was given, so one card with a bullet and one without
	// still line up on their titles when they are the same height.
	content := CardHeight(s) - cardPadY*2
	top := text.Y + (text.H-content)/2

	colour := p.Text
	if s.Muted {
		colour = p.Muted
	}

	title := draw.Rect{X: text.X, Y: top, W: text.W, H: cardTitleH}
	label := draw.Clip(s.Title, title.W, 12)
	draw.Text(b, label, title.X, baseline(title.MidY(), 12), 12, colour, "500", "")

	if s.Struck {
		// A line through the label rather than a lighter colour: dropped work
		// is not less important, it is not happening.
		y := baseline(title.MidY(), 12) - 4
		fmt.Fprintf(b, `<path d="M%.1f %.1fH%.1f" stroke="%s" stroke-width="1"/>`,
			title.X, y, title.X+draw.TextWidth(label, 12), colour)
	}

	at := title.Bottom()
	if s.Meta != "" {
		meta := draw.Rect{X: text.X, Y: at, W: text.W, H: cardMetaH}
		draw.Text(b, draw.Clip(s.Meta, meta.W, 10), meta.X, baseline(meta.MidY(), 10), 10, p.Faint, "", "")
		at = meta.Bottom()
	}

	for _, bullet := range s.Bullets {
		line := draw.Rect{X: text.X, Y: at, W: text.W, H: cardBulletH}
		cardBullet(b, line, p, bullet)
		at = line.Bottom()
	}
}

// cardBullet draws one described line: a mark, then the words.
//
// The mark is a dot unless the line has a kind worth naming, in which case it
// is a glyph from the same set the statuses use. The text stays quiet whatever
// the mark is coloured — a card with three loud lines has no title left.
func cardBullet(b *strings.Builder, r draw.Rect, p Palette, bullet Bullet) {
	colour := bullet.Colour
	if colour == "" {
		colour = p.Faint
	}

	const size = 10.0
	text := r

	if bullet.Icon != "" {
		var mark draw.Rect
		mark, text = r.SplitLeft(15)
		spot := draw.Rect{X: mark.X, Y: mark.Y, W: 11, H: mark.H}.Fit(11, 11)
		draw.Icon(b, bullet.Icon, spot.X, spot.Y, 11, colour)
	} else {
		fmt.Fprintf(b, `<circle cx="%.1f" cy="%.1f" r="1.8" fill="%s"/>`, r.X+3, r.MidY(), colour)
		_, text = r.SplitLeft(11)
	}

	draw.Text(b, draw.Clip(bullet.Text, text.W, size), text.X, baseline(text.MidY(), size), size, p.Muted, "", "")
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

// StatCounting is Stat with the number arriving rather than having always been
// there.
//
// The steps are drawn as a column of values and the column is stepped past a
// window, so nothing interpolates: every frame shows a number that was actually
// in the list. A counter that eases through 7.4 on its way to 12 is telling the
// reader something untrue about a total, and totals are the one thing a plan
// diagram exists to report.
//
// The last step is what it settles on, so the still frame — printed, or read
// with motion switched off — is the real figure.
func StatCounting(b *strings.Builder, r draw.Rect, p Palette, steps []string, caption, colour string) {
	if len(steps) == 0 {
		return
	}
	if len(steps) == 1 {
		Stat(b, r, p, steps[0], caption, colour)
		return
	}
	if colour == "" {
		colour = p.Text
	}
	draw.RoundRect(b, r.X, r.Y, r.W, r.H, RadiusCard, p.Card, p.Border)

	box := r.InsetXY(14, 9)
	value, caption_ := box, draw.Rect{}
	if caption != "" {
		value, caption_ = box.SplitTop(box.H - 13)
	}

	// Sized for the widest step, or the column would jump about as it rolls.
	widest := steps[0]
	for _, step := range steps {
		if draw.TextWidth(step, 26) > draw.TextWidth(widest, 26) {
			widest = step
		}
	}
	size := draw.FitSize(widest, value.W, minOf(26, value.H), 12)
	line := size * 1.25

	id := fmt.Sprintf("dt-roll-%.0f-%.0f", r.X, r.Y)
	fmt.Fprintf(b, `<defs><clipPath id="%s"><rect x="%.1f" y="%.1f" width="%.1f" height="%.1f"/></clipPath></defs>`,
		id, value.X, value.MidY()-line/2, value.W, line)

	fmt.Fprintf(b, `<g clip-path="url(#%s)"><g class="%s" style="--dt-roll-y:%.1fpx;--dt-roll-n:%d">`,
		id, draw.ClassRoll, -line*float64(len(steps)-1), len(steps)-1)
	for i, step := range steps {
		draw.Text(b, step, value.X, baseline(value.MidY()+float64(i)*line, size), size, colour, "600", "")
	}
	b.WriteString(`</g></g>`)

	if caption != "" {
		draw.Text(b, draw.Clip(caption, caption_.W, 10), caption_.X, caption_.Bottom()-1, 10, p.Faint, "", "")
	}
}
