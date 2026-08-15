# The file format

`.devtree/tree.yaml` is a deliberately strict subset of YAML: a flat list of nodes whose fields are
all scalars. It is parsed by hand rather than by a YAML library, and the trade buys two things — the
binary keeps its zero dependencies, and every error names the line it is on.

## The document

```yaml
# Development tree. Edit by hand or through the `devtree` command.
# After editing, run `devtree render` to refresh the diagram.
version: 1
project: "Payment Gateway"
repo: "https://github.com/acme/pay"
outputs: "TREE.md, docs/tree.svg"
nodes:
  - id: "mvp"
    title: "MVP"
    status: "in_progress"
```

| Key | Meaning |
|---|---|
| `version` | Format version. Currently always `1`. |
| `project` | Name shown above every diagram. |
| `repo` | Repository URL, used to build issue and pull request links. Optional; a trailing slash is trimmed. |
| `outputs` | Comma-separated list of files to write. The name decides the format — see [SVG output](svg-output.md). |
| `nodes` | The list. Everything below is a field of one node. |

## A node

| Field | Required | Notes |
|---|---|---|
| `id` | yes | Letters, digits, dot, dash, underscore; must start with a letter or digit. Unique. |
| `title` | yes in practice | What a human reads. Falls back to the id if empty. |
| `status` | written always | One of `todo`, `in_progress`, `blocked`, `done`, `dropped`. |
| `parent` | no | The id of the node above it. Absent means it is a root. |
| `branch` | no | Branch name. `devtree sync` uses it to close finished work. |
| `issue` | no | Issue number. A leading `#` is trimmed on the way in. |
| `pr` | no | Pull request number. Same trimming. |
| `owner` | no | A name or handle. A leading `@` is trimmed. |
| `tags` | no | Comma-separated labels. |
| `note` | no | Free text. Says why blocked work is blocked. |

Empty fields are not written at all, so a node carries only what it actually has.

## Why the list is flat

The hierarchy could have been nesting. It is a `parent` field instead, and that is the single most
consequential decision in the format.

Nesting means that adding a task rewrites an existing block: the parent's child list grows, its
indentation shifts, and two branches that both added work under the same milestone now disagree
about the same lines. That is a merge conflict in a file nobody wants to resolve by hand.

A flat list turns "add a task" into "append lines at the end". Two branches append in different
places, git takes both, and `init` writes `merge=union` into `.gitattributes` so it does that
without asking. The worst outcome is two branches picking the same id, and `devtree check` catches
that immediately, by name.

## What the parser refuses

Validation happens in two stages, and both report the line.

**Parsing** rejects anything outside the subset:

```text
devtree: .devtree/tree.yaml: line 14: unknown node field "assignee"
devtree: .devtree/tree.yaml: line 9: unknown status "nearly"
devtree: .devtree/tree.yaml: line 3: expected "key: value", got "nodes"
```

**Building** rejects a list that parses but is not a tree:

- a duplicate id
- a `parent` that does not exist
- a node that is its own parent
- a cycle of any length

Not supported, and not planned: anchors, aliases, multi-line block scalars, nested mappings, lists
of lists. If a plan needs those, it has stopped being a plan.

## Things that are forgiving on purpose

- **Quotes are optional on input.** `title: MVP` and `title: "MVP"` both parse; the writer always
  quotes, so the file stays uniform.
- **CRLF and a leading byte order mark** are stripped before parsing. Windows editors add both.
- **Status shorthand** is accepted everywhere a status is: `wip`, `doing`, `b`, `d`, `x`, `ok`, and
  a dozen more. The canonical spelling is what lands in the file.
- **`#12` and `12`** are the same issue number; the `#` is decoration.

## Round trips are byte-stable

Field order is fixed, empty fields are omitted, and every value is quoted the same way. Parsing a
file and writing it back produces the same bytes. That is what lets the pre-commit hook and the CI
check treat "the file changed" as "something actually changed", instead of crying wolf on every
commit.

## The archive

`.devtree/archive.yaml` uses this exact format — same parser, same serializer. It is read more
leniently in one respect: an archived node keeps the `parent` it had, which is usually still in the
active plan, so the archive is parsed but not built as a tree. See [Finished work](finished-work.md).
