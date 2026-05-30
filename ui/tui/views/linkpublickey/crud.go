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
	"github.com/toeirei/keymaster/tags"
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
	link                 client.Link
	account              client.Account
	linkedPublicKeyCount int
}

type recordCreateT = struct {
	Account    client.Account `form:"account"`
	TagMatcher string         `form:"tag_matcher"`
	ExpiresAt  string         `form:"expires_at"`
}

type recordUpdateT = recordCreateT

type recordIdT = client.LinkId

type filterT = struct{}

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

func formRows[T comparable](c client.Client) func() []form.FormOpt[T] {
	return func() []form.FormOpt[T] {
		return []form.FormOpt[T]{
			form.WithRowItem[T]("account", formelement.NewPopup("Account",
				func(returnValue func(value client.Account) tea.Cmd) tea.Cmd {
					return selectpopup.Open(
						"Select Account",
						// load Accounts
						func(ctx context.Context) ([]client.Account, error) { return c.ListAccounts(ctx) },
						// return selected Account
						func(r client.Account) tea.Cmd { return returnValue(r) },
						// display Accounts
						tablecontroll.New(tablecontroll.Columns[client.Account]{
							{Title: func() string { return "Username" }, View: func(r client.Account) string { return r.Username }},
							{Title: func() string { return "Host" }, View: func(r client.Account) string { return r.Host }},
							{Title: func() string { return "Port" }, View: func(r client.Account) string { return fmt.Sprint(r.Port) }},
							{Title: func() string { return "Deploy Method" }, View: func(r client.Account) string { return r.DeployMethod }},
						}),
						// extra options
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
				func(value client.Account) string {
					if value == util.NewZero[client.Account]() {
						return lipgloss.NewStyle().Italic(true).Render("none")
					}
					return value.String()
				},
			)),
			form.WithRowItem[T]("tag_matcher", formelement.NewText("Tag Matcher", "text to match tags of public keys")),
			form.WithRowItem[T]("expires_at", formelement.NewText("Expires At", "date on witch this link will expire and its public keys will loose access")),
		}
	}
}

func NewCrud(c client.Client, rc router.Controll, publicKey client.PublicKey) *crud.Crud[recordT, recordCreateT, recordUpdateT, recordIdT, filterT] {
	return crud.New(
		crud.Texts{
			EntityNameSingular: func() string { return "Link" },
			EntityNameMultiple: func() string { return "Links" },
		},

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

				link, err := c.CreateLink(ctx, recordCreate.Account.Id, expr.String(), expiresAt)
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
					recordUpdate.Account.Id,
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
			{Title: func() string { return "Tag Matcher" }, View: func(r recordT) string { return r.link.TagMatcher }},
			{Title: func() string { return "Expires At" }, View: func(r recordT) string { return util.StringifyTime(r.link.ExpiresAt) }},
			{Title: func() string { return "Account" }, View: func(r recordT) string { return r.account.String() }},
			{Title: func() string { return "Public Keys" }, View: func(r recordT) string { return fmt.Sprint(r.linkedPublicKeyCount) }},
		}).RenderBubblesTable,
		func(record recordT) recordUpdateT {
			return recordUpdateT{
				record.account,
				record.link.TagMatcher,
				util.StringifyTime(record.link.ExpiresAt),
			}
		},

		formRows[recordCreateT](c),
		formRows[recordUpdateT](c),

		rc,

		crud.WithListDuplicateAction[recordT, recordCreateT, recordUpdateT, recordIdT, filterT](func(record recordT) recordCreateT {
			return recordCreateT{
				record.account,
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
	)
}
