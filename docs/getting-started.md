# Getting started

Fifteen minutes, from an empty repository to a diagram that keeps itself current.

## 1. Start a plan

```bash
cd your-project
devtree init --project "Payment Gateway" --repo https://github.com/acme/pay
```

Three things appear:

- `.devtree/tree.yaml` — the plan, with two sample tasks so the first diagram is not empty. Pass
  `--empty` to skip them.
- `.gitattributes` — gains a `merge=union` rule for the plan file. This is what makes two branches
  that each added a task merge without a conflict.
- `TREE.md` — the first rendered diagram.

The `--repo` URL is optional but worth setting: it turns issue and pull request numbers into links.
Without it devtree falls back to relative links, which work from a file in the repository root and
survive a fork.

## 2. Describe the work you already know about

```bash
devtree add "Authentication" -p mvp -b feat/auth -i 12 -o ann -s wip
devtree add "OAuth providers" -p authentication -b feat/oauth
devtree add "Password reset"  -p authentication -s blocked -n "waiting on SMTP"
```

Each flag is optional, and each one buys something:

| Flag | What it gets you |
|---|---|
| `-p` | Puts the task under a milestone. Parents count their children automatically. |
| `-b` | The branch the work lives on. `devtree sync` later closes the task when that branch is merged. |
| `-i` / `--pr` | A number that becomes a link in the table under the diagram. |
| `-o` | Who has it. Shows on the card and on the board. |
| `-n` | Why it is where it is. Required in spirit for blocked work — `check` says so. |
| `--tags` | Free-form labels, for your own filtering. |

The identifier comes from the title: "OAuth providers" becomes `oauth-providers`. Non-Latin titles
are transliterated so the ID stays typeable on any keyboard. Use `--id` to choose one yourself.

## 3. Look at it

```bash
devtree ls       # the tree, with progress rolled up per milestone
devtree board    # the same work, grouped by status
```

`ls` answers how the work is organized. `board` answers what state it is in this morning. Both read
the same file; neither stores anything.

Both narrow the same way — `-s blocked`, `-o ann`, `--tag billing`, `--root mvp` — and the filters
combine. When you want the thing a task points at rather than the task itself:

```bash
devtree open authentication          # opens the pull request, issue, or branch
devtree open authentication --print  # prints the URL, for a machine with no browser
```

## 4. Put the diagram where people will see it

The plan lists its own output files, and the file name decides what gets drawn:

```yaml
outputs: "README.md, docs/tree.svg, docs/tree-dark.svg, docs/board.svg"
```

- `.md` files get a Mermaid block, written between `<!-- devtree:begin -->` and
  `<!-- devtree:end -->` markers. Put the empty pair wherever you want the diagram; everything
  outside it is left alone.
- `.svg` files get devtree's own drawing. A `-dark` suffix picks the dark palette, and a `board`
  name picks the board layout. See [SVG output](svg-output.md).

Then `devtree render` rewrites all of them. Running it twice changes nothing the second time, which
is what lets the checks in the next step be meaningful.

## 5. Stop it from going stale

```bash
devtree install hook      # validate and re-render before every commit
devtree install action    # fail a pull request whose diagram is out of date
```

Both are covered in [Automation](automation.md). The short version: the hook keeps your own commits
honest, and the Action keeps everyone else's honest without requiring them to install anything.

## 6. Keep it a plan, not a log

Months later, most of the plan is finished and the board is mostly a wall of done work:

```bash
devtree archive          # what could move
devtree archive --all    # move it into .devtree/archive.yaml
```

Nothing is lost — `devtree archive --list` shows the record, and `devtree restore ID` brings any of
it back. See [Finished work](finished-work.md).

## Where to go next

- [The file format](file-format.md), if you would rather edit the plan by hand — it is designed for
  that, and the parser tells you the line number when it disagrees.
- [The board](board.md), for the day-to-day view.
- [Architecture](architecture.md), if you are here to read the code.
