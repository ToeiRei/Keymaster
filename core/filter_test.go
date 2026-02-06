// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package core

import (
	"testing"

	"github.com/toeirei/keymaster/core/model"
)

func TestFilterKeys(t *testing.T) {
	keys := []model.PublicKey{
		{ID: 1, Comment: "alice@example.com", Algorithm: "ed25519"},
		{ID: 2, Comment: "bob@example.com", Algorithm: "rsa"},
		{ID: 3, Comment: "service-key", Algorithm: "ecdsa"},
	}

	res := FilterKeys(keys, "ed25519")
	if len(res) != 1 || res[0].ID != 1 {
		t.Fatalf("expected 1 key with ed25519, got %v", res)
	}

	res = FilterKeys(keys, "bob")
	if len(res) != 1 || res[0].ID != 2 {
		t.Fatalf("expected 1 key matching 'bob', got %v", res)
	}

	res = FilterKeys(keys, "")
	if len(res) != 3 {
		t.Fatalf("expected all keys when query empty, got %d", len(res))
	}
}
