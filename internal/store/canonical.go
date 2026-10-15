package store

import (
	"encoding/json"
	"sort"

	"github.com/receipts/receipts/internal/model"
)

// MarshalCanonical returns deterministic canonical JSON for a receipt.
// It normalizes nil slices to [], sorts claims, evidence, and flags by ID,
// converts timestamps to UTC, validates against the model contract, then
// marshals with Go struct field order and sorted map keys. Warnings keep
// input order. It returns an error for invalid receipts.
func MarshalCanonical(r model.Receipt) ([]byte, error) {
	n := normalizeReceipt(r)
	if err := model.ValidateReceipt(n); err != nil {
		return nil, err
	}
	return json.Marshal(n)
}

// UnmarshalCanonical decodes receipt JSON and validates the result.
func UnmarshalCanonical(data []byte) (model.Receipt, error) {
	var r model.Receipt
	if err := json.Unmarshal(data, &r); err != nil {
		return model.Receipt{}, err
	}
	if err := model.ValidateReceipt(r); err != nil {
		return model.Receipt{}, err
	}
	return r, nil
}

func normalizeReceipt(r model.Receipt) model.Receipt {
	out := r
	out.CreatedAt = r.CreatedAt.UTC()
	// Use empty non nil base so empty inputs stay [] not null in JSON.
	// append to a nil base with zero elements would return nil.
	out.Claims = append([]model.Claim{}, r.Claims...)
	out.Flags = append([]model.Flag{}, r.Flags...)
	out.Evidence = append([]model.Evidence{}, r.Evidence...)
	out.Warnings = append([]string{}, r.Warnings...)
	for i := range out.Claims {
		out.Claims[i].EvidenceRefs = append([]string{}, out.Claims[i].EvidenceRefs...)
		if out.Claims[i].Source.Timestamp != nil {
			ts := out.Claims[i].Source.Timestamp.UTC()
			out.Claims[i].Source.Timestamp = &ts
		}
	}
	for i := range out.Flags {
		out.Flags[i].EvidenceRefs = append([]string{}, out.Flags[i].EvidenceRefs...)
	}
	for i := range out.Evidence {
		out.Evidence[i].Timestamp = out.Evidence[i].Timestamp.UTC()
	}
	sort.Slice(out.Claims, func(a, b int) bool { return out.Claims[a].ID < out.Claims[b].ID })
	sort.Slice(out.Evidence, func(a, b int) bool { return out.Evidence[a].ID < out.Evidence[b].ID })
	sort.Slice(out.Flags, func(a, b int) bool { return out.Flags[a].ID < out.Flags[b].ID })
	return out
}
