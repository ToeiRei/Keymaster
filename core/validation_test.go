// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package core

import "testing"

func TestValidateBootstrapParams(t *testing.T) {
	cases := []struct {
		name     string
		username string
		hostname string
		wantErr  bool
	}{
		{name: "valid", username: "alice", hostname: "example.com", wantErr: false},
		{name: "empty username", username: "", hostname: "host", wantErr: true},
		{name: "empty hostname", username: "u", hostname: "", wantErr: true},
		{name: "whitespace trimmed", username: "  bob  ", hostname: "  host  ", wantErr: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateBootstrapParams(tc.username, tc.hostname, "", "")
			if (err != nil) != tc.wantErr {
				t.Fatalf("ValidateBootstrapParams() error = %v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}
