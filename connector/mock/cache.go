// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package mock

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/toeirei/keymaster/connector"
	"github.com/toeirei/keymaster/ui/i18n"
)

// cache holds the fingerprint of the deploy data the mock last saw.
type cache struct {
	Hash string `json:"hash"`
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

// validate rejects a hash that cannot have come from [Connector.hash]. The
// check is duplicated from the ssh connector rather than shared: what a cache
// holds is each connector's own business.
func (c *cache) validate() error {
	if c.Hash == "" {
		return nil
	}
	if len(c.Hash) != hex.EncodedLen(sha256.Size) {
		return i18n.NewError("errors.connector.cache_hash", c.Hash)
	}
	if _, err := hex.DecodeString(c.Hash); err != nil {
		return i18n.NewError("errors.connector.cache_hash", c.Hash)
	}
	return nil
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
