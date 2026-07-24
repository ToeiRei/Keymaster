// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package deploy

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/ui/tui/popups/messagepopup"
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
	return runInteractive(ctx, "Deploying Accounts", "deployment", c.DeployAccounts, accounts...)
}
