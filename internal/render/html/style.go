package html

import (
	"fmt"
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/render/svg"
	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

// stylesheet writes the whole appearance and the whole interface.
//
// The filtering is CSS because a page that needs JavaScript to hide a column
// is a page that shows nothing when JavaScript is off — and because :has()
// turns "hide everything that is not ann's" into one line. Every control in
// the toolbar is a real radio or checkbox, so the keyboard and the screen
// reader get the behavior for free.
func stylesheet(t *tree.Tree) string {
	var b strings.Builder
	b.WriteString("<style>\n")

	fmt.Fprintf(&b, ":root{%s;--tip-bg:#222222;--tip-fg:#f0f0f0}\n", palette(svg.Light))
	fmt.Fprintf(&b, "@media (prefers-color-scheme:dark){:root{%s;--tip-bg:#30363d;--tip-fg:#e6edf3}}\n",
		palette(svg.Dark))

	b.WriteString(base)
	b.WriteString(rules(t))

	b.WriteString("</style>")
	return b.String()
}

// base is everything that does not depend on the plan.
const base = `
*{box-sizing:border-box}
body{margin:0 auto;padding:26px 20px 40px;max-width:1080px;background:var(--canvas);color:var(--text);
 font:14px/1.5 system-ui,-apple-system,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif}
h1{display:flex;align-items:center;gap:10px;margin:0 0 14px;font-size:26px}
h1 .wordmark{width:26px;height:26px}
svg{flex:none}
.status{width:18px;height:18px}
.tiny{width:13px;height:13px;vertical-align:-2px}
.bar{height:8px;border-radius:4px;background:var(--track);overflow:hidden;max-width:320px}
.bar span{display:block;height:100%;background:var(--st-done)}
.count{margin:6px 0 20px;color:var(--muted);font-size:12px}

.toolbar{display:flex;flex-wrap:wrap;gap:18px;margin-bottom:22px;padding-bottom:18px;
 border-bottom:1px solid var(--border)}
.toolbar fieldset{display:flex;flex-wrap:wrap;align-items:center;gap:6px;margin:0;padding:0;border:0}
.toolbar legend{padding:0 6px 0 0;color:var(--muted);font-size:11px;text-transform:uppercase;
 letter-spacing:.04em;float:left}
.toolbar input{position:absolute;width:1px;height:1px;opacity:0;pointer-events:none}
.toolbar label{display:inline-flex;align-items:center;gap:5px;padding:4px 10px;border-radius:999px;
 border:1px solid var(--border);background:var(--card);color:var(--muted);font-size:12px;cursor:pointer}
.toolbar input:checked+label{color:var(--text);border-color:var(--st-in-progress);
 box-shadow:inset 0 0 0 1px var(--st-in-progress)}
.toolbar input:focus-visible+label{outline:2px solid var(--st-in-progress);outline-offset:2px}

.card{display:flex;align-items:center;gap:10px;padding:9px 12px;margin:6px 0;border-radius:9px;
 border:1px solid var(--border);border-left:4px solid var(--border);background:var(--card);
 color:inherit;text-decoration:none;position:relative}
a.card:hover{border-color:var(--st-in-progress)}
.card .body{display:flex;flex-direction:column;min-width:0}
.card .title{font-weight:500}
.card .meta{color:var(--muted);font-size:11.5px;display:flex;align-items:center;gap:5px;flex-wrap:wrap}
.card .dot{color:var(--border)}
.card .ratio{margin-left:auto;color:var(--muted);font-size:11px;white-space:nowrap}
.card[data-status="dropped"] .title{text-decoration:line-through;color:var(--muted)}

.tree ul{list-style:none;margin:0;padding-left:24px;border-left:1px solid var(--edge)}
.tree>ul{padding-left:0;border-left:0}
.board{display:none;grid-template-columns:repeat(auto-fit,minmax(230px,1fr));gap:16px;align-items:start}
.column h2{display:flex;align-items:center;gap:6px;margin:0 0 4px;padding-bottom:7px;font-size:13px;
 border-bottom:2px solid currentColor}
.column .tally{margin-left:auto;color:var(--muted);font-weight:400}
.empty{color:var(--muted)}

footer{display:flex;flex-wrap:wrap;gap:14px;margin-top:28px;padding-top:16px;
 border-top:1px solid var(--border);color:var(--muted);font-size:11.5px}
footer span{display:inline-flex;align-items:center;gap:5px}
footer .by{margin-left:auto}

/* The tooltip: the wrap is the hover target, not the glyph, so the pointer can
   drift onto the bubble without it flickering away. The delay lives only in
   the hover rule, so leaving snaps it back to zero and the bubble goes at once. */
.tip-wrap{position:relative;display:inline-flex;margin-left:auto}
/* Anchored to the right rather than centred: the glyph sits at the end of a
   card that reaches the edge of the page, and a centred bubble would hang off
   it. Wrapping with a max width keeps a long note on screen too. */
.tip{position:absolute;bottom:calc(100% + 8px);right:-6px;transform:translateY(2px) scale(.98);
 transform-origin:100% 100%;padding:8px 12px;border-radius:8px;background:var(--tip-bg);color:var(--tip-fg);
 width:max-content;max-width:260px;font-size:12px;line-height:1.4;opacity:0;pointer-events:none;z-index:2;
 box-shadow:0 0 0 1px rgba(0,0,0,.06),0 2px 6px rgba(0,0,0,.05),0 4px 42px rgba(0,0,0,.06);
 transition:opacity 50ms ease-out,transform 50ms ease-out}
.tip-wrap:hover .tip,.tip-wrap:focus-within .tip{opacity:1;transform:none;
 transition-duration:150ms;transition-timing-function:ease-out;transition-delay:80ms}

@media (prefers-reduced-motion:reduce){.tip{transition:none}}
@media print{
 .toolbar{display:none}
 .board{display:none}
 .card{break-inside:avoid}
 body{max-width:none}
}
`

// rules are the parts that depend on this plan: the status colors, the view
// switch, and one filter rule per owner and per tag.
func rules(t *tree.Tree) string {
	var b strings.Builder

	for _, s := range tree.Statuses {
		fmt.Fprintf(&b, ".card[data-status=%q]{border-left-color:%s}\n", s, cssVar(s))
		fmt.Fprintf(&b, ".column[data-status=%q] h2{color:%s}\n", s, cssVar(s))
	}

	// Choosing an owner or a tag switches to the board on its own. Those are
	// questions about people and work in flight, and the tree answers them
	// badly: a milestone has no owner, so an owner filter would empty it out.
	const boardTriggers = `body:has(#view-board:checked),
body:has(.owners input:not(#owner-all):checked),
body:has(.tags input:not(#tag-all):checked)`
	for _, sel := range strings.Split(boardTriggers, ",\n") {
		fmt.Fprintf(&b, "%s .tree{display:none}\n", sel)
		fmt.Fprintf(&b, "%s .board{display:grid}\n", sel)
	}

	// When a filter forces the board, the toolbar has to agree with the page.
	// A Tree chip that still looks selected while the board is on screen is a
	// small lie, and small lies in an interface are the ones people trip on.
	for _, sel := range strings.Split(boardTriggers, ",\n")[1:] {
		fmt.Fprintf(&b, "%s label[for=view-board]{color:var(--text);"+
			"border-color:var(--st-in-progress);box-shadow:inset 0 0 0 1px var(--st-in-progress)}\n", sel)
		fmt.Fprintf(&b, "%s label[for=view-tree]{color:var(--muted);"+
			"border-color:var(--border);box-shadow:none}\n", sel)
	}

	// Hiding a status takes the whole branch in the tree and the whole column
	// on the board.
	for _, s := range tree.Statuses {
		id := "#hide-" + slug(string(s))
		fmt.Fprintf(&b, "body:has(%s:checked) .tree li:has(>.card[data-status=%q]){display:none}\n", id, s)
		fmt.Fprintf(&b, "body:has(%s:checked) .board .column[data-status=%q]{display:none}\n", id, s)
	}

	for _, owner := range values(t, func(n *tree.Node) []string {
		if n.Owner == "" {
			return nil
		}
		return []string{n.Owner}
	}) {
		value := strings.ToLower(owner)
		id := "#owner-" + slug(owner)
		fmt.Fprintf(&b, "body:has(%s:checked) .card:not([data-owner=%q]){display:none}\n", id, value)
		fmt.Fprintf(&b, "body:has(%s:checked) .column:not(:has(.card[data-owner=%q])){display:none}\n", id, value)
	}

	for _, tag := range values(t, func(n *tree.Node) []string { return n.Tags }) {
		value := strings.ToLower(tag)
		id := "#tag-" + slug(tag)
		fmt.Fprintf(&b, "body:has(%s:checked) .card:not([data-tags~=%q]){display:none}\n", id, value)
		fmt.Fprintf(&b, "body:has(%s:checked) .column:not(:has(.card[data-tags~=%q])){display:none}\n", id, value)
	}

	return b.String()
}
