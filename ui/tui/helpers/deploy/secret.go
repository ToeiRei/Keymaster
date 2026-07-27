// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package deploy

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/ui/i18n"
)

// UpdateSecret installs a new connector secret on an account's target, through
// the same interactive machinery as a deploy. That is what gives the connector's
// host key handshake and the client's question about an unfinished secret update
// somewhere to ask: both arrive as user requests on the progress popup.
func UpdateSecret(ctx context.Context, c client.Client, account client.Account, connectorSecret map[string]string) tea.Cmd {
	return runInteractive(
		ctx,
		i18n.Text("deploy.op_title_update_secret"),
		i18n.Text("deploy.op_noun_update_secret"),
		singleAccountOperation(func(ctx context.Context, userRequester client.UserRequester, progress chan<- client.UpdateSecretProgressAccount, accountId client.AccountId) error {
			return c.UpdateAccountSecret(ctx, userRequester, progress, accountId, connectorSecret)
		}),
		account,
	)
}

// singleAccountOperation adapts a one-account operation to the aggregate shape
// [runInteractive] drives, mirroring what the client does in the other direction
// when it projects a batch down onto a single account.
func singleAccountOperation(
	op func(ctx context.Context, userRequester client.UserRequester, progress chan<- client.UpdateSecretProgressAccount, accountId client.AccountId) error,
) accountOperation {
	return func(ctx context.Context, userRequester client.UserRequester, aggregate chan<- client.ProgressAccounts, accountIds ...client.AccountId) error {
		if len(accountIds) != 1 {
			return i18n.NewError("deploy.op_single_account_only")
		}
		accountId := accountIds[0]

		// A fresh snapshot per update, rather than one entry mutated in place: the
		// receiver renders what it was handed while this goroutine carries on, so a
		// shared entry could be read half-advanced.
		send := func(progress client.ProgressAccount, err error) {
			aggregate <- client.ProgressAccounts{Accounts: map[client.AccountId]*client.ProgressAccountWithError{
				accountId: {progress, err},
			}}
		}

		// Sent before the operation can fail: runInteractive builds its final
		// message from the last snapshot it saw, and reads a missing one as "never
		// started", which would report a failure as a cancellation.
		send(client.ProgressAccount{0, i18n.Text("client.status.not_started")}, nil)

		single := make(chan client.UpdateSecretProgressAccount)
		var opErr error
		go func() {
			defer close(single)
			opErr = op(ctx, userRequester, single, accountId)
		}()
		for p := range single {
			send(p, nil)
		}

		if opErr != nil {
			// Carried on the account rather than returned, so it renders as this
			// account's own failure the way a batch operation's would.
			send(client.ProgressAccount{1, i18n.Text("client.status.error")}, opErr)
		}
		return nil
	}
}
