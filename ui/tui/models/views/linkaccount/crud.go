// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package linkaccount

import (
	"context"
	"fmt"
	"time"

	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/tags"
	"github.com/toeirei/keymaster/ui/tui/models/components/router"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/crud"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	formelement "github.com/toeirei/keymaster/ui/tui/models/helpers/form/element"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/table"
	"github.com/toeirei/keymaster/util/slicest"
)

const (
	timeLayout1 string = "2006.01.02 15:04:05"
	timeLayout2 string = "2006.01.02 15:04"
	timeLayout3 string = "2006.01.02"
	timeLayout4 string = "02.01.2006 15:04:05"
	timeLayout5 string = "02.01.2006 15:04"
	timeLayout6 string = "02.01.2006"
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

func parseExpiresAt(expiresAtStr string) (time.Time, error) {
	timeParseLayouts := []string{timeLayout1, timeLayout2, timeLayout3, timeLayout4, timeLayout5, timeLayout6}

	var expiresAt time.Time
	var err error
	for _, layout := range timeParseLayouts {
		expiresAt, err = time.Parse(layout, expiresAtStr)
		if err == nil {
			break
		}
	}

	return expiresAt, err
}

func formRows[T comparable]() []form.FormOpt[T] {
	return []form.FormOpt[T]{
		form.WithRowItem[T]("tag_matcher", formelement.NewText("Tag Matcher", "text to match tags of public keys")),
		form.WithRowItem[T]("expires_at", formelement.NewText("Expires At", "date on witch this link will expire and its public keys will loose access")),
	}
}

func NewCrud(c client.Client, rc router.Controll, account client.Account) *crud.Crud[recordT, recordCreateT, recordUpdateT, recordIdT, filterT] {
	return crud.New(
		crud.Texts{"Link", "Links"},

		func(record recordT) recordIdT { return record.link.Id },
		func(filter filterT) ([]recordT, error) {
			links, err := c.ListLinksForAccount(context.Background(), account.Id, true)
			if err != nil {
				return nil, err
			}

			return slicest.MapX(links, func(link client.Link) (recordT, error) {
				return linkToRecord(context.Background(), c, link)
			})
		},
		func(id recordIdT) (recordT, error) {
			link, err := c.GetLink(context.Background(), id)
			if err != nil {
				return recordT{}, err
			}

			return linkToRecord(context.Background(), c, link)
		},
		func(recordCreate recordCreateT) (recordT, error) {
			expr, err := tags.ParseMatcher(recordCreate.TagMatcher)
			if err != nil {
				return recordT{}, err
			}

			expiresAt, err := parseExpiresAt(recordCreate.ExpiresAt)
			if err != nil {
				return recordT{}, err
			}

			link, err := c.CreateLink(
				context.Background(),
				account.Id,
				expr.String(),
				expiresAt,
			)
			if err != nil {
				return recordT{}, err
			}

			return linkToRecord(context.Background(), c, link)
		},
		func(id recordIdT, recordUpdate recordUpdateT) (recordT, error) {
			expr, err := tags.ParseMatcher(recordUpdate.TagMatcher)
			if err != nil {
				return recordT{}, err
			}

			expiresAt, err := parseExpiresAt(recordUpdate.ExpiresAt)
			if err != nil {
				return recordT{}, err
			}

			if err := c.UpdateLink(
				context.Background(),
				id,
				account.Id,
				expr.String(),
				expiresAt,
			); err != nil {
				return recordT{}, err
			}

			link, err := c.GetLink(context.Background(), id)
			if err != nil {
				return recordT{}, err
			}

			return linkToRecord(context.Background(), c, link)
		},
		func(id recordIdT) error {
			return c.DeleteLinks(context.Background(), id)
		},

		table.NewBubblesTableRenderer(table.Columns[recordT]{
			{Title: "Account", View: func(r recordT) string { return account.String() }},
			{Title: "Tag Matcher", View: func(r recordT) string { return r.link.TagMatcher }},
			{Title: "Expires At", View: func(r recordT) string { return fmt.Sprint(r.link.ExpiresAt) }},
			{Title: "Public Keys", View: func(r recordT) string { return fmt.Sprint(r.linkedPublicKeyCount) }},
		}),
		func(record recordT) recordUpdateT {
			return recordUpdateT{
				record.link.TagMatcher,
				record.link.ExpiresAt.Format(timeLayout1),
			}
		},

		formRows[recordCreateT],
		formRows[recordUpdateT],

		rc,

		crud.WithListDuplicateAction[recordT, recordCreateT, recordUpdateT, recordIdT, filterT](func(record recordT) recordCreateT {
			return recordCreateT{
				record.link.TagMatcher,
				record.link.ExpiresAt.Format(timeLayout1),
			}
		}),
		crud.WithListReloadAfterChange[recordT, recordCreateT, recordUpdateT, recordIdT, filterT](true),
	)
}
