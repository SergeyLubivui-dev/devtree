package draw

import (
	"github.com/SergeyLubivui-dev/devtree/internal/icons"
	"strings"
	"testing"
)

func TestTextWidthGrowsWithTheString(t *testing.T) {
	if TextWidth("", 12) != 0 {
		t.Error("an empty string has no width")
	}
	if TextWidth("iii", 12) >= TextWidth("MMM", 12) {
		t.Error("narrow letters should measure narrower than wide ones")
	}
	if TextWidth("abc", 12) >= TextWidth("abc", 24) {
		t.Error("width should scale with font size")
	}
}

func TestClipKeepsWhatFits(t *testing.T) {
	if got := Clip("short", 500, 12); got != "short" {
		t.Errorf("Clip trimmed a string that already fits: %q", got)
	}
}

func TestClipShortensWithoutBreakingRunes(t *testing.T) {
	const original = "Аутентификация через OAuth 2.0"

	got := Clip(original, 60, 12)
	if !strings.HasSuffix(got, "…") {
		t.Errorf("Clip did not mark the string as shortened: %q", got)
	}
	if len([]rune(got)) >= len([]rune(original)) {
		t.Errorf("Clip returned %q, which is no shorter than the original", got)
	}
	if strings.ContainsRune(got, '�') {
		t.Errorf("Clip cut a multi-byte character in half: %q", got)
	}
	if TextWidth(got, 12) > 60 {
		t.Errorf("Clip returned %q, still wider than the limit", got)
	}
}

func TestWrapFillsLinesToTheLimit(t *testing.T) {
	const sentence = "The node list is flat, so two branches that each added a task merge cleanly."

	lines := Wrap(sentence, 160, 11)
	if len(lines) < 2 {
		t.Fatalf("expected the sentence to wrap, got %d line(s): %q", len(lines), lines)
	}
	for _, line := range lines {
		if w := TextWidth(line, 11); w > 160 {
			t.Errorf("line %q measures %.1f, over the 160 limit", line, w)
		}
	}
	if joined := strings.Join(lines, " "); joined != sentence {
		t.Errorf("wrapping changed the text:\n got %q\nwant %q", joined, sentence)
	}
}

func TestWrapClipsAWordThatCannotFit(t *testing.T) {
	lines := Wrap("supercalifragilisticexpialidocious", 40, 11)
	if len(lines) != 1 {
		t.Fatalf("expected one clipped line, got %q", lines)
	}
	if !strings.HasSuffix(lines[0], "…") {
		t.Errorf("an unbreakable word should be clipped, got %q", lines[0])
	}
}

func TestWrapHandlesEmptyInput(t *testing.T) {
	if lines := Wrap("   ", 100, 11); len(lines) != 0 {
		t.Errorf("blank input should produce no lines, got %q", lines)
	}
}

func TestEscape(t *testing.T) {
	got := Escape(`a & b < c > "d" 'e'`)
	want := `a &amp; b &lt; c &gt; &quot;d&quot; &apos;e&apos;`
	if got != want {
		t.Errorf("Escape() = %q, want %q", got, want)
	}
	if strings.Contains(Escape("<b>"), "&amp;lt;") {
		t.Error("escapes were double-escaped")
	}
}

func TestIconOnlyDrawsWhatItKnows(t *testing.T) {
	var b strings.Builder
	Icon(&b, "check-circle", 10, 20, 18, "#1a7f37")
	if !strings.Contains(b.String(), "translate(10.0,20.0)") || !strings.Contains(b.String(), "#1a7f37") {
		t.Errorf("glyph was not placed or colored:\n%s", b.String())
	}
	if !strings.Contains(b.String(), "scale(0.7500)") {
		t.Errorf("18px on a 24px grid should scale by 0.75:\n%s", b.String())
	}

	b.Reset()
	Icon(&b, "no-such-icon", 0, 0, 18, "#000")
	if b.Len() != 0 {
		t.Errorf("an unknown glyph should draw nothing, got %q", b.String())
	}
}

func TestIconClassKeepsThePlacementTransform(t *testing.T) {
	var b strings.Builder
	IconClass(&b, "check-circle", ClassSpin, 10, 20, 18, "#1a7f37")

	out := b.String()
	if !strings.Contains(out, `transform="translate(10.0,20.0) scale(0.7500)"`) {
		t.Errorf("the placement transform was lost:\n%s", out)
	}
	// The class must sit on an inner group. A CSS transform replaces the
	// transform attribute outright, so sharing one element would fling the
	// glyph to the origin the moment it starts turning.
	if !strings.Contains(out, `><g class="`+ClassSpin+`"><path`) {
		t.Errorf("the animation class should wrap the glyph, not the placement:\n%s", out)
	}
}

func TestEntranceStaggerIsCapped(t *testing.T) {
	var early, late strings.Builder
	OpenRise(&early, 1)
	OpenRise(&late, 500)

	if !strings.Contains(early.String(), "animation-delay:0.04s") {
		t.Errorf("got %q", early.String())
	}
	// A plan with hundreds of nodes must not take minutes to finish appearing.
	if !strings.Contains(late.String(), "animation-delay:0.70s") {
		t.Errorf("the stagger should stop growing, got %q", late.String())
	}
}

func TestPrintingAndReducedMotionStopEverything(t *testing.T) {
	// A renderer that freezes the first frame of a fade-in would print a blank
	// card, so printing has to switch the animations off entirely.
	if !strings.Contains(Stylesheet, "print") {
		t.Error("the stylesheet should disable animation for print")
	}
	if !strings.Contains(Stylesheet, "backwards") || strings.Contains(Stylesheet, "dt-rise .55s cubic-bezier(.2,.8,.2,1) both") {
		t.Error("the entrance should end on the element's own style, not in the animation's last frame")
	}
}

