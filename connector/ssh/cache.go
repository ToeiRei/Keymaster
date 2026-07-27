// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package ssh

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/toeirei/keymaster/connector"
	"github.com/toeirei/keymaster/ui/i18n"
	"golang.org/x/crypto/ssh"
)

// cache records what Keymaster last saw on a target: the fingerprint of the
// deployed authorized_keys and the host key the target presented.
type cache struct {
	AuthorizedKeysHash string `json:"authorized_keys_hash"`
	KnownHost          string `json:"known_host,omitempty"`
}

// implements [connector.Cache]
var _ connector.Cache = (*cache)(nil)

func (c *cache) Serialize() (string, error) {
	raw, err := json.Marshal(c)
	if err != nil {
		return "", i18n.WrapError(err, "errors.connector.serialize_cache")
	}
	return string(raw), nil
}

func (c *Connector) ParseCache(raw string) (connector.Cache, error) {
	parsed := &cache{}
	if strings.TrimSpace(raw) == "" {
		return parsed, nil
	}
	if err := json.Unmarshal([]byte(raw), parsed); err != nil {
		return nil, i18n.WrapError(err, "errors.connector.parse_cache")
	}
	if err := parsed.validate(); err != nil {
		return nil, err
	}
	return parsed, nil
}

// validate rejects a cache that cannot have come out of a deploy: the hash is
// always what hashAuthorizedKeys wrote, and the known host is always the
// canonical rendering resolveKnownHost compares a presented key against. A
// value that only nearly matches would read as drift or as a changed host key,
// so it is refused here rather than misread later.
func (c *cache) validate() error {
	if c.AuthorizedKeysHash != "" && !isSHA256Hex(c.AuthorizedKeysHash) {
		return i18n.NewError("errors.connector.cache_hash", c.AuthorizedKeysHash)
	}
	if c.KnownHost == "" {
		return nil
	}
	knownHost, _, _, _, err := ssh.ParseAuthorizedKey([]byte(c.KnownHost))
	if err != nil {
		return i18n.WrapError(err, "errors.connector.parse_known_host")
	}
	if marshalKnownHost(knownHost) != c.KnownHost {
		return i18n.NewError("errors.connector.known_host_not_canonical", c.KnownHost)
	}
	return nil
}

// isSHA256Hex reports whether str is what hashAuthorizedKeys produces.
func isSHA256Hex(str string) bool {
	if len(str) != hex.EncodedLen(sha256.Size) {
		return false
	}
	for _, r := range str {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

// cacheOf narrows the deploy data's cache to this connector's type. A missing
// cache is an account that has never been deployed, not an error.
func cacheOf(deployData connector.DeployData) (*cache, error) {
	if deployData.Cache == nil {
		return &cache{}, nil
	}
	parsed, ok := deployData.Cache.(*cache)
	if !ok {
		return nil, i18n.NewError("errors.connector.cache_type", fmt.Sprintf("%T", deployData.Cache))
	}
	return parsed, nil
}
