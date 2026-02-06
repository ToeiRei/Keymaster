// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package deploy

import (
	"testing"
)

func TestDeployer_Close_InvokesSftpClose(t *testing.T) {
	mock := newMockSftpClient()
	d := &Deployer{sftp: mock}

	d.Close()

	// Ensure the mock recorded a close call
	found := false
	for _, a := range mock.actions {
		if a == "close" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected sftp Close to be called, actions: %v", mock.actions)
	}
}

func TestGetAuthorizedKeys_OpenError(t *testing.T) {
	mock := newMockSftpClient()
	// Ensure file does not exist; Open will return os.ErrNotExist
	d := &Deployer{sftp: mock}

	if _, err := d.GetAuthorizedKeys(); err == nil {
		t.Fatalf("expected GetAuthorizedKeys to fail when file missing")
	}
}
