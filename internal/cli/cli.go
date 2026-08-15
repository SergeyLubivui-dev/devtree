// Package cli is the outermost layer: it parses arguments, calls the domain
// and storage packages, and owns every line printed to the terminal.
//
// The layers below never print and never call os.Exit. That is why App takes
// its working directory and its two writers as fields — a test can run a whole
// command against a temporary directory and read back exactly what a user
// would have seen.
package cli

import (
	"fmt"
	"io"
	"os"
)

// Version is stamped at build time with
// -ldflags "-X github.com/SergeyLubivui-dev/devtree/internal/cli.Version=v1.2.3".
//
// The default is what a plain `go build` reports, so it tracks the last
// release rather than sitting at whatever the first one happened to be.
var Version = "0.5.0"

// App holds everything a command needs from the outside world.
type App struct {
	Dir string    // working directory the command was invoked from
	Out io.Writer // normal output
	Err io.Writer // errors and warnings
}

// Run wires App to the real process and returns the exit code, leaving the
// os.Exit call to main so nothing below it can end the program mid-flight.
func Run(args []string, stdout, stderr io.Writer) int {
	dir, err := os.Getwd()
	if err != nil {
		dir = "."
	}
	app := &App{Dir: dir, Out: stdout, Err: stderr}
	return app.Execute(args)
}

// Execute dispatches one command and returns a process exit code:
//
//	0  success
//	1  the command failed
//	2  the command line itself was wrong
func (a *App) Execute(args []string) int {
	if len(args) == 0 {
		a.printUsage()
		return 2
	}

	name, rest := args[0], args[1:]

	var err error
	switch name {
	case "init":
		err = a.cmdInit(rest)
	case "add":
		err = a.cmdAdd(rest)
	case "set":
		err = a.cmdSet(rest)
	case "done":
		err = a.cmdDone(rest)
	case "mv", "move":
		err = a.cmdMove(rest)
	case "rm", "remove":
		err = a.cmdRemove(rest)
	case "ls", "tree", "list":
		err = a.cmdList(rest)
	case "board", "b":
		err = a.cmdBoard(rest)
	case "open":
		err = a.cmdOpen(rest)
	case "archive":
		err = a.cmdArchive(rest)
	case "restore":
		err = a.cmdRestore(rest)
	case "sync":
		err = a.cmdSync(rest)
	case "history":
		err = a.cmdHistory(rest)
	case "serve", "edit":
		err = a.cmdServe(rest)
	case "render":
		err = a.cmdRender(rest)
	case "check":
		err = a.cmdCheck(rest)
	case "install":
		err = a.cmdInstall(rest)
	case "outputs":
		err = a.cmdOutputs(rest)
	case "version", "--version", "-v":
		fmt.Fprintln(a.Out, "devtree "+Version)
	case "help", "--help", "-h":
		a.printUsage()
	default:
		fmt.Fprintf(a.Err, "devtree: unknown command %q — run `devtree help` for the list\n", name)
		return 2
	}

	if err != nil {
		fmt.Fprintln(a.Err, "devtree: "+err.Error())
		return 1
	}
	return 0
}

