package db

import (
	"context"
	"database/sql"
	"fmt"
	"os/user"
	"strings"

	"github.com/toeirei/keymaster/internal/model"
	"github.com/uptrace/bun"
)

// SystemKeyModel is a local mapping used by Bun for queries.
type SystemKeyModel struct {
	bun.BaseModel `bun:"table:system_keys"`
	ID            int    `bun:"id,pk,autoincrement"`
	Serial        int    `bun:"serial"`
	PublicKey     string `bun:"public_key"`
	PrivateKey    string `bun:"private_key"`
	IsActive      bool   `bun:"is_active"`
}

// GetActiveSystemKeyBun returns the active system key using Bun for SQLite.
// This is a small, focused adapter used incrementally by the sqlite store.
func GetActiveSystemKeyBun(bdb *bun.DB) (*model.SystemKey, error) {
	ctx := context.Background()

	var sk SystemKeyModel
	err := bdb.NewSelect().Model(&sk).Where("is_active = ?", 1).Limit(1).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	m := systemKeyModelToModel(sk)
	return &m, nil
}

// RotateSystemKeyBun deactivates existing keys and inserts a new active key
// within a single transaction using Bun on SQLite.
func RotateSystemKeyBun(bdb *bun.DB, publicKey, privateKey string) (int, error) {
	ctx := context.Background()

	tx, err := bdb.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Deactivate existing keys. Use a raw UPDATE because Bun requires a WHERE
	// clause for Update/Delete queries to prevent accidental full-table updates.
	if _, err := tx.NewRaw("UPDATE system_keys SET is_active = FALSE").Exec(ctx); err != nil {
		return 0, fmt.Errorf("failed to deactivate old system keys: %w", err)
	}

	// Get current max serial
	var max sql.NullInt64
	if err := tx.NewRaw("SELECT MAX(serial) FROM system_keys").Scan(ctx, &max); err != nil {
		return 0, err
	}
	newSerial := 1
	if max.Valid {
		newSerial = int(max.Int64) + 1
	}

	// Insert new key
	res, err := tx.NewInsert().Model(&SystemKeyModel{
		Serial:     newSerial,
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		IsActive:   true,
	}).Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to insert new system key: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	_ = res // result not used for now
	return newSerial, nil
}

// AccountModel maps the `accounts` table for Bun queries.
type AccountModel struct {
	bun.BaseModel `bun:"table:accounts"`
	ID            int            `bun:"id,pk,autoincrement"`
	Username      string         `bun:"username"`
	Hostname      string         `bun:"hostname"`
	Label         sql.NullString `bun:"label"`
	Tags          sql.NullString `bun:"tags"`
	Serial        int            `bun:"serial"`
	IsActive      bool           `bun:"is_active"`
}

// PublicKeyModel maps the subset of public_keys used in joins.
type PublicKeyModel struct {
	bun.BaseModel `bun:"table:public_keys"`
	ID            int    `bun:"id,pk,autoincrement"`
	Algorithm     string `bun:"algorithm"`
	KeyData       string `bun:"key_data"`
	Comment       string `bun:"comment"`
}

// AuditLogModel maps the audit_log table.
type AuditLogModel struct {
	bun.BaseModel `bun:"table:audit_log"`
	ID            int    `bun:"id,pk,autoincrement"`
	Timestamp     string `bun:"timestamp"`
	Username      string `bun:"username"`
	Action        string `bun:"action"`
	Details       string `bun:"details"`
}

// KnownHostModel maps known_hosts.
type KnownHostModel struct {
	bun.BaseModel `bun:"table:known_hosts"`
	Hostname      string `bun:"hostname,pk"`
	Key           string `bun:"key"`
}

// BootstrapSessionModel maps bootstrap_sessions for export/import.
type BootstrapSessionModel struct {
	bun.BaseModel `bun:"table:bootstrap_sessions"`
	ID            string `bun:"id,pk"`
	Username      string `bun:"username"`
	Hostname      string `bun:"hostname"`
	Label         string `bun:"label"`
	Tags          string `bun:"tags"`
	TempPublicKey string `bun:"temp_public_key"`
	CreatedAt     string `bun:"created_at"`
	ExpiresAt     string `bun:"expires_at"`
	Status        string `bun:"status"`
}

