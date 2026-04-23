// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package popupviews

import (
	"sync/atomic"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/popup"
	"github.com/toeirei/keymaster/ui/tui/util"
)

var progressId atomic.Uint32

type ProgressModel struct {
	id     uint32
	title  string
	status string

	size          util.Size
	progress      float64
	progressChan  ProgressChan
	progressModel progress.Model
}

type Progress struct {
	Progress float64
	Status   string
}

type ProgressMsg struct {
	pid      uint32
	progress Progress
}

type ProgressDoneMsg struct {
	pid uint32
}

// use to send current progress and close to finish task/close progress popup
type ProgressChan chan Progress

func OpenProgress(title string) (tea.Cmd, ProgressChan) {
	model, progressChan := newProgress(title)
	return popup.Open(util.ModelPointer(model)), progressChan
}

func newProgress(title string) (*ProgressModel, ProgressChan) {
	progressChan := make(ProgressChan)
	progressModel := progress.New()
	progressModel.ShowPercentage = false
	return &ProgressModel{
		id:            progressId.Add(1),
		title:         title,
		progressChan:  progressChan,
		progressModel: progressModel,
	}, progressChan
}

func (m ProgressModel) Init() tea.Cmd {
	return tea.Sequence(
		m.progressModel.Init(),
		m.ListenProgressCmd,
	)
}

func (m *ProgressModel) Update(msg tea.Msg) tea.Cmd {
	if m.size.UpdateFromMsg(msg) {
		m.progressModel.Width = util.Clamp(20, m.size.Width/2, m.size.Width)
		return nil
	}

	if msg, ok := msg.(ProgressMsg); ok && msg.pid == m.id {
		m.progress = msg.progress.Progress
		m.status = msg.progress.Status

		return m.ListenProgressCmd
	}

	if msg, ok := msg.(ProgressDoneMsg); ok && msg.pid == m.id {
		return popup.Close()
	}

	return nil
}

func (m *ProgressModel) ListenProgressCmd() tea.Msg {
	progress, ok := <-m.progressChan
	if !ok {
		return ProgressDoneMsg{m.id}
	}
	return ProgressMsg{m.id, progress}
}

func (m ProgressModel) View() string {
	lines := make([]string, 0, 3)

	if m.title != "" {
		lines = append(lines, lipgloss.NewStyle().Bold(true).Render(m.title))
	}
	lines = append(lines, m.progressModel.ViewAs(m.progress))
	if m.status != "" {
		lines = append(lines, lipgloss.NewStyle().Italic(true).Render(m.status))
	}

	return lipgloss.JoinVertical(lipgloss.Center, lines...)
}

func (m *ProgressModel) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	return util.AnnounceKeyMapCmd(parentKeyMap)
}

func (m *ProgressModel) Blur() {}

// *[ProgressModel] implements [util.Model]
var _ util.Model = (*ProgressModel)(nil)
