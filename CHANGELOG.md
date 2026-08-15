# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[semantic versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `devtree open ID` opens what a task points at — its pull request, its issue, or its branch, in
  that order of specificity, or whichever one `--issue`, `--pr`, or `--branch` names. `--print`
  writes the URL instead, and a machine with no browser gets the URL rather than an error.
- Filters shared by both views: `ls` and `board` take `-s STATUS`, `-o OWNER`, `--tag a,b`, and
  `--root ID`. They combine with and, except tags, which combine with or. Owners and tags ignore
  case.
- `--root` on `render`, so one branch of a plan can have a picture of its own:
  `devtree render --root mvp --file docs/mvp.svg`. It works by slicing the plan into a smaller plan,
  which is why every renderer, rollup, and filter supports it without knowing it exists.
- An HTML export. Name an output `.html` and devtree writes one self-contained page where every task
  that points somewhere is a link to its pull request, issue, or branch — the one thing neither
  diagram backend can do, because GitHub's Mermaid ignores `click` and an SVG served as an image is
  sandboxed. The page carries a tree view, a board view, filters by status, owner and tag, and notes
  as tooltips. No script, no fonts, no images: the whole interface is CSS built on `:has()`.
- `.editorconfig`, a security policy, issue forms, and a pull request template, all under `.github/`
  where GitHub looks for them.

### Changed

- The repository root holds only what belongs there. The four translated READMEs moved to
  `docs/i18n/`, leaving one README at the top, and every relative link inside them was rewritten and
  checked.
- The README is the tour again: the deep sections it had grown — the full file format, the hook line
  by line, SVG naming and motion — now live in `docs/`, which is where they belong and where they
  are not duplicated.
- CONTRIBUTING is organised around scenarios rather than rules: a typo, a bug, a new field, a new
  command, a change to a drawing, a translation.

### Fixed

- The architecture picture was missing `internal/vcs`, which arrived with `devtree sync`.

## [0.3.0] — 2026-08-15

Finished work can leave the plan, git can close tasks on its own, the pictures move where movement
means something, and none of it needs anything installed.

### Added

- `devtree archive`, `devtree archive --all`, `devtree archive ID`, `devtree archive --list`, and
  `devtree restore ID` — finished branches of a plan move to `.devtree/archive.yaml` and come back
  from it. A node qualifies only when its whole subtree is done or dropped, so live work can never
  leave with the milestone above it, and plain `devtree archive` reports without moving anything.
  The archive uses the same format as the plan, so there is no second parser.
- `devtree sync` closes tasks whose branch git has already merged. It lists the changes and only
  writes them with `--apply`, because git knows which branches were merged but not which were merged
  *finished*. `internal/vcs` is the only package that runs another program, and it is reached only
  from this command.
- A container image on GHCR, `ghcr.io/sergeylubivui-dev/devtree`, for `amd64` and `arm64`: `latest`
  follows the release, `edge` follows `main`. The Dockerfile cross-compiles rather than emulating,
  and is built on Alpine rather than scratch so that `devtree sync` has a git to read.
- Two more things that move, both carrying information rather than decoration: a glyph breathes when
  a task is blocked, and cards fade up in sequence on load. The documentation banner carries an
  endless strip of cards — two copies of the strip slid by half their own width, so the loop has no
  seam.
- A documentation folder with eight pages: getting started, the file format, the board, finished
  work, automation, SVG output, the container, and the architecture. Generated pictures moved to
  `docs/assets/`.

### Fixed

- Animations no longer risk printing a blank card: the entrance uses fill mode `backwards` so an
  element ends on its own style, and printing switches every animation off, for renderers that
  freeze the first frame of a fade-in.

## [0.2.0] — 2026-08-15

devtree draws its own pictures now, and knows how to lay work out as a board.

### Added

- A native SVG renderer. Name an output `.svg` and devtree draws the plan itself instead of emitting
  a Mermaid block: status glyphs, a progress bar and ratio on every parent, branch / issue / pull
  request / owner / tags under each title, a status key, and rounded elbow connectors. The output
  format follows the file extension, so `outputs` stays a list of destinations rather than a list of
  settings.
