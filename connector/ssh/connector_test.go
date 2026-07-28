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
	"golang.org/x/crypto/ssh"
)

type fakeSSHDeployer struct {
	deployed  string
	closed    bool
	remote    []byte
	deployErr error
}

func (f *fakeSSHDeployer) DeployAuthorizedKeys(content string) error {
	if f.deployErr != nil {
		return f.deployErr
	}
	f.deployed = content
	return nil
}

func (f *fakeSSHDeployer) GetAuthorizedKeys() ([]byte, error) {
	return f.remote, nil
}

func (f *fakeSSHDeployer) Close() {
	f.closed = true
}

// stubDeployer points newDeployer at the given client for the test's duration.
func stubDeployer(t *testing.T, client deployerClient) {
	t.Helper()
	old := newDeployer
	t.Cleanup(func() { newDeployer = old })
	newDeployer = func(string, string, security.Secret, []byte, *deploy.ConnectionConfig, bool) (deployerClient, error) {
		return client, nil
	}
}

// testSecret builds a usable secret and the public key that belongs to it, since
// the connector only ever installs the stored public half.
func testSecret(t *testing.T) *secret {
	t.Helper()
	privateKey, err := generateTestPrivateKeyPEM()
	if err != nil {
		t.Fatalf("generate test private key: %v", err)
	}
	publicKey, _, err := publicKeyFromPrivateKey(privateKey, "")
	if err != nil {
		t.Fatalf("publicKeyFromPrivateKey: %v", err)
	}
	return &secret{privateKey, "", false, publicKey}
}

func testRecords() []connector.DeployRecord {
	return []connector.DeployRecord{{
		Algorithm: "ssh-ed25519",
		Data:      "AAAAC3NzaC1lZDI1NTE5AAAAIexample",
		Comment:   "alice@example",
	}}
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

func TestOpenConnection_PinsTheResolvedHostKeyAndPassesTheSecret(t *testing.T) {
	connectorSecret := testSecret(t)
	hostKey := testHostKey(t)
	stubHostKeyProbe(t, hostKey, nil)

	deployer := &fakeSSHDeployer{}
	old := newDeployer
	t.Cleanup(func() { newDeployer = old })
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
		// The resolved host key has to be pinned on the config: core/deploy's own
		// callback would look the host up in the global core/db store, which the
		// bunrewrite client never initializes.
		if config.HostKeyCallback == nil {
			t.Error("expected the resolved host key to be pinned on the connection config")
		} else {
			if err := config.HostKeyCallback("host.example:22", nil, hostKey); err != nil {
				t.Errorf("pinned callback rejected the resolved host key: %v", err)
			}
			if err := config.HostKeyCallback("host.example:22", nil, testHostKey(t)); err == nil {
				t.Error("pinned callback accepted a host key other than the resolved one")
			}
		}
		if isBootstrap {
			t.Fatal("expected regular deploy path, not bootstrap")
		}
		return deployer, nil
	}

	c := &Connector{}
	conn, err := c.OpenConnection(context.Background(), connectorSecret, &cache{KnownHost: marshalKnownHost(hostKey)}, "alice", "host.example", 22, nil)
	if err != nil {
		t.Fatalf("OpenConnection returned error: %v", err)
	}

	conn.Close()
	if !deployer.closed {
		t.Fatal("expected Close to close the underlying deployer")
	}
}

// TestOpenConnection_HostKeyTrust covers what opening does with the user's answer
// to an untrusted host key: trusting writes the new key into the cache the
// connection hands back, allowing once leaves the cache as it was, and refusing
// stops the operation before it ever connects.
func TestOpenConnection_HostKeyTrust(t *testing.T) {
	connectorSecret := testSecret(t)
	hostKey := testHostKey(t)
	stale := marshalKnownHost(testHostKey(t))

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
		t.Run(tc.name, func(t *testing.T) {
			stubHostKeyProbe(t, hostKey, nil)

			connected := false
			old := newDeployer
			t.Cleanup(func() { newDeployer = old })
			newDeployer = func(string, string, security.Secret, []byte, *deploy.ConnectionConfig, bool) (deployerClient, error) {
				connected = true
				return &fakeSSHDeployer{}, nil
			}

			c := &Connector{}
			conn, err := c.OpenConnection(context.Background(), connectorSecret, &cache{"stalehash", stale}, "alice", "host.example", 22, &fakeUserRequester{choice: tc.choice})

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected OpenConnection to abort on a refused host key")
				}
			} else if err != nil {
				t.Fatalf("OpenConnection returned error: %v", err)
			}
			if connected != tc.wantConnect {
				t.Fatalf("connected = %v, want %v", connected, tc.wantConnect)
			}
			if tc.wantErr {
				return
			}
			if got := conn.Close().(*cache).KnownHost; got != tc.want {
				t.Fatalf("known host = %q, want %q", got, tc.want)
			}
		})
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
