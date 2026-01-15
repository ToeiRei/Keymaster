// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"sort"
	"strings"

	"github.com/toeirei/keymaster/internal/model"
)

const untaggedLabel = "(no tags)"

// BuildAccountsByTag groups accounts by tag. Accounts with no tags are
// grouped under the special key "(no tags)". Tag strings are trimmed.
func BuildAccountsByTag(accounts []model.Account) map[string][]model.Account {
	m := make(map[string][]model.Account)
	for _, acc := range accounts {
		if strings.TrimSpace(acc.Tags) == "" {
			m[untaggedLabel] = append(m[untaggedLabel], acc)
			continue
		}
		for _, t := range strings.Split(acc.Tags, ",") {
			tag := strings.TrimSpace(t)
			if tag == "" {
				continue
			}
			m[tag] = append(m[tag], acc)
		}
	}
	return m
}

// UniqueTags returns a sorted slice of unique tags present in the provided
// accounts. If there are accounts without tags, the special "(no tags)"
// value will be appended to the end of the slice.
func UniqueTags(accounts []model.Account) []string {
	set := make(map[string]struct{})
	hasUntagged := false
	for _, acc := range accounts {
		if strings.TrimSpace(acc.Tags) == "" {
			hasUntagged = true
			continue
		}
		for _, t := range strings.Split(acc.Tags, ",") {
			tag := strings.TrimSpace(t)
			if tag == "" {
				continue
			}
			set[tag] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for tag := range set {
		out = append(out, tag)
	}
	sort.Strings(out)
	if hasUntagged {
		out = append(out, untaggedLabel)
	}
	return out
}

