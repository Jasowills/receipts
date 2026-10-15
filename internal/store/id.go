package store

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
)

// ErrIDCollision is returned when a unique receipt ID cannot be found
// within the retry budget. Callers should retry with a fresh source
// or surface a store failure. IDs are display identifiers only,
// not security tokens.
var ErrIDCollision = errors.New("store: receipt ID collision, retries exhausted")

// DefaultMaxIDAttempts bounds collision retries for the 16 bit ID space.
const DefaultMaxIDAttempts = 100

// DefaultRandomUint16 returns two crypto random bytes as uint16.
// It is the production source for receipt IDs.
func DefaultRandomUint16() (uint16, error) {
	var b [2]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(b[:]), nil
}

// GenerateID returns a unique RC-XXXX display ID, retrying when
// exists reports a collision. It uses DefaultMaxIDAttempts.
func GenerateID(exists func(string) bool, next func() (uint16, error)) (string, error) {
	return GenerateIDWithRetries(exists, next, DefaultMaxIDAttempts)
}

// GenerateIDWithRetries returns a unique RC-XXXX ID within maxAttempts.
// A nil exists check never collides. A nil next source uses crypto rand.
func GenerateIDWithRetries(exists func(string) bool, next func() (uint16, error), maxAttempts int) (string, error) {
	if maxAttempts <= 0 {
		return "", ErrIDCollision
	}
	if next == nil {
		next = DefaultRandomUint16
	}
	for i := 0; i < maxAttempts; i++ {
		v, err := next()
		if err != nil {
			return "", err
		}
		id := fmt.Sprintf("RC-%04X", v)
		if exists == nil || !exists(id) {
			return id, nil
		}
	}
	return "", ErrIDCollision
}

// fixedSource is a deterministic uint16 source for tests.
type fixedSource struct {
	values []uint16
	at     int
}

// FixedSource returns a deterministic source that cycles through values.
// It is intended for tests that need predictable receipt IDs.
func FixedSource(values []uint16) *fixedSource {
	return &fixedSource{values: values}
}

// Next returns the next preset value, cycling when exhausted.
func (s *fixedSource) Next() (uint16, error) {
	if len(s.values) == 0 {
		return 0, errors.New("store: fixed source has no values")
	}
	v := s.values[s.at%len(s.values)]
	s.at++
	return v, nil
}
