// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package ui

import (
	"reflect"
	"testing"
)

func TestSuggestTags(t *testing.T) {
	tests := []struct {
		name    string
		allTags []string
		current string
		want    []string
	}{
		{"prefix match", []string{"vault", "dev", "prod", "staging"}, "v", []string{"vault"}},
		{"exclude existing", []string{"vault", "dev", "prod"}, "dev, v", []string{"vault"}},
		{"empty input", []string{"vault", "dev"}, "", nil},
		{"trailing comma", []string{"vault", "dev"}, "dev, ", nil},
		{"case preserve", []string{"Vault", "DEV", "Prod"}, "v", []string{"Vault"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SuggestTags(tc.allTags, tc.current)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("SuggestTags(%v, %q) = %v; want %v", tc.allTags, tc.current, got, tc.want)
			}
		})
	}
}

