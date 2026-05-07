// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package linkpublickey

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/tags"
	"github.com/toeirei/keymaster/ui/tui/models/components/router"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/crud"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	formelement "github.com/toeirei/keymaster/ui/tui/models/helpers/form/element"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/tablecontroll"
	popupviews "github.com/toeirei/keymaster/ui/tui/models/views/popup"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/util/slicest"
)

type recordT = struct {
	link                 client.Link
	account              client.Account
	linkedPublicKeyCount int
}

type recordCreateT = struct {
	AccountId   client.AccountId `form:"account_id"`
	AccountName string           `form:"account_name"`
	TagMatcher  string           `form:"tag_matcher"`
	ExpiresAt   string           `form:"expires_at"`
}

type recordUpdateT = recordCreateT

type recordIdT = client.LinkId

type filterT = struct{}

type accountSelectedMsg struct {
	account client.Account
}

func linkToRecord(ctx context.Context, c client.Client, link client.Link) (recordT, error) {
	account, err := c.GetAccount(ctx, link.AccountId)
	if err != nil {
		return recordT{}, err
	}

	publicKeys, err := c.ListPublicKeys(ctx, link.TagMatcher)
	if err != nil {
		return recordT{}, err
	}

	return recordT{link, account, len(publicKeys)}, nil
}

// func resolveAccountByName(ctx context.Context, c client.Client, accountName string) (client.Account, error) {
// 	accounts, err := c.ListAccounts(ctx)
// 	if err != nil {
// 		return client.Account{}, err
// 	}

// 	account := slicest.Find(accounts, func(account client.Account) bool { return account.String() == accountName })
// 	if account == nil {
// 		return client.Account{}, fmt.Errorf(`Account with "protocol user@host:port" = %q does not exist`, accountName)
// 	}

// 	return *account, nil
// }

func formRows[T comparable](c client.Client) func() []form.FormOpt[T] {
	return func() []form.FormOpt[T] {
		return []form.FormOpt[T]{
			form.WithRowItem[T]("account_id", formelement.NewInternalValue()),
			form.WithRow(
				form.WithItem[T]("_select_account", formelement.NewButton("Select Account", formelement.WithButtonAction(func() (tea.Cmd, form.Action) {
					return popupviews.OpenSelect(
						"Select Account",
						func(ctx context.Context) ([]client.Account, error) {
							return c.ListAccounts(ctx)
						},
						func(r client.Account) tea.Cmd {
							return util.TeaMsgToCmd(accountSelectedMsg{r})
						},
						tablecontroll.New(tablecontroll.Columns[client.Account]{
							{Title: "Username", View: func(r client.Account) string { return r.Username }},
							{Title: "Host", View: func(r client.Account) string { return r.Host }},
							{Title: "Port", View: func(r client.Account) string { return fmt.Sprint(r.Port) }},
							{Title: "Deploy Method", View: func(r client.Account) string { return r.DeployMethod }},
						}),
						popupviews.WithSelectFilter(func(filter string, records []client.Account) []client.Account {
							return slicest.Filter(records, func(record client.Account) bool {
								return strings.Contains(record.Username, filter) ||
									strings.Contains(record.Host, filter) ||
									strings.Contains(fmt.Sprint(record.Port), filter) ||
									strings.Contains(record.DeployMethod, filter)
							})
						}),
					), form.ActionNone
				}))),
				form.WithItem[T]("account_name", formelement.NewText(
					"Account",
					"not selected",
					formelement.WithTextDisable(),
				)),
				form.WithAlign[T](form.Left),
			),
			form.WithRowItem[T]("tag_matcher", formelement.NewText("Tag Matcher", "text to match tags of public keys")),
			form.WithRowItem[T]("expires_at", formelement.NewText("Expires At", "date on witch this link will expire and its public keys will loose access")),
		}
	}
}

