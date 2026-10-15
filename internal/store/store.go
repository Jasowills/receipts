package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/receipts/receipts/internal/model"
)

// Store is a file backed receipt store rooted at Dir (.receipts).
// Layout: receipts/RC-XXXX.json, events.jsonl, tmp/.
// JSONL alone is sufficient for v0.1; no index is maintained.
// The hash chain is tamper evident only if history is not rewritten;
// it is not a signature and not a security boundary.
type Store struct {
	Dir string
	Now func() time.Time
}

// New returns a Store rooted at dir.
func New(dir string) *Store {
	return &Store{Dir: dir}
}

func (s *Store) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s *Store) receiptsDir() string { return filepath.Join(s.Dir, "receipts") }
func (s *Store) eventsPath() string  { return filepath.Join(s.Dir, "events.jsonl") }
func (s *Store) lockPath() string    { return filepath.Join(s.Dir, ".lock") }
func (s *Store) tmpDir() string      { return filepath.Join(s.Dir, "tmp") }

func (s *Store) ensureDirs() error {
	for _, d := range []string{s.Dir, s.receiptsDir(), s.tmpDir()} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	return nil
}

// withLock runs fn while holding a lock file with bounded wait.
// It uses O_CREATE|O_EXCL for portability on macOS and Linux.
func (s *Store) withLock(fn func() error) error {
	if err := s.ensureDirs(); err != nil {
		return err
	}
	const tries = 500
	const wait = 10 * time.Millisecond
	for i := 0; i < tries; i++ {
		f, err := os.OpenFile(s.lockPath(), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err == nil {
			f.Close()
			defer os.Remove(s.lockPath())
			return fn()
		}
		if !os.IsExist(err) {
			return err
		}
		time.Sleep(wait)
	}
	return fmt.Errorf("store: lock busy, remove stale %s if no process holds it", s.lockPath())
}

// Save validates and stores a receipt, appending one chain event.
// It requires receipt.ID to be set. Saving identical content twice is
// idempotent and appends no duplicate event. Saving different content
// under an existing ID fails. Receipt chain linkage is filled on write.
func (s *Store) Save(ctx context.Context, r model.Receipt) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.ID == "" {
		return fmt.Errorf("store: receipt id is required")
	}
	if r.Chain.Algorithm == "" {
		r.Chain.Algorithm = model.ChainAlgorithmSHA256
	}
	var saveErr error
	if err := s.withLock(func() error {
		saveErr = s.saveLocked(r)
		return saveErr
	}); err != nil {
		return err
	}
	return saveErr
}

func (s *Store) saveLocked(r model.Receipt) error {
	newPayload, err := payloadHash(r)
	if err != nil {
		return err
	}
	finalPath := filepath.Join(s.receiptsDir(), r.ID+".json")
	if existing, err := os.ReadFile(finalPath); err == nil {
		var er model.Receipt
		if uerr := json.Unmarshal(existing, &er); uerr == nil {
			if eph, perr := payloadHash(er); perr == nil && eph == newPayload {
				return nil
			}
		}
		return fmt.Errorf("store: duplicate receipt id %s with different content", r.ID)
	} else if !os.IsNotExist(err) {
		return err
	}

	events, err := s.readEventsLocked()
	if err != nil {
		return fmt.Errorf("store: cannot extend corrupt chain: %w", err)
	}
	sequence := len(events) + 1
	previous := ZeroHash
	if len(events) > 0 {
		previous = events[len(events)-1].EventHash
	}
	event := Event{
		Sequence:     sequence,
		Timestamp:    s.now().Truncate(time.Millisecond),
		ReceiptID:    r.ID,
		EventType:    EventTypeReceiptSaved,
		PayloadHash:  newPayload,
		PreviousHash: previous,
	}
	event.EventHash = computeEventHash(event.Sequence, event.Timestamp, event.ReceiptID, event.EventType, event.PayloadHash, event.PreviousHash)

	r.Chain.EventHash = event.EventHash
	r.Chain.PreviousHash = event.PreviousHash
	r.Chain.Algorithm = model.ChainAlgorithmSHA256
	finalBytes, err := MarshalCanonical(r)
	if err != nil {
		return err
	}
	if err := s.writeAtomic(finalPath, finalBytes); err != nil {
		return err
	}
	eb, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return appendLine(s.eventsPath(), eb)
}

