package claims

import (
	"fmt"
	"sort"
	"strings"

	"github.com/receipts/receipts/internal/model"
)

// MergeAdditional folds extra claims (for example validated LLM
// proposals) into a deterministic base. Duplicates of existing claims
// drop out; survivors append in order and every ID is reassigned so the
// sequence stays stable. Only conflicts involving at least one new
// claim are reported, since base conflicts were warned already.
func MergeAdditional(base, extra []model.Claim) ([]model.Claim, []model.ExtractionWarning) {
	keys := map[string]bool{}
	for _, c := range base {
		keys[mergeKey(c)] = true
	}
	merged := append([]model.Claim{}, base...)
	addedIdx := map[int]bool{}
	for _, c := range extra {
		k := mergeKey(c)
		if keys[k] {
			continue
		}
		keys[k] = true
		addedIdx[len(merged)] = true
		merged = append(merged, c)
	}
	for i := range merged {
		merged[i].ID = fmt.Sprintf("C-%03d", i+1)
	}
	var warnings []model.ExtractionWarning
	for ni := range addedIdx {
		n := merged[ni]
		if n.Type != model.ClaimFilesChanged {
			continue
		}
		np := filePathsKey(n.Text)
		for _, c := range merged {
			if c.ID == n.ID || c.Type != model.ClaimFilesChanged {
				continue
			}
			if filePathsKey(c.Text) != np {
				a, b := c.ID, n.ID
				if a > b {
					a, b = b, a
				}
				warnings = append(warnings, model.ExtractionWarning{
					Code:    model.ExtractionConflictAgents,
					Message: fmt.Sprintf("%s and %s state different changed files", a, b),
				})
			}
		}
	}
	return merged, warnings
}

func mergeKey(c model.Claim) string {
	key := string(c.Type) + "\x00" + normalizeMeaning(c.Text)
	if c.Type == model.ClaimFilesChanged {
		key += "\x00" + filePathsKey(c.Text)
	}
	return key
}

func filePathsKey(text string) string {
	for _, re := range filesChangedRes {
		if sm := re.FindStringSubmatch(matchable(text)); sm != nil {
			if p := extractPaths(sm[1]); p != nil {
				cp := append([]string{}, p...)
				sort.Strings(cp)
				return strings.Join(cp, ",")
			}
		}
	}
	return ""
}
