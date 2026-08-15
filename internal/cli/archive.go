package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/store"
	"github.com/SergeyLubivui-dev/devtree/internal/tree"
	"github.com/SergeyLubivui-dev/devtree/internal/vcs"
)

// cmdArchive moves finished work out of the active plan.
//
// A plan that keeps every task ever completed stops being a plan and becomes a
// log: the board fills with columns of done work and the diagram grows a tail
// nobody reads. Archiving keeps the record without keeping the noise.
//
// With no arguments it only says what it would do. Moving a chunk of the plan
// is not something to discover after the fact.
func (a *App) cmdArchive(args []string) error {
	fs := newFlagSet("archive")
	all := fs.Bool("all", false, "archive everything that is finished")
	list := fs.Bool("list", false, "show what is already archived")
	positional, err := parseArgs(fs, args)
	if err != nil {
		return err
	}

	root, t, err := a.load()
	if err != nil {
		return err
	}
	archive, err := store.LoadArchive(root)
	if err != nil {
		return err
	}

	if *list {
		return a.showArchive(archive)
	}

	// Work out what is eligible: everything finished, or just what was named.
	var targets []*tree.Node
	switch {
	case len(positional) > 0:
		for _, id := range positional {
			n, ok := t.Get(id)
			if !ok {
				return fmt.Errorf("no such task: %q", id)
			}
			if !t.Finished(n) {
				return fmt.Errorf("%s still has unfinished work under it — "+
					"finish or drop it first, or archive one of its children", id)
			}
			targets = append(targets, n)
		}
	default:
		targets = t.Archivable()
	}

	if len(targets) == 0 {
		fmt.Fprintln(a.Out, "  nothing is finished yet")
		return nil
	}

	if len(positional) == 0 && !*all {
		fmt.Fprintf(a.Out, "  %d branch(es) of the plan are finished:\n", len(targets))
		for _, n := range targets {
			// The whole subtree moves, dropped work included, so count that
			// rather than the progress total — the number in the hint has to
			// match what actually leaves the plan.
			suffix := ""
			if under := len(t.Subtree(n)) - 1; under > 0 {
				suffix = fmt.Sprintf("  (+%d underneath)", under)
			}
			fmt.Fprintf(a.Out, "    %s %s  (%s)%s\n", n.Status.Icon(), n.Title, n.ID, suffix)
		}
		fmt.Fprintln(a.Out, "\n  archive them with:  devtree archive --all")
		fmt.Fprintln(a.Out, "  or one at a time:   devtree archive ID")
		return nil
	}

	moved := 0
	for _, n := range targets {
		subtree := t.Subtree(n)
		for _, x := range subtree {
			archive.Nodes = append(archive.Nodes, x)
		}
		if _, err := t.Remove(n.ID, true); err != nil {
			return err
		}
		moved += len(subtree)
		fmt.Fprintf(a.Out, "  → archived %s  (%s)\n", n.Title, n.ID)
	}

	if archive.Project == "" {
		archive.Project = t.Project
	}
	if err := store.SaveArchive(root, archive); err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "  %d task(s) moved to %s\n", moved,
		strings.ReplaceAll(store.DirName+"/"+store.ArchiveFileName, "\\", "/"))

	return a.saveAndRender(root, t, false)
}

// showArchive prints what has already left the plan.
func (a *App) showArchive(archive *tree.Tree) error {
	if len(archive.Nodes) == 0 {
		fmt.Fprintln(a.Out, "  the archive is empty")
		return nil
	}
	for _, n := range archive.Nodes {
		title := n.Title
		if title == "" {
			title = n.ID
		}
		context := ""
		if n.Parent != "" {
			context = "  ← " + n.Parent
		}
		fmt.Fprintf(a.Out, "  %s %s  (%s)%s\n", n.Status.Icon(), title, n.ID, context)
	}
	fmt.Fprintf(a.Out, "\n  %d task(s) archived\n", len(archive.Nodes))
	return nil
}

