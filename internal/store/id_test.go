package store

import (
	"errors"
	"regexp"
	"testing"
)

var idPattern = regexp.MustCompile(`^RC-[A-F0-9]{4}$`)

func TestGenerateIDFormat(t *testing.T) {
	src := FixedSource([]uint16{0x4F2A})
	id, err := GenerateID(func(string) bool { return false }, src.Next)
	if err != nil {
		t.Fatalf("GenerateID failed: %v", err)
	}
	if id != "RC-4F2A" {
		t.Errorf("id = %q, want RC-4F2A", id)
	}
	if !idPattern.MatchString(id) {
		t.Errorf("id %q does not match pattern", id)
	}
}

func TestGenerateIDRetriesOnCollision(t *testing.T) {
	src := FixedSource([]uint16{0x0001, 0x0002})
	exists := func(id string) bool { return id == "RC-0001" }
	id, err := GenerateID(exists, src.Next)
	if err != nil {
		t.Fatalf("GenerateID failed: %v", err)
	}
	if id != "RC-0002" {
		t.Errorf("id = %q, want RC-0002 after collision retry", id)
	}
}

func TestGenerateIDFailsWhenExhausted(t *testing.T) {
	src := FixedSource([]uint16{0x0001, 0x0001, 0x0001})
	exists := func(string) bool { return true }
	_, err := GenerateIDWithRetries(exists, src.Next, 3)
	if !errors.Is(err, ErrIDCollision) {
		t.Errorf("err = %v, want ErrIDCollision", err)
	}
}

func TestGenerateIDZeroValue(t *testing.T) {
	src := FixedSource([]uint16{0x0000})
	id, err := GenerateID(func(string) bool { return false }, src.Next)
	if err != nil {
		t.Fatalf("GenerateID failed: %v", err)
	}
	if id != "RC-0000" {
		t.Errorf("id = %q, want RC-0000", id)
	}
}
