// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// This is a one-time cleanup utility to remove global keys from the account_keys table.
// Global keys should only exist in public_keys with is_global=1, not in account_keys.

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/toeirei/keymaster/core/db"
)

func main() {
	// Initialize the database
	store, err := db.New("sqlite", "keymaster.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	bdb := store.BunDB()
	ctx := context.Background()

	// Find all global keys that are incorrectly in account_keys
	query := `
		SELECT DISTINCT pk.id, pk.comment, COUNT(ak.account_id) as assignment_count
		FROM public_keys pk
		INNER JOIN account_keys ak ON pk.id = ak.key_id
		WHERE pk.is_global = 1
		GROUP BY pk.id, pk.comment
	`

	type result struct {
		ID              int
		Comment         string
		AssignmentCount int
	}

	var results []result
	err = bdb.NewRaw(query).Scan(ctx, &results)
	if err != nil {
		log.Fatalf("Failed to query global keys in account_keys: %v", err)
	}

	if len(results) == 0 {
		fmt.Println("✓ No global keys found in account_keys table. Database is clean!")
		return
	}

	fmt.Printf("Found %d global key(s) incorrectly assigned in account_keys:\n", len(results))
	for _, r := range results {
		fmt.Printf("  - Key ID %d (%s): %d assignment(s)\n", r.ID, r.Comment, r.AssignmentCount)
	}

	// Delete all global keys from account_keys
	deleteQuery := `
		DELETE FROM account_keys
		WHERE key_id IN (
			SELECT id FROM public_keys WHERE is_global = 1
		)
	`

	deleteResult, err := db.ExecRaw(ctx, bdb, deleteQuery)
	if err != nil {
		log.Fatalf("Failed to delete global keys from account_keys: %v", err)
	}

	rowsAffected, _ := deleteResult.RowsAffected()
	fmt.Printf("\n✓ Removed %d incorrect assignment(s) from account_keys table.\n", rowsAffected)

	// Mark all accounts as dirty since key assignments changed
	_, err = db.ExecRaw(ctx, bdb, "UPDATE accounts SET is_dirty = 1")
	if err != nil {
		log.Fatalf("Failed to mark accounts dirty: %v", err)
	}

	fmt.Println("✓ Marked all accounts as dirty for redeployment.")
	fmt.Println("\nCleanup complete! Global keys will now only be deployed via the is_global flag,")
	fmt.Println("not through account_keys assignments.")
}
