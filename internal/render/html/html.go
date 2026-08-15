// Package html exports a plan as a page you can click.
//
// It exists because of one hard limit the other two backends share: GitHub's
// Mermaid ignores click directives, and an SVG served as an image runs in a
// sandbox. Neither can carry a link to the issue a task closes. A page can —
// and once it is a page, it can also filter itself and show a note on hover.
//
// The output is one self-contained file: no scripts, no fonts, no images, no
// stylesheet to fetch. It works from a file:// URL, from GitHub Pages, and
// from inside a zip attached to an email. The filtering is plain CSS built on
// :has(), because a page that needs JavaScript to hide a column is a page that
// shows nothing when JavaScript is off.
package html

import (
	"fmt"
	"sort"
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/draw"
	"github.com/SergeyLubivui-dev/devtree/internal/icons"
	"github.com/SergeyLubivui-dev/devtree/internal/render"
	"github.com/SergeyLubivui-dev/devtree/internal/render/svg"
	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

// statusIcon mirrors the drawing: five circles, outlined while the work moves,
// filled once it has settled.
var statusIcon = map[tree.Status]string{
	tree.Todo:       "clock-circle",
	tree.InProgress: "circle-half-dotted-check",
	tree.Blocked:    "lock-circle",
	tree.Done:       "check-circle",
	tree.Dropped:    "close-circle",
}

// cssVar is the custom property a status paints with.
func cssVar(s tree.Status) string {
	return "var(--st-" + strings.ReplaceAll(string(s), "_", "-") + ")"
}

// Render turns a plan into a complete HTML document.
func Render(t *tree.Tree) string {
	var b strings.Builder

	title := t.Project
	if title == "" {
		title = "Development tree"
	}

	b.WriteString("<!doctype html>\n<html lang=\"en\">\n<head>\n")
	b.WriteString(`<meta charset="utf-8">`)
	b.WriteString(`<meta name="viewport" content="width=device-width,initial-scale=1">`)
	fmt.Fprintf(&b, "<title>%s</title>\n", esc(title))
	b.WriteString(stylesheet(t))
	b.WriteString("\n</head>\n<body>\n")

	header(&b, t, title)
	toolbar(&b, t)

	b.WriteString(`<main>`)
	if len(t.Nodes) == 0 {
		b.WriteString(`<p class="empty">Nothing planned yet — add a task with <code>devtree add</code>.</p>`)
	} else {
		treeView(&b, t)
		boardView(&b, t)
	}
	b.WriteString(`</main>`)

	legend(&b)
	b.WriteString("\n</body>\n</html>\n")
	return b.String()
}

// header carries the project name and how far along it is.
func header(b *strings.Builder, t *tree.Tree, title string) {
	done, total := t.Totals()
	percent := 0.0
	if total > 0 {
		percent = float64(done) / float64(total) * 100
	}

	b.WriteString(`<header>`)
	b.WriteString(`<h1>` + glyph("tree", "wordmark", "var(--st-done)") + esc(title) + `</h1>`)
	fmt.Fprintf(b, `<div class="bar"><span style="width:%.1f%%"></span></div>`, percent)
	fmt.Fprintf(b, `<p class="count">%d / %d tasks done</p>`, done, total)
	b.WriteString(`</header>`)
}

// toolbar is the whole interface: which view, which statuses, whose work, and
// which tag. Every control is a radio or a checkbox, and every rule that acts
// on them is CSS — see the note on the stylesheet.
func toolbar(b *strings.Builder, t *tree.Tree) {
	b.WriteString(`<form class="toolbar" onsubmit="return false">`)

	b.WriteString(`<fieldset class="views"><legend>View</legend>`)
	b.WriteString(`<input type="radio" name="view" id="view-tree" checked><label for="view-tree">Tree</label>`)
	b.WriteString(`<input type="radio" name="view" id="view-board"><label for="view-board">Board</label>`)
	b.WriteString(`</fieldset>`)

	b.WriteString(`<fieldset class="statuses"><legend>Hide</legend>`)
	for _, s := range tree.Statuses {
		id := "hide-" + slug(string(s))
		fmt.Fprintf(b, `<input type="checkbox" id="%s"><label for="%s">%s%s</label>`,
			id, id, glyph(statusIcon[s], "tiny", cssVar(s)), esc(s.Label()))
	}
	b.WriteString(`</fieldset>`)

	if owners := values(t, func(n *tree.Node) []string {
		if n.Owner == "" {
			return nil
		}
		return []string{n.Owner}
	}); len(owners) > 0 {
		radioSet(b, "owner", "Owner", owners)
	}

	if tags := values(t, func(n *tree.Node) []string { return n.Tags }); len(tags) > 0 {
		radioSet(b, "tag", "Tag", tags)
	}

	b.WriteString(`</form>`)
}

// radioSet writes one exclusive group of filters.
//
// Radios rather than checkboxes, and that is a design decision rather than a
// shortcut: two checked owner boxes would mean "show ann's work or bob's", and
// CSS can express "hide everything that is not ann's" but not the union of two
// such rules. One at a time is honest; a filter that silently shows nothing is
// not.
func radioSet(b *strings.Builder, name, label string, options []string) {
	fmt.Fprintf(b, `<fieldset class="%ss"><legend>%s</legend>`, name, esc(label))
	fmt.Fprintf(b, `<input type="radio" name="%s" id="%s-all" checked><label for="%s-all">All</label>`,
		name, name, name)
	for _, value := range options {
		id := name + "-" + slug(value)
		fmt.Fprintf(b, `<input type="radio" name="%s" id="%s"><label for="%s">%s</label>`,
			name, id, id, esc(value))
	}
	b.WriteString(`</fieldset>`)
}

// treeView is the plan as nested lists, which is what a tree is in HTML.
func treeView(b *strings.Builder, t *tree.Tree) {
	b.WriteString(`<section class="tree" aria-label="Tree">`)
	var walk func(nodes []*tree.Node)
	walk = func(nodes []*tree.Node) {
		b.WriteString(`<ul>`)
		for _, n := range nodes {
			b.WriteString(`<li>`)
			card(b, t, n, true)
			if kids := n.Children(); len(kids) > 0 {
				walk(kids)
			}
			b.WriteString(`</li>`)
		}
		b.WriteString(`</ul>`)
	}
	walk(t.Roots())
	b.WriteString(`</section>`)
}

// boardView is the same work in columns. Only leaves appear, for the reason
// the terminal board gives: a milestone is a container, not a card.
func boardView(b *strings.Builder, t *tree.Tree) {
	columns := map[tree.Status][]*tree.Node{}
	for _, n := range t.Nodes {
		if len(n.Children()) == 0 {
			columns[n.Status] = append(columns[n.Status], n)
		}
	}

	b.WriteString(`<section class="board" aria-label="Board">`)
	for _, s := range tree.Statuses {
		nodes := columns[s]
		if len(nodes) == 0 {
			continue
		}
		fmt.Fprintf(b, `<div class="column" data-status="%s">`, s)
		fmt.Fprintf(b, `<h2>%s%s<span class="tally">%d</span></h2>`,
			glyph(statusIcon[s], "tiny", cssVar(s)), esc(s.Label()), len(nodes))
		for _, n := range nodes {
			card(b, t, n, false)
		}
		b.WriteString(`</div>`)
	}
	b.WriteString(`</section>`)
}

// card is one task. It becomes a link when the task points at something, which
// is the entire reason this backend exists.
func card(b *strings.Builder, t *tree.Tree, n *tree.Node, showProgress bool) {
	url, _ := render.URL(t, n, "")

	tag, close := `<div`, `</div>`
	if url != "" {
		tag, close = `<a href="`+esc(url)+`" target="_blank" rel="noopener"`, `</a>`
	}

	fmt.Fprintf(b, `%s class="card" data-status="%s" data-owner="%s" data-tags="%s">`,
		tag, n.Status, esc(strings.ToLower(n.Owner)), esc(strings.ToLower(strings.Join(n.Tags, " "))))

	b.WriteString(glyph(statusIcon[n.Status], "status", cssVar(n.Status)))
	b.WriteString(`<div class="body">`)

	title := n.Title
	if title == "" {
		title = n.ID
	}
	fmt.Fprintf(b, `<span class="title">%s</span>`, esc(title))

	if meta := metaOf(n); meta != "" {
		fmt.Fprintf(b, `<span class="meta">%s</span>`, meta)
	}
	b.WriteString(`</div>`)

	if showProgress {
		if done, total := t.Progress(n); total > 0 {
			fmt.Fprintf(b, `<span class="ratio">%d/%d</span>`, done, total)
		}
	}

	// The note rides along as a tooltip rather than a line of its own: it is
	// the answer to "why is this blocked", and that question is only asked
	// about one card at a time.
	if n.Note != "" {
		fmt.Fprintf(b, `<span class="tip-wrap">%s<span class="tip" role="tooltip">%s</span></span>`,
			glyph("note", "tiny", "var(--muted)"), esc(n.Note))
	}

	b.WriteString(close)
}

// metaOf is the second line: branch, issue, pull request, owner, tags.
func metaOf(n *tree.Node) string {
	var parts []string
	add := func(icon, text string) {
		parts = append(parts, glyph(icon, "tiny", "var(--muted)")+esc(text))
	}
	if n.Branch != "" {
		add("nodes", n.Branch)
	}
	if n.Issue != "" {
		add("hashtag", n.Issue)
	}
	if n.PR != "" {
		add("code-circle", n.PR)
	}
	if n.Owner != "" {
		add("user", n.Owner)
	}
	if len(n.Tags) > 0 {
		add("bookmark", strings.Join(n.Tags, ", "))
	}
	return strings.Join(parts, `<span class="dot">·</span>`)
}

func legend(b *strings.Builder) {
	b.WriteString(`<footer>`)
	for _, s := range tree.Statuses {
		fmt.Fprintf(b, `<span>%s%s</span>`, glyph(statusIcon[s], "tiny", cssVar(s)), esc(s.Label()))
	}
	b.WriteString(`<span class="by">generated by devtree</span></footer>`)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// glyph inlines a vendored path as a standalone svg element.
func glyph(name, class, color string) string {
	body, ok := icons.Get(name)
	if !ok {
		return ""
	}
	return fmt.Sprintf(`<svg class="%s" viewBox="0 0 24 24" aria-hidden="true" style="color:%s">%s</svg>`,
		class, color, body)
}

// values collects every distinct owner or tag in the plan, sorted, so the
// toolbar is the same on every render.
func values(t *tree.Tree, of func(*tree.Node) []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, n := range t.Nodes {
		for _, v := range of(n) {
			if v = strings.TrimSpace(v); v != "" && !seen[strings.ToLower(v)] {
				seen[strings.ToLower(v)] = true
				out = append(out, v)
			}
		}
	}
	sort.Strings(out)
	return out
}

// slug makes a value usable as an element id.
func slug(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return b.String()
}

// esc escapes text for HTML. The same rules as XML, and the same function, so
// the two backends cannot disagree about what is dangerous.
func esc(s string) string { return draw.Escape(s) }

// palette writes one theme's custom properties.
func palette(th svg.Theme) string {
	return fmt.Sprintf(
		"--text:%s;--muted:%s;--canvas:%s;--card:%s;--border:%s;--edge:%s;--track:%s;"+
			"--st-todo:%s;--st-in-progress:%s;--st-blocked:%s;--st-done:%s;--st-dropped:%s",
		th.Text, th.Muted, th.Canvas, th.Card, th.Border, th.Edge, th.Track,
		th.Todo, th.InProgress, th.Blocked, th.Done, th.Dropped)
}
