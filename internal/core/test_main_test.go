package core

import (
	"os"
	"testing"

	"github.com/toeirei/keymaster/internal/db"
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

func TestMain(m *testing.M) {
	SetDefaultKeyReader(testKeyReader{})
	SetDefaultKeyLister(testKeyLister{})
	os.Exit(m.Run())
}
