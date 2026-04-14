// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package publickey

import (
	"strings"

	"github.com/bobg/go-generics/v4/slices"
)

func parseTags(tags string) []string {
	return slices.Filter( // remove empty user provided tags
		slices.Map( // trim user provided tags
			strings.Split(tags, ","), // split user provided tags
			func(tag string) string { return strings.TrimSpace(tag) },
		),
		func(tag string) bool { return tag != "" },
	)
}

func stringifyTags(tags []string) string {
	return strings.Join(tags, ", ")
}
