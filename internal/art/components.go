package main

import (
	"fmt"
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/draw"
	"github.com/SergeyLubivui-dev/devtree/internal/draw/parts"
	"github.com/SergeyLubivui-dev/devtree/internal/render/svg"
)

// components draws the whole vocabulary once, in a grid.
//
// It is documentation that cannot go stale: the gallery walks parts.Names(),
// so a component added without a demo here fails the build rather than quietly
// going undocumented. It is also the library's first real caller — a component
// that is awkward to use shows up as awkward code on this page before anybody
// builds a diagram out of it.
func components(th svg.Theme) string {
	p := th.Palette()

	const (
		cols     = 2
		cellW    = 366.0
		cellH    = 104.0
		gutter   = 20.0
		headerH  = 86.0
		captionH = 18.0
	)

	demos := gallery(p)
	rows := (len(parts.Names()) + cols - 1) / cols
	height := headerH + float64(rows)*cellH + pad

	var b strings.Builder
	open(&b, height, th, "devtree drawing components")

	draw.Icon(&b, "layers", pad, pad-2, 22, th.Done)
	draw.Text(&b, "Drawing components", pad+30, pad+15, 17, th.Text, "600", "")
	draw.Text(&b, draw.Plural(len(parts.Names()), "component", "components")+
		" — the vocabulary every devtree diagram is assembled from",
		pad, pad+36, 11, th.Muted, "", "")

	grid := draw.Rect{X: pad, Y: headerH, W: width - pad*2, H: float64(rows) * cellH}

	for i, name := range parts.Names() {
		cell := draw.Rect{
			X: grid.X + float64(i%cols)*(cellW+gutter),
			Y: grid.Y + float64(i/cols)*cellH,
			W: cellW,
			H: cellH,
		}

		caption, stage := cell.SplitTop(captionH)
		draw.Text(&b, fmt.Sprintf("%02d", i+1), caption.X, caption.Y+10, 9.5, th.Muted, "600", "")
		draw.Text(&b, name, caption.X+22, caption.Y+10, 10.5, th.Text, "600", "")

		demo, ok := demos[name]
		if !ok {
			// Every named component has a demo or the gallery is lying about
			// what the library can do.
			fail(fmt.Errorf("no gallery demo for %s", name))
		}
		demo(&b, stage.Pad(0, 0, 14, 0))
	}

	return closeDoc(&b)
}

