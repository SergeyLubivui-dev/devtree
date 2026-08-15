package cli

import (
	"flag"
	"io"
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

// newFlagSet returns a flag set that reports errors as values instead of
// writing to stderr and exiting. Every message the user sees comes from
// Execute, in one voice.
func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	return fs
}

// nodeFlags is the field set shared by `add` and `set`. Declaring it once
// means the two commands cannot drift apart — a flag added for one is
// available to the other, spelled the same way.
type nodeFlags struct {
	title  string
	parent string
	status string
	branch string
	issue  string
	pr     string
	owner  string
	tags   string
	note   string
	id     string
}

// register binds the flags. Short and long names point at the same variable,
// so -o and --owner are genuinely interchangeable.
func (nf *nodeFlags) register(fs *flag.FlagSet, withTitle, withID bool) {
	if withTitle {
		fs.StringVar(&nf.title, "title", "", "task title")
	}
	if withID {
		fs.StringVar(&nf.id, "id", "", "explicit id instead of one derived from the title")
	}
	fs.StringVar(&nf.parent, "p", "", "id of the parent task")
	fs.StringVar(&nf.parent, "parent", "", "id of the parent task")
	fs.StringVar(&nf.status, "s", "", "status: "+tree.StatusList())
	fs.StringVar(&nf.status, "status", "", "status: "+tree.StatusList())
	fs.StringVar(&nf.branch, "b", "", "branch name")
	fs.StringVar(&nf.branch, "branch", "", "branch name")
	fs.StringVar(&nf.issue, "i", "", "issue number")
	fs.StringVar(&nf.issue, "issue", "", "issue number")
	fs.StringVar(&nf.pr, "pr", "", "pull request number")
	fs.StringVar(&nf.owner, "o", "", "who owns the task")
	fs.StringVar(&nf.owner, "owner", "", "who owns the task")
	fs.StringVar(&nf.tags, "tags", "", "comma-separated tags")
	fs.StringVar(&nf.note, "n", "", "free-form note")
	fs.StringVar(&nf.note, "note", "", "free-form note")
}

// parseArgs parses flags wherever they appear, before or after the positional
// arguments.
//
// The standard library stops at the first non-flag argument, which would make
//
//	devtree add "Authentication" -p mvp
//
// silently drop -p — the shape people type first and the one that would fail
// most quietly. Parsing in a loop, peeling off one positional at a time, keeps
// both orders working.
func parseArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return positional, nil
		}
		positional = append(positional, rest[0])
		args = rest[1:]
	}
}

// wasSet reports whether the user actually passed one of these flags.
//
// `set` needs the difference between "not mentioned" and "set to empty":
// the first leaves a field alone, the second clears it.
func wasSet(fs *flag.FlagSet, names ...string) bool {
	found := false
	fs.Visit(func(f *flag.Flag) {
		for _, n := range names {
			if f.Name == n {
				found = true
			}
		}
	})
	return found
}

// statusHelp formats the status vocabulary for the help screen.
func statusHelp() string {
	return tree.StatusList() + "\n  Shorthand accepted: " + tree.AliasList()
}

// trimMarker strips the decoration people paste along with a number or a name:
// "#12" is an issue, "@ann" is an owner, and both should store bare.
func trimMarker(value, marker string) string {
	return strings.TrimPrefix(strings.TrimSpace(value), marker)
}
