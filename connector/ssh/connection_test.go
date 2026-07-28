// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package ssh

import (
	"context"
	"errors"
	"testing"

	"github.com/toeirei/keymaster/connector"
	"github.com/toeirei/keymaster/ui/i18n"
)

// openTestConnection opens against a stubbed deployer with the host key already
// pinned in the cache, so nothing prompts.
func openTestConnection(t *testing.T, c *Connector, connectorSecret *secret, deployer deployerClient) connector.Connection {
	t.Helper()
	hostKey := testHostKey(t)
	stubHostKeyProbe(t, hostKey, nil)
	stubDeployer(t, deployer)

	conn, err := c.OpenConnection(context.Background(), connectorSecret, &cache{KnownHost: marshalKnownHost(hostKey)}, "alice", "host.example", 22, nil)
	if err != nil {
		t.Fatalf("OpenConnection: %v", err)
	}
	return conn
}

func TestConnectionDeploy_WritesRenderedAuthorizedKeysToRemote(t *testing.T) {
	c := &Connector{}
	connectorSecret := testSecret(t)
	records := testRecords()
	deployer := &fakeSSHDeployer{}

	conn := openTestConnection(t, c, connectorSecret, deployer)

	var (
		deployErr error
		sawDone   bool
	)
	progress := make(chan connector.Progress)
	go func() {
		defer close(progress)
		deployErr = conn.Deploy(context.Background(), connector.Deployment{connectorSecret, records}, progress)
	}()
	for progressUpdate := range progress {
		if progressUpdate.Status.String() == i18n.T("connector.status.done") {
			sawDone = true
		}
	}

	if deployErr != nil {
		t.Fatalf("Deploy returned error: %v", deployErr)
	}
	if !sawDone {
		t.Fatal("expected final progress update after deploy")
	}

	want := c.makeAuthorizedKeys(connectorSecret.PublicKey, records)
	if deployer.deployed != want {
		t.Fatalf("unexpected authorized_keys payload\n got: %q\nwant: %q", deployer.deployed, want)
	}

	newCache := conn.Close().(*cache)
	if newCache.AuthorizedKeysHash != c.hashAuthorizedKeys(want) {
		t.Fatalf("expected Close to report the deployed hash, got %q", newCache.AuthorizedKeysHash)
	}
	if newCache.KnownHost == "" {
		t.Fatal("expected Close to carry the known host forward")
	}
	if !deployer.closed {
		t.Fatal("expected the underlying deployer to be closed")
	}
}

// A deployment keyed to another secret installs that secret's public half, which
// is what makes a secret update an ordinary deploy over the old credential.
func TestConnectionDeploy_KeyedToAnotherSecret(t *testing.T) {
	c := &Connector{}
	oldSecret := testSecret(t)
	newSecret := testSecret(t)
	records := testRecords()
	deployer := &fakeSSHDeployer{}

	// Authenticated with the old secret ...
	conn := openTestConnection(t, c, oldSecret, deployer)

	// ... but writing the file the new secret belongs in.
	var deployErr error
	drain(func(progress chan<- connector.Progress) {
		deployErr = conn.Deploy(context.Background(), connector.Deployment{newSecret, records}, progress)
	})
	if deployErr != nil {
		t.Fatalf("Deploy returned error: %v", deployErr)
	}

	if want := c.makeAuthorizedKeys(newSecret.PublicKey, records); deployer.deployed != want {
		t.Fatalf("expected the new secret's public key to be installed\n got: %q\nwant: %q", deployer.deployed, want)
	}
	if got := conn.Close().(*cache).AuthorizedKeysHash; got != c.hashAuthorizedKeys(c.makeAuthorizedKeys(newSecret.PublicKey, records)) {
		t.Fatalf("expected the cache to record the new secret's rendering, got %q", got)
	}
}

func TestConnectionVerify_ComparesAgainstTheRemote(t *testing.T) {
	c := &Connector{}
	connectorSecret := testSecret(t)
	records := testRecords()
	deployment := connector.Deployment{connectorSecret, records}

	expected := c.makeAuthorizedKeys(connectorSecret.PublicKey, records)

	for name, tc := range map[string]struct {
		remote string
		wantOk bool
	}{
		"match": {expected, true},
		"drift": {expected + "\nssh-ed25519 AAAAsomethingelse hacker@example\n", false},
	} {
		t.Run(name, func(t *testing.T) {
			deployer := &fakeSSHDeployer{remote: []byte(tc.remote)}
			conn := openTestConnection(t, c, connectorSecret, deployer)

			var (
				ok        bool
				verifyErr error
			)
			drain(func(progress chan<- connector.Progress) {
				ok, verifyErr = conn.Verify(context.Background(), deployment, progress)
			})
			if verifyErr != nil {
				t.Fatalf("Verify returned error: %v", verifyErr)
			}
			if ok != tc.wantOk {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOk)
			}

			// The cache records what the target actually holds, so drift stays visible.
			if got := conn.Close().(*cache).AuthorizedKeysHash; got != c.hashAuthorizedKeys(tc.remote) {
				t.Fatalf("expected the cache to hold the remote's hash, got %q", got)
			}
		})
	}
}

// A failed operation must not cost the caller the known host that was resolved
// while opening, which is the whole reason Close returns the cache.
func TestConnectionClose_KeepsTheKnownHostAfterAFailedDeploy(t *testing.T) {
	c := &Connector{}
	connectorSecret := testSecret(t)
	hostKey := testHostKey(t)

	stubHostKeyProbe(t, hostKey, nil)
	stubDeployer(t, &fakeSSHDeployer{deployErr: errors.New("upload refused")})

	conn, err := c.OpenConnection(context.Background(), connectorSecret, nil, "alice", "host.example", 22, &fakeUserRequester{choice: hostKeyTrust})
	if err != nil {
		t.Fatalf("OpenConnection: %v", err)
	}

	var deployErr error
	drain(func(progress chan<- connector.Progress) {
		deployErr = conn.Deploy(context.Background(), connector.Deployment{connectorSecret, testRecords()}, progress)
	})
	if deployErr == nil {
		t.Fatal("expected Deploy to report the upload failure")
	}

	newCache := conn.Close().(*cache)
	if got := newCache.KnownHost; got != marshalKnownHost(hostKey) {
		t.Fatalf("known host = %q, want the newly trusted %q", got, marshalKnownHost(hostKey))
	}
	if newCache.AuthorizedKeysHash != "" {
		t.Fatalf("expected no deployed hash after a failed deploy, got %q", newCache.AuthorizedKeysHash)
	}
}

func TestConnectionClose_IsRepeatableAndBlocksFurtherOperations(t *testing.T) {
	c := &Connector{}
	connectorSecret := testSecret(t)
	conn := openTestConnection(t, c, connectorSecret, &fakeSSHDeployer{})

	first := conn.Close().(*cache)
	second := conn.Close().(*cache)
	if *first != *second {
		t.Fatalf("second Close returned %+v, want %+v", second, first)
	}

	var deployErr error
	drain(func(progress chan<- connector.Progress) {
		deployErr = conn.Deploy(context.Background(), connector.Deployment{connectorSecret, testRecords()}, progress)
	})
	if deployErr == nil {
		t.Fatal("expected Deploy on a closed connection to fail")
	}
}
