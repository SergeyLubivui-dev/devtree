package render

import (
	"fmt"
	"strings"

	"github.com/SergeyLubivui-dev/devtree/internal/tree"
)

// Link kinds, in the order a node is searched when nobody says which one they
// want: a pull request is the most specific thing a task can point at, then the
// issue it closes, then the branch the work lives on.
const (
	KindPR     = "pr"
	KindIssue  = "issue"
	KindBranch = "branch"
)

// URL returns the address a node points at, and which kind of address it is.
//
// The empty kind means the caller asked for whatever is most specific. An empty
// URL means the node points at nothing, or at nothing that can be addressed
// without knowing the repository.
func URL(t *tree.Tree, n *tree.Node, kind string) (string, string) {
	repo := strings.TrimRight(t.Repo, "/")
	if repo == "" {
		return "", ""
	}

	switch kind {
	case KindPR:
		if n.PR != "" {
			return fmt.Sprintf("%s/pull/%s", repo, n.PR), KindPR
		}
	case KindIssue:
		if n.Issue != "" {
			return fmt.Sprintf("%s/issues/%s", repo, n.Issue), KindIssue
		}
	case KindBranch:
		if n.Branch != "" {
			return fmt.Sprintf("%s/tree/%s", repo, n.Branch), KindBranch
		}
	default:
		for _, try := range []string{KindPR, KindIssue, KindBranch} {
			if url, found := URL(t, n, try); url != "" {
				return url, found
			}
		}
	}
	return "", ""
}
