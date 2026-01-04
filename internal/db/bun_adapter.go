package db

import (
	"context"
	"database/sql"
	"fmt"
	"os/user"
	"strings"
	"time"

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
	defer func() { _ = tx.Rollback() }()

	// Deactivate existing keys. Use a raw UPDATE because Bun requires a WHERE
	// clause for Update/Delete queries to prevent accidental full-table updates.
	if _, err := ExecRaw(ctx, tx, "UPDATE system_keys SET is_active = FALSE"); err != nil {
		return 0, fmt.Errorf("failed to deactivate old system keys: %w", err)
	}

	// Get current max serial
	var max sql.NullInt64
	if err := QueryRawInto(ctx, tx, &max, "SELECT MAX(serial) FROM system_keys"); err != nil {
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
	ID            int          `bun:"id,pk,autoincrement"`
	Algorithm     string       `bun:"algorithm"`
	KeyData       string       `bun:"key_data"`
	Comment       string       `bun:"comment"`
	ExpiresAt     sql.NullTime `bun:"expires_at"`
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
	ID            string         `bun:"id,pk"`
	Username      string         `bun:"username"`
	Hostname      string         `bun:"hostname"`
	Label         sql.NullString `bun:"label"`
	Tags          sql.NullString `bun:"tags"`
	TempPublicKey string         `bun:"temp_public_key"`
	CreatedAt     time.Time      `bun:"created_at"`
	ExpiresAt     time.Time      `bun:"expires_at"`
	Status        string         `bun:"status"`
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

func bootstrapSessionModelToModel(bsm BootstrapSessionModel) model.BootstrapSession {
	bs := model.BootstrapSession{
		ID:            bsm.ID,
		Username:      bsm.Username,
		Hostname:      bsm.Hostname,
		TempPublicKey: bsm.TempPublicKey,
		CreatedAt:     bsm.CreatedAt,
		ExpiresAt:     bsm.ExpiresAt,
		Status:        bsm.Status,
	}
	if bsm.Label.Valid {
		bs.Label = bsm.Label.String
	}
	if bsm.Tags.Valid {
		bs.Tags = bsm.Tags.String
	}
	return bs
}

func publicKeyModelToModel(p PublicKeyModel) model.PublicKey {
	pk := model.PublicKey{ID: p.ID, Algorithm: p.Algorithm, KeyData: p.KeyData, Comment: p.Comment}
	if p.ExpiresAt.Valid {
		pk.ExpiresAt = p.ExpiresAt.Time
	}
	return pk
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
	// Use Bun's NewInsert with Returning to support Postgres and MySQL
	am := &AccountModel{
		Username: username,
		Hostname: hostname,
		Label:    sql.NullString{String: label, Valid: label != ""},
		Tags:     sql.NullString{String: tags, Valid: tags != ""},
	}
	// Try to insert and return the assigned ID in a DB-agnostic way.
	// Insert only the fields we want the DB to default (like is_active, serial).
	if _, err := bdb.NewInsert().Model(am).Column("username", "hostname", "label", "tags").Returning("id").Exec(ctx); err != nil {
		return 0, MapDBError(err)
	}
	return am.ID, nil
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
	_, err := ExecRaw(ctx, bdb, "INSERT INTO account_keys(key_id, account_id) VALUES(?, ?)", keyID, accountID)
	return MapDBError(err)
}

// UnassignKeyFromAccountBun removes an association from account_keys.
func UnassignKeyFromAccountBun(bdb *bun.DB, keyID, accountID int) error {
	ctx := context.Background()
	_, err := ExecRaw(ctx, bdb, "DELETE FROM account_keys WHERE key_id = ? AND account_id = ?", keyID, accountID)
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

// SearchAccountsBun performs a portable fuzzy search over accounts using
// simple tokenized LIKE matching across username, hostname, and label.
// This emulates more advanced Postgres full-text search in a DB-agnostic way.
func SearchAccountsBun(bdb *bun.DB, q string) ([]model.Account, error) {
	ctx := context.Background()
	tokens := TokenizeSearchQuery(q)
	var am []AccountModel
	qb := bdb.NewSelect().Model(&am)
	if len(tokens) > 0 {
		// Build WHERE clause with AND of ORs: for each token, require it matches one of the columns
		// e.g., WHERE (username LIKE '%t1%' OR hostname LIKE '%t1%' OR label LIKE '%t1%')
		for _, tok := range tokens {
			like := "%" + tok + "%"
			// Use LOWER(...) for case-insensitive matching across engines
			qb = qb.Where("(LOWER(username) LIKE ? OR LOWER(hostname) LIKE ? OR LOWER(label) LIKE ?)", like, like, like)
		}
	}
	if err := qb.OrderExpr("label, hostname, username").Scan(ctx); err != nil {
		return nil, err
	}
	out := make([]model.Account, 0, len(am))
	for _, a := range am {
		out = append(out, accountModelToModel(a))
	}
	return out, nil
}

// TokenizeSearchQuery is a pure helper that splits a query into lower-cased tokens.
// It was extracted to improve testability of search-related logic.
// Use TokenizeSearchQuery instead of the internal helper.
// (Implementation lives in internal/db/search.go)

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
	_, err = ExecRaw(ctx, bdb, "INSERT INTO audit_log (username, action, details) VALUES (?, ?, ?)", username, action, details)
	return MapDBError(err)
}

// ExportDataForBackupBun exports all tables' data into a model.BackupData using a Bun transaction.
func ExportDataForBackupBun(bdb *bun.DB) (*model.BackupData, error) {
	ctx := context.Background()
	var backup *model.BackupData
	err := WithTx(ctx, bdb, func(ctx context.Context, tx bun.Tx) error {
		backup = &model.BackupData{SchemaVersion: 1}

		// Accounts
		var accounts []AccountModel
		if err := tx.NewSelect().Model(&accounts).Scan(ctx); err != nil {
			return err
		}
		for _, a := range accounts {
			backup.Accounts = append(backup.Accounts, accountModelToModel(a))
		}

		// Public keys
		var pks []PublicKeyModel
		if err := tx.NewSelect().Model(&pks).Scan(ctx); err != nil {
			return err
		}
		for _, p := range pks {
			backup.PublicKeys = append(backup.PublicKeys, publicKeyModelToModel(p))
		}

		// Account keys
		type akRow struct{ KeyID, AccountID int }
		var aks []akRow
		if err := QueryRawInto(ctx, tx, &aks, "SELECT key_id, account_id FROM account_keys"); err != nil {
			return err
		}
		for _, r := range aks {
			backup.AccountKeys = append(backup.AccountKeys, model.AccountKey{KeyID: r.KeyID, AccountID: r.AccountID})
		}

		// System keys
		var sks []SystemKeyModel
		if err := tx.NewSelect().Model(&sks).Scan(ctx); err != nil {
			return err
		}
		for _, s := range sks {
			backup.SystemKeys = append(backup.SystemKeys, systemKeyModelToModel(s))
		}

		// Known hosts
		var khs []KnownHostModel
		if err := tx.NewSelect().Model(&khs).Scan(ctx); err != nil {
			return err
		}
		for _, k := range khs {
			backup.KnownHosts = append(backup.KnownHosts, model.KnownHost{Hostname: k.Hostname, Key: k.Key})
		}

		// Audit log
		var als []AuditLogModel
		if err := tx.NewSelect().Model(&als).Scan(ctx); err != nil {
			return err
		}
		for _, a := range als {
			backup.AuditLogEntries = append(backup.AuditLogEntries, model.AuditLogEntry{ID: a.ID, Timestamp: a.Timestamp, Username: a.Username, Action: a.Action, Details: a.Details})
		}

		// Bootstrap sessions
		var bss []BootstrapSessionModel
		if err := tx.NewSelect().Model(&bss).Scan(ctx); err != nil {
			return err
		}
		for _, b := range bss {
			bs := model.BootstrapSession{ID: b.ID, Username: b.Username, Hostname: b.Hostname, TempPublicKey: b.TempPublicKey, CreatedAt: b.CreatedAt, ExpiresAt: b.ExpiresAt, Status: b.Status}
			if b.Label.Valid {
				bs.Label = b.Label.String
			}
			if b.Tags.Valid {
				bs.Tags = b.Tags.String
			}
			backup.BootstrapSessions = append(backup.BootstrapSessions, bs)
		}

		return nil
	})
	return backup, err
}

// ImportDataFromBackupBun performs a full wipe-and-replace using a Bun transaction.
func ImportDataFromBackupBun(bdb *bun.DB, backup *model.BackupData) error {
	ctx := context.Background()
	return WithTx(ctx, bdb, func(ctx context.Context, tx bun.Tx) error {
		// Wipe tables
		tables := []string{"account_keys", "bootstrap_sessions", "audit_log", "known_hosts", "system_keys", "public_keys", "accounts"}
		for _, t := range tables {
			if _, err := ExecRaw(ctx, tx, fmt.Sprintf("DELETE FROM %s", t)); err != nil {
				return err
			}
		}

		// Insert accounts
		for _, acc := range backup.Accounts {
			if _, err := ExecRaw(ctx, tx, "INSERT INTO accounts (id, username, hostname, label, tags, serial, is_active) VALUES (?, ?, ?, ?, ?, ?, ?)", acc.ID, acc.Username, acc.Hostname, acc.Label, acc.Tags, acc.Serial, acc.IsActive); err != nil {
				return MapDBError(err)
			}
		}
		// Public keys
		for _, pk := range backup.PublicKeys {
			if _, err := ExecRaw(ctx, tx, "INSERT INTO public_keys (id, algorithm, key_data, comment, is_global) VALUES (?, ?, ?, ?, ?)", pk.ID, pk.Algorithm, pk.KeyData, pk.Comment, pk.IsGlobal); err != nil {
				return MapDBError(err)
			}
		}
		// AccountKeys
		for _, ak := range backup.AccountKeys {
			if _, err := ExecRaw(ctx, tx, "INSERT INTO account_keys (key_id, account_id) VALUES (?, ?)", ak.KeyID, ak.AccountID); err != nil {
				return MapDBError(err)
			}
		}
		// SystemKeys
		for _, sk := range backup.SystemKeys {
			if _, err := ExecRaw(ctx, tx, "INSERT INTO system_keys (id, serial, public_key, private_key, is_active) VALUES (?, ?, ?, ?, ?)", sk.ID, sk.Serial, sk.PublicKey, sk.PrivateKey, sk.IsActive); err != nil {
				return MapDBError(err)
			}
		}
		// KnownHosts
		for _, kh := range backup.KnownHosts {
			if _, err := ExecRaw(ctx, tx, "INSERT INTO known_hosts (hostname, key) VALUES (?, ?)", kh.Hostname, kh.Key); err != nil {
				return MapDBError(err)
			}
		}
		// AuditLog: convert RFC3339 timestamps to time.Time when possible so MySQL accepts them.
		for _, ale := range backup.AuditLogEntries {
			var ts interface{} = ale.Timestamp
			if ale.Timestamp != "" {
				if parsed, err := time.Parse(time.RFC3339, ale.Timestamp); err == nil {
					ts = parsed
				} else {
					// Fallback: convert 'T' separator to space and strip trailing 'Z' if present.
					s := ale.Timestamp
					s = strings.Replace(s, "T", " ", 1)
					s = strings.TrimSuffix(s, "Z")
					ts = s
				}
			}
			if _, err := ExecRaw(ctx, tx, "INSERT INTO audit_log (id, timestamp, username, action, details) VALUES (?, ?, ?, ?, ?)", ale.ID, ts, ale.Username, ale.Action, ale.Details); err != nil {
				return MapDBError(err)
			}
		}
		// Bootstrap sessions: include CreatedAt/ExpiresAt when importing
		for _, bs := range backup.BootstrapSessions {
			if _, err := ExecRaw(ctx, tx, "INSERT INTO bootstrap_sessions (id, username, hostname, label, tags, temp_public_key, created_at, expires_at, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", bs.ID, bs.Username, bs.Hostname, bs.Label, bs.Tags, bs.TempPublicKey, bs.CreatedAt, bs.ExpiresAt, bs.Status); err != nil {
				return MapDBError(err)
			}
		}
		return nil
	})
}

// IntegrateDataFromBackupBun performs a non-destructive restore using INSERT OR IGNORE semantics.
func IntegrateDataFromBackupBun(bdb *bun.DB, backup *model.BackupData) error {
	ctx := context.Background()
	return WithTx(ctx, bdb, func(ctx context.Context, tx bun.Tx) error {
		for _, acc := range backup.Accounts {
			if _, err := ExecRaw(ctx, tx, "INSERT OR IGNORE INTO accounts (id, username, hostname, label, tags, serial, is_active) VALUES (?, ?, ?, ?, ?, ?, ?)", acc.ID, acc.Username, acc.Hostname, acc.Label, acc.Tags, acc.Serial, acc.IsActive); err != nil {
				return err
			}
		}
		for _, pk := range backup.PublicKeys {
			if _, err := ExecRaw(ctx, tx, "INSERT OR IGNORE INTO public_keys (id, algorithm, key_data, comment, is_global) VALUES (?, ?, ?, ?, ?)", pk.ID, pk.Algorithm, pk.KeyData, pk.Comment, pk.IsGlobal); err != nil {
				return err
			}
		}
		for _, ak := range backup.AccountKeys {
			if _, err := ExecRaw(ctx, tx, "INSERT OR IGNORE INTO account_keys (key_id, account_id) VALUES (?, ?)", ak.KeyID, ak.AccountID); err != nil {
				return err
			}
		}
		return nil
	})
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
	_, err := ExecRaw(ctx, bdb, "INSERT INTO public_keys(algorithm, key_data, comment, is_global) VALUES(?, ?, ?, ?)", algorithm, keyData, comment, isGlobal)
	return MapDBError(err)
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
	res, err := ExecRaw(ctx, bdb, "INSERT INTO public_keys (algorithm, key_data, comment, is_global) VALUES (?, ?, ?, ?)", algorithm, keyData, comment, isGlobal)
	if err != nil {
		return nil, MapDBError(err)
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
	_, err := ExecRaw(ctx, bdb, "UPDATE public_keys SET is_global = NOT is_global WHERE id = ?", id)
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
	_, err := ExecRaw(ctx, bdb, "DELETE FROM public_keys WHERE id = ?", id)
	return err
}

// GetPublicKeyByIDBun retrieves a public key by its numeric ID.
func GetPublicKeyByIDBun(bdb *bun.DB, id int) (*model.PublicKey, error) {
	ctx := context.Background()
	var pk PublicKeyModel
	err := bdb.NewSelect().Model(&pk).Where("id = ?", id).Limit(1).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	m := publicKeyModelToModel(pk)
	return &m, nil
}

// SearchPublicKeysBun performs a tokenized, case-insensitive search against
// public keys. Tokens are ANDed together; within each token we match
// against comment, algorithm, or key_data using SQL LIKE.
func SearchPublicKeysBun(bdb *bun.DB, q string) ([]model.PublicKey, error) {
	ctx := context.Background()
	toks := TokenizeSearchQuery(q)
	var pks []PublicKeyModel
	sel := bdb.NewSelect().Model(&pks).OrderExpr("comment")
	// If no tokens, return all
	if len(toks) == 0 {
		if err := sel.Scan(ctx); err != nil {
			return nil, err
		}
	} else {
		for _, t := range toks {
			like := "%" + t + "%"
			// Each token must match at least one of the columns; chain WHEREs to AND tokens.
			sel = sel.Where("(lower(comment) LIKE ? OR lower(algorithm) LIKE ? OR lower(key_data) LIKE ?)", like, like, like)
		}
		if err := sel.Scan(ctx); err != nil {
			return nil, err
		}
	}
	out := make([]model.PublicKey, 0, len(pks))
	for _, p := range pks {
		out = append(out, publicKeyModelToModel(p))
	}
	return out, nil
}

// --- Known hosts helpers ---
func GetKnownHostKeyBun(bdb *bun.DB, hostname string) (string, error) {
	ctx := context.Background()
	var kh KnownHostModel
	err := bdb.NewSelect().Model(&kh).Where("hostname = ?", hostname).Limit(1).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return kh.Key, nil
}

func AddKnownHostKeyBun(bdb *bun.DB, hostname, key string) error {
	ctx := context.Background()
	_, err := ExecRaw(ctx, bdb, "INSERT OR REPLACE INTO known_hosts (hostname, key) VALUES (?, ?)", hostname, key)
	return MapDBError(err)
}

// --- Bootstrap session helpers ---
func SaveBootstrapSessionBun(bdb *bun.DB, id, username, hostname, label, tags, tempPublicKey string, expiresAt time.Time, status string) error {
	ctx := context.Background()
	_, err := ExecRaw(ctx, bdb, `INSERT INTO bootstrap_sessions (id, username, hostname, label, tags, temp_public_key, expires_at, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, id, username, hostname, label, tags, tempPublicKey, expiresAt, status)
	return MapDBError(err)
}

func GetBootstrapSessionBun(bdb *bun.DB, id string) (*model.BootstrapSession, error) {
	ctx := context.Background()
	var bsm BootstrapSessionModel
	err := bdb.NewSelect().Model(&bsm).Where("id = ?", id).Limit(1).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	m := bootstrapSessionModelToModel(bsm)
	return &m, nil
}

func DeleteBootstrapSessionBun(bdb *bun.DB, id string) error {
	ctx := context.Background()
	_, err := ExecRaw(ctx, bdb, "DELETE FROM bootstrap_sessions WHERE id = ?", id)
	return err
}

func UpdateBootstrapSessionStatusBun(bdb *bun.DB, id string, status string) error {
	ctx := context.Background()
	_, err := ExecRaw(ctx, bdb, "UPDATE bootstrap_sessions SET status = ? WHERE id = ?", status, id)
	return err
}

func GetExpiredBootstrapSessionsBun(bdb *bun.DB) ([]*model.BootstrapSession, error) {
	ctx := context.Background()
	var bss []BootstrapSessionModel
	// SQLite: compare against datetime('now'); Bun will pass through the query.
	if err := bdb.NewSelect().Model(&bss).Where("expires_at < datetime('now')").Scan(ctx); err != nil {
		return nil, err
	}
	out := make([]*model.BootstrapSession, 0, len(bss))
	for _, b := range bss {
		bs := bootstrapSessionModelToModel(b)
		out = append(out, &bs)
	}
	return out, nil
}

func GetOrphanedBootstrapSessionsBun(bdb *bun.DB) ([]*model.BootstrapSession, error) {
	ctx := context.Background()
	var bss []BootstrapSessionModel
	if err := bdb.NewSelect().Model(&bss).Where("status = 'orphaned'").Scan(ctx); err != nil {
		return nil, err
	}
	out := make([]*model.BootstrapSession, 0, len(bss))
	for _, b := range bss {
		bs := bootstrapSessionModelToModel(b)
		out = append(out, &bs)
	}
	return out, nil
}

// --- Account update helpers ---
func GetAccountByIDBun(bdb *bun.DB, id int) (*model.Account, error) {
	ctx := context.Background()
	var am AccountModel
	err := bdb.NewSelect().Model(&am).Where("id = ?", id).Limit(1).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	m := accountModelToModel(am)
	return &m, nil
}

func UpdateAccountSerialBun(bdb *bun.DB, id, serial int) error {
	ctx := context.Background()
	_, err := ExecRaw(ctx, bdb, "UPDATE accounts SET serial = ? WHERE id = ?", serial, id)
	return err
}

func ToggleAccountStatusBun(bdb *bun.DB, id int) (bool, error) {
	ctx := context.Background()
	if _, err := ExecRaw(ctx, bdb, "UPDATE accounts SET is_active = NOT is_active WHERE id = ?", id); err != nil {
		return false, err
	}
	var am AccountModel
	if err := bdb.NewSelect().Model(&am).Where("id = ?", id).Limit(1).Scan(ctx); err != nil {
		return false, err
	}
	return am.IsActive, nil
}

func UpdateAccountLabelBun(bdb *bun.DB, id int, label string) error {
	ctx := context.Background()
	_, err := ExecRaw(ctx, bdb, "UPDATE accounts SET label = ? WHERE id = ?", label, id)
	return err
}

func UpdateAccountHostnameBun(bdb *bun.DB, id int, hostname string) error {
	ctx := context.Background()
	_, err := ExecRaw(ctx, bdb, "UPDATE accounts SET hostname = ? WHERE id = ?", hostname, id)
	return err
}

func UpdateAccountTagsBun(bdb *bun.DB, id int, tags string) error {
	ctx := context.Background()
	_, err := ExecRaw(ctx, bdb, "UPDATE accounts SET tags = ? WHERE id = ?", tags, id)
	return err
}

// --- System key helpers ---
func GetSystemKeyBySerialBun(bdb *bun.DB, serial int) (*model.SystemKey, error) {
	ctx := context.Background()
	var sk SystemKeyModel
	err := bdb.NewSelect().Model(&sk).Where("serial = ?", serial).Limit(1).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	m := systemKeyModelToModel(sk)
	return &m, nil
}

func HasSystemKeysBun(bdb *bun.DB) (bool, error) {
	ctx := context.Background()
	var count int
	if err := QueryRawInto(ctx, bdb, &count, "SELECT COUNT(id) FROM system_keys"); err != nil {
		return false, err
	}
	return count > 0, nil
}

func CreateSystemKeyBun(bdb *bun.DB, publicKey, privateKey string) (int, error) {
	ctx := context.Background()
	// Get max serial
	var max sql.NullInt64
	if err := QueryRawInto(ctx, bdb, &max, "SELECT MAX(serial) FROM system_keys"); err != nil {
		return 0, err
	}
	newSerial := 1
	if max.Valid {
		newSerial = int(max.Int64) + 1
	}
	// Insert new key (do not deactivate others)
	if _, err := ExecRaw(ctx, bdb, "INSERT INTO system_keys(serial, public_key, private_key, is_active) VALUES(?, ?, ?, ?)", newSerial, publicKey, privateKey, true); err != nil {
		return 0, err
	}
	return newSerial, nil
}
