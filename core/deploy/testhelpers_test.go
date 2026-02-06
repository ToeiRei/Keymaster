// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy_test

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

// Use `testutil.BytesFromString` directly in tests; helper removed.