// gallery is one demo per component, each given the box its cell leaves it and
// drawn with the plan-shaped content it exists for — a real branch name, a real
// rollup — rather than lorem ipsum that hides how it behaves with the truth.
func gallery(p parts.Palette) map[string]func(b *strings.Builder, r draw.Rect) {
	const (
		red    = "#cf222e"
		amber  = "#d29922"
		green  = "#3fb950"
		greyed = "#8c959f"
	)

	return map[string]func(b *strings.Builder, r draw.Rect){
		"Card": func(b *strings.Builder, r draw.Rect) {
			parts.Card(b, draw.Rect{X: r.X, Y: r.MidY() - 23, W: r.W, H: 46}, p, parts.CardStyle{
				Accent: amber, Title: "Authentication", Meta: "feat/auth · ann · #12",
				Glyph: "circle-half-dotted-check", Motion: draw.ClassSpin,
				Trailer: func(b *strings.Builder, box draw.Rect) {
					parts.Bar(b, draw.Rect{X: box.X, Y: box.MidY() - 3, W: 40, H: 5}, p, 3, 7, green, draw.ClassGrow)
				},
			})
		},
		"Cluster": func(b *strings.Builder, r draw.Rect) {
			box := draw.Rect{X: r.X + 2, Y: r.Y + 10, W: r.W - 4, H: r.H - 18}
			parts.Cluster(b, box, p, "MVP")
			parts.Chip(b, box.X+16, box.MidY()-8, p, "check-circle", "shipped", green)
		},
		"Callout": func(b *strings.Builder, r draw.Rect) {
			parts.Callout(b, draw.Rect{X: r.X, Y: r.Y + 4, W: r.W, H: r.H - 8}, p,
				"note", "The plan is one file. Every picture here is drawn from it.", "")
		},
		"Swimlane": func(b *strings.Builder, r draw.Rect) {
			lane := parts.Swimlane(b, draw.Rect{X: r.X, Y: r.MidY() - 21, W: r.W, H: 42}, p, "Backend", 76)
			parts.Chip(b, lane.X, lane.MidY()-8, p, "code-circle", "api", "")
		},
		"ColumnHead": func(b *strings.Builder, r draw.Rect) {
			parts.ColumnHead(b, draw.Rect{X: r.X, Y: r.MidY() - 14, W: r.W, H: 26}, p,
				"lock-circle", "blocked", 2, red, draw.ClassPulse)
		},
		"Stat": func(b *strings.Builder, r draw.Rect) {
			parts.Stat(b, draw.Rect{X: r.X, Y: r.MidY() - 24, W: 128, H: 48}, p, "23 / 25", "tasks done", "")
			parts.Stat(b, draw.Rect{X: r.X + 140, Y: r.MidY() - 24, W: 96, H: 48}, p, "92%", "complete", green)
		},

		"Chip": func(b *strings.Builder, r draw.Rect) {
			y := r.MidY() - 8
			at := r.X
			at += parts.Chip(b, at, y, p, "share", "feat/auth", "") + 8
			at += parts.Chip(b, at, y, p, "user", "ann", "") + 8
			parts.Chip(b, at, y, p, "hashtag", "backend", "")
		},
		"Badge": func(b *strings.Builder, r draw.Rect) {
			y := r.MidY() - 8
			at := r.X
			at += parts.Badge(b, at, y, p, "6/6", green) + 8
			at += parts.Badge(b, at, y, p, "3/7", "") + 8
			parts.Badge(b, at, y, p, "12", "")
		},
		"Pill": func(b *strings.Builder, r draw.Rect) {
			y := r.MidY() - 10
			at := r.X
			at += parts.Pill(b, at, y, p, "check-circle", "done", green, "") + 8
			at += parts.Pill(b, at, y, p, "lock-circle", "blocked", red, draw.ClassPulse) + 8
			parts.Pill(b, at, y, p, "clock-circle", "not started", greyed, "")
		},
		"Avatar": func(b *strings.Builder, r draw.Rect) {
			for i, name := range []string{"Ann Marie", "bob", "Юлия Петрова"} {
				parts.Avatar(b, r.X+float64(i)*36, r.MidY()-14, 28, p, name, "")
			}
		},
		"Milestone": func(b *strings.Builder, r draw.Rect) {
			for i, label := range []string{"v0.1", "v0.2", "v1.0"} {
				colour := green
				if i == 2 {
					colour = greyed
				}
				parts.Milestone(b, r.X+22+float64(i)*84, r.MidY()-6, 13, p, label, colour)
			}
		},
		"Legend": func(b *strings.Builder, r draw.Rect) {
			parts.Legend(b, r.X, r.MidY()-16, p, []parts.LegendItem{
				{Icon: "check-circle", Label: "done", Colour: green},
				{Icon: "circle-half-dotted-check", Label: "in progress", Colour: amber, Motion: draw.ClassSpin},
			}, 18)
			parts.Legend(b, r.X, r.MidY()+3, p, []parts.LegendItem{
				{Icon: "lock-circle", Label: "blocked", Colour: red, Motion: draw.ClassPulse},
				{Icon: "clock-circle", Label: "not started", Colour: greyed},
			}, 18)
		},

		"Bar": func(b *strings.Builder, r draw.Rect) {
			parts.Bar(b, draw.Rect{X: r.X, Y: r.MidY() - 12, W: r.W - 60, H: 6}, p, 23, 25, green, draw.ClassGrow)
			parts.Bar(b, draw.Rect{X: r.X, Y: r.MidY() + 4, W: r.W - 60, H: 6}, p, 1, 4, amber, draw.ClassGrow)
			draw.Text(b, "23 / 25", r.Right(), r.MidY()-6, 10, p.Muted, "", "end")
			draw.Text(b, "1 / 4", r.Right(), r.MidY()+10, 10, p.Muted, "", "end")
		},
		"Ring": func(b *strings.Builder, r draw.Rect) {
			for i, v := range []struct {
				done, total int
				colour      string
			}{{6, 6, green}, {3, 7, amber}, {0, 5, greyed}} {
				cx := r.X + 22 + float64(i)*70
				parts.Ring(b, cx, r.MidY(), 17, 4, p, v.done, v.total, v.colour)
				draw.Text(b, fmt.Sprintf("%d%%", draw.Percent(v.done, v.total)), cx, r.MidY()+3.5, 9.5, p.Muted, "600", "middle")
			}
		},
		"Sparkline": func(b *strings.Builder, r draw.Rect) {
			parts.Sparkline(b, draw.Rect{X: r.X, Y: r.MidY() - 18, W: r.W - 70, H: 34}, p,
				[]float64{2, 4, 4, 7, 9, 11, 14, 18, 21, 23}, green)
			draw.Text(b, "10 weeks", r.Right(), r.MidY()+4, 10, p.Faint, "", "end")
		},
		"Timeline": func(b *strings.Builder, r draw.Rect) {
			parts.Timeline(b, draw.Rect{X: r.X + 10, Y: r.MidY() - 14, W: r.W - 20, H: 24}, p, []parts.Tick{
				{At: 0, Label: "v0.1", Mark: "check-circle", Color: green},
				{At: 0.45, Label: "v0.2", Mark: "check-circle", Color: green},
				{At: 1, Label: "v1.0", Mark: "clock-circle", Color: greyed},
			})
		},

		"Divider": func(b *strings.Builder, r draw.Rect) {
			parts.Divider(b, draw.Rect{X: r.X, Y: r.MidY() - 14, W: r.W, H: 12}, p, "")
			parts.Divider(b, draw.Rect{X: r.X, Y: r.MidY() + 6, W: r.W, H: 12}, p, "archived")
		},
		"Connector": func(b *strings.Builder, r draw.Rect) {
			parts.Connector(b, draw.Point{X: r.X, Y: r.MidY() - 18}, draw.Point{X: r.Right() - 6, Y: r.MidY() - 18}, p, "", "")
			parts.Connector(b, draw.Point{X: r.X, Y: r.MidY() + 4}, draw.Point{X: r.Right() - 6, Y: r.MidY() + 22}, p, green, draw.ClassFlow)
		},
		"Arrow": func(b *strings.Builder, r draw.Rect) {
			parts.Arrow(b, draw.Point{X: r.X, Y: r.MidY() - 14}, draw.Point{X: r.Right() - 6, Y: r.MidY() - 14}, p, "", "")
			parts.Arrow(b, draw.Point{X: r.X, Y: r.MidY() + 6}, draw.Point{X: r.Right() - 6, Y: r.MidY() + 22}, p, amber, draw.ClassFlow)
		},
		"Brace": func(b *strings.Builder, r draw.Rect) {
			const label = "shipped together"
			parts.Brace(b, r.X+6, r.Y+6, r.Bottom()-6, p, label, 14)
			// Asking the component where its label ends, rather than guessing a
			// number that goes wrong the day the label changes.
			at := r.X + 6 + parts.BraceLabelOffset(14, label)
			for i, name := range []string{"parser", "renderer", "cli"} {
				parts.Chip(b, at, r.Y+8+float64(i)*22, p, "", name, "")
			}
		},
	}
}
