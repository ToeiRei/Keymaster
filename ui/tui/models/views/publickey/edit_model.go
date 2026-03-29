// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package publickey

import (
	"context"
	"strings"

	"github.com/bobg/go-generics/v4/slices"
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/ui/tui/models/components/router"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	formelement "github.com/toeirei/keymaster/ui/tui/models/helpers/form/element"
	"github.com/toeirei/keymaster/ui/tui/util"
)

type editFormData struct {
	comment string `mapstructure:"comment"`
	tags    string `mapstructure:"tags"`
}

type EditModel struct {
	// state
	publicKeyId  client.ID
	publicKey    client.PublicKey
	locked       *string
	loadingError error
	focussed     bool

	// util
	client client.Client
	rc     router.Controll
	size   util.Size

	// sub models
	form *form.Form[editFormData]
}

func NewEdit(c client.Client, rc router.Controll, id client.ID) *EditModel {
	return &EditModel{
		client: c,
		rc:     rc,
		form: util.NewPointer(form.New(
			form.WithSingleElementRow[editFormData]("comment", formelement.NewText("Comment", "comment that will also be deployed to authorized_keys file")),
			form.WithSingleElementRow[editFormData]("tags", formelement.NewText("tags", "comma seperated list of tags")),
			form.WithRow(
				form.WithElement[editFormData]("", formelement.NewButton("Cancel", false, func() (tea.Cmd, form.Action) {
					return nil, form.ActionCancel
				})),
				form.WithElement[editFormData]("", formelement.NewButton("Save", false, func() (tea.Cmd, form.Action) {
					return nil, form.ActionSubmit
				})),
			),
		)),
	}
}

// Init implements util.Model.
func (m *EditModel) Init() tea.Cmd {
	m.form.OnSubmit = func(result editFormData, err error) tea.Cmd {
		m.locked = util.NewPointer("Updating PublicKey...")
		return func() tea.Msg {
			err := m.client.UpdatePublicKey(context.Background(), m.publicKeyId, client.PublicKey{
				Comment: result.comment,
				Tags: slices.Filter(
					slices.Map(
						strings.Split(result.tags, ","),
						func(tag string) string { return strings.TrimSpace(tag) },
					),
					func(tag string) bool { return tag != "" },
				),
			})
			return editMsgUpdateResult{err}
		}
	}
	m.form.OnCancel = func() tea.Cmd {
		m.refreshForm()
		return nil
	}
	return m.load()
}

// Update implements util.Model.
func (m *EditModel) Update(msg tea.Msg) tea.Cmd {
	// Handle resizing
	if m.size.UpdateFromMsg(msg) {
		return m.form.Update(msg)
	}

	// Handle messages
	switch msg := msg.(type) {
	case editMsgLoadResult:
		m.locked = nil
		m.publicKey = msg.publicKey
		m.loadingError = msg.err
		m.refreshForm()
		return nil

	case editMsgUpdateResult:
		m.locked = nil
		m.loadingError = msg.err

	case editMsgCancel:
		m.refreshForm()

	case tea.KeyMsg:
		if !m.focussed || m.locked != nil {
			return nil
		}
		switch {
		// case key.Matches(msg, ListBaseKeyMap.Edit):
		// 	// TODO replace mock with open edit page
		// 	return m.rc.Push(util.ModelPointer(NewList(m.client, m.rc)))

		default:
			// pass key msg to form
			return m.form.Update(msg)
		}

	default:
		return nil
	}

	return nil
}

// View implements util.Model.
func (m *EditModel) View() string {
	if m.locked != nil {
		return *m.locked
	}
	if m.loadingError != nil {
		return m.loadingError.Error()
	}
	return m.form.View()
}

// Focus implements util.Model.
func (m *EditModel) Focus(baseKeyMap help.KeyMap) tea.Cmd {
	m.focussed = true
	return m.form.Focus(baseKeyMap)
}

// Blur implements util.Model.
func (m *EditModel) Blur() {
	m.focussed = false
	m.form.Blur()
}

// *EditModel implements util.Model
var _ util.Model = (*EditModel)(nil)

func (m *EditModel) load() tea.Cmd {
	m.locked = util.NewPointer("Loading PublicKey...")
	return func() tea.Msg {
		publicKey, err := m.client.GetPublicKey(context.Background(), m.publicKeyId)
		return editMsgLoadResult{publicKey, err}
	}
}

func (m *EditModel) refreshForm() error {
	return m.form.Set(editFormData{
		m.publicKey.Comment,
		strings.Join(m.publicKey.Tags, ", "),
	})
}
