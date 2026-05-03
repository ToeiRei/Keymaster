// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package publickey

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/tags"
	"github.com/toeirei/keymaster/ui/tui/models/components/router"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/crud"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	formelement "github.com/toeirei/keymaster/ui/tui/models/helpers/form/element"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/popup"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/table"
	"github.com/toeirei/keymaster/ui/tui/models/views/linkpublickey"
	popupviews "github.com/toeirei/keymaster/ui/tui/models/views/popup"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
	"github.com/toeirei/keymaster/util/slicest"
)

type recordT = struct {
	publicKey                client.PublicKey
	activeLinkCount          int
	activeLinkedAccountCount int
	totalLinkCount           int
	totalLinkedAccountCount  int
}

type recordCreateT = struct {
	Algorithm string `form:"algorithm"`
	Data      string `form:"data"`
	Comment   string `form:"comment"`
	Tags      string `form:"tags"`
}

type recordUpdateT struct {
	Comment string `form:"comment"`
	Tags    string `form:"tags"`
}

type recordIdT = client.PublicKeyId

type filterT = struct{}

type importForm struct {
	Key string `form:"key"`
}

type importMsg struct {
	data      string
	algorithm string
	comment   string
}

func publicKeyToRecord(ctx context.Context, c client.Client, publicKey client.PublicKey) (recordT, error) {
	activeLinks, err := c.ListLinksForPublicKey(ctx, publicKey.Id, false)
	if err != nil {
		return recordT{}, err
	}

	activeAccounts, err := c.ListAccountsLinkedToPublicKey(ctx, publicKey.Id, false)
	if err != nil {
		return recordT{}, err
	}

	allLinks, err := c.ListLinksForPublicKey(ctx, publicKey.Id, true)
	if err != nil {
		return recordT{}, err
	}

	allAccounts, err := c.ListAccountsLinkedToPublicKey(ctx, publicKey.Id, true)
	if err != nil {
		return recordT{}, err
	}

	return recordT{
		publicKey,
		len(activeLinks),
		len(activeAccounts),
		len(allLinks),
		len(allAccounts),
	}, nil
}

