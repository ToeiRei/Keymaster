// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package ssh

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/toeirei/keymaster/connector"
	"github.com/toeirei/keymaster/core/deploy"
	"github.com/toeirei/keymaster/core/security"
	"github.com/toeirei/keymaster/ui/i18n"
	"golang.org/x/crypto/ssh"
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

	hostKey := testHostKey(t)
	stubHostKeyProbe(t, hostKey, nil)

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
	progress := make(chan connector.Progress)

	var (
		cache     connector.Cache
		deployErr error
	)
	go func() {
		defer close(progress)
		cache, deployErr = c.Deploy(context.Background(), connector.DeployData{
			Records: []connector.DeployRecord{{
				Algorithm: "ssh-ed25519",
				Data:      "AAAAC3NzaC1lZDI1NTE5AAAAIexample",
				Comment:   "alice@example",
			}},
			Secret:          &Secret{PrivateKey: secret},
			Cache:           &Cache{KnownHost: marshalKnownHost(hostKey)},
			SystemKeySerial: 7,
		}, connector.ConnectionData{Username: "alice", Host: "host.example", Port: 22}, nil, progress)
	}()

	var sawDone bool
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
	if cache == nil || cache.(*Cache).AuthorizedKeysHash == "" {
		t.Fatal("expected Deploy to return a non-empty authorized_keys hash")
	}
	if got := cache.(*Cache).KnownHost; got != marshalKnownHost(hostKey) {
		t.Fatalf("expected Deploy to carry the known host forward, got %q", got)
	}
	if deployer.closed == false {
		t.Fatal("expected deployer to be closed")
	}

	key, err := publicKeyFromPrivateKey(secret, "")
	if err != nil {
		t.Fatalf("publicKeyFromPrivateKey: %v", err)
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

// TestConnectorHostKeyTrust covers what Deploy and Verify do with the user's
// answer to an untrusted host key: trusting writes the new key into the cache
// they return, allowing once leaves the cache as it was, and refusing stops the
// operation before it ever connects.
func TestConnectorHostKeyTrust(t *testing.T) {
	secret, err := generateTestPrivateKeyPEM()
	if err != nil {
		t.Fatalf("generate test private key: %v", err)
	}
	hostKey := testHostKey(t)
	stale := marshalKnownHost(testHostKey(t))

	operations := map[string]func(*Connector, connector.DeployData, connector.UserRequester, chan<- connector.Progress) (connector.Cache, error){
		"deploy": func(c *Connector, deployData connector.DeployData, requester connector.UserRequester, progress chan<- connector.Progress) (connector.Cache, error) {
			return c.Deploy(context.Background(), deployData, connector.ConnectionData{Username: "alice", Host: "host.example", Port: 22}, requester, progress)
		},
		"verify": func(c *Connector, deployData connector.DeployData, requester connector.UserRequester, progress chan<- connector.Progress) (connector.Cache, error) {
			_, cache, err := c.Verify(context.Background(), deployData, connector.ConnectionData{Username: "alice", Host: "host.example", Port: 22}, requester, progress)
			return cache, err
		},
	}

	for operation, run := range operations {
		for _, tc := range []struct {
			name        string
			choice      int
			want        string
			wantErr     bool
			wantConnect bool
		}{
			{"trust", hostKeyTrust, marshalKnownHost(hostKey), false, true},
			{"allow once", hostKeyAllowOnce, stale, false, true},
			{"refuse", -1, "", true, false},
		} {
			t.Run(operation+" "+tc.name, func(t *testing.T) {
				stubHostKeyProbe(t, hostKey, nil)

				connected := false
				oldNewDeployer := newDeployer
				defer func() { newDeployer = oldNewDeployer }()
				newDeployer = func(string, string, security.Secret, []byte, *deploy.ConnectionConfig, bool) (deployerClient, error) {
					connected = true
					return &fakeSSHDeployer{}, nil
				}

				requester := &fakeUserRequester{choice: tc.choice}
				progress := make(chan connector.Progress)

				var (
					cache connector.Cache
					opErr error
					data  = connector.DeployData{nil, &Secret{PrivateKey: secret}, &Cache{"stalehash", stale}, 7}
				)
				go func() {
					defer close(progress)
					cache, opErr = run(&Connector{}, data, requester, progress)
				}()
				for range progress { //nolint:revive // drain the progress channel
				}

				if tc.wantErr {
					if opErr == nil {
						t.Fatalf("expected %s to abort on a refused host key", operation)
					}
				} else if opErr != nil {
					t.Fatalf("%s returned error: %v", operation, opErr)
				}
				if connected != tc.wantConnect {
					t.Fatalf("connected = %v, want %v", connected, tc.wantConnect)
				}
				if tc.wantErr {
					return
				}
				if got := cache.(*Cache).KnownHost; got != tc.want {
					t.Fatalf("known host = %q, want %q", got, tc.want)
				}
			})
		}
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

func generateEncryptedTestPrivateKeyPEM(passphrase string) (string, error) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", err
	}
	block, err := ssh.MarshalPrivateKeyWithPassphrase(privateKey, "", []byte(passphrase))
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(block)), nil
}
