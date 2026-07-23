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
	"github.com/toeirei/keymaster/ui/tui/popups/messagepopup"
	"github.com/toeirei/keymaster/ui/tui/popups/progresspopup"
	"github.com/toeirei/keymaster/util/slicest"
)

func DeployAll(ctx context.Context, c client.Client) tea.Cmd {
	accounts, err := c.ListAccounts(ctx)
	if err != nil {
		return messagepopup.Open(messagepopup.Error, err.Error(), nil)
	}

	return Deploy(ctx, c, accounts...)
}

func DeployDirty(ctx context.Context, c client.Client) tea.Cmd {
	accounts, err := c.ListAccountsDirty(ctx)
	if err != nil {
		return messagepopup.Open(messagepopup.Error, err.Error(), nil)
	}

	return Deploy(ctx, c, accounts...)
}

func Deploy(ctx context.Context, c client.Client, accounts ...client.Account) tea.Cmd {
	if len(accounts) == 0 {
		return messagepopup.Open(messagepopup.Info, "No Accounts found for deployment.", nil)
	}

	ids := slicest.Map(accounts, func(account client.Account) client.AccountId { return account.Id })
	accountNamesMap := slicest.ToMap(accounts, func(account client.Account) (client.AccountId, string) { return account.Id, account.String() })
	accountNamesWidth := slicest.Reduce(slicest.MapValues(accountNamesMap), func(accountName string, width int) int { return max(width, len(accountName)) })
	accountNameRenderer := lipgloss.NewStyle().Width(accountNamesWidth)

	requester := newUserRequester()

	return progresspopup.Open(
		progresspopup.Bar,
		"Deploying Accounts",
		func(ctx context.Context, pc progresspopup.ProgressChan) tea.Cmd {
			defer requester.Close()

			dpc, err := c.DeployAccounts(ctx, requester, ids...)
			if err != nil {
				return messagepopup.Open(messagepopup.Error, err.Error(), nil)
			}

			var dp client.DeployProgressAccounts

			// map [client.DeployProgressAccounts] chan to [progresspopup.Progress] chan
			for dp = range dpc {
				pc <- progresspopup.Progress{
					Progress: dp.Progress(),
					Status: strings.Join(
						slicest.Map(ids, func(id client.AccountId) string {
							return fmt.Sprintf("%s [%s]", accountNameRenderer.Render(accountNamesMap[id]), dp.Accounts[id].Status)
						}),
						"\n",
					),
				}
			}

			severity := messagepopup.Success
			if slicest.Contains(slicest.MapValues(dp.Accounts), func(dpa *client.DeployProgressAccount) bool { return dpa.Err != nil }) {
				severity = messagepopup.Error
			}

			return messagepopup.Open(
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
		progresspopup.WithContext(ctx),
		progresspopup.WithCancel(),
	)
}
