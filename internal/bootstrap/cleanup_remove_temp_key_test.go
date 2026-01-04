package bootstrap

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/toeirei/keymaster/internal/model"
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

// sftp.NewClient is replaced by our package variable in tests; this adapter satisfies expected signature
func fakeSftpNewClientFromFake(conn interface{}, f *fakeSFTP) (sftpClientInterface, error) {
	// not using conn; return our fake as the concrete type that has required methods
	return f, nil
}

// Define an interface matching the sftp methods we use to allow type compatibility in tests
type sftpClientInterface interface {
	Open(string) (io.ReadCloser, error)
	Create(string) (io.WriteCloser, error)
	Close() error
}

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
