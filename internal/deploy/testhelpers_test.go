// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy_test

import (
	"bytes"

	"github.com/toeirei/keymaster/internal/testutil"
)

// fakeDeployer implements core.RemoteDeployer for tests and is shared
// so multiple test files can override `core.NewDeployerFactory` without
// redeclaring the type.
type fakeDeployer struct {
	content []byte
	seen    string
}

func (f *fakeDeployer) DeployAuthorizedKeys(content string) error { f.seen = content; return nil }
func (f *fakeDeployer) GetAuthorizedKeys() ([]byte, error)        { return f.content, nil }
func (f *fakeDeployer) Close()                                    {}

// bytesFromString is a small helper used by tests to create buffers.
func bytesFromString(s string) *bytes.Buffer { return testutil.BytesFromString(s) }

