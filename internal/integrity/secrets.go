package integrity

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/receipts/receipts/internal/model"
)

type secretPattern struct {
	name     string
	re       *regexp.Regexp
	severity model.Severity
}

var secretPatterns = []secretPattern{
	{"private-key-block", regexp.MustCompile(`-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----`), model.SeverityHigh},
	{"aws-access-key", regexp.MustCompile(`AKIA[0-9A-Z]{16}`), model.SeverityHigh},
	{"github-token", regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{20,}`), model.SeverityHigh},
	{"slack-token", regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]{10,}`), model.SeverityHigh},
	{"stripe-live-key", regexp.MustCompile(`sk_live_[A-Za-z0-9]{10,}`), model.SeverityHigh},
	{"credential-assignment", regexp.MustCompile(`(?i)(api[_-]?key|secret|password|passwd|pwd|token)\s*[:=]\s*["'][^"']{4,}["']`), model.SeverityWarning},
	{"jwt-like", regexp.MustCompile(`eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`), model.SeverityWarning},
}

// secretLockfiles skips integrity hash content that would otherwise trip
// high entropy detection.
var secretLockfiles = map[string]bool{
	"package-lock.json": true, "yarn.lock": true, "pnpm-lock.yaml": true,
	"poetry.lock": true, "cargo.lock": true, "go.sum": true,
	"pipfile.lock": true, "gemfile.lock": true, "composer.lock": true,
}

// placeholderValues are dummy credential contents that must not flag.
var placeholderValues = []string{
	"password", "123456", "changeme", "example", "test", "testing",
	"placeholder", "xxx", "****", "dummy", "sample", "your-password",
	"your-api-key", "todo", "fixme",
}

func lockBase(p string) string {
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return strings.ToLower(p[i+1:])
	}
	return strings.ToLower(p)
}

// containsSecretLike reports whether text trips any secret pattern.
// Shared with the skip check so redacted context stays consistent.
func containsSecretLike(text string) bool {
	for _, p := range secretPatterns {
		if p.re.MatchString(text) && !placeholderHit(p.re.FindString(text)) {
			return true
		}
	}
	return hasHighEntropyToken(text)
}

func placeholderHit(matched string) bool {
	// Extract a quoted value when present and test it against the denylist.
	if q1 := strings.IndexAny(matched, `"'`); q1 >= 0 {
		if q2 := strings.IndexAny(matched[q1+1:], `"'`); q2 >= 0 {
			val := strings.ToLower(matched[q1+1 : q1+1+q2])
			for _, ph := range placeholderValues {
				if strings.Contains(val, ph) {
					return true
				}
			}
		}
	}
	return false
}

// RedactSecrets replaces secret shaped substrings with [redacted].
// Placeholder values survive, mirroring detection. Best effort across
// write boundaries is the caller's concern; this function redacts one
// complete string.
func RedactSecrets(s string) string {
	out := s
	for _, p := range secretPatterns {
		out = p.re.ReplaceAllStringFunc(out, func(m string) string {
			if placeholderHit(m) {
				return m
			}
			return "[redacted]"
		})
	}
	return redactEntropy(out)
}

func redactEntropy(s string) string {
	var b strings.Builder
	for _, field := range strings.Fields(s) {
		trimmed := strings.Trim(field, `"'.,;:()[]{}<>`)
		if len(trimmed) >= 32 && entropyClasses(trimmed) >= 3 {
			b.WriteString(strings.Replace(field, trimmed, "[redacted]", 1))
		} else {
			b.WriteString(field)
		}
		b.WriteByte(' ')
	}
	return strings.TrimRight(b.String(), " ")
}

func entropyClasses(tok string) int {
	var upper, lower, digit, sym bool
	for _, r := range tok {
		switch {
		case r >= 'A' && r <= 'Z':
			upper = true
		case r >= 'a' && r <= 'z':
			lower = true
		case r >= '0' && r <= '9':
			digit = true
		default:
			sym = true
		}
	}
	classes := 0
	for _, b := range []bool{upper, lower, digit, sym} {
		if b {
			classes++
		}
	}
	return classes
}

func hasHighEntropyToken(text string) bool {
	for _, tok := range strings.Fields(text) {
		tok = strings.Trim(tok, `"'.,;:()[]{}<>`)
		if len(tok) < 32 {
			continue
		}
		if entropyClasses(tok) >= 3 {
			return true
		}
	}
	return false
}

// checkSecrets scans added lines only, never file contents at large.
// Raw matched values are never stored; details carry pattern name,
// path, line number, and a SHA256 over the match.
func checkSecrets(files []FileContent, added map[string][]AddedLine) []model.Flag {
	var flags []model.Flag
	for _, f := range files {
		if f.Binary || secretLockfiles[lockBase(f.Path)] {
			continue
		}
		lines, ok := added[f.Path]
		if !ok {
			if !f.HasAfter {
				continue
			}
			// Fallback without diff lines: scan After with line numbers.
			lines = nil
			for i, ln := range strings.Split(f.After, "\n") {
				lines = append(lines, AddedLine{No: i + 1, Text: ln})
			}
		}
		for _, l := range lines {
			for _, p := range secretPatterns {
				m := p.re.FindString(l.Text)
				if m == "" || placeholderHit(m) {
					continue
				}
				sum := sha256.Sum256([]byte(m))
				flags = append(flags, newFlag("SECRET_LIKE_VALUE_ADDED", p.severity,
					"Secret-like value added",
					fmt.Sprintf("possible secret (%s) at %s:%d (value redacted, sha256:%s)",
						p.name, f.Path, l.No, hex.EncodeToString(sum[:])),
					[]string{f.Path}))
				break
			}
			if hasHighEntropyToken(l.Text) {
				// Skip when a named pattern already flagged this line.
				dup := false
				for _, fl := range flags {
					if fl.Code == "SECRET_LIKE_VALUE_ADDED" && len(fl.Paths) == 1 && fl.Paths[0] == f.Path &&
						strings.Contains(fl.Details, fmt.Sprintf("%s:%d ", f.Path, l.No)) {
						dup = true
						break
					}
				}
				if !dup {
					sum := sha256.Sum256([]byte(l.Text))
					flags = append(flags, newFlag("SECRET_LIKE_VALUE_ADDED", model.SeverityWarning,
						"Secret-like value added",
						fmt.Sprintf("possible secret (high-entropy) at %s:%d (value redacted, sha256:%s)",
							f.Path, l.No, hex.EncodeToString(sum[:])),
						[]string{f.Path}))
				}
			}
		}
	}
	return flags
}
