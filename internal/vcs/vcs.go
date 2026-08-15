// Package vcs asks git what it already knows.
//
// It is the only place in devtree that runs another program. Everything else
// works on a file and nothing else, and that is worth protecting: a plan you
// can read, validate, and render without a repository is a plan you can use in
// a tarball, a container, or a CI job that never checked anything out.
//
// So this package is small, it is optional, and it is reached from exactly two
// commands: `sync`, which asks which branches were merged, and `history`, which
// asks what the plan used to say. If git is missing or the directory is not a
// repository, the caller gets an error it can explain, not a crash.
package vcs

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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

// Revision is one past version of a file, as git remembers it.
type Revision struct {
	Hash    string
	When    time.Time
	Content []byte
}

// History returns the revisions of one file, oldest first, at most limit of
// them (0 means all).
//
// This is the whole trick behind `devtree history`: the plan has been in the
// repository all along, so its own past is already recorded. Nothing has to be
// tracked, stored, or synced to know what the board looked like in March.
//
// A revision git cannot show — a commit that deleted the file, say — is
// skipped rather than fatal. A gap in a history is not an error.
func History(dir, path string, limit int) ([]Revision, error) {
	args := []string{"log", "--format=%H %at", "--"}
	if limit > 0 {
		args = []string{"log", fmt.Sprintf("-%d", limit), "--format=%H %at", "--"}
	}
	args = append(args, filepath.ToSlash(path))

	out, err := run(dir, args...)
	if err != nil {
		// A repository with no commits at all makes git log fail rather than
		// say nothing. Nothing committed is an empty history, not an error.
		if strings.Contains(err.Error(), "does not have any commits yet") {
			return nil, nil
		}
		return nil, err
	}

	var revisions []Revision
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		hash, stamp, ok := strings.Cut(strings.TrimSpace(line), " ")
		if !ok {
			continue
		}
		seconds, err := strconv.ParseInt(stamp, 10, 64)
		if err != nil {
			continue
		}

		content, err := run(dir, "show", hash+":"+filepath.ToSlash(path))
		if err != nil {
			continue // the commit that removed it, or a rename we did not follow
		}
		revisions = append(revisions, Revision{
			Hash:    hash,
			When:    time.Unix(seconds, 0).UTC(),
			Content: []byte(content),
		})
	}

	// git log is newest first; a history reads better forwards.
	for i, j := 0, len(revisions)-1; i < j; i, j = i+1, j-1 {
		revisions[i], revisions[j] = revisions[j], revisions[i]
	}
	return revisions, nil
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
