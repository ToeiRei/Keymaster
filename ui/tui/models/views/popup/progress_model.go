// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package popupviews

import (
	"context"
	"sync/atomic"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	formelement "github.com/toeirei/keymaster/ui/tui/models/helpers/form/element"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/popup"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

type progressMode int

const (
	ProgressSpinner = iota
	ProgressBar
)

type progressId uint32

var progressIdCounter atomic.Uint32

type ProgressModel struct {
	id        progressId
	title     string
	status    string
	mode      progressMode
	ctx       context.Context
	ctxCancel context.CancelFunc

	size         util.Size
	progress     float64
	progressChan ProgressChan

	spinnerModel spinner.Model
	barModel     progress.Model
	formModel    *form.Form[struct{}]
}

type Progress struct {
	Progress float64
	Status   string
}

// This channel is for reporting progress to the Progress-Popup-Listener. Do not close the channel, as this will be done by the Progress Popup after returning!
type ProgressChan = chan Progress

type ProgressOption func(m *ProgressModel)

func WithCancel() ProgressOption {
	return func(m *ProgressModel) { m.ctx, m.ctxCancel = context.WithCancel(m.ctx) }
}

// overrides default context and all previos cancel settings
func WithContext(ctx context.Context) ProgressOption {
	return func(m *ProgressModel) { m.ctx, m.ctxCancel = ctx, nil }
}

func OpenProgress(mode progressMode, title string, fn func(ctx context.Context, pc ProgressChan) tea.Cmd, opts ...ProgressOption) tea.Cmd {
	id := progressId(progressIdCounter.Add(1))
	progressChan := make(ProgressChan)
	model := &ProgressModel{
		id:           id,
		title:        title,
		mode:         mode,
		ctx:          context.Background(),
		ctxCancel:    nil,
		progressChan: progressChan,
	}

	switch mode {
	case ProgressSpinner:
		model.spinnerModel = spinner.New(spinner.WithSpinner(spinner.Points))
	case ProgressBar:
		model.barModel = progress.New(progress.WithoutPercentage())
	}

	for _, opt := range opts {
		opt(model)
	}

	return tea.Sequence(
		popup.Open(util.ModelPointer(model)),
		func() tea.Msg { return progressDoneMsg{model.id, fn(model.ctx, model.progressChan)} },
	)
}

func (m *ProgressModel) Init() tea.Cmd {
	var subModelCmd tea.Cmd
	switch m.mode {
	case ProgressSpinner:
		subModelCmd = m.spinnerModel.Tick
	case ProgressBar:
		subModelCmd = m.barModel.Init()
	}

	var formOpts []form.FormOpt[struct{}]
	if m.ctxCancel != nil {
		formOpts = append(formOpts, form.WithRowItem[struct{}]("_cancel", formelement.NewButton("Cancel",
			formelement.WithButtonActionCancel(),
			formelement.WithButtonGlobalKeyBindings(keys.Cancel()),
		)))
	}

	m.formModel = util.NewPointer(form.New(
		append([]form.FormOpt[struct{}]{
			form.WithDefaultRowAlign[struct{}](form.Center),
			form.WithOnCancel[struct{}](func() tea.Cmd {
				m.ctxCancel()
				return nil
			}),
		}, formOpts...)...,
	))

	return tea.Sequence(
		tea.Batch(subModelCmd, m.formModel.Init()),
		m.ListenProgressCmd,
	)
}

func (m *ProgressModel) Update(msg tea.Msg) tea.Cmd {
	if m.size.UpdateFromMsg(msg) {
		m.barModel.Width = util.Clamp(20, m.size.Width/2, m.size.Width)
		return m.formModel.Update(util.Size{m.size.Width, 3}.ToMsg())
	}

	if msg, ok := msg.(progressMsg); ok && msg.id() == m.id {
		switch msg := msg.(type) {
		case progressProgressMsg:
			m.progress = msg.progress
			m.status = msg.status

			return m.ListenProgressCmd

		case progressDoneMsg:
			close(m.progressChan)
			return tea.Sequence(
				popup.Close(),
				msg.cmd,
			)
		}
	}

	cmd := m.formModel.Update(msg)

	if m.mode == ProgressSpinner {
		cmd = tea.Batch(cmd, util.UpdateTeaModelInplace(msg, &m.spinnerModel))
	}

	return cmd
}

func (m *ProgressModel) ListenProgressCmd() tea.Msg {
	if progress, ok := <-m.progressChan; ok {
		return progressProgressMsg{m.id, progress.Progress, progress.Status}
	}
	return nil
}

func (m ProgressModel) View() string {
	blocks := make([]string, 0, 4)

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

	formView := m.formModel.ViewLazy()
	if formView != "" {
		blocks = append(blocks, formView)
	}

	return lipgloss.JoinVertical(lipgloss.Center, blocks...)
}

func (m *ProgressModel) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	return m.formModel.Focus(parentKeyMap)
}

func (m *ProgressModel) Blur() { m.formModel.Blur() }

// *[ProgressModel] implements [util.Model]
var _ util.Model = (*ProgressModel)(nil)
