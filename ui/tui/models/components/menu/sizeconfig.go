package menu

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/models/components/stack"
	"github.com/toeirei/keymaster/ui/tui/util"
)

const min_size int = 20
const max_size int = 40

var SizeConfig = &sizeConfig{}

type sizeConfig struct{}

var _ stack.SizeConfig = (*sizeConfig)(nil)

func (s *sizeConfig) Priority() int { return 20 }

func (s *sizeConfig) Caltulate(model util.Model, remaining_size int, _ int) int {
	if menu, ok := model.(*Model); ok {
		if !menu.focused {
			return min_size
		}
		// clamp needed size
		return util.Clamp(
			// min
			min_size,
			// wanted
			lipgloss.Width(menu.view())+2,
			// max
			min(max_size, remaining_size),
		)
	}
	return min_size
}
