# The editor

```bash
devtree serve            # http://127.0.0.1:9312
devtree serve --open     # ...and open a browser at it
devtree serve --port 8080
```

The plan on the left, drawn on the right, live. Click a task to edit it; the drawing redraws as you
save.

## What it is, and what it deliberately is not

It is a **local editor**, not a service. It binds to the loopback interface and nothing else, it is
started by a command and dies with it, and it stores nothing of its own: every change goes through
the same domain rules as `devtree set` and lands in the same `.devtree/tree.yaml`. You can have the
editor open in one window and a terminal in the other, edit in both, and neither loses a change —
the page listens for the file changing and reloads itself within a second.

There is no account, no database, no sync, and no build step. The page is three files compiled into
the binary, and the API is a handful of JSON endpoints.

## The panes

**Left — the plan.** Every task as a card: the status glyph, the title, the branch, issue, pull
request, owner and tags beneath it, the rollup on the right. Depth is indentation. Selecting a card
opens the editor panel; `Add task` and `Add child` create; `Delete` removes, and its children move up
to its parent rather than disappearing.

**Right — the drawing.** Five views behind the tabs:

| Tab | What you get |
|---|---|
| Tree | the SVG devtree renders into `.svg` outputs |
| Board | the same work in columns, by status |
| Page | the HTML export, in a frame, filters and all |
| Mermaid | the block that goes into a `.md` output |
| YAML | the plan file exactly as it would be written |

They are rendered on demand and **nothing is written to disk** while you look at them. A preview
that touched the working tree would turn every keystroke into a diff. `Write outputs` is the button
that does write, and it does exactly what `devtree render` does.

The divider between the panes is draggable, and it responds to the arrow keys when focused.

## The motion

Every animation on the page has a job, and each one is a
[Transitions.dev](https://transitions.dev) snippet used as published:

| What moves | When | Why |
|---|---|---|
| a pill slides between tabs | changing the view | the eye follows the pill to the new content |
| a menu drops | choosing a status | it opens from the control it belongs to |
| an icon swaps with a blur | switching light and dark | one control, two states, no flicker |
| a check draws itself | after a save | the answer to "did that take?" |
| a skeleton pulses, then reveals | while a drawing is fetched | the layout does not jump when it arrives |
| the title shimmers | before the plan has loaded | it stops the moment there is a name to show |
| the panes tween | releasing the divider | dragging tracks the cursor exactly; the keyboard steps glide |

All of it stops under `prefers-reduced-motion`. And the live reload can be switched off with
`?live=0` — for a proxy that will not hold a connection open, and for tools that wait for a page to
go quiet before capturing it.

## The API, if you want it

The page has no privileged access; anything it does, `curl` can do:

```bash
curl 127.0.0.1:9312/api/plan
curl -X POST 127.0.0.1:9312/api/task -d '{"title":"Refunds","parent":"billing"}'
curl -X PATCH 127.0.0.1:9312/api/task/refunds -d '{"status":"wip","owner":"ann"}'
curl -X DELETE '127.0.0.1:9312/api/task/refunds?cascade=true'
curl 127.0.0.1:9312/api/view/board?theme=dark
curl -X POST 127.0.0.1:9312/api/render
```

Errors come back as `{"error": "..."}` with the same wording the command line uses, because the
person reading them is the same person.

## Why a port at all

devtree is a file and a binary, and adding a server to that is a real trade. It earns its place by
what it removes: the loop of edit, run, look, edit that a drawing tool otherwise imposes. Everything
else in devtree still works with no server in sight — and if you never run `devtree serve`, nothing
in the program ever opens a socket.
