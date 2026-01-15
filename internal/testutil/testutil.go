// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package testutil

import "bytes"

// FakeRemoteDeployer is a simple in-memory deployer used by tests to avoid
// real network operations.
type FakeRemoteDeployer struct {
	content []byte
	seen    string
}

func (f *FakeRemoteDeployer) DeployAuthorizedKeys(content string) error { f.seen = content; return nil }
func (f *FakeRemoteDeployer) GetAuthorizedKeys() ([]byte, error)        { return f.content, nil }
func (f *FakeRemoteDeployer) Close()                                    {}

// BytesFromString returns a buffer containing the provided string.
func BytesFromString(s string) *bytes.Buffer { return bytes.NewBufferString(s) }

// (No audit writer here — use testutil.FakeAuditWriter or db.WithAuditWriter helpers.)
