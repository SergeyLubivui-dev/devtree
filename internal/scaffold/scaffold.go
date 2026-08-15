// Package scaffold writes the files that make a plan keep itself up to date:
// a pre-commit hook, a GitHub Actions workflow, and a .gitattributes rule.
//
// Nothing here prints. Each installer returns a Result describing what it did
// so the CLI owns every line the user sees, and so the behavior can be tested
// without capturing output.
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Result describes one installed file.
type Result struct {
	Path    string // path relative to the project root
	Note    string // extra detail worth showing, e.g. a backup that was made
	Skipped bool   // already in place, nothing written
}

// HookScript runs before every commit. It is deliberately forgiving about a
// missing binary — a teammate who has not installed devtree should still be
// able to commit — but strict about a broken plan, which is a real error worth
// stopping for.
const HookScript = `#!/bin/sh
# devtree pre-commit hook
# Validates the plan and refreshes the diagram before the commit is written.
set -e

if ! command -v devtree >/dev/null 2>&1; then
  echo "devtree is not on PATH — skipping the tree update" >&2
  exit 0
fi

devtree check
devtree render --quiet

# Stage the refreshed files so the commit contains a diagram that matches the
# plan it was generated from.
git add .devtree/tree.yaml $(devtree outputs) 2>/dev/null || true
`

// ActionWorkflow enforces the same rule in CI, from two directions: a pull
// request fails if the diagram is stale, and a push to the default branch
// refreshes it. Contributors are never blocked by a missing local hook, and
// the branch never carries a diagram that disagrees with the plan.
const ActionWorkflow = `name: devtree

on:
  push:
    branches: [ main, master ]
  pull_request:

permissions:
  contents: write

jobs:
  tree:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7

      - uses: actions/setup-go@v7
        with:
          go-version: 'stable'
          # Your repository may not be a Go project at all, so there is
          # nothing for the module cache to key on.
          cache: false

      - name: Install devtree
        run: |
          go install github.com/SergeyLubivui-dev/devtree@latest
          echo "$(go env GOPATH)/bin" >> "$GITHUB_PATH"

      - name: Validate the plan
        run: devtree check

      - name: Render the diagram
        run: devtree render

      # On a pull request the diagram is checked, not fixed: the author should
      # see the change in their own diff rather than have a bot amend it.
      - name: Fail if the diagram is stale
        if: github.event_name == 'pull_request'
        run: |
          if ! git diff --quiet; then
            echo "::error::The diagram is out of date. Run 'devtree render' and commit the result."
            git diff --stat
            exit 1
          fi

      - name: Commit the refreshed diagram
        if: github.event_name == 'push'
        run: |
          if git diff --quiet; then exit 0; fi
          git config user.name  "github-actions[bot]"
          git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
          git commit -am "chore(devtree): refresh the tree diagram"
          git push
`

// GitAttributes tells git to keep both sides of a conflicting plan.
//
// It pairs with the flat node list: two branches that each appended a task
// produce two sets of added lines, and union merge keeps them both. Worst
// case you get a duplicate ID, which `devtree check` catches immediately —
// far better than a conflict marker in the middle of a YAML file.
const GitAttributes = `# devtree: a flat task list merges line by line
.devtree/tree.yaml merge=union
`

// InstallHook writes .git/hooks/pre-commit, backing up whatever was there.
func InstallHook(root string) (Result, error) {
	gitDir, err := gitDir(root)
	if err != nil {
		return Result{}, err
	}

	hooks := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		return Result{}, err
	}

	path := filepath.Join(hooks, "pre-commit")
	res := Result{Path: relative(root, path)}

	if existing, err := os.ReadFile(path); err == nil && len(existing) > 0 {
		if string(existing) == HookScript {
			res.Skipped = true
			res.Note = "already installed"
			return res, nil
		}
		// Never destroy someone's hook silently: keep it next to the new one
		// so it can be merged back or restored.
		backup := path + ".devtree-backup"
		if err := os.WriteFile(backup, existing, 0o755); err != nil {
			return Result{}, err
		}
		res.Note = "previous hook saved as " + filepath.Base(backup)
	}

	if err := os.WriteFile(path, []byte(HookScript), 0o755); err != nil {
		return Result{}, err
	}
	return res, nil
}

// InstallAction writes .github/workflows/devtree.yml.
func InstallAction(root string) (Result, error) {
	dir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Result{}, err
	}

	path := filepath.Join(dir, "devtree.yml")
	res := Result{Path: relative(root, path)}

	if existing, err := os.ReadFile(path); err == nil && string(existing) == ActionWorkflow {
		res.Skipped = true
		res.Note = "already installed"
		return res, nil
	}
	if err := os.WriteFile(path, []byte(ActionWorkflow), 0o644); err != nil {
		return Result{}, err
	}
	return res, nil
}

// EnsureGitAttributes adds the merge rule, appending to an existing file
// rather than replacing it.
func EnsureGitAttributes(root string) (Result, error) {
	path := filepath.Join(root, ".gitattributes")
	res := Result{Path: relative(root, path)}

	existing, err := os.ReadFile(path)
	if err == nil {
		if strings.Contains(string(existing), ".devtree/tree.yaml") {
			res.Skipped = true
			res.Note = "rule already present"
			return res, nil
		}
		body := string(existing)
		if !strings.HasSuffix(body, "\n") {
			body += "\n"
		}
		return res, os.WriteFile(path, []byte(body+"\n"+GitAttributes), 0o644)
	}
	if !os.IsNotExist(err) {
		return Result{}, err
	}
	return res, os.WriteFile(path, []byte(GitAttributes), 0o644)
}

// gitDir resolves the real git directory.
//
// In a worktree or a submodule, .git is a file holding "gitdir: <path>" rather
// than a directory, and writing hooks into it would silently do nothing.
func gitDir(root string) (string, error) {
	path := filepath.Join(root, ".git")

	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("not a git repository: %s not found — run `git init` first", path)
	}
	if info.IsDir() {
		return path, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(string(data))
	if !strings.HasPrefix(line, "gitdir:") {
		return "", fmt.Errorf("%s is not a git directory pointer", path)
	}
	target := strings.TrimSpace(strings.TrimPrefix(line, "gitdir:"))
	if !filepath.IsAbs(target) {
		target = filepath.Join(root, target)
	}
	return target, nil
}

// relative renders a path the way the user typed the project root, falling
// back to the absolute path when the two share no common prefix.
func relative(root, path string) string {
	if rel, err := filepath.Rel(root, path); err == nil {
		return filepath.ToSlash(rel)
	}
	return path
}
