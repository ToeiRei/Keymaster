// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package mock

import (
	"context"
	"testing"

	"github.com/toeirei/keymaster/connector"
)

func testDeployData(succeed bool) connector.DeployData {
	return connector.DeployData{
		Records: []connector.DeployRecord{{
			Algorithm: "ssh-ed25519",
			Data:      "AAAAC3NzaC1lZDI1NTE5AAAAIexample",
			Comment:   "alice@example",
		}},
		Secret:          &Secret{succeed},
		SystemKeySerial: 7,
	}
}

func TestConnectorDeploy_SecretSucceed_Succeeds(t *testing.T) {
	c := &Connector{}
	progress := make(chan connector.Progress)

	var cache connector.Cache
	var err error
	go func() {
		defer close(progress)
		cache, err = c.Deploy(context.Background(), testDeployData(true), connector.ConnectionData{User: "alice", Host: "host.example", Port: 22}, nil, progress)
	}()
	for range progress {
	}

	if err != nil {
		t.Fatalf("Deploy returned error: %v", err)
	}
	if cache == nil {
		t.Fatal("expected Deploy to return a cache")
	}
	if cache.(*Cache).Hash == "" {
		t.Fatal("expected Deploy to return a non-empty cache hash")
	}
}

func TestConnectorDeploy_SecretNotSucceed_Fails(t *testing.T) {
	c := &Connector{}
	progress := make(chan connector.Progress)

	var err error
	go func() {
		defer close(progress)
		_, err = c.Deploy(context.Background(), testDeployData(false), connector.ConnectionData{User: "alice", Host: "host.example", Port: 22}, nil, progress)
	}()
	for range progress {
	}

	if err == nil {
		t.Fatal("expected Deploy to return an error when the secret does not succeed")
	}
}

func TestConnectorVerify_SecretSucceed_Succeeds(t *testing.T) {
	c := &Connector{}
	progress := make(chan connector.Progress)

	var ok bool
	var cache connector.Cache
	var err error
	go func() {
		defer close(progress)
		ok, cache, err = c.Verify(context.Background(), testDeployData(true), connector.ConnectionData{User: "alice", Host: "host.example", Port: 22}, nil, progress)
	}()
	for range progress {
	}

	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected Verify to report ok=true")
	}
	if cache == nil || cache.(*Cache).Hash == "" {
		t.Fatal("expected Verify to return a non-empty cache hash")
	}
}

func TestConnectorVerify_SecretNotSucceed_Fails(t *testing.T) {
	c := &Connector{}
	progress := make(chan connector.Progress)

	var ok bool
	var err error
	go func() {
		defer close(progress)
		ok, _, err = c.Verify(context.Background(), testDeployData(false), connector.ConnectionData{User: "alice", Host: "host.example", Port: 22}, nil, progress)
	}()
	for range progress {
	}

	if err == nil {
		t.Fatal("expected Verify to return an error when the secret does not succeed")
	}
	if ok {
		t.Fatal("expected Verify to report ok=false on failure")
	}
}

func TestConnectorVerifyOffline(t *testing.T) {
	c := &Connector{}

	deployData := testDeployData(true)
	if ok, err := c.VerifyOffline(context.Background(), deployData); err != nil || ok {
		t.Fatalf("expected VerifyOffline to be false with no cache, got ok=%v err=%v", ok, err)
	}

	deployData.Cache = &Cache{c.hash(deployData)}
	if ok, err := c.VerifyOffline(context.Background(), deployData); err != nil || !ok {
		t.Fatalf("expected VerifyOffline to match cache, got ok=%v err=%v", ok, err)
	}

	deployData.Cache = &Cache{"stale"}
	if ok, err := c.VerifyOffline(context.Background(), deployData); err != nil || ok {
		t.Fatalf("expected VerifyOffline to report mismatch, got ok=%v err=%v", ok, err)
	}
}
