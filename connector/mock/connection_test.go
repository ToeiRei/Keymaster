// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package mock

import (
	"context"
	"testing"

	"github.com/toeirei/keymaster/connector"
)

func TestConnectionDeploy_AdvancesTheCache(t *testing.T) {
	c := &Connector{}
	deployment := testDeployment(true)

	conn, err := open(t, c, deployment)
	if err != nil {
		t.Fatalf("OpenConnection: %v", err)
	}

	var deployErr error
	drain(func(progress chan<- connector.Progress) {
		deployErr = conn.Deploy(context.Background(), deployment, progress)
	})
	if deployErr != nil {
		t.Fatalf("Deploy returned error: %v", deployErr)
	}

	newCache := conn.Close()
	if newCache == nil {
		t.Fatal("expected Close to return a cache")
	}
	want, err := c.hash(deployment)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if got := newCache.(*cache).Hash; got != want {
		t.Fatalf("cache hash = %q, want %q", got, want)
	}
}

func TestConnectionVerify_ReportsAMatch(t *testing.T) {
	c := &Connector{}
	deployment := testDeployment(true)

	conn, err := open(t, c, deployment)
	if err != nil {
		t.Fatalf("OpenConnection: %v", err)
	}

	var ok bool
	var verifyErr error
	drain(func(progress chan<- connector.Progress) {
		ok, verifyErr = conn.Verify(context.Background(), deployment, progress)
	})
	if verifyErr != nil {
		t.Fatalf("Verify returned error: %v", verifyErr)
	}
	if !ok {
		t.Fatal("expected Verify to report ok=true")
	}
	if conn.Close().(*cache).Hash == "" {
		t.Fatal("expected Verify to leave a non-empty cache hash")
	}
}

// Close hands back what the connection knows even when nothing was deployed, so
// a caller can persist the cache on every path.
func TestConnectionClose_KeepsTheOpeningCacheAndRepeats(t *testing.T) {
	c := &Connector{}
	deployment := testDeployment(true)
	opening, err := c.hash(deployment)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	conn, err := c.OpenConnection(context.Background(), deployment.Secret, &cache{opening}, "alice", "host.example", 22, nil)
	if err != nil {
		t.Fatalf("OpenConnection: %v", err)
	}

	if got := conn.Close().(*cache).Hash; got != opening {
		t.Fatalf("cache hash = %q, want the opening hash %q", got, opening)
	}
	if got := conn.Close().(*cache).Hash; got != opening {
		t.Fatalf("second Close returned %q, want %q", got, opening)
	}
}

func TestConnectionDeploy_AfterClose_Fails(t *testing.T) {
	c := &Connector{}
	deployment := testDeployment(true)

	conn, err := open(t, c, deployment)
	if err != nil {
		t.Fatalf("OpenConnection: %v", err)
	}
	conn.Close()

	var deployErr error
	drain(func(progress chan<- connector.Progress) {
		deployErr = conn.Deploy(context.Background(), deployment, progress)
	})
	if deployErr == nil {
		t.Fatal("expected Deploy on a closed connection to fail")
	}
}
