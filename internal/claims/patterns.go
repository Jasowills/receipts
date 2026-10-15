package claims

import (
	"regexp"
	"strings"

	"github.com/receipts/receipts/internal/gitstate"
	"github.com/receipts/receipts/internal/model"
)

// candidate is one matched sentence before dedup.
type candidate struct {
	typ       model.ClaimType
	text      string
	patternID string
	paths     []string
}

var (
	codeSpanRe    = regexp.MustCompile("`[^`]*`")
	negationRe    = regexp.MustCompile(`(?i)\b(did\s+not|didn't|do\s+not|don't|does\s+not|doesn't|never|not|n't\b|failed|failing|failure|couldn't|could\s+not|wouldn't|shouldn't|without)\b`)
	conditionalRe = regexp.MustCompile(`(?i)\b(should|would|could|may|might|whether|will\b|'ll\b|going\s+to|plan\s+to)\b|\bif\s`)
	noPassRe      = regexp.MustCompile(`(?i)\bno\b[^.!?]*\bpass`)
	exceptRe      = regexp.MustCompile(`(?i)\b(except|excluding|other\s+than|but\s+not)\b`)

	testPassRes = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\ball\s+tests?\s+(pass|passes|passing|passed)\b`),
		regexp.MustCompile(`(?i)\btests?\s+(have\s+|has\s+)?pass(ed|es|ing)?\b`),
		regexp.MustCompile(`(?i)\btest\s+suite\s+passes?\b`),
		regexp.MustCompile(`(?i)\btest\s+suite\s+(is|are)\s+(passing|green)\b`),
		regexp.MustCompile(`(?i)\btests?\s+are\s+passing\b`),
		regexp.MustCompile(`(?i)\b((npm|pnpm|yarn|bun)\s+test|go\s+test|pytest|cargo\s+test|jest|vitest|make\s+test)\b[^!?]*\bpass(ed|es|ing)?\b`),
		regexp.MustCompile(`(?i)\bi\s+ran\s+(the\s+)?tests?\s+and\s+they\s+passed\b`),
	}
	buildOKRes = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bbuild\s+(succeeds|succeeded|passed|passes|completed\s+successfully|successful|is\s+successful|is\s+green|is\s+passing)\b`),
		regexp.MustCompile(`(?i)\bproduction\s+build\s+completed\s+successfully\b`),
		regexp.MustCompile(`(?i)\bnpm\s+run\s+build\s+passed\b`),
	}
	lintCleanRes = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\blint\s+(passed|passes|is\s+clean|clean)\b`),
		regexp.MustCompile(`(?i)\bno\s+lint\s+errors\b`),
		regexp.MustCompile(`(?i)\beslint\s+(passed|clean)\b`),
		regexp.MustCompile(`(?i)\blinting\s+is\s+clean\b`),
	}
	filesChangedRes = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bi\s+only\s+changed\s+(.+)`),
		regexp.MustCompile(`(?i)\bchanged\s+files?\s*:(.+)`),
		regexp.MustCompile(`(?i)\bmodified\s+only\s+(.+)`),
		regexp.MustCompile(`(?i)\btouched\s+only\s+(.+)`),
	}
	noOtherRes = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bno\s+other\s+files?\s+(were\s+|was\s+)?changed\b`),
		regexp.MustCompile(`(?i)\bno\s+unrelated\s+changes?\b`),
		regexp.MustCompile(`(?i)\bnothing\s+else\s+was\s+modified\b`),
		regexp.MustCompile(`(?i)\bno\s+other\s+changes?\b`),
		regexp.MustCompile(`(?i)\bno\s+unrelated\s+modifications?\b`),
	}
	depAddedRes = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\badded\s+(\S+)\s+as\s+a\s+(dev\s+)?dependency\b`),
		regexp.MustCompile(`(?i)\bupdated\s+package\.json\s+to\s+include\s+(\S+)\b`),
	}
	// depDenialRe matches explicit no dependency change statements so the
	// verdict layer can contradict them against manifest evidence. It
	// mirrors the verdict matcher; both stay narrow by design.
	depDenialRe   = regexp.MustCompile(`(?i)\b(no\s+new\s+dependenc|didn'?t\s+add\s+any\s+dependenc|did\s+not\s+add\s+any\s+dependenc|no\s+dependency\s+changes|dependenc\w*\s+unchanged|no\s+packages?\s+(added|installed))`)
	issueFixedRes = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bfixed\s+the\s+(bug|issue|problem|error)\b`),
		regexp.MustCompile(`(?i)\bthe\s+(issue|bug|problem)\s+is\s+(resolved|fixed)\b`),
		regexp.MustCompile(`(?i)\bthis\s+resolves\s+the\s+reported\s+problem\b`),
		regexp.MustCompile(`(?i)\bissue\s+fixed\b`),
	}
	pathTokenRe = regexp.MustCompile(`^[A-Za-z0-9_@./-]+\.[A-Za-z0-9]+$|/`)
)

// splitSentences splits assistant text into matchable sentences, tracking
// fenced code blocks and skipping quotes. Periods split only before
// whitespace plus an uppercase letter or end of line, so command paths
// like ./... and dotted names like package.json stay intact.
func splitSentences(text string) []string {
	var out []string
	inFence := false
	for _, line := range splitLines(text) {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if inFence || strings.HasPrefix(trimmed, ">") || trimmed == "" {
			continue
		}
		for _, s := range splitBoundaries(line) {
			if strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
	}
	return out
}

func splitLines(s string) []string {
	return strings.Split(s, "\n")
}

// splitBoundaries splits on newlines (already handled), !, ?, and periods
// that look like sentence ends: followed by whitespace plus an uppercase
// letter or nothing at all.
func splitBoundaries(line string) []string {
	var parts []string
	start := 0
	runes := []rune(line)
	i := 0
	for i < len(runes) {
		r := runes[i]
		if r == '!' || r == '?' {
			parts = append(parts, string(runes[start:i+1]))
			i++
			start = i
			continue
		}
		if r == '.' {
			j := i
			for j < len(runes) && runes[j] == '.' {
				j++
			}
			k := j
			for k < len(runes) && (runes[k] == ' ' || runes[k] == '\t') {
				k++
			}
			if k >= len(runes) || (runes[k] >= 'A' && runes[k] <= 'Z') {
				parts = append(parts, string(runes[start:i]))
				i = j
				start = i
				continue
			}
			i = j
			continue
		}
		i++
	}
	parts = append(parts, string(runes[start:]))
	return parts
}

// matchable returns the sentence with inline code blanked for matching.
func matchable(s string) string {
	return codeSpanRe.ReplaceAllString(s, " ")
}

// blocked reports negation or conditional language in a sentence.
// Negation wins over positive matches in the same sentence.
func blocked(s string) bool {
	if noPassRe.MatchString(s) {
		return true
	}
	if negationRe.MatchString(s) {
		return true
	}
	return conditionalRe.MatchString(s)
}

func anyMatch(s string, res []*regexp.Regexp) bool {
	for _, re := range res {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

// extractPaths pulls repo relative paths from a file list construction by
// scanning words, so leading prose never drops trailing paths. Tokens
// need a slash or a sufficiently long extension. It returns nil when
// uncertain rather than guessing.
func extractPaths(list string) []string {
	clean := strings.ReplaceAll(list, "`", " ")
	seen := map[string]bool{}
	var paths []string
	for _, w := range strings.Fields(clean) {
		tok := strings.Trim(strings.Trim(w, " \"'[]()."), " ")
		tok = strings.TrimRight(tok, ".,!?:;")
		if tok == "" || strings.Contains(tok, " ") || seen[tok] {
			continue
		}
		if strings.EqualFold(tok, "and") || strings.EqualFold(tok, "or") {
			continue
		}
		if !strings.Contains(tok, "/") && !(pathTokenRe.MatchString(tok) && len(tok) >= 5) {
			continue
		}
		norm, err := gitstate.NormalizePath(".", tok)
		if err != nil {
			continue
		}
		seen[tok] = true
		paths = append(paths, norm)
	}
	if len(paths) == 0 {
		return nil
	}
	return paths
}

