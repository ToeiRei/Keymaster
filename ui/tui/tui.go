// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/client/mock"
	"github.com/toeirei/keymaster/client/testui"
	"github.com/toeirei/keymaster/tags"
	"github.com/toeirei/keymaster/ui/tui/views/root"
)

func Run() error {
	ct := testui.NewClient()

	// test accounts
	_, _ = ct.CreateAccount(context.Background(), "root", "1.2.3.4", 22, "ssh", "password123")
	_, _ = ct.CreateAccount(context.Background(), "user", "1.2.3.4", 22, "ssh", "password123")
	_, _ = ct.CreateAccount(context.Background(), "srv", "10.0.0.1", 22, "ssh", "password123")
	_, _ = ct.CreateAccount(context.Background(), "mark", "1.2.3.4", 22, "ssh", "password123")
	_, _ = ct.CreateAccount(context.Background(), "admin", "10.20.0.1", 222, "cisco", "password123")
	// test publicKeys
	_, _ = ct.CreatePublicKey(context.Background(), "Sha-your-mom ashtdjhk-fbaskjdfhal_sdvkhaösdljhask-zdpjwb", "my-key", tags.Tags{"user:jannes", "company:work", "server-ci"})
	_, _ = ct.CreatePublicKey(context.Background(), "Sha-your-mom ashtdjhk-fbaskjdfhal_sdvkhaösdljhask-öutyfb", "my-key", tags.Tags{"user:jannes", "company:none"})
	_, _ = ct.CreatePublicKey(context.Background(), "Sha-420 asdjhk-fbaskdasral_jklkhathrösdljhask-fdjfb", "419", tags.Tags{"user:toeirei", "company:big_money"})
	_, _ = ct.CreatePublicKey(context.Background(), "Sha-420 asdjhk-fbaskjdfhal_sdvtzuthrösdljhaha-ögjfb", "420", tags.Tags{"user:toeirei", "company:work", "server-ci"})
	_, _ = ct.CreatePublicKey(context.Background(), "Sha-420 asdjhk-fbaskjterhl_sdvkhaghdjfdljhask-ödhfb", "421", tags.Tags{"user:toeirei", "company:none"})
	_, _ = ct.CreatePublicKey(context.Background(), "Sha-69 asdjkhk-fbdfhtdftrhhal_sdvkhaösu656zsk-ödjhtfb", "69", tags.Tags{"user:somebodyelse", "company:evilgoogle", "server-ci"})
	// test links
	_, _ = ct.CreateLink(context.Background(), 1, "(user:jannes | user:toeirei) & !company:work", time.Now().Add(time.Hour))
	_, _ = ct.CreateLink(context.Background(), 2, "!user:somebodyelse", time.Now().Add(time.Hour))
	_, _ = ct.CreateLink(context.Background(), 3, "server-ci", time.Now().Add(time.Hour))
	_, _ = ct.CreateLink(context.Background(), 4, "company:evilgoogle", time.Now().Add(time.Hour))
	_, _ = ct.CreateLink(context.Background(), 5, "company:work", time.Now().Add(time.Hour))
	_, _ = ct.CreateLink(context.Background(), 5, "company:big_money", time.Now())
	// test auditLogs
	// TODO add more mock data for testing
	_ = ct.AddAuditLog(map[string]string{}, "Doing", "Something")

	// add delay "middleware"
	cm := mock.NewClient(mock.WitchBaseClient(ct), mock.WitchPre(func(method string, args map[string]any) error {
		time.Sleep(time.Millisecond * 100)
		if ctx, ok := args["ctx"].(context.Context); ok {
			return ctx.Err()
		}
		return nil
	}))

	// create and run tea programm
	_, err := tea.NewProgram(root.New(cm), tea.WithAltScreen()).Run()
	return err
}
