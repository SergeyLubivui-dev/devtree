package render

import (
	"fmt"
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

// classDefs colors the nodes with GitHub's own status palette, so the diagram
// looks native in both light and dark themes: the fills are light enough for
// the dark theme's white-ish canvas and the explicit `color:` keeps the label
// readable when GitHub inverts the page.
const classDefs = `    classDef todo fill:#f6f8fa,stroke:#8c959f,color:#1f2328
    classDef in_progress fill:#fff8c5,stroke:#bf8700,color:#1f2328
    classDef blocked fill:#ffebe9,stroke:#cf222e,color:#1f2328
    classDef done fill:#dafbe1,stroke:#1a7f37,color:#1f2328
    classDef dropped fill:#eaeef2,stroke:#8c959f,color:#6e7781,stroke-dasharray:4 3
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
