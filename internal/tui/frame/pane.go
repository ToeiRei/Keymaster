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

	Header      string
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

// SetSize sets the pane's total size and resizes the internal viewport if present.
func (p *Pane) SetSize(width, height int) {
	p.Width = width
	p.Height = height
	headerLines := 1 + strings.Count(p.Header, "\n")
	footerLines := 1
	bodyHeight := p.Height - headerLines - footerLines
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	if p.Viewport != nil {
		p.Viewport.Width = p.Width
		p.Viewport.Height = bodyHeight
	}
}

// View renders the pane as a single string combining header, body and footer.
func (p *Pane) View() string {
	var b strings.Builder
	b.WriteString(p.Header)
	b.WriteString("\n")
	if p.Viewport != nil {
		b.WriteString(p.Viewport.View())
	}
	b.WriteString("\n")
	b.WriteString(Footer(p.FooterLeft, p.FooterRight, p.Width))
	return b.String()
}
