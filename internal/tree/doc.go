// Package tree is the domain layer: the nodes of a development plan, the
// parent/child relationships between them, and the rules that keep those
// relationships a tree rather than a tangle.
//
// It is the innermost layer of the program and depends on nothing but the
// standard library. Storage (internal/store), presentation (internal/render),
// scaffolding (internal/scaffold), and the command line (internal/cli) all
// depend on this package; nothing here depends on them. That direction is
// deliberate: the rules about what a valid plan *is* should not change when
// the file format or the output format does.
//
// Nothing in this package prints, reads files, or touches the network. Errors
// come back as values so the caller decides how to report them.
package tree