func TestStylesheetIsSafeAndSwitchable(t *testing.T) {
	for _, want := range []string{ClassFlow, ClassGrow, ClassSpin, ClassRise, ClassPulse, ClassMarquee, "prefers-reduced-motion"} {
		if !strings.Contains(Stylesheet, want) {
			t.Errorf("the stylesheet is missing %q", want)
		}
	}
	// GitHub serves repository SVGs under default-src 'none' with sandbox: a
	// script would silently never run, so none may be emitted.
	if strings.Contains(Stylesheet, "<script") || strings.Contains(Stylesheet, "url(") {
		t.Error("the stylesheet must not pull in script or external resources")
	}
}

func TestFlowPathLeavesTheLineUnderneathIntact(t *testing.T) {
	var b strings.Builder
	FlowPath(&b, "M0 0 H50", "#bf8700")

	out := b.String()
	if !strings.Contains(out, `d="M0 0 H50"`) {
		t.Error("the dash should follow the same route as the line it decorates")
	}
	if !strings.Contains(out, `class="`+ClassFlow+`"`) || !strings.Contains(out, "stroke-dasharray") {
		t.Errorf("expected an animated dash pattern:\n%s", out)
	}
}

func TestTextAndRoundRectStayOptional(t *testing.T) {
	var b strings.Builder
	Text(&b, "plain", 1, 2, 12, "#000", "", "")
	if strings.Contains(b.String(), "font-weight") || strings.Contains(b.String(), "text-anchor") {
		t.Errorf("empty options should not be written:\n%s", b.String())
	}

	b.Reset()
	RoundRect(&b, 0, 0, 10, 10, 2, "#fff", "")
	if strings.Contains(b.String(), "stroke") {
		t.Errorf("an empty stroke should be left off entirely:\n%s", b.String())
	}
}

func TestAGlyphCanNominateThePartThatMoves(t *testing.T) {
	// The progress mark is a ring around a check. Turning the whole glyph
	// tumbles the check with it, which reads as a mark falling over rather
	// than as work in flight.
	var b strings.Builder
	IconClass(&b, "circle-half-dotted-check", ClassSpin, 0, 0, 16, "#d29922")
	out := b.String()

	ring := strings.Index(out, `<g class="`+ClassSpin+`">`)
	if ring < 0 {
		t.Fatal("nothing in the glyph is turning")
	}
	end := strings.Index(out[ring:], "</g>")
	if end < 0 {
		t.Fatal("the turning group is never closed")
	}

	turning := out[ring : ring+end]
	if !strings.Contains(turning, "<path") || !strings.Contains(turning, "<circle") {
		t.Error("the ring is not what is turning")
	}
	if strings.Contains(turning, "<polyline") {
		t.Error("the check is inside the turning group")
	}
	if !strings.Contains(out[ring+end:], "<polyline") {
		t.Error("the check is missing from the glyph, or drawn before the ring closes")
	}

	// A still drawing should not carry the nomination as a dead class.
	var still strings.Builder
	Icon(&still, "circle-half-dotted-check", 0, 0, 16, "#d29922")
	if strings.Contains(still.String(), "dt-moving-part") {
		t.Error("a still glyph kept the marker")
	}
	// Stripping the class must leave a clean group rather than `<g >`.
	if !strings.Contains(still.String(), "<g><") || strings.Contains(still.String(), "<g >") {
		t.Errorf("stripping the marker left the group malformed: %s", still.String())
	}
}

func TestTheNewMotionIsGuardedLikeTheRest(t *testing.T) {
	// A drawing that is printed, or read by somebody who has asked for less
	// motion, must show the finished state — not the first frame of it. For
	// the two that carry their own dash and offset that means saying so
	// explicitly rather than only switching the animation off.
	guard := Stylesheet[strings.Index(Stylesheet, "prefers-reduced-motion"):]

	for _, class := range []string{ClassDraw, ClassRoll} {
		if !strings.Contains(guard, class) {
			t.Errorf("%s keeps moving under reduced motion", class)
		}
	}
	if !strings.Contains(guard, "stroke-dashoffset: 0") {
		t.Error("a drawn-on path would stay half-drawn with the motion off")
	}
	if !strings.Contains(guard, "--dt-roll-y") {
		t.Error("a counter would stay on its first value with the motion off")
	}
}

func TestTheTurningPartIsAnchoredToTheGrid(t *testing.T) {
	// A rotation is centred on the box of what is being rotated. The ring's own
	// geometry sits half a unit off the icon's centre — the arc covers the right
	// half, the dots the left — so without an anchor the ring orbits instead of
	// turning, and a ring wobbling around a stationary check reads as the whole
	// mark being thrown about.
	body, ok := icons.Get("circle-half-dotted-check")
	if !ok {
		t.Fatal("the progress glyph is missing")
	}

	open := strings.Index(body, `<g class="`+icons.MovingPart+`">`)
	if open < 0 {
		t.Fatal("the glyph names no moving part")
	}
	group := body[open : strings.Index(body[open:], "</g>")+open]

	if !strings.Contains(group, `width="24" height="24" fill="none"`) {
		t.Error("the turning group is not anchored to the icon's grid")
	}
	// The anchor has to be inside the group that turns, or it anchors nothing.
	if strings.Index(body, `width="24" height="24"`) > strings.Index(body, "</g>") {
		t.Error("the anchor is outside the group it is meant to centre")
	}
}
