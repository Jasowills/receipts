package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/receipts/receipts/internal/model"
)

// ZeroHash is the previous hash for the first chain event (64 zeroes).
const ZeroHash = "0000000000000000000000000000000000000000000000000000000000000000"

// EventTypeReceiptSaved is the only event type written in v0.1.
const EventTypeReceiptSaved = "receipt_saved"

// Event is one append-only chain record in events.jsonl.
type Event struct {
	Sequence     int       `json:"sequence"`
	Timestamp    time.Time `json:"timestamp"`
	ReceiptID    string    `json:"receipt_id"`
	EventType    string    `json:"event_type"`
	PayloadHash  string    `json:"payload_hash"`
	PreviousHash string    `json:"previous_hash"`
	EventHash    string    `json:"event_hash"`
}

// computeEventHash returns hex SHA256 over canonical JSON of the event
// fields excluding event_hash itself. Field order is fixed by struct order.
func computeEventHash(sequence int, ts time.Time, receiptID, eventType, payloadHash, previousHash string) string {
	in := struct {
		Sequence     int       `json:"sequence"`
		Timestamp    time.Time `json:"timestamp"`
		ReceiptID    string    `json:"receipt_id"`
		EventType    string    `json:"event_type"`
		PayloadHash  string    `json:"payload_hash"`
		PreviousHash string    `json:"previous_hash"`
	}{
		Sequence:     sequence,
		Timestamp:    ts.UTC(),
		ReceiptID:    receiptID,
		EventType:    eventType,
		PayloadHash:  payloadHash,
		PreviousHash: previousHash,
	}
	b, _ := json.Marshal(in)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// hashOf returns hex SHA256 of data.
func hashOf(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// payloadHash returns hex SHA256 of the canonical receipt with chain
// event linkage zeroed. This avoids circular hashing: the event commits
// to receipt content, and the stored receipt points back at the event.
func payloadHash(r model.Receipt) (string, error) {
	c := r
	c.Chain.EventHash = ""
	c.Chain.PreviousHash = ""
	if c.Chain.Algorithm == "" {
		c.Chain.Algorithm = model.ChainAlgorithmSHA256
	}
	b, err := MarshalCanonical(c)
	if err != nil {
		return "", err
	}
	return hashOf(b), nil
}
