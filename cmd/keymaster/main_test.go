// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/i18n"
)

// setupTestDB initializes an in-memory SQLite database for isolated testing.
// It configures viper to use this database and ensures the i18n system is ready.
func setupTestDB(t *testing.T) {
	t.Helper()

	// Use a unique in-memory database for each test run.
	// "cache=shared" is crucial to allow multiple connections to the same in-memory DB.
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"

	// Configure viper to use our in-memory test DB
	viper.Set("database.type", "sqlite")
	viper.Set("database.dsn", dsn)
	viper.Set("language", "en") // Use a consistent language for tests

	// Initialize i18n and the database
	i18n.Init("en")
	if err := db.InitDB("sqlite", dsn); err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
}

// executeCommand runs a cobra command with the given arguments and captures its output.
func executeCommand(t *testing.T, args ...string) string {
	t.Helper()

	// Redirect stdout to a buffer
	oldOut := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		os.Stdout = oldOut
	}()

	// Create a new root command for each test to ensure isolation
	root := newRootCmd()
	root.SetArgs(args)

	// Execute the command
	if err := root.Execute(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	// Read the output from the buffer
	w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read command output: %v", err)
	}

	return buf.String()
}

func TestImportCmd(t *testing.T) {
	// 1. Setup
	setupTestDB(t)

	// Create a temporary authorized_keys file for the test
	content := `
# This is a comment, should be ignored
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGy5E/P9Ea45T/k+s/p3g4zJzE4Q3g== user@example.com
invalid-key-line
ssh-ed25519 BBBBC3NzaC1lZDI1NTE5AAAAIGy5E/P9Ea45T/k+s/p3g4zJzE4Q3g==
ssh-ed25519 CCCCC3NzaC1lZDI1NTE5AAAAIGy5E/P9Ea45T/k+s/p3g4zJzE4Q3g== user@example.com
`
	tmpfile, err := os.CreateTemp("", "authorized_keys_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // Clean up

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// 2. Execute
	output := executeCommand(t, "import", tmpfile.Name())

	// 3. Assertions
	t.Run("should print success message for imported key", func(t *testing.T) {
		if !strings.Contains(output, "Imported key: user@example.com") {
			t.Errorf("Expected output to contain import success message for 'user@example.com', but it didn't. Output:\n%s", output)
		}
	})

	t.Run("should print skip message for key with no comment", func(t *testing.T) {
		if !strings.Contains(output, "Skipping key with empty comment") {
			t.Errorf("Expected output to contain skip message for key with no comment, but it didn't. Output:\n%s", output)
		}
	})

	t.Run("should print skip message for duplicate key", func(t *testing.T) {
		if !strings.Contains(output, "Skipping duplicate key (comment exists): user@example.com") {
			t.Errorf("Expected output to contain skip message for duplicate key, but it didn't. Output:\n%s", output)
		}
	})

	t.Run("should print correct import summary", func(t *testing.T) {
		if !strings.Contains(output, "Import complete. Imported 1 keys, skipped 3.") {
			t.Errorf("Expected summary 'Import complete. Imported 1 keys, skipped 3.', but it was different. Output:\n%s", output)
		}
	})

	t.Run("database should contain exactly one key", func(t *testing.T) {
		keys, err := db.GetAllPublicKeys()
		if err != nil {
			t.Fatalf("Failed to get public keys from DB: %v", err)
		}

		if len(keys) != 1 {
			t.Fatalf("Expected 1 key to be in the database, but found %d", len(keys))
		}

		if keys[0].Comment != "user@example.com" {
			t.Errorf("Expected imported key to have comment 'user@example.com', but got '%s'", keys[0].Comment)
		}
	})
}
