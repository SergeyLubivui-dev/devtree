package tree

import (
	"fmt"
	"sort"
	"strings"
)

// Status is where a node stands. The set is intentionally small: five states
// fit on one line of a legend, and anything finer belongs in the note field or
// in the linked issue.
type Status string

const (
	Todo       Status = "todo"
	InProgress Status = "in_progress"
	Blocked    Status = "blocked"
	Done       Status = "done"
	Dropped    Status = "dropped"
)

// Statuses lists every status in the order a reader expects to meet them:
// not started, moving, stuck, finished, abandoned.
var Statuses = []Status{Todo, InProgress, Blocked, Done, Dropped}

// icons and labels are kept next to each other so a new status cannot be added
// with an icon but no label, or the other way around.
var (
	icons = map[Status]string{
		Todo:       "☐",
		InProgress: "◐",
		Blocked:    "⛔",
		Done:       "✔",
		Dropped:    "✖",
	}

	labels = map[Status]string{
		Todo:       "not started",
		InProgress: "in progress",
		Blocked:    "blocked",
		Done:       "done",
		Dropped:    "dropped",
	}
)

// aliases accepts the shorthand people actually type. The canonical spelling
// is what lands in the file, so the stored plan stays uniform no matter which
// spelling was used at the prompt.
var aliases = map[string]Status{
	"todo": Todo, "t": Todo, "new": Todo, "open": Todo,
	"in_progress": InProgress, "in-progress": InProgress, "wip": InProgress,
	"doing": InProgress, "p": InProgress, "active": InProgress,
	"blocked": Blocked, "b": Blocked, "block": Blocked, "stuck": Blocked,
	"done": Done, "d": Done, "ok": Done, "finished": Done,
	"dropped": Dropped, "x": Dropped, "drop": Dropped, "cancel": Dropped, "wontfix": Dropped,
}

// ParseStatus resolves user input to a canonical status. The error names the
// legal values because this is the mistake people make most often, and a list
// beats a lookup in the docs.
func ParseStatus(s string) (Status, error) {
	if v, ok := aliases[strings.ToLower(strings.TrimSpace(s))]; ok {
		return v, nil
	}
	return "", fmt.Errorf("unknown status %q (valid: %s)", s, StatusList())
}

// StatusList renders the canonical statuses for help text and error messages.
func StatusList() string {
	names := make([]string, 0, len(Statuses))
	for _, s := range Statuses {
		names = append(names, string(s))
	}
	return strings.Join(names, ", ")
}

// AliasList renders the accepted shorthand, sorted so help output is stable.
func AliasList() string {
	names := make([]string, 0, len(aliases))
	for a, s := range aliases {
		if string(s) != a {
			names = append(names, a)
		}
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// Valid reports whether s is one of the five canonical statuses.
func (s Status) Valid() bool {
	_, ok := icons[s]
	return ok
}

// Icon is the glyph used in diagrams and terminal output.
func (s Status) Icon() string { return icons[s] }

// Label is the human-readable name used in legends and tables.
func (s Status) Label() string { return labels[s] }

func (s Status) String() string { return string(s) }
