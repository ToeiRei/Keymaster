// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// Package db contains the data-access layer and small DI helpers used by
// Keymaster.
//
// This package exposes a small set of lightweight interfaces and package-level
// helpers that make it easy to inject fakes for tests while preserving a
// centralized Bun-based implementation for production.
//
// DI helpers
//   - `Default*` functions return a sensible default implementation when the
//     package-level `store` has been initialized (via `InitDB`) or when a
//     package-level override has been set by tests.
//   - `SetDefault*` and `ClearDefault*` functions allow tests to inject simple
//     fakes that implement the same small interface (`AccountManager`,
//     `KeyManager`, `AccountSearcher`, `KeySearcher`, `AuditWriter`, etc.).
//
// KeyManager guidance
//   - The `KeyManager` interface centralizes public-key operations including
//     key<->account assignments. Prefer using `DefaultKeyManager()` in callers
//     (or inject a `KeyManager`) instead of calling low-level `Store` methods
//     directly. This makes code simpler to test and decouples UI/CLI code from
//     store implementation details.
//   - Low-level Bun helpers (used for SQL queries) live in `bun_adapter.go`.
//     The `KeyManager` adapter calls those helpers and is responsible for
//     higher-level concerns such as audit logging.
//
// Migration & deprecation
//   - When migrating callers, update them to use `DefaultKeyManager()` and add
//     nil checks where appropriate. Tests should use `searcher_fake.go`'s
//     `FakeKeyManager` to inject deterministic behavior.
//   - After migrating all call sites, the legacy assignment methods on the
//     `Store` interface may be removed. Keep the Bun helpers in
//     `bun_adapter.go` for low-level access; these helpers are not public API
//     and are intended to be used by the `KeyManager` adapter.
//
// Testing notes
//   - Prefer `db.InitDB("sqlite", ":memory:")` in tests that need real DB
//     semantics and migrations.
//   - For fast unit tests that don't need a DB, inject `testutil.FakeKeyManager` or
//     `testutil.FakeAccountManager` (from `internal/testutil`) via
//     `SetDefaultKeyManager` / `SetDefaultAccountManager`.
package db

// DI contract (concise)
// - Callers should depend on the smallest practical interface (for example
//   use `AccountManager` instead of `Store` when only adding/deleting
//   accounts). Use `Default*` helpers to obtain a package-default backed by
//   the global `store` when available, or inject a fake via `SetDefault*`
//   in tests.
// - `SetDefault*` / `ClearDefault*` are primarily for tests. They allow a
//   test to install a deterministic fake (see `internal/testutil`) without
//   initializing a real DB.
// - `KeyManager` centralizes public-key CRUD and assignment behavior. It
//   should be the primary surface used by UI/CLI code for key operations.
// - Low-level Bun helpers remain in `bun_adapter.go`. They are implementation
//   details and intended to be called by adapters (e.g., `bunKeyManager`),
//   not by high-level code.
