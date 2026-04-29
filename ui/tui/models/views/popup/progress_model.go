// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package popupviews

import (
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/popup"
	"github.com/toeirei/keymaster/ui/tui/util"
)

type progressMode int

const (
	ProgressSpinner = iota
	ProgressBar
)

type progressId uint32

var progressIdCounter atomic.Uint32

type ProgressModel struct {
	id     progressId
	title  string
	status string

	mode         progressMode
	show         bool
	size         util.Size
	progress     float64
	progressChan ProgressChan

	spinnerModel spinner.Model
	barModel     progress.Model
}

type Progress struct {
	Progress float64
	Status   string
}

// This channel is for reporting progress to the Progress-Popup-Listener. Do not close the channel, as this will be done by the Progress Popup after returning!
type ProgressChan = chan Progress

func OpenProgress(mode progressMode, title string, fn func(ProgressChan) tea.Msg) tea.Cmd {
	id := progressId(progressIdCounter.Add(1))
	progressChan := make(ProgressChan, 1)
	model := &ProgressModel{
		id:           id,
		title:        title,
		mode:         mode,
		progressChan: progressChan,
	}

	switch mode {
	case ProgressSpinner:
		model.spinnerModel = spinner.New(spinner.WithSpinner(spinner.Points))
	case ProgressBar:
		model.barModel = progress.New(progress.WithoutPercentage())
	}

	return tea.Sequence(
		popup.Open(util.ModelPointer(model)),
		func() tea.Msg {
			msg := fn(model.progressChan)
			return progressDoneMsg{model.id, msg}
		},
	)
}

func (m ProgressModel) Init() tea.Cmd {
	var subModelCmd tea.Cmd
	switch m.mode {
	case ProgressSpinner:
		subModelCmd = m.spinnerModel.Tick
	case ProgressBar:
		subModelCmd = m.barModel.Init()
	}

	return tea.Sequence(
		subModelCmd,
		tea.Batch(
			m.ListenProgressCmd,
			func() tea.Msg {
				// give a graceperiod until the progress popup fades in, or show on first [progressProgressMsg]
				time.Sleep(time.Millisecond * 0) // TODO confirm thes is even wanted/needed
				return progressFadeInMsg{m.id}
			},
		),
	)
}

func (m *ProgressModel) Update(msg tea.Msg) tea.Cmd {
	if m.size.UpdateFromMsg(msg) {
		m.barModel.Width = util.Clamp(20, m.size.Width/2, m.size.Width)
		return nil
	}

	if msg, ok := msg.(progressMsg); ok && msg.id() == m.id {
		switch msg := msg.(type) {
		case progressFadeInMsg:
			m.show = true

		case progressProgressMsg:
			m.show = true
			m.progress = msg.progress
			m.status = msg.status

			return m.ListenProgressCmd

		case progressDoneMsg:
			return tea.Sequence(
				popup.Close(),
				func() tea.Msg { return msg.msg },
			)
		}
	}

	if m.mode == ProgressSpinner {
		return util.UpdateTeaModelInplace(msg, &m.spinnerModel)
	}

	return nil
}

func (m *ProgressModel) ListenProgressCmd() tea.Msg {
	if progress, ok := <-m.progressChan; ok {
		return progressProgressMsg{m.id, progress.Progress, progress.Status}
	}
	return nil
}

func (m ProgressModel) View() string {
	if !m.show {
		return ""
	}

	blocks := make([]string, 0, 3)

	switch m.mode {
	case ProgressSpinner:
		blocks = append(blocks, m.spinnerModel.View()+" "+lipgloss.NewStyle().Bold(true).Render(m.title))
	case ProgressBar:
		if m.title != "" {
			blocks = append(blocks, lipgloss.NewStyle().Bold(true).Render(m.title))
		}
		blocks = append(blocks, m.barModel.ViewAs(m.progress))
	}

	if m.status != "" {
		blocks = append(blocks, lipgloss.NewStyle().Italic(true).Render(m.status))
	}

	return lipgloss.JoinVertical(lipgloss.Center, blocks...)
}

func (m *ProgressModel) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	return util.AnnounceKeyMapCmd(parentKeyMap)
}

func (m *ProgressModel) Blur() {}

// *[ProgressModel] implements [util.Model]
var _ util.Model = (*ProgressModel)(nil)
