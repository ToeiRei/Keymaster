// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package account

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/ui/tui/components/router"
	"github.com/toeirei/keymaster/ui/tui/helpers/crud"
	"github.com/toeirei/keymaster/ui/tui/helpers/form"
	formelement "github.com/toeirei/keymaster/ui/tui/helpers/form/element"
	"github.com/toeirei/keymaster/ui/tui/helpers/tablecontroll"
	"github.com/toeirei/keymaster/ui/tui/popups/messagepopup"
	"github.com/toeirei/keymaster/ui/tui/popups/selectpopup"
	"github.com/toeirei/keymaster/ui/tui/views/linkaccount"
	"github.com/toeirei/keymaster/util/slicest"
)

type recordT = struct {
	account                    client.Account
	isDirty                    bool
	activeLinkCount            int
	activeLinkedPublicKeyCount int
	totalLinkCount             int
	totalLinkedPublicKeyCount  int
}

type recordCreateT = struct {
	Username     string `form:"username"`
	Host         string `form:"host"`
	Port         string `form:"port"`
	DeployMethod string `form:"deploy_method"`
	DeploySecret string `form:"deploy_secret"`
}

type recordUpdateT = recordCreateT

type recordIdT = client.AccountId

type filterT = struct{}

func accountToRecord(ctx context.Context, c client.Client, account client.Account) (recordT, error) {
	activeLinks, err := c.ListLinksForAccount(ctx, account.Id, false)
	if err != nil {
		return recordT{}, err
	}

	activePublicKeys, err := c.ListPublicKeysLinkedToAccount(ctx, account.Id, false)
	if err != nil {
		return recordT{}, err
	}

	allLinks, err := c.ListLinksForAccount(ctx, account.Id, true)
	if err != nil {
		return recordT{}, err
	}

	allPublicKeys, err := c.ListPublicKeysLinkedToAccount(ctx, account.Id, true)
	if err != nil {
		return recordT{}, err
	}

	isDirty, err := c.IsAccountDirty(ctx, account)
	if err != nil {
		return recordT{}, err
	}

	return recordT{
		account,
		isDirty,
		len(activeLinks),
		len(activePublicKeys),
		len(allLinks),
		len(allPublicKeys),
	}, nil
}

func formRows[T comparable](c client.Client) []form.FormOpt[T] {
	return []form.FormOpt[T]{
		form.WithRowItem[T]("username", formelement.NewText("Username", "eg. user/root/...")),
		form.WithRowItem[T]("host", formelement.NewText("Host", "ip/domain to connect to")),
		form.WithRowItem[T]("port", formelement.NewText("Port", "eg. 22")),
		form.WithRowItem[T]("deploy_method", formelement.NewPopup("Deploy Method",
			func(returnValue func(value string) tea.Cmd) tea.Cmd {
				return selectpopup.Open(
					"Select Deploy Method",
					func(ctx context.Context) ([]string, error) { return c.ListConnectorKeys(ctx) },
					func(r string) tea.Cmd { return returnValue(r) },
					tablecontroll.New(tablecontroll.Columns[string]{
						{Title: func() string { return "Connector" }, View: func(r string) string { return r }},
					}),
				)
			},
			func(v string) string { return v },
		)),
		form.WithRowItem[T]("deploy_secret", formelement.NewTextarea("Deploy Secret", "", 3, 5)),
	}
}

