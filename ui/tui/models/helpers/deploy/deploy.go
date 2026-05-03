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

func DeployAll(ctx context.Context, c client.Client) tea.Cmd {
	accounts, err := c.ListAccounts(ctx)
	if err != nil {
		return popupviews.OpenMessage(popupviews.MessageError, err.Error(), nil)
	}

	return DeployMany(ctx, c, accounts...)
}

func DeployDirty(ctx context.Context, c client.Client) tea.Cmd {
	accounts, err := c.ListAccountsDirty(ctx)
	if err != nil {
		return popupviews.OpenMessage(popupviews.MessageError, err.Error(), nil)
	}

	return DeployMany(ctx, c, accounts...)
}

// func DeployOne(ctx context.Context, c client.Client, ids ...client.AccountId) tea.Cmd

func DeployMany(ctx context.Context, c client.Client, accounts ...client.Account) tea.Cmd {
	if len(accounts) == 0 {
		return popupviews.OpenMessage(popupviews.MessageInfo, "No Accounts found for deployment.", nil)
	}

	ids := slicest.Map(accounts, func(account client.Account) client.AccountId { return account.Id })
	accountNamesMap := slicest.ToMap(accounts, func(account client.Account) (client.AccountId, string) { return account.Id, account.String() })
	accountNamesWidth := slicest.Reduce(slicest.MapValues(accountNamesMap), func(accountName string, width int) int { return max(width, len(accountName)) })
	accountNameRenderer := lipgloss.NewStyle().Width(accountNamesWidth)

	return popupviews.OpenProgress(
		popupviews.ProgressBar,
		"Deploying Accounts",
		func(ctx context.Context, pc popupviews.ProgressChan) tea.Cmd {
			dpc, err := c.DeployAccounts(ctx, ids...)
			if err != nil {
				return popupviews.OpenMessage(popupviews.MessageError, err.Error(), nil)
			}

			var dp client.DeployProgressAccounts

			// map [client.DeployProgressAccounts] chan to [popupviews.Progress] chan
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
			if slicest.Contains(slicest.MapValues(dp.Accounts), func(dpa *client.DeployProgressAccount) bool { return dpa.Err != nil }) {
				severity = popupviews.MessageError
			}

			return popupviews.OpenMessage(
				severity,
				strings.Join(
					slicest.Map(ids, func(id client.AccountId) string {
						if dp.Accounts[id].Err != nil {
							return fmt.Sprintf("%s Error: %s", accountNameRenderer.Render(accountNamesMap[id]), dp.Accounts[id].Err.Error())
						}
						return fmt.Sprintf("%s Success", accountNameRenderer.Render(accountNamesMap[id]))
					}),
					"\n",
				),
				nil,
			)
		},
		popupviews.WithContext(ctx),
		popupviews.WithCancel(),
	)
}
