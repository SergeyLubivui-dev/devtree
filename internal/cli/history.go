package cli

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/render"
	"github.com/SergeyLubivui-dev/devtree/internal/store"
	"github.com/SergeyLubivui-dev/devtree/internal/tree"
	"github.com/SergeyLubivui-dev/devtree/internal/vcs"
)

// snapshot is what one past commit said about the plan.
type snapshot struct {
	date  string
	done  int
	total int
	added int // tasks that appeared in this revision
}

// cmdHistory shows how the plan changed over time.
//
// Nothing is tracked to make this work. The plan has been in the repository all
// along, so its own past is already recorded, commit by commit — reading it
// back is a matter of asking git for old versions of one file and counting.
// That is the argument for keeping a plan in a repository, made by the plan.
func (a *App) cmdHistory(args []string) error {
	fs := newFlagSet("history")
	limit := fs.Int("limit", 25, "how many revisions of the plan to read")
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}

	root, t, err := a.load()
	if err != nil {
		return err
	}

	relative := path.Join(store.DirName, store.FileName)
	revisions, err := vcs.History(root, relative, *limit)
	if err != nil {
		switch {
		case errors.Is(err, vcs.ErrNoGit):
			return errors.New("git is not on PATH, so there is no history to read")
		case errors.Is(err, vcs.ErrNotARepository):
			return fmt.Errorf("%s is not a git repository", root)
		}
		return err
	}
	if len(revisions) == 0 {
		return fmt.Errorf("%s has never been committed — there is no history yet", relative)
	}

	var (
		points []snapshot
		before int
	)
	skipped := 0
	for _, rev := range revisions {
		past, err := store.Parse(rev.Content)
		if err != nil {
			// A revision written by an older devtree may use a field this one
			// no longer knows. A gap in a history is not a reason to refuse
			// the rest of it.
			skipped++
			continue
		}
		if err := past.Build(); err != nil {
			skipped++
			continue
		}

		done, total := past.Totals()
		points = append(points, snapshot{
			date:  rev.When.Format("2006-01-02"),
			done:  done,
			total: total,
			added: total - before,
		})
		before = total
	}

	if len(points) == 0 {
		return fmt.Errorf("none of the %d revisions could be read", len(revisions))
	}

	fmt.Fprintf(a.Out, "%s — %d revision(s) of the plan\n\n", t.Project, len(points))

	previous := 0
	for i, p := range points {
		delta := ""
		if i > 0 {
			if change := p.done - previous; change > 0 {
				delta = fmt.Sprintf("  +%d done", change)
			}
		}
		if p.added > 0 && i > 0 {
			delta += fmt.Sprintf("  +%d planned", p.added)
		}
		fmt.Fprintf(a.Out, "%s  %s  %s%s\n",
			p.date, render.ProgressBar(p.done, p.total),
			fmt.Sprintf("%d/%d", p.done, p.total), delta)
		previous = p.done
	}

	first, last := points[0], points[len(points)-1]
	fmt.Fprintf(a.Out, "\n%d of %d done", last.done, last.total)
	if gained := last.done - first.done; gained > 0 {
		fmt.Fprintf(a.Out, ", up %d since %s", gained, first.date)
	}
	fmt.Fprintf(a.Out, "\n%s\n", statusBreakdown(t))

	if skipped > 0 {
		fmt.Fprintf(a.Err, "  ! %d revision(s) could not be read and were skipped\n", skipped)
	}
	return nil
}

// statusBreakdown ends the history on where things stand today: a line of
// counts is the natural answer to "and now?".
func statusBreakdown(t *tree.Tree) string {
	counts := map[tree.Status]int{}
	for _, n := range t.Nodes {
		counts[n.Status]++
	}

	var parts []string
	for _, s := range tree.Statuses {
		if counts[s] > 0 {
			parts = append(parts, fmt.Sprintf("%s %d %s", s.Icon(), counts[s], s.Label()))
		}
	}
	return strings.Join(parts, "   ")
}
