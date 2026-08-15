# devtree documentation

The [main README](../README.md) is the tour. These pages are the detail underneath it — what each
thing does, and why it was built that way.

| Page | What it covers |
|---|---|
| [Getting started](getting-started.md) | A first plan, from an empty repository to a diagram in the README |
| [The file format](file-format.md) | Every field, every rule, and what the parser refuses |
| [The board](board.md) | Working by status instead of by structure |
| [Finished work](finished-work.md) | Archiving, restoring, and closing tasks from merged branches |
| [Automation](automation.md) | The pre-commit hook and the GitHub Action, line by line |
| [SVG output](svg-output.md) | File naming, palettes, the animation vocabulary, and the CSP it survives |
| [The container](container.md) | Running devtree without installing anything |
| [Architecture](architecture.md) | The layering, the rules that hold it, and how it is tested |

Everything in `assets/` is generated. `devtree render` writes the plan diagrams and the board;
`go run ./internal/art` draws the pictures the README uses. Both are checked in CI, so a stale file
fails the build rather than quietly misleading a reader.
