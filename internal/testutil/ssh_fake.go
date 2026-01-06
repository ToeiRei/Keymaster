// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package testutil

// FakeSSHClient is a lightweight test double implementing the minimal
// Close() behavior used by the deploy package. Tests can use it to simulate
// a client without constructing a real *ssh.Client.
type FakeSSHClient struct {
	// CloseFunc, if set, is called when Close() is invoked.
	CloseFunc func() error
	closed    bool
}

// Close marks the fake client as closed and calls CloseFunc if provided.
func (f *FakeSSHClient) Close() error {
	if f == nil {
		return nil
	}
	f.closed = true
	if f.CloseFunc != nil {
		return f.CloseFunc()
	}
	return nil
}

// NewFakeSSHClient returns a ready-to-use FakeSSHClient pointer.
func NewFakeSSHClient() *FakeSSHClient {
	return &FakeSSHClient{}
}
