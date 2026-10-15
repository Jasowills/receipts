package verdict

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/receipts/receipts/internal/model"
)

// The verdict layer depends on model only. Claim text parsing here stays
// minimal and local: the extractor owns full phrase matching, and the
// engine only recovers small structured sets (paths, package names) that
// the table needs to join claims to diff evidence.

var pathTokenRe = regexp.MustCompile(`^[A-Za-z0-9_@./-]+\.[A-Za-z0-9]+$|/`)

var depNameRes = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\badded\s+(\S+)\s+as\s+a\s+(dev\s+)?dependency\b`),
	regexp.MustCompile(`(?i)\bupdated\s+package\.json\s+to\s+include\s+(\S+)\b`),
	regexp.MustCompile(`(?i)\binstalled\s+(\S+)`),
}

var depDenialRe = regexp.MustCompile(`(?i)\b(no\s+new\s+dependenc|didn'?t\s+add\s+any\s+dependenc|did\s+not\s+add\s+any\s+dependenc|no\s+dependency\s+changes|dependenc\w*\s+unchanged|no\s+packages?\s+(added|installed))`)

// claimedPaths recovers path-like tokens from a file scope sentence by
// scanning words, so leading prose never drops trailing paths. Tokens
// need a slash or a sufficiently long extension to avoid matching
// abbreviations like e.g. Unparseable tokens are skipped.
func claimedPaths(text string) []string {
	clean := strings.ReplaceAll(text, "`", " ")
	seen := map[string]bool{}
	var out []string
	for _, w := range strings.Fields(clean) {
		tok := strings.Trim(strings.Trim(w, " \"'[]()."), " ")
		tok = strings.TrimRight(tok, ".,!?:;")
		if tok == "" || strings.Contains(tok, " ") || seen[tok] {
			continue
		}
		if !strings.Contains(tok, "/") && !(pathTokenRe.MatchString(tok) && len(tok) >= 5) {
			continue
		}
		norm, err := normalizeClaimPath(tok)
		if err != nil {
			continue
		}
		seen[tok] = true
		out = append(out, norm)
	}
	sort.Strings(out)
	return out
}

// normalizeClaimPath validates a candidate token as a repo relative path.
func normalizeClaimPath(tok string) (string, error) {
	if strings.Contains(tok, "..") {
		return "", fmt.Errorf("rejected")
	}
	tok = strings.TrimPrefix(tok, "./")
	if strings.HasPrefix(tok, "/") || tok == "." || tok == "" {
		return "", fmt.Errorf("rejected")
	}
	return tok, nil
}

// claimedPackage recovers a dependency name from an addition sentence.
func claimedPackage(text string) string {
	for _, re := range depNameRes {
		if m := re.FindStringSubmatch(text); m != nil {
			name := strings.Trim(m[1], "\"'`.,!?:;()[]")
			if name != "" {
				return name
			}
		}
	}
	return ""
}

func deniesDependencies(text string) bool {
	return depDenialRe.MatchString(text)
}

// changedPaths reads the path set from diff evidence data. It accepts the
// JSON decoded []any form and the in memory []string form.
func changedPaths(e model.Evidence) []string {
	return strList(e.Data, "changed_paths")
}

func strList(data map[string]any, key string) []string {
	raw, ok := data[key]
	if !ok || raw == nil {
		return nil
	}
	if ss, ok := raw.([]string); ok {
		out := append([]string{}, ss...)
		sort.Strings(out)
		return out
	}
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, item := range list {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func subsetOf(sub, set []string) bool {
	in := map[string]bool{}
	for _, s := range set {
		in[s] = true
	}
	for _, s := range sub {
		if !in[s] {
			return false
		}
	}
	return true
}

// verdictOfRun maps one runnable result to a claim verdict.
func verdictOfRun(exit int, reverify bool) (model.Verdict, string) {
	prefix := ""
	if reverify {
		prefix = "independent re-verification "
	}
	if exit == 0 {
		return model.VerdictVerified, fmt.Sprintf("%srecorded run passed", prefix)
	}
	return model.VerdictContradicted, fmt.Sprintf("%srecorded run failed", prefix)
}
