# Automation

Two pieces of scaffolding keep the diagram from drifting away from the plan. They cover different
failure modes, and installing both is the point.

```bash
devtree install hook      # your own commits
devtree install action    # everybody else's
devtree install all
```

## The pre-commit hook

`devtree install hook` writes `.git/hooks/pre-commit`:

```sh
#!/bin/sh
set -e

if ! command -v devtree >/dev/null 2>&1; then
  echo "devtree is not on PATH — skipping the tree update" >&2
  exit 0
fi

devtree check
devtree render --quiet

git add .devtree/tree.yaml $(devtree outputs) 2>/dev/null || true
```

Four decisions in nine lines:

- **A missing binary is not an error.** A teammate who has not installed devtree should still be
  able to commit. The Action below catches what the hook skips.
- **A broken plan is an error.** `check` runs first and `set -e` stops the commit. A duplicate id or
  a dangling parent is worth interrupting for.
- **The refreshed files are staged**, so the commit contains a diagram that matches the plan it was
  generated from — not one that matches the previous commit.
- **`devtree outputs` supplies the file list** rather than a hard-coded `TREE.md`, so renaming an
  output does not silently stop staging it.

An existing hook is never destroyed. It is copied to `pre-commit.devtree-backup` first, and devtree
says so.

## The GitHub Action

`devtree install action` writes `.github/workflows/devtree.yml`. It installs devtree, validates, and
renders — then behaves differently depending on what triggered it:

- **On `pull_request`**, it fails if rendering changed anything. The author fixes it in their own
  diff, where a reviewer can see it.
- **On `push` to the default branch**, it commits the refreshed diagram itself.

That split is deliberate. Amending someone's pull request from a bot hides the change from the
review; leaving the default branch stale means the front page of the repository lies to visitors.

The workflow uses `cache: false` on `setup-go`, because the repository it runs in may not be a Go
project at all, and the cache step would otherwise warn on every run.

## The merge rule

`devtree init` adds one line to `.gitattributes`:

```
.devtree/tree.yaml merge=union
```

Union merge keeps both sides of a conflicting change. It is the right tool here precisely because
the node list is flat: two branches append lines in different places, git takes both, and the worst
case is a duplicate id that `devtree check` names on the spot. See
[the file format](file-format.md) for why the list is flat in the first place.

## Closing tasks from git

`devtree sync` reads which branches are already merged and proposes the matching tasks as done. It
is covered in [Finished work](finished-work.md), along with why it proposes rather than acts.

## Using it in someone else's CI

Nothing about devtree requires the workflow it installs. The two commands worth running anywhere
are:

```bash
devtree check --strict   # fails on warnings too
devtree render           # then assert the working tree is clean
```

`--strict` turns advisory warnings into a failure: a task marked done with unfinished work under it,
a blocked task with no note, a node with no title. Those are fine to ignore locally and worth
blocking a merge over.

Both work in [the container](container.md) if you would rather not install a Go toolchain on the
runner.
