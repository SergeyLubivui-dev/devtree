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
	const height = 196.0
	var b strings.Builder
	open(&b, height, th, "devtree")

	draw.Icon(&b, "tree", pad+4, 30, 30, th.Done)
	draw.Text(&b, "devtree", pad+44, 56, 31, th.Text, "700", "")

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
		x += chip(&b, x, 138, c.icon, c.label, th) + 10
	}

	// A skeleton of the real output, drawn without text: it says "this tool
	// makes cards" before the reader has scrolled to a single example.
	motif(&b, 560, 34, th)

	return closeDoc(&b)
}

// motif is the miniature tree in the hero: three cards and two connectors, at
// a size where words would be unreadable, so it carries none.
func motif(b *strings.Builder, x, y float64, th svg.Theme) {
	skeleton := func(x, y, w float64, status, color string) {
		draw.RoundRect(b, x, y, w, 34, 7, th.Card, th.Border)
		draw.RoundRect(b, x+1, y+8, 3, 18, 1.5, color, "")
		draw.Icon(b, status, x+11, y+9, 16, color)
		draw.RoundRect(b, x+33, y+11, w-48, 4.5, 2.25, th.Muted, "")
		draw.RoundRect(b, x+33, y+20, (w-48)*0.55, 4, 2, th.Track, "")
	}

	parent, child := x, x+112
	skeleton(parent, y+46, 96, "circle-half-dotted-check", th.InProgress)
	skeleton(child, y, 104, "check-circle", th.Done)
	skeleton(child, y+92, 104, "clock-circle", th.Todo)

	elbow(b, parent+96, y+63, child, y+17, th.Edge)
	elbow(b, parent+96, y+63, child, y+109, th.Edge)
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
		h   = 134.0
	)
	for i, c := range cards {
		x := pad + float64(i)*(w+gap)
		draw.RoundRect(&b, x, y, w, h, 10, th.Card, th.Border)
		draw.RoundRect(&b, x+1, y+12, 3, h-24, 1.5, c.color, "")

		draw.Icon(&b, c.icon, x+18, y+18, 20, c.color)
		draw.Text(&b, c.title, x+18, y+62, 13.5, th.Text, "600", "")

		for j, line := range draw.Wrap(c.body, w-36, 11.5) {
			draw.Text(&b, line, x+18, y+86+float64(j)*16, 11.5, th.Muted, "", "")
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
		{"document-code", ".devtree/tree.yaml", "edit by hand or by command", th.Todo},
		{"code-square", "devtree render", "validate, lay out, draw", th.InProgress},
		{"layers", "TREE.md · tree.svg", "rewritten between markers", th.Done},
		{"monitor", "GitHub draws it", "no extension, nothing hosted", th.Done},
	}

	const (
		arrow = 34.0
		w     = (width - pad*2 - arrow*3) / 4
		y     = 30.0
		h     = 92.0
	)
	for i, s := range stages {
		x := pad + float64(i)*(w+arrow)
		draw.RoundRect(&b, x, y, w, h, 10, th.Card, th.Border)
		draw.Icon(&b, s.icon, x+16, y+16, 18, s.color)
		draw.Text(&b, draw.Clip(s.title, w-32, 12.5), x+16, y+54, 12.5, th.Text, "600", "")
		for j, line := range draw.Wrap(s.body, w-32, 10.5) {
			draw.Text(&b, line, x+16, y+72+float64(j)*13, 10.5, th.Muted, "", "")
		}
		if i < len(stages)-1 {
			rightArrow(&b, x+w+8, y+h/2, arrow-16, th.Edge)
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
	draw.RoundRect(b, 0.5, 0.5, width-1, height-1, 12, th.Canvas, th.Border)
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

// rightArrow points from one stage to the next.
func rightArrow(b *strings.Builder, x, y, length float64, color string) {
	fmt.Fprintf(b, `<path d="M%.1f %.1f H%.1f" stroke="%s" stroke-width="1.5" fill="none"/>`,
		x, y, x+length-5, color)
	fmt.Fprintf(b, `<path d="M%.1f %.1f l-6 -4 v8 z" fill="%s"/>`, x+length, y, color)
}

// elbow connects two points with a right angle and rounded corners, the same
// shape the plan diagram uses for parent and child.
func elbow(b *strings.Builder, x1, y1, x2, y2 float64, color string) {
	mid := x2 - 14
	r := 8.0
	if half := absF(y2-y1) / 2; r > half {
		r = half
	}
	dir := 1.0
	if y2 < y1 {
		dir = -1
	}
	fmt.Fprintf(b, `<path d="M%.1f %.1f H%.1f Q%.1f %.1f %.1f %.1f V%.1f Q%.1f %.1f %.1f %.1f H%.1f" `+
		`fill="none" stroke="%s" stroke-width="1.5"/>`,
		x1, y1, mid-r, mid, y1, mid, y1+r*dir, y2-r*dir, mid, y2, mid+r, y2, x2, color)
}

func absF(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
