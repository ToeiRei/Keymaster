// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package connector

import (
	"fmt"
	"slices"
)

var connectors = make(map[string]Connector)

func Register(key string, connector Connector) {
	connectors[key] = connector
}

// Keys returns the registered connector keys, sorted for stable ordering.
func Keys() []string {
	keys := make([]string, 0, len(connectors))
	for k := range connectors {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func Resolve(key string) (Connector, error) {
	if connector, ok := connectors[key]; ok {
		return connector, nil
	} else {
		return nil, fmt.Errorf("No registered connector with the key: %s", key)
	}
}
