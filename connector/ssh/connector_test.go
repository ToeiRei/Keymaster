// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package ssh

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/toeirei/keymaster/connector"
	"github.com/toeirei/keymaster/core/deploy"
	"github.com/toeirei/keymaster/core/security"
)

type fakeSSHDeployer struct {
	deployed string
	closed   bool
	remote   []byte
}

func (f *fakeSSHDeployer) DeployAuthorizedKeys(content string) error {
	f.deployed = content
	return nil
}

func (f *fakeSSHDeployer) GetAuthorizedKeys() ([]byte, error) {
	return f.remote, nil
}

func (f *fakeSSHDeployer) Close() {
	f.closed = true
}

func TestConnectorDeploy_WritesRenderedAuthorizedKeysToRemote(t *testing.T) {
	secret, err := generateTestPrivateKeyPEM()
	if err != nil {
		t.Fatalf("generate test private key: %v", err)
	}

	deployer := &fakeSSHDeployer{}
	oldNewDeployer := newDeployer
	defer func() { newDeployer = oldNewDeployer }()

	newDeployer = func(host, user string, privateKey security.Secret, passphrase []byte, config *deploy.ConnectionConfig, isBootstrap bool) (deployerClient, error) {
		if host != "host.example:22" {
			t.Fatalf("unexpected host: %q", host)
		}
		if user != "alice" {
			t.Fatalf("unexpected user: %q", user)
		}
		if len(privateKey) == 0 {
			t.Fatal("expected private key secret to be passed through")
		}
		if config == nil {
			t.Fatal("expected connection config to be provided")
		}
		if isBootstrap {
			t.Fatal("expected regular deploy path, not bootstrap")
		}
		return deployer, nil
	}

	c := &Connector{}
	progress, err := c.Deploy(context.Background(), connector.DeployData{
		Records: []connector.DeployRecord{{
			Algorithm: "ssh-ed25519",
			Data:      "AAAAC3NzaC1lZDI1NTE5AAAAIexample",
			Comment:   "alice@example",
		}},
		Secret:           secret,
		SystemKeySerial:  7,
	}, connector.ConnectionData{Username: "alice", Host: "host.example", Port: 22}, nil)
	if err != nil {
		t.Fatalf("Deploy returned error: %v", err)
	}

	var sawDone bool
	for progressUpdate := range progress {
		if progressUpdate.Status == "done" {
			sawDone = true
		}
	}
	if !sawDone {
		t.Fatal("expected final progress update after deploy")
	}
	if deployer.closed == false {
		t.Fatal("expected deployer to be closed")
	}

	key, err := c.publicKeyFromSecret(secret)
	if err != nil {
		t.Fatalf("publicKeyFromSecret: %v", err)
	}
	want := c.makeAuthorizedKeys(7, key, []connector.DeployRecord{{
		Algorithm: "ssh-ed25519",
		Data:      "AAAAC3NzaC1lZDI1NTE5AAAAIexample",
		Comment:   "alice@example",
	}})
	if deployer.deployed != want {
		t.Fatalf("unexpected authorized_keys payload\n got: %q\nwant: %q", deployer.deployed, want)
	}
}

func generateTestPrivateKeyPEM() (string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}
	der := x509.MarshalPKCS1PrivateKey(privateKey)
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}
	return string(pem.EncodeToMemory(block)), nil
}
