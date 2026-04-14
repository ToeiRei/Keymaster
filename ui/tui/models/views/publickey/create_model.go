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
	"github.com/toeirei/keymaster/ui/tui/models/helpers/popup"
	popupviews "github.com/toeirei/keymaster/ui/tui/models/views/popup"
	"github.com/toeirei/keymaster/ui/tui/util"
)

type createFormData struct {
	Data      string `form:"data"`
	Algorithm string `form:"algorithm"`
	Comment   string `form:"comment"`
	Tags      string `form:"tags"`
}

type createFormImport struct {
	Key string `form:"key"`
}

type CreateModel struct {
	// state
	publicKey client.PublicKey
	locked    *string
	focussed  bool

	// util
	client client.Client
	rc     router.Controll
	size   util.Size

	// sub models
	form *form.Form[createFormData]
}

func NewCreate(c client.Client, rc router.Controll) *CreateModel {
	return &CreateModel{
		client: c,
		rc:     rc,
	}
}

// Init implements util.Model.
func (m *CreateModel) Init() tea.Cmd {
	m.form = util.NewPointer(form.New(
		form.WithRowItem[createFormData]("_import", formelement.NewButton("Import", formelement.WithButtonAction(func() (tea.Cmd, form.Action) {
			return popupviews.OpenForm(form.New(
				form.WithRowItem[createFormImport]("key", formelement.NewText("", "")),
				form.WithRow(
					form.WithItem[createFormImport]("_cancel", formelement.NewButton("Cancel", formelement.WithButtonActionCancel())),
					form.WithItem[createFormImport]("_import", formelement.NewButton("Import", formelement.WithButtonActionSubmit())),
				),
				form.WithOnCancel[createFormImport](func() tea.Cmd { return popup.Close() }),
				form.WithOnSubmit(func(result createFormImport, err error) tea.Cmd {
					if err != nil {
						return popupviews.OpenMessage(
							popupviews.MessageError,
							err.Error(),
							nil,
							50, 20,
						)
					}

					// TODO parse result.key
					parts := strings.Split(result.Key, " ")

					if len(parts) < 2 || len(parts) > 3 {
						return popupviews.OpenMessage(
							popupviews.MessageError,
							"unable to parse public key",
							nil,
							50, 20,
						)
					}

					var data, algorithm, comment string
					data, algorithm = parts[0], parts[1]
					if len(parts) == 3 {
						comment = parts[2]
					}

					return tea.Sequence(popup.Close(), func() tea.Msg { return createMsgImportResult{data, algorithm, comment} })
				}),
			), 50, 20,
			), form.ActionNone
		}))),
		form.WithRowItem[createFormData]("data", formelement.NewText("Data", "public key content")),
		form.WithRowItem[createFormData]("algorithm", formelement.NewText("Algorithm", "public key algorithm")),
		form.WithRowItem[createFormData]("comment", formelement.NewText("Comment", "comment that will also be deployed to authorized_keys file")),
		form.WithRowItem[createFormData]("tags", formelement.NewText("Tags", "comma seperated list of tags")),
		form.WithRow(
			form.WithItem[createFormData]("_reset", formelement.NewButton("Reset", formelement.WithButtonActionReset())),
			form.WithItem[createFormData]("_cancel", formelement.NewButton("Cancel", formelement.WithButtonActionCancel())),
			form.WithItem[createFormData]("_save", formelement.NewButton("Save", formelement.WithButtonActionSubmit())),
		),
		form.WithOnSubmit(func(result createFormData, err error) tea.Cmd {
			m.locked = util.NewPointer("Creating PublicKey...")
			return func() tea.Msg {
				publicKey, err := m.client.CreatePublicKey(
					context.Background(),
					result.Algorithm+" "+result.Data,
					result.Comment,
					slices.Filter( // remove empty user provided tags
						slices.Map( // trim user provided tags
							strings.Split(result.Tags, ","), // split user provided tags
							func(tag string) string { return strings.TrimSpace(tag) },
						),
						func(tag string) bool { return tag != "" },
					),
				)

				return createMsgCreateResult{publicKey.Id, err}
			}
		}),
		form.WithOnCancel[createFormData](func() tea.Cmd {
			return m.rc.Pop(1)
		}),
	))

	return nil
}

// Update implements util.Model.
func (m *CreateModel) Update(msg tea.Msg) tea.Cmd {
	// Handle resizing
	if m.size.UpdateFromMsg(msg) {
		return m.form.Update(msg)
	}

	// Handle messages
	switch msg := msg.(type) {
	case createMsgImportResult:
		data, _ := m.form.Get()

		// apply import result
		data.Data = msg.data
		data.Algorithm = msg.algorithm
		if msg.comment != "" {
			data.Comment = msg.comment
		}

		_ = m.form.Set(data)

	case createMsgCreateResult:
		m.locked = nil
		if msg.err != nil {
			// TODO open popup displaying error
			return nil
		}
		return tea.Sequence(m.rc.Pop(1), func() tea.Msg { return CreateMsgCreated{msg.publicKeyId} })

	case tea.KeyMsg:
		if !m.focussed || m.locked != nil {
			return nil
		}
		switch {
		// case key.Matches(msg, ListBaseKeyMap.Create):
		// 	// TODO replace mock with open create page
		// 	return m.rc.Push(util.ModelPointer(NewList(m.client, m.rc)))

		default:
			// pass key msg to form
			return m.form.Update(msg)
		}

	}

	return nil
}

// View implements util.Model.
func (m *CreateModel) View() string {
	if m.locked != nil {
		return *m.locked
	}
	return m.form.View()
}

// Focus implements util.Model.
func (m *CreateModel) Focus(baseKeyMap help.KeyMap) tea.Cmd {
	m.focussed = true
	return m.form.Focus(baseKeyMap)
}

// Blur implements util.Model.
func (m *CreateModel) Blur() {
	m.focussed = false
	m.form.Blur()
}

// *[CreateModel] implements [util.Model]
var _ util.Model = (*CreateModel)(nil)
