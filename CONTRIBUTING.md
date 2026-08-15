# Contributing

devtree is a small tool with a small surface: one binary, no dependencies, five statuses, one flat
file. The goal of a review is to keep it that way while it grows.

Issues and pull requests are both welcome, and a question counts as a contribution.

## Set up

```bash
git clone https://github.com/SergeyLubivui-dev/devtree.git
cd devtree
go test ./...
```

Go 1.22 or newer. There is nothing else to install.

## Before you push

```bash
gofmt -l .             # must print nothing
go vet ./...
go test -race ./...
go run . check --strict
go run . render        # must leave the working tree clean
go run ./internal/art  # likewise — the documentation pictures are generated
```

CI runs exactly these on Linux, macOS, and Windows. The last two are the same checks devtree
installs into other people's repositories: the project plans itself with its own tool, so a change
that makes a diagram stale fails the build.

> Use `go run .`, not a binary you built earlier. A stale binary writing generated files is the one
> mistake that gets past a local run and fails in CI.

## Where things live

```text
main.go        wiring, and nothing else
internal/      the code — see the table at the bottom
docs/          documentation; assets/ is generated, i18n/ holds the translations
.devtree/      devtree's own plan, and its archive
.github/       workflows, issue forms, the security policy
Dockerfile     the image published to GHCR
```

---

## Find your scenario

### 1. A typo, a clearer sentence, a broken link

Edit the file, push. That is the whole process — no test, no regeneration.

If it is a user-facing sentence in `README.md`, the same sentence probably exists in
`docs/i18n/README.*.md`. Change what you can; leaving three languages untouched is better than
guessing at them, and a note in the pull request is enough.

### 2. A bug you can reproduce

Write the test first, watch it fail, then fix it. Every test in this repository can be run against a
temporary directory with no setup:

```go
func TestArchiveRefusesToTakeLiveWorkWithIt(t *testing.T) {
	h := newHarness(t)
	h.run("init", "--empty")
	h.run("add", "v2", "-s", "wip")
	h.run("add", "Filters", "-p", "v2")

	msg := h.runExpectingFailure("archive", "v2")
	if !strings.Contains(msg, "unfinished") {
		t.Fatalf("got %q", msg)
	}
}
```

`h.run` fails the test if the command exits non-zero, `h.runExpectingFailure` fails it if the
command succeeds, and `h.read("TREE.md")` gives you back what was written. That is the whole harness.

**Files:** the test next to the code, and the fix.

### 3. A new field on a task

Say you want `due`, a date on a card. It crosses four files, in this order:

```go
// internal/tree/tree.go — the domain learns the field exists
type Node struct {
	...
	Owner string
	Due   string
}
```

```go
// internal/store/codec.go — reading it
case "due":
	n.Due = value

// ...and writing it, in the fixed field order
optional("due", n.Due)
```

```go
// internal/render/svg/layout.go — showing it on a card
if n.Due != "" {
	out = append(out, metaItem{"clock-circle", n.Due})
}
```

```markdown
<!-- docs/file-format.md — the field table -->
| `due` | no | A date, in whatever form you write dates. Shown on the card. |
```

**What CI will check for you:** the round-trip test in `internal/store` proves a plan carrying the
new field comes back byte-identical, and the SVG tests prove it is escaped. Add the field to the
sample plan in `codec_test.go` and both happen automatically.

**Files:** `tree`, `store`, one renderer, `docs/file-format.md`, and the README table if it is worth
a mention there.

### 4. A new command

Say `devtree next`, printing the task you should pick up:

```go
// internal/cli/cli.go — dispatch and help
case "next":
	err = a.cmdNext(rest)
```

```go
// internal/cli/commands.go — the command itself
func (a *App) cmdNext(args []string) error {
	fs := newFlagSet("next")
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}

	_, t, err := a.load()
	if err != nil {
		return err
	}
	// ...work out the answer, then print it to a.Out
	return nil
}
```

Three things a review will look for:

- it prints to `a.Out` and `a.Err`, never to `os.Stdout` directly;
- it returns an error instead of exiting;
- it has a harness test like the one in scenario 2.

**Files:** `internal/cli`, its test, the help text, the README command table, and a line in `docs/`.

### 5. Changing what a drawing looks like

Two places draw: `internal/render/svg` draws plans, `internal/art` draws the pictures in the
documentation. Both build on `internal/draw`, so a change to spacing, text measurement, or a glyph
lands in both — that is deliberate, and it is why they cannot drift apart.

```bash
go run . render
go run ./internal/art
```

Then look at the result. A headless browser is the quickest honest check:

```bash
chrome --headless --screenshot=out.png --window-size=860,900 \
  --virtual-time-budget=2000 file:///path/to/docs/assets/tree.svg
```

Two things to keep in mind, both learned the hard way:

- **Determinism.** Rendering the same plan twice must produce identical bytes. No maps iterated into
  output, no timestamps. There is a test for this.
- **Motion must be optional.** Anything animated has to read correctly with animation switched off —
  the travelling dash sits on a line that is already solid, the growing bar already has its final
  width. `prefers-reduced-motion` and `print` turn it all off.

**Files:** the renderer, its test, and the regenerated files under `docs/assets/`.

### 6. A translation

`README.md` is the source; `docs/i18n/README.{ru,zh-CN,de,fr}.md` follow it. They cover what a user
needs and point at `docs/` for the pages aimed at people reading the code.

Update them when a change alters what a user does or sees, and leave them alone when it does not. If
you add an example, run it first — every command and every id in those files came from a real run,
because a translated example that quietly lies is worse than no translation.

---

## The two rules a review leans on

- **Only `internal/cli` prints.** Everywhere else, return a value or an error and let the caller
  decide how to say it. This is what makes whole commands testable without capturing output.
- **Nothing calls `os.Exit` except `main`.** Commands return errors; `Execute` turns them into an
  exit code.

## Which package

Dependencies point inward. The right home for something is usually the innermost box that can hold
it:

| Package | Holds | May import |
|---|---|---|
| `internal/tree` | nodes, parents, statuses, validation | standard library only |
| `internal/icons` | vendored glyph paths | standard library only |
| `internal/draw` | measuring, wrapping, escaping, glyph placement | `icons` |
| `internal/store` | the file format: parse, marshal, save, the archive | `tree` |
| `internal/render` | Mermaid, Markdown injection, terminal output | `tree` |
| `internal/render/svg` | layout, cards, themes, output naming | `tree`, `draw` |
| `internal/scaffold` | hook, workflow, `.gitattributes` templates | — |
| `internal/vcs` | the only place that runs git | — |
| `internal/cli` | flags, dispatch, everything the user sees | all of the above |
| `internal/art` | the documentation pictures; a tool, not part of the binary | `draw`, `render/svg` |

The long version, with the reasoning: [docs/architecture.md](docs/architecture.md).

## Style

Comments explain *why*, not *what* — the code already says what it does. If a line looks strange,
the comment above it should name the failure it prevents. Two examples worth copying:
`internal/render/svg/inject.go` on why only markers at the start of a line count, and the note in
`internal/tree/tree.go` on why the node list is flat.

## Reporting a bug

Use the [bug form](https://github.com/SergeyLubivui-dev/devtree/issues/new?template=bug_report.yml).
Three things make a report actionable: the plan that triggered it, the command you ran, and what
devtree printed. Parse errors always name a line number — paste that line too.

Something that looks like a security issue goes to a
[private advisory](https://github.com/SergeyLubivui-dev/devtree/security/advisories/new) instead;
if it turns out to be an ordinary bug, nothing is lost by having started there.
