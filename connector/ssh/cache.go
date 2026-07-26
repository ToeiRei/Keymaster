// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package ssh

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/toeirei/keymaster/connector"
	"github.com/toeirei/keymaster/ui/i18n"
)

// Cache records what Keymaster last saw on a target: the fingerprint of the
// deployed authorized_keys and the host key the target presented.
type Cache struct {
	AuthorizedKeysHash string `json:"authorized_keys_hash"`
	KnownHost          string `json:"known_host,omitempty"`
}

// *[Cache] implements [connector.Cache]
var _ connector.Cache = (*Cache)(nil)

func (c *Cache) Serialize() (string, error) {
	raw, err := json.Marshal(c)
	if err != nil {
		return "", i18n.WrapError(err, "errors.connector.serialize_cache")
	}
	return string(raw), nil
}

func (c *Connector) NewCache(raw string) (connector.Cache, error) {
	cache := &Cache{}
	if strings.TrimSpace(raw) == "" {
		return cache, nil
	}
	if err := json.Unmarshal([]byte(raw), cache); err != nil {
		return nil, i18n.WrapError(err, "errors.connector.parse_cache")
	}
	return cache, nil
}

// cacheOf narrows the deploy data's cache to this connector's type. A missing
// cache is an account that has never been deployed, not an error.
func cacheOf(deployData connector.DeployData) (*Cache, error) {
	if deployData.Cache == nil {
		return &Cache{}, nil
	}
	cache, ok := deployData.Cache.(*Cache)
	if !ok {
		return nil, i18n.NewError("errors.connector.cache_type", fmt.Sprintf("%T", deployData.Cache))
	}
	return cache, nil
}
