package integrity

import (
	"fmt"
	"strings"

	"github.com/receipts/receipts/internal/model"
)

// skipMarkers are framework skip spellings matched case sensitively.
// Plain words like "skip this" without a marker never match.
var skipMarkers = []string{
	".skip", "xit(", "xit ", "xdescribe", "@pytest.mark.skip",
	"pytest.skip", "t.Skip", "#[ignore]", "#[cfg(ignore)]", "DISABLED_",
}

func lineHasSkip(line string) (string, bool) {
	for _, m := range skipMarkers {
		if strings.Contains(line, m) {
			return m, true
		}
	}
	return "", false
}

// checkSkips flags added skip markers. Added diff lines are scanned
// precisely when provided; otherwise marker counts are compared.
func checkSkips(files []FileContent, added map[string][]AddedLine) []model.Flag {
	var flags []model.Flag
	for _, f := range files {
		if f.Binary {
			continue
		}
		if lines, ok := added[f.Path]; ok {
			for _, l := range lines {
				marker, hit := lineHasSkip(l.Text)
				if !hit {
					continue
				}
				context := strings.TrimSpace(l.Text)
				if len(context) > 120 {
					context = context[:120]
				}
				if containsSecretLike(context) {
					context = "[redacted]"
				}
				flags = append(flags, newFlag("SKIP_ADDED", model.SeverityWarning,
					"Skip marker added",
					fmt.Sprintf("%s:%d added %q: %s", f.Path, l.No, marker, context), []string{f.Path}))
			}
			continue
		}
		if !f.HasBefore || !f.HasAfter {
			continue
		}
		beforeN := countSkips(f.Before)
		afterN := countSkips(f.After)
		if afterN > beforeN {
			flags = append(flags, newFlag("SKIP_ADDED", model.SeverityWarning,
				"Skip marker added",
				fmt.Sprintf("%s: skip markers %d -> %d", f.Path, beforeN, afterN), []string{f.Path}))
		}
	}
	return flags
}

func countSkips(text string) int {
	n := 0
	for _, line := range strings.Split(text, "\n") {
		if _, hit := lineHasSkip(line); hit {
			n++
		}
	}
	return n
}
