package draw

import (
	"fmt"
	"strings"
	"unicode"
)

// Rect is a box on the canvas, and the unit every component is placed with.
//
// Diagram code that carries x, y, w and h as four loose numbers spends most of
// its length recomputing them: a card's title is at x+14, its meta line at
// y+34, its badge at x+w-40. Every one of those is a place to be off by two,
// and none of them says what it meant. A Rect that can be inset and split says
// it once — "the header is the top 34 of the card, padded by 12" — and the
// arithmetic stops being the interesting part of the file.
type Rect struct{ X, Y, W, H float64 }

// Point is a position with no size, for the ends of connectors.
type Point struct{ X, Y float64 }

func (r Rect) Right() float64   { return r.X + r.W }
func (r Rect) Bottom() float64  { return r.Y + r.H }
func (r Rect) MidX() float64    { return r.X + r.W/2 }
func (r Rect) MidY() float64    { return r.Y + r.H/2 }
func (r Rect) Empty() bool      { return r.W <= 0 || r.H <= 0 }
func (r Rect) Center() Point    { return Point{r.MidX(), r.MidY()} }
func (r Rect) TopLeft() Point   { return Point{r.X, r.Y} }
func (r Rect) LeftEdge() Point  { return Point{r.X, r.MidY()} }
func (r Rect) RightEdge() Point { return Point{r.Right(), r.MidY()} }

// Inset shrinks a box on every side. A negative amount grows it, which is how
// a highlight ring around a card is drawn without a second set of numbers.
func (r Rect) Inset(by float64) Rect { return r.InsetXY(by, by) }

// InsetXY shrinks horizontally and vertically by different amounts, the common
// case for text inside a box: more room at the sides than above and below.
func (r Rect) InsetXY(x, y float64) Rect {
	return Rect{r.X + x, r.Y + y, r.W - x*2, r.H - y*2}
}

// Pad insets each side separately, in the order a stylesheet writes them.
func (r Rect) Pad(top, right, bottom, left float64) Rect {
	return Rect{r.X + left, r.Y + top, r.W - left - right, r.H - top - bottom}
}

// SplitTop takes a band off the top and returns it with what is left. The
// remainder comes second so a caller can keep splitting the rest:
//
//	head, body := card.SplitTop(34)
//	meta, body := body.SplitBottom(20)
func (r Rect) SplitTop(h float64) (Rect, Rect) {
	h = clamp(h, 0, r.H)
	return Rect{r.X, r.Y, r.W, h}, Rect{r.X, r.Y + h, r.W, r.H - h}
}

func (r Rect) SplitBottom(h float64) (Rect, Rect) {
	h = clamp(h, 0, r.H)
	return Rect{r.X, r.Bottom() - h, r.W, h}, Rect{r.X, r.Y, r.W, r.H - h}
}

func (r Rect) SplitLeft(w float64) (Rect, Rect) {
	w = clamp(w, 0, r.W)
	return Rect{r.X, r.Y, w, r.H}, Rect{r.X + w, r.Y, r.W - w, r.H}
}

func (r Rect) SplitRight(w float64) (Rect, Rect) {
	w = clamp(w, 0, r.W)
	return Rect{r.Right() - w, r.Y, w, r.H}, Rect{r.X, r.Y, r.W - w, r.H}
}

// Rows divides a box into n bands with a gap between them.
func (r Rect) Rows(n int, gap float64) []Rect {
	if n <= 0 {
		return nil
	}
	h := (r.H - gap*float64(n-1)) / float64(n)
	out := make([]Rect, n)
	for i := range out {
		out[i] = Rect{r.X, r.Y + float64(i)*(h+gap), r.W, h}
	}
	return out
}

// Columns divides a box into n columns with a gap between them.
func (r Rect) Columns(n int, gap float64) []Rect {
	if n <= 0 {
		return nil
	}
	w := (r.W - gap*float64(n-1)) / float64(n)
	out := make([]Rect, n)
	for i := range out {
		out[i] = Rect{r.X + float64(i)*(w+gap), r.Y, w, r.H}
	}
	return out
}

// Fit centres a box of the given size inside this one. Used for glyphs, which
// are square and rarely the same size as the space they sit in.
func (r Rect) Fit(w, h float64) Rect {
	return Rect{r.X + (r.W-w)/2, r.Y + (r.H-h)/2, w, h}
}

// Move shifts a box without resizing it.
func (r Rect) Move(dx, dy float64) Rect { return Rect{r.X + dx, r.Y + dy, r.W, r.H} }

// Union is the smallest box containing both. An empty box contributes nothing,
// so a running union can start from the zero value.
func (r Rect) Union(o Rect) Rect {
	if r.Empty() {
		return o
	}
	if o.Empty() {
		return r
	}
	x, y := min(r.X, o.X), min(r.Y, o.Y)
	return Rect{x, y, max(r.Right(), o.Right()) - x, max(r.Bottom(), o.Bottom()) - y}
}

// ---------------------------------------------------------------------------
// text, measured and shaped
// ---------------------------------------------------------------------------

// FitSize shrinks a font size until the text fits the width, and gives up at
// the floor rather than shrinking to nothing.
//
// It is for the few places where the text is the point and the box is fixed —
// the number on a stat tile, a project name in a header. Everywhere else,
// clipping with an ellipsis is the honest answer, because a label that shrank
// to nine pixels is not more readable than one that says it ran out of room.
func FitSize(s string, width, size, floor float64) float64 {
	for size > floor {
		if TextWidth(s, size) <= width {
			return size
		}
		size -= 0.5
	}
	return floor
}

// Lines wraps text and clips the last line with an ellipsis if it overflows,
// so a note in a fixed box ends with "…" rather than simply stopping.
func Lines(s string, width, size float64, max int) []string {
	all := Wrap(s, width, size)
	if max <= 0 || len(all) <= max {
		return all
	}
	kept := all[:max]
	last := kept[max-1]
	if len(all) > max {
		last = Clip(last+" "+strings.Join(all[max:], " "), width, size)
	}
	kept[max-1] = last
	return kept
}

// Initials reduces a name to the one or two letters an avatar shows.
//
// A handle is one word and gets one letter; a person's name gets the first
// letter of the first and last words. Anything that is not a letter is skipped,
// so "@ann-marie" and "Ann Marie" both come out as expected.
func Initials(name string) string {
	fields := strings.FieldsFunc(name, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	if len(fields) == 0 {
		return ""
	}
	first := []rune(fields[0])
	out := strings.ToUpper(string(first[0]))
	if len(fields) > 1 {
		last := []rune(fields[len(fields)-1])
		out += strings.ToUpper(string(last[0]))
	}
	return out
}

// Plural picks the word for a count, so a label reads "1 task" and "2 tasks".
func Plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

// Percent is progress as a whole number, with nothing to divide by treated as
// nothing done rather than as an error nobody can act on.
func Percent(done, total int) int {
	if total <= 0 {
		return 0
	}
	return int(float64(done) / float64(total) * 100)
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
