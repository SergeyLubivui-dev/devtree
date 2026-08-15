package draw

import (
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
