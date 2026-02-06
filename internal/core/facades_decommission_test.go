package core

import (
	"context"
	"errors"
	"testing"

	"github.com/toeirei/keymaster/internal/core/model"
	"github.com/toeirei/keymaster/internal/security"
)

type fakeStoreForDecom struct {
	sys  *model.SystemKey
	ferr error
}

func (f *fakeStoreForDecom) GetActiveSystemKey() (*model.SystemKey, error) { return f.sys, f.ferr }

// other Store methods (stubs) to satisfy the interface
func (f *fakeStoreForDecom) GetAccounts() ([]model.Account, error)          { return nil, nil }
func (f *fakeStoreForDecom) GetAllActiveAccounts() ([]model.Account, error) { return nil, nil }
func (f *fakeStoreForDecom) GetAllAccounts() ([]model.Account, error)       { return nil, nil }
func (f *fakeStoreForDecom) GetAccount(id int) (*model.Account, error)      { return nil, nil }
func (f *fakeStoreForDecom) AddAccount(username, hostname, label, tags string) (int, error) {
	return 0, nil
}
func (f *fakeStoreForDecom) DeleteAccount(accountID int) error                         { return nil }
func (f *fakeStoreForDecom) AssignKeyToAccount(keyID, accountID int) error             { return nil }
func (f *fakeStoreForDecom) UpdateAccountIsDirty(id int, dirty bool) error             { return nil }
func (f *fakeStoreForDecom) CreateSystemKey(publicKey, privateKey string) (int, error) { return 0, nil }
func (f *fakeStoreForDecom) RotateSystemKey(publicKey, privateKey string) (int, error) { return 0, nil }
func (f *fakeStoreForDecom) AddKnownHostKey(hostname, key string) error                { return nil }
func (f *fakeStoreForDecom) ExportDataForBackup() (*model.BackupData, error)           { return nil, nil }
func (f *fakeStoreForDecom) ImportDataFromBackup(*model.BackupData) error              { return nil }
func (f *fakeStoreForDecom) IntegrateDataFromBackup(*model.BackupData) error           { return nil }

type fakeDMForFacades struct {
	single ResAndErr
	bulk   []DecommissionResult
	bErr   error
}

// helper type to store return pair
type ResAndErr struct {
	R DecommissionResult
	E error
}

func (f *fakeDMForFacades) DecommissionAccount(account model.Account, systemPrivateKey security.Secret, options interface{}) (DecommissionResult, error) {
	return f.single.R, f.single.E
}
func (f *fakeDMForFacades) BulkDecommissionAccounts(targets []model.Account, systemPrivateKey security.Secret, options interface{}) ([]DecommissionResult, error) {
	return f.bulk, f.bErr
}

// remaining methods satisfy interface but are unused
func (f *fakeDMForFacades) DeployForAccount(model.Account, bool) error        { return nil }
func (f *fakeDMForFacades) FetchAuthorizedKeys(model.Account) ([]byte, error) { return nil, nil }
func (f *fakeDMForFacades) AuditSerial(model.Account) error                   { return nil }
func (f *fakeDMForFacades) AuditStrict(model.Account) error                   { return nil }
func (f *fakeDMForFacades) GetRemoteHostKey(string) (string, error)           { return "", nil }
func (f *fakeDMForFacades) CanonicalizeHostPort(host string) string           { return host }
func (f *fakeDMForFacades) ParseHostPort(host string) (string, string, error) { return host, "", nil }
func (f *fakeDMForFacades) ImportRemoteKeys(account model.Account) ([]model.PublicKey, int, string, error) {
	return nil, 0, "", nil
}
func (f *fakeDMForFacades) IsPassphraseRequired(err error) bool { return false }

func TestDecommissionAccounts_Single_Skipped(t *testing.T) {
	st := &fakeStoreForDecom{sys: &model.SystemKey{Serial: 1, PublicKey: "k"}}
	dm := &fakeDMForFacades{single: ResAndErr{R: DecommissionResult{Skipped: true}, E: nil}}
	summary, err := DecommissionAccounts(context.TODO(), []model.Account{{ID: 1}}, nil, dm, st, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.Skipped != 1 {
		t.Fatalf("expected skipped=1, got %+v", summary)
	}
}

func TestDecommissionAccounts_Single_Success(t *testing.T) {
	st := &fakeStoreForDecom{sys: &model.SystemKey{Serial: 1, PublicKey: "k"}}
	dm := &fakeDMForFacades{single: ResAndErr{R: DecommissionResult{DatabaseDeleteDone: true}, E: nil}}
	summary, err := DecommissionAccounts(context.TODO(), []model.Account{{ID: 2}}, nil, dm, st, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.Successful != 1 {
		t.Fatalf("expected successful=1, got %+v", summary)
	}
}

func TestDecommissionAccounts_Bulk_MixedResults(t *testing.T) {
	st := &fakeStoreForDecom{sys: &model.SystemKey{Serial: 1, PublicKey: "k"}}
	dm := &fakeDMForFacades{bulk: []DecommissionResult{
		{AccountID: 1, Skipped: true},
		{AccountID: 2, DatabaseDeleteError: errors.New("dbfail")},
		{AccountID: 3, DatabaseDeleteDone: true},
	}}
	targets := []model.Account{{ID: 1}, {ID: 2}, {ID: 3}}
	summary, err := DecommissionAccounts(context.TODO(), targets, nil, dm, st, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.Skipped != 1 || summary.Failed != 1 || summary.Successful != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}
