package parts

import (
	"fmt"
	"math"
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/draw"
)

// ---------------------------------------------------------------------------
// 13. Bar
// ---------------------------------------------------------------------------

// Bar is progress along a line: a track, a fill, and nothing else.
//
// The fill grows in once on load when a motion class is given. It is the one
// place in a diagram where the eye should be pulled first, and it settles
// immediately after — an indicator that keeps moving stops indicating.
func Bar(b *strings.Builder, r draw.Rect, p Palette, done, total int, colour, motion string) {
	if colour == "" {
		colour = p.Accent
	}
	radius := r.H / 2
	draw.RoundRect(b, r.X, r.Y, r.W, r.H, radius, p.Track, "")

	if total <= 0 || done <= 0 {
		return
	}
	width := r.W * float64(done) / float64(total)
	if motion == "" {
		draw.RoundRect(b, r.X, r.Y, width, r.H, radius, colour, "")
		return
	}
	draw.RoundRectClass(b, motion, r.X, r.Y, width, r.H, radius, colour, "")
}

// ---------------------------------------------------------------------------
// 14. Ring
// ---------------------------------------------------------------------------

// Ring is the same progress in a circle, for when the number matters more than
// the comparison — a milestone's own completeness rather than its share of a
// project.
//
// Drawn with a dash offset rather than an arc path: an arc has to special-case
// the half-way point, and a value that renders wrong at exactly 50% is the kind
// of bug that ships.
func Ring(b *strings.Builder, cx, cy, radius, thickness float64, p Palette, done, total int, colour string) {
	if colour == "" {
		colour = p.Accent
	}
	circumference := 2 * math.Pi * radius

	fmt.Fprintf(b, `<circle cx="%.1f" cy="%.1f" r="%.1f" fill="none" stroke="%s" stroke-width="%.1f"/>`,
		cx, cy, radius, p.Track, thickness)

	if total <= 0 || done <= 0 {
		return
	}
	filled := circumference * float64(done) / float64(total)
	fmt.Fprintf(b, `<circle cx="%.1f" cy="%.1f" r="%.1f" fill="none" stroke="%s" stroke-width="%.1f" `+
		`stroke-linecap="round" stroke-dasharray="%.2f %.2f" transform="rotate(-90 %.1f %.1f)"/>`,
		cx, cy, radius, colour, thickness, filled, circumference-filled, cx, cy)
}

// ---------------------------------------------------------------------------
// 15. Sparkline
// ---------------------------------------------------------------------------

// Sparkline is a shape rather than a chart: how a number has been moving, in
// the space of a word.
//
// No axes, no labels, no grid. Everything a full chart would add is a thing the
// reader has to decode, and the only question a sparkline answers is "which
// way". A flat series draws down the middle instead of dividing by zero.
func Sparkline(b *strings.Builder, r draw.Rect, p Palette, values []float64, colour string) {
	if colour == "" {
		colour = p.Accent
	}
	if len(values) < 2 {
		return
	}

	lo, hi := values[0], values[0]
	for _, v := range values {
		lo, hi = math.Min(lo, v), math.Max(hi, v)
	}
	span := hi - lo

	var d strings.Builder
	for i, v := range values {
		x := r.X + r.W*float64(i)/float64(len(values)-1)
		y := r.MidY()
		if span > 0 {
			y = r.Bottom() - (v-lo)/span*r.H
		}
		if i == 0 {
			fmt.Fprintf(&d, "M%.1f %.1f", x, y)
			continue
		}
		fmt.Fprintf(&d, "L%.1f %.1f", x, y)
	}

	fmt.Fprintf(b, `<path d="%s" fill="none" stroke="%s" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>`,
		d.String(), colour)
}

// ---------------------------------------------------------------------------
// 16. Timeline
// ---------------------------------------------------------------------------

// Tick is one labelled position on a timeline, at a fraction of its length.
type Tick struct {
	At    float64 // 0 to 1
	Label string
	Mark  string // an icon at the tick, optional
	Color string
}

// Timeline is an axis with labelled ticks: releases, sprints, the dates a plan
// is measured against.
//
// Positions are fractions rather than dates, because devtree does not know what
// a date is and should not learn here. Whoever has the dates works out the
// fractions, and this draws them.
func Timeline(b *strings.Builder, r draw.Rect, p Palette, ticks []Tick) {
	axis := r.MidY()
	fmt.Fprintf(b, `<path d="M%.1f %.1fH%.1f" stroke="%s" stroke-width="1"/>`,
		r.X, axis, r.Right(), p.Border)

	for _, tick := range ticks {
		colour := tick.Color
		if colour == "" {
			colour = p.Muted
		}
		x := r.X + r.W*clamp01(tick.At)

		if tick.Mark != "" {
			draw.Icon(b, tick.Mark, x-6, axis-6, 12, colour)
		} else {
			fmt.Fprintf(b, `<circle cx="%.1f" cy="%.1f" r="3" fill="%s"/>`, x, axis, colour)
		}
		if tick.Label != "" {
			draw.Text(b, tick.Label, x, axis+18, 10, p.Faint, "", "middle")
		}
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
