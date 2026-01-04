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
//   - For fast unit tests that don't need a DB, inject `FakeKeyManager` or
//     `FakeAccountManager` via `SetDefaultKeyManager` / `SetDefaultAccountManager`.
package db
