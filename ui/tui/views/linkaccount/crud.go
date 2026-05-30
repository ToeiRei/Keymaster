// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package linkaccount

import (
	"context"
	"fmt"

	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/tags"
	"github.com/toeirei/keymaster/ui/tui/components/router"
	"github.com/toeirei/keymaster/ui/tui/helpers/crud"
	"github.com/toeirei/keymaster/ui/tui/helpers/form"
	formelement "github.com/toeirei/keymaster/ui/tui/helpers/form/element"
	"github.com/toeirei/keymaster/ui/tui/helpers/tablecontroll"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/util/slicest"
)

type recordT = struct {
	link                 client.Link
	linkedPublicKeyCount int
}

type recordCreateT = struct {
	TagMatcher string `form:"tag_matcher"`
	ExpiresAt  string `form:"expires_at"`
}

type recordUpdateT = recordCreateT

type recordIdT = client.LinkId

type filterT = struct{}

func linkToRecord(ctx context.Context, c client.Client, link client.Link) (recordT, error) {
	publicKeys, err := c.ListPublicKeys(ctx, link.TagMatcher)
	if err != nil {
		return recordT{}, err
	}

	return recordT{link, len(publicKeys)}, nil
}

func formRows[T comparable]() []form.FormOpt[T] {
	return []form.FormOpt[T]{
		form.WithRowItem[T]("tag_matcher", formelement.NewText("Tag Matcher", "text to match tags of public keys")),
		form.WithRowItem[T]("expires_at", formelement.NewText("Expires At", "date on witch this link will expire and its public keys will loose access")),
	}
}

func NewCrud(c client.Client, rc router.Controll, account client.Account) *crud.Crud[recordT, recordCreateT, recordUpdateT, recordIdT, filterT] {
	return crud.New(
		crud.Texts{EntityNameSingular: "Link", EntityNameMultiple: "Links"},

		func(record recordT) recordIdT { return record.link.Id },
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

				link, err := c.CreateLink(
					ctx,
					account.Id,
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

				link, err := c.UpdateLink(ctx, id, account.Id, expr.String(), expiresAt)
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
			{Title: "Account", View: func(r recordT) string { return account.String() }},
			{Title: "Public Keys", View: func(r recordT) string { return fmt.Sprint(r.linkedPublicKeyCount) }},
		}).RenderBubblesTable,
		func(record recordT) recordUpdateT {
			return recordUpdateT{
				record.link.TagMatcher,
				util.StringifyTime(record.link.ExpiresAt),
			}
		},

		formRows[recordCreateT],
		formRows[recordUpdateT],

		rc,

		crud.WithListDuplicateAction[recordT, recordCreateT, recordUpdateT, recordIdT, filterT](func(record recordT) recordCreateT {
			return recordCreateT{
				record.link.TagMatcher,
				util.StringifyTime(record.link.ExpiresAt),
			}
		}),
		crud.WithListReloadAfterChange[recordT, recordCreateT, recordUpdateT, recordIdT, filterT](true),
	)
}
