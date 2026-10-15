package integrity

import (
	"strings"
	"testing"
)

func TestRedactSecrets(t *testing.T) {
	secret := "AKIAIOSFODNN7EXAMPLE"
	got := RedactSecrets("key=" + secret + " end")
	if strings.Contains(got, secret) {
		t.Errorf("secret must be redacted, got %q", got)
	}
	if !strings.Contains(got, "[redacted]") {
		t.Errorf("marker missing, got %q", got)
	}
	plain := "just a normal line of output"
	if RedactSecrets(plain) != plain {
		t.Errorf("plain text changed: %q", RedactSecrets(plain))
	}
	if RedactSecrets(`password = "changeme123"`) != `password = "changeme123"` {
		t.Errorf("placeholders must survive redaction")
	}
}
