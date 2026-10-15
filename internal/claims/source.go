package claims

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

// ExcerptLimit bounds claim text stored per claim.
const ExcerptLimit = 2000

var spaceRe = regexp.MustCompile(`\s+`)

var fillerWords = map[string]bool{
	"all": true, "the": true, "a": true, "an": true, "i": true, "we": true,
	"ran": true, "run": true, "have": true, "has": true, "had": true,
	"are": true, "is": true, "was": true, "were": true, "and": true,
	"they": true, "them": true, "my": true, "our": true, "just": true,
	"now": true, "successfully": true, "suite": true, "locally": true,
}

var runnerTokens = map[string]bool{
	"npm": true, "pnpm": true, "yarn": true, "bun": true, "test": true,
	"tests": true, "go": true, "pytest": true, "jest": true, "vitest": true,
	"cargo": true, "make": true, "bundle": true, "exec": true, "rspec": true,
	"./...": true,
}

var verbStems = map[string]string{
	"passes": "pass", "passing": "pass", "passed": "pass",
	"succeeds": "succeed", "succeeding": "succeed", "succeeded": "succeed",
	"completed": "complete", "completes": "complete",
	"changed": "change", "changes": "change",
	"modified": "modify", "modifies": "modify",
	"fixed": "fix", "fixes": "fix",
	"resolved": "resolve", "resolves": "resolve",
}

// normalizeMeaning reduces claim text to a comparable core so repeats
// with different filler dedup while distinct claims stay separate.
func normalizeMeaning(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "`", " ")
	s = strings.ReplaceAll(s, `"`, " ")
	s = strings.ReplaceAll(s, `'`, " ")
	words := spaceRe.Split(s, -1)
	var kept []string
	for _, w := range words {
		w = strings.Trim(w, ".,!?;:()[]{}")
		if w == "" || fillerWords[w] || runnerTokens[w] {
			continue
		}
		if stem, ok := verbStems[w]; ok {
			w = stem
		}
		kept = append(kept, w)
	}
	return strings.Join(kept, " ")
}

// excerptHash returns hex SHA256 over the bounded claim excerpt.
func excerptHash(text string) string {
	if len(text) > ExcerptLimit {
		text = text[:ExcerptLimit]
	}
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}
