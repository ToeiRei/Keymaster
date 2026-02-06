// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package header

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/models/components/stack"
	"github.com/toeirei/keymaster/ui/tui/util"
)

var SizeConfig = &sizeConfig{}

type sizeConfig struct{}

var _ stack.SizeConfig = (*sizeConfig)(nil)

func (s *sizeConfig) Priority() int { return 10 }

func (s *sizeConfig) Caltulate(model util.Model, _ int, totalSize int) int {
	if totalSize >= 10+1+lipgloss.Height(logo) {
		return lipgloss.Height(logo) + 1
	}
	return 0
}