// cmdRestore brings work back from the archive.
//
// A task whose parent is gone comes back as a root rather than being refused:
// the point of a restore is to get the work back in front of you, and a
// re-parent is one command away.
func (a *App) cmdRestore(args []string) error {
	fs := newFlagSet("restore")
	positional, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(positional) == 0 {
		return errors.New("name what to bring back: devtree restore ID [ID...]")
	}

	root, t, err := a.load()
	if err != nil {
		return err
	}
	archive, err := store.LoadArchive(root)
	if err != nil {
		return err
	}

	index := make(map[string]*tree.Node, len(archive.Nodes))
	for _, n := range archive.Nodes {
		index[n.ID] = n
	}

	restored := map[string]bool{}
	for _, id := range positional {
		if _, ok := index[id]; !ok {
			return fmt.Errorf("%q is not in the archive — see `devtree archive --list`", id)
		}

		// Bring the whole subtree back, in the order it was archived.
		wanted := map[string]bool{id: true}
		for _, x := range archive.Nodes {
			if x.Parent != "" && wanted[x.Parent] {
				wanted[x.ID] = true
			}
		}
		for _, x := range archive.Nodes {
			if !wanted[x.ID] || restored[x.ID] {
				continue
			}
			if x.Parent != "" {
				if _, ok := t.Get(x.Parent); !ok && !wanted[x.Parent] {
					fmt.Fprintf(a.Err, "  ! %s came back as a root: its parent %q is gone\n", x.ID, x.Parent)
					x.Parent = ""
				}
			}
			if err := t.Add(x); err != nil {
				return err
			}
			restored[x.ID] = true
			fmt.Fprintf(a.Out, "  ← restored %s  (%s)\n", x.Title, x.ID)
		}
	}

	kept := archive.Nodes[:0]
	for _, x := range archive.Nodes {
		if !restored[x.ID] {
			kept = append(kept, x)
		}
	}
	archive.Nodes = kept

	if err := store.SaveArchive(root, archive); err != nil {
		return err
	}
	return a.saveAndRender(root, t, false)
}

// cmdSync closes tasks whose branch has already been merged.
//
// It proposes and does not act, unless asked. git knows which branches were
// merged, but it does not know which of them were merged *finished*; a branch
// merged behind a feature flag is not a task that is done. So the human keeps
// the decision, and --apply is how they give it.
func (a *App) cmdSync(args []string) error {
	fs := newFlagSet("sync")
	apply := fs.Bool("apply", false, "actually mark the tasks done")
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}

	root, t, err := a.load()
	if err != nil {
		return err
	}

	merged, err := vcs.MergedBranches(root)
	if err != nil {
		switch {
		case errors.Is(err, vcs.ErrNoGit):
			return errors.New("git is not on PATH, so there is nothing to sync from")
		case errors.Is(err, vcs.ErrNotARepository):
			return fmt.Errorf("%s is not a git repository", root)
		}
		return err
	}

	isMerged := make(map[string]bool, len(merged))
	for _, b := range merged {
		isMerged[b] = true
	}

	var ready []*tree.Node
	for _, n := range t.Nodes {
		if n.Branch != "" && isMerged[n.Branch] && n.Status != tree.Done && n.Status != tree.Dropped {
			ready = append(ready, n)
		}
	}

	if len(ready) == 0 {
		fmt.Fprintln(a.Out, "  nothing to sync: no unfinished task points at a merged branch")
		return nil
	}

	for _, n := range ready {
		fmt.Fprintf(a.Out, "  %s → done   %s  (%s)\n", n.Branch, n.Title, n.ID)
	}

	if !*apply {
		fmt.Fprintf(a.Out, "\n  %d task(s) would change. Apply with:  devtree sync --apply\n", len(ready))
		return nil
	}

	for _, n := range ready {
		n.Status = tree.Done
	}
	fmt.Fprintf(a.Out, "\n  %d task(s) marked done\n", len(ready))
	return a.saveAndRender(root, t, false)
}
