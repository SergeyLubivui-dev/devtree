# The editor

```bash
devtree serve            # http://127.0.0.1:9312
devtree serve --open     # ...and open a browser at it
devtree serve --port 8080
```

In a container, add `--host 0.0.0.0` — see [the container](container.md#the-editor-from-the-container).

Three regions, butted together: a rail of sections, a panel listing whatever the rail points at, and
the drawing. Click a card — in the panel or in the drawing itself — and it opens for editing.

## What it is, and what it deliberately is not

It is a **local editor**, not a service. It binds to the loopback interface, it is started by a
command and dies with it, and it stores nothing of its own: every change goes through the same
domain rules as `devtree set` and lands in the same `.devtree/tree.yaml`. You can have the editor
open in one window and a terminal in the other, edit in both, and neither loses a change — the page
listens for the file changing and reloads itself within a second.

There is no account, no database, no sync, and no build step. The page is three files compiled into
the binary, and the API is a handful of JSON endpoints.

`--host` is the one way out of loopback, and it exists for containers, where loopback belongs to the
container and a published port would reach nothing. Anywhere else it hands an editor with no
authentication to whoever can reach the machine, which is why devtree says so on startup whenever
the address is not `127.0.0.1`.

## The regions

**The rail** switches what the panel is listing, and holds the two buttons that act on the whole
plan: the theme, and `Write outputs`.

**The panel — Plan.** Every task as a row: the status glyph, the title, and the rollup on the right.
Nesting is drawn, not indented by guesswork — a child hangs off its parent's line, and the last one
in a group ends the line rather than letting it run on. `Search` narrows the list and keeps parents
of anything that matched, so filtering never orphans work. `All / Open` hides what is finished.

**The panel — Documents.** Every destination in `outputs`, with what its name says it will hold:

| The name | What devtree writes there |
|---|---|
| `docs/tree.svg` | the tree, light |
| `docs/board-dark.svg` | the board, dark |
| `docs/plan.html` | the interactive page |
| `TREE.md` | the Mermaid block, injected between the markers |

That table is not configuration — it is the naming rule, read back out of the file name. Adding a
document is naming a file; the drawing follows from the name, which is why `outputs` can stay a list
of paths and `devtree render` needs no arguments.

Adding one does not write it. Naming a file you might want is not the same as putting it in the
diff, so `Write outputs` stays the one button that touches the working tree — and it does exactly
what `devtree render` does.

**The drawing** is the editing surface, not a picture of one. Every card carries the id of the task
it is drawing, so clicking a card opens that task and hovering one offers a `+` that adds work
underneath it. The hierarchy is built in the diagram, where you can see the shape it is making.

Five views behind the tabs:

| Tab | What you get |
|---|---|
| Tree | the SVG devtree renders into `.svg` outputs |
| Board | the same work in columns, by status |
| Page | the HTML export, in a frame, filters and all |
| Mermaid | the block that goes into a `.md` output |
| YAML | the plan file exactly as it would be written |

They are rendered on demand and **nothing is written to disk** while you look at them. A preview
that touched the working tree would turn every keystroke into a diff.

## The drawer

Everything that edits or asks happens in one panel, arriving from the right edge: the task editor,
the new-document form, and the confirmations.

Those confirmations are deliberately not `window.confirm`. A browser dialog blocks the page, cannot
be styled, and lands wherever the browser decides — and the one question worth asking carefully,
"this task has twelve tasks under it, are you sure?", is exactly the one a native dialog cannot
phrase. So the editor asks in its own panel, with the consequence spelled out, and a test keeps it
that way.

Escape closes it, so does the backdrop, and focus stays inside it while it is open.

## The motion

Every animation has a job, and each one is a [Transitions.dev](https://transitions.dev) snippet used
as published:

| What moves | When | Why |
|---|---|---|
| a pill slides between tabs | changing the view or the scope | the eye follows the pill to the new content |
| a panel arrives from the right | editing, adding, confirming | it comes from the edge it lives on |
| a menu drops | choosing a status | it opens from the control it belongs to |
| an icon swaps with a blur | switching light and dark | one control, two states, no flicker |
| a check draws itself | after the outputs are written | the answer to "did that take?" |
| a skeleton pulses, then reveals | while a drawing is fetched | the layout does not jump when it arrives |
| the title shimmers | before the plan has loaded | it stops the moment there is a name to show |

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
curl -X POST 127.0.0.1:9312/api/document -d '{"path":"docs/board-dark.svg"}'
curl -X DELETE 127.0.0.1:9312/api/document/docs/board-dark.svg
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
