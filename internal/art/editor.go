package main

import (
	"fmt"
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/draw"
	"github.com/SergeyLubivui-dev/devtree/internal/draw/parts"
	"github.com/SergeyLubivui-dev/devtree/internal/render/svg"
)

// editor draws the local editor: the rail, the panel, and the plan drawn beside
// them.
//
// A screenshot would have been quicker and would have been wrong within a
// week — it goes stale the moment a colour or a control changes, and nobody
// notices until a reader does. This is built from the same components the
// editor's own drawings are, so when the vocabulary changes the picture changes
// with it. It is also the component library's first composition: if a card and
// a chip and a connector cannot be assembled into something that looks like the
// product, the library is wrong.
func editor(th svg.Theme) string {
	p := th.Palette()
	const height = 344.0

	var b strings.Builder
	open(&b, height, th, "The devtree editor: the plan on the left, drawn on the right")

	frame := draw.Rect{X: pad, Y: pad, W: width - pad*2, H: height - pad*2}
	draw.RoundRect(&b, frame.X, frame.Y, frame.W, frame.H, 10, th.Card, th.Border)

	rail, rest := frame.SplitLeft(44)
	panel, stage := rest.SplitLeft(214)

	hairline(&b, rail.Right(), frame.Y, frame.Bottom(), th)
	hairline(&b, panel.Right(), frame.Y, frame.Bottom(), th)

	editorRail(&b, rail, th)
	editorPanel(&b, panel, p, th)
	editorStage(&b, stage, p, th)

	return closeDoc(&b)
}

func hairline(b *strings.Builder, x, top, bottom float64, th svg.Theme) {
	fmt.Fprintf(b, `<path d="M%.1f %.1fV%.1f" stroke="%s" stroke-width="1"/>`, x, top, bottom, th.Border)
}

// editorRail is the column of sections: the mark, what the panel can list, and
// the two buttons that act on the whole plan.
func editorRail(b *strings.Builder, r draw.Rect, th svg.Theme) {
	draw.Icon(b, "tree", r.MidX()-9, r.Y+12, 18, th.Done)

	for i, icon := range []string{"nodes", "document-code"} {
		y := r.Y + 48 + float64(i)*32
		if i == 0 {
			draw.RoundRect(b, r.MidX()-13, y-6, 26, 26, 6, th.Track, "")
			draw.RoundRect(b, r.X+1, y-3, 2, 20, 1, th.Done, "")
		}
		draw.Icon(b, icon, r.MidX()-7, y, 14, th.Muted)
	}

	for i, icon := range []string{"monitor", "flash"} {
		draw.Icon(b, icon, r.MidX()-7, r.Bottom()-46+float64(i)*26, 14, th.Muted)
	}
}

// editorPanel is the plan as a list: nesting drawn with guides, and a rollup on
// the right of anything that contains work.
func editorPanel(b *strings.Builder, r draw.Rect, p parts.Palette, th svg.Theme) {
	head, body := r.Pad(12, 12, 10, 14).SplitTop(22)

	draw.Text(b, "Plan", head.X, head.Y+11, 11.5, th.Text, "600", "")
	plus(b, head.Right()-8, head.Y+5, th.Muted)

	// The search line and the All/Open switch, drawn as the shapes they are
	// rather than as a screenshot of them.
	search, rest := body.SplitTop(24)
	magnifier(b, search.X, search.Y+3, 11, th.Muted)
	draw.Text(b, "Search...", search.X+17, search.Y+11, 10, th.Muted, "", "")

	scope, list := rest.SplitTop(28)
	segment(b, draw.Rect{X: scope.X, Y: scope.Y, W: scope.W, H: 21}, th, "All", "Open")

	rows := []struct {
		label  string
		depth  int
		icon   string
		colour string
		motion string
		rollup string
		last   bool
	}{
		{"v0.1 — first release", 0, "check-circle", th.Done, "", "6/6", false},
		{"Domain model", 1, "check-circle", th.Done, "", "", false},
		{"Mermaid renderer", 1, "check-circle", th.Done, "", "", true},
		{"v0.2 — day-to-day use", 0, "circle-half-dotted-check", th.InProgress, draw.ClassSpin, "12/12", false},
		{"Board layout", 1, "check-circle", th.Done, "", "", false},
		{"Local editor", 1, "circle-half-dotted-check", th.InProgress, draw.ClassSpin, "", true},
	}

	for i, row := range rows {
		y := list.Y + 6 + float64(i)*22
		indent := float64(row.depth) * 16

		if row.depth > 0 {
			// The guide: a line from the parent that stops at the elbow of the
			// last child rather than running past it.
			gy := y + 8
			height := 22.0
			if row.last {
				height = 8
			}
			fmt.Fprintf(b, `<path d="M%.1f %.1fv%.1fM%.1f %.1fh6" stroke="%s" stroke-width="1"/>`,
				list.X+7, gy-14, height, list.X+7, gy, th.Border)
		}

		draw.IconClass(b, row.icon, row.motion, list.X+indent, y+1, 13, row.colour)
		colour := th.Text
		if row.depth > 0 {
			colour = th.Muted
		}
		draw.Text(b, draw.Clip(row.label, list.W-indent-46, 10.5), list.X+indent+19, y+11, 10.5, colour, "", "")

		if row.rollup != "" {
			width := draw.TextWidth(row.rollup, 9.5) + 10
			parts.Badge(b, list.Right()-width, y+1, p, row.rollup, th.Done)
		}
	}
}

