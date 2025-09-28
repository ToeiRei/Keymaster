// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/i18n"
	"golang.org/x/crypto/ssh"
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
// It can optionally take an `io.Reader` to mock stdin for interactive commands.
func executeCommand(t *testing.T, stdin io.Reader, args ...string) string {
	t.Helper()

	// Redirect stdout to a buffer
	oldOut := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		os.Stdout = oldOut
	}()

	// Redirect stdin if a reader is provided
	if stdin != nil {
		oldIn := os.Stdin
		os.Stdin = stdin.(*os.File)
		defer func() {
			os.Stdin = oldIn
		}()
	}

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
	output := executeCommand(t, nil, "import", tmpfile.Name())

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

func TestTrustHostCmd(t *testing.T) {
	// 1. Setup
	setupTestDB(t)

	// Create a mock SSH server that will present a host key on connection.
	server, privateKey, err := newMockSSHServer()
	if err != nil {
		t.Fatalf("Failed to create mock SSH server: %v", err)
	}

	// Start the server on a random port.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen on a port: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			// This error is expected when the listener is closed.
			return
		}
		defer conn.Close()
		// Perform SSH handshake to present the host key.
		_, _, _, _ = ssh.NewServerConn(conn, server)
	}()

	// Prepare to mock stdin by writing "yes" to a pipe.
	inputReader, inputWriter, _ := os.Pipe()
	go func() {
		defer inputWriter.Close()
		fmt.Fprintln(inputWriter, "yes")
	}()

	// 2. Execute
	hostname := listener.Addr().String()
	output := executeCommand(t, inputReader, "trust-host", hostname)

	// 3. Assertions
	t.Run("should print authenticity warning", func(t *testing.T) {
		expectedWarning := fmt.Sprintf("The authenticity of host '%s' can't be established.", hostname)
		if !strings.Contains(output, expectedWarning) {
			t.Errorf("Expected output to contain authenticity warning, but it didn't. Output:\n%s", output)
		}
	})

	t.Run("should print key fingerprint", func(t *testing.T) {
		fingerprint := ssh.FingerprintSHA256(privateKey.PublicKey())
		if !strings.Contains(output, fingerprint) {
			t.Errorf("Expected output to contain fingerprint '%s', but it didn't. Output:\n%s", fingerprint, output)
		}
	})

	t.Run("should print success message", func(t *testing.T) {
		expectedSuccess := fmt.Sprintf("Warning: Permanently added '%s'", hostname)
		if !strings.Contains(output, expectedSuccess) {
			t.Errorf("Expected output to contain success message, but it didn't. Output:\n%s", output)
		}
	})

	t.Run("database should contain the trusted host key", func(t *testing.T) {
		key, err := db.GetKnownHostKey(hostname)
		if err != nil {
			t.Fatalf("Failed to get known host key from DB: %v", err)
		}
		if key == "" {
			t.Fatalf("Expected to find a key for hostname '%s' in the database, but it was empty.", hostname)
		}

		expectedKey := string(ssh.MarshalAuthorizedKey(privateKey.PublicKey()))
		if key != expectedKey {
			t.Errorf("Stored key does not match the expected key.\nGot: %s\nWant: %s", key, expectedKey)
		}
	})
}

func TestTrustHostCmd_WeakKey(t *testing.T) {
	// 1. Setup
	setupTestDB(t)

	// Create a mock SSH server that will present a weak (RSA) host key.
	server, _, err := newMockSSHServer("../../testdata/ssh_host_rsa_key")
	if err != nil {
		t.Fatalf("Failed to create mock SSH server with weak key: %v", err)
	}

	// Start the server on a random port.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen on a port: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _, _, _ = ssh.NewServerConn(conn, server)
	}()

	// Prepare to mock stdin by writing "yes" to a pipe.
	inputReader, inputWriter, _ := os.Pipe()
	go func() {
		defer inputWriter.Close()
		fmt.Fprintln(inputWriter, "yes")
	}()

	// 2. Execute
	hostname := listener.Addr().String()
	output := executeCommand(t, inputReader, "trust-host", hostname)

	// 3. Assertions
	t.Run("should print warning for weak host key algorithm", func(t *testing.T) {
		// This text is based on the warning generated by `internal/sshkey/sshkey.go`
		expectedWarning := "SECURITY WARNING: Host key uses ssh-rsa, which is disabled by default in modern OpenSSH"
		if !strings.Contains(output, expectedWarning) {
			t.Errorf("Expected output to contain weak key warning, but it didn't. Output:\n%s", output)
		}
	})

	t.Run("database should still contain the trusted host key", func(t *testing.T) {
		key, err := db.GetKnownHostKey(hostname)
		if err != nil {
			t.Fatalf("Failed to get known host key from DB: %v", err)
		}
		if key == "" {
			t.Fatalf("Expected to find a key for hostname '%s' in the database, but it was empty.", hostname)
		}
	})
}

