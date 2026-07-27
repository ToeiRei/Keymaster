// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package account

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/toeirei/keymaster/client"
	clientmock "github.com/toeirei/keymaster/client/mock"
)

type forceCall struct {
	id           client.AccountId
	connectorKey string
	secret       map[string]string
}

// forceClient records every override attempt, so a test can assert one happened
// with the submitted values -- or that none did.
func forceClient(err error) (client.Client, *[]forceCall) {
	var calls []forceCall
	c := clientmock.NewClient(clientmock.WitchOverwrites(clientmock.ClientOverwrites{
		UpdateAccountConnectorForce: func(_ context.Context, id client.AccountId, connectorKey string, secret map[string]string) (client.Account, error) {
			calls = append(calls, forceCall{id, connectorKey, secret})
			return client.Account{}, err
		},
	}))
	return c, &calls
}

func testAccount() client.Account {
	return client.Account{Id: 7, Username: "root", Host: "example.com", Port: 22, Connector: "mock"}
}

func testSubmitted(connectorKey string) connectorData {
	return connectorData{connectorKey, map[string]string{"succeed": "true", "key": "manual"}}
}

// Cancel is first so it holds focus, and carries no command at all, so the safe
// option cannot reach the client even by accident.
func TestConnectorForceChoices_CancelIsFirstAndInert(t *testing.T) {
	c, calls := forceClient(nil)

	choices := connectorForceChoices(c, testAccount(), testSubmitted("mock"), func() {
		t.Error("no choice was made, nothing should have been recorded")
	})

	if len(choices) != 2 {
		t.Fatalf("expected a cancel and a confirm choice, got %d", len(choices))
	}
	if choices[0].Cmd != nil {
		t.Error("the cancel choice must not carry a command")
	}
	if len(choices[0].KeyBindings) == 0 {
		t.Error("the cancel choice should be reachable by the cancel key")
	}
	if choices[1].Cmd == nil {
		t.Fatal("the confirm choice must carry the override command")
	}
	if len(*calls) != 0 {
		t.Fatalf("building the guard must not record anything, got %+v", *calls)
	}
}

func TestConnectorForceChoices_ConfirmRecordsAndReloads(t *testing.T) {
	c, calls := forceClient(nil)
	account := testAccount()
	submitted := testSubmitted("mock")

	reloaded := false
	choices := connectorForceChoices(c, account, submitted, func() { reloaded = true })
	choices[1].Cmd()

	if len(*calls) != 1 {
		t.Fatalf("expected exactly one override, got %+v", *calls)
	}
	got := (*calls)[0]
	if got.id != account.Id {
		t.Errorf("overrode account %v, want %v", got.id, account.Id)
	}
	if got.connectorKey != submitted.Key {
		t.Errorf("overrode connector %q, want %q", got.connectorKey, submitted.Key)
	}
	if got.secret["key"] != "manual" {
		t.Errorf("submitted values did not reach the client: %+v", got.secret)
	}
	if !reloaded {
		t.Error("an override changes the dirty column, so the list has to reload")
	}
}

// A failed override must not report success, and must not mark the list as
// changed, since nothing was.
func TestConnectorForceChoices_FailureDoesNotReload(t *testing.T) {
	c, calls := forceClient(errors.New("write refused"))

	choices := connectorForceChoices(c, testAccount(), testSubmitted("mock"), func() {
		t.Error("a failed override must not mark the list for reload")
	})
	choices[1].Cmd()

	if len(*calls) != 1 {
		t.Fatalf("expected the override to be attempted once, got %+v", *calls)
	}
}

// Replacing the connector is a bigger change than replacing the secret -- the
// account keeps its key assignments and loses everything else -- so the question
// has to say so rather than reuse the milder wording.
func TestConnectorForceQuestion_CallsOutAConnectorChange(t *testing.T) {
	account := testAccount()

	same := connectorForceQuestion(account, testSubmitted(account.Connector)).String()
	changed := connectorForceQuestion(account, testSubmitted("ssh")).String()

	if same == changed {
		t.Fatal("a connector change should be worded differently from a plain secret override")
	}
	for _, want := range []string{account.Connector, "ssh"} {
		if !strings.Contains(changed, want) {
			t.Errorf("the question should name %q, got %q", want, changed)
		}
	}
	// Both name the target, which is what stops it being fired at the wrong row.
	for _, question := range []string{same, changed} {
		if !strings.Contains(question, account.Host) {
			t.Errorf("the question should name the target, got %q", question)
		}
	}
}
