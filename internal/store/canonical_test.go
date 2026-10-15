package store

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/receipts/receipts/internal/model"
)

func testReceipt() string {
	return filepath.Join("..", "..", "testdata", "golden", "v0.1-valid.json")
}

func loadGolden(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(testReceipt())
	if err != nil {
		t.Fatalf("read golden failed: %v", err)
	}
	return data
}

func TestMarshalCanonicalIsDeterministic(t *testing.T) {
	data := loadGolden(t)
	var r1 map[string]any
	if err := json.Unmarshal(data, &r1); err != nil {
		t.Fatalf("unmarshal golden failed: %v", err)
	}
	// Decode via canonical path twice from same bytes.
	rec1, err := UnmarshalCanonical(data)
	if err != nil {
		t.Fatalf("UnmarshalCanonical failed: %v", err)
	}
	b1, err := MarshalCanonical(rec1)
	if err != nil {
		t.Fatalf("MarshalCanonical failed: %v", err)
	}
	b2, err := MarshalCanonical(rec1)
	if err != nil {
		t.Fatalf("second MarshalCanonical failed: %v", err)
	}
	if !bytes.Equal(b1, b2) {
		t.Errorf("canonical marshal not deterministic")
	}
}

func TestMarshalCanonicalNormalizesNilSlices(t *testing.T) {
	data := loadGolden(t)
	rec, err := UnmarshalCanonical(data)
	if err != nil {
		t.Fatalf("UnmarshalCanonical failed: %v", err)
	}
	rec.Claims = nil
	rec.Flags = nil
	rec.Evidence = nil
	rec.Warnings = nil
	out, err := MarshalCanonical(rec)
	if err != nil {
		t.Fatalf("MarshalCanonical with nil slices failed: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal canonical failed: %v", err)
	}
	for _, k := range []string{"claims", "flags", "evidence", "warnings"} {
		v, ok := m[k]
		if !ok {
			t.Errorf("canonical JSON missing key %q", k)
			continue
		}
		arr, ok := v.([]any)
		if !ok {
			t.Errorf("key %q is %T, want array (nil must become [])", k, v)
			continue
		}
		if len(arr) != 0 {
			t.Errorf("key %q len = %d, want 0", k, len(arr))
		}
	}
}

func TestMarshalCanonicalRejectsInvalid(t *testing.T) {
	data := loadGolden(t)
	rec, err := UnmarshalCanonical(data)
	if err != nil {
		t.Fatalf("UnmarshalCanonical failed: %v", err)
	}
	rec.ID = "bad"
	if _, err := MarshalCanonical(rec); err == nil {
		t.Errorf("expected error for bad id, got nil")
	}
}

func TestUnmarshalCanonicalRejectsBadJSON(t *testing.T) {
	if _, err := UnmarshalCanonical([]byte(`{"schema_version":"1.0.0"}`)); err == nil {
		t.Errorf("expected error for incomplete receipt, got nil")
	}
}

func TestCanonicalStableSliceOrdering(t *testing.T) {
	data := loadGolden(t)
	base, err := UnmarshalCanonical(data)
	if err != nil {
		t.Fatalf("UnmarshalCanonical failed: %v", err)
	}
	ts := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	e1 := model.Evidence{ID: "E-001", Kind: model.EvidenceDiff, Timestamp: ts, Data: map[string]any{"a": "1"}}
	e2 := model.Evidence{ID: "E-002", Kind: model.EvidenceDiff, Timestamp: ts, Data: map[string]any{"a": "1"}}
	a := base
	a.Evidence = []model.Evidence{e2, e1}
	b := base
	b.Evidence = []model.Evidence{e1, e2}
	ba, err := MarshalCanonical(a)
	if err != nil {
		t.Fatalf("MarshalCanonical a failed: %v", err)
	}
	bb, err := MarshalCanonical(b)
	if err != nil {
		t.Fatalf("MarshalCanonical b failed: %v", err)
	}
	if !bytes.Equal(ba, bb) {
		t.Errorf("canonical marshal is order dependent for evidence slices")
	}
}

func TestMarshalCanonicalNilAndEmptySlicesMatch(t *testing.T) {
	data := loadGolden(t)
	base, err := UnmarshalCanonical(data)
	if err != nil {
		t.Fatalf("UnmarshalCanonical failed: %v", err)
	}
	a := base
	a.Claims = nil
	a.Flags = nil
	a.Evidence = nil
	a.Warnings = nil
	b := base
	b.Claims = []model.Claim{}
	b.Flags = []model.Flag{}
	b.Evidence = []model.Evidence{}
	b.Warnings = []string{}
	ba, err := MarshalCanonical(a)
	if err != nil {
		t.Fatalf("MarshalCanonical nil failed: %v", err)
	}
	bb, err := MarshalCanonical(b)
	if err != nil {
		t.Fatalf("MarshalCanonical empty failed: %v", err)
	}
	if !bytes.Equal(ba, bb) {
		t.Errorf("nil and empty slices must produce identical canonical JSON")
	}
	if bytes.Contains(ba, []byte(`"claims":null`)) {
		t.Errorf("canonical JSON must not contain null arrays")
	}
}
