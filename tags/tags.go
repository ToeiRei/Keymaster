// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package tags

import (
	"fmt"
	"strings"

	"github.com/toeirei/keymaster/util/slicest"
)

const SEPERATOR string = ","

type Tag string
type Tags []Tag

// [Tags] implements [fmt.Stringer]
var _ fmt.Stringer = Tags{}

func (t Tags) String() string {
	return Stringify(t)
}

func Parse(str string) Tags {
	strs := strings.Split(str, SEPERATOR)
	// remove whitespace
	tags := slicest.Map(strs, func(s string) Tag { return Tag(strings.TrimSpace(s)) })
	// filter empty
	return slicest.Filter(tags, func(tag Tag) bool { return tag != "" })
}

func Stringify(tags Tags) string {
	strs := slicest.Map(tags, func(tag Tag) string { return string(tag) })
	return strings.Join(strs, SEPERATOR+" ")
}
