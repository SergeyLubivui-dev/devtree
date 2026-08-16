// Package svg draws a plan as a standalone SVG file.
//
// It exists because the Mermaid block has a ceiling. GitHub renders Mermaid
// with a strict sanitizer and a page-level CSP, so a diagram node cannot carry
// an image or a link no matter how it is written. A file we generate ourselves
// has no such ceiling: the glyphs are paths inside the document, the colors are
// ours, and the layout is ours.
//
// The output is deliberately self-contained — no external fonts, no images, no
// script, no CSS custom properties. GitHub serves repository SVGs with
// `default-src 'none'; sandbox`, and everything here survives that on purpose.
//
// Like the rest of internal/render, this package is pure: a tree goes in, a
// string comes out. The CLI decides where it lands.
package svg

import (
	"github.com/SergeyLubivui-dev/devtree/internal/draw/parts"
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

// Theme is the full palette of one rendering. Two files rather than one
// self-theming file is a deliberate choice: GitHub wraps a <picture> in its own
// component and switches it with the GitHub theme toggle, while a media query
// inside a single SVG can only follow the operating system. Two files track
// what the reader actually set.
type Theme struct {
	Name string

	Text   string // titles
	Muted  string // metadata, legend
	Canvas string // the panel behind everything
	Card   string // node fill
	Border string // node stroke
	Edge   string // connectors
	Track  string // empty part of a progress bar

	Todo       string
	InProgress string
	Blocked    string
	Done       string
	Dropped    string
}

// Light and Dark follow GitHub's own palettes, so a diagram embedded in a
// README looks like part of the page rather than a screenshot pasted into it.
var (
	Light = Theme{
		Name:   "light",
		Text:   "#1f2328",
		Muted:  "#59636e",
		Canvas: "#f6f8fa",
		Card:   "#ffffff",
		Border: "#d1d9e0",
		Edge:   "#d1d9e0",
		Track:  "#eaeef2",

		Todo:       "#59636e",
		InProgress: "#bf8700",
		Blocked:    "#cf222e",
		Done:       "#1a7f37",
		Dropped:    "#8c959f",
	}

	Dark = Theme{
		Name:   "dark",
		Text:   "#e6edf3",
		Muted:  "#9198a1",
		Canvas: "#0d1117",
		Card:   "#151b23",
		Border: "#3d444d",
		Edge:   "#3d444d",
		Track:  "#262c36",

		Todo:       "#9198a1",
		InProgress: "#d29922",
		Blocked:    "#f85149",
		Done:       "#3fb950",
		Dropped:    "#6e7681",
	}
)

// color returns the accent for a status.
func (t Theme) color(s tree.Status) string {
	switch s {
	case tree.InProgress:
		return t.InProgress
	case tree.Blocked:
		return t.Blocked
	case tree.Done:
		return t.Done
	case tree.Dropped:
		return t.Dropped
	default:
		return t.Todo
	}
}

// ThemeFor picks a palette by name. Unknown names fall back to light, because
// a misspelled theme should still produce a readable diagram.
func ThemeFor(name string) Theme {
	if strings.EqualFold(strings.TrimSpace(name), "dark") {
		return Dark
	}
	return Light
}

// ThemeForFilename reads the palette out of the output file name: anything
// ending in -dark.svg is the dark rendering.
//
// Encoding it in the name keeps the plan honest — `outputs` lists exactly the
// files that get written, with no hidden second file appearing next to them.
func ThemeForFilename(name string) Theme {
	// Lowercase before trimming, or an upper-case .SVG on Windows would leave
	// the extension attached and hide the -dark suffix behind it.
	base := strings.TrimSuffix(strings.ToLower(name), ".svg")
	if strings.HasSuffix(base, "-dark") || strings.HasSuffix(base, "_dark") {
		return Dark
	}
	return Light
}

// Palette hands a theme to the component library.
//
// The two are deliberately different types. A Theme knows the five task
// statuses, which is exactly the knowledge a drawing component must not have —
// a Pill that knew what "blocked" looks like could only ever draw plans. This
// is the one place the translation happens, so a new colour in the theme
// reaches every component without any of them learning what it means.
func (t Theme) Palette() parts.Palette {
	return parts.Palette{
		Text:   t.Text,
		Muted:  t.Muted,
		Faint:  t.Muted,
		Canvas: t.Canvas,
		Card:   t.Card,
		Track:  t.Track,
		Border: t.Border,
		Accent: t.Done,
	}
}
