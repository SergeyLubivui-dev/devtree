package tree

import "strings"

// translit maps Cyrillic to Latin so a title written in Russian or Ukrainian
// still produces an ID a person can type on any keyboard. IDs travel through
// Mermaid node names, shell arguments, and Markdown anchors, and only ASCII
// survives all three intact.
var translit = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "e",
	'ж': "zh", 'з': "z", 'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m",
	'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
	'ф': "f", 'х': "h", 'ц': "c", 'ч': "ch", 'ш': "sh", 'щ': "sch",
	'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
	'і': "i", 'ї': "yi", 'є': "e", 'ґ': "g",
}

// maxSlugLen keeps generated IDs short enough to read inside a diagram label
// and to type without tab completion.
const maxSlugLen = 32

// Slugify turns a title into a lowercase ASCII identifier: letters and digits
// are kept, Cyrillic is transliterated, and every other run of characters
// collapses into a single dash.
func Slugify(s string) string {
	var b strings.Builder
	dashPending := false

	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dashPending = false
		default:
			if latin, ok := translit[r]; ok {
				b.WriteString(latin)
				dashPending = false
				continue
			}
			// Leading separators are dropped, and a run of them yields one
			// dash, so "OAuth 2.0 / PKCE" becomes "oauth-2-0-pkce".
			if !dashPending && b.Len() > 0 {
				b.WriteByte('-')
				dashPending = true
			}
		}
	}

	out := strings.Trim(b.String(), "-")
	if len(out) > maxSlugLen {
		out = strings.Trim(out[:maxSlugLen], "-")
	}
	return out
}