// --- Mapping helpers (centralized conversions) ---
func accountModelToModel(a AccountModel) model.Account {
	acc := model.Account{
		ID:       a.ID,
		Username: a.Username,
		Hostname: a.Hostname,
		Serial:   a.Serial,
		IsActive: a.IsActive,
	}
	if a.Label.Valid {
		acc.Label = a.Label.String
	}
	if a.Tags.Valid {
		acc.Tags = a.Tags.String
	}
	return acc
}

func publicKeyModelToModel(p PublicKeyModel) model.PublicKey {
	return model.PublicKey{ID: p.ID, Algorithm: p.Algorithm, KeyData: p.KeyData, Comment: p.Comment}
}

func systemKeyModelToModel(skm SystemKeyModel) model.SystemKey {
	return model.SystemKey{ID: skm.ID, Serial: skm.Serial, PublicKey: skm.PublicKey, PrivateKey: skm.PrivateKey, IsActive: skm.IsActive}
}

// GetAllAccountsBun returns all accounts ordered by label, hostname, username.
func GetAllAccountsBun(bdb *bun.DB) ([]model.Account, error) {
	ctx := context.Background()
	var am []AccountModel
	err := bdb.NewSelect().Model(&am).OrderExpr("label, hostname, username").Scan(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]model.Account, 0, len(am))
	for _, a := range am {
		out = append(out, accountModelToModel(a))
	}
	return out, nil
}

// GetAllActiveAccountsBun returns all active accounts.
func GetAllActiveAccountsBun(bdb *bun.DB) ([]model.Account, error) {
	ctx := context.Background()
	var am []AccountModel
	err := bdb.NewSelect().Model(&am).Where("is_active = ?", 1).OrderExpr("label, hostname, username").Scan(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]model.Account, 0, len(am))
	for _, a := range am {
		out = append(out, accountModelToModel(a))
	}
	return out, nil
}

// AddAccountBun inserts a new account and returns its ID.
func AddAccountBun(bdb *bun.DB, username, hostname, label, tags string) (int, error) {
	ctx := context.Background()
	// Use raw INSERT to allow DB defaults (serial, is_active) to apply.
	res, err := bdb.NewRaw("INSERT INTO accounts(username, hostname, label, tags) VALUES(?, ?, ?, ?)", username, hostname, label, tags).Exec(ctx)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

// DeleteAccountBun removes an account by id.
func DeleteAccountBun(bdb *bun.DB, id int) error {
	ctx := context.Background()
	_, err := bdb.NewDelete().Model((*AccountModel)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

// AssignKeyToAccountBun creates an association in account_keys.
func AssignKeyToAccountBun(bdb *bun.DB, keyID, accountID int) error {
	ctx := context.Background()
	// Use raw insert since account_keys likely has no PK model in codebase.
	_, err := bdb.NewRaw("INSERT INTO account_keys(key_id, account_id) VALUES(?, ?)", keyID, accountID).Exec(ctx)
	return err
}

// UnassignKeyFromAccountBun removes an association from account_keys.
func UnassignKeyFromAccountBun(bdb *bun.DB, keyID, accountID int) error {
	ctx := context.Background()
	_, err := bdb.NewRaw("DELETE FROM account_keys WHERE key_id = ? AND account_id = ?", keyID, accountID).Exec(ctx)
	return err
}

// GetKeysForAccountBun returns public keys for a given account.
func GetKeysForAccountBun(bdb *bun.DB, accountID int) ([]model.PublicKey, error) {
	ctx := context.Background()
	var pks []PublicKeyModel
	err := bdb.NewSelect().Model(&pks).
		TableExpr("public_keys AS pk").
		Join("JOIN account_keys ak ON pk.id = ak.key_id").
		Where("ak.account_id = ?", accountID).
		OrderExpr("pk.comment").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]model.PublicKey, 0, len(pks))
	for _, p := range pks {
		out = append(out, publicKeyModelToModel(p))
	}
	return out, nil
}

// GetAccountsForKeyBun returns accounts that have a given key assigned.
func GetAccountsForKeyBun(bdb *bun.DB, keyID int) ([]model.Account, error) {
	ctx := context.Background()
	var am []AccountModel
	err := bdb.NewSelect().Model(&am).
		TableExpr("accounts AS a").
		Join("JOIN account_keys ak ON a.id = ak.account_id").
		Where("ak.key_id = ?", keyID).
		OrderExpr("a.label, a.hostname, a.username").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]model.Account, 0, len(am))
	for _, a := range am {
		out = append(out, accountModelToModel(a))
	}
	return out, nil
}

