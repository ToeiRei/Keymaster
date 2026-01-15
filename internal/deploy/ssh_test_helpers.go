// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy

// FakeSSHClient is a lightweight test double implementing the minimal
// `sshClientIface` used by this package. Tests can use it to simulate a
// client without constructing a real *ssh.Client.
type FakeSSHClient struct {
	// CloseFunc, if set, is called when Close() is invoked.
	CloseFunc func() error
	closed    bool
}

// Ensure FakeSSHClient implements sshClientIface
var _ sshClientIface = (*FakeSSHClient)(nil)

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

// NewFakeSSHClient returns a ready-to-use FakeSSHClient as sshClientIface.
func NewFakeSSHClient() sshClientIface {
	return &FakeSSHClient{}
}
