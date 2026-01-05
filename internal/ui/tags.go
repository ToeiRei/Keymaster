// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package ui

import (
	"strings"

	"github.com/toeirei/keymaster/internal/db"
)

// TagSuggester provides access to known tags and suggestion helpers.
type TagSuggester interface {
	AllTags() ([]string, error)
	Suggest(currentVal string) []string
}

type dbTagSuggester struct{}

func (d *dbTagSuggester) AllTags() ([]string, error) {
	allAccounts, err := db.GetAllAccounts()
	if err != nil {
		return nil, err
	}
	tagSet := make(map[string]struct{})
	for _, acc := range allAccounts {
		if acc.Tags == "" {
			continue
		}
		for _, tag := range strings.Split(acc.Tags, ",") {
			t := strings.TrimSpace(tag)
			if t != "" {
				tagSet[t] = struct{}{}
			}
		}
	}
	tags := make([]string, 0, len(tagSet))
	for t := range tagSet {
		tags = append(tags, t)
	}
	return tags, nil
}

func (d *dbTagSuggester) Suggest(currentVal string) []string {
	tags, err := d.AllTags()
	if err != nil {
		return nil
	}
	return SuggestTags(tags, currentVal)
}

// DefaultTagSuggester returns a TagSuggester backed by the database.
func DefaultTagSuggester() TagSuggester {
	return &dbTagSuggester{}
}