- Light and dark palettes, chosen by file name: an output ending in `-dark.svg` renders dark. Point
  a `<picture>` at the pair and GitHub switches it with the reader's theme.
- `internal/icons`, holding twenty-one glyph paths vendored from Reicon (MIT, see NOTICE). No
  network access at render time and no new dependency — the binary is still self-contained.
- `internal/draw`, the primitives both renderers share: text measurement, wrapping, clipping,
  escaping, and glyph placement. The tree diagram and the documentation artwork are drawn by the
  same code, so they cannot drift into different visual languages.
- The README's own diagrams — the hero, the three reasons, the pipeline, the status key, and the
  architecture — are generated by `go run ./internal/art` from that same palette and glyph set. CI
  redraws them and fails if the committed files differ.
- Motion in the SVG output, carrying information rather than decoration: a dash travels down the
  edges that lead into work in progress, the glyph on an in-progress task turns, and progress bars
  grow in once on load. It is declarative CSS inside the file — the only kind of animation that
  survives GitHub's `default-src 'none'; sandbox` — and it stops entirely for readers who ask their
  system for reduced motion, leaving a picture that still reads correctly.

- A board view, in both places it is useful: `devtree board` groups the work by status in the
  terminal, and an output named `board.svg` (or `anything.board.svg`) draws the same thing as
  columns. Only leaves appear — a milestone is a container, not a card — and each task carries its
  milestone as a breadcrumb. Empty columns are left out.
- A recipes section in the README: starting a feature, breaking work down, picking up a ticket,
  parking something blocked, and the three commands worth running on a Monday.
- Translations of the README into Russian, Chinese, German, and French, reachable from a language
  bar at the top of each one. Every example in them was run before it was written down — the
  Russian quickstart shows the IDs transliteration actually produces, not plausible-looking ones.

### Removed

- The throwaway `SPIKE.md` and `spike/` files, which proved that GitHub renders a repository SVG
  with a themed `<picture>` before the renderer was built. The real drawing in `docs/` replaced them.

### Changed

- Cards now carry the status color as a stripe down the full height of their left edge, clipped to
  the rounded corners, instead of a short inset bar.
- In the generated documentation artwork, a glyph and its heading share a line instead of stacking,
  so the eye reads them as one label.

### Fixed

- Markdown injection no longer treats prose that *mentions* the `devtree:begin` / `devtree:end`
  markers as the region to replace. Only markers that start their own line count, so a README that
  documents devtree can safely be a devtree output itself.

## [0.1.0] — 2026-08-15

First public release.

### Added

- `.devtree/tree.yaml` as the source of truth: a flat, merge-friendly list of nodes in a strict
  subset of YAML, parsed by hand so errors can name the line they are on.
- Mermaid diagram generation into any Markdown file, written between `<!-- devtree:begin -->` and
  `<!-- devtree:end -->` markers so an existing README is never disturbed. Re-rendering an unchanged
  plan rewrites nothing.
- A collapsed link table under the diagram for branches, issues, and pull requests, built from the
  configured `repo` URL or from relative links when there is none.
- Commands: `init`, `add`, `set`, `done`, `mv`, `rm`, `ls`, `render`, `check`, `install`, `outputs`.
- Five statuses — `todo`, `in_progress`, `blocked`, `done`, `dropped` — with shorthand spellings
  accepted anywhere a status is.
- Progress rollups per node and for the whole plan, counting neither the node itself nor dropped
  work.
- IDs derived from titles, with Cyrillic transliterated so they stay typeable, and numeric suffixes
  on collision.
- A pre-commit hook and a GitHub Actions workflow, installable with `devtree install`, plus a
  `merge=union` rule written into `.gitattributes` by `init`.
- Validation covering duplicate IDs, dangling parents, cycles, unknown fields, and unknown statuses,
  with `check --strict` turning advisory warnings into a failure for CI.

[Unreleased]: https://github.com/SergeyLubivui-dev/devtree/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/SergeyLubivui-dev/devtree/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/SergeyLubivui-dev/devtree/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/SergeyLubivui-dev/devtree/releases/tag/v0.1.0
