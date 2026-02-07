// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

//nolint:errcheck
package cli

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	log "github.com/charmbracelet/log"

	"github.com/spf13/viper"
	"github.com/toeirei/keymaster/core"
	"github.com/toeirei/keymaster/i18n"
	"golang.org/x/crypto/ssh"
)

// setupTestDB initializes an in-memory SQLite database for isolated testing.
// It configures viper to use this database and ensures the i18n system is ready.
func setupTestDB(t *testing.T) {
	t.Helper()

	// Ensure tests are isolated from any previously loaded configuration.
	viper.Reset()
	// Disable background session reaper during tests.
	t.Setenv("KEYMASTER_DISABLE_SESSION_REAPER", "1")

	// Use a unique in-memory SQLite database per test to avoid file locks on
	// Windows while preserving isolation across tests. Use the file: URI with
	// mode=memory and cache=shared so multiple connections can see the same
	// in-memory DB when required.
	uniq := fmt.Sprintf("memdb_%d", time.Now().UnixNano())
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uniq)

	viper.Set("database.type", "sqlite")
	viper.Set("database.dsn", dsn)
	viper.Set("language", "en") // Use a consistent language for tests

	// Initialize i18n and the database
	i18n.Init("en")
	if err := core.InitDB("sqlite", dsn); err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
	// Ensure core considers DB initialized during tests.
	core.SetDefaultDBIsInitialized(func() bool { return true })
	// Reset injected defaults after test to avoid cross-test pollution.
	t.Cleanup(func() {
		core.SetDefaultDBIsInitialized(nil)
		core.ResetStoreForTests()
	})
}