// GetAllAuditLogEntriesBun retrieves audit log entries ordered by timestamp desc.
func GetAllAuditLogEntriesBun(bdb *bun.DB) ([]model.AuditLogEntry, error) {
	ctx := context.Background()
	var am []AuditLogModel
	if err := bdb.NewSelect().Model(&am).OrderExpr("timestamp DESC").Scan(ctx); err != nil {
		return nil, err
	}
	out := make([]model.AuditLogEntry, 0, len(am))
	for _, a := range am {
		out = append(out, model.AuditLogEntry{ID: a.ID, Timestamp: a.Timestamp, Username: a.Username, Action: a.Action, Details: a.Details})
	}
	return out, nil
}

// LogActionBun inserts an audit log entry with the current OS user.
func LogActionBun(bdb *bun.DB, action string, details string) error {
	ctx := context.Background()
	curUser, err := user.Current()
	username := "unknown"
	if err == nil {
		if parts := strings.Split(curUser.Username, `\`); len(parts) > 1 {
			username = parts[1]
		} else {
			username = curUser.Username
		}
	}
	_, err = bdb.NewRaw("INSERT INTO audit_log (username, action, details) VALUES (?, ?, ?)", username, action, details).Exec(ctx)
	return err
}

// ExportDataForBackupBun exports all tables' data into a model.BackupData using a Bun transaction.
func ExportDataForBackupBun(bdb *bun.DB) (*model.BackupData, error) {
	ctx := context.Background()
	tx, err := bdb.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	backup := &model.BackupData{SchemaVersion: 1}

	// Accounts
	var accounts []AccountModel
	if err := tx.NewSelect().Model(&accounts).Scan(ctx); err != nil {
		return nil, err
	}
	for _, a := range accounts {
		backup.Accounts = append(backup.Accounts, accountModelToModel(a))
	}

	// Public keys
	var pks []PublicKeyModel
	if err := tx.NewSelect().Model(&pks).Scan(ctx); err != nil {
		return nil, err
	}
	for _, p := range pks {
		backup.PublicKeys = append(backup.PublicKeys, publicKeyModelToModel(p))
	}

	// Account keys
	type akRow struct{ KeyID, AccountID int }
	var aks []akRow
	if err := tx.NewRaw("SELECT key_id, account_id FROM account_keys").Scan(ctx, &aks); err != nil {
		return nil, err
	}
	for _, r := range aks {
		backup.AccountKeys = append(backup.AccountKeys, model.AccountKey{KeyID: r.KeyID, AccountID: r.AccountID})
	}

	// System keys
	var sks []SystemKeyModel
	if err := tx.NewSelect().Model(&sks).Scan(ctx); err != nil {
		return nil, err
	}
	for _, s := range sks {
		backup.SystemKeys = append(backup.SystemKeys, systemKeyModelToModel(s))
	}

	// Known hosts
	var khs []KnownHostModel
	if err := tx.NewSelect().Model(&khs).Scan(ctx); err != nil {
		return nil, err
	}
	for _, k := range khs {
		backup.KnownHosts = append(backup.KnownHosts, model.KnownHost{Hostname: k.Hostname, Key: k.Key})
	}

	// Audit log
	var als []AuditLogModel
	if err := tx.NewSelect().Model(&als).Scan(ctx); err != nil {
		return nil, err
	}
	for _, a := range als {
		backup.AuditLogEntries = append(backup.AuditLogEntries, model.AuditLogEntry{ID: a.ID, Timestamp: a.Timestamp, Username: a.Username, Action: a.Action, Details: a.Details})
	}

	// Bootstrap sessions
	var bss []BootstrapSessionModel
	if err := tx.NewSelect().Model(&bss).Scan(ctx); err != nil {
		return nil, err
	}
	for _, b := range bss {
		// Note: CreatedAt/ExpiresAt are strings; the model expects time.Time. We leave parsing to caller if needed.
		bs := model.BootstrapSession{ID: b.ID, Username: b.Username, Hostname: b.Hostname, Label: b.Label, Tags: b.Tags, TempPublicKey: b.TempPublicKey, Status: b.Status}
		backup.BootstrapSessions = append(backup.BootstrapSessions, bs)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return backup, nil
}

// ImportDataFromBackupBun performs a full wipe-and-replace using a Bun transaction.
func ImportDataFromBackupBun(bdb *bun.DB, backup *model.BackupData) error {
	ctx := context.Background()
	tx, err := bdb.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Wipe tables
	tables := []string{"account_keys", "bootstrap_sessions", "audit_log", "known_hosts", "system_keys", "public_keys", "accounts"}
	for _, t := range tables {
		if _, err := tx.NewRaw(fmt.Sprintf("DELETE FROM %s", t)).Exec(ctx); err != nil {
			return err
		}
	}

	// Insert accounts
	for _, acc := range backup.Accounts {
		if _, err := tx.NewRaw("INSERT INTO accounts (id, username, hostname, label, tags, serial, is_active) VALUES (?, ?, ?, ?, ?, ?, ?)", acc.ID, acc.Username, acc.Hostname, acc.Label, acc.Tags, acc.Serial, acc.IsActive).Exec(ctx); err != nil {
			return err
		}
	}
	// Public keys
	for _, pk := range backup.PublicKeys {
		if _, err := tx.NewRaw("INSERT INTO public_keys (id, algorithm, key_data, comment, is_global) VALUES (?, ?, ?, ?, ?)", pk.ID, pk.Algorithm, pk.KeyData, pk.Comment, pk.IsGlobal).Exec(ctx); err != nil {
			return err
		}
	}
	// AccountKeys
	for _, ak := range backup.AccountKeys {
		if _, err := tx.NewRaw("INSERT INTO account_keys (key_id, account_id) VALUES (?, ?)", ak.KeyID, ak.AccountID).Exec(ctx); err != nil {
			return err
		}
	}
	// SystemKeys
	for _, sk := range backup.SystemKeys {
		if _, err := tx.NewRaw("INSERT INTO system_keys (id, serial, public_key, private_key, is_active) VALUES (?, ?, ?, ?, ?)", sk.ID, sk.Serial, sk.PublicKey, sk.PrivateKey, sk.IsActive).Exec(ctx); err != nil {
			return err
		}
	}
	// KnownHosts
	for _, kh := range backup.KnownHosts {
		if _, err := tx.NewRaw("INSERT INTO known_hosts (hostname, key) VALUES (?, ?)", kh.Hostname, kh.Key).Exec(ctx); err != nil {
			return err
		}
	}
	// AuditLog
	for _, ale := range backup.AuditLogEntries {
		if _, err := tx.NewRaw("INSERT INTO audit_log (id, timestamp, username, action, details) VALUES (?, ?, ?, ?, ?)", ale.ID, ale.Timestamp, ale.Username, ale.Action, ale.Details).Exec(ctx); err != nil {
			return err
		}
	}
	// Bootstrap sessions: skipping CreatedAt/ExpiresAt parsing; insert core fields
	for _, bs := range backup.BootstrapSessions {
		if _, err := tx.NewRaw("INSERT INTO bootstrap_sessions (id, username, hostname, label, tags, temp_public_key, status) VALUES (?, ?, ?, ?, ?, ?, ?)", bs.ID, bs.Username, bs.Hostname, bs.Label, bs.Tags, bs.TempPublicKey, bs.Status).Exec(ctx); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// IntegrateDataFromBackupBun performs a non-destructive restore using INSERT OR IGNORE semantics.
func IntegrateDataFromBackupBun(bdb *bun.DB, backup *model.BackupData) error {
	ctx := context.Background()
	tx, err := bdb.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, acc := range backup.Accounts {
		if _, err := tx.NewRaw("INSERT OR IGNORE INTO accounts (id, username, hostname, label, tags, serial, is_active) VALUES (?, ?, ?, ?, ?, ?, ?)", acc.ID, acc.Username, acc.Hostname, acc.Label, acc.Tags, acc.Serial, acc.IsActive).Exec(ctx); err != nil {
			return err
		}
	}
	for _, pk := range backup.PublicKeys {
		if _, err := tx.NewRaw("INSERT OR IGNORE INTO public_keys (id, algorithm, key_data, comment, is_global) VALUES (?, ?, ?, ?, ?)", pk.ID, pk.Algorithm, pk.KeyData, pk.Comment, pk.IsGlobal).Exec(ctx); err != nil {
			return err
		}
	}
	for _, ak := range backup.AccountKeys {
		if _, err := tx.NewRaw("INSERT OR IGNORE INTO account_keys (key_id, account_id) VALUES (?, ?)", ak.KeyID, ak.AccountID).Exec(ctx); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// --- Public key helpers ---
// GetAllPublicKeysBun retrieves all public keys ordered by comment.
func GetAllPublicKeysBun(bdb *bun.DB) ([]model.PublicKey, error) {
	ctx := context.Background()
	var pks []PublicKeyModel
	if err := bdb.NewSelect().Model(&pks).OrderExpr("comment").Scan(ctx); err != nil {
		return nil, err
	}
	out := make([]model.PublicKey, 0, len(pks))
	for _, p := range pks {
		out = append(out, publicKeyModelToModel(p))
	}
	return out, nil
}

// GetPublicKeyByCommentBun retrieves a public key by comment.
func GetPublicKeyByCommentBun(bdb *bun.DB, comment string) (*model.PublicKey, error) {
	ctx := context.Background()
	var pk PublicKeyModel
	err := bdb.NewSelect().Model(&pk).Where("comment = ?", comment).Limit(1).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	m := publicKeyModelToModel(pk)
	return &m, nil
}

// AddPublicKeyBun inserts a public key.
func AddPublicKeyBun(bdb *bun.DB, algorithm, keyData, comment string, isGlobal bool) error {
	ctx := context.Background()
	_, err := bdb.NewRaw("INSERT INTO public_keys(algorithm, key_data, comment, is_global) VALUES(?, ?, ?, ?)", algorithm, keyData, comment, isGlobal).Exec(ctx)
	return err
}

// AddPublicKeyAndGetModelBun inserts a public key if not exists and returns the model.
// Returns (nil, nil) when duplicate.
func AddPublicKeyAndGetModelBun(bdb *bun.DB, algorithm, keyData, comment string, isGlobal bool) (*model.PublicKey, error) {
	// Check for existing
	existing, err := GetPublicKeyByCommentBun(bdb, comment)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, nil
	}
	ctx := context.Background()
	res, err := bdb.NewRaw("INSERT INTO public_keys (algorithm, key_data, comment, is_global) VALUES (?, ?, ?, ?)", algorithm, keyData, comment, isGlobal).Exec(ctx)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &model.PublicKey{ID: int(id), Algorithm: algorithm, KeyData: keyData, Comment: comment, IsGlobal: isGlobal}, nil
}

// TogglePublicKeyGlobalBun flips is_global for a key by id.
func TogglePublicKeyGlobalBun(bdb *bun.DB, id int) error {
	ctx := context.Background()
	_, err := bdb.NewRaw("UPDATE public_keys SET is_global = NOT is_global WHERE id = ?", id).Exec(ctx)
	return err
}

// GetGlobalPublicKeysBun returns public keys where is_global = 1.
func GetGlobalPublicKeysBun(bdb *bun.DB) ([]model.PublicKey, error) {
	ctx := context.Background()
	var pks []PublicKeyModel
	if err := bdb.NewSelect().Model(&pks).Where("is_global = ?", 1).OrderExpr("comment").Scan(ctx); err != nil {
		return nil, err
	}
	out := make([]model.PublicKey, 0, len(pks))
	for _, p := range pks {
		out = append(out, publicKeyModelToModel(p))
	}
	return out, nil
}

// DeletePublicKeyBun deletes a public key by id.
func DeletePublicKeyBun(bdb *bun.DB, id int) error {
	ctx := context.Background()
	_, err := bdb.NewRaw("DELETE FROM public_keys WHERE id = ?", id).Exec(ctx)
	return err
}
