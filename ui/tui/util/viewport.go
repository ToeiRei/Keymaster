// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package util

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func RenderContentInViewport(
	content string,
	viewportHeight,
	targetY int,
	targetHeight int,
) string {
	contentHeight := lipgloss.Height(content)
	if contentHeight <= viewportHeight {
		return content
	}

	percentScroll := float64(targetY) / float64(contentHeight-targetHeight)
	offsetY := int(math.Round(percentScroll * float64(contentHeight-viewportHeight)))
	lines := strings.Split(content, "\n")
	return strings.Join(lines[offsetY:offsetY+viewportHeight], "\n")
}
