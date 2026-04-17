// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package util

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func RenderContentInViewportSmooth(
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

func RenderContentInViewportAligned(
	content string,
	viewportHeight,
	targetY int,
	targetHeight int,
	align lipgloss.Position,
) string {
	contentHeight := lipgloss.Height(content)
	if contentHeight <= viewportHeight {
		return content
	}

	offsetY := Clamp(
		0,
		targetY-int(math.Round(float64(viewportHeight-targetHeight)*float64(align))),
		contentHeight-viewportHeight,
	)

	lines := strings.Split(content, "\n")
	return strings.Join(lines[offsetY:offsetY+viewportHeight], "\n")
}
