package ui

import "github.com/toeirei/keymaster/internal/model"

// AccountSearcher provides a small abstraction for searching accounts.
type AccountSearcher interface {
	SearchAccounts(q string) ([]model.Account, error)
}