func NewCrud(c client.Client, rc router.Controll, publicKey client.PublicKey) *crud.Crud[recordT, recordCreateT, recordUpdateT, recordIdT, filterT] {
	return crud.New(
		crud.Texts{"Link", "Links"},

		func(record recordT) recordIdT { return record.link.Id },
		func(ctx context.Context, filter filterT) ([]recordT, error) {
			links, err := c.ListLinksForPublicKey(ctx, publicKey.Id, true)
			if err != nil {
				return nil, err
			}

			return slicest.MapX(links, func(link client.Link) (recordT, error) {
				return linkToRecord(ctx, c, link)
			})
		},
		func(ctx context.Context, id recordIdT) (recordT, error) {
			link, err := c.GetLink(ctx, id)
			if err != nil {
				return recordT{}, err
			}

			return linkToRecord(ctx, c, link)
		},
		func(ctx context.Context, recordCreate recordCreateT) (recordT, error) {
			var record recordT
			err := c.WithTransaction(ctx, func(c client.Client) error {
				expr, err := tags.ParseMatcher(recordCreate.TagMatcher)
				if err != nil {
					return err
				}

				expiresAt, err := util.ParseTime(recordCreate.ExpiresAt)
				if err != nil {
					return err
				}

				link, err := c.CreateLink(ctx, recordCreate.AccountId, expr.String(), expiresAt)
				if err != nil {
					return err
				}

				record, err = linkToRecord(ctx, c, link)
				return err
			})
			return record, err
		},
		func(ctx context.Context, id recordIdT, recordUpdate recordUpdateT) (recordT, error) {
			var record recordT
			err := c.WithTransaction(ctx, func(c client.Client) error {
				expr, err := tags.ParseMatcher(recordUpdate.TagMatcher)
				if err != nil {
					return err
				}

				expiresAt, err := util.ParseTime(recordUpdate.ExpiresAt)
				if err != nil {
					return err
				}

				link, err := c.UpdateLink(
					ctx,
					id,
					recordUpdate.AccountId,
					expr.String(),
					expiresAt,
				)
				if err != nil {
					return err
				}

				record, err = linkToRecord(ctx, c, link)
				return err
			})
			return record, err
		},
		func(ctx context.Context, id recordIdT) error {
			return c.DeleteLinks(ctx, id)
		},

		tablecontroll.New(tablecontroll.Columns[recordT]{
			{Title: "Tag Matcher", View: func(r recordT) string { return r.link.TagMatcher }},
			{Title: "Expires At", View: func(r recordT) string { return util.StringifyTime(r.link.ExpiresAt) }},
			{Title: "Account", View: func(r recordT) string { return r.account.String() }},
			{Title: "Public Keys", View: func(r recordT) string { return fmt.Sprint(r.linkedPublicKeyCount) }},
		}).RenderBubblesTable,
		func(record recordT) recordUpdateT {
			return recordUpdateT{
				record.account.Id,
				record.account.String(),
				record.link.TagMatcher,
				util.StringifyTime(record.link.ExpiresAt),
			}
		},

		formRows[recordCreateT](c),
		formRows[recordUpdateT](c),

		rc,

		crud.WithListDuplicateAction[recordT, recordCreateT, recordUpdateT, recordIdT, filterT](func(record recordT) recordCreateT {
			return recordCreateT{
				record.account.Id,
				record.account.String(),
				record.link.TagMatcher,
				util.StringifyTime(record.link.ExpiresAt),
			}
		}),
		crud.WithListReloadAfterChange[recordT, recordCreateT, recordUpdateT, recordIdT, filterT](true),
		crud.WithCreateRecordPreset[recordT, recordCreateT, recordUpdateT, recordIdT, filterT](func() recordCreateT {
			return recordCreateT{
				TagMatcher: strings.Join(publicKey.Tags.Slice(), " & "),
			}
		}),
		crud.WithCreateMsgInterceptor(func(msg tea.Msg, ctx crud.CreateMsgInterceptorCtx[recordT, recordCreateT, recordUpdateT, recordIdT, filterT]) (cmd tea.Cmd, done bool) {
			if msg, ok := msg.(accountSelectedMsg); ok {
				done = true
				ctx.Form.SetItem("account_id", msg.account.Id)
				ctx.Form.SetItem("account_name", msg.account.String())
			}
			return
		}),
		crud.WithUpdateMsgInterceptor(func(msg tea.Msg, ctx crud.UpdateMsgInterceptorCtx[recordT, recordCreateT, recordUpdateT, recordIdT, filterT]) (cmd tea.Cmd, done bool) {
			if msg, ok := msg.(accountSelectedMsg); ok {
				done = true
				ctx.Form.SetItem("account_id", msg.account.Id)
				ctx.Form.SetItem("account_name", msg.account.String())
			}
			return
		}),
	)
}
