// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package util

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func RenderContentInViewportSmooth(
	content string,
	viewportHeight int,
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

func RenderContentInViewportAlign(
	content string,
	viewportSize Size,
	targetPos Vec[int],
	targetSize Size,
	align Vec[lipgloss.Position],
) string {
	return RenderContentInViewportAlignX(
		RenderContentInViewportAlignY(
			content,
			viewportSize.Height,
			targetPos.Y,
			targetSize.Height,
			align.Y,
		),
		viewportSize.Width,
		targetPos.X,
		targetSize.Width,
		align.X,
	)
}

func RenderContentInViewportAlignX(
	content string,
	viewportWidth int,
	targetX int,
	targetWidth int,
	align lipgloss.Position,
) string {
	contentWidth := lipgloss.Width(content)

	// // enlarge content
	// if contentSize.Width <= viewportSize.Width {
	// 	content = lipgloss.PlaceHorizontal(viewportSize.Width, lipgloss.Left, content)
	// }

	// downsize viewport
	viewportWidth = Clamp(targetWidth, viewportWidth, contentWidth)

	// early exit when content fits into viewport
	if contentWidth <= viewportWidth {
		return content
	}

	// ensure sane values to prevent panics
	targetWidth = Clamp(1, targetWidth, viewportWidth)
	targetX = Clamp(0, targetX, contentWidth-targetWidth)
	align = Clamp(lipgloss.Left, align, lipgloss.Right)

	// calculate offsets
	offset := Clamp(
		0,
		targetX-int(math.Round(float64(viewportWidth-targetWidth)*float64(align))),
		contentWidth-viewportWidth,
	)

	// cut X
	lines := strings.Split(content, "\n")
	for i := range lines {
		lines[i] = ansi.TruncateLeft(
			ansi.Truncate(
				lines[i],
				offset+viewportWidth,
				"",
			),
			offset,
			"",
		)
	}

	return strings.Join(lines, "\n")
}

func RenderContentInViewportAlignY(
	content string,
	viewportHeight int,
	targetY int,
	targetHeight int,
	align lipgloss.Position,
) string {
	contentHeight := lipgloss.Height(content)

	// // enlarge content
	// if contentSize.Height <= viewportSize.Height {
	// 	content = lipgloss.PlaceHorizontal(viewportSize.Height, lipgloss.Top, content)
	// }

	// downsize viewport
	viewportHeight = Clamp(targetHeight, viewportHeight, contentHeight)

	// early exit when content fits into viewport
	if contentHeight <= viewportHeight {
		return content
	}

	// ensure sane values to prevent panics
	targetHeight = Clamp(1, targetHeight, viewportHeight)
	targetY = Clamp(0, targetY, contentHeight-targetHeight)
	align = Clamp(lipgloss.Top, align, lipgloss.Bottom)

	// calculate offsets
	offset := Clamp(
		0,
		targetY-int(math.Round(float64(viewportHeight-targetHeight)*float64(align))),
		contentHeight-viewportHeight,
	)

	// cut Y
	lines := strings.Split(content, "\n")[offset : offset+viewportHeight]

	return strings.Join(lines, "\n")
}
