// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package bootstrap

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/pkg/sftp"
	"github.com/toeirei/keymaster/internal/core/model"
	"golang.org/x/crypto/ssh"
)

// minimal fake sftp client implementing only what's needed by removeTempKeyFromRemoteHost
type fakeSFTP struct {
	files map[string][]byte
}

func (f *fakeSFTP) Open(path string) (io.ReadCloser, error) {
	if f.files == nil {
		return nil, errors.New("no files")
	}
	b, ok := f.files[path]
	if !ok {
		return nil, errors.New("file not found")
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

type fakeWriteCloser struct {
	buf     *bytes.Buffer
	onClose func([]byte)
}

func (f *fakeWriteCloser) Write(p []byte) (int, error) { return f.buf.Write(p) }
func (f *fakeWriteCloser) Close() error {
	if f.onClose != nil {
		f.onClose(f.buf.Bytes())
	}
	return nil
}

func (f *fakeSFTP) Create(path string) (io.WriteCloser, error) {
	if f.files == nil {
		f.files = make(map[string][]byte)
	}
	w := &fakeWriteCloser{buf: &bytes.Buffer{}, onClose: func(b []byte) { f.files[path] = b }}
	return w, nil
}

func (f *fakeSFTP) Close() error { return nil }

// sftp.NewClient is replaced by our package variable in tests; tests will call sftpNewClient directly.

// Define an interface matching the sftp methods we use to allow type compatibility in tests
// test uses the package-level sftpClientIface defined in cleanup.go

// Test successful removal of temporary key from authorized_keys
func TestRemoveTempKeyFromRemoteHost_Success(t *testing.T) {
	// exercise the core string-manipulation behavior via removeLine
	tk, err := generateTemporaryKeyPair()
	if err != nil {
		t.Fatalf("generateTemporaryKeyPair: %v", err)
	}
	content := tk.GetPublicKey() + "\nsecond\nthird\n"
	out := removeLine(content, tk.GetPublicKey())
	if bytes.Contains([]byte(out), []byte(tk.GetPublicKey())) {
		t.Fatalf("expected temp key to be removed")
	}
}

// Test SFTP creation error path
func TestRemoveTempKeyFromRemoteHost_SftpError(t *testing.T) {
	origSshDial := sshDialFunc
	origSftpNew := sftpNewClient
	defer func() {
		sshDialFunc = origSshDial
		sftpNewClient = origSftpNew
	}()

	// minimal temp key
	tk, err := generateTemporaryKeyPair()
	if err != nil {
		t.Fatalf("generateTemporaryKeyPair: %v", err)
	}
	sess := &BootstrapSession{PendingAccount: model.Account{Username: "test", Hostname: "example.com"}, TempKeyPair: tk}

	// simulate ssh.Dial failing to avoid triggering host key callback in unit test
	sshDialFunc = func(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
		return nil, errors.New("dial failure")
	}

	err = removeTempKeyFromRemoteHost(sess)
	if err == nil {
		t.Fatalf("expected error when ssh.Dial fails")
	}
}

// Full success path: override sshDialFunc and sftpNewClient to use a fake SFTP
func TestRemoveTempKeyFromRemoteHost_FullSuccess(t *testing.T) {
	origSshDial := sshDialFunc
	origSftpNew := sftpNewClient
	defer func() {
		sshDialFunc = origSshDial
		sftpNewClient = origSftpNew
	}()

	tk, err := generateTemporaryKeyPair()
	if err != nil {
		t.Fatalf("generateTemporaryKeyPair: %v", err)
	}

	sess := &BootstrapSession{PendingAccount: model.Account{Username: "u", Hostname: "h"}, TempKeyPair: tk}

	// fake SFTP implementation using the package sftpClientIface
	fake := &fakeSFTP{files: make(map[string][]byte)}
	fake.files[".ssh/authorized_keys"] = []byte(tk.GetPublicKey() + "\nother-key\n")

	sshDialFunc = func(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
		return &ssh.Client{}, nil
	}

	sftpNewClient = func(conn *ssh.Client, opts ...sftp.ClientOption) (sftpClientIface, error) {
		return fake, nil
	}

	if err := removeTempKeyFromRemoteHost(sess); err != nil {
		t.Fatalf("removeTempKeyFromRemoteHost failed: %v", err)
	}

	// verify the written file no longer contains the temp public key
	got := fake.files[".ssh/authorized_keys"]
	if bytes.Contains(got, []byte(tk.GetPublicKey())) {
		t.Fatalf("expected temp key removed from written authorized_keys")
	}
}

// helper writer that always fails on Write
type badWriter struct{}

func (b *badWriter) Write(p []byte) (int, error) { return 0, errors.New("write fail") }
func (b *badWriter) Close() error                { return nil }

// helper reader that always fails on Read
type errReader struct{}

func (e *errReader) Read(p []byte) (int, error) { return 0, errors.New("read fail") }
func (e *errReader) Close() error               { return nil }

// wrapper that uses fakeSFTP but overrides Create
type createFail struct{ *fakeSFTP }

func (c *createFail) Create(path string) (io.WriteCloser, error) { return &badWriter{}, nil }

// wrapper that uses fakeSFTP but overrides Open
type openErr struct{ *fakeSFTP }

func (o *openErr) Open(path string) (io.ReadCloser, error) { return &errReader{}, nil }

func TestRemoveTempKeyFromRemoteHost_OpenFail(t *testing.T) {
	origSshDial := sshDialFunc
	origSftpNew := sftpNewClient
	defer func() { sshDialFunc = origSshDial; sftpNewClient = origSftpNew }()

	tk, err := generateTemporaryKeyPair()
	if err != nil {
		t.Fatalf("generateTemporaryKeyPair: %v", err)
	}
	sess := &BootstrapSession{PendingAccount: model.Account{Username: "u", Hostname: "h"}, TempKeyPair: tk}

	sshDialFunc = func(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error) { return &ssh.Client{}, nil }
	// Return a fake SFTP whose Open fails
	sftpNewClient = func(conn *ssh.Client, opts ...sftp.ClientOption) (sftpClientIface, error) {
		return &fakeSFTP{files: nil}, nil
	}

	if err := removeTempKeyFromRemoteHost(sess); err == nil {
		t.Fatalf("expected error when Open fails")
	}
}

func TestRemoveTempKeyFromRemoteHost_CreateWriteFail(t *testing.T) {
	origSshDial := sshDialFunc
	origSftpNew := sftpNewClient
	defer func() { sshDialFunc = origSshDial; sftpNewClient = origSftpNew }()

	tk, err := generateTemporaryKeyPair()
	if err != nil {
		t.Fatalf("generateTemporaryKeyPair: %v", err)
	}
	sess := &BootstrapSession{PendingAccount: model.Account{Username: "u", Hostname: "h"}, TempKeyPair: tk}

	fake := &fakeSFTP{files: map[string][]byte{".ssh/authorized_keys": []byte(tk.GetPublicKey() + "\nother\n")}}
	sftpNewClient = func(conn *ssh.Client, opts ...sftp.ClientOption) (sftpClientIface, error) {
		return &createFail{fake}, nil
	}

	sshDialFunc = func(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error) { return &ssh.Client{}, nil }

	if err := removeTempKeyFromRemoteHost(sess); err == nil {
		t.Fatalf("expected error when Create/Write fails")
	}
}

func TestRemoveTempKeyFromRemoteHost_ReadError(t *testing.T) {
	origSshDial := sshDialFunc
	origSftpNew := sftpNewClient
	defer func() { sshDialFunc = origSshDial; sftpNewClient = origSftpNew }()

	tk, err := generateTemporaryKeyPair()
	if err != nil {
		t.Fatalf("generateTemporaryKeyPair: %v", err)
	}
	sess := &BootstrapSession{PendingAccount: model.Account{Username: "u", Hostname: "h"}, TempKeyPair: tk}

	fake := &fakeSFTP{files: make(map[string][]byte)}
	sftpNewClient = func(conn *ssh.Client, opts ...sftp.ClientOption) (sftpClientIface, error) {
		return &openErr{fake}, nil
	}

	sshDialFunc = func(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error) { return &ssh.Client{}, nil }

	if err := removeTempKeyFromRemoteHost(sess); err == nil {
		t.Fatalf("expected error when Read fails")
	}
}
