package draw

import (
	"strings"
	"testing"
)

func TestSplittingABoxLosesNothing(t *testing.T) {
	// The whole point of splitting rather than doing the arithmetic by hand is
	// that the pieces still add up to what you started with.
	r := Rect{X: 10, Y: 20, W: 300, H: 80}

	head, body := r.SplitTop(30)
	if head.H+body.H != r.H {
		t.Errorf("%.0f + %.0f is not %.0f", head.H, body.H, r.H)
	}
	if head.Y != r.Y || body.Y != r.Y+30 || body.Bottom() != r.Bottom() {
		t.Errorf("the pieces are in the wrong places: %+v %+v", head, body)
	}

	left, rest := r.SplitLeft(120)
	if left.W+rest.W != r.W || left.X != r.X || rest.Right() != r.Right() {
		t.Errorf("a horizontal split does not add up: %+v %+v", left, rest)
	}

	right, remainder := r.SplitRight(50)
	if right.Right() != r.Right() || remainder.X != r.X {
		t.Errorf("splitting off the right moved the wrong edge: %+v %+v", right, remainder)
	}

	// Asking for more than there is takes everything rather than producing a
	// negative box that draws inside out.
	all, nothing := r.SplitTop(500)
	if all.H != r.H || nothing.H != 0 {
		t.Errorf("an oversized split gave %+v and %+v", all, nothing)
	}
}

func TestRowsAndColumnsFillTheBox(t *testing.T) {
	r := Rect{X: 0, Y: 0, W: 320, H: 200}

	cols := r.Columns(4, 10)
	if len(cols) != 4 {
		t.Fatalf("%d columns", len(cols))
	}
	if cols[0].X != 0 {
		t.Errorf("the first column starts at %.1f", cols[0].X)
	}
	if got := cols[3].Right(); !closeTo(got, r.Right()) {
		t.Errorf("the last column ends at %.4f, not %.4f", got, r.Right())
	}
	if gap := cols[1].X - cols[0].Right(); !closeTo(gap, 10) {
		t.Errorf("the gap is %.4f", gap)
	}

	rows := r.Rows(3, 8)
	if got := rows[2].Bottom(); !closeTo(got, r.Bottom()) {
		t.Errorf("the last row ends at %.4f, not %.4f", got, r.Bottom())
	}
	if r.Rows(0, 4) != nil || r.Columns(-1, 4) != nil {
		t.Error("asking for no rows should give none, not a panic")
	}
}

func TestUnionStartsFromNothing(t *testing.T) {
	// A running union has to be able to start from the zero value, or every
	// caller needs a special case for its first item.
	var box Rect
	for _, r := range []Rect{{10, 10, 20, 20}, {50, 5, 10, 40}} {
		box = box.Union(r)
	}
	want := Rect{10, 5, 50, 40}
	if box != want {
		t.Errorf("union is %+v, want %+v", box, want)
	}
}

func TestFitCentresWithoutResizing(t *testing.T) {
	spot := Rect{X: 0, Y: 0, W: 40, H: 40}.Fit(10, 10)
	if spot.X != 15 || spot.Y != 15 || spot.W != 10 {
		t.Errorf("a 10x10 glyph in a 40x40 box landed at %+v", spot)
	}
}

func TestFitSizeShrinksButGivesUp(t *testing.T) {
	long := "a rather long value indeed"

	if got := FitSize("42", 200, 26, 13); got != 26 {
		t.Errorf("something that already fits was shrunk to %.1f", got)
	}
	if got := FitSize(long, 40, 26, 13); got != 13 {
		t.Errorf("shrinking stopped at %.1f rather than the floor", got)
	}
	if got := FitSize(long, 400, 26, 13); got > 26 {
		t.Errorf("fitting grew the text to %.1f", got)
	}
}

func TestLinesEndWithAnEllipsisRatherThanStopping(t *testing.T) {
	text := "waiting on the payment provider to confirm the sandbox account before this can be tested"
	lines := Lines(text, 120, 11, 2)

	if len(lines) != 2 {
		t.Fatalf("%d lines, want 2", len(lines))
	}
	if !strings.HasSuffix(lines[1], "…") {
		t.Errorf("the last line just stops: %q", lines[1])
	}
	if got := TextWidth(lines[1], 11); got > 120 {
		t.Errorf("the clipped line is %.0f wide, over the 120 it was given", got)
	}

	// Text that fits is left alone entirely.
	short := Lines("short enough", 200, 11, 4)
	if len(short) != 1 || strings.Contains(short[0], "…") {
		t.Errorf("short text came back as %q", short)
	}
}

func TestInitialsReadNamesAndHandles(t *testing.T) {
	cases := map[string]string{
		"Ann Marie Smith": "AS",
		"ann":             "A",
		"@bob":            "B",
		"ann-marie":       "AM",
		"":                "",
		"...":             "",
		"Юлия Петрова":    "ЮП",
	}
	for in, want := range cases {
		if got := Initials(in); got != want {
			t.Errorf("Initials(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPluralAndPercent(t *testing.T) {
	if got := Plural(1, "task", "tasks"); got != "1 task" {
		t.Errorf("one task reads %q", got)
	}
	if got := Plural(0, "task", "tasks"); got != "0 tasks" {
		t.Errorf("no tasks reads %q", got)
	}
	if got := Percent(3, 4); got != 75 {
		t.Errorf("3 of 4 is %d%%", got)
	}
	// Nothing to divide by is nothing done, not a panic and not a hundred.
	if got := Percent(0, 0); got != 0 {
		t.Errorf("an empty plan is %d%% done", got)
	}
}

// closeTo compares positions the way a drawing cares about them. Dividing a
// box into three leaves a remainder that binary floating point cannot express,
// and a diagram is not wrong by a ten-thousandth of a pixel.
func closeTo(a, b float64) bool {
	d := a - b
	return d < 0.001 && d > -0.001
}