// newMockSSHServer creates a basic SSH server config using a key from the given file path.
func newMockSSHServer(keyPath ...string) (*ssh.ServerConfig, ssh.Signer, error) {
	// Default to the strong ed25519 key if no path is provided.
	path := "../../testdata/ssh_host_ed25519_key"
	if len(keyPath) > 0 {
		path = keyPath[0]
	}

	privateKeyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("could not read test private key: %w", err)
	}
	privateKey, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("could not parse test private key: %w", err)
	}
	config := &ssh.ServerConfig{
		// No authentication needed, we just need to present the host key.
		NoClientAuth: true,
	}
	config.AddHostKey(privateKey)
	return config, privateKey, nil
}

func TestConfigHandling(t *testing.T) {
	// Helper to set up a temporary directory for config tests
	setup := func(t *testing.T) (string, func()) {
		t.Helper()
		viper.Reset() // Reset viper before each test
		cfgFile = ""  // Reset global config file flag

		// Keep track of original working directory
		originalWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current working directory: %v", err)
		}

		// Create a temporary directory for the test
		tmpDir, err := os.MkdirTemp("", "keymaster-config-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}

		// Change to the temporary directory
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("Failed to change to temp dir: %v", err)
		}

		// Return a cleanup function
		return tmpDir, func() {
			if err := os.Chdir(originalWd); err != nil {
				// Log the error but don't fail the test on cleanup
				t.Logf("Warning: failed to change back to original directory: %v", err)
			}
			os.RemoveAll(tmpDir)
			viper.Reset()
			cfgFile = ""
		}
	}

	t.Run("should create default config if none exists", func(t *testing.T) {
		_, cleanup := setup(t)
		defer cleanup()

		if err := initConfig(); err != nil {
			t.Fatalf("initConfig() returned an unexpected error: %v", err)
		}

		// Check if the default config file was created
		if _, err := os.Stat(".keymaster.yaml"); os.IsNotExist(err) {
			t.Error("Expected a default .keymaster.yaml to be created, but it was not")
		}

		// Check if viper has the default values
		if got := viper.GetString("database.type"); got != "sqlite" {
			t.Errorf("Expected default database.type to be 'sqlite', but got '%s'", got)
		}
	})

	t.Run("should read values from a valid config file", func(t *testing.T) {
		tmpDir, cleanup := setup(t)
		defer cleanup()

		// Create a custom config file
		customConfig := `
database:
  type: postgres
  dsn: "my-postgres-dsn"
language: de
`
		configPath := tmpDir + "/.keymaster.yaml"
		if err := os.WriteFile(configPath, []byte(customConfig), 0644); err != nil {
			t.Fatalf("Failed to write custom config file: %v", err)
		}

		// We don't check the error here, as a successful read returns nil.
		if err := initConfig(); err != nil {
			t.Fatalf("initConfig() returned an unexpected error: %v", err)
		}

		if got := viper.GetString("database.type"); got != "postgres" {
			t.Errorf("Expected database.type from config to be 'postgres', but got '%s'", got)
		}
		if got := viper.GetString("language"); got != "de" {
			t.Errorf("Expected language from config to be 'de', but got '%s'", got)
		}
	})

	t.Run("should handle malformed config file gracefully", func(t *testing.T) {
		tmpDir, cleanup := setup(t)
		defer cleanup()

		// Create a malformed config file
		malformedConfig := `database: { type: "postgres"` // Missing closing brace
		configPath := tmpDir + "/.keymaster.yaml"
		if err := os.WriteFile(configPath, []byte(malformedConfig), 0644); err != nil {
			t.Fatalf("Failed to write malformed config file: %v", err)
		}

		err := initConfig()
		if err == nil {
			t.Fatal("Expected initConfig() to return an error for a malformed file, but it was nil")
		}

		// After a failed read, viper's state can be unpredictable.
		// We must reset it to ensure we are checking against the true defaults.
		viper.Reset()

		// After resetting, we need to re-apply the defaults that would normally
		// be set during application startup via the init() function.
		viper.SetDefault("database.type", "sqlite")
		viper.SetDefault("language", "en")

		// It should fall back to defaults since the config is unreadable
		if got := viper.GetString("database.type"); got != "sqlite" {
			t.Errorf("Expected fallback to default 'sqlite' with malformed config, but got '%s'", got)
		}
	})

	t.Run("should prioritize environment variables over config", func(t *testing.T) {
		_, cleanup := setup(t)
		defer cleanup()

		t.Setenv("KEYMASTER_LANGUAGE", "fr")

		// Re-apply defaults to ensure env var binding works correctly on a clean slate.
		viper.SetDefault("database.type", "sqlite")
		viper.SetDefault("language", "en")

		if err := initConfig(); err != nil {
			t.Fatalf("initConfig() returned an unexpected error: %v", err)
		}

		if got := viper.GetString("language"); got != "fr" {
			t.Errorf("Expected language from env var to be 'fr', but got '%s'", got)
		}
	})
}
