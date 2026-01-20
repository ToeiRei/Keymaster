package windowtitle

import tea "github.com/charmbracelet/bubbletea"

func NewHandler(base string, delimiter string) *TitleHandler {
	return &TitleHandler{
		Base:      base,
		Delimiter: delimiter,
	}
}

type TitleHandler struct {
	Base      string
	Delimiter string
	current   string
}

func (t TitleHandler) render() tea.Cmd {
	if t.current != "" {
		return tea.SetWindowTitle(t.Base + t.Delimiter + t.current)
	} else {
		return tea.SetWindowTitle(t.Base)
	}
}

func (t TitleHandler) Init() tea.Cmd {
	return t.render()
}

func (t *TitleHandler) Handle(msg tea.Msg) tea.Cmd {
	if title, ok := msg.(titleMsg); ok {
		if t.current != string(title) {
			t.current = string(title)
			return t.render()
		}
	}
	return nil
}
