package claims

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/receipts/receipts/internal/adapters"
	"github.com/receipts/receipts/internal/model"
	"github.com/receipts/receipts/internal/verdict"
)

func pad3n(n int) string {
	s := ""
	for i := 0; i < 3; i++ {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func timeNow() time.Time { return time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC) }

func splitCmd(s string) []any {
	out := []any{}
	for _, f := range strings.Fields(s) {
		out = append(out, f)
	}
	return out
}

type labeledEvidence struct {
	Command      *string       `json:"command"`
	Exit         *int          `json:"exit"`
	Commands     *[]labeledCmd `json:"commands"`
	TestResult   *bool         `json:"test_result"`
	DiffPaths    []string      `json:"diff_paths"`
	Confidence   *string       `json:"confidence"`
	ManifestAdds []string      `json:"manifest_adds"`
	Manifests    []string      `json:"manifests"`
}

type labeledCmd struct {
	Command string `json:"command"`
	Exit    int    `json:"exit"`
}

type labeledItem struct {
	ID              string           `json:"id"`
	Text            string           `json:"text"`
	ExpectedType    *string          `json:"expected_type"`
	ShouldExtract   bool             `json:"should_extract"`
	Evidence        *labeledEvidence `json:"evidence"`
	ExpectedVerdict *string          `json:"expected_verdict"`
}

func loadLabeled(t *testing.T) []labeledItem {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "..", "testdata", "labeled-claims.jsonl"))
	if err != nil {
		t.Fatalf("open dataset: %v", err)
	}
	defer f.Close()
	var items []labeledItem
	seen := map[string]bool{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var it labeledItem
		if err := json.Unmarshal([]byte(line), &it); err != nil {
			t.Fatalf("parse line: %v", err)
		}
		if seen[it.ID] {
			t.Fatalf("duplicate id %s", it.ID)
		}
		seen[it.ID] = true
		items = append(items, it)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	return items
}

// TestLabeledDataset measures extraction precision and recall plus verdict
// accuracy, gating the v0.1 quality targets: precision at least 95%,
// recall at least 80%, and zero unexpected CONTRADICTED verdicts.
func TestLabeledDataset(t *testing.T) {
	items := loadLabeled(t)
	if len(items) != 200 {
		t.Fatalf("dataset has %d items, want 200", len(items))
	}
	ctx := context.Background()
	ex := DeterministicExtractor{}
	var extracted, correctType, wantExtract, gotExpected int
	var verdictTotal, verdictCorrect, badContradictions int
	for _, it := range items {
		claims, _, err := ex.Extract(ctx, []adapters.Message{{ID: it.ID, Role: adapters.RoleAssistant, Text: it.Text}})
		if err != nil {
			t.Fatalf("%s extract: %v", it.ID, err)
		}
		if it.ShouldExtract {
			wantExtract++
			if len(claims) == 1 && string(claims[0].Type) == *it.ExpectedType {
				correctType++
				gotExpected++
			} else {
				t.Errorf("%s: got %+v, want one %s claim", it.ID, claims, *it.ExpectedType)
				if len(claims) > 0 {
					extracted++
				}
				continue
			}
			extracted++
		} else if len(claims) > 0 {
			t.Errorf("%s: got %+v, want none", it.ID, claims)
			extracted++
		}
		if it.ExpectedVerdict == nil {
			continue
		}
		if len(claims) != 1 {
			t.Errorf("%s: need one claim for verdict check", it.ID)
			continue
		}
		verdictTotal++
		got := runVerdictHarness(t, claims[0], it)
		if string(got) == *it.ExpectedVerdict {
			verdictCorrect++
		} else {
			t.Errorf("%s: verdict = %s, want %s", it.ID, got, *it.ExpectedVerdict)
		}
		if got == model.VerdictContradicted && *it.ExpectedVerdict != string(model.VerdictContradicted) {
			badContradictions++
		}
	}
	precision := 0.0
	if extracted > 0 {
		precision = float64(correctType) / float64(extracted)
	}
	recall := float64(gotExpected) / float64(wantExtract)
	t.Logf("extraction: precision %.3f (%d/%d) recall %.3f (%d/%d) verdict %d/%d badContradictions %d",
		precision, correctType, extracted, recall, gotExpected, wantExtract, verdictCorrect, verdictTotal, badContradictions)
	if precision < 0.95 {
		t.Errorf("precision %.3f below 0.95 target", precision)
	}
	if recall < 0.80 {
		t.Errorf("recall %.3f below 0.80 target", recall)
	}
	if badContradictions > 0 {
		t.Errorf("%d unexpected CONTRADICTED verdicts", badContradictions)
	}
}

func runVerdictHarness(t *testing.T, claim model.Claim, it labeledItem) model.Verdict {
	t.Helper()
	ev := it.Evidence
	tier := model.Tier0
	conf := model.BaselineConfidence("high")
	var evidences []model.Evidence
	n := 0
	add := func(e model.Evidence) {
		n++
		e.ID = "E-" + pad3n(n)
		evidences = append(evidences, e)
	}
	ts := timeNow()
	if ev != nil {
		if ev.Command != nil {
			tier = model.Tier1
			add(model.Evidence{Kind: model.EvidenceCommand, Timestamp: ts,
				Data: map[string]any{"argv": splitCmd(*ev.Command), "exit_code": float64(*ev.Exit)}})
		}
		if ev.Commands != nil {
			tier = model.Tier1
			for _, c := range *ev.Commands {
				add(model.Evidence{Kind: model.EvidenceCommand, Timestamp: ts,
					Data: map[string]any{"argv": splitCmd(c.Command), "exit_code": float64(c.Exit)}})
			}
		}
		if ev.TestResult != nil {
			tier = model.Tier1
			add(model.Evidence{Kind: model.EvidenceTestResult, Timestamp: ts,
				Data: map[string]any{"passed": *ev.TestResult}})
		}
		if ev.DiffPaths != nil || ev.Manifests != nil {
			if ev.Confidence != nil {
				conf = model.BaselineConfidence(*ev.Confidence)
			}
			data := map[string]any{}
			if ev.DiffPaths != nil {
				data["changed_paths"] = ev.DiffPaths
			}
			if ev.ManifestAdds != nil {
				data["added_dependencies"] = ev.ManifestAdds
			}
			if ev.Manifests != nil {
				data["manifests_changed"] = ev.Manifests
			}
			add(model.Evidence{Kind: model.EvidenceDiff, Timestamp: ts, Data: data})
		}
	}
	r := &model.Receipt{
		SchemaVersion: model.SchemaVersion, ID: "RC-0001", CreatedAt: ts,
		Agent: model.Agent{Name: "test"}, Tier: tier,
		Repo:   model.Repo{Root: ".", BaselineConfidence: conf},
		Claims: []model.Claim{claim}, Flags: []model.Flag{},
		Evidence: evidences, Warnings: []string{}, Summary: model.Summary{},
		Chain: model.Chain{Algorithm: model.ChainAlgorithmSHA256},
	}
	eng := verdict.DeterministicEngine{}
	got, _, err := eng.Evaluate(r)
	if err != nil {
		t.Fatalf("%s evaluate: %v", it.ID, err)
	}
	return got[0].Verdict
}
