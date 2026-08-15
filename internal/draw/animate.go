package draw

import (
	"fmt"
	"strings"
)

// Animation classes. They are shared by both renderers so a moving thing means
// the same thing wherever it appears: something is flowing, something is
// growing, something is turning.
const (
	// ClassFlow sends a dash travelling along a stroke — work moving down an
	// edge, a stage feeding the next one.
	ClassFlow = "dt-flow"

	// ClassGrow fills a bar from its left edge, once, on load.
	ClassGrow = "dt-grow"

	// ClassSpin turns a glyph slowly, for work that is genuinely in flight.
	ClassSpin = "dt-spin"
)

// Stylesheet is the whole animation vocabulary, written into every drawing
// that uses it.
//
// CSS rather than SMIL: GitHub serves repository SVGs under
// `default-src 'none'; style-src 'unsafe-inline'; sandbox`, which permits an
// inline stylesheet and forbids script. Declarative animation is the only kind
// that survives that, and it is the only kind worth having in a diagram
// anyway.
//
// Timings are slow on purpose. This sits in a README that people read; motion
// is there to say "this is the live path", not to hold attention.
const Stylesheet = `<style>
@keyframes dt-flow { to { stroke-dashoffset: -16 } }
@keyframes dt-grow { from { transform: scaleX(0) } to { transform: scaleX(1) } }
@keyframes dt-spin { to { transform: rotate(360deg) } }
.dt-flow { animation: dt-flow 1.1s linear infinite }
.dt-grow { animation: dt-grow .9s cubic-bezier(.2,.8,.2,1) both; transform-box: fill-box; transform-origin: left center }
.dt-spin { animation: dt-spin 8s linear infinite; transform-box: fill-box; transform-origin: center }
@media (prefers-reduced-motion: reduce) {
  .dt-flow, .dt-grow, .dt-spin { animation: none }
}
</style>`

// FlowPath overlays a travelling dash on a stroke that has already been drawn.
//
// It is a second path rather than a dash pattern on the first, so the
// connector still reads as a solid line when the animation is off — which is
// what a reader with reduced motion, a static export, or a PDF gets.
//
// The 16-point cycle matches the keyframe, and it is short enough that even a
// stub of an arrow between two boxes always has a dash somewhere on it.
func FlowPath(b *strings.Builder, d, color string) {
	fmt.Fprintf(b, `<path d="%s" fill="none" stroke="%s" stroke-width="2" `+
		`stroke-linecap="round" stroke-dasharray="4 12" class="%s"/>`, d, color, ClassFlow)
}
