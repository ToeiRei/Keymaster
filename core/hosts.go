// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package core

import (
	"sort"
	"strings"

	"github.com/toeirei/keymaster/core/model"
)

const unknownHostLabel = "(no hostname)"

// BuildAccountsByHost groups accounts by hostname. Accounts with no hostname are
// grouped under the special key "(no hostname)".
func BuildAccountsByHost(accounts []model.Account) map[string][]model.Account {
	m := make(map[string][]model.Account)
	for _, acc := range accounts {
		hostname := strings.TrimSpace(acc.Hostname)
		if hostname == "" {
			m[unknownHostLabel] = append(m[unknownHostLabel], acc)
		} else {
			m[hostname] = append(m[hostname], acc)
		}
	}
	return m
}

// UniqueHosts returns a sorted slice of unique hostnames present in the provided
// accounts. If there are accounts without a hostname, the special "(no hostname)"
// value will be appended to the end of the slice.
func UniqueHosts(accounts []model.Account) []string {
	set := make(map[string]struct{})
	hasUnknown := false
	for _, acc := range accounts {
		hostname := strings.TrimSpace(acc.Hostname)
		if hostname == "" {
			hasUnknown = true
		} else {
			set[hostname] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for host := range set {
		out = append(out, host)
	}
	sort.Strings(out)
	if hasUnknown {
		out = append(out, unknownHostLabel)
	}
	return out
}
