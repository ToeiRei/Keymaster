package db

import (
	"fmt"
	"strings"

	"github.com/toeirei/keymaster/internal/model"
)

var (
	// store is the active database connection, wrapped in our interface.
	// It is initialized by InitDB.
	store Store
)

// InitDB initializes the database connection based on the provided type and DSN.
// It creates tables if they don't exist.
func InitDB(dbType, dsn string) error {
	var err error

	switch strings.ToLower(dbType) {
	case "sqlite":
		store, err = NewSqliteStore(dsn)
	case "postgres":
		// The pgx driver is imported in postgres.go
		store, err = NewPostgresStore(dsn)
	default:
		return fmt.Errorf("unsupported database type: '%s'", dbType)
	}

	if err != nil {
		return fmt.Errorf("failed to initialize %s store: %w", dbType, err)
	}
	return nil
}

// GetAllAccounts retrieves all accounts from the database.
func GetAllAccounts() ([]model.Account, error) {
	return store.GetAllAccounts()
}

// AddAccount adds a new account to the database.
func AddAccount(username, hostname, label string) error {
	return store.AddAccount(username, hostname, label)
}

// DeleteAccount removes an account from the database by its ID.
func DeleteAccount(id int) error {
	return store.DeleteAccount(id)
}

// UpdateAccountSerial sets the serial for a given account ID to a specific value.
func UpdateAccountSerial(id, serial int) error {
	return store.UpdateAccountSerial(id, serial)
}

// ToggleAccountStatus flips the active status of an account.
func ToggleAccountStatus(id int) error {
	return store.ToggleAccountStatus(id)
}

// UpdateAccountLabel updates the label for a given account.
func UpdateAccountLabel(id int, label string) error {
	return store.UpdateAccountLabel(id, label)
}

// GetAllActiveAccounts retrieves all active accounts from the database.
func GetAllActiveAccounts() ([]model.Account, error) {
	return store.GetAllActiveAccounts()
}

// AddPublicKey adds a new public key to the database.
func AddPublicKey(algorithm, keyData, comment string) error {
	return store.AddPublicKey(algorithm, keyData, comment)
}

// GetAllPublicKeys retrieves all public keys from the database.
func GetAllPublicKeys() ([]model.PublicKey, error) {
	return store.GetAllPublicKeys()
}

// GetPublicKeyByComment retrieves a single public key by its unique comment.
func GetPublicKeyByComment(comment string) (*model.PublicKey, error) {
	return store.GetPublicKeyByComment(comment)
}

// AddPublicKeyAndGetModel adds a public key to the database if it doesn't already
// exist (based on the comment) and returns the full key model.
// It returns (nil, nil) if the key is a duplicate.
func AddPublicKeyAndGetModel(algorithm, keyData, comment string) (*model.PublicKey, error) {
	return store.AddPublicKeyAndGetModel(algorithm, keyData, comment)
}

// GetKnownHostKey retrieves the trusted public key for a given hostname.
func GetKnownHostKey(hostname string) (string, error) {
	return store.GetKnownHostKey(hostname)
}

// AddKnownHostKey adds a new trusted host key to the database.
func AddKnownHostKey(hostname, key string) error {
	return store.AddKnownHostKey(hostname, key)
}

// CreateSystemKey adds a new system key to the database. It determines the correct serial automatically.
func CreateSystemKey(publicKey, privateKey string) (int, error) {
	return store.CreateSystemKey(publicKey, privateKey)
}

// RotateSystemKey deactivates all current system keys and adds a new one as active.
// This should be performed within a transaction to ensure atomicity.
func RotateSystemKey(publicKey, privateKey string) (int, error) {
	return store.RotateSystemKey(publicKey, privateKey)
}

// GetActiveSystemKey retrieves the currently active system key for deployments.
func GetActiveSystemKey() (*model.SystemKey, error) {
	return store.GetActiveSystemKey()
}

// GetSystemKeyBySerial retrieves a system key by its serial number.
func GetSystemKeyBySerial(serial int) (*model.SystemKey, error) {
	return store.GetSystemKeyBySerial(serial)
}

// HasSystemKeys checks if any system keys exist in the database.
func HasSystemKeys() (bool, error) {
	return store.HasSystemKeys()
}

// DeletePublicKey removes a public key and all its associations.
// The ON DELETE CASCADE constraint handles the associations in account_keys.
func DeletePublicKey(id int) error {
	return store.DeletePublicKey(id)
}

// AssignKeyToAccount creates an association between a key and an account.
func AssignKeyToAccount(keyID, accountID int) error {
	return store.AssignKeyToAccount(keyID, accountID)
}

// UnassignKeyFromAccount removes an association between a key and an account.
func UnassignKeyFromAccount(keyID, accountID int) error {
	return store.UnassignKeyFromAccount(keyID, accountID)
}

// GetKeysForAccount retrieves all public keys assigned to a specific account.
func GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	return store.GetKeysForAccount(accountID)
}

// GetAccountsForKey retrieves all accounts that have a specific public key assigned.
func GetAccountsForKey(keyID int) ([]model.Account, error) {
	return store.GetAccountsForKey(keyID)
}

// GetAllAuditLogEntries retrieves all entries from the audit log, most recent first.
func GetAllAuditLogEntries() ([]model.AuditLogEntry, error) {
	return store.GetAllAuditLogEntries()
}

// LogAction records an audit trail event.
func LogAction(action string, details string) error {
	return store.LogAction(action, details)
}
