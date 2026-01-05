// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import "github.com/toeirei/keymaster/internal/model"

// AccountSearcherFunc is a lightweight adapter type so core doesn't need to
// import UI searcher interfaces. Callers may pass a function that performs
// server-side searching and returns ([]model.Account, error).
type AccountSearcherFunc func(query string) ([]model.Account, error)

// FilterAccounts filters accounts by the provided query. If searcher is
// non-nil it will be preferred when it returns a non-empty result with no
// error; otherwise a local in-memory fallback is used.
func FilterAccounts(accounts []model.Account, query string, searcher AccountSearcherFunc) []model.Account {
	if query == "" {
		return accounts
	}

	// Local in-memory filtering
	localResults := make([]model.Account, 0, len(accounts))
	for _, acc := range accounts {
		combined := acc.Username + " " + acc.Hostname + " " + acc.Label + " " + acc.Tags
		if ContainsIgnoreCase(combined, query) {
			localResults = append(localResults, acc)
		}
	}

	if searcher != nil {
		if res, err := searcher(query); err == nil && len(res) > 0 {
			return res
		}
	}

	return localResults
}

// FilterKeys filters public keys by simple criteria (comment and algorithm).
func FilterKeys(keys []model.PublicKey, query string) []model.PublicKey {
	if query == "" {
		return keys
	}
	var out []model.PublicKey
	for _, k := range keys {
		if ContainsIgnoreCase(k.Comment, query) || ContainsIgnoreCase(k.Algorithm, query) {
			out = append(out, k)
		}
	}
	return out
}
