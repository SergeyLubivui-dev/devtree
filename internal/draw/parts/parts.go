// Package parts is the drawing vocabulary: twenty components that sit between
// the primitives in draw and a finished diagram.
//
// The renderers used to assemble every card, badge and column head out of
// rectangles and text runs, which meant the same fourteen-pixel inset appeared
// in four files and drifted in three of them. A component says the intent once
// — "a card with an accent edge", "a meter that grows on load" — and the
// geometry lives in one place where it can be corrected once.
//
// Two rules keep this package honest:
//
// It knows nothing about development plans. There is no status here, no task
// and no tree: a Pill takes a colour and a label, and it is the renderer above
// that decides "blocked" is red. That is what lets the illustrations in the
// documentation use the same components as the diagrams without either one
// bending to the other.
//
// It writes SVG and nothing else. No component measures the browser, fetches
// anything, or depends on a font being installed — the same bytes have to draw
// identically in a README on GitHub, in a terminal-launched preview, and in a
// PDF printed from a page nobody has network for.
package parts

import (
	"fmt"
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/draw"
)

// Palette is the colour vocabulary a component draws with.
//
// Deliberately not the renderer's theme: that one carries the five task
// statuses, and a component library that knew what "blocked" looks like could
// not be used to draw anything else. Callers pass a colour where the meaning
// belongs to them.
type Palette struct {
	Text   string // primary labels
	Muted  string // secondary labels
	Faint  string // captions, axis ticks, anything at the edge of legible
	Canvas string // the page behind everything
	Card   string // a raised surface
	Track  string // the unfilled part of a meter
	Border string // hairlines
	Accent string // the one colour that means "this one"
}

// Radius values, so a diagram made of these components has one corner style
// rather than five.
const (
	RadiusCard  = 9.0
	RadiusChip  = 5.0
	RadiusPill  = 99.0
	strokeThin  = 1.0
	glyphInset  = 1.5
	labelBottom = 0.35 // baseline nudge: text sits on the line, not in the middle
)

// baseline returns the y a label should be drawn at to look vertically centred
// in a band. SVG text hangs from its baseline, so centring means measuring down
// from the middle by roughly a third of the size — the fudge every drawing
// program keeps, written down once here instead of sprinkled as +4 and +5.
func baseline(mid, size float64) float64 { return mid + size*labelBottom }

// glyph draws an icon if the caller named one, and reports the width it used
// including the gap after it. Naming nothing costs nothing, which is what lets
// every component take an optional icon without a second variant.
func glyph(b *strings.Builder, name, class string, box draw.Rect, size float64, color string) float64 {
	if name == "" {
		return 0
	}
	spot := box.Fit(size, size)
	if class == "" {
		draw.Icon(b, name, spot.X, spot.Y, size, color)
	} else {
		draw.IconClass(b, name, class, spot.X, spot.Y, size, color)
	}
	return size + 6
}

// group opens a group with an optional class, and returns the closer. Components
// that animate use it so the animation is on a wrapper rather than on the shape
// carrying a transform — a CSS transform replaces the attribute outright.
func group(b *strings.Builder, class string) func() {
	if class == "" {
		return func() {}
	}
	fmt.Fprintf(b, `<g class="%s">`, class)
	return func() { b.WriteString(`</g>`) }
}
