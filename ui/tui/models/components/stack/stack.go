package stack

import (
	"github.com/bobg/go-generics/v4/slices"
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/util/slicest"
)

const (
	Vertical   Orientation = true
	Horizontal Orientation = false
)

type Orientation bool

type Model struct {
	Orientation Orientation
	Align       lipgloss.Position
	Gap         int
	MsgFilters  []MsgFilter

	items         []Item
	size          util.Size
	focussedIndex Focus
}

type Item struct {
	Model      *util.Model
	SizeConfig SizeConfig
	MsgFilters []MsgFilter
	size       int
	old_size   int
}

func (s Model) Init() tea.Cmd {
	return tea.Batch(slices.Map(s.items, func(item Item) tea.Cmd {
		return (*item.Model).Init()
	})...)
}

func (s *Model) Update(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	if s.size.Update(msg) {
		s.calculateItemSizes()
		cmds = append(cmds, s.updateResizedItems(true)...)
	} else {
		cmds = append(cmds, slicest.Map(s.items, func(item Item) tea.Cmd {
			// apply message filtes
			msg = applyMessageFilters(*item.Model, msg, item.MsgFilters)
			msg = applyMessageFilters(*item.Model, msg, s.MsgFilters)
			if msg == nil {
				return nil
			}

			// update model
			return (*item.Model).Update(msg)
		})...)

		s.calculateItemSizes()
		cmds = append(cmds, s.updateResizedItems(false)...)
	}

	return tea.Batch(cmds...)
}

func (s Model) View() string {
	// prepare based on orientation
	var joiner func(pos lipgloss.Position, strs ...string) string
	var styler func(size int, margin int) lipgloss.Style
	switch s.Orientation {
	case Vertical:
		joiner = lipgloss.JoinVertical
		styler = func(size int, margin int) lipgloss.Style {
			return lipgloss.
				NewStyle().
				Width(s.size.Width).
				Height(size + margin).
				MaxWidth(s.size.Width).
				MaxHeight(size + margin).
				MarginTop(margin)
		}
	case Horizontal:
		joiner = lipgloss.JoinHorizontal
		styler = func(size int, margin int) lipgloss.Style {
			return lipgloss.
				NewStyle().
				Width(size + margin).
				Height(s.size.Height).
				MaxWidth(size + margin).
				MaxHeight(s.size.Height).
				MarginLeft(margin)
		}
	}

	// join rendered items
	return joiner(
		s.Align,
		slicest.MapI(s.items, func(i int, item Item) string {
			// no gap on first item
			margin := s.Gap * min(i, 1)
			// render item
			if item.size == 0 {
				return ""
			}
			tmp1 := (*item.Model).View()
			tmp2 := styler(item.size, margin).Render(tmp1)
			return tmp2
		})...,
	)
}

func (m *Model) Focus() (tea.Cmd, help.KeyMap) {
	if m.focussedIndex == Focus(-1) {
		cmds := make([]tea.Cmd, len(m.items))
		keyMaps := make([]help.KeyMap, len(m.items))

		for i, item := range m.items {
			cmds[i], keyMaps[i] = (*item.Model).Focus()
		}

		return tea.Batch(cmds...), util.MergeKeyMaps(keyMaps...)
	} else {
		return (*m.items[m.focussedIndex].Model).Focus()
	}
}

func (m *Model) Blur() {
	if m.focussedIndex == Focus(-1) {
		for _, item := range m.items {
			(*item.Model).Blur()
		}
	} else {
		(*m.items[m.focussedIndex].Model).Blur()
	}
}

// *Model implements util.Model
var _ util.Model = (*Model)(nil)

type Focus int

func FocusAll() Focus        { return -1 }
func FocusIndex(i int) Focus { return Focus(i) }

func (m *Model) SetFocus(focus Focus) (tea.Cmd, help.KeyMap) {
	m.Blur()
	m.focussedIndex = util.Clamp(Focus(-1), focus, Focus(len(m.items)-1))
	return m.Focus()
}