// editorStage is the drawing, with the header that chooses which one.
func editorStage(b *strings.Builder, r draw.Rect, p parts.Palette, th svg.Theme) {
	head, body := r.SplitTop(38)

	draw.Text(b, "devtree", head.X+16, head.Y+22, 12, th.Text, "600", "")
	chevron(b, head.X+16+draw.TextWidth("devtree", 12)+8, head.Y+20, th.Muted)
	draw.Text(b, "23 / 25", head.X+90, head.Y+22, 10.5, th.Muted, "", "")

	// The view picker, in the shape the header actually wears it.
	picker := draw.Rect{X: head.Right() - 92, Y: head.Y + 9, W: 76, H: 21}
	draw.RoundRect(b, picker.X, picker.Y, picker.W, picker.H, 7, th.Track, "")
	draw.Icon(b, "nodes", picker.X+8, picker.Y+5, 11, th.Muted)
	draw.Text(b, "Tree", picker.X+24, picker.Y+14, 10.5, th.Text, "600", "")
	chevron(b, picker.Right()-12, picker.Y+12, th.Muted)

	hairline2(b, r.X, head.Bottom(), r.Right(), th)

	// The plan itself: a parent and its children, joined the way the tree
	// renderer joins them.
	stage := body.Pad(18, 18, 16, 20)
	parent := draw.Rect{X: stage.X, Y: stage.MidY() - 21, W: 150, H: 42}

	parts.Card(b, parent, p, parts.CardStyle{
		Accent: th.InProgress, Title: "v0.2", Meta: "12 / 12 done",
		Glyph: "circle-half-dotted-check", Motion: draw.ClassSpin,
	})

	children := []struct {
		title, meta, icon, colour, motion string
	}{
		{"Board layout", "docs · shipped", "check-circle", th.Done, ""},
		{"Local editor", "feat/serve · ann", "circle-half-dotted-check", th.InProgress, draw.ClassSpin},
		{"Archive", "waiting on review", "lock-circle", th.Blocked, draw.ClassPulse},
	}

	right := draw.Rect{X: stage.X + 214, Y: stage.Y, W: stage.Right() - stage.X - 214, H: stage.H}
	for i, kid := range children {
		box := draw.Rect{X: right.X, Y: right.Y + float64(i)*(42+14), W: right.W, H: 42}

		parts.Connector(b, draw.Point{X: parent.Right(), Y: parent.MidY()},
			draw.Point{X: box.X, Y: box.MidY()}, p, th.Edge, motionFor(i))

		parts.Card(b, box, p, parts.CardStyle{
			Accent: kid.colour, Title: kid.title, Meta: kid.meta,
			Glyph: kid.icon, Motion: kid.motion,
		})

		// The affordance the drawing carries: hover a card and a + offers to
		// add work underneath it.
		if i == 1 {
			cx, cy := box.Right()-4, box.MidY()
			fmt.Fprintf(b, `<circle cx="%.1f" cy="%.1f" r="10" fill="%s" stroke="%s" stroke-width="1"/>`,
				cx, cy, th.Card, th.Done)
			plus(b, cx-5, cy-5, th.Done)
		}
	}
}

// motionFor puts the travelling dash on the one path that is live. Every
// connector flowing would say nothing; one flowing says which branch is moving.
func motionFor(i int) string {
	if i == 1 {
		return draw.ClassFlow
	}
	return ""
}

func hairline2(b *strings.Builder, x1, y, x2 float64, th svg.Theme) {
	fmt.Fprintf(b, `<path d="M%.1f %.1fH%.1f" stroke="%s" stroke-width="1"/>`, x1, y, x2, th.Border)
}

func plus(b *strings.Builder, x, y float64, colour string) {
	fmt.Fprintf(b, `<path d="M%.1f %.1fv10M%.1f %.1fh10" stroke="%s" stroke-width="1.4" stroke-linecap="round"/>`,
		x+5, y, x, y+5, colour)
}

// magnifier is drawn rather than taken from the icon set on purpose. The
// vendored glyphs are the status vocabulary — the marks a plan is read with —
// and the interface's own furniture is stroked, so the two never compete. The
// editor's page draws this same mark the same way.
func magnifier(b *strings.Builder, x, y, size float64, colour string) {
	r := size * 0.34
	cx, cy := x+r+1, y+r+1
	fmt.Fprintf(b, `<g fill="none" stroke="%s" stroke-width="1.2" stroke-linecap="round">`+
		`<circle cx="%.1f" cy="%.1f" r="%.1f"/><path d="M%.1f %.1fl%.1f %.1f"/></g>`,
		colour, cx, cy, r, cx+r*0.75, cy+r*0.75, size*0.28, size*0.28)
}

func chevron(b *strings.Builder, x, y float64, colour string) {
	fmt.Fprintf(b, `<path d="M%.1f %.1fl3.5 3.5 3.5 -3.5" fill="none" stroke="%s" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/>`,
		x-3.5, y-1.5, colour)
}

// segment draws the two-state switch the panel filters with.
func segment(b *strings.Builder, r draw.Rect, th svg.Theme, first, second string) {
	draw.RoundRect(b, r.X, r.Y, r.W, r.H, 7, th.Canvas, th.Border)
	half := r.W / 2
	draw.RoundRect(b, r.X+2, r.Y+2, half-2, r.H-4, 5, th.Track, "")
	draw.Text(b, first, r.X+half/2, r.Y+r.H/2+3.5, 10, th.Text, "600", "middle")
	draw.Text(b, second, r.X+half+half/2, r.Y+r.H/2+3.5, 10, th.Muted, "", "middle")
}
