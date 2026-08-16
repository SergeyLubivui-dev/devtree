package parts

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/SergeyLubivui-dev/devtree/internal/draw"
)

// test is a palette with a distinct colour per role, so a test can tell which
// one a component reached for.
var test = Palette{
	Text:   "#111111",
	Muted:  "#222222",
	Faint:  "#333333",
	Canvas: "#444444",
	Card:   "#555555",
	Track:  "#666666",
	Border: "#777777",
	Accent: "#00aa00",
}

// every draws each component once, into its own fragment, keyed by name. The
// map is what the table-driven tests below walk, and it is also the check that
// Names() and the package agree.
func every() map[string]string {
	box := draw.Rect{X: 10, Y: 10, W: 240, H: 60}
	out := map[string]string{}

	draw1 := func(name string, f func(b *strings.Builder)) {
		var b strings.Builder
		f(&b)
		out[name] = b.String()
	}

	draw1("Card", func(b *strings.Builder) {
		Card(b, box, test, CardStyle{
			Accent: "#cc0000", Title: "Authentication", Meta: "feat/auth · ann",
			Glyph: "clock-circle", Motion: draw.ClassSpin,
			Trailer: func(b *strings.Builder, r draw.Rect) { Badge(b, r.X, r.MidY()-8, test, "3/7", "") },
		})
	})
	draw1("Cluster", func(b *strings.Builder) { Cluster(b, box, test, "MVP") })
	draw1("Callout", func(b *strings.Builder) {
		Callout(b, box, test, "note", "The plan is a file. Everything here is drawn from it.", "")
	})
	draw1("Swimlane", func(b *strings.Builder) { Swimlane(b, box, test, "Backend", 70) })
	draw1("ColumnHead", func(b *strings.Builder) {
		ColumnHead(b, draw.Rect{X: 10, Y: 10, W: 200, H: 24}, test, "check-circle", "done", 22, "#1a7f37", "")
	})
	draw1("Stat", func(b *strings.Builder) { Stat(b, box, test, "23 / 25", "tasks done", "") })

	draw1("Chip", func(b *strings.Builder) { Chip(b, 10, 10, test, "hashtag", "backend", "") })
	draw1("Badge", func(b *strings.Builder) { Badge(b, 10, 10, test, "6/6", "") })
	draw1("Pill", func(b *strings.Builder) { Pill(b, 10, 10, test, "lock-circle", "blocked", "#cf222e", "") })
	draw1("Avatar", func(b *strings.Builder) { Avatar(b, 10, 10, 26, test, "Ann Marie", "") })
	draw1("Milestone", func(b *strings.Builder) { Milestone(b, 40, 30, 14, test, "v1.0", "") })
	draw1("Legend", func(b *strings.Builder) {
		Legend(b, 10, 10, test, []LegendItem{
			{Icon: "check-circle", Label: "done", Colour: "#1a7f37"},
			{Icon: "clock-circle", Label: "not started"},
		}, 16)
	})

	draw1("Bar", func(b *strings.Builder) {
		Bar(b, draw.Rect{X: 10, Y: 10, W: 200, H: 6}, test, 3, 4, "", draw.ClassGrow)
	})
	draw1("Ring", func(b *strings.Builder) { Ring(b, 40, 40, 18, 4, test, 3, 4, "") })
	draw1("Sparkline", func(b *strings.Builder) {
		Sparkline(b, box, test, []float64{1, 4, 2, 7, 6, 9}, "")
	})
	draw1("Timeline", func(b *strings.Builder) {
		Timeline(b, box, test, []Tick{{At: 0, Label: "v0.1"}, {At: 0.5, Label: "v0.2", Mark: "package"}, {At: 1, Label: "v1.0"}})
	})

	draw1("Divider", func(b *strings.Builder) { Divider(b, box, test, "later") })
	draw1("Connector", func(b *strings.Builder) {
		Connector(b, draw.Point{X: 10, Y: 10}, draw.Point{X: 200, Y: 80}, test, "", draw.ClassFlow)
	})
	draw1("Arrow", func(b *strings.Builder) {
		Arrow(b, draw.Point{X: 10, Y: 10}, draw.Point{X: 200, Y: 80}, test, "", "")
	})
	draw1("Brace", func(b *strings.Builder) { Brace(b, 30, 10, 90, test, "shipped together", 10) })

	return out
}

func TestEveryNamedComponentDraws(t *testing.T) {
	drawn := every()

	// Names() is what the gallery and the documentation walk. A component that
	// is not in it is one nobody will ever see.
	for _, name := range Names() {
		fragment, ok := drawn[name]
		if !ok {
			t.Errorf("%s is named but never drawn", name)
			continue
		}
		if strings.TrimSpace(fragment) == "" {
			t.Errorf("%s drew nothing", name)
		}
	}
	if len(drawn) != len(Names()) {
		t.Errorf("%d components drawn, %d named", len(drawn), len(Names()))
	}
	if len(Names()) != 20 {
		t.Errorf("the vocabulary is %d components", len(Names()))
	}
}

