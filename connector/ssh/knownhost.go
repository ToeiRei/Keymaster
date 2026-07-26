// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package ssh

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/toeirei/keymaster/connector"
	"github.com/toeirei/keymaster/ui/i18n"
	"golang.org/x/crypto/ssh"
)

// The choices offered when a target's host key is not the one the cache pinned,
// in the order they are presented to the user.
const (
	hostKeyAllowOnce = iota
	hostKeyTrust
)

// hostKeyProbe is the seam the host-key handshake goes through, so tests can
// hand [resolveKnownHost] a key without a network.
var hostKeyProbe = probeHostKey

// probeHostKey opens a throwaway handshake to read the host key a target
// presents. It offers no authentication method, so the handshake fails right
// after the key exchange — by then the callback has captured the key, which is
// all this needs.
func probeHostKey(addr string, timeout time.Duration) (ssh.PublicKey, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, i18n.WrapError(err, "errors.connector.hostkey_probe", addr)
	}
	defer conn.Close() //nolint:errcheck // throwaway probe connection
	_ = conn.SetDeadline(time.Now().Add(timeout))

	var presented ssh.PublicKey
	clientConn, chans, reqs, err := ssh.NewClientConn(conn, addr, &ssh.ClientConfig{
		HostKeyCallback: func(_ string, _ net.Addr, key ssh.PublicKey) error {
			presented = key
			return nil
		},
		Timeout: timeout,
	})
	if err == nil {
		go ssh.DiscardRequests(reqs)
		go func() {
			for ch := range chans {
				_ = ch.Reject(ssh.Prohibited, "")
			}
		}()
		_ = clientConn.Close()
	}

	if presented == nil {
		return nil, i18n.WrapError(err, "errors.connector.hostkey_probe", addr)
	}
	return presented, nil
}

// resolveKnownHost pins the host key of the target at addr. It returns the
// known host to store in the cache: the pinned one when the target still
// presents it, the newly presented one when the user trusts it, and the pinned
// one unchanged when the user allows the connection just this once.
//
// A cache with no known host is a target that has never been reached — it gets
// the same prompt as a mismatch, only worded as a first contact rather than a
// warning. Declining, cancelling, or having no one to ask aborts.
func resolveKnownHost(addr, knownHost string, userRequester connector.UserRequester, timeout time.Duration) (string, error) {
	presentedKey, err := hostKeyProbe(addr, timeout)
	if err != nil {
		return "", err
	}
	presented := marshalKnownHost(presentedKey)

	if strings.TrimSpace(knownHost) == presented {
		return knownHost, nil
	}

	if userRequester == nil {
		return "", knownHostError(addr, knownHost, presentedKey)
	}

	fingerprint := ssh.FingerprintSHA256(presentedKey)
	var choices []fmt.Stringer
	if strings.TrimSpace(knownHost) == "" {
		choices = []fmt.Stringer{
			i18n.Text("connector.ssh.hostkey.unknown.once", addr, fingerprint),
			i18n.Text("connector.ssh.hostkey.unknown.trust", addr, fingerprint),
		}
	} else {
		choices = []fmt.Stringer{
			i18n.Text("connector.ssh.hostkey.mismatch.once", addr, fingerprint),
			i18n.Text("connector.ssh.hostkey.mismatch.trust", addr, fingerprint),
		}
	}

	switch userRequester.RequestChoice(choices) {
	case hostKeyAllowOnce:
		return knownHost, nil
	case hostKeyTrust:
		return presented, nil
	default:
		return "", knownHostError(addr, knownHost, presentedKey)
	}
}

// knownHostError distinguishes a first contact from a key that changed under
// us: the latter is the one worth shouting about.
func knownHostError(addr, knownHost string, presented ssh.PublicKey) error {
	if strings.TrimSpace(knownHost) == "" {
		return i18n.NewError("errors.connector.hostkey_unknown", addr, ssh.FingerprintSHA256(presented))
	}
	return i18n.NewError("errors.connector.hostkey_mismatch", addr, ssh.FingerprintSHA256(presented))
}

// marshalKnownHost renders a host key the way the cache stores it: the
// authorized_keys wire format without the trailing newline MarshalAuthorizedKey
// appends, matching what the legacy known_hosts backfill wrote.
func marshalKnownHost(key ssh.PublicKey) string {
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
}
