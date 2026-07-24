// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package ssh

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"slices"
	"strconv"
	"strings"

	"github.com/toeirei/keymaster/connector"
	"github.com/toeirei/keymaster/core/deploy"
	"github.com/toeirei/keymaster/core/security"
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

type deployerClient interface {
	DeployAuthorizedKeys(content string) error
	GetAuthorizedKeys() ([]byte, error)
	Close()
}

var newDeployer = func(host, user string, privateKey security.Secret, passphrase []byte, config *deploy.ConnectionConfig, isBootstrap bool) (deployerClient, error) {
	return deploy.NewDeployerWithConfig(host, user, privateKey, passphrase, config, isBootstrap)
}

// *[Connector] implements [connector.Connector]
var _ connector.Connector = (*Connector)(nil)

func (c *Connector) Deploy(ctx context.Context, deployData connector.DeployData, connectionData connector.ConnectionData, userRequester connector.UserRequester, progress chan<- connector.Progress) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	progress <- connector.Progress{Progress: 0.1, Status: "rendering authorized_keys"}
	internalPublicKey, err := c.publicKeyFromSecret(deployData.Secret)
	if err != nil {
		return "", fmt.Errorf("failed to parse deploy secret: %w", err)
	}

	authorizedKeys := c.makeAuthorizedKeys(deployData.SystemKeySerial, internalPublicKey, deployData.Records)
	progress <- connector.Progress{Progress: 0.25, Status: "connecting to remote host"}

	addr := canonicalSSHAddress(connectionData.Host, connectionData.Port)
	client, err := newDeployer(addr, connectionData.Username, security.FromString(deployData.Secret), nil, deploy.DefaultConnectionConfig(), false)
	if err != nil {
		return "", fmt.Errorf("failed to connect to %s@%s: %w", connectionData.Username, addr, err)
	}
	defer client.Close()

	progress <- connector.Progress{Progress: 0.6, Status: "uploading authorized_keys"}
	if err := client.DeployAuthorizedKeys(authorizedKeys); err != nil {
		return "", fmt.Errorf("failed to deploy authorized_keys: %w", err)
	}

	progress <- connector.Progress{Progress: 1, Status: "done"}
	return c.hashAuthorizedKeys(authorizedKeys), nil
}

func (c *Connector) Verify(ctx context.Context, deployData connector.DeployData, connectionData connector.ConnectionData, userRequester connector.UserRequester, progress chan<- connector.Progress) (bool, string, error) {
	if ctx.Err() != nil {
		return false, "", ctx.Err()
	}

	progress <- connector.Progress{Progress: 0.1, Status: "rendering authorized_keys"}
	internalPublicKey, err := c.publicKeyFromSecret(deployData.Secret)
	if err != nil {
		return false, "", fmt.Errorf("failed to parse deploy secret: %w", err)
	}

	expected := c.makeAuthorizedKeys(deployData.SystemKeySerial, internalPublicKey, deployData.Records)
	progress <- connector.Progress{Progress: 0.3, Status: "connecting to remote host"}

	addr := canonicalSSHAddress(connectionData.Host, connectionData.Port)
	client, err := newDeployer(addr, connectionData.Username, security.FromString(deployData.Secret), nil, deploy.DefaultConnectionConfig(), false)
	if err != nil {
		return false, "", fmt.Errorf("failed to connect to %s@%s: %w", connectionData.Username, addr, err)
	}
	defer client.Close()

	progress <- connector.Progress{Progress: 0.6, Status: "reading remote authorized_keys"}
	remoteBytes, err := client.GetAuthorizedKeys()
	if err != nil {
		return false, "", fmt.Errorf("failed to read authorized_keys: %w", err)
	}

	remoteHash := c.hashAuthorizedKeys(string(remoteBytes))
	expectedHash := c.hashAuthorizedKeys(expected)

	ok := remoteHash == expectedHash
	if ok {
		progress <- connector.Progress{Progress: 1, Status: "verified"}
	} else {
		progress <- connector.Progress{Progress: 1, Status: "drift detected"}
	}

	return ok, remoteHash, nil
}

func (c *Connector) VerifyOffline(ctx context.Context, deployData connector.DeployData) (bool, error) {
	if deployData.Cache == "" {
		return false, nil
	}

	internalPublicKey, err := c.publicKeyFromSecret(deployData.Secret)
	if err != nil {
		return false, err
	}
	authorizedKeys := c.makeAuthorizedKeys(deployData.SystemKeySerial, internalPublicKey, deployData.Records)
	localHash := c.hashAuthorizedKeys(authorizedKeys)

	return localHash == deployData.Cache, nil
}

func canonicalSSHAddress(host string, port int) string {
	if port <= 0 {
		port = 22
	}
	return net.JoinHostPort(strings.TrimSpace(host), strconv.Itoa(port))
}

// makeAuthorizedKeys renders the authorized_keys content for an account. The
// restricted Keymaster system key (internalPublicKey) is always written first,
// followed by the deduplicated, non-expired user keys sorted deterministically
// so the resulting content — and therefore its fingerprint — is stable across
// runs and platforms.
func (c *Connector) makeAuthorizedKeys(serial int, internalPublicKey string, records []connector.DeployRecord) string {
	lines := make([]string, 0, 10+len(records))
	managedKeyFingerprint := c.managedKeyFingerprint(internalPublicKey)
	keyPayload := c.userKeyPayload(records)
	payloadHash := c.hashAuthorizedKeys(keyPayload)

	lines = append(lines,
		fmt.Sprintf("# Keymaster Managed Keys (Serial: %d)", serial),
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
