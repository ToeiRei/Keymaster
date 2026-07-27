// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package mock

import (
	"context"
	"testing"

	"github.com/toeirei/keymaster/connector"
)

func testDeployment(succeed bool) connector.Deployment {
	return connector.Deployment{
		&secret{succeed, "primary"},
		[]connector.DeployRecord{{
			Algorithm: "ssh-ed25519",
			Data:      "AAAAC3NzaC1lZDI1NTE5AAAAIexample",
			Comment:   "alice@example",
		}},
	}
}

// open dials with the deployment's own secret, the way an ordinary deploy does.
func open(t *testing.T, c *Connector, deployment connector.Deployment) (connector.Connection, error) {
	t.Helper()
	return c.OpenConnection(context.Background(), deployment.Secret, nil, "alice", "host.example", 22, nil)
}

// drain runs op with a progress channel the way a client does, so a connector
// that forgets to send progress cannot deadlock the test.
func drain(op func(progress chan<- connector.Progress)) {
	progress := make(chan connector.Progress)
	go func() {
		defer close(progress)
		op(progress)
	}()
	for range progress {
	}
}

func TestOpenConnection_SecretSucceed_Succeeds(t *testing.T) {
	c := &Connector{}

	conn, err := open(t, c, testDeployment(true))
	if err != nil {
		t.Fatalf("OpenConnection returned error: %v", err)
	}
	if conn == nil {
		t.Fatal("expected OpenConnection to return a connection")
	}
}

func TestOpenConnection_SecretNotSucceed_Fails(t *testing.T) {
	c := &Connector{}

	if _, err := open(t, c, testDeployment(false)); err == nil {
		t.Fatal("expected OpenConnection to fail when the secret does not succeed")
	}
}

func TestVerifyOffline(t *testing.T) {
	c := &Connector{}

	deployment := testDeployment(true)
	if ok, err := c.VerifyOffline(context.Background(), nil, deployment); err != nil || ok {
		t.Fatalf("expected VerifyOffline to be false with no cache, got ok=%v err=%v", ok, err)
	}

	hash, err := c.hash(deployment)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if ok, err := c.VerifyOffline(context.Background(), &cache{hash}, deployment); err != nil || !ok {
		t.Fatalf("expected VerifyOffline to match cache, got ok=%v err=%v", ok, err)
	}

	if ok, err := c.VerifyOffline(context.Background(), &cache{"stale"}, deployment); err != nil || ok {
		t.Fatalf("expected VerifyOffline to report mismatch, got ok=%v err=%v", ok, err)
	}
}

// The secret is part of the hash because a real connector installs its public
// half on the target, so a secret update has to look like a change.
func TestHash_ChangesWithTheSecret(t *testing.T) {
	c := &Connector{}

	deployment := testDeployment(true)
	before, err := c.hash(deployment)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	deployment.Secret = &secret{true, "rotated"}
	after, err := c.hash(deployment)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	if before == after {
		t.Fatal("expected the hash to change when the deployment is keyed to another secret")
	}
}
