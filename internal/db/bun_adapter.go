package db

import (
	"context"
	"database/sql"
	"fmt"

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