// matchSentence returns at most one candidate per sentence, checking
// qualified test claims first so they never overstate as TESTS_PASS.
// Questions are never assertions.
func matchSentence(sentence string) *candidate {
	if strings.HasSuffix(strings.TrimSpace(sentence), "?") {
		return nil
	}
	m := matchable(sentence)
	if blocked(m) {
		return nil
	}
	isTest := anyMatch(m, testPassRes)
	if isTest && exceptRe.MatchString(m) {
		return &candidate{typ: model.ClaimOther, text: sentence, patternID: "test-qualified"}
	}
	if isTest {
		return &candidate{typ: model.ClaimTestsPass, text: sentence, patternID: "tests-pass"}
	}
	if anyMatch(m, buildOKRes) {
		return &candidate{typ: model.ClaimBuildOK, text: sentence, patternID: "build-ok"}
	}
	if anyMatch(m, lintCleanRes) {
		return &candidate{typ: model.ClaimLintClean, text: sentence, patternID: "lint-clean"}
	}
	for _, re := range filesChangedRes {
		if sm := re.FindStringSubmatch(m); sm != nil {
			paths := extractPaths(sm[1])
			if paths == nil {
				return nil
			}
			return &candidate{typ: model.ClaimFilesChanged, text: sentence, patternID: "files-changed", paths: paths}
		}
	}
	if anyMatch(m, noOtherRes) {
		return &candidate{typ: model.ClaimNoOtherChanges, text: sentence, patternID: "no-other-changes"}
	}
	if depDenialRe.MatchString(m) {
		return &candidate{typ: model.ClaimNoOtherChanges, text: sentence, patternID: "no-dep-changes"}
	}
	for _, re := range depAddedRes {
		if re.MatchString(m) {
			return &candidate{typ: model.ClaimDepAdded, text: sentence, patternID: "dep-added"}
		}
	}
	if strings.Contains(strings.ToLower(m), "installed") && strings.Contains(strings.ToLower(m), "depend") {
		toks := strings.Fields(m)
		for i, tok := range toks {
			if strings.EqualFold(tok, "installed") && i+1 < len(toks) {
				name := strings.Trim(toks[i+1], "\"'`.,!?:;()[]")
				if pathTokenRe.MatchString(name) {
					return &candidate{typ: model.ClaimDepAdded, text: sentence, patternID: "dep-installed"}
				}
			}
		}
	}
	if anyMatch(m, issueFixedRes) {
		return &candidate{typ: model.ClaimIssueFixed, text: sentence, patternID: "issue-fixed"}
	}
	return nil
}
