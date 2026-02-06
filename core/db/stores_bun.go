// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"fmt"
	"time"

	"github.com/toeirei/keymaster/internal/core/model"
	"github.com/uptrace/bun"
)

// BunStore is the consolidated bun-backed Store implementation used for all
// supported database engines. It delegates operations to centralized Bun
// helpers in this package.
type BunStore struct {
	bun *bun.DB
}

// BunDB returns the underlying *bun.DB for advanced callers.
func (s *BunStore) BunDB() *bun.DB { return s.bun }

func (s *BunStore) GetAllAccounts() ([]model.Account, error) { return GetAllAccountsBun(s.bun) }
func (s *BunStore) GetAccounts() ([]model.Account, error) {
	return s.GetAllAccounts()
}
func (s *BunStore) GetAccount(id int) (*model.Account, error) {
	return GetAccountByIDBun(s.bun, id)
}
func (s *BunStore) AddAccount(username, hostname, label, tags string) (int, error) {
	id, err := AddAccountBun(s.bun, username, hostname, label, tags)
	if err == nil {
		_ = s.LogAction("ADD_ACCOUNT", fmt.Sprintf("account: %s@%s", username, hostname))
	}
	return id, err
}
func (s *BunStore) DeleteAccount(id int) error {
	details := fmt.Sprintf("id: %d", id)
	if acc, err2 := GetAccountByIDBun(s.bun, id); err2 == nil && acc != nil {
		details = fmt.Sprintf("account: %s@%s", acc.Username, acc.Hostname)
	}
	err := DeleteAccountBun(s.bun, id)
	if err == nil {
		_ = s.LogAction("DELETE_ACCOUNT", details)
	}
	return err
}
func (s *BunStore) AssignKeyToAccount(keyID, accountID int) error {
	err := AssignKeyToAccountBun(s.bun, keyID, accountID)
	if err == nil {
		_ = s.LogAction("ASSIGN_KEY", fmt.Sprintf("keyID: %d, accountID: %d", keyID, accountID))
	}
	return err
}
func (s *BunStore) UpdateAccountSerial(id, serial int) error {
	return UpdateAccountSerialBun(s.bun, id, serial)
}
func (s *BunStore) ToggleAccountStatus(id int) error {
	acc, err := GetAccountByIDBun(s.bun, id)
	if err != nil {
		return err
	}
	if acc == nil {
		return fmt.Errorf("account not found: %d", id)
	}
	newStatus, err := ToggleAccountStatusBun(s.bun, id)
	if err == nil {
		_ = s.LogAction("TOGGLE_ACCOUNT_STATUS", fmt.Sprintf("account: %s@%s, new_status: %t", acc.Username, acc.Hostname, newStatus))
	}
	return err
}
func (s *BunStore) UpdateAccountLabel(id int, label string) error {
	err := UpdateAccountLabelBun(s.bun, id, label)
	if err == nil {
		_ = s.LogAction("UPDATE_ACCOUNT_LABEL", fmt.Sprintf("account_id: %d, new_label: '%s'", id, label))
	}
	return err
}
func (s *BunStore) UpdateAccountHostname(id int, hostname string) error {
	return UpdateAccountHostnameBun(s.bun, id, hostname)
}
func (s *BunStore) UpdateAccountTags(id int, tags string) error {
	err := UpdateAccountTagsBun(s.bun, id, tags)
	if err == nil {
		_ = s.LogAction("UPDATE_ACCOUNT_TAGS", fmt.Sprintf("account_id: %d, new_tags: '%s'", id, tags))
	}
	return err
}
func (s *BunStore) UpdateAccountIsDirty(id int, dirty bool) error {
	return UpdateAccountIsDirtyBun(s.bun, id, dirty)
}
func (s *BunStore) GetAllActiveAccounts() ([]model.Account, error) {
	return GetAllActiveAccountsBun(s.bun)
}
func (s *BunStore) GetKnownHostKey(hostname string) (string, error) {
	return GetKnownHostKeyBun(s.bun, hostname)
}
func (s *BunStore) AddKnownHostKey(hostname, key string) error {
	err := AddKnownHostKeyBun(s.bun, hostname, key)
	if err == nil {
		_ = s.LogAction("TRUST_HOST", fmt.Sprintf("hostname: %s", hostname))
	}
	return err
}
func (s *BunStore) CreateSystemKey(publicKey, privateKey string) (int, error) {
	newSerial, err := CreateSystemKeyBun(s.bun, publicKey, privateKey)
	if err == nil {
		_ = s.LogAction("CREATE_SYSTEM_KEY", fmt.Sprintf("serial: %d", newSerial))
	}
	return newSerial, err
}
func (s *BunStore) RotateSystemKey(publicKey, privateKey string) (int, error) {
	newSerial, err := RotateSystemKeyBun(s.bun, publicKey, privateKey)
	if err == nil {
		_ = s.LogAction("ROTATE_SYSTEM_KEY", fmt.Sprintf("new_serial: %d", newSerial))
	}
	return newSerial, err
}
func (s *BunStore) GetActiveSystemKey() (*model.SystemKey, error) {
	return GetActiveSystemKeyBun(s.bun)
}
func (s *BunStore) GetSystemKeyBySerial(serial int) (*model.SystemKey, error) {
	return GetSystemKeyBySerialBun(s.bun, serial)
}
func (s *BunStore) HasSystemKeys() (bool, error) { return HasSystemKeysBun(s.bun) }
func (s *BunStore) SearchAccounts(query string) ([]model.Account, error) {
	return NewBunAccountSearcher(s.bun).SearchAccounts(query)
}
func (s *BunStore) GetAllAuditLogEntries() ([]model.AuditLogEntry, error) {
	return GetAllAuditLogEntriesBun(s.bun)
}
func (s *BunStore) LogAction(action string, details string) error {
	return LogActionBun(s.bun, action, details)
}
func (s *BunStore) SaveBootstrapSession(id, username, hostname, label, tags, tempPublicKey string, expiresAt time.Time, status string) error {
	return SaveBootstrapSessionBun(s.bun, id, username, hostname, label, tags, tempPublicKey, expiresAt, status)
}
func (s *BunStore) GetBootstrapSession(id string) (*model.BootstrapSession, error) {
	return GetBootstrapSessionBun(s.bun, id)
}
func (s *BunStore) DeleteBootstrapSession(id string) error {
	return DeleteBootstrapSessionBun(s.bun, id)
}
func (s *BunStore) UpdateBootstrapSessionStatus(id string, status string) error {
	return UpdateBootstrapSessionStatusBun(s.bun, id, status)
}
func (s *BunStore) GetExpiredBootstrapSessions() ([]*model.BootstrapSession, error) {
	return GetExpiredBootstrapSessionsBun(s.bun)
}
func (s *BunStore) GetOrphanedBootstrapSessions() ([]*model.BootstrapSession, error) {
	return GetOrphanedBootstrapSessionsBun(s.bun)
}
func (s *BunStore) ExportDataForBackup() (*model.BackupData, error) {
	return ExportDataForBackupBun(s.bun)
}
func (s *BunStore) ImportDataFromBackup(backup *model.BackupData) error {
	return ImportDataFromBackupBun(s.bun, backup)
}
func (s *BunStore) IntegrateDataFromBackup(backup *model.BackupData) error {
	return IntegrateDataFromBackupBun(s.bun, backup)
}
