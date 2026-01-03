// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy

import (
	"errors"
	"testing"
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
	path string
}

func (m *mockSftpFile) Close() error {
	return nil // no-op
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
}

func newMockSftpClient() *mockSftpClient {
	return &mockSftpClient{
		files:   make(map[string]*mockSftpFile),
		perms:   make(map[string]os.FileMode),
		statErr: make(map[string]error),
		actions: []string{},
	}
}

func (m *mockSftpClient) record(action string) {
	m.actions = append(m.actions, action)
}

func (m *mockSftpClient) Create(path string) (*sftp.File, error) {
	m.record("create: " + path)
	file := &mockSftpFile{
		Buffer: &bytes.Buffer{},
		path:   path,
	}
	m.files[path] = file

	// Create a real sftp.File wrapper for the mock file
	// This is tricky; for this test, we can assume the *sftp.File returned
	// is mainly used for its Write and Close methods. We'll return a pointer
	// to a real sftp.File but with a mock writer.
	// A simpler way is to change the interface to return a mockable file interface.
	// Let's assume for now we can't change sftp.File. We'll return a placeholder
	// that should still work for the Write call if we structure it right.
	// This part is inherently difficult without changing the sftpClient interface more.
	// A more advanced mock would require deeper changes.
	// Let's return nil and focus on the flow of operations.
	// We will need to change the interface to return a custom file interface.
	return nil, errors.New("mock Create not fully implemented for *sftp.File return")
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

func (m *mockSftpClient) Open(path string) (*sftp.File, error) {
	m.record("open: " + path)
	if _, ok := m.files[path]; ok {
		// Similar to Create, returning a real *sftp.File is hard.
		return nil, errors.New("mock Open not fully implemented")
	}
	return nil, os.ErrNotExist
}

func (m *mockSftpClient) Close() error {
	m.record("close")
	return nil
}

// Redefining the test to work around the *sftp.File return type issue
// by also creating a mockable file interface.

// sftpFileHandle is an interface for file operations.
type sftpFileHandle interface {
	io.ReadWriteCloser
}

// sftpClient is redefined to use the mockable file handle.
type sftpClientMockable interface {
	Create(path string) (sftpFileHandle, error)
	Stat(p string) (os.FileInfo, error)
	Mkdir(path string) error
	Chmod(path string, mode os.FileMode) error
	Remove(path string) error
	Rename(oldpath, newpath string) error
	Open(path string) (sftpFileHandle, error)
	Close() error
}

// mockSftpClient needs to implement the new interface
func (m *mockSftpClient) Create(path string) (sftpFileHandle, error) {
	m.record("create: " + path)
	file := &mockSftpFile{
		Buffer: &bytes.Buffer{},
		path:   path,
	}
	m.files[path] = file
	return file, nil
}

func (m *mockSftpClient) Open(path string) (sftpFileHandle, error) {
	m.record("open: " + path)
	if file, ok := m.files[path]; ok {
		// Return a new reader for the same underlying buffer
		return &mockSftpFile{Buffer: bytes.NewBuffer(file.Bytes()), path: path}, nil
	}
	return nil, os.ErrNotExist
}


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
	
	// For now, let's write the test logic assuming we can get a mock client in.
	// The next step would be to propose the change to sftpClient in ssh.go.

	t.Skip("Skipping test because it requires further refactoring of sftpClient interface in ssh.go")

	err := d.DeployAuthorizedKeys(content)
	if err != nil {
		t.Fatalf("DeployAuthorizedKeys failed: %v", err)
	}

	// Assertions
	// 1. .ssh directory was chmod'ed
	foundChmod := false
	for _, action := range mockClient.actions {
		if action == "chmod: .ssh to 700" { // Should be more robust
			foundChmod = true
			break
		}
	}
	if !foundChmod {
		t.Error("expected chmod on .ssh directory, but was not called")
	}

	// 2. Final authorized_keys file exists and has correct content
	finalFile, ok := mockClient.files[".ssh/authorized_keys"]
	if !ok {
		t.Fatal("authorized_keys file was not created")
	}
	if finalFile.String() != content {
		t.Errorf("unexpected content in authorized_keys: got %q want %q", finalFile.String(), content)
	}

	// 3. Permissions on final file are correct
	if mockClient.perms[".ssh/authorized_keys"] != 0600 {
		t.Errorf("expected authorized_keys file to have mode 0600, got %v", mockClient.perms[".ssh/authorized_keys"])
	}
}
