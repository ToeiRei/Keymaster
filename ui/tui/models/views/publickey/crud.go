// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package publickey

import (
	"context"
	"strings"

	"github.com/bobg/go-generics/v4/slices"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/ui/tui/models/components/router"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	formelement "github.com/toeirei/keymaster/ui/tui/models/helpers/form/element"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/popup"
	"github.com/toeirei/keymaster/ui/tui/models/views/crud"
	popupviews "github.com/toeirei/keymaster/ui/tui/models/views/popup"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
	"github.com/toeirei/keymaster/util/slicest"
)

type createFormData struct {
	Algorithm string `form:"algorithm"`
	Data      string `form:"data"`
	Comment   string `form:"comment"`
	Tags      string `form:"tags"`
}

type editFormData struct {
	Comment string `form:"comment"`
	Tags    string `form:"tags"`
}

type formImport struct {
	Key string `form:"key"`
}

type msgImportResult struct {
	data      string
	algorithm string
	comment   string
}

func NewCrud(c client.Client, rc router.Controll) *crud.Crud[client.PublicKey, createFormData, editFormData, client.ID, struct{}] {
	return crud.New(
		crud.Texts{"Public Key", "Public Keys"},

		func(record client.PublicKey) client.ID { return record.Id },
		func(filter struct{}) ([]client.PublicKey, error) {
			return c.ListPublicKeys(context.Background(), "")
		},
		func(id client.ID) (client.PublicKey, error) {
			return c.GetPublicKey(context.Background(), id)
		},
		func(record createFormData) (client.PublicKey, error) {
			return c.CreatePublicKey(
				context.Background(),
				record.Algorithm+" "+record.Data,
				record.Comment,
				tagsParse(record.Tags),
			)
		},
		func(id client.ID, record editFormData) (client.PublicKey, error) {
			if err := c.UpdatePublicKey(
				context.Background(),
				id,
				record.Comment,
				tagsParse(record.Tags),
			); err != nil {
				return client.PublicKey{}, err
			}
			return c.GetPublicKey(context.Background(), id)
		},
		func(id client.ID) error {
			return c.DeletePublicKeys(context.Background(), id)
		},
		func(record []client.PublicKey, width int) ([]table.Column, []table.Row) {
			commentWidth := slicest.Reduce(record, func(k client.PublicKey, w int) int { return max(w, len(k.Comment)) })
			algorithmWidth := slicest.Reduce(record, func(k client.PublicKey, w int) int { return max(w, len(k.Algorithm)) })
			tagsWidth := slicest.Reduce(record, func(k client.PublicKey, w int) int { return max(w, len(strings.Join(k.Tags, ", "))) })
			// tags take 50% screen max
			tagsWidth = min((width-6)/2, tagsWidth)

			remainingWidth := width - 6 - algorithmWidth - commentWidth - tagsWidth

			columns := []table.Column{
				{Title: "Comment", Width: commentWidth + remainingWidth/3},
				{Title: "Algorithm", Width: algorithmWidth + remainingWidth/3},
				{Title: "Tags", Width: tagsWidth + remainingWidth/3},
			}

			rows := slices.Map(record, func(publicKey client.PublicKey) table.Row {
				return table.Row{
					// column: Comment
					publicKey.Comment,
					// column: Algorithm
					publicKey.Algorithm,
					// column: Tags
					strings.Join(publicKey.Tags, ", "),
				}
			})

			return columns, rows
		},
		func(record client.PublicKey) editFormData {
			return editFormData{
				record.Comment,
				tagsStringify(record.Tags),
			}
		},

		func() []form.FormOpt[createFormData] {
			return []form.FormOpt[createFormData]{
				form.WithRowItem[createFormData]("_import", formelement.NewButton("Import", formelement.WithButtonAction(func() (tea.Cmd, form.Action) {
					return popupviews.OpenForm(form.New(
						form.WithRowItem[formImport]("key", formelement.NewText("", "")),
						form.WithRow(
							form.WithItem[formImport]("_cancel", formelement.NewButton("Cancel",
								formelement.WithButtonActionCancel(),
								formelement.WithButtonGlobalKeyBindings(keys.Cancel()),
							)),
							form.WithItem[formImport]("_import", formelement.NewButton("Import", formelement.WithButtonActionSubmit())),
						),
						form.WithOnCancel[formImport](func() tea.Cmd { return popup.Close() }),
						form.WithOnSubmit(func(result formImport, err error) tea.Cmd {
							if err != nil {
								return popupviews.OpenMessage(popupviews.MessageError, err.Error(), nil)
							}

							var data, algorithm, comment string
							// TODO parse result.key... using this mock for now:
							{
								parts := strings.Split(result.Key, " ")

								if len(parts) < 2 || len(parts) > 3 {
									return popupviews.OpenMessage(popupviews.MessageError, "unable to parse public key", nil)
								}

								algorithm, data = parts[0], parts[1]
								if len(parts) == 3 {
									comment = parts[2]
								}
							}

							return tea.Sequence(popup.Close(), func() tea.Msg { return msgImportResult{data, algorithm, comment} })
						}),
					), 50, 20,
					), form.ActionNone
				}))),
				form.WithRowItem[createFormData]("comment", formelement.NewText("Comment", "comment that will also be deployed to authorized_keys file")),
				form.WithRowItem[createFormData]("algorithm", formelement.NewText("Algorithm", "public key algorithm")),
				form.WithRowItem[createFormData]("data", formelement.NewText("Data", "public key content")),
				form.WithRowItem[createFormData]("tags", formelement.NewText("Tags", "comma seperated list of tags")),
			}
		},
		func() []form.FormOpt[editFormData] {
			return []form.FormOpt[editFormData]{
				form.WithRowItem[editFormData]("comment", formelement.NewText("Comment", "comment that will also be deployed to authorized_keys file")),
				form.WithRowItem[editFormData]("tags", formelement.NewText("Tags", "comma seperated list of tags")),
			}
		},

		rc,
		crud.WithListKeyBindings[client.PublicKey, createFormData, editFormData, client.ID, struct{}](keys.Duplicate()),
		crud.WithListMsgInterceptor(func(msg tea.Msg, ctx crud.ListMsgInterceptorCtx[client.PublicKey, createFormData, editFormData, client.ID, struct{}]) (tea.Cmd, bool) {
			if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, keys.Duplicate()) {
				if ctx.SelectedRecord == nil {
					return popupviews.OpenMessage(popupviews.MessageError, "Please select a Record to duplicate.", nil), true
				}
				return ctx.Crud.OpenCreate(&createFormData{
					ctx.SelectedRecord.Algorithm,
					ctx.SelectedRecord.Data,
					ctx.SelectedRecord.Comment,
					tagsStringify(ctx.SelectedRecord.Tags),
				}), true
			}
			return nil, false
		}),
		crud.WithCreateMsgInterceptor(func(msg tea.Msg, ctx crud.CreateMsgInterceptorCtx[client.PublicKey, createFormData, editFormData, client.ID, struct{}]) (tea.Cmd, bool) {
			if msg, ok := msg.(msgImportResult); ok {

				data, _ := ctx.Form.Get()
				data.Data = msg.data
				data.Algorithm = msg.algorithm
				if msg.comment != "" {
					data.Comment = msg.comment
				}
				_ = ctx.Form.Set(data)

				return nil, true
			}
			return nil, false
		}),
	)
}
