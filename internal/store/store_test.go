package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/receipts/receipts/internal/model"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := filepath.Join(t.TempDir(), ".receipts")
	return New(dir)
}

func testReceiptWithID(id string) model.Receipt {
	ts := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	dirty := true
	return model.Receipt{
		SchemaVersion: model.SchemaVersion,
		ID:            id,
		CreatedAt:     ts,
		Agent:         model.Agent{Name: "opencode", SessionID: "s-1", AdapterVersion: "1"},
		Tier:          model.Tier0,
		DurationMS:    100,
		Repo: model.Repo{
			Root:               ".",
			BaselineConfidence: model.BaselineLow,
			DirtyBefore:        model.DirtyUnknown,
			DirtyAfter:         &dirty,
		},
		Claims:   []model.Claim{},
		Flags:    []model.Flag{},
		Evidence: []model.Evidence{},
		Warnings: []string{},
		Summary:  model.Summary{},
		Chain:    model.Chain{Algorithm: model.ChainAlgorithmSHA256},
	}
}

func TestSaveGetRoundTrip(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	r := testReceiptWithID("RC-0001")
	if err := st.Save(ctx, r); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	got, err := st.Get(ctx, "RC-0001")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != "RC-0001" {
		t.Errorf("Get ID = %q, want RC-0001", got.ID)
	}
	if got.Chain.EventHash == "" {
		t.Errorf("stored receipt should have chain event hash filled")
	}
}

func TestLatestAndList(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	if err := st.Save(ctx, testReceiptWithID("RC-0001")); err != nil {
		t.Fatalf("Save 1 failed: %v", err)
	}
	if err := st.Save(ctx, testReceiptWithID("RC-0002")); err != nil {
		t.Fatalf("Save 2 failed: %v", err)
	}
	latest, err := st.Latest(ctx)
	if err != nil {
		t.Fatalf("Latest failed: %v", err)
	}
	if latest.ID != "RC-0002" {
		t.Errorf("Latest = %q, want RC-0002", latest.ID)
	}
	list, err := st.List(ctx, 10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("List len = %d, want 2", len(list))
	}
	if list[0].ID != "RC-0001" || list[1].ID != "RC-0002" {
		t.Errorf("List order = %q,%q, want RC-0001,RC-0002", list[0].ID, list[1].ID)
	}
}

func TestVerifyChainPasses(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	for _, id := range []string{"RC-0001", "RC-0002", "RC-0003"} {
		if err := st.Save(ctx, testReceiptWithID(id)); err != nil {
			t.Fatalf("Save %s failed: %v", id, err)
		}
	}
	v, err := st.VerifyChain(ctx, "")
	if err != nil {
		t.Fatalf("VerifyChain failed: %v", err)
	}
	if !v.OK {
		t.Errorf("VerifyChain OK = false, reason %q seq %d", v.Reason, v.FailedSequence)
	}
}

func TestSaveIsIdempotentForSameContent(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	r := testReceiptWithID("RC-0001")
	if err := st.Save(ctx, r); err != nil {
		t.Fatalf("first Save failed: %v", err)
	}
	if err := st.Save(ctx, r); err != nil {
		t.Fatalf("second Save same content failed: %v", err)
	}
	list, err := st.List(ctx, 10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("idempotent Save should not duplicate, list len = %d, want 1", len(list))
	}
}

func TestSaveRejectsDuplicateIDWithDifferentContent(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	if err := st.Save(ctx, testReceiptWithID("RC-0001")); err != nil {
		t.Fatalf("first Save failed: %v", err)
	}
	r2 := testReceiptWithID("RC-0001")
	r2.DurationMS = 999
	if err := st.Save(ctx, r2); err == nil {
		t.Errorf("expected error for duplicate ID with different content, got nil")
	}
}

func TestVerifyChainFailsOnCorruptLine(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	if err := st.Save(ctx, testReceiptWithID("RC-0001")); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	eventsPath := filepath.Join(st.Dir, "events.jsonl")
	f, err := os.OpenFile(eventsPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("open events failed: %v", err)
	}
	if _, err := f.WriteString("not json\n"); err != nil {
		t.Fatalf("append corrupt failed: %v", err)
	}
	f.Close()
	v, err := st.VerifyChain(ctx, "")
	if err != nil {
		t.Fatalf("VerifyChain returned error: %v", err)
	}
	if v.OK {
		t.Errorf("VerifyChain should fail on corrupt line")
	}
	if v.FailedSequence != 2 {
		t.Errorf("FailedSequence = %d, want 2", v.FailedSequence)
	}
}

func TestVerifyChainFailsAfterTamper(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	for _, id := range []string{"RC-0001", "RC-0002"} {
		if err := st.Save(ctx, testReceiptWithID(id)); err != nil {
			t.Fatalf("Save %s failed: %v", id, err)
		}
	}
	eventsPath := filepath.Join(st.Dir, "events.jsonl")
	data, err := os.ReadFile(eventsPath)
	if err != nil {
		t.Fatalf("read events failed: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 events, got %d", len(lines))
	}
	// Tamper with first line payload hash.
	tampered := strings.Replace(lines[0], "a", "b", 1)
	out := tampered + "\n" + lines[1] + "\n"
	if err := os.WriteFile(eventsPath, []byte(out), 0644); err != nil {
		t.Fatalf("write tampered failed: %v", err)
	}
	v, _ := st.VerifyChain(ctx, "")
	if v.OK {
		t.Errorf("VerifyChain should fail after tamper")
	}
}

func TestVerifyChainFailsOnReorder(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	for _, id := range []string{"RC-0001", "RC-0002"} {
		if err := st.Save(ctx, testReceiptWithID(id)); err != nil {
			t.Fatalf("Save %s failed: %v", id, err)
		}
	}
	eventsPath := filepath.Join(st.Dir, "events.jsonl")
	data, _ := os.ReadFile(eventsPath)
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	swapped := lines[1] + "\n" + lines[0] + "\n"
	os.WriteFile(eventsPath, []byte(swapped), 0644)
	v, _ := st.VerifyChain(ctx, "")
	if v.OK {
		t.Errorf("VerifyChain should fail on reorder")
	}
}

func TestVerifyChainFailsOnDelete(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	for _, id := range []string{"RC-0001", "RC-0002", "RC-0003"} {
		if err := st.Save(ctx, testReceiptWithID(id)); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}
	eventsPath := filepath.Join(st.Dir, "events.jsonl")
	data, _ := os.ReadFile(eventsPath)
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	kept := lines[0] + "\n" + lines[2] + "\n"
	os.WriteFile(eventsPath, []byte(kept), 0644)
	v, _ := st.VerifyChain(ctx, "")
	if v.OK {
		t.Errorf("VerifyChain should fail on delete")
	}
}

func TestConcurrentAppends(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	const n = 10
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("RC-%04X", 0x1000+i)
			errs[i] = st.Save(ctx, testReceiptWithID(id))
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("concurrent Save %d failed: %v", i, err)
		}
	}
	list, err := st.List(ctx, 100)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != n {
		t.Fatalf("List len = %d, want %d", len(list), n)
	}
	v, err := st.VerifyChain(ctx, "")
	if err != nil {
		t.Fatalf("VerifyChain failed: %v", err)
	}
	if !v.OK {
		t.Errorf("chain should verify after concurrent appends, reason %q", v.Reason)
	}
}

func TestNoTempFilesRemain(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	if err := st.Save(ctx, testReceiptWithID("RC-0001")); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(st.Dir, "receipts"))
	if err != nil {
		t.Fatalf("read receipts dir failed: %v", err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("temp file remains: %s", e.Name())
		}
	}
}
