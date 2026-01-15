// Adapters to `internal/core` were previously placed here but caused an import
// cycle while `internal/core` still imports parts of `internal/db`.
//
// To avoid import cycles, adapters that bridge `internal/db` -> `internal/core`
// live in `internal/ui` (which already imports both packages). Keep this file
// as a placeholder to document that adapters live elsewhere.

package db
