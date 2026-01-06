// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package frame

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
)

// Pane is a simple layout primitive that composes a header, a body (optionally
// backed by a viewport) and a footer. It computes the body height based on the
// total width/height and renders a complete string for the TUI to display.
type Pane struct {
	Width  int
	Height int

	Header string
	// Nav is an optional single-line navigation row rendered below the header.
	Nav string
	// BodyMargin specifies left/right padding inside the pane body.
	BodyMargin  int
	FooterLeft  string
	FooterRight string

	Viewport *viewport.Model
}

// NewPane creates an empty Pane.
func NewPane() *Pane {
	return &Pane{}
}

// SetViewport attaches a viewport to the pane. The pane will resize the
// viewport when SetSize is called.
func (p *Pane) SetViewport(vp *viewport.Model) {
	p.Viewport = vp
}

// SetHeader sets the header text (may include multiple lines).
func (p *Pane) SetHeader(h string) { p.Header = h }

// SetFooterTokens sets the left/right footer tokens which will be combined
// using frame.Footer when rendering.
func (p *Pane) SetFooterTokens(left, right string) {
	p.FooterLeft = left
	p.FooterRight = right
}

// SetNav sets an optional single-line navigation row rendered below the header.
func (p *Pane) SetNav(n string) { p.Nav = n }

// SetBodyMargin sets left/right padding inside the body area.
func (p *Pane) SetBodyMargin(m int) { p.BodyMargin = m }

// SetSize sets the pane's total size and resizes the internal viewport if present.
func (p *Pane) SetSize(width, height int) {
	p.Width = width
	p.Height = height
	headerLines := 1 + strings.Count(p.Header, "\n")
	navLines := 0
	if p.Nav != "" {
		navLines = 1
	}
	// Pane no longer reserves space for an internal footer; footer is rendered
	// externally as a status bar to save vertical space.
	footerLines := 0
	bodyHeight := p.Height - headerLines - navLines - footerLines
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	if p.Viewport != nil {
		// Account for body margins when sizing the viewport.
		innerW := p.Width - (p.BodyMargin * 2)
		if innerW < 1 {
			innerW = 1
		}
		p.Viewport.Width = innerW
		p.Viewport.Height = bodyHeight
	}
}

// View renders the pane as a single string combining header, body and footer.
func (p *Pane) View() string {
	var b strings.Builder
	// Header (left-aligned — Pane will not attempt to center the header when
	// a left column is present; consumers should provide the desired title.)
	if p.Header != "" {
		for _, hl := range strings.Split(p.Header, "\n") {
			// truncate header lines to available width
			if p.Width > 0 {
				runes := []rune(hl)
				if len(runes) > p.Width {
					hl = string(runes[:p.Width])
				}
			}
			b.WriteString(hl + "\n")
		}
		if p.Width > 0 {
			b.WriteString(strings.Repeat("─", p.Width))
			b.WriteString("\n")
		}
	}
	// Nav (optional, left-aligned)
	if p.Nav != "" {
		b.WriteString(p.Nav + "\n")
		if p.Width > 0 {
			b.WriteString(strings.Repeat("─", p.Width))
			b.WriteString("\n")
		}
	}
	// Body (with margins)
	// compute body height to render empty lines when no viewport is present
	headerLines := 0
	if p.Header != "" {
		headerLines = 1 + strings.Count(p.Header, "\n")
	}
	navLines := 0
	if p.Nav != "" {
		navLines = 1
	}
	footerLines := 1
	bodyLines := p.Height - headerLines - navLines - footerLines
	if bodyLines < 1 {
		bodyLines = 1
	}

	if p.Viewport != nil {
		// Render viewport lines and add left/right margins. Truncate long
		// lines to the inner width (body area) and append an ellipsis when
		// appropriate so layout remains table-like.
		vpLines := strings.Split(strings.TrimRight(p.Viewport.View(), "\n"), "\n")
		margin := strings.Repeat(" ", p.BodyMargin)
		innerW := p.Width - (p.BodyMargin * 2)
		if innerW < 0 {
			innerW = 0
		}
		for i := 0; i < bodyLines; i++ {
			ln := ""
			if i < len(vpLines) {
				ln = vpLines[i]
			}
			// truncate to inner width
			content := ln
			if innerW > 0 {
				runes := []rune(ln)
				if len(runes) > innerW {
					if innerW > 1 {
						// leave room for ellipsis
						content = string(runes[:innerW-1]) + "…"
					} else {
						content = string(runes[:innerW])
					}
				}
			}
			b.WriteString(margin)
			b.WriteString(content)
			// pad the right side so lines are consistent width
			rightPad := p.Width - p.BodyMargin - len([]rune(content))
			if rightPad < 0 {
				rightPad = 0
			}
			b.WriteString(strings.Repeat(" ", rightPad))
			b.WriteString("\n")
		}
	} else {
		// render blank body lines with margins
		margin := strings.Repeat(" ", p.BodyMargin)
		for i := 0; i < bodyLines; i++ {
			// pad the right side so lines are consistent width
			rightPad := p.Width - (p.BodyMargin * 2)
			if rightPad < 0 {
				rightPad = 0
			}
			b.WriteString(margin + strings.Repeat(" ", rightPad) + margin + "\n")
		}
	}
	return b.String()
}
