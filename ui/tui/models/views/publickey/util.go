// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package publickey

import (
	"strings"

	"github.com/bobg/go-generics/v4/slices"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	popupviews "github.com/toeirei/keymaster/ui/tui/models/views/popup"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

func tagsParse(tags string) []string {
	return slices.Filter( // remove empty user provided tags
		slices.Map( // trim user provided tags
			strings.Split(tags, ","), // split user provided tags
			func(tag string) string { return strings.TrimSpace(tag) },
		),
		func(tag string) bool { return tag != "" },
	)
}

func tagsStringify(tags []string) string {
	return strings.Join(tags, ", ")
}

func discardGuard(confirmCmd tea.Cmd) tea.Cmd {
	return popupviews.OpenChoice(
		"You have unsaved changes. Do you want to discard them?",
		popupviews.Choices{
			{Name: "Cancel", Cmd: nil, KeyBindings: form.GlobalKeyMap{keys.Cancel()}},
			{Name: "Discard", Cmd: confirmCmd},
		},
		40, 40,
	)
}
