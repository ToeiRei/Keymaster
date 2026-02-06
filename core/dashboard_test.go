// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"reflect"
	"testing"
	"time"

	"github.com/toeirei/keymaster/internal/core/model"
	"github.com/uptrace/bun"
)

// fakeStore implements db.Store methods needed for dashboard
type fakeStore struct {
	accounts []model.Account
	sysKey   *model.SystemKey
	logs     []model.AuditLogEntry
}

func (f fakeStore) GetAllAccounts() ([]model.Account, error)              { return f.accounts, nil }
func (f fakeStore) GetActiveSystemKey() (*model.SystemKey, error)         { return f.sysKey, nil }
func (f fakeStore) GetAllAuditLogEntries() ([]model.AuditLogEntry, error) { return f.logs, nil }

// Stub methods to satisfy db.Store interface (not used by BuildDashboardData)
func (f fakeStore) GetAllPublicKeys() ([]model.PublicKey, error)                   { return nil, nil }
func (f fakeStore) GetGlobalPublicKeys() ([]model.PublicKey, error)                { return nil, nil }
func (f fakeStore) GetKeysForAccount(accountID int) ([]model.PublicKey, error)     { return nil, nil }
func (f fakeStore) AddAccount(username, hostname, label, tags string) (int, error) { return 0, nil }
func (f fakeStore) DeleteAccount(id int) error                                     { return nil }
func (f fakeStore) UpdateAccountSerial(id, serial int) error                       { return nil }
func (f fakeStore) ToggleAccountStatus(id int) error                               { return nil }
func (f fakeStore) UpdateAccountLabel(id int, label string) error                  { return nil }
func (f fakeStore) UpdateAccountHostname(id int, hostname string) error            { return nil }
func (f fakeStore) UpdateAccountTags(id int, tags string) error                    { return nil }
func (f fakeStore) GetAllActiveAccounts() ([]model.Account, error)                 { return nil, nil }
func (f fakeStore) UpdateAccountIsDirty(id int, dirty bool) error                  { return nil }
func (f fakeStore) GetKnownHostKey(hostname string) (string, error)                { return "", nil }
func (f fakeStore) AddKnownHostKey(hostname, key string) error                     { return nil }
func (f fakeStore) CreateSystemKey(publicKey, privateKey string) (int, error)      { return 0, nil }
func (f fakeStore) RotateSystemKey(publicKey, privateKey string) (int, error)      { return 0, nil }
func (f fakeStore) GetSystemKeyBySerial(serial int) (*model.SystemKey, error)      { return nil, nil }
func (f fakeStore) HasSystemKeys() (bool, error)                                   { return false, nil }
func (f fakeStore) SearchAccounts(query string) ([]model.Account, error)           { return nil, nil }
func (f fakeStore) LogAction(action, details string) error                         { return nil }
func (f fakeStore) SaveBootstrapSession(id, username, hostname, label, tags, tempPublicKey string, expiresAt time.Time, status string) error {
	return nil
}
func (f fakeStore) GetBootstrapSession(id string) (*model.BootstrapSession, error)   { return nil, nil }
func (f fakeStore) DeleteBootstrapSession(id string) error                           { return nil }
func (f fakeStore) UpdateBootstrapSessionStatus(id string, status string) error      { return nil }
func (f fakeStore) GetExpiredBootstrapSessions() ([]*model.BootstrapSession, error)  { return nil, nil }
func (f fakeStore) GetOrphanedBootstrapSessions() ([]*model.BootstrapSession, error) { return nil, nil }
func (f fakeStore) ExportDataForBackup() (*model.BackupData, error)                  { return nil, nil }
func (f fakeStore) ImportDataFromBackup(*model.BackupData) error                     { return nil }
func (f fakeStore) IntegrateDataFromBackup(*model.BackupData) error                  { return nil }
func (f fakeStore) BunDB() *bun.DB                                                   { return nil }

func (f fakeStore) AddPublicKeyAndGetModel(algorithm, keyData, comment string, isGlobal bool, expiresAt time.Time) (*model.PublicKey, error) {
	return nil, nil
}

func TestBuildDashboardData(t *testing.T) {
	accounts := []model.Account{
		{ID: 1, IsActive: true, Serial: 100},
		{ID: 2, IsActive: false, Serial: 0},
		{ID: 3, IsActive: true, Serial: 99},
	}
	sys := &model.SystemKey{Serial: 100}
	logs := []model.AuditLogEntry{{Timestamp: "t1", Username: "u", Action: "a", Details: "d"}}

	store := fakeStore{
		accounts: accounts,
		sysKey:   sys,
		logs:     logs,
	}

	out, err := BuildDashboardData(store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.AccountCount != 3 {
		t.Fatalf("expected 3 accounts, got %d", out.AccountCount)
	}
	if out.ActiveAccountCount != 2 {
		t.Fatalf("expected 2 active accounts, got %d", out.ActiveAccountCount)
	}
	if out.HostsUpToDate != 1 || out.HostsOutdated != 1 {
		t.Fatalf("unexpected hosts up-to-date/outdated: %d/%d", out.HostsUpToDate, out.HostsOutdated)
	}
	// Note: PublicKeyCount and GlobalKeyCount are commented out in DashboardData
	// Remove these assertions as they test fields that no longer exist
	if out.SystemKeySerial != 100 {
		t.Fatalf("expected system key serial 100, got %d", out.SystemKeySerial)
	}
	if !reflect.DeepEqual(out.RecentLogs, logs) {
		t.Fatalf("unexpected recent logs")
	}
}
