// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package account

import (
	"context"
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/ui/i18n"
	"github.com/toeirei/keymaster/ui/tui/components/router"
	"github.com/toeirei/keymaster/ui/tui/helpers/crud"
	"github.com/toeirei/keymaster/ui/tui/helpers/form"
	formelement "github.com/toeirei/keymaster/ui/tui/helpers/form/element"
	"github.com/toeirei/keymaster/ui/tui/helpers/tablecontroll"
	"github.com/toeirei/keymaster/ui/tui/popups/formpopup"
	"github.com/toeirei/keymaster/ui/tui/popups/messagepopup"
	"github.com/toeirei/keymaster/ui/tui/popups/selectpopup"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
	"github.com/toeirei/keymaster/ui/tui/views/linkaccount"
	"github.com/toeirei/keymaster/util/slicest"
)

type connectorData struct {
	Key string
	// values keyed by [connector.SecretField.Key], as CreateAccount wants them
	Secret map[string]string
}

func accountConnectorData(account client.Account) connectorData {
	values := make(map[string]string)
	if account.ConnectorSecret != nil {
		for _, field := range account.ConnectorSecret.Fields() {
			values[field.Key] = field.Value
		}
	}
	return connectorData{account.Connector, values}
}

type recordT = struct {
	account                    client.Account
	isDirty                    bool
	activeLinkCount            int
	activeLinkedPublicKeyCount int
	totalLinkCount             int
	totalLinkedPublicKeyCount  int
}

type recordCreateT = struct {
	Username  string        `form:"username"`
	Host      string        `form:"host"`
	Port      string        `form:"port"`
	Connector connectorData `form:"connector"`
}

// recordUpdateT has no connector: the edit form cannot change how an account
// authenticates, only where it points. Changing the secret goes through
// [client.Client.UpdateAccountSecret], which confirms it against the target.
type recordUpdateT = struct {
	Username string `form:"username"`
	Host     string `form:"host"`
	Port     string `form:"port"`
}

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

// secretFormValues returns the values the secret form should open on. Picking the
// connector the account already uses keeps what it holds; switching to another one
// starts from that connector's template. The current values are overlaid rather
// than substituted, so a field the connector has since added keeps its template
// default, and one it no longer declares is dropped.
func secretFormValues(secretFields []client.SecretField, connectorKey string, current connectorData) map[string]string {
	values := make(map[string]string, len(secretFields))
	for _, secretField := range secretFields {
		values[secretField.Key] = secretField.Value
	}

	if connectorKey != current.Key {
		return values
	}

	for key, value := range current.Secret {
		if _, declared := values[key]; declared {
			values[key] = value
		}
	}
	return values
}

// recordFormRows are the rows every account form has: where the account points.
func recordFormRows[T any]() []form.FormOpt[T] {
	return []form.FormOpt[T]{
		form.WithRowItem[T]("username", formelement.NewText(i18n.Text("account.form.username_label"), i18n.Text("account.form.username_placeholder"))),
		form.WithRowItem[T]("host", formelement.NewText(i18n.Text("account.form.host_label"), i18n.Text("account.form.host_placeholder"))),
		form.WithRowItem[T]("port", formelement.NewText(i18n.Text("account.form.port_label"), i18n.Text("account.form.port_placeholder"))),
	}
}

// openConnectorSelect lists the registered connectors and hands the chosen one to
// onSelect.
func openConnectorSelect(c client.Client, onSelect func(connectorKey string) tea.Cmd) tea.Cmd {
	return selectpopup.Open(
		i18n.Text("account.select_connector"),
		func(ctx context.Context) ([]string, error) { return c.ListConnectorKeys(ctx) },
		onSelect,
		tablecontroll.New(tablecontroll.Columns[string]{
			{Title: i18n.Text("account.col_connector"), View: func(r string) string { return r }},
		}),
	)
}

