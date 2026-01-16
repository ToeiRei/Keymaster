// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"testing"
	"time"

	"github.com/toeirei/keymaster/internal/model"
)

// Ensure basic package-level wrappers delegate to the configured Store.
func TestAccountAndKeyWrappers(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		// Add an account via store helper and verify wrapper returns it.
		id, err := s.AddAccount("deploy", "host.example", "mylabel", "env:prod")
		if err != nil {
			t.Fatalf("AddAccount failed: %v", err)
		}

		accs, err := GetAllAccounts()
		if err != nil {
			t.Fatalf("GetAllAccounts failed: %v", err)
		}
		if len(accs) != 1 || accs[0].ID != id {
			t.Fatalf("unexpected accounts: %+v", accs)
		}

		// Update serial and verify
		if err := UpdateAccountSerial(id, 42); err != nil {
			t.Fatalf("UpdateAccountSerial failed: %v", err)
		}
		accs2, _ := GetAllAccounts()
		var found *model.Account
		for i := range accs2 {
			if accs2[i].ID == id {
				found = &accs2[i]
				break
			}
		}
		if found == nil || found.Serial != 42 {
			t.Fatalf("serial not updated: %+v", found)
		}

		// Toggle status
		prev := found.IsActive
		if err := ToggleAccountStatus(id); err != nil {
			t.Fatalf("ToggleAccountStatus failed: %v", err)
		}
		accs3, _ := GetAllAccounts()
		var found2 *model.Account
		for i := range accs3 {
			if accs3[i].ID == id {
				found2 = &accs3[i]
				break
			}
		}
		if found2 == nil || found2.IsActive == prev {
			t.Fatalf("ToggleAccountStatus did not change IsActive: before=%v after=%v", prev, found2)
		}

		// Update label/hostname/tags
		if err := UpdateAccountLabel(id, "newlabel"); err != nil {
			t.Fatalf("UpdateAccountLabel: %v", err)
		}
		if err := UpdateAccountHostname(id, "new-host"); err != nil {
			t.Fatalf("UpdateAccountHostname: %v", err)
		}
		if err := UpdateAccountTags(id, "k:v"); err != nil {
			t.Fatalf("UpdateAccountTags: %v", err)
		}
		accs4, _ := GetAllAccounts()
		var found3 *model.Account
		for i := range accs4 {
			if accs4[i].ID == id {
				found3 = &accs4[i]
				break
			}
		}
		if found3 == nil || found3.Label != "newlabel" || found3.Hostname != "new-host" || found3.Tags != "k:v" {
			t.Fatalf("updates not applied: %+v", found3)
		}

		// Known host key operations
		if err := AddKnownHostKey("host.example", "ssh-ed25519 AAAA"); err != nil {
			t.Fatalf("AddKnownHostKey: %v", err)
		}
		k, err := GetKnownHostKey("host.example")
		if err != nil {
			t.Fatalf("GetKnownHostKey: %v", err)
		}
		if k == "" {
			t.Fatalf("expected known host key, got empty")
		}

		// System key lifecycle
		serial, err := CreateSystemKey("ssh-ed25519 AAAA", "private-data")
		if err != nil {
			t.Fatalf("CreateSystemKey: %v", err)
		}
		if serial <= 0 {
			t.Fatalf("invalid serial: %d", serial)
		}
		sk, err := GetActiveSystemKey()
		if err != nil {
			t.Fatalf("GetActiveSystemKey: %v", err)
		}
		if sk == nil {
			t.Fatalf("expected active system key")
		}

		// Secret conversion
		sec := SecretFromModelSystemKey(sk)
		if sec == nil {
			t.Fatalf("SecretFromModelSystemKey returned nil")
		}
		// Ensure bytes match the private key
		if string(sec.Bytes()) != "private-data" {
			t.Fatalf("secret contents mismatch: %q", string(sec.Bytes()))
		}

		// GetActiveSystemKeySecret wrapper
		ssec, sk2, err := GetActiveSystemKeySecret()
		if err != nil {
			t.Fatalf("GetActiveSystemKeySecret error: %v", err)
		}
		if sk2 == nil || ssec == nil {
			t.Fatalf("GetActiveSystemKeySecret returned nils")
		}
		if string(ssec.Bytes()) != "private-data" {
			t.Fatalf("secret mismatch from wrapper")
		}

		// HasSystemKeys
		has, err := HasSystemKeys()
		if err != nil {
			t.Fatalf("HasSystemKeys: %v", err)
		}
		if !has {
			t.Fatalf("expected HasSystemKeys true")
		}

		// Audit log: LogAction and GetAllAuditLogEntries
		if err := LogAction("TEST", "details"); err != nil {
			t.Fatalf("LogAction: %v", err)
		}
		entries, err := GetAllAuditLogEntries()
		if err != nil {
			t.Fatalf("GetAllAuditLogEntries: %v", err)
		}
		if len(entries) == 0 {
			t.Fatalf("expected audit entries, got none")
		}
		foundEntry := false
		for _, e := range entries {
			if e.Action == "TEST" && e.Details == "details" {
				foundEntry = true
				break
			}
		}
		if !foundEntry {
			t.Fatalf("expected audit entry not found: %+v", entries)
		}

		// Export/Import backup paths (basic round-trip)
		b, err := ExportDataForBackup()
		if err != nil {
			t.Fatalf("ExportDataForBackup: %v", err)
		}
		if b == nil {
			t.Fatalf("expected backup data")
		}
		if err := ImportDataFromBackup(b); err != nil {
			t.Fatalf("ImportDataFromBackup: %v", err)
		}
		if err := IntegrateDataFromBackup(b); err != nil {
			t.Fatalf("IntegrateDataFromBackup: %v", err)
		}

		// Save/Get/Delete bootstrap session
		expires := time.Now().Add(1 * time.Hour)
		if err := SaveBootstrapSession("sid1", "user", "h", "lbl", "t", "tmpkey", expires, "active"); err != nil {
			t.Fatalf("SaveBootstrapSession: %v", err)
		}
		bs, err := GetBootstrapSession("sid1")
		if err != nil {
			t.Fatalf("GetBootstrapSession: %v", err)
		}
		if bs == nil || bs.Username != "user" {
			t.Fatalf("unexpected bootstrap session: %+v", bs)
		}
		if err := DeleteBootstrapSession("sid1"); err != nil {
			t.Fatalf("DeleteBootstrapSession: %v", err)
		}
	})
}

func TestSecretFromModelSystemKey_Nil(t *testing.T) {
	var sk *model.SystemKey
	if sec := SecretFromModelSystemKey(sk); sec != nil {
		t.Fatalf("expected nil secret for nil model.SystemKey")
	}
}
