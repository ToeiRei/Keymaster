package tui

import (
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
