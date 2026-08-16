# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[semantic versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- A card can carry described lines under its title. A card says what the work is; a bullet says what
  a reader would otherwise have to open an issue to find out — what it is waiting on, what was
  decided. A line with no kind gets a dot, one with a kind gets a glyph from the same set the
  statuses use, and the words stay quiet either way, because a card with three loud lines has no
  title left. `CardHeight` reports the room a card needs for what it has been given: a component
  that can grow has to be able to say by how much, or every caller works it out again and one of
  them gets it wrong.

- Two more ways for a drawing to move, both declarative. `ClassDraw` writes a stroked path on —
  a sparkline, an elbow connector, the arc of a progress ring — and works on any path of any length
  because the element declares `pathLength="1"`, so the renderer measures it as one unit and a single
  rule covers all three. `ClassRoll` steps a number up to its value one written value at a time
  rather than interpolating: a counter easing through 7.4 on its way to 12 is telling the reader
  something untrue about a total. Both settle on the finished state when motion is off, rather than
  freezing on the first frame of it.
- The component gallery moves, because movement is its subject: the rings and the sparkline draw
  themselves on, the meters fill, the count arrives. The plan diagrams stay as restrained as they
  were — there, motion is spent on the live path, work in flight, work blocked, and progress gained.

- The README shows the editor and the component vocabulary. Both pictures are drawn by devtree from
  the same components its diagrams are, rather than captured: a screenshot is wrong the week after
  somebody changes a colour, and nobody notices until a reader does. The editor illustration is also
  the component library's first composition — if a card, a chip and a connector cannot be assembled
  into something that looks like the product, the library is wrong.

- A drawing vocabulary: twenty components — card, cluster, callout, swimlane, column head, stat,
  chip, badge, pill, avatar, milestone, legend, bar, ring, sparkline, timeline, divider, connector,
  arrow, brace — sitting between the primitives that write SVG and a finished picture. They know
  nothing about development plans: a pill takes a colour and a label, and the renderer above decides
  that "blocked" is red, which is what lets the documentation's illustrations and the plan diagrams
  use one library rather than two. A generated gallery draws every one of them, walking the list, so
  a component added without a demo fails the build instead of going undocumented.
- Layout instrumentation to place them with: a `Rect` that insets, splits and divides into rows and
  columns, so a card's header is "the top 34, padded by 12" rather than four numbers recomputed at
  every use. With it, the measuring that drawing text actually needs — fitting a size until it fits,
  wrapping into at most n lines with the last one ending in an ellipsis, initials from a name,
  plurals, and a percentage that treats nothing to divide by as nothing done.

- `devtree serve` — a local editor at `http://127.0.0.1:9312`. The plan on the left, drawn on the
  right, live: five views behind the tabs (tree, board, page, Mermaid, YAML), rendered on demand,
  with nothing written to disk until you ask. It binds to the loopback interface only, stores
  nothing of its own, and every change goes through the same domain rules and lands in the same
  plan file the command line writes, so the editor and a terminal can be used in the same minute.
  The page listens for the file changing, so an edit made in either shows up in the other.
- The editor is three regions now: a rail of sections, a panel listing the plan or the outputs, and
  the drawing. The panel draws nesting with guides rather than indentation, filters with a search
  box that keeps the parents of whatever matched, and hides finished work behind All/Open.
- Documents are a section of their own. Every destination in `outputs` is listed with what its name
  says it holds — tree or board, light or dark, page or Mermaid block — because that is the rule the
  writer follows, read back out of the name. New destinations can be named from there; naming one
  does not write it, which is still what the render button is for.
- Every card in a drawing carries the id of the task it draws, so the diagram is the editing
  surface: click a card to open it, hover one for a `+` that adds work underneath it. The attribute
  is on the written `.svg` too — the editor gets no special version.
- The editor asks in its own panel instead of calling up a browser dialog. Deleting a task with work
  under it now says how much is under it and what happens to it, which is the sentence a native
  confirm box cannot phrase. A test keeps the page from quietly going back.
- `devtree serve --host` for running the editor from the container, where loopback belongs to the
  container and a published port would otherwise reach nothing. Binding anywhere but `127.0.0.1`
  prints a warning, because the editor has no authentication.
- The editor's motion is the Transitions.dev set, each snippet doing a job: a pill slides between
  tabs, a panel arrives from the edge it lives on, the status menu drops from its control, the theme
  icon swaps with a blur, a check draws itself once the outputs are written, a skeleton holds the
  layout while a drawing is fetched, and the title shimmers until the plan has a name.
  All of it stops under `prefers-reduced-motion`, and `?live=0` turns off the change stream.
- The editor is dark by default, follows the reader's system setting when they have expressed one,
  and remembers the choice after that. Its scrollbars are drawn rather than inherited: the system
  scrollbar is a light-mode object on most machines and landed on a near-black page as a bright
  stripe.

