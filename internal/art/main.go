// Command art draws the diagrams the README uses.
//
// Run it after changing anything it depicts:
//
//	go run ./internal/art
//
// The pictures are generated rather than hand-drawn for the same reason the
// tree diagram is: a hand-drawn SVG goes stale the moment the thing it shows
// changes, and nobody notices until a reader does. These are built from the
// same palette, the same glyph set, and the same drawing primitives as the
// plan diagram, so the README and the tool cannot drift into different
// visual languages. CI regenerates them and fails if the result differs.
//
// Everything is 800 points wide, which sits inside GitHub's README column at
// full size on a desktop and scales down cleanly on a phone.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/draw"
	"github.com/SergeyLubivui-dev/devtree/internal/render/svg"
)

const (
	width = 800.0
	pad   = 24.0
)

// asset is one picture, drawn once per theme.
type asset struct {
	name string
	draw func(th svg.Theme) string
}

func main() {
	assets := []asset{
		{"hero", hero},
		{"why", why},
		{"pipeline", pipeline},
		{"statuses", statuses},
		{"architecture", architecture},
	}

	dir := "docs"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fail(err)
	}

	for _, a := range assets {
		for _, th := range []svg.Theme{svg.Light, svg.Dark} {
			name := a.name + ".svg"
			if th.Name == "dark" {
				name = a.name + "-dark.svg"
			}
			path := filepath.Join(dir, name)
			if err := write(path, a.draw(th)); err != nil {
				fail(err)
			}
		}
	}
}

// write leaves an identical file alone so a no-op run stays out of the diff.
func write(path, content string) error {
	if existing, err := os.ReadFile(path); err == nil && string(existing) == content {
		fmt.Println("  unchanged  " + filepath.ToSlash(path))
		return nil
	}
	fmt.Println("  written    " + filepath.ToSlash(path))
	return os.WriteFile(path, []byte(content), 0o644)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "art: "+err.Error())
	os.Exit(1)
}

// ---------------------------------------------------------------------------
// the pictures
// ---------------------------------------------------------------------------

// hero introduces the project: wordmark, one sentence, and the promises that
// the rest of the README has to keep.
func hero(th svg.Theme) string {
	const height = 250.0
	var b strings.Builder
	open(&b, height, th, "devtree")

	draw.Icon(&b, "tree", pad+4, 28, 32, th.Done)
	draw.Text(&b, "devtree", pad+46, 56, 33, th.Text, "700", "")

	draw.Text(&b, "Tree-shaped development planning that lives inside your repository.",
		pad+4, 88, 13.5, th.Text, "", "")
	draw.Text(&b, "One Go binary. No service, no account, no browser extension.",
		pad+4, 110, 13.5, th.Muted, "", "")

	x := pad + 4
	for _, c := range []struct{ icon, label string }{
		{"package", "one binary"},
		{"flash", "zero dependencies"},
		{"shield-check", "MIT licensed"},
		{"code-square", "Go 1.22+"},
	} {
		x += chip(&b, x, 136, c.icon, c.label, th) + 10
	}

	// A strip of real cards, drifting past forever: the reader sees the visual
	// language of the tool before scrolling to a single example.
	marquee(&b, pad, 182, width-pad*2, 46, th)

	return closeDoc(&b)
}

// ticker is one card on the moving strip.
type ticker struct {
	icon, title, meta, color string
	spin                     bool
}

