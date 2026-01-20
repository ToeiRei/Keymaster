// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package footer

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/models/components/stack"
	"github.com/toeirei/keymaster/ui/tui/util"
)

var SizeConfig = &sizeConfig{}

type sizeConfig struct{}

var _ stack.SizeConfig = (*sizeConfig)(nil)

func (s *sizeConfig) Priority() int { return 20 }

func (s *sizeConfig) Caltulate(model util.Model, remaining_size int, _ int) int {
	if footer, ok := model.(*Model); ok {
		return lipgloss.Height(footer.view()) + 1
	}
	return 2
}