func (a *App) printUsage() {
	fmt.Fprint(a.Out, `devtree `+Version+` — tree-shaped development planning that lives in your repository.

The plan is a file: .devtree/tree.yaml. It sits next to the code, it is
reviewed in the pull request, and it renders as a Mermaid diagram that GitHub
draws right in the viewport.

USAGE
  devtree <command> [flags] [arguments]

COMMANDS
  init      Create .devtree/tree.yaml and the first diagram
  add       Add a task
  set       Change fields on a task
  done      Mark a task done
  mv        Re-parent a task
  rm        Delete a task
  ls        Print the tree in the terminal
  board     Print the work grouped by status, like a board
  open      Open what a task points at: its pull request, issue, or branch
  archive   Move finished work out of the active plan
  restore   Bring archived work back
  sync      Close tasks whose branch has already been merged
  history   Show how the plan changed, from the repository's own history
  serve     Open the local editor: the plan on the left, drawn on the right
  render    Regenerate the diagram
  check     Validate the plan, for CI and hooks
  install   Install the pre-commit hook, the GitHub Action, or the GitLab job
  outputs   Print the files the diagram is written to
  version   Print the version
  help      Print this message

INIT
  devtree init [--project NAME] [--repo URL] [--outputs FILES]
               [--hook] [--action] [--empty]

    --project  Project name shown above the diagram (default: directory name)
    --repo     Repository URL, used to build issue and pull request links
    --outputs  Comma-separated files to render into (default: TREE.md)
    --hook     Also install the pre-commit hook
    --action   Also install the GitHub Actions workflow
    --empty    Skip the two sample tasks

ADD
  devtree add "Title" [-p PARENT] [-s STATUS] [-b BRANCH] [-i ISSUE]
                      [--pr PR] [-o OWNER] [--tags a,b] [-n NOTE] [--id ID]

    Without -p the task becomes a root. The id is derived from the title
    unless --id says otherwise.

SET
  devtree set ID [--title TITLE] [-s STATUS] [-p PARENT] [-b BRANCH]
                 [-i ISSUE] [--pr PR] [-o OWNER] [--tags a,b] [-n NOTE]

    Only the flags you pass are changed; passing an empty value clears a
    field, as in: devtree set auth -b ""

OTHER COMMANDS
  devtree done ID                 Mark a task done
  devtree mv ID PARENT|root       Re-parent a task
  devtree rm ID [--cascade]       Delete a task; --cascade takes its subtree
  devtree ls [filters]            Print the tree
  devtree board [filters]         Print the work grouped by status
  devtree open ID [--print]       Open its pull request, issue, or branch
  devtree render [--file F] [--root ID] [--quiet]
  devtree check [--strict]        --strict turns warnings into a failure
  devtree install hook|action|gitlab|all

FILTERS
  ls and board take the same four, and they combine:

    -s STATUS       only this status
    -o OWNER        only this person's tasks
    --tag a,b       only tasks carrying any of these tags
    --root ID       start from this task instead of the whole plan

  render takes --root too, so one branch of a plan can have a picture of its
  own:  devtree render --root mvp --file docs/mvp.svg

FINISHED WORK
  devtree archive                 Show which branches of the plan are finished
  devtree archive --all           Move all of them to .devtree/archive.yaml
  devtree archive ID [ID...]      Move only these, with everything under them
  devtree archive --list          Show what is already archived
  devtree restore ID [ID...]      Bring archived work back into the plan

  devtree sync                    List tasks whose branch is already merged
  devtree sync --apply            ...and mark them done

  devtree history [--limit N]     Read past versions of the plan out of git
                                  and show how far along it was each time

THE EDITOR
  devtree serve [--port 9312] [--open]
        A two-pane page on 127.0.0.1: edit the plan on the left, watch it
        drawn on the right. Nothing is stored anywhere but the plan file, so
        the editor and the terminal can be used in the same minute.

OUTPUT FILES
  The name decides the drawing. TREE.md and README.md get the Mermaid block;
  an .svg file gets devtree's own drawing, as a tree or as a board:

    docs/tree.svg        the tree, light palette
    docs/tree-dark.svg   the tree, dark palette
    docs/board.svg       the board
    docs/plan.board.svg  the board, named something else
    docs/plan.html       a page you can click: links, filters, tooltips

STATUSES
  `+statusHelp()+`

EXAMPLES
  devtree init --project "Payment Gateway" --repo https://github.com/acme/pay --hook
  devtree add "Authentication" -p mvp -b feat/auth -i 12 -o ann -s wip
  devtree add "OAuth providers" -p authentication
  devtree done oauth-providers
`)
}
