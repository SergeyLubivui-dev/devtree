package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallHookBacksUpWhatWasThere(t *testing.T) {
	root := t.TempDir()
	hooks := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}

	existing := "#!/bin/sh\nnpm test\n"
	if err := os.WriteFile(filepath.Join(hooks, "pre-commit"), []byte(existing), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := InstallHook(root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Note, "backup") {
		t.Errorf("the user should be told about the backup, got %q", res.Note)
	}

	backup, err := os.ReadFile(filepath.Join(hooks, "pre-commit.devtree-backup"))
	if err != nil {
		t.Fatalf("the previous hook was destroyed: %v", err)
	}
	if string(backup) != existing {
		t.Errorf("the backup does not match the original:\n%s", backup)
	}

	installed, err := os.ReadFile(filepath.Join(hooks, "pre-commit"))
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != HookScript {
		t.Error("the new hook was not written")
	}
}

func TestInstallHookIsIdempotent(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := InstallHook(root); err != nil {
		t.Fatal(err)
	}
	res, err := InstallHook(root)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Skipped {
		t.Error("installing the same hook twice should be a no-op")
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "hooks", "pre-commit.devtree-backup")); err == nil {
		t.Error("devtree backed up its own hook")
	}
}

func TestInstallHookFollowsAGitDirPointer(t *testing.T) {
	// In a worktree or a submodule, .git is a file rather than a directory.
	// Writing hooks into it would silently do nothing.
	root := t.TempDir()
	real := filepath.Join(t.TempDir(), "actual-git-dir")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: "+real+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := InstallHook(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(real, "hooks", "pre-commit")); err != nil {
		t.Fatalf("the hook did not land in the real git directory: %v", err)
	}
}

func TestInstallHookNeedsARepository(t *testing.T) {
	_, err := InstallHook(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "git init") {
		t.Fatalf("got %v, want a message pointing at `git init`", err)
	}
}

func TestEnsureGitAttributesAppendsRatherThanReplaces(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".gitattributes")
	existing := "*.png binary\n"
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := EnsureGitAttributes(root); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "*.png binary") {
		t.Errorf("the existing rules were replaced:\n%s", got)
	}
	if !strings.Contains(string(got), "merge=union") {
		t.Errorf("the devtree rule is missing:\n%s", got)
	}

	// A second call must not add the rule twice.
	if res, err := EnsureGitAttributes(root); err != nil || !res.Skipped {
		t.Fatalf("second call: skipped=%v err=%v", res.Skipped, err)
	}
	after, _ := os.ReadFile(path)
	if strings.Count(string(after), "merge=union") != 1 {
		t.Errorf("the rule was written twice:\n%s", after)
	}
}

func TestInstallActionWritesTheWorkflow(t *testing.T) {
	root := t.TempDir()

	res, err := InstallAction(root)
	if err != nil {
		t.Fatal(err)
	}
	if res.Path != ".github/workflows/devtree.yml" {
		t.Errorf("path reported as %q", res.Path)
	}

	got, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "devtree.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"devtree check", "devtree render", "go install github.com/SergeyLubivui-dev/devtree@latest"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("the workflow is missing %q", want)
		}
	}

	if res, err := InstallAction(root); err != nil || !res.Skipped {
		t.Fatalf("second call: skipped=%v err=%v", res.Skipped, err)
	}
}

func TestInstallGitLabLeavesTheProjectsOwnPipelineAlone(t *testing.T) {
	root := t.TempDir()

	// A project's .gitlab-ci.yml belongs to the project. Scaffolding writes a
	// separate file and says how to include it.
	own := "stages: [test]\n"
	if err := os.WriteFile(filepath.Join(root, ".gitlab-ci.yml"), []byte(own), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := InstallGitLab(root)
	if err != nil {
		t.Fatal(err)
	}
	if res.Path != ".gitlab/devtree.yml" {
		t.Errorf("wrote %q", res.Path)
	}
	if !strings.Contains(res.Note, "include") {
		t.Errorf("the user should be told to include it, got %q", res.Note)
	}

	after, err := os.ReadFile(filepath.Join(root, ".gitlab-ci.yml"))
	if err != nil || string(after) != own {
		t.Error("the project's own pipeline was touched")
	}

	body, err := os.ReadFile(filepath.Join(root, ".gitlab", "devtree.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"devtree check --strict", "devtree render", "ghcr.io/sergeylubivui-dev/devtree"} {
		if !strings.Contains(string(body), want) {
			t.Errorf("the job is missing %q", want)
		}
	}

	if res, err := InstallGitLab(root); err != nil || !res.Skipped {
		t.Fatalf("second call: skipped=%v err=%v", res.Skipped, err)
	}
}
