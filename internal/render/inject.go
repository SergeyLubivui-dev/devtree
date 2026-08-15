package render

import "strings"

// Inject places a generated block into existing Markdown and reports whether
// anything changed.
//
// Three cases, in the order they are checked:
//
//	markers present  → the region between them is replaced, in place
//	no markers       → the block is appended, markers and all
//	nothing to change → the input is returned untouched
//
// The third case is what makes `devtree render` idempotent: running it twice
// produces the same bytes, so the pre-commit hook does not create empty
// commits and the CI check can treat "file changed" as "the diagram is stale".
func Inject(existing, block string) (result string, changed bool) {
	if strings.TrimSpace(existing) == "" {
		return block + "\n", true
	}

	start := markerIndex(existing, MarkerBegin, 0)
	end := -1
	if start >= 0 {
		end = markerIndex(existing, MarkerEnd, start+len(MarkerBegin))
	}

	switch {
	case start >= 0 && end > start:
		result = existing[:start] + block + existing[end+len(MarkerEnd):]
	default:
		// Keep exactly one blank line between the author's last paragraph and
		// the generated block, whether or not the file ended with a newline.
		separator := "\n\n"
		if strings.HasSuffix(existing, "\n") {
			separator = "\n"
		}
		result = existing + separator + block + "\n"
	}

	return result, result != existing
}

// markerIndex finds a marker that starts its own line, at or after from.
//
// The line-start rule matters more than it looks. A README that documents
// devtree — this project's own, for one — mentions the markers mid-sentence,
// and a plain substring search would treat that sentence as the region to
// replace and eat the paragraph around it. Generated markers are always
// written on a line of their own, so requiring that is enough to tell the real
// region from prose about it.
func markerIndex(s, marker string, from int) int {
	for i := from; i <= len(s); {
		offset := strings.Index(s[i:], marker)
		if offset < 0 {
			return -1
		}
		pos := i + offset
		if pos == 0 || s[pos-1] == '\n' {
			return pos
		}
		i = pos + len(marker)
	}
	return -1
}
