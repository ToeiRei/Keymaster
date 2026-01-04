package db

import (
	"strconv"
	"time"

	"github.com/toeirei/keymaster/internal/model"
)

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

// FakeAccountManager is a minimal fake used by tests for account add/delete.
type FakeAccountManager struct {
	// Calls records method name and arguments for assertions.
	Calls [][3]string // method, arg1, arg2 (arg2 unused for Delete)
	// NextID is returned from AddAccount (default 1)
	NextID int
	// Err to return from methods if set
	Err error
}

func (f *FakeAccountManager) AddAccount(username, hostname, label, tags string) (int, error) {
	if f.Err != nil {
		return 0, f.Err
	}
	f.Calls = append(f.Calls, [3]string{"AddAccount", username, hostname})
	if f.NextID == 0 {
		f.NextID = 1
	}
	return f.NextID, nil
}

func (f *FakeAccountManager) DeleteAccount(id int) error {
	if f.Err != nil {
		return f.Err
	}
	f.Calls = append(f.Calls, [3]string{"DeleteAccount", strconv.Itoa(id), ""})
	return nil
}

// FakeKeyManager is a minimal fake used by tests to manage public keys.
type FakeKeyManager struct {
	Calls     [][3]string // method, arg1, arg2 (arg2 used for IDs as string)
	NextKeyID int
	Results   []model.PublicKey
	Err       error
}

func (f *FakeKeyManager) AddPublicKey(algorithm, keyData, comment string, isGlobal bool, expiresAt time.Time) error {
	if f.Err != nil {
		return f.Err
	}
	f.Calls = append(f.Calls, [3]string{"AddPublicKey", algorithm, comment})
	return nil
}

func (f *FakeKeyManager) AddPublicKeyAndGetModel(algorithm, keyData, comment string, isGlobal bool, expiresAt time.Time) (*model.PublicKey, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	id := f.NextKeyID
	if id == 0 {
		id = 1
	}
	pk := &model.PublicKey{ID: id, Algorithm: algorithm, KeyData: keyData, Comment: comment, IsGlobal: isGlobal}
	if !expiresAt.IsZero() {
		pk.ExpiresAt = expiresAt
	}
	f.Calls = append(f.Calls, [3]string{"AddPublicKeyAndGetModel", algorithm, comment})
	return pk, nil
}

func (f *FakeKeyManager) DeletePublicKey(id int) error {
	if f.Err != nil {
		return f.Err
	}
	f.Calls = append(f.Calls, [3]string{"DeletePublicKey", strconv.Itoa(id), ""})
	return nil
}

func (f *FakeKeyManager) TogglePublicKeyGlobal(id int) error {
	if f.Err != nil {
		return f.Err
	}
	f.Calls = append(f.Calls, [3]string{"TogglePublicKeyGlobal", strconv.Itoa(id), ""})
	return nil
}

func (f *FakeKeyManager) GetAllPublicKeys() ([]model.PublicKey, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	return f.Results, nil
}

func (f *FakeKeyManager) GetPublicKeyByComment(comment string) (*model.PublicKey, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	for _, p := range f.Results {
		if p.Comment == comment {
			return &p, nil
		}
	}
	return nil, nil
}

func (f *FakeKeyManager) GetGlobalPublicKeys() ([]model.PublicKey, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	var out []model.PublicKey
	for _, p := range f.Results {
		if p.IsGlobal {
			out = append(out, p)
		}
	}
	return out, nil
}

func (f *FakeKeyManager) AssignKeyToAccount(keyID, accountID int) error {
	if f.Err != nil {
		return f.Err
	}
	f.Calls = append(f.Calls, [3]string{"AssignKeyToAccount", strconv.Itoa(keyID), strconv.Itoa(accountID)})
	return nil
}

func (f *FakeKeyManager) UnassignKeyFromAccount(keyID, accountID int) error {
	if f.Err != nil {
		return f.Err
	}
	f.Calls = append(f.Calls, [3]string{"UnassignKeyFromAccount", strconv.Itoa(keyID), strconv.Itoa(accountID)})
	return nil
}

func (f *FakeKeyManager) GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	return f.Results, nil
}

func (f *FakeKeyManager) GetAccountsForKey(keyID int) ([]model.Account, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	return []model.Account{}, nil
}

func (f *FakeKeyManager) SetPublicKeyExpiry(id int, expiresAt time.Time) error {
	if f.Err != nil {
		return f.Err
	}
	f.Calls = append(f.Calls, [3]string{"SetPublicKeyExpiry", strconv.Itoa(id), expiresAt.UTC().Format(time.RFC3339)})
	return nil
}
