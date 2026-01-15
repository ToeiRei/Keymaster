package tui

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriteKeyFilePermissions(t *testing.T) {
	dir := t.TempDir()
	fname := filepath.Join(dir, "authorized_keys_test")
	content := []byte("ssh-ed25519 AAAAB3Nza testkey")

	if err := WriteKeyFile(fname, content); err != nil {
		t.Fatalf("WriteKeyFile failed: %v", err)
	}

	// Verify content
	b, err := os.ReadFile(fname)
	if err != nil {
		t.Fatalf("read back failed: %v", err)
	}
	if string(b) != string(content) {
		t.Fatalf("content mismatch")
	}

	// On non-Windows, ensure file perms are 0600
	if runtime.GOOS != "windows" {
		fi, err := os.Stat(fname)
		if err != nil {
			t.Fatalf("stat failed: %v", err)
		}
		perm := fi.Mode().Perm()
		if perm != 0600 {
			t.Fatalf("unexpected file mode: %v (want 0600)", perm)
		}
	} else {
		t.Log("Windows: skipping file mode assertions")
	}
}
