// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package account

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bobg/go-generics/v4/slices"
	"github.com/charmbracelet/bubbles/table"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/ui/tui/models/components/router"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	formelement "github.com/toeirei/keymaster/ui/tui/models/helpers/form/element"
	"github.com/toeirei/keymaster/ui/tui/models/views/crud"
	"github.com/toeirei/keymaster/util/slicest"
)

type createFormData struct {
	Name         string `form:"name"`
	Host         string `form:"host"`
	Port         string `form:"port"`
	DeployMethod string `form:"deploy_method"`
	DeploySecret string `form:"deploy_secret"`
}

type updateFormData = createFormData

func formRows[T comparable]() []form.FormOpt[T] {
	return []form.FormOpt[T]{
		form.WithRowItem[T]("name", formelement.NewText("Name", "eg. user/root/...")),
		form.WithRowItem[T]("host", formelement.NewText("Host", "ip/domain to connect to")),
		form.WithRowItem[T]("port", formelement.NewText("Port", "eg. 22")),
		form.WithRowItem[T]("deploy_method", formelement.NewText("Deploy Method", "ssh/cisco/...")),
		form.WithRowItem[T]("deploy_secret", formelement.NewTextarea("Deploy Secret", "", 3, 5)),
	}
}

func NewCrud(c client.Client, rc router.Controll) *crud.Crud[client.Account, createFormData, updateFormData, client.AccountId, struct{}] {
	return crud.New(
		crud.Texts{"Account", "Accounts"},
		func(record client.Account) client.AccountId { return record.Id },
		func(filter struct{}) ([]client.Account, error) {
			return c.ListAccounts(context.Background())
		},
		func(id client.AccountId) (client.Account, error) {
			return c.GetAccount(context.Background(), id)
		},
		func(record createFormData) (client.Account, error) {
			port, _ := strconv.Atoi(record.Port)
			return c.CreateAccount(
				context.Background(),
				record.Name,
				record.Host,
				port,
				record.DeployMethod,
				record.DeploySecret,
			)
		},
		func(id client.AccountId, record updateFormData) (client.Account, error) {
			port, err := strconv.Atoi(record.Port)
			if err != nil {
				return client.Account{}, err
			}

			if err := c.UpdateAccount(
				context.Background(),
				id,
				record.Name,
				record.Host,
				port,
				record.DeployMethod,
				record.DeploySecret,
			); err != nil {
				return client.Account{}, err
			}

			return c.GetAccount(context.Background(), id)
		},
		func(id client.AccountId) error {
			return c.DeleteAccounts(context.Background(), id)
		},
		func(record []client.Account, width int) ([]table.Column, []table.Row) {
			nameWidth := slicest.Reduce(record, func(a client.Account, w int) int { return max(w, len(a.Name)) })
			hostWidth := slicest.Reduce(record, func(a client.Account, w int) int { return max(w, len(a.Host)) })
			portWidth := slicest.Reduce(record, func(a client.Account, w int) int { return max(w, len(fmt.Sprint(a.Port))) })
			deployMethodWidth := slicest.Reduce(record, func(a client.Account, w int) int { return max(w, len(a.DeployMethod)) })

			remainingWidth := width - 6 - nameWidth - hostWidth - portWidth - deployMethodWidth

			columns := []table.Column{
				{Title: "Name", Width: nameWidth + remainingWidth/4},
				{Title: "Host", Width: hostWidth + remainingWidth/4},
				{Title: "Port", Width: portWidth + remainingWidth/4},
				{Title: "Deploy Method", Width: deployMethodWidth + remainingWidth/4},
			}

			rows := slices.Map(record, func(a client.Account) table.Row {
				return table.Row{
					// column: Name
					a.Name,
					// column: Host
					a.Host,
					// column: Port
					fmt.Sprint(a.Port),
					// column: Deploy Method
					a.DeployMethod,
				}
			})

			return columns, rows
		},
		func(record client.Account) updateFormData {
			return updateFormData{
				record.Name,
				record.Host,
				fmt.Sprint(record.Port),
				record.DeployMethod,
				record.DeploySecret,
			}
		},

		formRows[createFormData],
		formRows[updateFormData],

		rc,

		crud.WithListDuplicateAction[client.Account, createFormData, updateFormData, client.AccountId, struct{}](func(record client.Account) createFormData {
			return createFormData{
				record.Name,
				record.Host,
				fmt.Sprint(record.Port),
				record.DeployMethod,
				record.DeploySecret,
			}
		}),
	)
}
