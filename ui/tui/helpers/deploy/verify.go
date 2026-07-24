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

func VerifyAll(ctx context.Context, c client.Client) tea.Cmd {
	accounts, err := c.ListAccounts(ctx)
	if err != nil {
		return messagepopup.Open(messagepopup.Error, err.Error(), nil)
	}

	return Verify(ctx, c, accounts...)
}

func VerifyDirty(ctx context.Context, c client.Client) tea.Cmd {
	accounts, err := c.ListAccountsDirty(ctx)
	if err != nil {
		return messagepopup.Open(messagepopup.Error, err.Error(), nil)
	}

	return Verify(ctx, c, accounts...)
}

func Verify(ctx context.Context, c client.Client, accounts ...client.Account) tea.Cmd {
	return runInteractive(ctx, "Verifying Accounts", "verify", c.VerifyAccounts, accounts...)
}
