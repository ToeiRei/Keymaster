package db

import "github.com/toeirei/keymaster/internal/model"

// Store defines the interface for all database operations in Keymaster.
// This allows for multiple database backends to be implemented.
type Store interface {
	// Account methods
	GetAllAccounts() ([]model.Account, error)
	AddAccount(username, hostname, label, tags string) error
	DeleteAccount(id int) error
	UpdateAccountSerial(id, serial int) error
	ToggleAccountStatus(id int) error
	UpdateAccountLabel(id int, label string) error
	UpdateAccountTags(id int, tags string) error
	GetAllActiveAccounts() ([]model.Account, error)

	// Public Key methods
	AddPublicKey(algorithm, keyData, comment string, isGlobal bool) error
	GetAllPublicKeys() ([]model.PublicKey, error)
	GetPublicKeyByComment(comment string) (*model.PublicKey, error)
	AddPublicKeyAndGetModel(algorithm, keyData, comment string, isGlobal bool) (*model.PublicKey, error)
	TogglePublicKeyGlobal(id int) error
	GetGlobalPublicKeys() ([]model.PublicKey, error)
	DeletePublicKey(id int) error

	// Host Key methods
	GetKnownHostKey(hostname string) (string, error)
	AddKnownHostKey(hostname, key string) error

	// System Key methods
	CreateSystemKey(publicKey, privateKey string) (int, error)
	RotateSystemKey(publicKey, privateKey string) (int, error)
	GetActiveSystemKey() (*model.SystemKey, error)
	GetSystemKeyBySerial(serial int) (*model.SystemKey, error)
	HasSystemKeys() (bool, error)

	// Assignment methods
	AssignKeyToAccount(keyID, accountID int) error
	UnassignKeyFromAccount(keyID, accountID int) error
	GetKeysForAccount(accountID int) ([]model.PublicKey, error)
	GetAccountsForKey(keyID int) ([]model.Account, error)

	// Audit Log methods
	GetAllAuditLogEntries() ([]model.AuditLogEntry, error)
	LogAction(action string, details string) error
}
