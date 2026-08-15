# SVG output

Mermaid has a ceiling. GitHub renders it with a strict sanitizer and a page-level Content Security
Policy, so a node in the diagram cannot carry an icon and cannot be a link, no matter how the label
is written. A file devtree draws itself has no such ceiling.

## The file name decides everything

There is no `--format` flag and no layout setting in the plan. `outputs` stays a list of
destinations, and `devtree render` needs no arguments to reproduce exactly what is checked in.

| Name | What gets drawn |
|---|---|
| `TREE.md`, `README.md`, anything `.md` | the Mermaid block, between markers |
| `docs/tree.svg` | the tree, light palette |
| `docs/tree-dark.svg` | the tree, dark palette |
| `docs/board.svg` | the board |
| `docs/plan.board-dark.svg` | the board, dark, under another name |

The rules, exactly: a `-dark` or `_dark` suffix before the extension picks the dark palette; a file
named `board` or ending in `.board` picks the board layout. `keyboard.svg` and `onboarding.svg` are
trees — the suffix has to be the whole name or a qualifier, not a coincidence.

## Two files, not one self-theming file

An SVG can theme itself with `@media (prefers-color-scheme: dark)` in an inline stylesheet, and that
does work through GitHub's delivery. It follows the reader's **operating system**, though, not their
GitHub theme.

Two files behind a `<picture>` follow the toggle the reader actually set, because GitHub wraps the
element in its own `themed-picture` component:

```html
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/tree-dark.svg">
  <img alt="Development tree" src="docs/tree.svg" width="800">
</picture>
```

## What the drawing carries

- Status glyphs on a 24×24 grid, vendored from [Reicon](https://reicon.dev) as path data.
- A stripe of the status color running the full height of each card, clipped to the rounded corners.
- A progress bar and a completion ratio on every parent.
- Branch, issue, pull request, owner, and tags on a second line under each title, each with its own
  small glyph. Items that will not fit are dropped whole rather than truncated into nonsense.
- Rounded elbow connectors that stay readable when a milestone grows a dozen children.
- A status key along the bottom, generated from the status list so it can never fall out of date.

## Motion that means something

| Class | What moves | What it says |
|---|---|---|
| `dt-flow` | a dash travels along an edge | that edge leads into work in progress |
| `dt-spin` | a glyph turns | that task is in flight right now |
| `dt-pulse` | a glyph breathes | that task is blocked |
| `dt-grow` | a bar fills, once | how much of a milestone is finished |
| `dt-rise` | a card fades up, staggered | it just arrived; the picture assembles |
| `dt-marquee` | a strip scrolls forever | used only in the documentation banner |

Everything else holds still. Three rules keep the motion from ever costing a reader anything:

- **`prefers-reduced-motion` switches it all off.** Every animated element is drawn over something
  that already reads correctly without it — the travelling dash sits on a connector that is already
  a solid line, the growing bar already has its final width.
- **Printing switches it all off too.** A renderer that froze the first frame of a fade-in would
  otherwise put a blank card on the page.
- **The entrance uses fill mode `backwards`**, so a card ends on its own base style rather than
  parked in an animation's final frame.

## Self-contained, because it has to be

GitHub serves a repository SVG from its own domain with this policy:

```
Content-Type: image/svg+xml
Content-Security-Policy: default-src 'none'; style-src 'unsafe-inline'; sandbox
```

An inline stylesheet is allowed; script is not; nothing external loads at all. So the output has no
web fonts (system stack only), no images, no scripts, and no external references of any kind. That
constraint was measured against the live headers before the renderer was written, not assumed.

The same sandbox is why an SVG rendered as an image cannot carry links. Branch and issue links live
in the collapsed table under the Mermaid block instead.

## Text without font metrics

Card widths come from an estimate — per-character-class averages — rather than real font metrics.
Shipping metrics would mean shipping a font, which is the one dependency this program is built to
avoid. Cards are sized a few pixels generously to compensate, and the widest card in a column gets a
little extra slack so it cannot clip the very title that set the column width.

## Where the pictures in the documentation come from

`docs/assets/` holds two kinds of generated file. The plan diagrams (`tree`, `board`) are written by
`devtree render` from this repository's own plan. The rest — the banner, the three reasons, the
pipeline, the status key, the architecture — come from `go run ./internal/art`, which uses the same
palette, glyph set, and drawing primitives. CI regenerates both and fails if a committed file
differs, so a picture in the documentation cannot quietly go stale.
