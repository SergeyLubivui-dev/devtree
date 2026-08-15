package browser

import (
	"runtime"
	"strings"
	"testing"
)

func TestCommandIsPlatformAppropriate(t *testing.T) {
	name, args := command("https://example.com/a?b=1&c=2")

	switch runtime.GOOS {
	case "darwin":
		if name != "open" {
			t.Errorf("got %q", name)
		}
	case "windows":
		// `cmd /c start` would treat the ampersands in a query string as
		// command separators, which is how a URL turns into three commands.
		if name != "rundll32" {
			t.Errorf("got %q", name)
		}
		if len(args) != 2 || !strings.HasPrefix(args[0], "url.dll") {
			t.Errorf("got %v", args)
		}
	case "linux":
		if name != "xdg-open" {
			t.Errorf("got %q", name)
		}
	default:
		if name != "" {
			t.Errorf("an unknown platform should report no opener, got %q", name)
		}
	}

	if name != "" && args[len(args)-1] != "https://example.com/a?b=1&c=2" {
		t.Errorf("the URL should be passed through untouched, got %v", args)
	}
}
