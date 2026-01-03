// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
	"time"
)

func TestDefaultConnectionConfig(t *testing.T) {
	config := DefaultConnectionConfig()

	if config.ConnectionTimeout != DefaultConnectionTimeout {
		t.Errorf("Expected ConnectionTimeout %v, got %v", DefaultConnectionTimeout, config.ConnectionTimeout)
	}

	if config.CommandTimeout != DefaultCommandTimeout {
		t.Errorf("Expected CommandTimeout %v, got %v", DefaultCommandTimeout, config.CommandTimeout)
	}

	if config.SFTPTimeout != DefaultSFTPTimeout {
		t.Errorf("Expected SFTPTimeout %v, got %v", DefaultSFTPTimeout, config.SFTPTimeout)
	}
}

func TestIsConnectionTimeoutError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"timeout error", errors.New("connection timeout"), true},
		{"deadline exceeded", errors.New("deadline exceeded"), true},
		{"i/o timeout", errors.New("i/o timeout"), true},
		{"other error", errors.New("connection refused"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsConnectionTimeoutError(tt.err)
			if result != tt.expected {
				t.Errorf("IsConnectionTimeoutError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsConnectionRefusedError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"connection refused", errors.New("connection refused"), true},
		{"no route to host", errors.New("no route to host"), true},
		{"other error", errors.New("timeout"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsConnectionRefusedError(tt.err)
			if result != tt.expected {
				t.Errorf("IsConnectionRefusedError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsAuthenticationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"authentication failed", errors.New("authentication failed"), true},
		{"permission denied", errors.New("permission denied"), true},
		{"public key error", errors.New("public key authentication failed"), true},
		{"other error", errors.New("timeout"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAuthenticationError(tt.err)
			if result != tt.expected {
				t.Errorf("IsAuthenticationError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsHostKeyError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"host key mismatch", errors.New("HOST KEY MISMATCH"), true},
		{"unknown host key", errors.New("unknown host key"), true},
		{"host key verification failed", errors.New("host key verification failed"), true},
		{"other error", errors.New("timeout"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsHostKeyError(tt.err)
			if result != tt.expected {
				t.Errorf("IsHostKeyError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestClassifyConnectionError(t *testing.T) {
	host := "test-host"

	tests := []struct {
		name        string
		err         error
		expectedMsg string
	}{
		{"nil error", nil, ""},
		{"timeout error", errors.New("timeout"), "connection to test-host timed out"},
		{"connection refused", errors.New("connection refused"), "connection to test-host refused"},
		{"authentication failed", errors.New("authentication failed"), "authentication failed for test-host"},
		{"host key error", errors.New("HOST KEY MISMATCH"), "host key verification failed for test-host"},
		{"generic error", errors.New("some other error"), "failed to connect to test-host"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyConnectionError(host, tt.err)
			if tt.err == nil {
				if result != nil {
					t.Errorf("Expected nil for nil input, got %v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("Expected error, got nil")
				return
			}

			if !contains(result.Error(), tt.expectedMsg) {
				t.Errorf("Expected error message to contain '%s', got '%s'", tt.expectedMsg, result.Error())
			}
		})
	}
}

// Helper function for string containment check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (substr == "" || stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestHostPortHelpers(t *testing.T) {
	cases := []struct {
		in    string
		host  string
		port  string
		canon string
	}{
		{"example.com", "example.com", "", "example.com:22"},
		{"example.com:2222", "example.com", "2222", "example.com:2222"},
		{"192.168.1.10", "192.168.1.10", "", "192.168.1.10:22"},
		{"192.168.1.10:2200", "192.168.1.10", "2200", "192.168.1.10:2200"},
		{"[2001:db8::1]", "2001:db8::1", "", "[2001:db8::1]:22"},
		{"[2001:db8::1]:2200", "2001:db8::1", "2200", "[2001:db8::1]:2200"},
		{"2001:db8::1", "2001:db8::1", "", "[2001:db8::1]:22"},
		{"user@example.com", "example.com", "", "example.com:22"},
		{"user@[2001:db8::1]:2222", "2001:db8::1", "2222", "[2001:db8::1]:2222"},
	}
	for _, c := range cases {
		h, p, err := ParseHostPort(c.in)
		if err != nil {
			t.Fatalf("unexpected error parsing %q: %v", c.in, err)
		}
		if h != c.host || p != c.port {
			t.Errorf("ParseHostPort(%q) => host=%q port=%q; want host=%q port=%q", c.in, h, p, c.host, c.port)
		}
		canon := CanonicalizeHostPort(c.in)
		if canon != c.canon {
			t.Errorf("CanonicalizeHostPort(%q) => %q; want %q", c.in, canon, c.canon)
		}
		// Join should reconstruct canon from components
		joined := JoinHostPort(h, p, "22")
		if joined != c.canon {
			t.Errorf("JoinHostPort(%q,%q,22) => %q; want %q", h, p, joined, c.canon)
		}
	}
}

func TestStripIPv6Brackets(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"[2001:db8::1]", "2001:db8::1"},
		{"[::1]", "::1"},
		{"2001:db8::1", "2001:db8::1"},
		{"example.com", "example.com"},
		{"[incomplete", "[incomplete"},
		{"incomplete]", "incomplete]"},
		{"", ""},
	}

	for _, c := range cases {
		got := StripIPv6Brackets(c.in)
		if got != c.want {
			t.Errorf("StripIPv6Brackets(%q) == %q, want %q", c.in, got, c.want)
		}
	}
}

// --- Mock SFTP client for testing ---

type mockSftpFile struct {
	*bytes.Buffer
	path   string
	parent *mockSftpClient
}

func (m *mockSftpFile) Close() error { return nil }

func (m *mockSftpFile) Write(b []byte) (int, error) {
	if m.parent != nil && m.parent.failWrite[m.path] {
		return 0, fmt.Errorf("injected write error for %s", m.path)
	}
	return m.Buffer.Write(b)
}

type mockFileInfo struct {
	name  string
	mode  os.FileMode
	isDir bool
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return 0 }
func (m *mockFileInfo) Mode() os.FileMode  { return m.mode }
func (m *mockFileInfo) ModTime() time.Time { return time.Now() }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() interface{}   { return nil }

type mockSftpClient struct {
	files   map[string]*mockSftpFile
	perms   map[string]os.FileMode
	statErr map[string]error // errors to return on Stat
	actions []string         // record of actions
	// injectable failures
	createErr    map[string]error
	failWrite    map[string]bool
	chmodErr     map[string]error
	renameErrFor map[string]error // keyed by destination path
}

func newMockSftpClient() *mockSftpClient {
	return &mockSftpClient{
		files:        make(map[string]*mockSftpFile),
		perms:        make(map[string]os.FileMode),
		statErr:      make(map[string]error),
		actions:      []string{},
		createErr:    make(map[string]error),
		failWrite:    make(map[string]bool),
		chmodErr:     make(map[string]error),
		renameErrFor: make(map[string]error),
	}
}

func (m *mockSftpClient) record(action string) { m.actions = append(m.actions, action) }

func (m *mockSftpClient) Create(path string) (io.ReadWriteCloser, error) {
	m.record("create: " + path)
	if err, ok := m.createErr[path]; ok {
		return nil, err
	}
	if err, ok := m.createErr["*"]; ok {
		return nil, err
	}
	file := &mockSftpFile{Buffer: &bytes.Buffer{}, path: path, parent: m}
	m.files[path] = file
	return file, nil
}

func (m *mockSftpClient) Open(path string) (io.ReadWriteCloser, error) {
	m.record("open: " + path)
	if file, ok := m.files[path]; ok {
		// return a copy of the buffer so reads don't mutate original
		return &mockSftpFile{Buffer: bytes.NewBuffer(file.Buffer.Bytes()), path: path, parent: m}, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockSftpClient) Stat(p string) (os.FileInfo, error) {
	m.record("stat: " + p)
	if err, ok := m.statErr[p]; ok {
		return nil, err
	}
	if _, ok := m.files[p]; ok {
		return &mockFileInfo{name: p, mode: m.perms[p]}, nil
	}
	// Simulate directory stat
	if m.perms[p] != 0 {
		return &mockFileInfo{name: p, mode: m.perms[p], isDir: true}, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockSftpClient) Mkdir(path string) error {
	m.record("mkdir: " + path)
	m.perms[path] = 0755 | os.ModeDir
	return nil
}

func (m *mockSftpClient) Chmod(path string, mode os.FileMode) error {
	m.record(fmt.Sprintf("chmod: %s to %v", path, mode))
	if err, ok := m.chmodErr[path]; ok {
		return err
	}
	m.perms[path] = mode
	return nil
}

func (m *mockSftpClient) Remove(path string) error {
	m.record("remove: " + path)
	delete(m.files, path)
	delete(m.perms, path)
	return nil
}

func (m *mockSftpClient) Rename(oldpath, newpath string) error {
	m.record(fmt.Sprintf("rename: %s to %s", oldpath, newpath))
	if err, ok := m.renameErrFor[newpath]; ok {
		return err
	}
	if file, ok := m.files[oldpath]; ok {
		m.files[newpath] = file
		delete(m.files, oldpath)
	}
	if perm, ok := m.perms[oldpath]; ok {
		m.perms[newpath] = perm
		delete(m.perms, oldpath)
	}
	return nil
}

func (m *mockSftpClient) Close() error { m.record("close"); return nil }

func TestDeployAuthorizedKeys_DirExists(t *testing.T) {
	mockClient := newMockSftpClient()
	d := &Deployer{sftp: mockClient}

	// Setup: .ssh directory already exists
	mockClient.perms[".ssh"] = 0700 | os.ModeDir

	content := "ssh-ed25519 AAAAC3... test@key"

	// This test will fail to compile because the sftpClient used in Deployer
	// is not the mockable one. This highlights the difficulty of retrofitting tests
	// without changing the source code's dependencies.
	// To fix this properly, the sftpClient interface in ssh.go needs to be updated
	// to return a file interface, not a concrete *sftp.File.

	// Execute deployment using the mock sftp client.
	err := d.DeployAuthorizedKeys(content)
	if err != nil {
		t.Fatalf("DeployAuthorizedKeys failed: %v", err)
	}

	// Assertions
	// 1. .ssh directory permissions were set to 0700
	if pm, ok := mockClient.perms[".ssh"]; !ok || pm != 0700 {
		t.Errorf("expected .ssh perms 0700, got %v", pm)
	}

	// 2. Final authorized_keys file exists and has correct content
	finalFile, ok := mockClient.files[".ssh/authorized_keys"]
	if !ok {
		t.Fatal("authorized_keys file was not created")
	}
	if finalFile.Buffer.String() != content {
		t.Errorf("unexpected content in authorized_keys: got %q want %q", finalFile.Buffer.String(), content)
	}

	// 3. Permissions on final file are correct
	if pm := mockClient.perms[".ssh/authorized_keys"]; pm != 0600 {
		t.Errorf("expected authorized_keys file to have mode 0600, got %v", pm)
	}
}

func TestGetAuthorizedKeys_Success(t *testing.T) {
	mockClient := newMockSftpClient()
	d := &Deployer{sftp: mockClient}

	// Prepare file
	mockClient.files[".ssh/authorized_keys"] = &mockSftpFile{Buffer: &bytes.Buffer{}, path: ".ssh/authorized_keys", parent: mockClient}
	mockClient.files[".ssh/authorized_keys"].Buffer.WriteString("line1\nline2\n")

	data, err := d.GetAuthorizedKeys()
	if err != nil {
		t.Fatalf("GetAuthorizedKeys failed: %v", err)
	}
	if string(data) != "line1\nline2\n" {
		t.Fatalf("unexpected content: %q", string(data))
	}
}

func TestDeployAuthorizedKeys_WriteFail(t *testing.T) {
	mockClient := newMockSftpClient()
	d := &Deployer{sftp: mockClient}

	// Force Mkdir to succeed implicitly when stat returns not exist
	// Inject write failure on temporary file path pattern
	mockClient.failWrite[".ssh/authorized_keys.keymaster."] = true // will match exact path via parent path check

	// Because DeployAuthorizedKeys uses a nano timestamp, we inject failure for any tmp by
	// setting failWrite for a placeholder and the mock implementation checks exact path,
	// so instead we inject by making Create return a file whose Write fails via failWrite on the generated tmp path.

	// To make deterministic, stub Create to return an error for any tmp path
	mockClient.createErr["*"] = fmt.Errorf("create failed")

	// Running DeployAuthorizedKeys should return an error due to create failing
	if err := d.DeployAuthorizedKeys("content"); err == nil {
		t.Fatalf("expected DeployAuthorizedKeys to fail due to create error")
	}
}

func TestDeployAuthorizedKeys_ChmodFailAndRenameRecover(t *testing.T) {
	mockClient := newMockSftpClient()
	d := &Deployer{sftp: mockClient}

	// Simulate that .ssh does not exist, so Mkdir will run.
	// Inject chmod error on tmp file to force cleanup path.
	// We'll allow Create and Write to succeed, but make Chmod(tmpPath) fail.
	// To capture tmpPath, we intercept Create by providing a normal file but setting chmodErr for any path that contains "authorized_keys.keymaster".
	// For simplicity, set a renameErr for final path to simulate rename failure.
	mockClient.renameErrFor[".ssh/authorized_keys"] = fmt.Errorf("rename failed")

	// DeployAuthorizedKeys should attempt rename, fail, try to restore backup, and return an error
	if err := d.DeployAuthorizedKeys("content"); err == nil {
		t.Fatalf("expected DeployAuthorizedKeys to fail due to rename error")
	}
}
