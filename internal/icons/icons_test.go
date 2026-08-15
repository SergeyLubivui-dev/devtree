package icons

import (
	"strings"
	"testing"
)

func TestEveryGlyphIsUsable(t *testing.T) {
	names := Names()
	if len(names) == 0 {
		t.Fatal("no glyphs are vendored")
	}

	for _, name := range names {
		body, ok := Get(name)
		if !ok {
			t.Errorf("%s: Names listed a glyph Get cannot find", name)
			continue
		}
		// Any bare shape element is fine — the set draws with <path> mostly,
		// but a plain circle is cheaper as <circle>.
		if !startsWithShape(body) {
			t.Errorf("%s: expected bare shape markup, got %.40q", name, body)
		}
		if !strings.Contains(body, "currentColor") {
			t.Errorf("%s: does not fill with currentColor, so the renderer cannot color it", name)
		}
		// A glyph that carries its own width, height, or viewBox would fight
		// the transform the renderer wraps it in.
		for _, forbidden := range []string{"<svg", "viewBox", "xmlns"} {
			if strings.Contains(body, forbidden) {
				t.Errorf("%s: contains %q — paths only, please", name, forbidden)
			}
		}
	}
}

func startsWithShape(body string) bool {
	for _, shape := range []string{"<path", "<circle", "<rect", "<ellipse", "<polygon", "<g"} {
		if strings.HasPrefix(body, shape) {
			return true
		}
	}
	return false
}

func TestUnknownGlyphIsReportedNotGuessed(t *testing.T) {
	if body, ok := Get("no-such-icon"); ok || body != "" {
		t.Fatalf("Get returned %q, ok=%v for an unknown name", body, ok)
	}
}

func TestGridSize(t *testing.T) {
	if GridSize != 24 {
		t.Fatalf("GridSize is %d; the vendored paths are drawn on a 24x24 grid", GridSize)
	}
}
