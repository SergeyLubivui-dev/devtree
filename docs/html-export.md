# The HTML export

Name an output `.html` and devtree writes a page instead of a diagram:

```bash
devtree init --outputs "README.md, docs/plan.html"
```

This repository's own is [docs/plan.html](plan.html) — open it from a checkout, or publish it and
send someone a link.

## Why there is a third backend

The other two share one hard limit. GitHub's Mermaid ignores `click` directives, and an SVG served
as an image runs in a sandbox: **neither can carry a link**. A plan that records the issue a task
closes and the branch it lives on cannot take you to either of them.

A page can. And once it is a page, two more things become free: it can filter itself, and it can
show a note on hover instead of spending a line on it.

## What is on it

- **The tree**, as nested lists, with every task that points somewhere rendered as a link to its
  pull request, its issue, or its branch — the most specific one it has.
- **The board**, the same work in columns by status, leaves only.
- **A toolbar**: which view, which statuses to hide, whose work, which tag.
- **Notes as tooltips**, so "why is this blocked" is one hover away rather than a line on every card.
- **Both palettes**, following `prefers-color-scheme`.

## No script, and no dependency

The whole interface is CSS. Every control is a real radio button or checkbox, and the rules that act
on them are `:has()` selectors:

```css
body:has(#owner-ann:checked) .card:not([data-owner="ann"]) { display: none }
body:has(#owner-ann:checked) .column:not(:has(.card[data-owner="ann"])) { display: none }
```

The second rule is the one worth pointing at: it hides a column that contains no card belonging to
ann. `:has()` asks about the document, not about what is visible, so an emptied column disappears
instead of leaving a heading over nothing.

Two consequences of choosing CSS over a script:

- The page works with JavaScript switched off, in a preview pane, and inside an email client that
  strips scripts. A page that needs a script to hide a column shows nothing without one.
- Owner and tag are **radio buttons, one at a time**. Two checked owner boxes would mean "ann's work
  or bob's", and CSS can express "hide everything that is not ann's" but not the union of two such
  rules. One at a time is honest; a filter that silently shows nothing is not.

Choosing an owner or a tag switches to the board on its own, because those are questions about
people and work in flight — and a milestone has no owner, so an owner filter would empty the tree.

## Publishing it

Anywhere that serves a file will do; there is nothing to build. On GitHub Pages, point the site at
the branch and the `docs/` folder in **Settings → Pages**, and the page is live at
`https://<user>.github.io/<repo>/plan.html`. It refreshes on every push, because `devtree render`
already runs there.

For an internal wiki or a shared drive, copy the one file. It has no siblings: no stylesheet, no
fonts, no images, no scripts.

## What it is not

It is a view, not an editor. Nothing on the page writes anything back — the plan file remains the
only place state lives, and `devtree set` remains the way to change it. That is deliberate: a page
that could edit the plan would need a server, an auth story, and a conflict story, and the whole
point of devtree is that it needs none of the three.