func (s *Store) writeAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(s.receiptsDir(), ".tmp-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0644); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// Get returns one receipt by ID. Reads use atomic rename files
// so no lock is needed for single file reads.
func (s *Store) Get(_ context.Context, id string) (model.Receipt, error) {
	data, err := os.ReadFile(filepath.Join(s.receiptsDir(), id+".json"))
	if err != nil {
		if os.IsNotExist(err) {
			return model.Receipt{}, fmt.Errorf("store: receipt %s not found", id)
		}
		return model.Receipt{}, err
	}
	return UnmarshalCanonical(data)
}

// Latest returns the most recently saved receipt.
func (s *Store) Latest(ctx context.Context) (model.Receipt, error) {
	if err := ctx.Err(); err != nil {
		return model.Receipt{}, err
	}
	var out model.Receipt
	var lerr error
	if err := s.withLock(func() error {
		events, err := s.readEventsLocked()
		if err != nil {
			lerr = err
			return err
		}
		if len(events) == 0 {
			lerr = fmt.Errorf("store: no receipts")
			return lerr
		}
		out, lerr = s.getLocked(events[len(events)-1].ReceiptID)
		return lerr
	}); err != nil {
		return model.Receipt{}, lerr
	}
	return out, lerr
}

// List returns receipt summaries in chain order (oldest first).
// A positive limit keeps the most recent limit entries.
func (s *Store) List(ctx context.Context, limit int) ([]model.ReceiptSummary, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var out []model.ReceiptSummary
	var lerr error
	if err := s.withLock(func() error {
		events, err := s.readEventsLocked()
		if err != nil {
			lerr = err
			return err
		}
		if limit > 0 && len(events) > limit {
			events = events[len(events)-limit:]
		}
		for _, e := range events {
			r, err := s.getLocked(e.ReceiptID)
			if err != nil {
				lerr = err
				return err
			}
			out = append(out, model.ReceiptSummary{
				ID:           r.ID,
				CreatedAt:    r.CreatedAt,
				AgentName:    r.Agent.Name,
				Tier:         r.Tier,
				Verified:     r.Summary.Verified,
				Contradicted: r.Summary.Contradicted,
				Unverified:   r.Summary.Unverified,
				Unverifiable: r.Summary.Unverifiable,
				Flags:        r.Summary.Flags,
			})
		}
		return nil
	}); err != nil {
		return nil, lerr
	}
	return out, lerr
}

// VerifyChain validates sequence continuity, previous hash linkage,
// event hashes, and receipt payload hashes. It reports the first
// failing sequence and never rewrites history. The id argument is
// reserved for future per receipt verification and currently ignored.
func (s *Store) VerifyChain(ctx context.Context, _ string) (model.ChainVerification, error) {
	if err := ctx.Err(); err != nil {
		return model.ChainVerification{}, err
	}
	var out model.ChainVerification
	var verr error
	if err := s.withLock(func() error {
		out, verr = s.verifyLocked()
		return verr
	}); err != nil {
		return model.ChainVerification{}, verr
	}
	return out, verr
}

func (s *Store) verifyLocked() (model.ChainVerification, error) {
	lines, err := readLines(s.eventsPath())
	if err != nil {
		return model.ChainVerification{}, err
	}
	var prev string
	for i, line := range lines {
		seq := i + 1
		var e Event
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return model.ChainVerification{OK: false, FailedSequence: seq, Reason: "corrupt event"}, nil
		}
		if e.Sequence != seq {
			return model.ChainVerification{OK: false, FailedSequence: seq, Reason: "sequence gap"}, nil
		}
		wantPrev := ZeroHash
		if i > 0 {
			wantPrev = prev
		}
		if e.PreviousHash != wantPrev {
			return model.ChainVerification{OK: false, FailedSequence: seq, Reason: "previous hash mismatch"}, nil
		}
		got := computeEventHash(e.Sequence, e.Timestamp, e.ReceiptID, e.EventType, e.PayloadHash, e.PreviousHash)
		if got != e.EventHash {
			return model.ChainVerification{OK: false, FailedSequence: seq, Reason: "event hash mismatch"}, nil
		}
		r, err := s.getLocked(e.ReceiptID)
		if err != nil {
			return model.ChainVerification{OK: false, FailedSequence: seq, Reason: "receipt missing"}, nil
		}
		actual, err := payloadHash(r)
		if err != nil || actual != e.PayloadHash {
			return model.ChainVerification{OK: false, FailedSequence: seq, Reason: "payload hash mismatch"}, nil
		}
		prev = e.EventHash
	}
	return model.ChainVerification{OK: true}, nil
}

func (s *Store) readEventsLocked() ([]Event, error) {
	lines, err := readLines(s.eventsPath())
	if err != nil {
		return nil, err
	}
	out := make([]Event, 0, len(lines))
	for _, line := range lines {
		var e Event
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return nil, fmt.Errorf("store: corrupt event: %w", err)
		}
		out = append(out, e)
	}
	return out, nil
}

func (s *Store) getLocked(id string) (model.Receipt, error) {
	data, err := os.ReadFile(filepath.Join(s.receiptsDir(), id+".json"))
	if err != nil {
		if os.IsNotExist(err) {
			return model.Receipt{}, fmt.Errorf("store: receipt %s not found", id)
		}
		return model.Receipt{}, err
	}
	return UnmarshalCanonical(data)
}
