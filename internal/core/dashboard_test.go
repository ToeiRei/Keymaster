// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"reflect"
	"testing"

	"github.com/toeirei/keymaster/internal/model"
)

// fake implementations for readers
type fakeAccountReader struct{ accts []model.Account }

func (f fakeAccountReader) GetAllAccounts() ([]model.Account, error) { return f.accts, nil }

type fakeKeyReader struct {
	keys []model.PublicKey
	sys  *model.SystemKey
}

func (f fakeKeyReader) GetAllPublicKeys() ([]model.PublicKey, error)  { return f.keys, nil }
func (f fakeKeyReader) GetActiveSystemKey() (*model.SystemKey, error) { return f.sys, nil }

type fakeAuditReader struct{ entries []model.AuditLogEntry }

func (f fakeAuditReader) GetAllAuditLogEntries() ([]model.AuditLogEntry, error) {
	return f.entries, nil
}

func TestBuildDashboardData(t *testing.T) {
	accounts := []model.Account{
		{ID: 1, IsActive: true, Serial: 100},
		{ID: 2, IsActive: false, Serial: 0},
		{ID: 3, IsActive: true, Serial: 99},
	}
	keys := []model.PublicKey{
		{ID: 1, Algorithm: "ssh-ed25519", IsGlobal: true},
		{ID: 2, Algorithm: "ssh-rsa", IsGlobal: false},
	}
	sys := &model.SystemKey{Serial: 100}
	logs := []model.AuditLogEntry{{Timestamp: "t1", Username: "u", Action: "a", Details: "d"}}

	out, err := BuildDashboardData(fakeAccountReader{accounts}, fakeKeyReader{keys, sys}, fakeAuditReader{logs})
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
	if out.PublicKeyCount != 2 {
		t.Fatalf("expected 2 keys, got %d", out.PublicKeyCount)
	}
	if out.GlobalKeyCount != 1 {
		t.Fatalf("expected 1 global key, got %d", out.GlobalKeyCount)
	}
	if !reflect.DeepEqual(out.RecentLogs, logs) {
		t.Fatalf("unexpected recent logs")
	}
}
