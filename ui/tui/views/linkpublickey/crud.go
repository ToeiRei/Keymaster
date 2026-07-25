// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package linkpublickey

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/ui/i18n"
	"github.com/toeirei/keymaster/ui/tui/components/router"
	"github.com/toeirei/keymaster/ui/tui/helpers/crud"
	"github.com/toeirei/keymaster/ui/tui/helpers/form"
	formelement "github.com/toeirei/keymaster/ui/tui/helpers/form/element"
	"github.com/toeirei/keymaster/ui/tui/helpers/tablecontroll"
	"github.com/toeirei/keymaster/ui/tui/popups/selectpopup"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/util/slicest"
)

type recordT = struct {
	link    client.Link
	account client.Account
}

type recordCreateT = struct {
	Account   client.Account `form:"account"`
	ExpiresAt string         `form:"expires_at"`
}

// Only the expiry is editable: a link is identified by its (account, public
// key) pair, so changing the account means creating a different link.
type recordUpdateT = struct {
	ExpiresAt string `form:"expires_at"`
}

// recordIdT is the composite key that now identifies a link.
type recordIdT = struct {
	AccountId   client.AccountId
	PublicKeyId client.PublicKeyId
}

type filterT = struct{}

func accountToString(account client.Account) string {
	if account == util.NewZero[client.Account]() {
		return lipgloss.NewStyle().Italic(true).Render(i18n.T("link.none"))
	}
	return account.String()
}

func linkToRecord(ctx context.Context, c client.Client, link client.Link) (recordT, error) {
	account, err := c.GetAccount(ctx, link.AccountId)
	if err != nil {
		return recordT{}, err
	}

	return recordT{link, account}, nil
}

func createFormRows(c client.Client) func() []form.FormOpt[recordCreateT] {
	return func() []form.FormOpt[recordCreateT] {
		return []form.FormOpt[recordCreateT]{
			form.WithRowItem[recordCreateT]("account", formelement.NewPopup(i18n.Text("link.form.account_label"),
				func(returnValue func(value client.Account) tea.Cmd) tea.Cmd {
					return selectpopup.Open(
						i18n.Text("link.select_account"),
						func(ctx context.Context) ([]client.Account, error) { return c.ListAccounts(ctx) },
						func(r client.Account) tea.Cmd { return returnValue(r) },
						tablecontroll.New(tablecontroll.Columns[client.Account]{
							{Title: i18n.Text("account.col_username"), View: func(r client.Account) string { return r.Username }},
							{Title: i18n.Text("account.col_host"), View: func(r client.Account) string { return r.Host }},
							{Title: i18n.Text("account.col_port"), View: func(r client.Account) string { return fmt.Sprint(r.Port) }},
							{Title: i18n.Text("account.col_deploy_method"), View: func(r client.Account) string { return r.DeployMethod }},
						}),
						selectpopup.WithFilter(func(filter string, records []client.Account) []client.Account {
							return slicest.Filter(records, func(record client.Account) bool {
								return strings.Contains(record.Username, filter) ||
									strings.Contains(record.Host, filter) ||
									strings.Contains(fmt.Sprint(record.Port), filter) ||
									strings.Contains(record.DeployMethod, filter)
							})
						}),
					)
				},
				accountToString,
			)),
			form.WithRowItem[recordCreateT]("expires_at", formelement.NewText(i18n.Text("link.form.expires_at_label"), i18n.Text("link.form.expires_at_placeholder"))),
		}
	}
}

func updateFormRows() []form.FormOpt[recordUpdateT] {
	return []form.FormOpt[recordUpdateT]{
		form.WithRowItem[recordUpdateT]("expires_at", formelement.NewText(i18n.Text("link.form.expires_at_label"), i18n.Text("link.form.expires_at_placeholder"))),
	}
}

func NewCrud(c client.Client, rc router.Controll, publicKey client.PublicKey) *crud.Crud[recordT, recordCreateT, recordUpdateT, recordIdT, filterT] {
	return crud.New(
		crud.Texts{
			EntityNameSingular: func() string { return i18n.T("link.entity_singular") },
			EntityNameMultiple: func() string { return i18n.T("link.entity_plural") },
		},

		func(record recordT) recordIdT {
			return recordIdT{record.link.AccountId, record.link.PublicKeyId}
		},
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
			link, err := c.GetLink(ctx, id.AccountId, id.PublicKeyId)
			if err != nil {
				return recordT{}, err
			}

			return linkToRecord(ctx, c, link)
		},
		func(ctx context.Context, recordCreate recordCreateT) (recordT, error) {
			var record recordT
			err := c.WithTransaction(ctx, func(ctx context.Context, c client.Client) error {
				expiresAt, err := util.ParseTime(recordCreate.ExpiresAt)
				if err != nil {
					return err
				}

				link, err := c.CreateLink(ctx, recordCreate.Account.Id, publicKey.Id, expiresAt)
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
			err := c.WithTransaction(ctx, func(ctx context.Context, c client.Client) error {
				expiresAt, err := util.ParseTime(recordUpdate.ExpiresAt)
				if err != nil {
					return err
				}

				link, err := c.UpdateLink(ctx, id.AccountId, publicKey.Id, expiresAt)
				if err != nil {
					return err
				}

				record, err = linkToRecord(ctx, c, link)
				return err
			})
			return record, err
		},
		func(ctx context.Context, id recordIdT) error {
			return c.DeleteLink(ctx, id.AccountId, id.PublicKeyId)
		},

		tablecontroll.New(tablecontroll.Columns[recordT]{
			{Title: i18n.Text("link.col_account"), View: func(r recordT) string { return accountToString(r.account) }},
			{Title: i18n.Text("link.col_expires_at"), View: func(r recordT) string { return util.RenderExpiry(r.link.ExpiresAt) }},
		}).RenderBubblesTable,
		func(record recordT) recordUpdateT {
			return recordUpdateT{
				util.StringifyTimeZero(record.link.ExpiresAt),
			}
		},

		createFormRows(c),
		updateFormRows,

		rc,

		crud.WithListDuplicateAction[recordT, recordCreateT, recordUpdateT, recordIdT, filterT](func(record recordT) recordCreateT {
			return recordCreateT{
				record.account,
				util.StringifyTimeZero(record.link.ExpiresAt),
			}
		}),
		crud.WithListReloadAfterChange[recordT, recordCreateT, recordUpdateT, recordIdT, filterT](true),
	)
}
