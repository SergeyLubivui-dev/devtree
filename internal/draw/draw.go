// Package draw holds the primitives every SVG devtree writes is built from:
// measuring text, escaping it, wrapping it, and stamping a glyph.
//
// It sits below both the tree renderer (internal/render/svg) and the artwork
// generator (internal/art) so the two cannot drift apart. A card in the README
// diagram and a card in your plan are the same card because they are drawn by
// the same code, not because someone remembered to copy a change across.
//
// Nothing here knows what a plan is. It imports the glyph set and the standard
// library, and that is all.
package draw

import (
	"fmt"
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/icons"
)

// FontStack is what every label renders in. Each family here ships with an
// operating system: an SVG served under `default-src 'none'` cannot fetch a web
// font, and a diagram that silently falls back to a serif face looks broken.
const FontStack = "system-ui,-apple-system,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif"

// TextWidth estimates the rendered width of a string.
//
// A real measurement needs font tables, and shipping those would mean shipping
// a font — the one dependency this program is built to avoid. So widths come
// from per-class averages instead, tuned slightly generous: a card a few pixels
// too wide looks fine, a card a few pixels too narrow clips its own title.
func TextWidth(s string, size float64) float64 {
	var units float64
	for _, r := range s {
		switch {
		case strings.ContainsRune("iljI.,:;'|!`", r):
			units += 0.30
		case strings.ContainsRune("frt()[]{}-/\\ ", r):
			units += 0.40
		case r >= 'A' && r <= 'Z', strings.ContainsRune("mwMW@#%&", r):
			units += 0.68
		case r > 0x7F:
			// Non-Latin scripts run wider per glyph and CJK is full-width.
			// Erring high keeps the label inside its box.
			units += 0.95
		default:
			units += 0.545
		}
	}
	return units * size
}

// Clip shortens a string to fit, ending it with an ellipsis. It trims by rune,
// so a multi-byte character is never cut in half.
func Clip(s string, maxWidth, size float64) string {
	if TextWidth(s, size) <= maxWidth {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		candidate := strings.TrimRight(string(runes), " ")
		if TextWidth(candidate+"…", size) <= maxWidth {
			return candidate + "…"
		}
	}
	return "…"
}

// Wrap breaks a sentence into lines that fit a width.
//
// A word longer than the whole line is clipped rather than hyphenated: this
// draws prose, and a URL that blows past the card edge is worse than an
// ellipsis.
func Wrap(s string, maxWidth, size float64) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}

	var lines []string
	line := words[0]
	for _, word := range words[1:] {
		candidate := line + " " + word
		if TextWidth(candidate, size) <= maxWidth {
			line = candidate
			continue
		}
		lines = append(lines, line)
		line = word
	}
	lines = append(lines, line)

	for i, l := range lines {
		lines[i] = Clip(l, maxWidth, size)
	}
	return lines
}

// Escape makes a string safe inside an SVG text node or attribute.
//
// Order matters: ampersands first, or the escapes introduced below would
// themselves be re-escaped into &amp;lt; and show up as literal text.
func Escape(s string) string {
	return strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	).Replace(s)
}

// Icon stamps a glyph at a size, colored through currentColor so the vendored
// path data never has to be rewritten. An unknown name draws nothing.
func Icon(b *strings.Builder, name string, x, y, size float64, color string) {
	IconClass(b, name, "", x, y, size, color)
}

// IconClass is Icon with an animation class on the glyph.
//
// The class goes on an inner group, never on the one carrying the placement
// transform: a CSS transform replaces the transform attribute outright, so a
// spin applied to the same element would fling the glyph to the origin.
func IconClass(b *strings.Builder, name, class string, x, y, size float64, color string) {
	body, ok := icons.Get(name)
	if !ok {
		return
	}
	fmt.Fprintf(b, `<g transform="translate(%.1f,%.1f) scale(%.4f)" style="color:%s">`,
		x, y, size/icons.GridSize, color)
	marker := ` class="` + icons.MovingPart + `"`
	switch {
	case class == "":
		// Nothing is moving, so the nomination is dead weight.
		b.WriteString(strings.ReplaceAll(body, marker, ""))
	case strings.Contains(body, marker):
		// The glyph has named the part of itself that moves. Turning the whole
		// mark would tumble the check inside a progress ring; only the ring
		// turns, and it turns around the check.
		b.WriteString(strings.ReplaceAll(body, marker, ` class="`+class+`"`))
	default:
		fmt.Fprintf(b, `<g class="%s">%s</g>`, class, body)
	}
	b.WriteString(`</g>`)
}

// Text writes a label. Weight and anchor are passed through as given so the
// caller keeps control of emphasis without a dozen variants of this function.
func Text(b *strings.Builder, s string, x, y, size float64, color, weight, anchor string) {
	fmt.Fprintf(b, `<text x="%.1f" y="%.1f" font-size="%.1f" fill="%s"`, x, y, size, color)
	if weight != "" {
		fmt.Fprintf(b, ` font-weight="%s"`, weight)
	}
	if anchor != "" {
		fmt.Fprintf(b, ` text-anchor="%s"`, anchor)
	}
	fmt.Fprintf(b, `>%s</text>`, Escape(s))
}

// RoundRect draws a filled, optionally stroked rounded rectangle. Passing an
// empty stroke leaves the shape borderless rather than drawing a black edge,
// which is what SVG would do by default.
func RoundRect(b *strings.Builder, x, y, w, h, r float64, fill, stroke string) {
	RoundRectClass(b, "", x, y, w, h, r, fill, stroke)
}

// RoundRectClass is RoundRect with an animation class.
func RoundRectClass(b *strings.Builder, class string, x, y, w, h, r float64, fill, stroke string) {
	fmt.Fprintf(b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="%.1f" fill="%s"`,
		x, y, w, h, r, fill)
	if stroke != "" {
		fmt.Fprintf(b, ` stroke="%s"`, stroke)
	}
	if class != "" {
		fmt.Fprintf(b, ` class="%s"`, class)
	}
	b.WriteString(`/>`)
}

// RoundRectTint draws a rounded box in one colour at two strengths: a wash for
// the fill and a firmer line for the edge.
//
// Opacity rather than a pre-mixed colour, because the mix depends on what is
// behind it. The same tag has to sit on a light page and a near-black one, and
// a colour blended against white by hand turns into a smear on the dark
// drawing. Letting the renderer do the blending keeps one value per status
// instead of one per status per theme.
func RoundRectTint(b *strings.Builder, x, y, w, h, r float64, colour string, fill, stroke float64) {
	fmt.Fprintf(b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="%.1f"`, x, y, w, h, r)
	if fill > 0 {
		fmt.Fprintf(b, ` fill="%s" fill-opacity="%.2f"`, colour, fill)
	} else {
		b.WriteString(` fill="none"`)
	}
	if stroke > 0 {
		fmt.Fprintf(b, ` stroke="%s" stroke-opacity="%.2f"`, colour, stroke)
	}
	b.WriteString(`/>`)
}
