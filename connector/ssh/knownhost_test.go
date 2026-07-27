// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/toeirei/keymaster/connector"
	"golang.org/x/crypto/ssh"
)

// fakeUserRequester answers every choice with choice and records what it was
// asked, so tests can assert on the wording the connector offered.
type fakeUserRequester struct {
	choice int
	asked  [][]fmt.Stringer
}

func (f *fakeUserRequester) RequestText(fmt.Stringer) string { return "" }

func (f *fakeUserRequester) RequestChoice(prompts []fmt.Stringer) int {
	f.asked = append(f.asked, prompts)
	return f.choice
}

func testHostKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate host key: %v", err)
	}
	key, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		t.Fatalf("ssh.NewPublicKey: %v", err)
	}
	return key
}

// stubHostKeyProbe makes the probe return key (or err) instead of dialing.
func stubHostKeyProbe(t *testing.T, key ssh.PublicKey, err error) {
	t.Helper()
	old := hostKeyProbe
	t.Cleanup(func() { hostKeyProbe = old })
	hostKeyProbe = func(string, time.Duration) (ssh.PublicKey, error) { return key, err }
}

func TestResolveKnownHost_MatchNeedsNoPrompt(t *testing.T) {
	key := testHostKey(t)
	stubHostKeyProbe(t, key, nil)

	requester := &fakeUserRequester{choice: hostKeyTrust}
	// A key stored with the trailing newline MarshalAuthorizedKey appends still
	// counts as a match.
	stored := marshalKnownHost(key) + "\n"

	hostKey, knownHost, err := resolveKnownHost("host.example:22", stored, requester, time.Second)
	if err != nil {
		t.Fatalf("resolveKnownHost: %v", err)
	}
	if knownHost != stored {
		t.Fatalf("expected the stored known host to be returned untouched, got %q", knownHost)
	}
	if marshalKnownHost(hostKey) != marshalKnownHost(key) {
		t.Fatalf("expected the presented key to be pinned for the connection")
	}
	if len(requester.asked) != 0 {
		t.Fatalf("expected no user request for a matching host key, got %d", len(requester.asked))
	}
}

func TestResolveKnownHost_UserChoices(t *testing.T) {
	key := testHostKey(t)
	presented := marshalKnownHost(key)
	other := marshalKnownHost(testHostKey(t))

	for _, tc := range []struct {
		name       string
		stored     string
		choice     int
		want       string
		wantPrompt string
	}{
		{"unknown allow once", "", hostKeyAllowOnce, "", "is not known yet"},
		{"unknown trust", "", hostKeyTrust, presented, "is not known yet"},
		{"mismatch allow once", other, hostKeyAllowOnce, other, "CHANGED"},
		{"mismatch trust", other, hostKeyTrust, presented, "CHANGED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stubHostKeyProbe(t, key, nil)
			requester := &fakeUserRequester{choice: tc.choice}

			hostKey, knownHost, err := resolveKnownHost("host.example:22", tc.stored, requester, time.Second)
			if err != nil {
				t.Fatalf("resolveKnownHost: %v", err)
			}
			if knownHost != tc.want {
				t.Fatalf("known host = %q, want %q", knownHost, tc.want)
			}
			// Allowing once leaves the cache alone but still has to let the
			// connection through with the key the user just approved.
			if marshalKnownHost(hostKey) != presented {
				t.Fatalf("pinned key = %q, want the presented %q", marshalKnownHost(hostKey), presented)
			}
			if len(requester.asked) != 1 {
				t.Fatalf("expected exactly one user request, got %d", len(requester.asked))
			}
			choices := requester.asked[0]
			if len(choices) != 2 {
				t.Fatalf("expected an allow-once and a trust choice, got %d", len(choices))
			}
			for _, choice := range choices {
				if !strings.Contains(choice.String(), tc.wantPrompt) {
					t.Errorf("choice %q does not describe the situation (%q)", choice, tc.wantPrompt)
				}
				if !strings.Contains(choice.String(), ssh.FingerprintSHA256(key)) {
					t.Errorf("choice %q does not show the presented fingerprint", choice)
				}
			}
		})
	}
}

func TestResolveKnownHost_AbortsWithoutConsent(t *testing.T) {
	key := testHostKey(t)
	other := marshalKnownHost(testHostKey(t))

	for _, tc := range []struct {
		name      string
		stored    string
		requester connector.UserRequester
	}{
		{"cancelled first contact", "", &fakeUserRequester{choice: -1}},
		{"cancelled mismatch", other, &fakeUserRequester{choice: -1}},
		{"unattended first contact", "", nil},
		{"unattended mismatch", other, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stubHostKeyProbe(t, key, nil)

			hostKey, knownHost, err := resolveKnownHost("host.example:22", tc.stored, tc.requester, time.Second)
			if err == nil {
				t.Fatalf("expected an error, got known host %q", knownHost)
			}
			if knownHost != "" {
				t.Fatalf("expected no known host on abort, got %q", knownHost)
			}
			if hostKey != nil {
				t.Fatalf("expected no pinned host key on abort")
			}
		})
	}
}

func TestResolveKnownHost_ProbeFailurePropagates(t *testing.T) {
	stubHostKeyProbe(t, nil, errors.New("dial tcp: connection refused"))

	requester := &fakeUserRequester{choice: hostKeyTrust}
	if _, _, err := resolveKnownHost("host.example:22", "", requester, time.Second); err == nil {
		t.Fatal("expected the probe failure to propagate")
	}
	if len(requester.asked) != 0 {
		t.Fatal("expected no user request when the host key could not be read")
	}
}
