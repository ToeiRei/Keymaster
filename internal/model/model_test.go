// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package model

import "testing"

func TestAccountString(t *testing.T) {
    a := Account{Username: "deploy", Hostname: "web-01"}
    if got := a.String(); got != "deploy@web-01" {
        t.Errorf("unexpected Account.String(): %q", got)
    }

    a.Label = "prod-web"
    if got := a.String(); got != "prod-web (deploy@web-01)" {
        t.Errorf("unexpected Account.String() with label: %q", got)
    }
}

func TestPublicKeyString(t *testing.T) {
    k := PublicKey{Algorithm: "ssh-ed25519", KeyData: "AAAAB3NzaC1lZDI1NTE5", Comment: "me@example.com"}
    want := "ssh-ed25519 AAAAB3NzaC1lZDI1NTE5 me@example.com"
    if got := k.String(); got != want {
        t.Errorf("unexpected PublicKey.String(): got %q want %q", got, want)
    }
}
