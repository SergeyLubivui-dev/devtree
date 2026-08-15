package tree

import "testing"

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Authentication":       "authentication",
		"OAuth 2.0 / PKCE":     "oauth-2-0-pkce",
		"  Trim   me  ":        "trim-me",
		"Payments (Stripe)":    "payments-stripe",
		"Аутентификация OAuth": "autentifikaciya-oauth",
		"C++ bindings":         "c-bindings",
		"":                     "",
		"---":                  "",
		"Ship it! 🚀":           "ship-it",
	}

	for input, want := range cases {
		if got := Slugify(input); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSlugifyStaysShortAndClean(t *testing.T) {
	got := Slugify("A very long milestone title that nobody would ever want to type")
	if len(got) > maxSlugLen {
		t.Fatalf("%q is %d characters, want at most %d", got, len(got), maxSlugLen)
	}
	if got[len(got)-1] == '-' {
		t.Fatalf("%q ends with a dash: truncation should not leave one behind", got)
	}
}
