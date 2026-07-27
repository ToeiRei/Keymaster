// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package account

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/ui/i18n"
	"github.com/toeirei/keymaster/ui/tui/helpers/crud"
	"github.com/toeirei/keymaster/ui/tui/helpers/deploy"
	"github.com/toeirei/keymaster/ui/tui/popups/choicepopup"
	"github.com/toeirei/keymaster/ui/tui/popups/messagepopup"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

// listActionCtx is the interceptor context the account crud hands its list
// actions, spelled out once so the two below stay readable.
type listActionCtx = crud.ListMsgInterceptorCtx[recordT, recordCreateT, recordUpdateT, recordIdT, filterT]

type crudOption = crud.Option[recordT, recordCreateT, recordUpdateT, recordIdT, filterT]

// withUpdateSecretAction installs a new secret on the selected account's target
// and only records it once the target has confirmed it. The connector cannot be
// chosen here: the operation authenticates as the account's existing connector, so
// changing it is a different, destructive act with its own action below.
func withUpdateSecretAction(c client.Client) crudOption {
	return crud.WithListAction(func(ctx listActionCtx) tea.Cmd {
		account, ok := selectedAccount(ctx)
		if !ok {
			return noSelection(ctx)
		}

		return openSecretForm(c, account.Connector, accountConnectorData(account), func(submitted connectorData) tea.Cmd {
			ctx.Crud.ReloadOnNextFocus = true
			return deploy.UpdateSecret(context.Background(), c, account, submitted.Secret)
		})
	}, key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", i18n.T("keys.update_secret")),
	))
}

// withConnectorForceAction records a connector and secret without contacting the
// target, for one whose credentials were changed outside Keymaster. Nothing
// confirms it, so it is guarded and the account is left dirty on purpose.
func withConnectorForceAction(c client.Client) crudOption {
	return crud.WithListAction(func(ctx listActionCtx) tea.Cmd {
		account, ok := selectedAccount(ctx)
		if !ok {
			return noSelection(ctx)
		}

		return openConnectorSelect(c, func(connectorKey string) tea.Cmd {
			return openSecretForm(c, connectorKey, accountConnectorData(account), func(submitted connectorData) tea.Cmd {
				return confirmConnectorForce(c, account, submitted, func() {
					ctx.Crud.ReloadOnNextFocus = true
				})
			})
		})
	}, key.NewBinding(
		key.WithKeys("S"),
		key.WithHelp("S", i18n.T("keys.force_connector")),
	))
}

func selectedAccount(ctx listActionCtx) (client.Account, bool) {
	if ctx.SelectedRecord == nil {
		return client.Account{}, false
	}
	return ctx.SelectedRecord.account, true
}

func noSelection(ctx listActionCtx) tea.Cmd {
	return messagepopup.Open(messagepopup.Error, i18n.Text("crud.select_generic", ctx.Crud.Texts.EntityNameSingular), nil)
}

// confirmConnectorForce guards an override Keymaster cannot verify.
func confirmConnectorForce(c client.Client, account client.Account, submitted connectorData, onRecorded func()) tea.Cmd {
	return choicepopup.Open(
		connectorForceQuestion(account, submitted),
		connectorForceChoices(c, account, submitted, onRecorded),
	)
}

// connectorForceQuestion names the target, so an unverifiable override cannot be
// fired at the wrong row by accident.
func connectorForceQuestion(account client.Account, submitted connectorData) fmt.Stringer {
	if submitted.Key != account.Connector {
		// A different connector keeps only the account's key assignments; everything
		// describing the target is replaced, including its cached state.
		return i18n.Text("account.secret_force.confirm_connector_changed", account, account.Connector, submitted.Key)
	}
	return i18n.Text("account.secret_force.confirm", account)
}

// connectorForceChoices puts Cancel first, carrying no command at all, the way
// every other destructive guard here is ordered: the safe option holds focus and
// cannot reach the client even by accident.
func connectorForceChoices(c client.Client, account client.Account, submitted connectorData, onRecorded func()) choicepopup.Choices {
	return choicepopup.Choices{
		{Name: i18n.Text("crud.btn_cancel"), Cmd: nil, KeyBindings: keys.KeyBindingList{keys.Cancel()}},
		{Name: i18n.Text("account.secret_force.confirm_btn"), Cmd: recordConnectorForce(c, account, submitted, onRecorded)},
	}
}

// recordConnectorForce performs the override and reports its outcome. onRecorded
// runs only on success, since it marks the list as needing a reload.
func recordConnectorForce(c client.Client, account client.Account, submitted connectorData, onRecorded func()) tea.Cmd {
	return func() tea.Msg {
		if _, err := c.UpdateAccountConnectorForce(context.Background(), account.Id, submitted.Key, submitted.Secret); err != nil {
			return messagepopup.Open(messagepopup.Error, i18n.WrapError(err, "account.secret_force.failed"), nil)()
		}
		onRecorded()
		return messagepopup.Open(messagepopup.Success, i18n.Text("account.secret_force.done", account), nil)()
	}
}
