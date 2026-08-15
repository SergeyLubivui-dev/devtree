// Package vcs asks git what it already knows.
//
// It is the only place in devtree that runs another program. Everything else
// works on a file and nothing else, and that is worth protecting: a plan you
// can read, validate, and render without a repository is a plan you can use in
// a tarball, a container, or a CI job that never checked anything out.
//
// So this package is small, it is optional, and it is reached only through the
// `sync` command. If git is missing or the directory is not a repository, the
// caller gets an error it can explain, not a crash.
package vcs

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrNoGit is returned when git is not installed or not on PATH.
var ErrNoGit = errors.New("git is not on PATH")

// ErrNotARepository is returned when the directory has no git history to read.
var ErrNotARepository = errors.New("not a git repository")

// MergedBranches lists the branches already merged into HEAD, excluding HEAD's
// own branch.
//
// Merged-and-then-deleted branches do not appear — git no longer has them —
// which is why sync proposes changes instead of making them: the absence of a
// branch is not evidence that the work is unfinished.
func MergedBranches(dir string) ([]string, error) {
	out, err := run(dir, "branch", "--merged", "HEAD", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}

	current, err := CurrentBranch(dir)
	if err != nil {
		current = ""
	}

	var branches []string
	for _, line := range strings.Split(out, "\n") {
		name := strings.TrimSpace(line)
		if name == "" || name == current {
			continue
		}
		branches = append(branches, name)
	}
	return branches, nil
}

// CurrentBranch is the branch HEAD points at, or an empty string when the
// repository is in a detached state.
func CurrentBranch(dir string) (string, error) {
	out, err := run(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(out)
	if name == "HEAD" {
		return "", nil // detached: no branch to name
	}
	return name, nil
}

// run executes a git command and turns its failures into errors a person can
// act on rather than an exit status.
func run(dir string, args ...string) (string, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return "", ErrNoGit
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	out, err := cmd.Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			message := strings.TrimSpace(string(exit.Stderr))
			if strings.Contains(message, "not a git repository") {
				return "", ErrNotARepository
			}
			if message != "" {
				return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), message)
			}
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}
