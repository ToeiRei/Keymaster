// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package core

import (
	"testing"

	"github.com/toeirei/keymaster/core/model"
)

func TestBuildBootstrapDeploymentPlan_Basic(t *testing.T) {
	params := BootstrapParams{
		Username:       "alice",
		Hostname:       "host.example",
		Label:          "web",
		Tags:           "env:prod",
		SelectedKeyIDs: []int{10, 20},
	}

	sk := &model.SystemKey{Serial: 1, PublicKey: "ssh-ed25519 AAAA-key"}

	global := []model.PublicKey{{ID: 1, Algorithm: "ssh-ed25519", KeyData: "AAAA-1", Comment: "g1", IsGlobal: true}}
	account := []model.PublicKey{{ID: 2, Algorithm: "ssh-ed25519", KeyData: "AAAA-2", Comment: "a1", IsGlobal: false}}

	plan, err := BuildBootstrapDeploymentPlan(params, sk, global, account)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.Account.Username != "alice" || plan.Account.Hostname != "host.example" {
		t.Fatalf("unexpected account in plan: %#v", plan.Account)
	}
	if len(plan.SelectedKeyIDs) != 2 {
		t.Fatalf("expected 2 selected keys, got %d", len(plan.SelectedKeyIDs))
	}
	if plan.AuthorizedKeysBlob == "" {
		t.Fatalf("expected authorized keys content, got empty string")
	}
}
