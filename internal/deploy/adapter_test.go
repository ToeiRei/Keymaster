// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy

import (
	"io"
	"testing"
)

func TestSftpClientAdapter_Delegates(t *testing.T) {
	mock := newMockSftpClient()
	adapter := &sftpClientAdapter{client: mock}

	// Create
	f, err := adapter.Create(".ssh/tmp")
	if err != nil || f == nil {
		t.Fatalf("Create failed: %v", err)
	}
	// Write and close
	if _, err := f.Write([]byte("x")); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	_ = f.Close()

	// Stat (non-existent should return os.ErrNotExist) -- ignore result
	_, _ = adapter.Stat(".ssh/tmp")

	// Mkdir
	if err := adapter.Mkdir(".ssh"); err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	// Chmod
	if err := adapter.Chmod(".ssh", 0700); err != nil {
		t.Fatalf("Chmod failed: %v", err)
	}

	// Rename
	if err := adapter.Rename(".ssh/tmp", ".ssh/f"); err != nil {
		t.Fatalf("Rename failed: %v", err)
	}

	// Open
	of, err := adapter.Open(".ssh/f")
	if err != nil && of == nil {
		// may be not exist; that's acceptable for delegation test
	} else if of != nil {
		_, _ = io.ReadAll(of)
		_ = of.Close()
	}

	// Remove
	_ = adapter.Remove(".ssh/f")

	// Close
	if err := adapter.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}
