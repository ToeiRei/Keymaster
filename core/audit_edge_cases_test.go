// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package core

import (
	"testing"

	"github.com/toeirei/keymaster/core/model"
	"github.com/toeirei/keymaster/core/state"
	"github.com/toeirei/keymaster/i18n"
)

// fakeKeyReaderForAudit is a minimal test implementation of KeyReader interface
// for audit edge case testing.
type fakeKeyReaderForAudit struct {
	returnNilKey bool
	returnErr    error
}

func (f *fakeKeyReaderForAudit) GetAllPublicKeys() ([]model.PublicKey, error) {
	return nil, nil
}

func (f *fakeKeyReaderForAudit) GetActiveSystemKey() (*model.SystemKey, error) {
	if f.returnErr != nil {
		return nil, f.returnErr
	}
	if f.returnNilKey {
		return nil, nil
	}
	return &model.SystemKey{Serial: 1}, nil
}

func (f *fakeKeyReaderForAudit) GetSystemKeyBySerial(serial int) (*model.SystemKey, error) {
	if f.returnErr != nil {
		return nil, f.returnErr
	}
	if f.returnNilKey {
		return nil, nil
	}
	return &model.SystemKey{Serial: serial}, nil
}

// TestAuditAccountStrict_NoSerial tests AuditAccountStrict with account having zero serial.
func TestAuditAccountStrict_NoSerial(t *testing.T) {
	i18n.Init("en")
	acct := model.Account{ID: 1, Username: "test", Hostname: "host.com", Serial: 0}
	err := AuditAccountStrict(acct)
	if err == nil {
		t.Fatalf("expected error for zero serial, got nil")
	}
	// Error should mention "not deployed" or "serial is 0"
	if err.Error() == "" {
		t.Fatalf("expected non-empty error message")
	}
}

// TestAuditAccountStrict_NoKeyReader tests AuditAccountStrict when KeyReader is not available.
func TestAuditAccountStrict_NoKeyReader(t *testing.T) {
	i18n.Init("en")
	origReader := DefaultKeyReader()
	SetDefaultKeyReader(nil)
	defer SetDefaultKeyReader(origReader)

	acct := model.Account{ID: 1, Username: "test", Hostname: "host.com", Serial: 5}
	err := AuditAccountStrict(acct)
	if err == nil {
		t.Fatalf("expected error when KeyReader is nil, got nil")
	}
}

// TestAuditAccountStrict_SerialKeyNotFound tests AuditAccountStrict when serial key doesn't exist.
// Uses a fake reader that returns nil for missing keys.
func TestAuditAccountStrict_SerialKeyNotFound(t *testing.T) {
	i18n.Init("en")
	fakeReader := &fakeKeyReaderForAudit{returnNilKey: true}
	SetDefaultKeyReader(fakeReader)
	defer SetDefaultKeyReader(nil)

	acct := model.Account{ID: 1, Username: "test", Hostname: "host.com", Serial: 5}
	err := AuditAccountStrict(acct)
	if err == nil {
		t.Fatalf("expected error when key not found")
	}
}

// TestAuditAccountSerial_NoSerial tests AuditAccountSerial with account having zero serial.
func TestAuditAccountSerial_NoSerial(t *testing.T) {
	i18n.Init("en")
	acct := model.Account{ID: 1, Username: "test", Hostname: "host.com", Serial: 0}
	err := AuditAccountSerial(acct)
	if err == nil {
		t.Fatalf("expected error for zero serial, got nil")
	}
}

// TestAuditAccountSerial_NoKeyReader tests AuditAccountSerial when KeyReader is not available.
func TestAuditAccountSerial_NoKeyReader(t *testing.T) {
	i18n.Init("en")
	origReader := DefaultKeyReader()
	SetDefaultKeyReader(nil)
	defer SetDefaultKeyReader(origReader)

	acct := model.Account{ID: 1, Username: "test", Hostname: "host.com", Serial: 5}
	err := AuditAccountSerial(acct)
	if err == nil {
		t.Fatalf("expected error when KeyReader is nil")
	}
}

// TestAuditAccountSerial_KeyNotFound tests AuditAccountSerial when the serial key doesn't exist.
func TestAuditAccountSerial_KeyNotFound(t *testing.T) {
	i18n.Init("en")
	fakeReader := &fakeKeyReaderForAudit{returnNilKey: true}
	SetDefaultKeyReader(fakeReader)
	defer SetDefaultKeyReader(nil)

	acct := model.Account{ID: 1, Username: "test", Hostname: "host.com", Serial: 5}
	err := AuditAccountSerial(acct)
	if err == nil {
		t.Fatalf("expected error when key not found")
	}
}

// TestPasswordCacheInteraction verifies that password cache state doesn't interfere
// with audit operations.
func TestPasswordCacheInteraction(t *testing.T) {
	i18n.Init("en")

	// Verify cache starts empty
	cached := state.PasswordCache.Get()
	if len(cached) != 0 {
		state.PasswordCache.Clear()
	}

	// Set a test password in the cache
	testPassword := []byte("testpass")
	state.PasswordCache.Set(testPassword)

	// Verify it's in cache
	cached = state.PasswordCache.Get()
	if len(cached) == 0 {
		t.Fatalf("password should be in cache after Set")
	}

	// Now audit should be able to read the cache without error
	// (though audit itself may fail due to missing dependencies)
	origReader := DefaultKeyReader()
	SetDefaultKeyReader(nil)
	defer SetDefaultKeyReader(origReader)

	acct := model.Account{ID: 1, Username: "test", Hostname: "host.com", Serial: 5}
	err := AuditAccountSerial(acct) // Will fail due to nil KeyReader
	if err == nil {
		t.Fatalf("expected audit to fail when KeyReader is nil")
	}

	// Cache can be cleared manually (in real code this happens in defer)
	state.PasswordCache.Clear()
	cached = state.PasswordCache.Get()
	if len(cached) != 0 {
		t.Fatalf("password cache should be empty after Clear")
	}
}
