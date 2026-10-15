package claims

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/receipts/receipts/internal/adapters"
	"github.com/receipts/receipts/internal/model"
)

// Extractor proposes completion claims from normalized messages.
type Extractor interface {
	Name() string
	Extract(ctx context.Context, messages []adapters.Message) ([]model.Claim, []model.ExtractionWarning, error)
}

// DeterministicExtractor matches high precision phrases only.
// After Extract, Seen holds the pattern id per returned claim for
// debugging; it is never part of the stored receipt. Initial verdicts
// are UNVERIFIED placeholders that the verdict engine overwrites.
type DeterministicExtractor struct {
	Seen []string
}

// Name returns the extractor name.
func (DeterministicExtractor) Name() string { return "deterministic" }

// Extract returns claims in message order with stable C-001 IDs.
func (e *DeterministicExtractor) Extract(ctx context.Context, messages []adapters.Message) ([]model.Claim, []model.ExtractionWarning, error) {
	type keyed struct {
		claim model.Claim
		pid   string
	}
	seen := map[string]bool{}
	var out []keyed
	for _, m := range messages {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		if m.Role != adapters.RoleAssistant {
			continue
		}
		for _, sentence := range splitSentences(m.Text) {
			c := matchSentence(sentence)
			if c == nil {
				continue
			}
			text := sentence
			if len(text) > ExcerptLimit {
				text = text[:ExcerptLimit]
			}
			key := string(c.typ) + "\x00" + normalizeMeaning(c.text)
			if c.typ == model.ClaimFilesChanged {
				cp := append([]string{}, c.paths...)
				sort.Strings(cp)
				key += "\x00" + strings.Join(cp, ",")
			}
			if seen[key] {
				continue
			}
			seen[key] = true
			claim := model.Claim{
				Type:         c.typ,
				Text:         text,
				Verdict:      model.VerdictUnverified,
				EvidenceRefs: []string{},
				Reason:       "",
			}
			claim.Source.MessageID = m.ID
			claim.Source.Role = string(m.Role)
			if !m.Timestamp.IsZero() {
				tcopy := m.Timestamp.UTC()
				claim.Source.Timestamp = &tcopy
			}
			claim.Source.ExcerptHash = excerptHash(text)
			out = append(out, keyed{claim: claim, pid: c.patternID})
		}
	}
	claims := make([]model.Claim, len(out))
	e.Seen = make([]string, len(out))
	for i := range out {
		out[i].claim.ID = fmt.Sprintf("C-%03d", i+1)
		claims[i] = out[i].claim
		e.Seen[i] = out[i].pid
	}
	return claims, detectConflicts(claims), nil
}

// detectConflicts retains every claim and warns about same type claims
// with incompatible specifics. In v0.1 only file scope claims qualify:
// differing changed path sets cannot both be exact. Pass versus fail
// contradictions belong to the verdict layer, where failed command
// evidence contradicts a pass claim.
func detectConflicts(claims []model.Claim) []model.ExtractionWarning {
	pathsOf := func(text string) []string {
		for _, re := range filesChangedRes {
			if sm := re.FindStringSubmatch(matchable(text)); sm != nil {
				if p := extractPaths(sm[1]); p != nil {
					cp := append([]string{}, p...)
					sort.Strings(cp)
					return cp
				}
			}
		}
		return nil
	}
	var warnings []model.ExtractionWarning
	for i := 0; i < len(claims); i++ {
		if claims[i].Type != model.ClaimFilesChanged {
			continue
		}
		pi := strings.Join(pathsOf(claims[i].Text), ",")
		for j := i + 1; j < len(claims); j++ {
			if claims[j].Type != model.ClaimFilesChanged {
				continue
			}
			if pj := strings.Join(pathsOf(claims[j].Text), ","); pj != pi {
				warnings = append(warnings, model.ExtractionWarning{
					Code:    model.ExtractionConflictAgents,
					Message: fmt.Sprintf("%s and %s state different changed files", claims[i].ID, claims[j].ID),
				})
			}
		}
	}
	return warnings
}
