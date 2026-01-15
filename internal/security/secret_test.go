package security

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestSecretRedactionAndJSON(t *testing.T) {
	s := FromString("supersecret")
	if fmt.Sprintf("%v", s) != "[SECRET]" {
		t.Fatalf("unexpected fmt output: %q", fmt.Sprintf("%v", s))
	}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if string(b) != "\"[SECRET]\"" {
		t.Fatalf("unexpected json marshal: %s", string(b))
	}
}

func TestSecretZero(t *testing.T) {
	s := FromString("abc123")
	// Zero the underlying secret
	(&s).Zero()
	b := s.Bytes()
	for i := range b {
		if b[i] != 0 {
			t.Fatalf("expected zeroed byte at index %d, got %d", i, b[i])
		}
	}
}
