// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package account

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bobg/go-generics/v4/slices"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/ui/tui/models/components/router"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	formelement "github.com/toeirei/keymaster/ui/tui/models/helpers/form/element"
	"github.com/toeirei/keymaster/ui/tui/models/views/crud"
	popupviews "github.com/toeirei/keymaster/ui/tui/models/views/popup"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
	"github.com/toeirei/keymaster/util/slicest"
)

type createFormData struct {
	Name         string `form:"name"`
	Host         string `form:"host"`
	Port         string `form:"port"`
	DeployMethod string `form:"deploy_method"`
	DeploySecret string `form:"deploy_secret"`
}

type editFormData struct {
	Name         string `form:"name"`
	Host         string `form:"host"`
	Port         string `form:"port"`
	DeployMethod string `form:"deploy_method"`
	DeploySecret string `form:"deploy_secret"`
}

func NewCrud(c client.Client, rc router.Controll) *crud.Crud[client.Account, createFormData, editFormData, client.ID, struct{}] {
	return crud.New(
		func(record client.Account) client.ID { return record.Id },
		func(filter struct{}) ([]client.Account, error) {
			return c.ListAccounts(context.Background())
		},
		func(id client.ID) (client.Account, error) {
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
		func(id client.ID, record editFormData) (client.Account, error) {
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
		func(id client.ID) error {
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
		func(record client.Account) editFormData {
			return editFormData{
				record.Name,
				record.Host,
				fmt.Sprint(record.Port),
				record.DeployMethod,
				record.DeploySecret,
			}
		},

		func() []form.FormOpt[createFormData] {
			return []form.FormOpt[createFormData]{
				form.WithRowItem[createFormData]("name", formelement.NewText("Name", "eg. user/root/...")),
				form.WithRowItem[createFormData]("host", formelement.NewText("Host", "ip/domain to connect to")),
				form.WithRowItem[createFormData]("port", formelement.NewText("Port", "eg. 22")),
				form.WithRowItem[createFormData]("deploy_method", formelement.NewText("Deploy Method", "ssh/cisco/...")),
				form.WithRowItem[createFormData]("deploy_secret", formelement.NewText("Deploy Secret", "")),
			}
		},
		func() []form.FormOpt[editFormData] {
			return []form.FormOpt[editFormData]{
				form.WithRowItem[editFormData]("name", formelement.NewText("Name", "eg. user/root/...")),
				form.WithRowItem[editFormData]("host", formelement.NewText("Host", "ip/domain to connect to")),
				form.WithRowItem[editFormData]("port", formelement.NewText("Port", "eg. 22")),
				form.WithRowItem[editFormData]("deploy_method", formelement.NewText("Deploy Method", "ssh/cisco/...")),
				form.WithRowItem[editFormData]("deploy_secret", formelement.NewText("Deploy Secret", "")),
			}
		},

		rc,

		crud.WithListKeyBindings[client.Account, createFormData, editFormData, client.ID, struct{}](keys.Duplicate()),
		crud.WithListMsgInterceptor(func(msg tea.Msg, ctx crud.ListMsgInterceptorCtx[client.Account, createFormData, editFormData, client.ID, struct{}]) (tea.Cmd, bool) {
			if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, keys.Duplicate()) {
				if ctx.SelectedRecord == nil {
					return popupviews.OpenMessage(popupviews.MessageError, "Please select a Record to duplicate.", nil), true
				}
				return ctx.Crud.OpenCreate(&createFormData{
					ctx.SelectedRecord.Name,
					ctx.SelectedRecord.Host,
					fmt.Sprint(ctx.SelectedRecord.Port),
					ctx.SelectedRecord.DeployMethod,
					ctx.SelectedRecord.DeploySecret,
				}), true
			}
			return nil, false
		}),
	)
}
