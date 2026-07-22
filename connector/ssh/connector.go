// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package ssh

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"

	"github.com/toeirei/keymaster/connector"
	"github.com/toeirei/keymaster/util/slicest"
	"golang.org/x/crypto/ssh"
)

// register Connector
func init() {
	connector.Register("ssh", &Connector{})
}

// sshInternalKeyOptions are the forced-command and restriction options prepended to the
// Keymaster system key line in authorized_keys. They confine the system key to
// SFTP-only access with no forwarding or PTY allocation.
const sshInternalKeyOptions = `command="internal-sftp",no-port-forwarding,no-x11-forwarding,no-agent-forwarding,no-pty`
const sshInternalKeyComment = `do not remove`

// globalKeyCommentPrefix is prepended to the comment of global keys in the
// deployed authorized_keys file so they can be identified as globally-assigned
// rather than account-specific.
const globalKeyCommentPrefix = "GLOBAL-KEY"

// expiryTimeLayout is the OpenSSH expiry-time timestamp layout. The trailing
// "Z" forces sshd to interpret the timestamp as UTC.
const expiryTimeLayout = "20060102150405"

type Connector struct{}

// *[Connector] implements [connector.Connector]
var _ connector.Connector = (*Connector)(nil)

func (c *Connector) Deploy(ctx context.Context, deployData connector.DeployData, connectionData connector.ConnectionData) (chan connector.Progress, error) {
	panic("unimplemented")
}

func (c *Connector) Verify(ctx context.Context, deployData connector.DeployData, connectionData connector.ConnectionData) (chan connector.Progress, error) {
	panic("unimplemented")
}

func (c *Connector) VerifyOffline(ctx context.Context, deployData connector.DeployData) (bool, error) {
	internalPublicKey, err := c.publicKeyFromSecret(deployData.Secret)
	if err != nil {
		return false, err
	}
	authorizedKeys := c.makeAuthorizedKeys(internalPublicKey, deployData.Records)
	localHash := c.hashAuthorizedKeys(authorizedKeys)

	return localHash == deployData.Cache, nil
}

// makeAuthorizedKeys renders the authorized_keys content for an account. The
// restricted Keymaster system key (internalPublicKey) is always written first,
// followed by the deduplicated, non-expired user keys sorted deterministically
// so the resulting content — and therefore its fingerprint — is stable across
// runs and platforms.
func (c *Connector) makeAuthorizedKeys(internalPublicKey string, records []connector.DeployRecord) string {
	lines := make([]string, 0, 5+len(records))

	lines = append(lines,
		"# Keymaster Managed Keys",
		sshInternalKeyOptions+" "+internalPublicKey+" "+sshInternalKeyComment,
	)

	userKeyLines := slicest.Map(records, func(r connector.DeployRecord) string {
		parts := make([]string, 0, 5)

		// options
		if !r.ExpiresAt.IsZero() {
			parts = append(parts, `expiry-time="`+r.ExpiresAt.UTC().Format(expiryTimeLayout)+`Z"`)
		}

		// algo & data
		parts = append(parts, r.Algorithm, r.Data)

		// comments
		if r.IsGlobal {
			parts = append(parts, globalKeyCommentPrefix)
		}
		recordComment := strings.TrimSpace(r.Comment)
		if recordComment != "" {
			parts = append(parts, recordComment)
		}

		// [options] algo data [comment]
		return strings.Join(parts, " ")
	})

	if len(userKeyLines) > 0 {
		// Sort for a deterministic ordering independent of the input order.
		slices.Sort(userKeyLines)

		lines = append(lines, "", "# User Keys")
		lines = append(lines, userKeyLines...)
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// hashAuthorizedKeys returns the SHA256 hex fingerprint of the given
// authorized_keys content. CRLF sequences are normalized to LF and trailing
// whitespace is trimmed per line so fingerprints stay stable when content is
// transferred between platforms.
func (c *Connector) hashAuthorizedKeys(str string) string {
	str = strings.ReplaceAll(str, "\r\n", "\n")
	lines := strings.Split(str, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:])
}

// publicKeyFromSecret parses the PEM-encoded private key held in secret and
// returns its public key in authorized_keys wire format (without a trailing
// newline). It returns an error if the secret cannot be parsed as a private key.
func (c *Connector) publicKeyFromSecret(secret string) (string, error) {
	signer, err := ssh.ParsePrivateKey([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to parse private key from secret: %w", err)
	}
	// MarshalAuthorizedKey appends a single trailing newline; strip just that
	// so the key can be embedded on a line of its own.
	pubKey := ssh.MarshalAuthorizedKey(signer.PublicKey())
	return strings.TrimSuffix(string(pubKey), "\n"), nil
}
