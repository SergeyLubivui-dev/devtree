# Finished work

A plan that keeps every task ever completed stops being a plan and becomes a log: the board fills
with columns of done work and the diagram grows a tail nobody reads. Two commands handle that from
opposite ends — one moves finished work out, the other closes work git already knows is finished.

## Archiving

```bash
devtree archive          # what is finished and could move
devtree archive --all    # move all of it
devtree archive v1       # move one branch of the plan
devtree archive --list   # what has already left
devtree restore v1       # bring it back, with everything under it
```

### Nothing moves until you say so

Plain `devtree archive` reports and stops:

```text
  2 branch(es) of the plan are finished:
    ✔ v1  (v1)  (+2 underneath)
    ✔ Search  (search)

  archive them with:  devtree archive --all
  or one at a time:   devtree archive ID
```

Discovering after the fact that a chunk of the plan relocated is a bad afternoon. The report says
exactly how many nodes each move would take with it.

### What qualifies

A node can be archived only when its **whole subtree** is `done` or `dropped`. Anything looser would
let live work vanish along with the milestone above it, and a board that quietly loses a task nobody
closed is worse than a cluttered one.

Only the topmost finished node is offered. If a milestone qualifies, its children qualify too, and
listing them separately would offer the same work twice.

Dropped counts as finished. Abandoned work is settled work; it just settled the other way.

### Where it goes

`.devtree/archive.yaml`, in the same format as the plan — same parser, same serializer, nothing new
to learn and nothing new to review in a diff. It is read with one relaxation: an archived node keeps
the `parent` it had, which is usually still in the active plan, so the archive is parsed but not
validated as a tree. It is a record, not a plan.

Emptying the archive removes the file rather than leaving a husk behind.

### Restoring

```bash
devtree restore v1
```

The whole subtree comes back, in the order it was archived. If the parent is gone by then, the task
returns as a root and says so:

```text
  ! search came back as a root: its parent "v2" is gone
```

Refusing would be the other option, and it would be the wrong one: the point of a restore is to get
the work back in front of you. Re-parenting is one command away — `devtree mv search v3`.

## Closing what git already knows about

```bash
devtree sync           # list tasks whose branch is already merged
devtree sync --apply   # mark them done
```

Any unfinished task whose `branch` is merged into `HEAD` is proposed:

```text
  feat/auth → done   Authentication  (authentication)

  1 task(s) would change. Apply with:  devtree sync --apply
```

### Why it proposes instead of acting

git knows which branches were merged. It does not know which of them were merged *finished* — a
branch merged behind a feature flag is not a task that is done, and neither is one merged to unblock
someone else. The judgment stays with a person, and `--apply` is how they give it.

### What it cannot see

A branch that was merged and then deleted no longer exists, so git cannot report it as merged and
sync will not propose it. This is another reason the command reports rather than decides: the
absence of a branch is not evidence that the work is unfinished.

### The one external process

`internal/vcs` is the only package in devtree that runs another program, and it is reached only from
this command. Everything else works on the plan file alone — which is what lets devtree run against
a tarball, inside a container, or in a CI job that never checked anything out. If git is missing or
the directory is not a repository, sync says so plainly instead of failing halfway:

```text
devtree: git is not on PATH, so there is nothing to sync from
devtree: /work is not a git repository
```