func NewCrud(c client.Client, rc router.Controll) *crud.Crud[recordT, recordCreateT, recordUpdateT, recordIdT, filterT] {
	return crud.New(
		crud.Texts{"Public Key", "Public Keys"},

		func(record recordT) recordIdT { return record.publicKey.Id },
		func(ctx context.Context, filter filterT) ([]recordT, error) {
			publicKeys, err := c.ListPublicKeys(ctx, "")
			if err != nil {
				return nil, err
			}

			return slicest.MapX(publicKeys, func(publicKey client.PublicKey) (recordT, error) {
				return publicKeyToRecord(ctx, c, publicKey)
			})
		},
		func(ctx context.Context, id recordIdT) (recordT, error) {
			publicKey, err := c.GetPublicKey(ctx, id)
			if err != nil {
				return recordT{}, err
			}

			return publicKeyToRecord(ctx, c, publicKey)
		},
		func(ctx context.Context, recordCreate recordCreateT) (recordT, error) {
			var record recordT
			err := c.WithTransaction(ctx, func(c client.Client) error {
				publicKey, err := c.CreatePublicKey(
					ctx,
					recordCreate.Algorithm+" "+recordCreate.Data,
					recordCreate.Comment,
					tags.Parse(recordCreate.Tags),
				)
				if err != nil {
					return err
				}

				record, err = publicKeyToRecord(ctx, c, publicKey)
				return err
			})
			return record, err
		},
		func(ctx context.Context, id recordIdT, recordCreate recordUpdateT) (recordT, error) {
			var record recordT
			err := c.WithTransaction(ctx, func(c client.Client) error {
				if err := c.UpdatePublicKey(
					ctx,
					id,
					recordCreate.Comment,
					tags.Parse(recordCreate.Tags),
				); err != nil {
					return err
				}

				publicKey, err := c.GetPublicKey(ctx, id)
				if err != nil {
					return err
				}

				record, err = publicKeyToRecord(ctx, c, publicKey)
				return err
			})
			return record, err
		},
		func(ctx context.Context, id recordIdT) error {
			return c.DeletePublicKeys(ctx, id)
		},

		table.NewBubblesTableRenderer(table.Columns[recordT]{
			{Title: "Comment", View: func(r recordT) string { return r.publicKey.Comment }},
			{Title: "Tags", View: func(r recordT) string { return r.publicKey.Tags.String() }, MaxWidth: 0.5},
			{Title: "Algorithm", View: func(r recordT) string { return r.publicKey.Algorithm }},
			{Title: "Links (active/total)", View: func(r recordT) string {
				return fmt.Sprintf("%d/%d", r.activeLinkCount, r.totalLinkCount)
			}},
			{Title: "Accounts (active/total)", View: func(r recordT) string {
				return fmt.Sprintf("%d/%d", r.activeLinkedAccountCount, r.totalLinkedAccountCount)
			}},
		}),
		func(record recordT) recordUpdateT {
			return recordUpdateT{
				record.publicKey.Comment,
				record.publicKey.Tags.String(),
			}
		},

		func() []form.FormOpt[recordCreateT] {
			return []form.FormOpt[recordCreateT]{
				form.WithRowItem[recordCreateT]("_import", formelement.NewButton("Import", formelement.WithButtonAction(func() (tea.Cmd, form.Action) {
					return popupviews.OpenForm(form.New(
						form.WithRowItem[importForm]("key", formelement.NewText("", "")),
						form.WithRow(
							form.WithItem[importForm]("_cancel", formelement.NewButton("Cancel",
								formelement.WithButtonActionCancel(),
								formelement.WithButtonGlobalKeyBindings(keys.Cancel()),
							)),
							form.WithItem[importForm]("_import", formelement.NewButton("Import", formelement.WithButtonActionSubmit())),
						),
						form.WithOnCancel[importForm](func() tea.Cmd { return popup.Close() }),
						form.WithOnSubmit(func(result importForm, err error) (tea.Cmd, bool) {
							if err != nil {
								return popupviews.OpenMessage(popupviews.MessageError, err.Error(), nil), false
							}

							var data, algorithm, comment string
							// TODO parse result.key... using this mock for now:
							{
								parts := strings.Split(result.Key, " ")

								if len(parts) < 2 || len(parts) > 3 {
									return popupviews.OpenMessage(popupviews.MessageError, "unable to parse public key", nil), false
								}

								algorithm, data = parts[0], parts[1]
								if len(parts) == 3 {
									comment = parts[2]
								}
							}

							return tea.Sequence(popup.Close(), util.TeaMsgToCmd(importMsg{data, algorithm, comment})), true
						}),
					)), form.ActionNone
				}))),
				form.WithRowItem[recordCreateT]("comment", formelement.NewText("Comment", "comment that will also be deployed to authorized_keys file")),
				form.WithRowItem[recordCreateT]("algorithm", formelement.NewText("Algorithm", "public key algorithm")),
				form.WithRowItem[recordCreateT]("data", formelement.NewText("Data", "public key content")),
				form.WithRowItem[recordCreateT]("tags", formelement.NewText("Tags", "comma seperated list of tags")),
			}
		},
		func() []form.FormOpt[recordUpdateT] {
			return []form.FormOpt[recordUpdateT]{
				form.WithRowItem[recordUpdateT]("comment", formelement.NewText("Comment", "comment that will also be deployed to authorized_keys file")),
				form.WithRowItem[recordUpdateT]("tags", formelement.NewText("Tags", "comma seperated list of tags")),
			}
		},

		rc,

		crud.WithListDuplicateAction[recordT, recordCreateT, recordUpdateT, recordIdT, filterT](func(record recordT) recordCreateT {
			return recordCreateT{
				record.publicKey.Algorithm,
				record.publicKey.Data,
				record.publicKey.Comment,
				record.publicKey.Tags.String(),
			}
		}),
		crud.WithListAction(
			func(ctx crud.ListMsgInterceptorCtx[recordT, recordCreateT, recordUpdateT, recordIdT, filterT]) tea.Cmd {
				if ctx.SelectedRecord == nil {
					return popupviews.OpenMessage(popupviews.MessageError, "Please select a "+ctx.Crud.Texts.EntityNameSingular+".", nil)
				}

				ctx.Crud.ReloadOnNextFocus = true
				return linkpublickey.NewCrud(c, rc, ctx.SelectedRecord.publicKey).OpenList()
			},
			key.NewBinding(
				key.WithKeys("l"),
				key.WithHelp("l", "links"),
			),
		),
		crud.WithCreateMsgInterceptor(func(msg tea.Msg, ctx crud.CreateMsgInterceptorCtx[recordT, recordCreateT, recordUpdateT, recordIdT, filterT]) (tea.Cmd, bool) {
			if msg, ok := msg.(importMsg); ok {
				// apply import popup result to form
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
		crud.WithListReloadAfterChange[recordT, recordCreateT, recordUpdateT, recordIdT, filterT](true),
	)
}
