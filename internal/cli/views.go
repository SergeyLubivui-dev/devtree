package cli

import (
	"errors"
	"flag"
	"fmt"

	"github.com/SergeyLubivui-dev/devtree/internal/browser"
	"github.com/SergeyLubivui-dev/devtree/internal/render"
	"github.com/SergeyLubivui-dev/devtree/internal/store"
	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

// viewFlags are the questions both the tree and the board accept: what state,
// whose, tagged what, and which part of the plan.
//
// Declaring them once means the two views cannot drift into supporting
// different subsets of the same idea.
type viewFlags struct {
	status string
	owner  string
	tags   string
	root   string
}

func (v *viewFlags) register(fs *flag.FlagSet) {
	fs.StringVar(&v.status, "s", "", "only this status")
	fs.StringVar(&v.status, "status", "", "only this status")
	fs.StringVar(&v.owner, "o", "", "only tasks owned by this person")
	fs.StringVar(&v.owner, "owner", "", "only tasks owned by this person")
	fs.StringVar(&v.tags, "tag", "", "only tasks carrying any of these tags")
	fs.StringVar(&v.tags, "tags", "", "only tasks carrying any of these tags")
	fs.StringVar(&v.root, "root", "", "start from this task instead of the whole plan")
}

// apply turns the flags into a filter and, when --root was given, narrows the
// plan itself to that branch.
func (v *viewFlags) apply(t *tree.Tree) (*tree.Tree, render.Filter, error) {
	var filter render.Filter

	if v.status != "" {
		status, err := tree.ParseStatus(v.status)
		if err != nil {
			return nil, filter, err
		}
		filter.Status = status
	}
	filter.Owner = trimMarker(v.owner, "@")
	filter.Tags = store.SplitList(v.tags)

	if v.root != "" {
		sliced, err := t.Slice(v.root)
		if err != nil {
			return nil, filter, err
		}
		return sliced, filter, nil
	}
	return t, filter, nil
}

// cmdOpen takes you from a task to the thing it points at.
//
// The plan carries a branch, an issue, and a pull request precisely so that the
// jump between planning and doing is short. This makes it one command instead
// of a search through a browser history.
func (a *App) cmdOpen(args []string) error {
	fs := newFlagSet("open")
	printOnly := fs.Bool("print", false, "print the URL instead of opening it")
	issue := fs.Bool("issue", false, "open the issue")
	pr := fs.Bool("pr", false, "open the pull request")
	branch := fs.Bool("branch", false, "open the branch")
	positional, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(positional) == 0 {
		return errors.New("name the task: devtree open ID [--issue|--pr|--branch] [--print]")
	}

	_, t, err := a.load()
	if err != nil {
		return err
	}
	n, ok := t.Get(positional[0])
	if !ok {
		return fmt.Errorf("no such task: %q", positional[0])
	}

	kind := ""
	switch {
	case *pr:
		kind = render.KindPR
	case *issue:
		kind = render.KindIssue
	case *branch:
		kind = render.KindBranch
	}

	url, found := render.URL(t, n, kind)
	if url == "" {
		if t.Repo == "" {
			return errors.New("the plan has no repo URL — add one to .devtree/tree.yaml to make links absolute")
		}
		if kind != "" {
			return fmt.Errorf("%s has no %s", n.ID, kind)
		}
		return fmt.Errorf("%s points at nothing yet — give it a branch, an issue, or a pull request", n.ID)
	}

	if *printOnly {
		fmt.Fprintln(a.Out, url)
		return nil
	}

	if err := browser.Open(url); err != nil {
		// A CI job, a container, a machine over SSH: printing the URL is the
		// useful thing to do, not failing.
		fmt.Fprintf(a.Err, "  ! no browser to open it with\n")
		fmt.Fprintln(a.Out, url)
		return nil
	}

	fmt.Fprintf(a.Out, "  → %s  (%s)\n", url, found)
	return nil
}
