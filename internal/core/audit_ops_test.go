// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import "testing"

func TestAuditActionRisk(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"DELETE_ACCOUNT_1", "high"},
		{"ADD_ACCOUNT", "low"},
		{"ASSIGN_KEY", "medium"},
		{"SOME_OTHER_ACTION", "info"},
	}

	for _, c := range cases {
		got := AuditActionRisk(c.in)
		if got != c.want {
			t.Fatalf("AuditActionRisk(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

