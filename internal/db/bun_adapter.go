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
	err := bdb.NewSelect().Model(&sk).Where("is_active = ?", true).Limit(1).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &model.SystemKey{
		ID:         sk.ID,
		Serial:     sk.Serial,
		PublicKey:  sk.PublicKey,
		PrivateKey: sk.PrivateKey,
		IsActive:   sk.IsActive,
	}, nil
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
