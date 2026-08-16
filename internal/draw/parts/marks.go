package parts

import (
	"fmt"
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/draw"
)

// ---------------------------------------------------------------------------
// 7. Chip
// ---------------------------------------------------------------------------

// Chip is a small labelled tag: a branch name, an owner, a tag.
//
// It returns the width it used so a caller can lay a row of them out without
// measuring the same string twice. Chips flow; they are never given a box,
// because the label decides the size and a fixed box would either clip a branch
// name or leave a hole after a short one.
func Chip(b *strings.Builder, x, y float64, p Palette, icon, label, colour string) float64 {
	if colour == "" {
		colour = p.Muted
	}
	const size, height = 10.0, 17.0

	text := draw.Clip(label, 180, size)
	width := draw.TextWidth(text, size) + 14
	if icon != "" {
		width += 14
	}

	draw.RoundRect(b, x, y, width, height, RadiusChip, "", p.Border)

	inner := draw.Rect{X: x + 6, Y: y, W: width - 12, H: height}
	if used := glyph(b, icon, "", draw.Rect{X: inner.X, Y: inner.Y, W: 10, H: inner.H}, 10, colour); used > 0 {
		_, inner = inner.SplitLeft(used - 2)
	}
	draw.Text(b, text, inner.X, baseline(inner.MidY(), size), size, colour, "", "")
	return width
}

// ---------------------------------------------------------------------------
// 8. Badge
// ---------------------------------------------------------------------------

// Badge is a count in a rounded box — 6/6, 12, 3 blocked.
//
// Tabular figures are not available without a font, so the box is sized from
// the text rather than fixed: a badge that fits "9" and clips "10" is worse
// than one that changes width.
func Badge(b *strings.Builder, x, y float64, p Palette, label, colour string) float64 {
	if colour == "" {
		colour = p.Muted
	}
	const size, height = 10.0, 16.0
	width := draw.TextWidth(label, size) + 12

	draw.RoundRect(b, x, y, width, height, RadiusChip, p.Track, "")
	draw.Text(b, label, x+width/2, baseline(y+height/2, size), size, colour, "600", "middle")
	return width
}

// ---------------------------------------------------------------------------
// 9. Pill
// ---------------------------------------------------------------------------

// Pill is a state said in a word: a glyph, a label, and a fully rounded edge.
//
// Rounded to the full height on purpose — the shape says "this is a state, not
// a thing", which is what keeps it from being read as another card.
func Pill(b *strings.Builder, x, y float64, p Palette, icon, label, colour, motion string) float64 {
	if colour == "" {
		colour = p.Accent
	}
	const size, height = 10.5, 20.0

	width := draw.TextWidth(label, size) + 18
	if icon != "" {
		width += 15
	}

	draw.RoundRect(b, x, y, width, height, RadiusPill, "", colour)

	inner := draw.Rect{X: x + 8, Y: y, W: width - 16, H: height}
	if used := glyph(b, icon, motion, draw.Rect{X: inner.X, Y: inner.Y, W: 11, H: inner.H}, 11, colour); used > 0 {
		_, inner = inner.SplitLeft(used - 2)
	}
	draw.Text(b, label, inner.X, baseline(inner.MidY(), size), size, colour, "600", "")
	return width
}

// ---------------------------------------------------------------------------
// 10. Avatar
// ---------------------------------------------------------------------------

// Avatar is an owner as one or two letters in a circle.
//
// No pictures: a diagram that fetches faces stops rendering the day the network
// is not there, and an SVG in a README has no network at all. Initials always
// draw, and they are the part a colleague recognises anyway.
func Avatar(b *strings.Builder, x, y, size float64, p Palette, name, colour string) {
	if colour == "" {
		colour = p.Accent
	}
	initials := draw.Initials(name)
	fmt.Fprintf(b, `<circle cx="%.1f" cy="%.1f" r="%.1f" fill="none" stroke="%s" stroke-width="1"/>`,
		x+size/2, y+size/2, size/2, colour)
	if initials == "" {
		return
	}
	// Two wide letters — "ЮП", "MW" — overflow a circle sized for "AB", so the
	// letters give way rather than the shape.
	font := draw.FitSize(initials, size*0.66, size*0.42, size*0.28)
	draw.Text(b, initials, x+size/2, baseline(y+size/2, font), font, colour, "600", "middle")
}

// ---------------------------------------------------------------------------
// 11. Milestone
// ---------------------------------------------------------------------------

// Milestone is a diamond with a label: a point in a plan rather than a span of
// work. The shape is the whole message, so it is drawn filled and small.
func Milestone(b *strings.Builder, x, y, size float64, p Palette, label, colour string) {
	if colour == "" {
		colour = p.Accent
	}
	h := size / 2
	fmt.Fprintf(b, `<path d="M%.1f %.1fL%.1f %.1fL%.1f %.1fL%.1f %.1fZ" fill="%s"/>`,
		x, y-h, x+h, y, x, y+h, x-h, y, colour)

	if label != "" {
		draw.Text(b, label, x, y+h+11, 10, p.Muted, "600", "middle")
	}
}

// ---------------------------------------------------------------------------
// 12. Legend
// ---------------------------------------------------------------------------

// LegendItem is one entry: a mark and what it means.
type LegendItem struct {
	Icon   string
	Label  string
	Colour string
	Motion string
}

// Legend is the key to a drawing, laid out in a row, and it returns the width
// it used.
//
// A legend is drawn from the same glyphs the diagram uses rather than from a
// second set of shapes kept in step by hand — the day those two disagree, the
// legend is worse than none.
func Legend(b *strings.Builder, x, y float64, p Palette, items []LegendItem, gap float64) float64 {
	at := x
	for _, item := range items {
		colour := item.Colour
		if colour == "" {
			colour = p.Muted
		}
		box := draw.Rect{X: at, Y: y, W: 13, H: 13}
		used := glyph(b, item.Icon, item.Motion, box, 13, colour)
		at += used
		draw.Text(b, item.Label, at, baseline(y+6.5, 10), 10, p.Muted, "", "")
		at += draw.TextWidth(item.Label, 10) + gap
	}
	if at > x {
		at -= gap
	}
	return at - x
}
