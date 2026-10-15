package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/receipts/receipts/internal/model"
	"github.com/receipts/receipts/internal/store"
)

func saveShowFixture(t *testing.T, dir, id string) {
	t.Helper()
	st := store.New(filepath.Join(dir, ".receipts"))
	ts := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	dirty := true
	r := model.Receipt{
		SchemaVersion: model.SchemaVersion,
		ID:            id,
		CreatedAt:     ts,
		Agent:         model.Agent{Name: "opencode"},
		Tier:          model.Tier0,
		Repo:          model.Repo{Root: ".", BaselineConfidence: model.BaselineLow, DirtyBefore: model.DirtyUnknown, DirtyAfter: &dirty},
		Claims:        []model.Claim{},
		Flags:         []model.Flag{},
		Evidence:      []model.Evidence{},
		Warnings:      []string{},
		Summary:       model.Summary{},
		Chain:         model.Chain{Algorithm: model.ChainAlgorithmSHA256},
	}
	if err := st.Save(context.Background(), r); err != nil {
		t.Fatalf("fixture save failed: %v", err)
	}
}

func TestShowLatestTerminal(t *testing.T) {
	dir := t.TempDir()
	saveShowFixture(t, dir, "RC-0001")
	saveShowFixture(t, dir, "RC-0002")
	var out, errBuf bytes.Buffer
	code := runShow([]string{"--workdir", dir}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("runShow exit = %d, stderr %q", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "RC-0002") {
		t.Errorf("show latest should print RC-0002, got:\n%s", out.String())
	}
}

func TestShowByIDMarkdown(t *testing.T) {
	dir := t.TempDir()
	saveShowFixture(t, dir, "RC-0001")
	var out, errBuf bytes.Buffer
	code := runShow([]string{"RC-0001", "--workdir", dir, "--format", "md"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("runShow exit = %d, stderr %q", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "# Receipt RC-0001") {
		t.Errorf("markdown show should contain header, got:\n%s", out.String())
	}
}

func TestShowJSONFormat(t *testing.T) {
	dir := t.TempDir()
	saveShowFixture(t, dir, "RC-0001")
	var out, errBuf bytes.Buffer
	code := runShow([]string{"RC-0001", "--workdir", dir, "--format", "json"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("runShow exit = %d, stderr %q", code, errBuf.String())
	}
	if !strings.Contains(out.String(), `"id":"RC-0001"`) {
		t.Errorf("json show should contain id, got:\n%s", out.String())
	}
}

func TestShowUnknownIDFails(t *testing.T) {
	dir := t.TempDir()
	saveShowFixture(t, dir, "RC-0001")
	var out, errBuf bytes.Buffer
	code := runShow([]string{"RC-9999", "--workdir", dir}, &out, &errBuf)
	if code != 4 {
		t.Errorf("unknown ID exit = %d, want 4", code)
	}
}

func TestShowMissingStoreFails(t *testing.T) {
	dir := t.TempDir()
	var out, errBuf bytes.Buffer
	code := runShow([]string{"--workdir", dir}, &out, &errBuf)
	if code != 4 {
		t.Errorf("missing store exit = %d, want 4", code)
	}
}

func TestShowInvalidFormatFails(t *testing.T) {
	dir := t.TempDir()
	saveShowFixture(t, dir, "RC-0001")
	var out, errBuf bytes.Buffer
	code := runShow([]string{"--workdir", dir, "--format", "pdf"}, &out, &errBuf)
	if code != 2 {
		t.Errorf("invalid format exit = %d, want 2", code)
	}
}

func TestShowOutFile(t *testing.T) {
	dir := t.TempDir()
	saveShowFixture(t, dir, "RC-0001")
	outPath := filepath.Join(dir, "out", "receipt.md")
	var out, errBuf bytes.Buffer
	code := runShow([]string{"RC-0001", "--workdir", dir, "--format", "md", "--out", outPath}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("runShow exit = %d, stderr %q", code, errBuf.String())
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read out file failed: %v", err)
	}
	if !strings.Contains(string(data), "# Receipt RC-0001") {
		t.Errorf("out file should contain receipt, got:\n%s", data)
	}
}

func TestShowViaRunDispatch(t *testing.T) {
	dir := t.TempDir()
	saveShowFixture(t, dir, "RC-0001")
	var out, errBuf bytes.Buffer
	code := Run([]string{"show", "--workdir", dir}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("Run show exit = %d, stderr %q", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "RC-0001") {
		t.Errorf("Run show should print receipt, got:\n%s", out.String())
	}
}

func TestShowHTMLFormat(t *testing.T) {
	dir := t.TempDir()
	saveShowFixture(t, dir, "RC-0001")
	var out, errBuf bytes.Buffer
	code := runShow([]string{"RC-0001", "--workdir", dir, "--format", "html"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit = %d, stderr %q", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "<html") || !strings.Contains(out.String(), "RC-0001") {
		t.Errorf("html output missing page or id")
	}
}

func TestShowSVGFormat(t *testing.T) {
	dir := t.TempDir()
	saveShowFixture(t, dir, "RC-0001")
	var out, errBuf bytes.Buffer
	code := runShow([]string{"RC-0001", "--workdir", dir, "--format", "svg"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit = %d, stderr %q", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "<svg") {
		t.Errorf("svg output missing svg root")
	}
}
