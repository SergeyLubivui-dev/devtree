package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/render"
	"github.com/SergeyLubivui-dev/devtree/internal/scaffold"
	"github.com/SergeyLubivui-dev/devtree/internal/store"
	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

// ---------------------------------------------------------------------------
// shared helpers
// ---------------------------------------------------------------------------

// load finds the project root and reads the plan. Every command except init
// starts here.
func (a *App) load() (string, *tree.Tree, error) {
	root, err := store.FindRoot(a.Dir)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return "", nil, fmt.Errorf("%w — run `devtree init` to start one", err)
		}
		return "", nil, err
	}
	t, err := store.Load(root)
	if err != nil {
		return "", nil, err
	}
	return root, t, nil
}

// saveAndRender is the tail of every mutating command: persist first, then
// refresh the diagram, so a render failure can never lose the change that was
// already made.
func (a *App) saveAndRender(root string, t *tree.Tree, quiet bool) error {
	if err := store.Save(root, t); err != nil {
		return err
	}
	return a.writeOutputs(root, t, quiet)
}

// writeOutputs injects the generated block into every configured output file.
func (a *App) writeOutputs(root string, t *tree.Tree, quiet bool) error {
	block := render.Block(t)
	for _, output := range t.Outputs {
		path := filepath.Join(root, filepath.FromSlash(output))
		state, err := writeBlock(path, block)
		if err != nil {
			return fmt.Errorf("%s: %w", output, err)
		}
		if !quiet {
			fmt.Fprintf(a.Out, "  %-20s %s\n", output, state)
		}
	}
	return nil
}

// writeBlock updates one file and reports what happened to it.
func writeBlock(path, block string) (string, error) {
	existing, err := os.ReadFile(path)
	switch {
	case os.IsNotExist(err):
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(path, []byte(block+"\n"), 0o644); err != nil {
			return "", err
		}
		return "created", nil
	case err != nil:
		return "", err
	}

	updated, changed := render.Inject(string(existing), block)
	if !changed {
		return "unchanged", nil
	}
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return "", err
	}
	return "updated", nil
}

// report prints one scaffold result.
func (a *App) report(res scaffold.Result) {
	mark := "✔"
	if res.Skipped {
		mark = "·"
	}
	line := fmt.Sprintf("  %s %s", mark, res.Path)
	if res.Note != "" {
		line += "  (" + res.Note + ")"
	}
	fmt.Fprintln(a.Out, line)
}

// ---------------------------------------------------------------------------
// commands
// ---------------------------------------------------------------------------

func (a *App) cmdInit(args []string) error {
	fs := newFlagSet("init")
	project := fs.String("project", "", "project name")
	repo := fs.String("repo", "", "repository URL, e.g. https://github.com/acme/pay")
	outputs := fs.String("outputs", tree.DefaultOutput, "comma-separated files to render into")
	hook := fs.Bool("hook", false, "install the pre-commit hook")
	action := fs.Bool("action", false, "install the GitHub Actions workflow")
	empty := fs.Bool("empty", false, "skip the sample tasks")
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}

	root, err := filepath.Abs(a.Dir)
	if err != nil {
		return err
	}
	if store.Exists(root) {
		return fmt.Errorf("%s already exists — nothing to do", filepath.ToSlash(filepath.Join(store.DirName, store.FileName)))
	}

	name := strings.TrimSpace(*project)
	if name == "" {
		name = filepath.Base(root)
	}

	t := tree.New(name)
	t.Repo = strings.TrimRight(strings.TrimSpace(*repo), "/")
	if outs := store.SplitList(*outputs); len(outs) > 0 {
		t.Outputs = outs
	}

	if !*empty {
		// A brand new plan with nothing in it renders an empty box, so seed
		// the shape people always end up building anyway: one milestone with
		// one task under it.
		if err := t.Add(&tree.Node{ID: "mvp", Title: "MVP", Status: tree.InProgress}); err != nil {
			return err
		}
		if err := t.Add(&tree.Node{ID: "first-task", Title: "First task", Status: tree.Todo, Parent: "mvp"}); err != nil {
			return err
		}
	}

	if err := store.Save(root, t); err != nil {
		return err
	}
	a.report(scaffold.Result{Path: filepath.ToSlash(filepath.Join(store.DirName, store.FileName))})

	attrs, err := scaffold.EnsureGitAttributes(root)
	if err != nil {
		return err
	}
	a.report(attrs)

	if err := a.writeOutputs(root, t, false); err != nil {
		return err
	}

	if *hook {
		res, err := scaffold.InstallHook(root)
		if err != nil {
			return err
		}
		a.report(res)
	}
	if *action {
		res, err := scaffold.InstallAction(root)
		if err != nil {
			return err
		}
		a.report(res)
	}

	fmt.Fprintf(a.Out, "\nReady. Next:  devtree add \"Task title\" -p mvp\n")
	return nil
}

