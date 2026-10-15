package integrity

import (
	"regexp"
	"sort"
	"strings"

	"github.com/receipts/receipts/internal/model"
)

// assertionTokens are language agnostic assertion families plus common
// framework spellings. Counts are heuristic token tallies, never proof
// that assertions were removed.
var assertionTokens = []string{
	"assert", "expect", "should", "require", "verify", "check", "ensure",
	"t.Errorf", "t.Fatalf",
	"pytest.raises",
	"assertEqual", "assertTrue", "assertFalse", "assertRaises", "assertIn",
	"assertIs", "assertIsNone", "assertIsNotNone", "assertAlmostEqual",
	"assertThat", "expectThat",
}

var tokenRes = func() map[string]*regexp.Regexp {
	m := map[string]*regexp.Regexp{}
	for _, tok := range assertionTokens {
		m[tok] = regexp.MustCompile(`(^|[^A-Za-z0-9_])` + regexp.QuoteMeta(tok) + `([^A-Za-z0-9_]|$)`)
	}
	return m
}()

// stripCode removes comments and string literals best effort by extension
// so tokens inside them do not count. Unknown extensions keep full text.
func stripCode(text, ext string) string {
	switch ext {
	case ".go", ".js", ".jsx", ".ts", ".tsx", ".rs", ".java", ".cs", ".swift", ".kt":
		return stripCStyle(text)
	case ".py":
		return stripPython(text)
	default:
		return text
	}
}

func stripCStyle(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		switch {
		case strings.HasPrefix(s[i:], "//"):
			for i < len(s) && s[i] != '\n' {
				i++
			}
		case strings.HasPrefix(s[i:], "/*"):
			i += 2
			for i < len(s) && !strings.HasPrefix(s[i:], "*/") {
				if s[i] == '\n' {
					b.WriteByte('\n')
				}
				i++
			}
			i += 2
		case s[i] == '"' || s[i] == '\'' || s[i] == '`':
			q := s[i]
			i++
			for i < len(s) && s[i] != q {
				if s[i] == '\\' {
					i++
				}
				i++
			}
			i++
			b.WriteByte(' ')
		default:
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

func stripPython(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		switch {
		case strings.HasPrefix(s[i:], "\"\"\"") || strings.HasPrefix(s[i:], "'''"):
			q := s[i : i+3]
			i += 3
			for i < len(s) && !strings.HasPrefix(s[i:], q) {
				if s[i] == '\n' {
					b.WriteByte('\n')
				}
				i++
			}
			i += 3
			b.WriteByte(' ')
		case s[i] == '#':
			for i < len(s) && s[i] != '\n' {
				i++
			}
		case s[i] == '"' || s[i] == '\'':
			q := s[i]
			i++
			for i < len(s) && s[i] != q && s[i] != '\n' {
				if s[i] == '\\' {
					i++
				}
				i++
			}
			i++
			b.WriteByte(' ')
		default:
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

func extOf(p string) string {
	if i := strings.LastIndexByte(p, '.'); i >= 0 {
		return strings.ToLower(p[i:])
	}
	return ""
}

func countTokens(stripped string) (int, []string) {
	total := 0
	var fams []string
	for _, tok := range assertionTokens {
		n := len(tokenRes[tok].FindAllStringIndex(stripped, -1))
		if n > 0 {
			total += n
			fams = append(fams, tok)
		}
	}
	sort.Strings(fams)
	return total, fams
}

// checkAssertions flags assertion-like token count decreases in changed
// test files. Wording reports token counts only, never a definitive
// removed assertion claim.
func checkAssertions(files []FileContent) []model.Flag {
	var flags []model.Flag
	for _, f := range files {
		if f.Binary || !f.HasBefore || !f.HasAfter || !isTestPath(f.Path) {
			continue
		}
		ext := extOf(f.Path)
		beforeN, _ := countTokens(stripCode(f.Before, ext))
		afterN, afterFams := countTokens(stripCode(f.After, ext))
		if afterN < beforeN {
			details := f.Path + ": " + itoa(beforeN) + " -> " + itoa(afterN) + " assertion-like tokens"
			if len(afterFams) > 0 {
				details += " (remaining families: " + strings.Join(afterFams, ", ") + ")"
			} else {
				details += " (no assertion-like tokens remain)"
			}
			flags = append(flags, newFlag("ASSERTION_TOKEN_COUNT_REDUCED", model.SeverityWarning,
				"Assertion token count reduced", details, []string{f.Path}))
		}
	}
	return flags
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [32]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
