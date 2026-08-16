package parts

import (
	"fmt"
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/draw"
)

// ---------------------------------------------------------------------------
// 17. Divider
// ---------------------------------------------------------------------------

// Divider is a hairline, optionally with a caption sitting in a gap in it.
//
// The caption interrupts the rule rather than sitting above it: a label with a
// line running behind it is two objects, and a label in a gap is one.
func Divider(b *strings.Builder, r draw.Rect, p Palette, caption string) {
	y := r.MidY()
	if caption == "" {
		fmt.Fprintf(b, `<path d="M%.1f %.1fH%.1f" stroke="%s" stroke-width="1"/>`, r.X, y, r.Right(), p.Border)
		return
	}

	const size = 10.0
	text := draw.Clip(caption, r.W-40, size)
	width := draw.TextWidth(text, size)
	gap := width + 16
	left := r.X + (r.W-gap)/2

	fmt.Fprintf(b, `<path d="M%.1f %.1fH%.1fM%.1f %.1fH%.1f" stroke="%s" stroke-width="1"/>`,
		r.X, y, left, left+gap, y, r.Right(), p.Border)
	draw.Text(b, text, r.MidX(), baseline(y, size), size, p.Faint, "", "middle")
}

// ---------------------------------------------------------------------------
// 18. Connector
// ---------------------------------------------------------------------------

// Connector joins two points with a rounded elbow.
//
// It leaves horizontally and arrives horizontally, turning once in the middle,
// because a diagram whose lines all enter their cards from the same side is
// read as a structure; one with lines at every angle is read as a plate of
// spaghetti. A motion class overlays a travelling dash on the line as a second
// path, so the connector still reads as solid when the motion is off.
func Connector(b *strings.Builder, from, to draw.Point, p Palette, colour, motion string) {
	if colour == "" {
		colour = p.Border
	}
	stroke(b, elbow(from, to), colour, motion)
}

// stroke writes a connector's path and gives it whatever motion it was asked
// for. The two kinds are different in kind, not degree: drawing on happens once
// and is about arrival, while a travelling dash repeats and is about a path
// being live. So one goes on the path itself and the other rides above it.
func stroke(b *strings.Builder, d, colour, motion string) {
	if motion == draw.ClassDraw {
		fmt.Fprintf(b, `<path d="%s" fill="none" stroke="%s" stroke-width="1.5" pathLength="1" class="%s"/>`,
			d, colour, motion)
		return
	}
	fmt.Fprintf(b, `<path d="%s" fill="none" stroke="%s" stroke-width="1.5"/>`, d, colour)
	if motion != "" {
		draw.FlowPath(b, d, colour)
	}
}

// elbow is the path itself, shared with Arrow.
func elbow(from, to draw.Point) string {
	mid := (from.X + to.X) / 2
	radius := 8.0

	if abs(to.Y-from.Y) < 1 {
		return fmt.Sprintf("M%.1f %.1fH%.1f", from.X, from.Y, to.X)
	}
	if r := abs(to.Y-from.Y) / 2; r < radius {
		radius = r
	}

	down := 1.0
	if to.Y < from.Y {
		down = -1
	}
	return fmt.Sprintf("M%.1f %.1fH%.1fQ%.1f %.1f %.1f %.1fV%.1fQ%.1f %.1f %.1f %.1fH%.1f",
		from.X, from.Y,
		mid-radius,
		mid, from.Y, mid, from.Y+radius*down,
		to.Y-radius*down,
		mid, to.Y, mid+radius, to.Y,
		to.X)
}

// ---------------------------------------------------------------------------
// 19. Arrow
// ---------------------------------------------------------------------------

// Arrow is a connector that says which way round the relationship goes.
//
// The head is a path rather than a marker element: markers inherit context in
// ways that differ between renderers, and an SVG in a README has to draw the
// same in GitHub's sanitizer, in a browser, and in whatever a reader printed it
// with.
func Arrow(b *strings.Builder, from, to draw.Point, p Palette, colour, motion string) {
	if colour == "" {
		colour = p.Border
	}
	const head = 5.0

	stroke(b, elbow(from, draw.Point{X: to.X - head, Y: to.Y}), colour, motion)
	fmt.Fprintf(b, `<path d="M%.1f %.1fL%.1f %.1fL%.1f %.1fZ" fill="%s"/>`,
		to.X, to.Y, to.X-head-1, to.Y-head*0.7, to.X-head-1, to.Y+head*0.7, colour)
}

// ---------------------------------------------------------------------------
// 20. Brace
// ---------------------------------------------------------------------------

// Brace is a curly bracket spanning a range, with its label at the point.
//
// It says "these belong together" without drawing a box, which matters when the
// things it is grouping are already in boxes.
func Brace(b *strings.Builder, x, top, bottom float64, p Palette, label string, reach float64) {
	if reach <= 0 {
		reach = 10
	}
	mid := (top + bottom) / 2

	// Two mirrored cubics meeting at the tip. The control points sit at the
	// far side of the reach on the outside and back at the spine on the
	// inside, which is what gives a brace its pinch rather than the lazy
	// bracket a single curve draws.
	fmt.Fprintf(b, `<path d="M%.1f %.1fC%.1f %.1f %.1f %.1f %.1f %.1fC%.1f %.1f %.1f %.1f %.1f %.1f" `+
		`fill="none" stroke="%s" stroke-width="1.2" stroke-linecap="round"/>`,
		x, top,
		x+reach, top, x, mid, x+reach, mid,
		x, mid, x+reach, bottom, x, bottom,
		p.Border)

	if label != "" {
		draw.Text(b, label, x+reach+8, baseline(mid, 10), 10, p.Muted, "600", "")
	}
}

// BraceLabelOffset is how far past the spine a brace's label starts, so a
// caller laying out what comes after it does not have to guess.
func BraceLabelOffset(reach float64, label string) float64 {
	if reach <= 0 {
		reach = 10
	}
	if label == "" {
		return reach + 8
	}
	return reach + 8 + draw.TextWidth(label, 10) + 12
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// Names is every component in this package, in the order the gallery draws
// them. It exists so the documentation cannot fall out of step with the code:
// a component added without a line here is a component the gallery never shows.
func Names() []string {
	return []string{
		"Card", "Cluster", "Callout", "Swimlane", "ColumnHead", "Stat",
		"Chip", "Badge", "Pill", "Avatar", "Milestone", "Legend",
		"Bar", "Ring", "Sparkline", "Timeline",
		"Divider", "Connector", "Arrow", "Brace",
	}
}
