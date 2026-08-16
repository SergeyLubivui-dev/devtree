# The board

The tree says how the work is organized. The board says what state it is in this morning. Same file,
different question — nothing is stored twice, and nothing can disagree.

```bash
devtree board
devtree board -s blocked    # one column only
```

```text
Storefront

☐ not started · 3
  Versioning and error shape   API architecture  #24
  Checkout                     Frontend  #51
  nginx and static hosting     Frontend  @bob

◐ in progress · 1
  Catalog pages                Frontend  feat/catalog  @ann

⛔ blocked · 1
  Password reset               Authentication  — waiting on the SMTP account

✔ done · 4
  Sessions and refresh tokens  Authentication  !31
  Resources and pagination     API architecture
  Docker image                 Backend  !40  @bob
  Framework: React or Vue      Frontend  — React — the team already knows it

█████░░░░░░░░░░░░░░░  4/14
```

## Only leaves appear

A milestone is a container, not a task. A board that lists containers next to the work inside them
makes both harder to scan: the milestone shows up as a card you cannot pick up, and its progress is
already visible from the cards underneath it.

So the board shows leaves, and each card carries its milestone as a breadcrumb instead. Nothing
loses its place, and every card on the board is something a person could actually start.

## Empty columns are dropped

A board with nothing blocked should not spend a fifth of its width saying so. Columns appear only
when they have work in them, in the canonical status order — not started, in progress, blocked,
done, dropped.

## Why the terminal stacks instead of columns

Real side-by-side columns wrap into porridge the moment a terminal is narrower than the layout
assumed, and terminals are narrow at exactly the moment you most want a quick look. Stacking the
statuses keeps every line readable at any width, and the group headings carry the counts.

Alignment counts runes, not bytes. A Cyrillic or CJK title would otherwise throw the second column
out by the number of multi-byte characters in it.

## The same board as a picture

Name an output `board.svg` — or `anything.board.svg` — and the board is what gets drawn:

```bash
devtree init --outputs "docs/tree.svg, docs/board.svg, docs/board-dark.svg"
```

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="assets/board-dark.svg">
  <img alt="devtree's own board" src="assets/board.svg">
</picture>

Columns get a heading with the status glyph and a count, a hairline in the status color, and cards
that carry the same full-height accent stripe as the tree diagram. The rules above hold here too:
leaves only, empty columns dropped.

## Narrowing it

The board and the tree take the same four filters, so a habit learned on one works on the other:

```bash
devtree board -s blocked              # one status
devtree board -o ann                  # one person
devtree board --tag billing,payments  # any of these tags
devtree board --root mvp              # one branch of the plan
```

They combine with and, except tags, which combine with or — "anything touching billing or payments"
is the question people actually ask. Owners and tags ignore case, and `-o @ann` works as well as
`-o ann`.

`--root` slices the plan to one branch and hands the slice to the same renderer, which is why it
costs nothing: progress rollups, filters, and drawings all work on a plan, and a branch of a plan is
a plan.

## Where you have been

The board is today. `devtree history` is the same numbers over time, read out of the repository's own
commits:

```bash
devtree history --limit 40
```

Nothing is tracked to make this work: the plan file has been committed all along, so `git log` on it
is already the history. devtree reads each past version, counts it, and prints the line. A revision
it cannot parse — one written by an older version carrying a field this one dropped — is skipped with
a note rather than taking the rest of the history down with it.

## What it is not

There are no WIP limits, no swimlanes, no drag and drop, and no state that lives anywhere but the
plan file. The board is a view. If you want a task to move, change its status:

```bash
devtree set apple-pay -s wip
devtree done apple-pay
```

And when the done column has grown into a wall, move it out of the way — see
[Finished work](finished-work.md).
