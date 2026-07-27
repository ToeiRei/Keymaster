// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package ssh2

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"

	"github.com/toeirei/keymaster/connector"
	"golang.org/x/crypto/ssh"
)

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

// renderAuthorizedKeys is the single place a deployment turns into file content,
// so what Deploy uploads, what Verify compares against and what VerifyOffline
// hashes cannot drift apart. The public key comes from the deployment's secret
// rather than the connection's: keying a deployment to another secret is how a
// secret update reuses an ordinary deploy.
func (c *Connector) renderAuthorizedKeys(deployment connector.Deployment) (string, error) {
	connectorSecret, err := narrowSecret(deployment.Secret)
	if err != nil {
		return "", err
	}
	internalPublicKey, err := connectorSecret.publicKey()
	if err != nil {
		return "", err
	}
	return c.makeAuthorizedKeys(internalPublicKey, deployment.Records), nil
}

// makeAuthorizedKeys renders the authorized_keys content for an account. The
// restricted Keymaster system key (internalPublicKey) is always written first,
// followed by the deduplicated, non-expired user keys sorted deterministically
// so the resulting content, and therefore its fingerprint, is stable across
// runs and platforms.
func (c *Connector) makeAuthorizedKeys(internalPublicKey string, records []connector.DeployRecord) string {
	lines := make([]string, 0, 10+len(records))
	managedKeyFingerprint := c.managedKeyFingerprint(internalPublicKey)
	keyPayload := c.userKeyPayload(records)
	payloadHash := c.hashAuthorizedKeys(keyPayload)

	lines = append(lines,
		"# Keymaster Managed Keys",
		fmt.Sprintf("# Managed Key Fingerprint: %s", managedKeyFingerprint),
		fmt.Sprintf("# Payload Hash: sha256:%s", payloadHash),
		"# Verify: sha256sum ~/.ssh/authorized_keys | awk '{print $1}'",
		sshInternalKeyOptions+" "+internalPublicKey+" "+sshInternalKeyComment,
	)

	if keyPayload != "" {
		lines = append(lines, "", keyPayload)
	}

	lines = append(lines, "")

	return strings.Join(lines, "\n")
}

func (c *Connector) userKeyPayload(records []connector.DeployRecord) string {
	lines := make([]string, 0, 7+len(records))

	userKeyLines := make([]string, 0)
	userKeyLinesGlobal := make([]string, 0)

	for _, r := range records {
		segments := make([]string, 0, 5)

		// options
		if !r.ExpiresAt.IsZero() {
			segments = append(segments, `expiry-time="`+r.ExpiresAt.UTC().Format(expiryTimeLayout)+`Z"`)
		}

		// algo & data
		segments = append(segments, r.Algorithm, r.Data)

		// comments
		if r.IsGlobal {
			segments = append(segments, globalKeyCommentPrefix)
		}
		recordComment := strings.TrimSpace(r.Comment)
		if recordComment != "" {
			segments = append(segments, recordComment)
		}

		// [options] algo data [comment]
		line := strings.Join(segments, " ")

		if r.IsGlobal {
			userKeyLinesGlobal = append(userKeyLinesGlobal, line)
		} else {
			userKeyLines = append(userKeyLines, line)
		}
	}

	if len(userKeyLinesGlobal) > 0 {
		// Sort for a deterministic ordering independent of the input order.
		slices.Sort(userKeyLinesGlobal)

		lines = append(lines, "", "# Global User Keys")
		lines = append(lines, userKeyLinesGlobal...)
	}

	if len(userKeyLines) > 0 {
		// Sort for a deterministic ordering independent of the input order.
		slices.Sort(userKeyLines)

		lines = append(lines, "", "# User Keys")
		lines = append(lines, userKeyLines...)
	}

	return strings.Join(lines, "\n")
}

func (c *Connector) managedKeyFingerprint(internalPublicKey string) string {
	key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(internalPublicKey))
	if err != nil {
		return ""
	}
	return ssh.FingerprintSHA256(key)
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
