package db

import (
	"strings"

	"github.com/toeirei/keymaster/internal/model"
)

// FilterAccountsByTokens returns the subset of `accounts` that match all tokens.
// Matching is case-insensitive and tests username, hostname, and label for
// substring containment. If `tokens` is nil or empty, the original slice is returned.
func FilterAccountsByTokens(accounts []model.Account, tokens []string) []model.Account {
	if len(tokens) == 0 {
		return accounts
	}
	out := make([]model.Account, 0, len(accounts))
	for _, a := range accounts {
		// prepare lowercase fields
		lbl := strings.ToLower(a.Label)
		user := strings.ToLower(a.Username)
		host := strings.ToLower(a.Hostname)

		matchedAll := true
		for _, tok := range tokens {
			tok = strings.ToLower(strings.TrimSpace(tok))
			if tok == "" {
				continue
			}
			if !strings.Contains(user, tok) && !strings.Contains(host, tok) && !strings.Contains(lbl, tok) {
				matchedAll = false
				break
			}
		}
		if matchedAll {
			out = append(out, a)
		}
	}
	return out
}