func NewCrud(c client.Client, rc router.Controll) *crud.Crud[recordT, recordCreateT, recordUpdateT, recordIdT, filterT] {
	return crud.New(
		crud.Texts{
			EntityNameSingular: func() string { return "Account" },
			EntityNameMultiple: func() string { return "Accounts" },
		},

		func(record recordT) recordIdT { return record.account.Id },
		func(ctx context.Context, filter filterT) ([]recordT, error) {
			accounts, err := c.ListAccounts(ctx)
			if err != nil {
				return nil, err
			}

			return slicest.MapX(accounts, func(account client.Account) (recordT, error) {
				return accountToRecord(ctx, c, account)
			})
		},
		func(ctx context.Context, id recordIdT) (recordT, error) {
			account, err := c.GetAccount(ctx, id)
			if err != nil {
				return recordT{}, err
			}

			return accountToRecord(ctx, c, account)
		},
		func(ctx context.Context, recordCreate recordCreateT) (recordT, error) {
			var record recordT
			err := c.WithTransaction(ctx, func(ctx context.Context, c client.Client) error {
				port, err := strconv.Atoi(recordCreate.Port)
				if err != nil {
					return err
				}

				account, err := c.CreateAccount(
					ctx,
					recordCreate.Username,
					recordCreate.Host,
					port,
					recordCreate.DeployMethod,
					recordCreate.DeploySecret,
				)
				if err != nil {
					return err
				}

				record, err = accountToRecord(ctx, c, account)
				return err
			})
			return record, err
		},
		func(ctx context.Context, id recordIdT, recordUpdate recordUpdateT) (recordT, error) {
			var record recordT
			err := c.WithTransaction(ctx, func(ctx context.Context, c client.Client) error {
				port, err := strconv.Atoi(recordUpdate.Port)
				if err != nil {
					return err
				}

				account, err := c.UpdateAccount(
					ctx,
					id,
					recordUpdate.Username,
					recordUpdate.Host,
					port,
					recordUpdate.DeployMethod,
					recordUpdate.DeploySecret,
				)
				if err != nil {
					return err
				}

				record, err = accountToRecord(ctx, c, account)
				return err
			})
			return record, err
		},
		func(ctx context.Context, id recordIdT) error {
			return c.DeleteAccounts(ctx, id)
		},

		tablecontroll.New(tablecontroll.Columns[recordT]{
			{Title: func() string { return "Username" }, View: func(r recordT) string { return r.account.Username }},
			{Title: func() string { return "Host" }, View: func(r recordT) string { return r.account.Host }},
			{Title: func() string { return "Port" }, View: func(r recordT) string { return fmt.Sprint(r.account.Port) }},
			{Title: func() string { return "Deploy Method" }, View: func(r recordT) string { return r.account.DeployMethod }},
			{Title: func() string { return "Dirty" }, View: func(r recordT) string { return fmt.Sprint(r.isDirty) }},
			{Title: func() string { return "Links (active/total)" }, View: func(r recordT) string {
				return fmt.Sprintf("%d/%d", r.activeLinkCount, r.totalLinkCount)
			}},
			{Title: func() string { return "Public Keys (active/total)" }, View: func(r recordT) string {
				return fmt.Sprintf("%d/%d", r.activeLinkedPublicKeyCount, r.totalLinkedPublicKeyCount)
			}},
		}).RenderBubblesTable,
		func(record recordT) recordUpdateT {
			return recordUpdateT{
				record.account.Username,
				record.account.Host,
				fmt.Sprint(record.account.Port),
				record.account.DeployMethod,
				record.account.DeploySecret,
			}
		},

		func() []form.FormOpt[recordCreateT] { return formRows[recordCreateT](c) },
		func() []form.FormOpt[recordUpdateT] { return formRows[recordUpdateT](c) },

		rc,

		crud.WithCreateRecordPreset[recordT, recordCreateT, recordUpdateT, recordIdT, filterT](
			func() recordCreateT {
				connectorKeys, err := c.ListConnectorKeys(context.Background())
				if err != nil || !slices.Contains(connectorKeys, "ssh") {
					return recordCreateT{}
				}
				return recordCreateT{DeployMethod: "ssh"}
			}),

		crud.WithListDuplicateAction[recordT, recordCreateT, recordUpdateT, recordIdT, filterT](func(record recordT) recordCreateT {
			return recordCreateT{
				record.account.Username,
				record.account.Host,
				fmt.Sprint(record.account.Port),
				record.account.DeployMethod,
				record.account.DeploySecret,
			}
		}),
		crud.WithListAction(
			func(ctx crud.ListMsgInterceptorCtx[recordT, recordCreateT, recordUpdateT, recordIdT, filterT]) tea.Cmd {
				if ctx.SelectedRecord == nil {
					return messagepopup.Open(messagepopup.Error, "Please select a "+ctx.Crud.Texts.EntityNameSingular()+".", nil)
				}

				ctx.Crud.ReloadOnNextFocus = true
				return linkaccount.NewCrud(c, rc, ctx.SelectedRecord.account).OpenList()
			},
			key.NewBinding(
				key.WithKeys("l"),
				key.WithHelp("l", "links"),
			),
		),
		crud.WithListReloadAfterChange[recordT, recordCreateT, recordUpdateT, recordIdT, filterT](true),
	)
}
