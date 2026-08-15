package svg

import (
	"path"
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

// Layout is which drawing an output file asks for.
type Layout string

const (
	LayoutTree  Layout = "tree"
	LayoutBoard Layout = "board"
)

// LayoutForFilename reads the drawing out of the file name, the same way
// ThemeForFilename reads the palette out of it.
//
// Two spellings work, because both are things a person actually types:
//
//	docs/board.svg        the file is the board
//	docs/plan.board.svg   a qualifier before the extension
//
// Everything else is the tree. Keeping this in the name rather than in a flag
// means `outputs` still lists exactly what gets written, and `devtree render`
// needs no arguments to reproduce it.
func LayoutForFilename(name string) Layout {
	base := strings.ToLower(path.Base(filepathToSlash(name)))
	base = strings.TrimSuffix(base, ".svg")
	base = strings.TrimSuffix(base, "-dark")
	base = strings.TrimSuffix(base, "_dark")

	if base == "board" || strings.HasSuffix(base, ".board") {
		return LayoutBoard
	}
	return LayoutTree
}

// RenderFor draws a plan the way a given output file asks for it.
//
// One entry point for the CLI keeps the naming rules in this package, where
// the tests for them live, instead of spreading them across the command layer.
func RenderFor(t *tree.Tree, name string) string {
	th := ThemeForFilename(name)
	if LayoutForFilename(name) == LayoutBoard {
		return Board(t, th)
	}
	return Render(t, th)
}

// filepathToSlash normalizes a Windows path without importing path/filepath,
// so the naming rules behave identically wherever the plan was written.
func filepathToSlash(name string) string {
	return strings.ReplaceAll(name, `\`, "/")
}
