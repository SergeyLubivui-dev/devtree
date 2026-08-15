# Security policy

## What devtree touches

It is worth being concrete about the attack surface, because it is small:

- devtree reads and writes files inside the project directory: `.devtree/tree.yaml`,
  `.devtree/archive.yaml`, and whatever `outputs` names.
- It makes no network requests, ever — not at install time, not at render time.
- It runs one external program, `git`, from one command, `devtree sync`, and only to read which
  branches are merged.
- Its output is self-contained SVG and Markdown: no scripts, no external references, nothing fetched
  when the picture is displayed.

The plan file is untrusted input as far as the parser is concerned: it is a strict subset of YAML,
parsed by hand, and anything outside that subset is rejected with a line number rather than
interpreted. Text from the plan is escaped before it reaches a diagram.

## Supported versions

The latest release. This is a small tool with a fast release cadence; fixes go into the next tag
rather than into patch branches.

## Reporting something

Open a [security advisory](https://github.com/SergeyLubivui-dev/devtree/security/advisories/new) —
that keeps the report private until there is a fix. Please include the plan file or the command that
triggers it, and what you expected instead.

If it turns out not to be a security issue, a normal issue is perfectly fine and nothing is lost by
having started privately.
