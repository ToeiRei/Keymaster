// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package linkaccount

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
	link      client.Link
	publicKey client.PublicKey
}

type recordCreateT = struct {
	PublicKey client.PublicKey `form:"public_key"`
	ExpiresAt string           `form:"expires_at"`
}

// Only the expiry is editable: a link is identified by its (account, public
// key) pair, so changing the key means creating a different link.
type recordUpdateT = struct {
	ExpiresAt string `form:"expires_at"`
}

// recordIdT is the composite key that now identifies a link.
type recordIdT = struct {
	AccountId   client.AccountId
	PublicKeyId client.PublicKeyId
}

type filterT = struct{}

func publicKeyToString(publicKey client.PublicKey) string {
	if publicKey == util.NewZero[client.PublicKey]() {
		return lipgloss.NewStyle().Italic(true).Render(i18n.T("link.none"))
	}
	if publicKey.Comment != "" {
		return fmt.Sprintf("%s (%s)", publicKey.Comment, publicKey.Algorithm)
	}
	return publicKey.Algorithm
}

func linkToRecord(ctx context.Context, c client.Client, link client.Link) (recordT, error) {
	publicKey, err := c.GetPublicKey(ctx, link.PublicKeyId)
	if err != nil {
		return recordT{}, err
	}

	return recordT{link, publicKey}, nil
}

func createFormRows(c client.Client) func() []form.FormOpt[recordCreateT] {
	return func() []form.FormOpt[recordCreateT] {
		return []form.FormOpt[recordCreateT]{
			form.WithRowItem[recordCreateT]("public_key", formelement.NewPopup(i18n.Text("link.form.public_key_label"),
				func(_ client.PublicKey, returnValue func(value client.PublicKey) tea.Cmd) tea.Cmd {
					return selectpopup.Open(
						i18n.Text("link.select_public_key"),
						func(ctx context.Context) ([]client.PublicKey, error) { return c.ListPublicKeys(ctx) },
						func(r client.PublicKey) tea.Cmd { return returnValue(r) },
						tablecontroll.New(tablecontroll.Columns[client.PublicKey]{
							{Title: i18n.Text("public_key.col_comment"), View: func(r client.PublicKey) string { return r.Comment }},
							{Title: i18n.Text("public_key.col_algorithm"), View: func(r client.PublicKey) string { return r.Algorithm }},
						}),
						selectpopup.WithFilter(func(filter string, records []client.PublicKey) []client.PublicKey {
							return slicest.Filter(records, func(record client.PublicKey) bool {
								return strings.Contains(record.Comment, filter) ||
									strings.Contains(record.Algorithm, filter)
							})
						}),
					)
				},
				publicKeyToString,
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

func NewCrud(c client.Client, rc router.Controll, account client.Account) *crud.Crud[recordT, recordCreateT, recordUpdateT, recordIdT, filterT] {
	return crud.New(
		crud.Texts{
			i18n.Text("link.entity_singular"),
			i18n.Text("link.entity_plural"),
		},

		func(record recordT) recordIdT {
			return recordIdT{record.link.AccountId, record.link.PublicKeyId}
		},
		func(ctx context.Context, filter filterT) ([]recordT, error) {
			links, err := c.ListLinksForAccount(ctx, account.Id, true)
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

				link, err := c.CreateLink(
					ctx,
					account.Id,
					recordCreate.PublicKey.Id,
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
		func(ctx context.Context, id recordIdT, recordUpdate recordUpdateT) (recordT, error) {
			var record recordT
			err := c.WithTransaction(ctx, func(ctx context.Context, c client.Client) error {
				expiresAt, err := util.ParseTime(recordUpdate.ExpiresAt)
				if err != nil {
					return err
				}

				link, err := c.UpdateLink(ctx, id.AccountId, id.PublicKeyId, expiresAt)
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
			{Title: i18n.Text("link.col_public_key"), View: func(r recordT) string { return publicKeyToString(r.publicKey) }},
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
				record.publicKey,
				util.StringifyTimeZero(record.link.ExpiresAt),
			}
		}),
		crud.WithListReloadAfterChange[recordT, recordCreateT, recordUpdateT, recordIdT, filterT](true),
	)
}