// openSecretForm renders a connector's secret fields, prefilled from current, and
// hands the submitted values to onSubmit. Shared by the account form's connector
// row and the standalone secret actions, so the runtime field handling and the
// value overlay live in one place.
func openSecretForm(c client.Client, connectorKey string, current connectorData, onSubmit func(connectorData) tea.Cmd) tea.Cmd {
	secretFields, err := c.ConnectorSecretFields(connectorKey)
	if err != nil {
		return messagepopup.Open(messagepopup.Error, i18n.WrapError(err, "errors.account.connector_secret_fields", connectorKey), nil)
	}

	// the secret fields are only known at runtime, so the form
	// is modelled as a map keyed by SecretField.Key
	formOpts := slicest.Map(secretFields, func(secretField client.SecretField) form.FormOpt[map[string]string] {
		if secretField.Multiline {
			// A textarea has no echo mode, so a field that is both
			// multiline and masked cannot be masked. No connector
			// declares one today.
			return form.WithRowItem[map[string]string](secretField.Key, formelement.NewTextarea(secretField.Label, i18n.RawText(""), 3, 10))
		}

		textOpts := make([]formelement.TextOption, 0, 1)
		if secretField.Masked {
			textOpts = append(textOpts, formelement.WithTextEchoPassword())
		}
		return form.WithRowItem[map[string]string](secretField.Key, formelement.NewText(secretField.Label, i18n.RawText(""), textOpts...))
	})

	formOpts = append(formOpts,
		form.WithRow(
			form.WithItem[map[string]string]("_cancel", formelement.NewButton(i18n.Text("crud.btn_cancel"),
				formelement.WithButtonActionCancel(),
				formelement.WithButtonGlobalKeyBindings(keys.Cancel()),
			)),
			form.WithItem[map[string]string]("_submit", formelement.NewButton(i18n.Text("crud.btn_save"), formelement.WithButtonActionSubmit())),
		),
		// No OnCancel: backing out just closes the popup, leaving
		// the element's current value alone, so cancelling does
		// not clear an already configured connector.
		form.WithOnSubmit(func(result map[string]string, err error) (tea.Cmd, bool) {
			if err != nil {
				return messagepopup.Open(messagepopup.Error, i18n.WrapError(err, "errors.account.connector_secret_invalid"), nil), false
			}
			return onSubmit(connectorData{connectorKey, result}), true
		}),
		form.WithInitialData(secretFormValues(secretFields, connectorKey, current)),
	)

	return formpopup.Open(form.New(formOpts...))
}

// connectorFormRow collects a connector and its secret. Only the create form and
// the force-update flow use it; a plain edit must not be able to overwrite a
// credential a target may already be holding.
func connectorFormRow[T any](c client.Client) form.FormOpt[T] {
	return form.WithRowItem[T]("connector", formelement.NewPopup(i18n.Text("account.form.connector_label"),
		func(current connectorData, returnValue func(value connectorData) tea.Cmd) tea.Cmd {
			return openConnectorSelect(c, func(connectorKey string) tea.Cmd {
				return openSecretForm(c, connectorKey, current, returnValue)
			})
		},
		func(data connectorData) string {
			if data.Key == "" {
				return lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("240")).Render(i18n.T("crud.none"))
			}
			return data.Key
		},
	))
}

func NewCrud(c client.Client, rc router.Controll) *crud.Crud[recordT, recordCreateT, recordUpdateT, recordIdT, filterT] {
	return crud.New(
		crud.Texts{
			i18n.Text("account.entity_singular"),
			i18n.Text("account.entity_plural"),
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
					recordCreate.Connector.Key,
					recordCreate.Connector.Secret,
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
			{Title: i18n.Text("account.col_username"), View: func(r recordT) string { return r.account.Username }},
			{Title: i18n.Text("account.col_host"), View: func(r recordT) string { return r.account.Host }},
			{Title: i18n.Text("account.col_port"), View: func(r recordT) string { return fmt.Sprint(r.account.Port) }},
			{Title: i18n.Text("account.col_connector"), View: func(r recordT) string { return r.account.Connector }},
			{Title: i18n.Text("account.col_dirty"), View: func(r recordT) string { return fmt.Sprint(r.isDirty) }},
			{Title: i18n.Text("account.col_links"), View: func(r recordT) string {
				return fmt.Sprintf("%d/%d", r.activeLinkCount, r.totalLinkCount)
			}},
			{Title: i18n.Text("account.col_public_keys"), View: func(r recordT) string {
				return fmt.Sprintf("%d/%d", r.activeLinkedPublicKeyCount, r.totalLinkedPublicKeyCount)
			}},
		}).RenderBubblesTable,
		func(record recordT) recordUpdateT {
			return recordUpdateT{
				record.account.Username,
				record.account.Host,
				fmt.Sprint(record.account.Port),
			}
		},

		func() []form.FormOpt[recordCreateT] {
			return append(recordFormRows[recordCreateT](), connectorFormRow[recordCreateT](c))
		},
		func() []form.FormOpt[recordUpdateT] { return recordFormRows[recordUpdateT]() },

		rc,

		crud.WithListDuplicateAction[recordT, recordCreateT, recordUpdateT, recordIdT, filterT](func(record recordT) recordCreateT {
			return recordCreateT{
				record.account.Username,
				record.account.Host,
				fmt.Sprint(record.account.Port),
				accountConnectorData(record.account),
			}
		}),
		crud.WithListAction(
			func(ctx crud.ListMsgInterceptorCtx[recordT, recordCreateT, recordUpdateT, recordIdT, filterT]) tea.Cmd {
				if ctx.SelectedRecord == nil {
					return messagepopup.Open(messagepopup.Error, i18n.Text("crud.select_generic", ctx.Crud.Texts.EntityNameSingular), nil)
				}

				ctx.Crud.ReloadOnNextFocus = true
				return linkaccount.NewCrud(c, rc, ctx.SelectedRecord.account).OpenList()
			},
			key.NewBinding(
				key.WithKeys("l"),
				key.WithHelp("l", i18n.T("keys.links")),
			),
		),
		withUpdateSecretAction(c),
		withConnectorForceAction(c),
		crud.WithListReloadAfterChange[recordT, recordCreateT, recordUpdateT, recordIdT, filterT](true),
	)
}
