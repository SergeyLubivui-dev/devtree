// Command devtree keeps a development plan inside the repository and renders
// it as a Mermaid diagram that GitHub and GitLab draw natively.
//
// The plan lives in .devtree/tree.yaml: it is versioned with the code,
// reviewed in the pull request, and merged line by line. See README.md for the
// full tour, or run `devtree help`.
//
// This file is deliberately thin. Everything it could do belongs in a package
// that can be tested without a process: the tree itself in internal/tree, the
// file format in internal/store, the output in internal/render, and the
// command line in internal/cli.
package main

import (
	"os"

	"github.com/SergeyLubivui-dev/devtree/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