func (a *App) cmdAdd(args []string) error {
	fs := newFlagSet("add")
	nf := &nodeFlags{}
	nf.register(fs, false, true)
	positional, err := parseArgs(fs, args)
	if err != nil {
		return err
	}

	// Joining the positionals means an unquoted title still works: what the
	// shell split into words is put back together.
	title := strings.TrimSpace(strings.Join(positional, " "))
	if title == "" {
		return errors.New("give the task a title: devtree add \"Task title\" [-p parent]")
	}

	root, t, err := a.load()
	if err != nil {
		return err
	}
	if nf.parent != "" {
		if _, ok := t.Get(nf.parent); !ok {
			return fmt.Errorf("no such parent: %q", nf.parent)
		}
	}

	n := &tree.Node{
		ID:     nf.id,
		Title:  title,
		Status: tree.Todo,
		Parent: nf.parent,
		Branch: nf.branch,
		Issue:  trimMarker(nf.issue, "#"),
		PR:     trimMarker(nf.pr, "#"),
		Owner:  trimMarker(nf.owner, "@"),
		Tags:   store.SplitList(nf.tags),
		Note:   nf.note,
	}
	if nf.status != "" {
		status, err := tree.ParseStatus(nf.status)
		if err != nil {
			return err
		}
		n.Status = status
	}
	if n.ID == "" {
		n.ID = t.NextID(title)
	}
	if err := t.Add(n); err != nil {
		return err
	}

	fmt.Fprintf(a.Out, "  + %s  (%s)\n", n.Title, n.ID)
	return a.saveAndRender(root, t, false)
}

func (a *App) cmdSet(args []string) error {
	fs := newFlagSet("set")
	nf := &nodeFlags{}
	nf.register(fs, true, false)
	positional, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(positional) == 0 {
		return errors.New("name the task: devtree set ID [flags]")
	}

	root, t, err := a.load()
	if err != nil {
		return err
	}
	n, ok := t.Get(positional[0])
	if !ok {
		return fmt.Errorf("no such task: %q", positional[0])
	}

	if wasSet(fs, "title") {
		n.Title = nf.title
	}
	if wasSet(fs, "s", "status") {
		status, err := tree.ParseStatus(nf.status)
		if err != nil {
			return err
		}
		n.Status = status
	}
	if wasSet(fs, "b", "branch") {
		n.Branch = nf.branch
	}
	if wasSet(fs, "i", "issue") {
		n.Issue = trimMarker(nf.issue, "#")
	}
	if wasSet(fs, "pr") {
		n.PR = trimMarker(nf.pr, "#")
	}
	if wasSet(fs, "o", "owner") {
		n.Owner = trimMarker(nf.owner, "@")
	}
	if wasSet(fs, "tags") {
		n.Tags = store.SplitList(nf.tags)
	}
	if wasSet(fs, "n", "note") {
		n.Note = nf.note
	}
	// Re-parenting goes last: Move validates and rolls itself back, so a
	// rejected move leaves the field edits above intact and unsaved.
	if wasSet(fs, "p", "parent") {
		if err := t.Move(n.ID, nf.parent); err != nil {
			return err
		}
	}
	if err := t.Build(); err != nil {
		return err
	}

	fmt.Fprintf(a.Out, "  ~ %s  (%s) → %s\n", n.Title, n.ID, n.Status.Label())
	return a.saveAndRender(root, t, false)
}

func (a *App) cmdDone(args []string) error {
	fs := newFlagSet("done")
	positional, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(positional) == 0 {
		return errors.New("name the task: devtree done ID [ID...]")
	}

	root, t, err := a.load()
	if err != nil {
		return err
	}
	for _, id := range positional {
		n, ok := t.Get(id)
		if !ok {
			return fmt.Errorf("no such task: %q", id)
		}
		n.Status = tree.Done
		fmt.Fprintf(a.Out, "  %s %s  (%s)\n", tree.Done.Icon(), n.Title, n.ID)
	}
	return a.saveAndRender(root, t, false)
}

func (a *App) cmdMove(args []string) error {
	fs := newFlagSet("mv")
	positional, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(positional) < 2 {
		return errors.New("usage: devtree mv ID PARENT|root")
	}

	root, t, err := a.load()
	if err != nil {
		return err
	}
	if err := t.Move(positional[0], positional[1]); err != nil {
		return err
	}

	fmt.Fprintf(a.Out, "  → %s now sits under %s\n", positional[0], positional[1])
	return a.saveAndRender(root, t, false)
}

