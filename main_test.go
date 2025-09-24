// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/i18n"
)

// setupTestDB initializes an in-memory SQLite database for testing.
// It returns a cleanup function that should be deferred by the caller.
func setupTestDB(t *testing.T) func() {
	t.Helper()

	// Use an in-memory SQLite database for tests to ensure they are isolated and fast.
	// The `cache=shared` is important to allow multiple connections to the same in-memory DB.
	dsn := "file::memory:?cache=shared"
	viper.Set("database.type", "sqlite")
	viper.Set("database.dsn", dsn)
	viper.Set("language", "en") // Set a predictable language for tests

	i18n.Init(viper.GetString("language"))

	// We need to call InitDB to run migrations on the in-memory database.
	if err := db.InitDB("sqlite", dsn); err != nil {
		t.Fatalf("Failed to initialize in-memory database: %v", err)
	}

	// The cleanup function can be used to reset viper if needed, though for
	// simple tests it might not be strictly necessary.
	return func() {
		viper.Reset()
	}
}

func TestImportCmd(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// 1. Create a temporary authorized_keys file for the test.
	tempDir := t.TempDir()
	keysFile := filepath.Join(tempDir, "authorized_keys")
	keysContent := `
# This is a comment, should be ignored.
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGpG/1pM7/3hM4/pM7/3hM4/pM7/3hM4/pM7/3hM4 user1@host
ssh-ed25519 BBBBC3NzaC1lZDI1NTE5AAAAIGpG/1pM7/3hM4/pM7/3hM4/pM7/3hM4/pM7/3hM4 user2@host
invalid-key-line
ssh-ed25519 CCCCC3NzaC1lZDI1NTE5AAAAIGpG/1pM7/3hM4/pM7/3hM4/pM7/3hM4/pM7/3hM4 user1@host
`
	if err := os.WriteFile(keysFile, []byte(keysContent), 0644); err != nil {
		t.Fatalf("Failed to write temp keys file: %v", err)
	}

	// 2. Capture the output of the command.
	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)

	// 3. Execute the 'import' command.
	rootCmd.SetArgs([]string{"import", keysFile})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("import command failed: %v", err)
	}

	// 4. Assert the output is correct.
	outStr := output.String()
	expectedImport1 := i18n.T("import.imported_key", "user1@host")
	if !strings.Contains(outStr, expectedImport1) {
		t.Errorf("Expected to see '%s', but didn't. Output:\n%s", expectedImport1, outStr)
	}
	expectedImport2 := i18n.T("import.imported_key", "user2@host")
	if !strings.Contains(outStr, expectedImport2) {
		t.Errorf("Expected to see '%s', but didn't. Output:\n%s", expectedImport2, outStr)
	}
	expectedInvalid := i18n.T("import.skip_invalid_line")
	// We check for the part before the format verb
	if !strings.Contains(outStr, strings.Split(expectedInvalid, ":")[0]) {
		t.Errorf("Expected to see '%s', but didn't. Output:\n%s", expectedInvalid, outStr)
	}
	// This is the crucial check for the duplicate key handling.
	expectedDuplicate := i18n.T("import.skip_duplicate", "user1@host")
	if !strings.Contains(outStr, expectedDuplicate) {
		t.Errorf("Expected to see '%s', but didn't. Output:\n%s", expectedDuplicate, outStr)
	}
	expectedSummary := i18n.T("import.summary", 2, 2)
	if !strings.Contains(outStr, expectedSummary) {
		t.Errorf("Expected summary '%s', but didn't. Output:\n%s", expectedSummary, outStr)
	}
}

func TestTrustHostCmd_ArgumentParsing(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		expectedHost  string
		expectSuccess bool
	}{
		{"Valid user@host format", []string{"trust-host", "user@example.com"}, "example.com", true},
		{"Valid hostname only format", []string{"trust-host", "example.com"}, "example.com", true},
		{"Missing argument", []string{"trust-host"}, "", false},
		{"Too many arguments", []string{"trust-host", "one", "two"}, "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := setupTestDB(t)
			defer cleanup()

			var output bytes.Buffer
			rootCmd.SetOut(&output)
			rootCmd.SetErr(&output)

			rootCmd.SetArgs(tc.args)
			err := rootCmd.Execute()

			if tc.expectSuccess {
				if err != nil {
					// We expect an error later from the network call, but not from arg parsing.
					// For this test, we only care about the initial output.
					expectedOutput := i18n.T("trust_host.retrieving_key", tc.expectedHost)
					if !strings.Contains(output.String(), expectedOutput) {
						t.Errorf("Expected output to contain '%s', but got: %s", expectedOutput, output.String())
					}
				}
			} else {
				if err == nil {
					t.Errorf("Expected an error for args %v, but got none", tc.args)
				}
			}
		})
	}
}