### Fixed

- The in-progress mark turns its ring, not itself. It is a ring around a check,
  and spinning the whole glyph tumbled the check with it — a mark falling over
  rather than work in flight. A glyph can now name the part of itself that
  moves; renderers put the motion there, and strip the marker when nothing is
  moving so a still drawing carries no dead classes. The editor honours the same
  nomination, because a status that turns in the diagram and sits still in the
  list would be two marks rather than one.
- The editor's regions scroll. The grid holding them had no row of its own, so
  it took its height from its content: the regions grew past the bottom of the
  window and the body clipped the rest, which reads as a page that has simply
  stopped scrolling. The row is the window now, and every region is allowed to
  be shorter than what is inside it.
- The drawing sits in the middle of the stage, centred with `safe` — plain
  centring pushes from both sides, so a drawing larger than the stage would have
  its top and left pushed out of reach: centred and unreachable.
- No drawing is narrower than its own summary. Sizing the board's columns for
  their cards let a one-column board shrink until "23 / 25 tasks done" ran off
  the edge of it; the header now states the width it needs and both drawings
  floor on it.
- The view is in the address (`#board`), so a reload comes back to it and it can
  be handed to somebody as a link.
- The view is chosen from a dropdown rather than a row of five tabs. Tabs spent
  the width of the header saying what you were not looking at, and gave each
  view a word where it could have had a sentence; in the list every view says
  what it gives you.
- The Mermaid block no longer fights the reader's theme. It pinned a light fill and a dark label
  color, which glared on a dark GitHub page; a translucent tint was tried instead and turned out
  unreadable there, because Mermaid keeps the label dark in its dark theme. Both were rendered
  against a real Mermaid build before the choice was made. The status now rides on the stroke and on
  the glyph, and nothing else is set, so the block looks native on either page.

### Changed

- The worked example is a web project rather than a payment gateway: one MVP split into a backend —
  authentication, the API's shape, a Docker image — and a frontend, where the framework choice is
  written down as a task with the reason in its note, next to the catalog, the checkout and nginx.
  It is the shape most teams recognise, and it puts the decisions that hold each half up in the plan
  instead of in somebody's head. The README, the board view, the getting-started walk and the file
  format reference all use it, and the terminal blocks are printed by the generator rather than
  transcribed, so they cannot drift from the plan they claim to show.
- Every picture in the README takes the width of the column it is given rather than stopping at 800.

- The pill is a rounded tag washed in the colour it means, rather than a fully rounded stadium.
  Stadium ends read as a button — something you could press — and a pill is a label reporting a
  state nobody clicks. The wash carries the meaning from across a diagram while the label keeps full
  contrast, which a saturated fill would have spent. It is drawn as opacity rather than a pre-mixed
  colour, so one value per status covers both the light and the dark drawing.
- The README leads with the drawing vocabulary — the gallery and the table right after the diagram
  itself, the code that uses them in a section of its own — and shows the editor as a capture of the
  real page in both themes rather than an illustration of it.

- "What it looks like" in the README is devtree's own drawing and the terminal view. The Mermaid
  block is still written, still native on GitHub and GitLab, and still one link away in `TREE.md` —
  it is simply no longer what a visitor meets first. This repository now renders into `TREE.md`
  rather than injecting into its own README.
- The board sizes its columns for the cards in them rather than for the canvas. Late in a project,
  when every status but one has emptied out, the board is now a column instead of one column
  stretched across the whole drawing.

## [0.5.0] — 2026-08-16

The plan can now tell you where it has been, and devtree meets you wherever you build.

### Added

- `devtree history` reads past versions of the plan out of git and shows how far along it was each
  time. Nothing is tracked to make this work: the plan has been committed all along, so its own past
  is already recorded. A revision that cannot be parsed — one written by an older devtree carrying a
  field this one dropped — is skipped with a note instead of taking the history down with it.
- `devtree install gitlab` writes `.gitlab/devtree.yml`, a job that runs the published container and
  fails a merge request whose diagram is stale. It is a separate file rather than an edit to
  `.gitlab-ci.yml`, which belongs to the project. `install all` leaves it out: a repository is almost
  never on both hosts.
- A Homebrew tap: `brew install SergeyLubivui-dev/tap/devtree`. The formula installs the released
  binary rather than building from source, and a scheduled workflow in the tap rewrites the version,
  the URLs, and the checksums whenever devtree publishes a release.

## [0.4.0] — 2026-08-16

Shorter distance between the plan and the work: jump to what a task points at, narrow a view to one
person or one branch, and export a page you can actually click.

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

[Unreleased]: https://github.com/SergeyLubivui-dev/devtree/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/SergeyLubivui-dev/devtree/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/SergeyLubivui-dev/devtree/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/SergeyLubivui-dev/devtree/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/SergeyLubivui-dev/devtree/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/SergeyLubivui-dev/devtree/releases/tag/v0.1.0