// marquee draws an endless band of cards inside a clipped window.
//
// The trick is boring on purpose: lay two identical copies of the strip end to
// end and slide the pair by exactly half its own width. When the animation
// wraps, copy two is standing precisely where copy one began, so there is no
// seam to catch. Nothing here needs script, which is what lets it survive
// GitHub's sandbox.
func marquee(b *strings.Builder, x, y, w, h float64, th svg.Theme) {
	items := []ticker{
		{"circle-half-dotted-check", "Authentication", "feat/auth · #12 · @ann", th.InProgress, true},
		{"check-circle", "Stripe", "!44", th.Done, false},
		{"lock-circle", "Password reset", "waiting on SMTP", th.Blocked, false},
		{"clock-circle", "Apple Pay", "#51", th.Todo, false},
		{"circle-half-dotted-check", "OAuth providers", "feat/oauth", th.InProgress, true},
		{"check-circle", "Test suite", "", th.Done, false},
		{"clock-circle", "Homebrew tap", "", th.Todo, false},
	}

	const gap = 14.0
	widths := make([]float64, len(items))
	strip := 0.0
	for i, it := range items {
		text := draw.TextWidth(it.title, 12)
		if m := draw.TextWidth(it.meta, 10); m > text {
			text = m
		}
		widths[i] = text + 52
		strip += widths[i] + gap
	}

	fmt.Fprintf(b, `<clipPath id="dt-band"><rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="10"/></clipPath>`,
		x, y, w, h)
	fmt.Fprintf(b, `<g clip-path="url(#dt-band)">`)
	fmt.Fprintf(b, `<g class="%s">`, draw.ClassMarquee)

	// Two copies, laid end to end. The second one is what the eye is looking
	// at while the first swings back around.
	for copyIndex := 0; copyIndex < 2; copyIndex++ {
		cursor := x + float64(copyIndex)*strip
		for i, it := range items {
			tickerCard(b, fmt.Sprintf("tick-%d-%d", copyIndex, i), it, cursor, y+3, widths[i], h-6, th)
			cursor += widths[i] + gap
		}
	}

	b.WriteString(`</g></g>`)

	// The band fades out at both ends instead of stopping dead against the
	// panel edge, so cards enter and leave rather than appearing and vanishing.
	fmt.Fprintf(b, `<defs><linearGradient id="dt-fade-l" x1="0" x2="1">`+
		`<stop offset="0" stop-color="%s"/><stop offset="1" stop-color="%s" stop-opacity="0"/></linearGradient>`+
		`<linearGradient id="dt-fade-r" x1="0" x2="1">`+
		`<stop offset="0" stop-color="%s" stop-opacity="0"/><stop offset="1" stop-color="%s"/></linearGradient></defs>`,
		th.Canvas, th.Canvas, th.Canvas, th.Canvas)
	fmt.Fprintf(b, `<rect x="%.1f" y="%.1f" width="34" height="%.1f" fill="url(#dt-fade-l)"/>`, x, y, h)
	fmt.Fprintf(b, `<rect x="%.1f" y="%.1f" width="34" height="%.1f" fill="url(#dt-fade-r)"/>`, x+w-34, y, h)
}

// tickerCard is a card small enough to read at a glance while it moves.
func tickerCard(b *strings.Builder, id string, it ticker, x, y, w, h float64, th svg.Theme) {
	card(b, id, x, y, w, h, th, it.color)

	glyph := ""
	if it.spin {
		glyph = draw.ClassSpin
	}
	draw.IconClass(b, it.icon, glyph, x+13, y+(h-16)/2, 16, it.color)
	draw.Text(b, draw.Clip(it.title, w-48, 12), x+36, y+17, 12, th.Text, "600", "")
	if it.meta != "" {
		draw.Text(b, draw.Clip(it.meta, w-48, 10), x+36, y+30, 10, th.Muted, "", "")
	}
}

// why turns the three-line pitch into three cards, because that is what the
// tool would do with it.
func why(th svg.Theme) string {
	const height = 186.0
	var b strings.Builder
	open(&b, height, th, "Why keep the plan in the repository")

	cards := []struct{ icon, title, body, color string }{
		{"eye", "It gets reviewed",
			"A change to the plan lands in the diff, right next to the change to the code.", th.Done},
		{"share", "It gets merged",
			"The node list is flat, so two branches that each added a task merge without a conflict.", th.InProgress},
		{"shield-check", "It stays honest",
			"A pre-commit hook and a CI check refuse to let the diagram drift from the plan.", th.Blocked},
	}

	const (
		gap = 20.0
		w   = (width - pad*2 - gap*2) / 3
		y   = 28.0
		h   = 130.0
	)
	for i, c := range cards {
		x := pad + float64(i)*(w+gap)
		card(&b, fmt.Sprintf("why-%d", i), x, y, w, h, th, c.color)

		// Glyph and heading share a line: the eye reads them as one label
		// instead of two stacked things it has to relate.
		draw.Icon(&b, c.icon, x+18, y+18, 20, c.color)
		draw.Text(&b, draw.Clip(c.title, w-66, 13.5), x+46, y+34, 13.5, th.Text, "600", "")

		for j, line := range draw.Wrap(c.body, w-38, 11.5) {
			draw.Text(&b, line, x+18, y+68+float64(j)*16, 11.5, th.Muted, "", "")
		}
	}

	return closeDoc(&b)
}

