package cli

import (
	"errors"
	"fmt"

	"github.com/SergeyLubivui-dev/devtree/internal/browser"
	"github.com/SergeyLubivui-dev/devtree/internal/serve"
)

// cmdServe opens the local editor.
//
// This is the one command that starts something long-lived, and the one that
// opens a port. Both are worth being explicit about: it binds to the loopback
// interface only, it stores nothing, and every edit lands in the same
// .devtree/tree.yaml the command line writes — so the editor and the terminal
// can be used in the same minute without either one losing a change.
func (a *App) cmdServe(args []string) error {
	fs := newFlagSet("serve")
	port := fs.Int("port", serve.DefaultPort, "port to listen on")
	host := fs.String("host", serve.DefaultHost, "interface to bind; 0.0.0.0 is needed inside a container")
	open := fs.Bool("open", false, "open the editor in a browser")
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}

	root, t, err := a.load()
	if err != nil {
		return err
	}

	url := serve.URL(*port)
	fmt.Fprintf(a.Out, "  %s — %s\n", t.Project, url)
	fmt.Fprintf(a.Out, "  editing %s\n", root)
	fmt.Fprintln(a.Out, "  stop with ctrl-c")

	// Binding anywhere but loopback hands an editor with no authentication to
	// whoever can reach the machine. Inside a container that is the point;
	// anywhere else it is worth saying out loud.
	if *host != serve.DefaultHost {
		fmt.Fprintf(a.Err, "  ! listening on %s — anyone who can reach this machine can edit the plan\n", *host)
	}

	if *open {
		if err := browser.Open(url); err != nil && !errors.Is(err, browser.ErrHeadless) {
			fmt.Fprintf(a.Err, "  ! could not open a browser: %v\n", err)
		}
	}

	server := &serve.Server{Root: root, Log: a.Err}
	return server.ListenAndServe(*host, *port)
}