func TestEveryComponentIsWellFormed(t *testing.T) {
	// Each fragment is wrapped in a root element and parsed. A component that
	// leaves a group open takes the whole drawing down with it, and the failure
	// shows up as a blank image rather than an error.
	for name, fragment := range every() {
		doc := `<svg xmlns="http://www.w3.org/2000/svg">` + fragment + `</svg>`
		var root struct{ XMLName xml.Name }
		if err := xml.Unmarshal([]byte(doc), &root); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestComponentsAreDeterministic(t *testing.T) {
	// The same plan drawn twice has to produce the same bytes, or `render` is a
	// diff generator and the CI check that compares them is noise.
	first, second := every(), every()
	for name, fragment := range first {
		if second[name] != fragment {
			t.Errorf("%s drew differently the second time", name)
		}
	}
}

func TestComponentsUseThePaletteTheyWereGiven(t *testing.T) {
	// A component with a colour baked into it cannot be used on a dark page,
	// and this library exists partly so the dark drawings are not a second
	// implementation.
	for name, fragment := range every() {
		for _, hardcoded := range []string{"#fff", "#ffffff", "#000", "#000000", "white", "black"} {
			if strings.Contains(strings.ToLower(fragment), `"`+hardcoded+`"`) {
				t.Errorf("%s has %s baked into it", name, hardcoded)
			}
		}
	}
}

func TestAComponentGivenNoColourFallsBackToThePalette(t *testing.T) {
	var b strings.Builder
	Pill(&b, 0, 0, test, "", "todo", "", "")
	if !strings.Contains(b.String(), test.Accent) {
		t.Error("a pill with no colour should take the palette's accent")
	}

	var bare strings.Builder
	Chip(&bare, 0, 0, test, "", "backend", "")
	if !strings.Contains(bare.String(), test.Muted) {
		t.Error("a chip with no colour should be muted, not accented")
	}
}

func TestOptionalPartsAreOptional(t *testing.T) {
	// Every component takes an icon or a label it can do without. Leaving one
	// out has to cost nothing rather than leaving a hole where it would have
	// been, or callers end up with two variants of everything.
	var withIcon, without strings.Builder
	iconWidth := Chip(&withIcon, 0, 0, test, "user", "ann", "")
	plainWidth := Chip(&without, 0, 0, test, "", "ann", "")

	if iconWidth <= plainWidth {
		t.Errorf("an icon added %.0f to the chip", iconWidth-plainWidth)
	}
	if strings.Contains(without.String(), "<g transform") {
		t.Error("a chip with no icon drew one anyway")
	}

	var noLabel strings.Builder
	Cluster(&noLabel, draw.Rect{W: 100, H: 50}, test, "")
	if strings.Contains(noLabel.String(), "<text") {
		t.Error("a cluster with no label drew a label")
	}
}

func TestMetersHandleTheEdges(t *testing.T) {
	// An empty plan is the first thing anybody sees, and dividing by its zero
	// is the first way a drawing tool crashes.
	for _, c := range []struct{ done, total int }{{0, 0}, {0, 10}, {10, 10}, {5, 0}} {
		var b strings.Builder
		Bar(&b, draw.Rect{W: 100, H: 6}, test, c.done, c.total, "", "")
		Ring(&b, 20, 20, 10, 3, test, c.done, c.total, "")
		if !strings.Contains(b.String(), test.Track) {
			t.Errorf("%d/%d drew no track", c.done, c.total)
		}
	}

	// A sparkline needs two points to have a direction; one is not a trend.
	var thin strings.Builder
	Sparkline(&thin, draw.Rect{W: 100, H: 20}, test, []float64{4}, "")
	if strings.TrimSpace(thin.String()) != "" {
		t.Error("a single value drew a line")
	}

	// A flat series must not divide by its own zero range.
	var flat strings.Builder
	Sparkline(&flat, draw.Rect{W: 100, H: 20}, test, []float64{3, 3, 3}, "")
	if !strings.Contains(flat.String(), "<path") {
		t.Error("a flat series drew nothing")
	}
	if strings.Contains(flat.String(), "NaN") {
		t.Error("a flat series divided by zero")
	}
}

func TestTicksOutsideTheTimelineAreClamped(t *testing.T) {
	var b strings.Builder
	Timeline(&b, draw.Rect{X: 0, Y: 0, W: 100, H: 20}, test, []Tick{{At: -3}, {At: 9}})
	if strings.Contains(b.String(), "-") && strings.Contains(b.String(), "cx=\"-") {
		t.Error("a tick before the start was drawn off the canvas")
	}
}
