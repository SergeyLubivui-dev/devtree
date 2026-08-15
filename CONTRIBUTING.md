# Contributing

Thanks for taking a look. This is a small tool with a small surface, and the goal is to keep it that
way: one binary, no dependencies, a plan file a person can read.

## Getting set up

```bash
git clone https://github.com/SergeyLubivui-dev/devtree.git
cd devtree
go test ./...
go build -o devtree .
```

Go 1.22 or newer. There is nothing else to install.

## Before you open a pull request

```bash
gofmt -l .             # must print nothing
go vet ./...
go test -race ./...
go run . check --strict
go run . render        # must leave the working tree clean
go run ./internal/art  # likewise — the README's pictures are generated
```

CI runs exactly these on Linux, macOS, and Windows. The last two are the same checks devtree
installs into other people's repositories — the project plans itself with its own tool, so a change
that makes the diagram stale fails the build.

## Where things live

```text
main.go        wiring, and nothing else
internal/      the code — see the table below
docs/          documentation, with generated pictures in assets/ and translations in i18n/
.devtree/      devtree's own plan, and its archive
.github/       workflows, issue templates, the security policy
Dockerfile     the container image published to GHCR
```

## Where code goes

The dependencies point inward, and pull requests are reviewed with that in mind:

| Package | Holds | May import |
|---|---|---|
| `internal/tree` | nodes, parents, statuses, validation | standard library only |
| `internal/icons` | vendored glyph paths | standard library only |
| `internal/draw` | measuring, wrapping, escaping, stamping a glyph | `icons` |
| `internal/store` | the strict YAML subset: parse, marshal, save | `tree` |
| `internal/render` | Mermaid, Markdown, ASCII — pure functions | `tree` |
| `internal/render/svg` | the native drawing: layout, cards, themes | `tree`, `draw` |
| `internal/scaffold` | hook, workflow, `.gitattributes` templates | — |
| `internal/cli` | flags, dispatch, everything the user sees | all of the above |
| `internal/art` | the README's pictures — a tool, not part of the binary | `draw`, `render/svg` |

Two rules follow from that table, and they are the ones most likely to come up in review:

- **Only `internal/cli` prints.** Everywhere else, return an error or a value and let the caller
  decide how to say it.
- **Nothing calls `os.Exit` except `main`.** Commands return errors; `Execute` turns them into an
  exit code.

## Translations

`README.md` is the source. The four translations live in `docs/i18n/` and cover everything a user
needs — install, quickstart, recipes, board, finished work, commands, format, automation, SVG
output, limits — then point at `docs/` for the pages that are aimed at people reading the code.

If a change alters what a user does or sees, update the translations in the same pull request. If it
only touches internals, leave them alone. A translation that quietly drifts is worse than one that
was never written, so the command output and IDs in every example are taken from real runs, not
invented.

## Style

Comments explain *why*, not *what* — the code already says what it does. If a line looks strange,
the comment above it should say which failure it prevents. Have a look at `internal/render/inject.go`
or the note about the flat node list in `internal/tree/tree.go` for the shape.

## Reporting a bug

Include the `.devtree/tree.yaml` that triggered it, the command you ran, and what devtree printed.
A parse error always names a line number — paste that too.
