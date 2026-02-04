package core

import (
	"os"
	"testing"
	"time"

	"github.com/toeirei/keymaster/internal/core/db"
	"github.com/toeirei/keymaster/internal/model"
)

type testKeyReader struct{}

func (testKeyReader) GetActiveSystemKey() (*model.SystemKey, error) { return db.GetActiveSystemKey() }
func (testKeyReader) GetSystemKeyBySerial(serial int) (*model.SystemKey, error) {
	return db.GetSystemKeyBySerial(serial)
}
func (testKeyReader) GetAllPublicKeys() ([]model.PublicKey, error) {
	km := db.DefaultKeyManager()
	if km == nil {
		return nil, nil
	}
	return km.GetAllPublicKeys()
}

type testKeyLister struct{}

func (testKeyLister) GetGlobalPublicKeys() ([]model.PublicKey, error) {
	km := db.DefaultKeyManager()
	if km == nil {
		return nil, nil
	}
	return km.GetGlobalPublicKeys()
}
func (testKeyLister) GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	km := db.DefaultKeyManager()
	if km == nil {
		return nil, nil
	}
	return km.GetKeysForAccount(accountID)
}
func (testKeyLister) GetAllPublicKeys() ([]model.PublicKey, error) {
	km := db.DefaultKeyManager()
	if km == nil {
		return nil, nil
	}
	return km.GetAllPublicKeys()
}

// testAccountSerialUpdater is a tiny test helper implementing AccountSerialUpdater
// by delegating to the real DB helper. This keeps tests simple while avoiding
// core importing internal/db directly in production code.
type testAccountSerialUpdater struct{}

func (testAccountSerialUpdater) UpdateAccountSerial(accountID int, serial int) error {
	return db.UpdateAccountSerial(accountID, serial)
}

type testKeyImporter struct{}

func (testKeyImporter) AddPublicKeyAndGetModel(algorithm, keyData, comment string, isGlobal bool, expiresAt time.Time) (*model.PublicKey, error) {
	km := db.DefaultKeyManager()
	if km == nil {
		return nil, nil
	}
	return km.AddPublicKeyAndGetModel(algorithm, keyData, comment, isGlobal, expiresAt)
}

type testAuditWriter struct{}

func (testAuditWriter) LogAction(action, details string) error {
	if w := db.DefaultAuditWriter(); w != nil {
		return w.LogAction(action, details)
	}
	return nil
}

type testAccountManager struct{}

func (testAccountManager) DeleteAccount(id int) error {
	if m := db.DefaultAccountManager(); m != nil {
		return m.DeleteAccount(id)
	}
	return nil
}

func TestMain(m *testing.M) {
	SetDefaultKeyReader(testKeyReader{})
	SetDefaultKeyLister(testKeyLister{})
	// Wire a tiny account serial updater for tests that need to mutate accounts.
	SetDefaultAccountSerialUpdater(testAccountSerialUpdater{})
	SetDefaultKeyImporter(testKeyImporter{})
	SetDefaultAuditWriter(testAuditWriter{})
	SetDefaultAccountManager(testAccountManager{})
	os.Exit(m.Run())
}
