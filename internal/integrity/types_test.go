package integrity

import (
	"testing"

	"github.com/receipts/receipts/internal/model"
)

func TestFillFlagIDs(t *testing.T) {
	flags := []model.Flag{
		{ID: "F-001", Code: "A"},
		{ID: "", Code: "B"},
		{ID: "", Code: "C"},
	}
	FillFlagIDs(flags)
	if flags[1].ID != "F-002" || flags[2].ID != "F-003" {
		t.Errorf("ids = %q %q, want F-002 F-003", flags[1].ID, flags[2].ID)
	}
	if flags[1].EvidenceRefs == nil {
		t.Errorf("refs must be non nil")
	}
}
