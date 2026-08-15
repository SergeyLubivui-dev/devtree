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
gofmt -l .        # must print nothing
go vet ./...
go test -race ./...
go run . check --strict
go run . render   # must leave the working tree clean
```

CI runs exactly these on Linux, macOS, and Windows. The last two are the same checks devtree
installs into other people's repositories — the project plans itself with its own tool, so a change
that makes the diagram stale fails the build.

## Where code goes

The dependencies point inward, and pull requests are reviewed with that in mind:

| Package | Holds | May import |
|---|---|---|
| `internal/tree` | nodes, parents, statuses, validation | standard library only |
| `internal/store` | the strict YAML subset: parse, marshal, save | `tree` |
| `internal/render` | Mermaid, Markdown, ASCII — pure functions | `tree` |
| `internal/scaffold` | hook, workflow, `.gitattributes` templates | — |
| `internal/cli` | flags, dispatch, everything the user sees | all of the above |

Two rules follow from that table, and they are the ones most likely to come up in review:

- **Only `internal/cli` prints.** Everywhere else, return an error or a value and let the caller
  decide how to say it.
- **Nothing calls `os.Exit` except `main`.** Commands return errors; `Execute` turns them into an
  exit code.

## Style

Comments explain *why*, not *what* — the code already says what it does. If a line looks strange,
the comment above it should say which failure it prevents. Have a look at `internal/render/inject.go`
or the note about the flat node list in `internal/tree/tree.go` for the shape.

## Reporting a bug

Include the `.devtree/tree.yaml` that triggered it, the command you ran, and what devtree printed.
A parse error always names a line number — paste that too.
