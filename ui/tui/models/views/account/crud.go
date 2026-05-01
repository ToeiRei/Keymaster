// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package account

import (
	"context"
	"fmt"
	"strconv"

	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/ui/tui/models/components/router"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	formelement "github.com/toeirei/keymaster/ui/tui/models/helpers/form/element"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/table"
	"github.com/toeirei/keymaster/ui/tui/models/views/crud"
	"github.com/toeirei/keymaster/util/slicest"
)

type recordT = struct {
	account              client.Account
	isDirty              bool
	linkCount            int
	linkedPublicKeyCount int
}

type recordCreateT = struct {
	Name         string `form:"name"`
	Host         string `form:"host"`
	Port         string `form:"port"`
	DeployMethod string `form:"deploy_method"`
	DeploySecret string `form:"deploy_secret"`
}

type recordUpdateT = recordCreateT

type recordIdT = client.AccountId

type filterT = struct{}

func accountToRecord(ctx context.Context, c client.Client, account client.Account) (recordT, error) {
	links, err := c.ListLinksAccount(ctx, account.Id)
	if err != nil {
		return recordT{}, err
	}

	publicKeys, err := c.ListPublicKeysForAccount(ctx, account.Id)
	if err != nil {
		return recordT{}, err
	}

	isDirty, err := c.IsAccountDirty(ctx, account)
	if err != nil {
		return recordT{}, err
	}

	return recordT{account, isDirty, len(links), len(publicKeys)}, nil
}

func formRows[T comparable]() []form.FormOpt[T] {
	return []form.FormOpt[T]{
		form.WithRowItem[T]("name", formelement.NewText("Name", "eg. user/root/...")),
		form.WithRowItem[T]("host", formelement.NewText("Host", "ip/domain to connect to")),
		form.WithRowItem[T]("port", formelement.NewText("Port", "eg. 22")),
		form.WithRowItem[T]("deploy_method", formelement.NewText("Deploy Method", "ssh/cisco/...")),
		form.WithRowItem[T]("deploy_secret", formelement.NewTextarea("Deploy Secret", "", 3, 5)),
	}
}

func NewCrud(c client.Client, rc router.Controll) *crud.Crud[recordT, recordCreateT, recordUpdateT, recordIdT, filterT] {
	return crud.New(
		crud.Texts{"Account", "Accounts"},

		func(record recordT) recordIdT { return record.account.Id },
		func(filter filterT) ([]recordT, error) {
			accounts, err := c.ListAccounts(context.Background())
			if err != nil {
				return nil, err
			}

			return slicest.MapX(accounts, func(account client.Account) (recordT, error) {
				return accountToRecord(context.Background(), c, account)
			})
		},
		func(id recordIdT) (recordT, error) {
			account, err := c.GetAccount(context.Background(), id)
			if err != nil {
				return recordT{}, err
			}

			return accountToRecord(context.Background(), c, account)
		},
		func(recordCreate recordCreateT) (recordT, error) {
			port, err := strconv.Atoi(recordCreate.Port)
			if err != nil {
				return recordT{}, err
			}

			account, err := c.CreateAccount(
				context.Background(),
				recordCreate.Name,
				recordCreate.Host,
				port,
				recordCreate.DeployMethod,
				recordCreate.DeploySecret,
			)
			if err != nil {
				return recordT{}, err
			}

			return accountToRecord(context.Background(), c, account)
		},
		func(id recordIdT, recordUpdate recordUpdateT) (recordT, error) {
			port, err := strconv.Atoi(recordUpdate.Port)
			if err != nil {
				return recordT{}, err
			}

			if err := c.UpdateAccount(
				context.Background(),
				id,
				recordUpdate.Name,
				recordUpdate.Host,
				port,
				recordUpdate.DeployMethod,
				recordUpdate.DeploySecret,
			); err != nil {
				return recordT{}, err
			}

			account, err := c.GetAccount(context.Background(), id)
			if err != nil {
				return recordT{}, err
			}

			return accountToRecord(context.Background(), c, account)
		},
		func(id recordIdT) error {
			return c.DeleteAccounts(context.Background(), id)
		},

		table.NewBubblesTableRenderer(table.Columns[recordT]{
			{Title: "Name", View: func(r recordT) string { return r.account.Name }},
			{Title: "Host", View: func(r recordT) string { return r.account.Host }},
			{Title: "Port", View: func(r recordT) string { return fmt.Sprint(r.account.Port) }},
			{Title: "Deploy Method", View: func(r recordT) string { return r.account.DeployMethod }},
			{Title: "Dirty", View: func(r recordT) string { return fmt.Sprint(r.isDirty) }},
			{Title: "Links", View: func(r recordT) string { return fmt.Sprint(r.linkCount) }},
			{Title: "Public Keys", View: func(r recordT) string { return fmt.Sprint(r.linkedPublicKeyCount) }},
		}),
		func(record recordT) recordUpdateT {
			return recordUpdateT{
				record.account.Name,
				record.account.Host,
				fmt.Sprint(record.account.Port),
				record.account.DeployMethod,
				record.account.DeploySecret,
			}
		},

		formRows[recordCreateT],
		formRows[recordUpdateT],

		rc,

		crud.WithListDuplicateAction[recordT, recordCreateT, recordUpdateT, recordIdT, filterT](func(record recordT) recordCreateT {
			return recordCreateT{
				record.account.Name,
				record.account.Host,
				fmt.Sprint(record.account.Port),
				record.account.DeployMethod,
				record.account.DeploySecret,
			}
		}),
	)
}
