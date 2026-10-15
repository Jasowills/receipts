package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func schemaPath(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "schemas", "receipt.schema.json")
}

func goldenPath(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "testdata", "golden", "v0.1-valid.json")
}

func TestReceiptSchemaFileExistsAndIsValidJSON(t *testing.T) {
	data, err := os.ReadFile(schemaPath(t))
	if err != nil {
		t.Fatalf("read schema failed: %v", err)
	}
	var v map[string]any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
	if v["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
		t.Errorf("schema $schema = %v, want draft 2020-12 URL", v["$schema"])
	}
}

func TestGoldenReceiptValidates(t *testing.T) {
	data, err := os.ReadFile(goldenPath(t))
	if err != nil {
		t.Fatalf("read golden failed: %v", err)
	}
	var r Receipt
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatalf("golden unmarshal failed: %v", err)
	}
	if err := ValidateReceipt(r); err != nil {
		t.Fatalf("golden receipt should validate, got: %v", err)
	}
}

func TestInvalidReceiptsFailValidation(t *testing.T) {
	base := validTestReceipt(t)

	cases := []struct {
		name   string
		mutate func(*Receipt)
	}{
		{"bad id pattern", func(r *Receipt) { r.ID = "bad-id" }},
		{"bad verdict enum", func(r *Receipt) {
			r.Claims = []Claim{{ID: "C-001", Type: ClaimTestsPass, Text: "t", Source: Source{MessageID: "m", Role: "assistant"}, Verdict: Verdict("NOPE"), EvidenceRefs: []string{}, Reason: "r"}}
		}},
		{"missing agent name", func(r *Receipt) { r.Agent.Name = "" }},
		{"bad claim id pattern", func(r *Receipt) {
			r.Claims = []Claim{{ID: "X-1", Type: ClaimTestsPass, Text: "t", Source: Source{MessageID: "m", Role: "assistant"}, Verdict: VerdictUnverified, EvidenceRefs: []string{}, Reason: "r"}}
		}},
		{"dangling evidence ref", func(r *Receipt) {
			r.Claims = []Claim{{ID: "C-001", Type: ClaimTestsPass, Text: "t", Source: Source{MessageID: "m", Role: "assistant"}, Verdict: VerdictUnverified, EvidenceRefs: []string{"E-999"}, Reason: "r"}}
			r.Evidence = []Evidence{}
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := base
			tc.mutate(&r)
			if err := ValidateReceipt(r); err == nil {
				t.Errorf("expected validation error for %s, got nil", tc.name)
			}
		})
	}
}

func validTestReceipt(t *testing.T) Receipt {
	t.Helper()
	data, err := os.ReadFile(goldenPath(t))
	if err != nil {
		t.Fatalf("read golden failed: %v", err)
	}
	var r Receipt
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatalf("golden unmarshal failed: %v", err)
	}
	return r
}
