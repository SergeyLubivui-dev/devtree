package svg

import "strings"

// fontStack is what the text renders in. Every family here ships with an
// operating system: an SVG served under `default-src 'none'` cannot fetch a
// web font, and a diagram that silently falls back to a serif face looks
// broken.
const fontStack = "system-ui,-apple-system,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif"

// textWidth estimates the rendered width of a string.
//
// A real measurement needs font tables, and shipping those would mean shipping
// a font — the one dependency this program is built to avoid. So widths come
// from per-class averages instead, tuned slightly generous: a card a few pixels
// too wide looks fine, a card a few pixels too narrow clips its own title.
func textWidth(s string, size float64) float64 {
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
			// Non-Latin scripts are typically wider per glyph, and CJK is
			// full-width. Erring high keeps the label inside the card.
			units += 0.95
		default:
			units += 0.545
		}
	}
	return units * size
}

// clip shortens a string to fit, ending it with an ellipsis. It trims by rune
// so a multi-byte character is never cut in half.
func clip(s string, maxWidth, size float64) string {
	if textWidth(s, size) <= maxWidth {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		candidate := strings.TrimRight(string(runes), " ")
		if textWidth(candidate+"…", size) <= maxWidth {
			return candidate + "…"
		}
	}
	return "…"
}

// escapeXML makes a string safe inside an SVG text node or attribute.
//
// Order matters: ampersands first, or the escapes introduced below would
// themselves be re-escaped into &amp;lt; and show up as literal text.
func escapeXML(s string) string {
	return strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	).Replace(s)
}
