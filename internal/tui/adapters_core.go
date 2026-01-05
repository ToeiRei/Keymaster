package tui

import (
	"time"

	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/ui"
)

// coreAccountReader adapts UI helpers to core.AccountReader.
type coreAccountReader struct{}

func (coreAccountReader) GetAllAccounts() ([]model.Account, error) { return ui.GetAllAccounts() }

// coreKeyReader adapts UI key manager to core.KeyReader.
type coreKeyReader struct{}

func (coreKeyReader) GetAllPublicKeys() ([]model.PublicKey, error) {
	km := ui.DefaultKeyManager()
	if km == nil {
		return nil, nil
	}
	return km.GetAllPublicKeys()
}

func (coreKeyReader) GetActiveSystemKey() (*model.SystemKey, error) { return ui.GetActiveSystemKey() }

// coreAuditReader adapts UI audit helpers to core.AuditReader.
type coreAuditReader struct{}

func (coreAuditReader) GetAllAuditLogEntries() ([]model.AuditLogEntry, error) {
	return ui.GetAllAuditLogEntries()
}

// coreAuditor adapts the TUI package-level audit helper to core.Auditor.
type coreAuditor struct{}

func (coreAuditor) LogAction(action, details string) error {
	return logAction(action, details)
}

// coreSystemKeyStore adapts UI system key helpers to core.SystemKeyStore.
type coreSystemKeyStore struct{}

func (coreSystemKeyStore) CreateSystemKey(publicKey, privateKey string) (int, error) {
	return ui.CreateSystemKey(publicKey, privateKey)
}

func (coreSystemKeyStore) RotateSystemKey(publicKey, privateKey string) (int, error) {
	return ui.RotateSystemKey(publicKey, privateKey)
}

// coreSessionStore adapts UI session helpers to core.SessionStore.
type coreSessionStore struct{}

func (coreSessionStore) SaveBootstrapSession(id, username, hostname, label, tags, tempPublicKey string, expiresAt time.Time, status string) error {
	return ui.SaveBootstrapSession(id, username, hostname, label, tags, tempPublicKey, expiresAt, status)
}

func (coreSessionStore) GetBootstrapSession(id string) (*model.BootstrapSession, error) {
	return ui.GetBootstrapSession(id)
}

func (coreSessionStore) DeleteBootstrapSession(id string) error {
	return ui.DeleteBootstrapSession(id)
}

func (coreSessionStore) UpdateBootstrapSessionStatus(id string, status string) error {
	return ui.UpdateBootstrapSessionStatus(id, status)
}

func (coreSessionStore) GetExpiredBootstrapSessions() ([]*model.BootstrapSession, error) {
	return ui.GetExpiredBootstrapSessions()
}

func (coreSessionStore) GetOrphanedBootstrapSessions() ([]*model.BootstrapSession, error) {
	return ui.GetOrphanedBootstrapSessions()
}
