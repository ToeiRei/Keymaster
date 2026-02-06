// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

// This file provides aliases to test helper implementations living in
// internal/testutil so tests can continue to refer to `db.Fake*` types while
// keeping the canonical implementations centralized in `internal/testutil`.

import (
	testutil "github.com/toeirei/keymaster/testutil"
)

type FakeAccountSearcher = testutil.FakeAccountSearcher
type FakeKeySearcher = testutil.FakeKeySearcher
type FakeAuditSearcher = testutil.FakeAuditSearcher
type FakeAuditWriter = testutil.FakeAuditWriter
type FakeAccountManager = testutil.FakeAccountManager
type FakeKeyManager = testutil.FakeKeyManager
