package render

import (
	"fmt"
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

// classDefs carries the status in the border and nothing else.
//
// One Mermaid block has to serve a light page and a dark one, and it cannot
// theme itself. Setting a fill and a label color — as this did — produces a
// readable diagram that glares on a dark page; setting a light label color
// instead makes it unreadable there, because Mermaid keeps a pale node fill in
// its dark theme. Both were tried against a real Mermaid build before this
// comment was written.
//
// So the fill and the text are left to whichever theme GitHub picked, and only
// the stroke is ours. The diagram then looks native on both pages, and the
// status still reads at a glance — from the border and from the glyph in the
// label.
const classDefs = `    classDef todo stroke:#8c959f,stroke-width:1.5px
    classDef in_progress stroke:#bf8700,stroke-width:2px
    classDef blocked stroke:#cf222e,stroke-width:2px
    classDef done stroke:#1a7f37,stroke-width:2px
    classDef dropped stroke:#8c959f,stroke-width:1.5px,stroke-dasharray:4 3
`

// Mermaid renders the diagram, fenced and ready to drop into Markdown.
func Mermaid(t *tree.Tree) string {
	var b strings.Builder

	b.WriteString("```mermaid\n")
	b.WriteString("flowchart TD\n")
	b.WriteString(classDefs)

	if len(t.Roots()) == 0 {
		// An empty diagram would render as a blank box with no explanation,
		// so say what to do instead.
		b.WriteString("    empty[\"Nothing planned yet — add a task: devtree add\"]:::todo\n")
		b.WriteString("```\n")
		return b.String()
	}

	names := mermaidNames(t)
	var walk func(n *tree.Node)
	walk = func(n *tree.Node) {
		fmt.Fprintf(&b, "    %s[\"%s\"]:::%s\n", names[n.ID], nodeLabel(t, n), n.Status)
		for _, c := range n.Children() {
			fmt.Fprintf(&b, "    %s --> %s\n", names[n.ID], names[c.ID])
			walk(c)
		}
	}
	for _, root := range t.Roots() {
		walk(root)
	}

	b.WriteString("```\n")
	return b.String()
}

// nodeLabel is the text inside a box: status glyph, title, and a subdued
// second line with whatever context exists — progress, branch, issue, owner.
func nodeLabel(t *tree.Tree, n *tree.Node) string {
	title := escapeMermaid(n.Title)
	if title == "" {
		title = escapeMermaid(n.ID)
	}
	label := n.Status.Icon() + " " + title

	var meta []string
	if len(n.Children()) > 0 {
		done, total := t.Progress(n)
		meta = append(meta, fmt.Sprintf("%d/%d", done, total))
	}
	if n.Branch != "" {
		meta = append(meta, escapeMermaid(n.Branch))
	}
	if n.Issue != "" {
		meta = append(meta, escapeMermaid("#"+n.Issue))
	}
	if n.Owner != "" {
		meta = append(meta, escapeMermaid("@"+n.Owner))
	}
	if len(meta) > 0 {
		label += "<br/><i>" + strings.Join(meta, " · ") + "</i>"
	}
	return label
}

// escapeMermaid replaces the characters that would end a quoted label early or
// be read as diagram syntax. Mermaid accepts HTML entities inside labels, and
// its entity form uses "#" instead of "&", which is why "#" itself has to be
// escaped first — the replacer does a single left-to-right pass, so the
// substitutions it introduces are never rescanned.
func escapeMermaid(s string) string {
	return strings.NewReplacer(
		"#", "#35;",
		`"`, "#quot;",
		"<", "#lt;",
		">", "#gt;",
		"{", "#123;",
		"}", "#125;",
		"|", "#124;",
	).Replace(s)
}

// mermaidNames maps node IDs to Mermaid-safe identifiers.
//
// Mermaid identifiers may not contain dots or dashes, so those collapse to
// underscores — which means "a.b" and "a-b" would land on the same name and
// silently merge into one box. Collisions therefore get a numeric suffix.
func mermaidNames(t *tree.Tree) map[string]string {
	names := make(map[string]string, len(t.Nodes))
	used := make(map[string]bool, len(t.Nodes))

	for _, n := range t.Nodes {
		var b strings.Builder
		b.WriteString("n_")
		for _, r := range n.ID {
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
				b.WriteRune(r)
			} else {
				b.WriteByte('_')
			}
		}
		base := b.String()
		candidate := base
		for i := 2; used[candidate]; i++ {
			candidate = fmt.Sprintf("%s%d", base, i)
		}
		used[candidate] = true
		names[n.ID] = candidate
	}
	return names
}