func (a *App) cmdRemove(args []string) error {
	fs := newFlagSet("rm")
	cascade := fs.Bool("cascade", false, "delete the whole subtree")
	positional, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(positional) == 0 {
		return errors.New("name the task: devtree rm ID [--cascade]")
	}

	root, t, err := a.load()
	if err != nil {
		return err
	}
	removed, err := t.Remove(positional[0], *cascade)
	if err != nil {
		return err
	}

	fmt.Fprintf(a.Out, "  − removed %d task(s)\n", removed)
	return a.saveAndRender(root, t, false)
}

func (a *App) cmdList(args []string) error {
	fs := newFlagSet("ls")
	statusFlag := fs.String("s", "", "show only tasks with this status")
	fs.StringVar(statusFlag, "status", "", "show only tasks with this status")
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}

	_, t, err := a.load()
	if err != nil {
		return err
	}

	var filter tree.Status
	if *statusFlag != "" {
		filter, err = tree.ParseStatus(*statusFlag)
		if err != nil {
			return err
		}
	}

	fmt.Fprintf(a.Out, "%s\n\n", t.Project)
	fmt.Fprint(a.Out, render.ASCII(t, filter))

	done, total := t.Totals()
	fmt.Fprintf(a.Out, "\n%s  %d/%d\n", render.ProgressBar(done, total), done, total)
	return nil
}

func (a *App) cmdRender(args []string) error {
	fs := newFlagSet("render")
	quiet := fs.Bool("quiet", false, "print nothing on success")
	file := fs.String("file", "", "render into this file instead of the configured outputs")
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}

	root, t, err := a.load()
	if err != nil {
		return err
	}
	if *file != "" {
		// A one-off override, not a saved setting: the plan on disk keeps its
		// own outputs list.
		t.Outputs = store.SplitList(*file)
	}
	return a.writeOutputs(root, t, *quiet)
}

func (a *App) cmdCheck(args []string) error {
	fs := newFlagSet("check")
	strict := fs.Bool("strict", false, "treat warnings as a failure")
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}

	// Loading is the check: parsing and Build together cover unknown fields,
	// bad statuses, duplicate ids, missing parents, and cycles.
	_, t, err := a.load()
	if err != nil {
		return err
	}

	warnings := inspect(t)
	fmt.Fprintf(a.Out, "  ✔ plan is valid: %d task(s), %d root(s)  (%s)\n",
		len(t.Nodes), len(t.Roots()), filepath.ToSlash(filepath.Join(store.DirName, store.FileName)))
	for _, w := range warnings {
		fmt.Fprintf(a.Err, "  ! %s\n", w)
	}

	if *strict && len(warnings) > 0 {
		return fmt.Errorf("%d warning(s) and --strict was set", len(warnings))
	}
	return nil
}

// inspect collects the problems that are worth mentioning but not worth
// refusing to save: a plan can be internally consistent and still say
// something its author did not mean.
func inspect(t *tree.Tree) []string {
	var warnings []string
	for _, n := range t.Nodes {
		if strings.TrimSpace(n.Title) == "" {
			warnings = append(warnings, fmt.Sprintf("%s has no title", n.ID))
		}
		if n.Status == tree.Blocked && strings.TrimSpace(n.Note) == "" {
			warnings = append(warnings, fmt.Sprintf("%s is blocked with no note saying why", n.ID))
		}
		if n.Status == tree.Done {
			for _, c := range n.Children() {
				if c.Status != tree.Done && c.Status != tree.Dropped {
					warnings = append(warnings, fmt.Sprintf(
						"%s is marked done, but its subtask %s is still %s", n.ID, c.ID, c.Status.Label()))
				}
			}
		}
	}
	return warnings
}

func (a *App) cmdInstall(args []string) error {
	fs := newFlagSet("install")
	positional, err := parseArgs(fs, args)
	if err != nil {
		return err
	}

	what := "all"
	if len(positional) > 0 {
		what = positional[0]
	}

	root, _, err := a.load()
	if err != nil {
		return err
	}

	install := func(fn func(string) (scaffold.Result, error)) error {
		res, err := fn(root)
		if err != nil {
			return err
		}
		a.report(res)
		return nil
	}

	switch what {
	case "hook":
		return install(scaffold.InstallHook)
	case "action":
		return install(scaffold.InstallAction)
	case "all":
		if err := install(scaffold.InstallHook); err != nil {
			return err
		}
		return install(scaffold.InstallAction)
	default:
		return fmt.Errorf("unknown target %q: expected hook, action, or all", what)
	}
}

// cmdOutputs prints the output files on one line. The pre-commit hook uses it
// to stage exactly the files that were rendered, whatever they were renamed to.
func (a *App) cmdOutputs(args []string) error {
	_, t, err := a.load()
	if err != nil {
		return err
	}
	fmt.Fprintln(a.Out, strings.Join(t.Outputs, " "))
	return nil
}