// pipeline shows the whole loop end to end: a file, a command, some files, a
// page. Four boxes and three arrows, because that is genuinely all it is.
func pipeline(th svg.Theme) string {
	const height = 150.0
	var b strings.Builder
	open(&b, height, th, "How devtree works")

	stages := []struct{ icon, title, body, color string }{
		{"document-code", "tree.yaml", "the plan, in .devtree/", th.Todo},
		{"code-square", "devtree render", "validate, lay out, draw", th.InProgress},
		{"layers", "your outputs", "Markdown or SVG, in place", th.Done},
		{"monitor", "GitHub", "draws them natively", th.Done},
	}

	const (
		arrow = 42.0
		w     = (width - pad*2 - arrow*3) / 4
		y     = 30.0
		h     = 88.0
	)
	for i, s := range stages {
		x := pad + float64(i)*(w+arrow)
		card(&b, fmt.Sprintf("stage-%d", i), x, y, w, h, th, s.color)

		draw.Icon(&b, s.icon, x+16, y+16, 18, s.color)
		draw.Text(&b, draw.Clip(s.title, w-58, 12.5), x+40, y+30, 12.5, th.Text, "600", "")
		for j, line := range draw.Wrap(s.body, w-34, 10.5) {
			draw.Text(&b, line, x+17, y+58+float64(j)*13, 10.5, th.Muted, "", "")
		}

		// The dash running between stages is the point of the picture: this is
		// a loop that keeps moving, not four things that happen to sit in a row.
		if i < len(stages)-1 {
			flowArrow(&b, x+w+10, y+h/2, arrow-20, th)
		}
	}

	return closeDoc(&b)
}

// statuses is the legend, spelled out with the shorthand the CLI accepts —
// the two things a reader needs at the same moment.
func statuses(th svg.Theme) string {
	const height = 118.0
	var b strings.Builder
	open(&b, height, th, "Statuses")

	items := []struct{ icon, label, short, color string }{
		{"clock-circle", "todo", "t · new · open", th.Todo},
		{"circle-half-dotted-check", "in_progress", "wip · doing · p", th.InProgress},
		{"lock-circle", "blocked", "b · block · stuck", th.Blocked},
		{"check-circle", "done", "d · ok · finished", th.Done},
		{"close-circle", "dropped", "x · drop · cancel", th.Dropped},
	}

	const (
		gap = 12.0
		w   = (width - pad*2 - gap*4) / 5
		y   = 28.0
		h   = 66.0
	)
	for i, it := range items {
		x := pad + float64(i)*(w+gap)
		draw.RoundRect(&b, x, y, w, h, 9, th.Card, th.Border)
		draw.Icon(&b, it.icon, x+14, y+13, 18, it.color)
		draw.Text(&b, it.label, x+38, y+27, 12, th.Text, "600", "")
		draw.Text(&b, draw.Clip(it.short, w-28, 10), x+14, y+52, 10, th.Muted, "", "")
	}

	return closeDoc(&b)
}

// architecture draws the layering as nested boxes rather than a list, because
// the point being made is containment: everything outside may depend on
// everything inside, and never the other way around.
func architecture(th svg.Theme) string {
	layers := []struct {
		label, role, color string
	}{
		{"internal/cli", "flags, dispatch, and every line the user sees", th.InProgress},
		{"internal/store · internal/render · internal/scaffold", "file format, Mermaid and Markdown, templates", th.Done},
		{"internal/render/svg · internal/draw", "layout, cards, themes, text measurement", th.Done},
		{"internal/tree · internal/icons", "the domain and the glyphs — standard library only", th.Blocked},
	}

	// The geometry comes from the number of layers rather than a fixed canvas,
	// so adding one grows the picture instead of leaving the innermost box
	// half empty.
	const (
		inset  = 22.0 // how far each layer sits inside the one around it
		band   = 34.0 // the label strip at the top of a layer
		bottom = 12.0 // breathing room under the innermost box
		top    = 44.0 // room for the note above the outermost box
		tail   = 40.0 // room for the note below it
	)
	count := float64(len(layers))
	inner := count*band + (count-1)*bottom + 10
	height := top + inner + tail

	var b strings.Builder
	open(&b, height, th, "How devtree is built")

	x, y := pad, top
	w, h := width-pad*2, inner

	for _, l := range layers {
		draw.RoundRect(&b, x, y, w, h, 10, th.Card, th.Border)
		draw.RoundRect(&b, x+1, y+10, 3, 20, 1.5, l.color, "")
		draw.Text(&b, l.label, x+14, y+22, 12, th.Text, "600", "")
		draw.Text(&b, draw.Clip(l.role, w-28-draw.TextWidth(l.label, 12), 10.5),
			x+w-14, y+22, 10.5, th.Muted, "", "end")

		x += inset
		y += band
		w -= inset * 2
		h -= band + bottom
	}

	draw.Text(&b, "main.go wires these together and exits — nothing below the outer layer prints, and nothing else calls os.Exit.",
		pad, 26, 11.5, th.Muted, "", "")
	draw.Text(&b, "Every box may import the boxes inside it. Nothing imports outward.",
		pad, height-20, 11.5, th.Muted, "", "")

	return closeDoc(&b)
}