// executeCommand runs a cobra command with the given arguments and captures its output.
// It can optionally take an `io.Reader` to mock stdin for interactive commands.
func executeCommand(t *testing.T, stdin io.Reader, args ...string) string {
	t.Helper()

	// Redirect stdout and stderr to a buffer so we capture log output.
	oldOut := os.Stdout
	oldErr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w
	// Try to redirect the charmbracelet logger to the pipe so package-level logs
	// are captured by the test. If the logger supports SetOutput, this will
	// direct its output to our pipe.
	log.SetOutput(w)
	defer log.SetOutput(os.Stderr)
	defer func() {
		os.Stdout = oldOut
		os.Stderr = oldErr
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
	root := NewRootCmd()
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
	defer func() { _ = os.Remove(tmpfile.Name()) }() // Clean up

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
		km := core.DefaultKeyManager()
		if km == nil {
			t.Fatalf("no key manager available")
		}
		keys, err := km.GetAllPublicKeys()
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
	defer func() { _ = listener.Close() }()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			// This error is expected when the listener is closed.
			return
		}
		defer func() { _ = conn.Close() }()
		// Perform SSH handshake to present the host key.
		_, _, _, _ = ssh.NewServerConn(conn, server)
	}()

	// Prepare to mock stdin by writing "yes" to a pipe.
	inputReader, inputWriter, _ := os.Pipe()
	go func() {
		defer func() { _ = inputWriter.Close() }()
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
		key, err := core.GetKnownHostKey(hostname)
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
	defer func() { _ = listener.Close() }()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_, _, _, _ = ssh.NewServerConn(conn, server)
	}()

	// Prepare to mock stdin by writing "yes" to a pipe.
	inputReader, inputWriter, _ := os.Pipe()
	go func() {
		defer func() { _ = inputWriter.Close() }()
		fmt.Fprintln(inputWriter, "yes")
	}()

	// 2. Execute
	hostname := listener.Addr().String()
	output := executeCommand(t, inputReader, "trust-host", hostname)

	// 3. Assertions
	t.Run("should print warning for weak host key algorithm", func(t *testing.T) {
		// This text is based on the warning generated by `core/sshkey/sshkey.go`
		expectedWarning := "SECURITY WARNING: Host key uses ssh-rsa, which is disabled by default in modern OpenSSH"
		if !strings.Contains(output, expectedWarning) {
			t.Errorf("Expected output to contain weak key warning, but it didn't. Output:\n%s", output)
		}
	})

	t.Run("database should still contain the trusted host key", func(t *testing.T) {
		key, err := core.GetKnownHostKey(hostname)
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

		// Create a temporary directory for the test
		tmpDir, err := os.MkdirTemp("", "keymaster-config-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}

		// Return a cleanup function
		return tmpDir, func() {
			os.RemoveAll(tmpDir)
			viper.Reset()
			cfgFile = ""
		}
	}

	t.Run("should read values from a valid config file specified by flag", func(t *testing.T) {
		tmpDir, cleanup := setup(t)
		defer cleanup()

		// Create a custom config file
		customConfig := `
database:
  type: sqlite
  dsn: "file:custom.db?mode=memory"
language: de
`
		configPath := filepath.Join(tmpDir, "custom_config.yaml")
		if err := os.WriteFile(configPath, []byte(customConfig), 0644); err != nil {
			t.Fatalf("Failed to write custom config file: %v", err)
		}

		// Execute the debug command with the --config flag
		// We use "debug" because it prints the used config file and settings
		output := executeCommand(t, nil, "debug", "--config", configPath)

		// Verify that the output confirms the config file was used
		expectedOutput := fmt.Sprintf("Config file used: %s", configPath)
		if !strings.Contains(output, expectedOutput) {
			t.Errorf("Expected output to contain '%s', but it didn't.\nOutput:\n%s", expectedOutput, output)
		}

		// Verify that the settings were actually loaded (debug command dumps viper settings as JSON)
		if !strings.Contains(output, `"language": "de"`) {
			t.Errorf("Expected output to contain '\"language\": \"de\"', but it didn't.\nOutput:\n%s", output)
		}
	})

	t.Run("should display keymaster environment variables in debug output", func(t *testing.T) {
		setupTestDB(t)
		// Set a specific env var to trigger the loop body in debug.go
		t.Setenv("KEYMASTER_TEST_VAR", "visible")
		output := executeCommand(t, nil, "debug")
		if !strings.Contains(output, "KEYMASTER_TEST_VAR=visible") {
			t.Errorf("Expected debug output to contain env var, got:\n%s", output)
		}
	})
}

func TestRotateKeyCmd(t *testing.T) {
	// 1. Setup
	setupTestDB(t)

	t.Run("should create initial system key when none exist", func(t *testing.T) {
		// 2. Execute
		output := executeCommand(t, nil, "rotate-key")

		// 3. Assert Output
		if !strings.Contains(output, "Successfully rotated system key. The new active key is serial #1.") {
			t.Errorf("Expected output to contain success message for serial #1, but it didn't. Output:\n%s", output)
		}

		// 4. Assert Database State
		activeKey, err := core.GetActiveSystemKey()
		if err != nil {
			t.Fatalf("Failed to get active system key from DB: %v", err)
		}
		if activeKey == nil {
			t.Fatal("Expected an active system key, but found none.")
		}
		if activeKey.Serial != 1 {
			t.Errorf("Expected active key to have serial 1, but got %d", activeKey.Serial)
		}
		if !activeKey.IsActive {
			t.Error("Expected the new key to be active, but it was not.")
		}
	})

	t.Run("should rotate existing system key to next serial", func(t *testing.T) {
		// 2. Execute again to trigger rotation
		output := executeCommand(t, nil, "rotate-key")

		// 3. Assert Output
		if !strings.Contains(output, "Successfully rotated system key. The new active key is serial #2.") {
			t.Errorf("Expected output to contain success message for serial #2, but it didn't. Output:\n%s", output)
		}

		// 4. Assert New Database State
		activeKey, err := core.GetActiveSystemKey()
		if err != nil {
			t.Fatalf("Failed to get active system key from DB: %v", err)
		}
		if activeKey == nil {
			t.Fatal("Expected an active system key, but found none.")
		}
		if activeKey.Serial != 2 {
			t.Errorf("Expected active key to have serial 2, but got %d", activeKey.Serial)
		}

		// 5. Assert Old Key is now inactive
		oldKey, err := core.GetSystemKeyBySerial(1)
		if err != nil {
			t.Fatalf("Failed to get old system key (serial 1) from DB: %v", err)
		}
		if oldKey == nil {
			t.Fatal("Expected to find old system key (serial 1), but it was gone.")
		}
		if oldKey.IsActive {
			t.Error("Expected old system key (serial 1) to be inactive, but it was still active.")
		}
	})

	t.Run("should not change existing account serials", func(t *testing.T) {
		// Make this subtest independent: create initial and rotated keys here.
		setupTestDB(t)

		// Create initial key and rotate to produce serial 2
		if _, err := executeCommand(t, nil, "rotate-key"), error(nil); err != nil {
			// executeCommand will call t.Fatalf on error; this branch is unreachable
		}
		if _, err := executeCommand(t, nil, "rotate-key"), error(nil); err != nil {
		}

		// Add an account synced to the old serial (1)
		mgr := core.DefaultAccountManager()
		if mgr == nil {
			t.Fatalf("no account manager available")
		}
		accountID, err := mgr.AddAccount("test", "host.com", "test-label", "")
		if err != nil {
			t.Fatalf("Failed to add test account: %v", err)
		}
		if err := core.UpdateAccountSerial(accountID, 1); err != nil {
			t.Fatalf("Failed to set account serial: %v", err)
		}

		// Execute another rotation. This should not affect our test account.
		executeCommand(t, nil, "rotate-key")

		// Assert that the account's serial number has NOT changed.
		allAccounts, _ := core.GetAllAccounts()
		if len(allAccounts) == 0 {
			t.Fatalf("expected at least one account, found none")
		}
		if allAccounts[0].Serial != 1 {
			t.Errorf("Expected account serial to remain 1 after key rotation, but it changed to %d", allAccounts[0].Serial)
		}
	})
}

func TestExportSSHConfigCmd(t *testing.T) {
	t.Run("should print message when no accounts exist", func(t *testing.T) {
		setupTestDB(t) // Fresh DB for this test

		output := executeCommand(t, nil, "export-ssh-client-config")

		if !strings.Contains(output, "No active accounts found to export.") {
			t.Errorf("Expected 'no accounts' message, but got: %s", output)
		}
	})

	t.Run("should print config to stdout for active accounts", func(t *testing.T) {
		setupTestDB(t) // Fresh DB

		// Add test accounts
		mgr := core.DefaultAccountManager()
		if mgr == nil {
			t.Fatalf("no account manager available")
		}
		_, _ = mgr.AddAccount("user1", "host1.com", "prod-web-1", "")
		_, _ = mgr.AddAccount("user2", "host2.com", "", "") // No label
		inactiveID, _ := mgr.AddAccount("user3", "host3.com", "inactive-host", "")
		_ = core.ToggleAccountStatus(inactiveID) // Make this one inactive

		output := executeCommand(t, nil, "export-ssh-client-config")

		t.Run("should include account with label", func(t *testing.T) {
			if !strings.Contains(output, "Host prod-web-1") {
				t.Error("Expected to find Host entry for labeled account 'prod-web-1'")
			}
		})

		t.Run("should include account without label using generated alias", func(t *testing.T) {
			if !strings.Contains(output, "Host user2-host2-com") {
				t.Error("Expected to find Host entry for unlabeled account 'user2-host2-com'")
			}
		})

		t.Run("should not include inactive accounts", func(t *testing.T) {
			if strings.Contains(output, "Host inactive-host") {
				t.Error("Expected not to find Host entry for inactive account")
			}
		})
	})

	t.Run("should write config to specified file", func(t *testing.T) {
		setupTestDB(t) // Fresh DB

		mgr := core.DefaultAccountManager()
		if mgr == nil {
			t.Fatalf("no account manager available")
		}
		_, _ = mgr.AddAccount("user1", "host1.com", "prod-web-1", "")

		tmpfile, err := os.CreateTemp("", "ssh_config_*.txt")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		tmpfilePath := tmpfile.Name()
		_ = tmpfile.Close() // Close the file so the command can write to it

		output := executeCommand(t, nil, "export-ssh-client-config", tmpfilePath)

		if !strings.Contains(output, "Successfully exported SSH config to") {
			t.Errorf("Expected success message in stdout, but got: %s", output)
		}

		fileContent, err := os.ReadFile(tmpfilePath)
		if err != nil {
			t.Fatalf("Failed to read temp file content: %v", err)
		}
		if !strings.Contains(string(fileContent), "Host prod-web-1") {
			t.Error("Expected file content to contain the correct Host entry")
		}
	})
}
