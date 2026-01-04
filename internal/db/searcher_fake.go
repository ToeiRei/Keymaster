package db

import "github.com/toeirei/keymaster/internal/model"

// FakeAccountSearcher is a minimal, configurable fake used by tests.
type FakeAccountSearcher struct {
	// Results to return from SearchAccounts. If nil, an empty slice is returned.
	Results []model.Account
	// Err to return from SearchAccounts if non-nil.
	Err error
}

// SearchAccounts implements AccountSearcher for the fake.
func (f *FakeAccountSearcher) SearchAccounts(query string) ([]model.Account, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	if f.Results == nil {
		return []model.Account{}, nil
	}
	return f.Results, nil
}

// FakeKeySearcher is a minimal, configurable fake used by tests for public
// key searches.
type FakeKeySearcher struct {
	// Results to return from SearchPublicKeys. If nil, an empty slice is returned.
	Results []model.PublicKey
	// Err to return from SearchPublicKeys if non-nil.
	Err error
}

// SearchPublicKeys implements KeySearcher for the fake.
func (f *FakeKeySearcher) SearchPublicKeys(query string) ([]model.PublicKey, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	if f.Results == nil {
		return []model.PublicKey{}, nil
	}
	return f.Results, nil
}

// FakeAuditSearcher is a minimal fake used by tests for audit lookups.
type FakeAuditSearcher struct {
	Results []model.AuditLogEntry
	Err     error
}

// GetAllAuditLogEntries implements AuditSearcher for the fake.
func (f *FakeAuditSearcher) GetAllAuditLogEntries() ([]model.AuditLogEntry, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	if f.Results == nil {
		return []model.AuditLogEntry{}, nil
	}
	return f.Results, nil
}

// FakeAuditWriter is a minimal fake used by tests for audit writes.
type FakeAuditWriter struct {
	// Calls records action/details tuples for assertions.
	Calls [][2]string
	// Err to return from LogAction if non-nil.
	Err error
}

// LogAction implements AuditWriter for the fake.
func (f *FakeAuditWriter) LogAction(action string, details string) error {
	if f.Err != nil {
		return f.Err
	}
	f.Calls = append(f.Calls, [2]string{action, details})
	return nil
}
