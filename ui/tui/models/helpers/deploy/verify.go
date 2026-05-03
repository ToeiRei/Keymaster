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
	popupviews "github.com/toeirei/keymaster/ui/tui/models/views/popup"
	"github.com/toeirei/keymaster/util/slicest"
)

func VerifyAll(ctx context.Context, c client.Client) tea.Cmd {
	accounts, err := c.ListAccounts(ctx)
	if err != nil {
		return popupviews.OpenMessage(popupviews.MessageError, err.Error(), nil)
	}

	return VerifyMany(ctx, c, accounts...)
}

func VerifyDirty(ctx context.Context, c client.Client) tea.Cmd {
	accounts, err := c.ListAccountsDirty(ctx)
	if err != nil {
		return popupviews.OpenMessage(popupviews.MessageError, err.Error(), nil)
	}

	return VerifyMany(ctx, c, accounts...)
}

// func VerifyOne(ctx context.Context, c client.Client, ids ...client.AccountId) tea.Cmd

func VerifyMany(ctx context.Context, c client.Client, accounts ...client.Account) tea.Cmd {
	if len(accounts) == 0 {
		return popupviews.OpenMessage(popupviews.MessageInfo, "No Accounts found for verifyment.", nil)
	}

	ids := slicest.Map(accounts, func(account client.Account) client.AccountId { return account.Id })
	accountNamesMap := slicest.ToMap(accounts, func(account client.Account) (client.AccountId, string) { return account.Id, account.String() })
	accountNamesWidth := slicest.Reduce(slicest.MapValues(accountNamesMap), func(accountName string, width int) int { return max(width, len(accountName)) })
	accountNameRenderer := lipgloss.NewStyle().Width(accountNamesWidth)

	dpc, err := c.VerifyAccounts(ctx, ids...)
	if err != nil {
		return popupviews.OpenMessage(popupviews.MessageError, err.Error(), nil)
	}

	return popupviews.OpenProgress(
		popupviews.ProgressBar,
		"Verifying Accounts",
		func(pc popupviews.ProgressChan) tea.Cmd {
			var dp client.VerifyProgressAccounts

			// map [client.VerifyProgressAccounts] chan to [popupviews.Progress] chan
			for dp = range dpc {
				pc <- popupviews.Progress{
					dp.Progress(),
					strings.Join(
						slicest.Map(ids, func(id client.AccountId) string {
							return fmt.Sprintf("%s [%s]", accountNameRenderer.Render(accountNamesMap[id]), dp.Accounts[id].Status)
						}),
						"\n",
					),
				}
			}

			severity := popupviews.MessageSuccess
			if slicest.Contains(slicest.MapValues(dp.Accounts), func(dpa *client.VerifyProgressAccount) bool { return dpa.Err != nil }) {
				severity = popupviews.MessageError
			}

			return popupviews.OpenMessage(
				severity,
				strings.Join(
					slicest.Map(ids, func(id client.AccountId) string {
						if dp.Accounts[id].Err != nil {
							return fmt.Sprintf("%s %s", accountNameRenderer.Render(accountNamesMap[id]), dp.Accounts[id].Err.Error())
						}
						return fmt.Sprintf("%s Success", accountNameRenderer.Render(accountNamesMap[id]))
					}),
					"\n",
				),
				nil,
			)
		},
	)
}
