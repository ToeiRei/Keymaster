// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package tui

import (
	"context"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/client/bun"
	"github.com/toeirei/keymaster/ui/tui/views/root"
)

func Run() error {
	logger := log.New(os.Stdout, "[tui] ", log.LstdFlags)
	cm, err := bun.NewDefaultBunClient(logger)
	if err != nil {
		return err
	}
	defer func() {
		_ = cm.Close(context.Background())
	}()

	return RunWithClient(cm)
}

func RunWithClient(cm client.Client) error {
	_, err := tea.NewProgram(root.New(cm), tea.WithAltScreen()).Run()
	return err
}
