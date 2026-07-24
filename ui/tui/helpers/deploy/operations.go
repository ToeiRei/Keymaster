// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package deploy

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/ui/i18n"
	"github.com/toeirei/keymaster/ui/tui/helpers/form"
	formelement "github.com/toeirei/keymaster/ui/tui/helpers/form/element"
	"github.com/toeirei/keymaster/ui/tui/helpers/popup"
	"github.com/toeirei/keymaster/ui/tui/popups/choicepopup"
	"github.com/toeirei/keymaster/ui/tui/popups/formpopup"
	"github.com/toeirei/keymaster/ui/tui/popups/messagepopup"
	"github.com/toeirei/keymaster/ui/tui/popups/progresspopup"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
	"github.com/toeirei/keymaster/util/slicest"
)

// accountOperation starts a deploy or verify over the given accounts, streaming
// aggregate progress on the provided channel. It is satisfied directly by
// [client.Client.DeployAccounts] and [client.Client.VerifyAccounts].
type accountOperation = func(context.Context, client.UserRequester, chan<- client.ProgressAccounts, ...client.AccountId) error

// runInteractive owns the operation and its context outside of the progress
// popup, so that a connector's user request can close the progress popup, show
// a form popup, and reopen the progress popup after the user replies. Context
// cancellation aborts the whole operation.
func runInteractive(parent context.Context, title, noun string, start accountOperation, accounts ...client.Account) tea.Cmd {
	if len(accounts) == 0 {
		return messagepopup.Open(messagepopup.Info, fmt.Sprintf(i18n.T("deploy.op_no_accounts"), i18n.T(noun)), nil)
	}

	ids := slicest.Map(accounts, func(account client.Account) client.AccountId { return account.Id })
	accountNamesMap := slicest.ToMap(accounts, func(account client.Account) (client.AccountId, string) { return account.Id, account.String() })
	accountNamesWidth := slicest.Reduce(slicest.MapValues(accountNamesMap), func(accountName string, width int) int { return max(width, len(accountName)) })
	accountNameRenderer := lipgloss.NewStyle().Width(accountNamesWidth)

	// The operation owns its own cancellable context, shared across every
	// reopened progress popup via [progresspopup.WithCancelFunc].
	ctx, cancel := context.WithCancel(parent)
	requester := newUserRequester(ctx)

	dpc := make(chan client.ProgressAccounts)
	var opErr error
	go func() {
		defer close(dpc)
		opErr = start(ctx, requester, dpc, ids...)
	}()

	statusText := func(dp client.ProgressAccounts) string {
		return strings.Join(
			slicest.Map(ids, func(id client.AccountId) string {
				return fmt.Sprintf("%s [%s]", accountNameRenderer.Render(accountNamesMap[id]), dp.Accounts[id].Status)
			}),
			"\n",
		)
	}

	finalMessage := func(dp client.ProgressAccounts) tea.Cmd {
		if opErr != nil {
			return messagepopup.Open(messagepopup.Error, opErr.Error(), nil)
		}
		if dp.Accounts == nil {
			// The operation ended before any progress was reported (e.g. it was
			// cancelled immediately).
			return messagepopup.Open(messagepopup.Warning, i18n.T("deploy.op_cancelled"), nil)
		}

		severity := messagepopup.Success
		switch {
		case ctx.Err() != nil:
			severity = messagepopup.Warning
		case slicest.Contains(slicest.MapValues(dp.Accounts), func(dpa *client.ProgressAccountWithError) bool { return dpa.Err != nil }):
			severity = messagepopup.Error
		}

		return messagepopup.Open(
			severity,
			strings.Join(
				slicest.Map(ids, func(id client.AccountId) string {
					if dp.Accounts[id].Err != nil {
						return fmt.Sprintf(i18n.T("deploy.op_account_error"), accountNameRenderer.Render(accountNamesMap[id]), dp.Accounts[id].Err.Error())
					}
					return fmt.Sprintf(i18n.T("deploy.op_account_success"), accountNameRenderer.Render(accountNamesMap[id]))
				}),
				"\n",
			),
			nil,
		)
	}

	// leg drives one progress popup: it forwards progress until the operation
	// finishes (-> final message) or a connector requests user input (-> form).
	// last carries the most recent aggregate progress across reopened popups.
	var last client.ProgressAccounts
	var leg func(ctx context.Context, pc progresspopup.ProgressChan) tea.Cmd
	var reopen func() tea.Cmd

	reopen = func() tea.Cmd {
		return progresspopup.Open(
			progresspopup.Bar,
			i18n.T(title),
			leg,
			progresspopup.WithContext(ctx),
			progresspopup.WithCancelFunc(cancel),
		)
	}

	leg = func(_ context.Context, pc progresspopup.ProgressChan) tea.Cmd {
		for {
			select {
			case dp, ok := <-dpc:
				if !ok {
					// All accounts finished (or fully wound down after cancel).
					requester.Close()
					return finalMessage(last)
				}
				last = dp
				pc <- progresspopup.Progress{Progress: dp.Progress(), Status: statusText(dp)}

			case req := <-requester.request:
				return requestForm(req, requester, cancel, reopen)
			}
		}
	}

	return reopen()
}

