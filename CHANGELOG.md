# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[semantic versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] — 2026-08-15

First public release.

### Added

- `.devtree/tree.yaml` as the source of truth: a flat, merge-friendly list of nodes in a strict
  subset of YAML, parsed by hand so errors can name the line they are on.
- Mermaid diagram generation into any Markdown file, written between `<!-- devtree:begin -->` and
  `<!-- devtree:end -->` markers so an existing README is never disturbed. Re-rendering an unchanged
  plan rewrites nothing.
- A collapsed link table under the diagram for branches, issues, and pull requests, built from the
  configured `repo` URL or from relative links when there is none.
- Commands: `init`, `add`, `set`, `done`, `mv`, `rm`, `ls`, `render`, `check`, `install`, `outputs`.
- Five statuses — `todo`, `in_progress`, `blocked`, `done`, `dropped` — with shorthand spellings
  accepted anywhere a status is.
- Progress rollups per node and for the whole plan, counting neither the node itself nor dropped
  work.
- IDs derived from titles, with Cyrillic transliterated so they stay typeable, and numeric suffixes
  on collision.
- A pre-commit hook and a GitHub Actions workflow, installable with `devtree install`, plus a
  `merge=union` rule written into `.gitattributes` by `init`.
- Validation covering duplicate IDs, dangling parents, cycles, unknown fields, and unknown statuses,
  with `check --strict` turning advisory warnings into a failure for CI.

[Unreleased]: https://github.com/SergeyLubivui-dev/devtree/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/SergeyLubivui-dev/devtree/releases/tag/v0.1.0