// ---------------------------------------------------------------------------
// shared drawing
// ---------------------------------------------------------------------------

// open starts a document with the panel every picture sits on.
func open(b *strings.Builder, height float64, th svg.Theme, label string) {
	fmt.Fprintf(b, `<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%.0f" `+
		`viewBox="0 0 %.0f %.0f" font-family="%s" role="img" aria-label="%s">`,
		width, height, width, height, draw.FontStack, draw.Escape(label))
	b.WriteString(draw.Stylesheet)
	draw.RoundRect(b, 0.5, 0.5, width-1, height-1, 12, th.Canvas, th.Border)
}

// card draws the shape every picture is made of: a rounded box with a status
// stripe running the full height of its left edge.
//
// The stripe is clipped to the card rather than inset, because a stripe that
// stops short of the corners reads as a decoration, and one that runs the
// whole edge reads as a property of the card.
func card(b *strings.Builder, id string, x, y, w, h float64, th svg.Theme, accent string) {
	const radius = 10.0
	draw.RoundRect(b, x, y, w, h, radius, th.Card, th.Border)
	fmt.Fprintf(b, `<clipPath id="%s"><rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="%.1f"/></clipPath>`,
		id, x, y, w, h, radius)
	fmt.Fprintf(b, `<g clip-path="url(#%s)">`, id)
	draw.RoundRect(b, x, y, 4, h, 0, accent, "")
	b.WriteString(`</g>`)
}

func closeDoc(b *strings.Builder) string {
	b.WriteString("</svg>\n")
	return b.String()
}

// chip is a pill: glyph, label, border. Returns its width so a row of them can
// be laid out without measuring twice.
func chip(b *strings.Builder, x, y float64, icon, label string, th svg.Theme) float64 {
	const h = 26.0
	w := 34 + draw.TextWidth(label, 11)
	draw.RoundRect(b, x, y, w, h, h/2, th.Card, th.Border)
	draw.Icon(b, icon, x+9, y+6, 14, th.Muted)
	draw.Text(b, label, x+28, y+17, 11, th.Text, "500", "")
	return w
}

// flowArrow points from one stage to the next, with a dash travelling along it.
func flowArrow(b *strings.Builder, x, y, length float64, th svg.Theme) {
	line := fmt.Sprintf("M%.1f %.1f H%.1f", x, y, x+length-5)
	fmt.Fprintf(b, `<path d="%s" stroke="%s" stroke-width="1.5" fill="none"/>`, line, th.Edge)
	draw.FlowPath(b, line, th.InProgress)
	fmt.Fprintf(b, `<path d="M%.1f %.1f l-6 -4 v8 z" fill="%s"/>`, x+length, y, th.Edge)
}

// elbowPath is the connector geometry — a right angle with rounded corners,
// the same shape the plan diagram uses between a parent and a child. It
// returns path data so the line and its travelling dash can share one route.
func elbowPath(x1, y1, x2, y2 float64) string {
	mid := x2 - 14
	r := 8.0
	if half := absF(y2-y1) / 2; r > half {
		r = half
	}
	dir := 1.0
	if y2 < y1 {
		dir = -1
	}
	return fmt.Sprintf("M%.1f %.1f H%.1f Q%.1f %.1f %.1f %.1f V%.1f Q%.1f %.1f %.1f %.1f H%.1f",
		x1, y1, mid-r, mid, y1, mid, y1+r*dir, y2-r*dir, mid, y2, mid+r, y2, x2)
}

func absF(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
