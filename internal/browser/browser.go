// Package browser opens a URL in whatever the operating system considers the
// default browser.
//
// It is the second and last place in devtree that runs another program, and
// like internal/vcs it is reached from exactly one command. The separation is
// the point: nothing in the rendering or storage path can accidentally launch
// anything.
//
// There is no third-party library here for the same reason there is no YAML
// library — the whole job is one exec call per platform, and a dependency for
// three lines is a bad trade.
package browser

import (
	"errors"
	"os/exec"
	"runtime"
)

// ErrHeadless is returned when there is nothing to open a URL with — a CI job,
// a container, a machine over SSH. Callers print the URL instead of failing.
var ErrHeadless = errors.New("no browser available")

// Open hands a URL to the desktop.
func Open(url string) error {
	name, args := command(url)
	if name == "" {
		return ErrHeadless
	}
	if _, err := exec.LookPath(name); err != nil {
		return ErrHeadless
	}

	// Start, not Run: the browser outlives the command that launched it, and
	// waiting for a user to close their browser would be an odd thing to do.
	return exec.Command(name, args...).Start()
}

// command returns the platform's opener.
func command(url string) (string, []string) {
	switch runtime.GOOS {
	case "darwin":
		return "open", []string{url}
	case "windows":
		// rundll32 rather than `cmd /c start`, which would treat the URL's
		// ampersands as command separators.
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}
	case "linux", "freebsd", "openbsd", "netbsd":
		return "xdg-open", []string{url}
	default:
		return "", nil
	}
}
