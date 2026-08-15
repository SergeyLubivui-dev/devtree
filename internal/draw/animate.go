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

	// ClassRise fades a card up into place, once, on load. Staggered across a
	// diagram it reads as the picture assembling itself rather than arriving
	// all at once — and after half a second it is a still image again.
	ClassRise = "dt-rise"

	// ClassPulse breathes, slowly. It is spent on exactly one thing: work that
	// is blocked, which is the one state a reader should not scroll past.
	ClassPulse = "dt-pulse"

	// ClassMarquee scrolls a strip of cards forever. The strip holds two
	// identical copies laid end to end and travels exactly half its own width,
	// so the loop has no seam to notice.
	ClassMarquee = "dt-marquee"
)

// riseStep is how much later each card arrives than the one before it, and
// riseCap is where the stagger stops growing: a plan with sixty nodes should
// not take three seconds to finish appearing.
const (
	riseStep = 0.035
	riseCap  = 0.7
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
//
// Two safety rules are worth the extra line each. The entrance uses fill mode
// `backwards` rather than `both`, so a card ends on its own base style instead
// of being parked in an animation's final frame. And printing switches every
// animation off, because a renderer that freezes the first frame of a fade-in
// would otherwise put a blank card on the page.
const Stylesheet = `<style>
@keyframes dt-flow { to { stroke-dashoffset: -16 } }
@keyframes dt-grow { from { transform: scaleX(0) } to { transform: scaleX(1) } }
@keyframes dt-spin { to { transform: rotate(360deg) } }
@keyframes dt-rise { from { opacity: 0; transform: translateY(7px) } to { opacity: 1; transform: none } }
@keyframes dt-pulse { 0%, 100% { opacity: 1 } 50% { opacity: .45 } }
@keyframes dt-marquee { from { transform: translateX(-50%) } to { transform: none } }
.dt-flow { animation: dt-flow 1.1s linear infinite }
.dt-grow { animation: dt-grow .9s cubic-bezier(.2,.8,.2,1) both; transform-box: fill-box; transform-origin: left center }
.dt-spin { animation: dt-spin 8s linear infinite; transform-box: fill-box; transform-origin: center }
.dt-rise { animation: dt-rise .55s cubic-bezier(.2,.8,.2,1) backwards }
.dt-pulse { animation: dt-pulse 2.4s ease-in-out infinite }
.dt-marquee { animation: dt-marquee 26s linear infinite; transform-box: fill-box }
@media (prefers-reduced-motion: reduce), print {
  .dt-flow, .dt-grow, .dt-spin, .dt-rise, .dt-pulse, .dt-marquee { animation: none }
}
</style>`

// FlowPath overlays a travelling dash on a stroke that has already been drawn.
//
// OpenRise starts a group that fades up on load, staggered by position. The
// caller closes it with CloseGroup.
//
// The delay rides on a style attribute rather than a class per index: the
// alternative is a stylesheet that grows a rule for every card in the plan.
func OpenRise(b *strings.Builder, index int) { OpenNode(b, index, "") }

// OpenNode is OpenRise with the task's identity written onto the group.
//
// A drawing that knows which task each card is can be pointed at: the editor
// hangs its controls off these, and anything scripting a written .svg can find
// a card without matching on the label a human typed. It costs a handful of
// bytes per card, and it is the same attribute whether the drawing is being
// previewed or written to disk — the editor draws no special version.
func OpenNode(b *strings.Builder, index int, id string) {
	delay := float64(index) * riseStep
	if delay > riseCap {
		delay = riseCap
	}
	fmt.Fprintf(b, `<g class="%s" style="animation-delay:%.2fs"`, ClassRise, delay)
	if id != "" {
		fmt.Fprintf(b, ` data-node="%s"`, Escape(id))
	}
	b.WriteString(`>`)
}

// CloseGroup closes whatever OpenRise or a caller's own group opened.
func CloseGroup(b *strings.Builder) { b.WriteString(`</g>`) }

// FlowPath overlays a travelling dash on a stroke that has already been drawn.
//
// It is a second path rather than a dash pattern on the first, so the connector
// still reads as a solid line when the animation is off — which is what a
// reader with reduced motion, a static export, or a PDF gets.
//
// The 16-point cycle matches the keyframe, and it is short enough that even a
// stub of an arrow between two boxes always has a dash somewhere on it.
func FlowPath(b *strings.Builder, d, color string) {
	fmt.Fprintf(b, `<path d="%s" fill="none" stroke="%s" stroke-width="2" `+
		`stroke-linecap="round" stroke-dasharray="4 12" class="%s"/>`, d, color, ClassFlow)
}