// requestForm builds the popup for a connector's user request. The user's reply
// unblocks the parked connector goroutine, then the progress popup is reopened.
// Cancelling a request aborts the whole operation.
func requestForm(req any, requester *userRequester, cancel context.CancelFunc, reopen func() tea.Cmd) tea.Cmd {
	switch req := req.(type) {
	case TextRequest:
		return textRequestForm(string(req), requester, cancel, reopen)

	case ChoiceRequest:
		return choiceRequestForm([]string(req), requester, cancel, reopen)

	default:
		// Unexpected request value (e.g. request channel closed) -> just resume.
		return reopen()
	}
}

type textAnswer struct {
	Answer string `form:"answer"`
}

// textRequestForm shows a text-input popup for a [TextRequest]. On submit it
// replies with the entered value; on cancel it aborts the whole operation.
func textRequestForm(prompt string, requester *userRequester, cancel context.CancelFunc, reopen func() tea.Cmd) tea.Cmd {
	return formpopup.Open(form.New(
		form.WithRowItem[textAnswer]("_prompt", formelement.NewLabel(i18n.Text(prompt))),
		form.WithRowItem[textAnswer]("answer", formelement.NewText(i18n.Text(""), i18n.Text(""))),
		form.WithRow(
			form.WithItem[textAnswer]("_submit", formelement.NewButton(i18n.Text("deploy.op_submit"), formelement.WithButtonActionSubmit())),
			form.WithItem[textAnswer]("_cancel", formelement.NewButton(i18n.Text("crud.btn_cancel"),
				formelement.WithButtonActionCancel(),
				formelement.WithButtonGlobalKeyBindings(keys.Cancel()),
			)),
		),
		form.WithOnSubmit(func(res textAnswer, _ error) (tea.Cmd, bool) {
			return tea.Sequence(popup.Close(), requester.replyTextCmd(res.Answer), reopen()), true
		}),
		form.WithOnCancel[textAnswer](func() tea.Cmd {
			cancel()
			return tea.Sequence(popup.Close(), reopen())
		}),
		form.WithDefaultRowAlign[textAnswer](form.Center),
	))
}

// choiceRequestForm shows a choice popup for a [ChoiceRequest]. Selecting an
// option replies with its index; a trailing Cancel choice aborts the whole
// operation.
func choiceRequestForm(choices []string, requester *userRequester, cancel context.CancelFunc, reopen func() tea.Cmd) tea.Cmd {
	popupChoices := slicest.MapI(choices, func(i int, label string) choicepopup.Choice {
		return choicepopup.Choice{
			Name: label,
			Cmd:  tea.Sequence(requester.replyChoiceCmd(i), reopen()),
		}
	})
	popupChoices = append(popupChoices, choicepopup.Choice{
		Name:        i18n.T("crud.btn_cancel"),
		Cmd:         tea.Sequence(cancelCmd(cancel), reopen()),
		KeyBindings: keys.KeyBindingList{keys.Cancel()},
	})
	return choicepopup.Open(i18n.T("deploy.op_connector_choice"), popupChoices)
}

// cancelCmd cancels the operation's context from within a [tea.Cmd].
func cancelCmd(cancel context.CancelFunc) tea.Cmd {
	return func() tea.Msg { cancel(); return nil }
}
