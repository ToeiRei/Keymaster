// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package tui provides the bootstrap TUI workflow for adding new hosts.
// This file implements a multi-step wizard that guides users through
// the process of securely bootstrapping a new host with temporary keys.
package tui // import "github.com/toeirei/keymaster/internal/tui"

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/bootstrap"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/deploy"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/ui"
	"golang.org/x/crypto/ssh"
)

// bootstrapStep represents the current step in the bootstrap workflow.
type bootstrapStep int

const (
	stepGenerateKey    bootstrapStep = iota // Generate temporary key and show command
	stepWaitConfirm                         // Wait for user to confirm key is installed
	stepVerifyHostKey                       // Show and verify server host key
	stepTestConnection                      // Test that temporary key works
	stepSelectKeys                          // Select which keys to assign
	stepConfirmDeploy                       // Show final confirmation
	stepDeploying                           // Execute deployment
	stepComplete                            // Show success/failure result
)

// bootstrapModel represents the state of the bootstrap workflow.
type bootstrapModel struct {
	step    bootstrapStep
	session *bootstrap.BootstrapSession

	// Account data for creating the session
	pendingUsername string
	pendingHostname string
	pendingLabel    string
	pendingTags     string

	// UI components
	confirmCursor int // For yes/no confirmations

	// Key selection (using same pattern as assign_keys.go)
	availableKeys []model.PublicKey // User-selectable keys (non-global)
	globalKeys    []model.PublicKey // Global keys (automatically deployed)
	keysCursor    int               // Current cursor position in key selection
	selectedKeys  map[int]struct{}  // Set of selected key IDs

	// State tracking
	deploymentResult error
	isCompleted      bool
	err              error

	// Connection testing
	connectionTested bool
	testInProgress   bool

	// Host key verification
	hostKey          string // The host key in authorized_keys format
	hostKeyRetrieved bool
	hostKeyVerified  bool

	// Clipboard status
	commandCopied        bool   // For bootstrap command
	verifyCommandCopied  bool   // For ssh-keygen verify command
	currentVerifyCommand string // Current verify command for copying
}

// Bootstrap workflow messages
type (
	// sessionCreatedMsg indicates a bootstrap session was successfully created
	sessionCreatedMsg struct {
		session *bootstrap.BootstrapSession
	}

	// hostKeyRetrievedMsg indicates host key retrieval completed
	hostKeyRetrievedMsg struct {
		hostKey string
		err     error
	}

	// connectionTestMsg indicates connection test completed
	connectionTestMsg struct {
		success bool
		err     error
	}

	// deploymentCompleteMsg indicates deployment finished
	deploymentCompleteMsg struct {
		account model.Account
		err     error
	}

	// stepCompleteMsg indicates the current step is done and should advance
	stepCompleteMsg struct{}
)

// newBootstrapModel creates a new bootstrap workflow model.
func newBootstrapModel(username, hostname, label, tags string) *bootstrapModel {
	m := &bootstrapModel{
		step:          stepGenerateKey,
		confirmCursor: 0,
		selectedKeys:  make(map[int]struct{}),
		// Store the account data for later use
		pendingUsername: username,
		pendingHostname: hostname,
		pendingLabel:    label,
		pendingTags:     tags,
	}

	return m
}

// Init initializes the bootstrap model.
func (m *bootstrapModel) Init() tea.Cmd {
	return m.createBootstrapSession()
}

// createBootstrapSession creates a new bootstrap session asynchronously.
func (m *bootstrapModel) createBootstrapSession() tea.Cmd {
	return func() tea.Msg {
		// Create session with the actual account data
		session, err := bootstrap.NewBootstrapSession(m.pendingUsername, m.pendingHostname, m.pendingLabel, m.pendingTags)
		if err != nil {
			return fmt.Errorf("failed to create bootstrap session: %w", err)
		}

		// Register session for cleanup
		bootstrap.RegisterSession(session)

		// Save session to database
		if err := session.Save(); err != nil {
			session.Cleanup()
			return fmt.Errorf("failed to save bootstrap session: %w", err)
		}

		return sessionCreatedMsg{session: session}
	}
}

// Update handles messages and updates the bootstrap model state.
func (m *bootstrapModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case sessionCreatedMsg:
		m.session = msg.session
		m.commandCopied = false // Reset copied flag for new session
		return m, nil

	case error:
		m.err = msg
		return m, nil

	case hostKeyRetrievedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.hostKey = msg.hostKey
			m.hostKeyRetrieved = true
		}
		return m, nil

	case connectionTestMsg:
		m.testInProgress = false
		if msg.success {
			m.connectionTested = true
			m.step = stepSelectKeys
			return m, m.loadAvailableKeys()
		} else {
			m.err = msg.err
		}
		return m, nil

	case deploymentCompleteMsg:
		m.deploymentResult = msg.err
		m.step = stepComplete
		m.isCompleted = true

		// Unregister session from cleanup registry
		if m.session != nil {
			bootstrap.UnregisterSession(m.session.ID)
		}

		if msg.err == nil {
			// Success - return to accounts list with new account
			return m, func() tea.Msg {
				return accountModifiedMsg{
					isNew:     true,
					username:  msg.account.Username,
					hostname:  msg.account.Hostname,
					accountID: msg.account.ID,
				}
			}
		}

		return m, nil

	case stepCompleteMsg:
		return m.advanceStep()
	}

	return m, nil
}

// handleKeyMsg processes keyboard input for the current step.
func (m *bootstrapModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle error state with recovery options
	if m.err != nil {
		return m.handleErrorKeys(msg)
	}

	switch msg.String() {
	case "esc":
		// Cancel bootstrap and go back
		if m.session != nil {
			// Log the abort
			_ = logAction("BOOTSTRAP_FAILED", fmt.Sprintf("%s@%s, reason: aborted by user",
				m.session.PendingAccount.Username, m.session.PendingAccount.Hostname))
			bootstrap.UnregisterSession(m.session.ID)
			_ = m.session.Delete()
		}
		return m, func() tea.Msg { return backToListMsg{} }

	case "q":
		if m.step == stepComplete {
			return m, func() tea.Msg { return backToListMsg{} }
		}
	}

	switch m.step {
	case stepGenerateKey:
		return m.handleGenerateKeyKeys(msg)
	case stepWaitConfirm:
		return m.handleWaitConfirmKeys(msg)
	case stepVerifyHostKey:
		return m.handleVerifyHostKeyKeys(msg)
	case stepTestConnection:
		return m.handleTestConnectionKeys(msg)
	case stepSelectKeys:
		return m.handleSelectKeysKeys(msg)
	case stepConfirmDeploy:
		return m.handleConfirmDeployKeys(msg)
	case stepDeploying:
		// No input during deployment
		return m, nil
	case stepComplete:
		if msg.String() == "enter" || msg.String() == "q" {
			return m, func() tea.Msg { return backToListMsg{} }
		}
	}

	return m, nil
}

// handleErrorKeys handles input during error state with recovery options.
func (m *bootstrapModel) handleErrorKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "left", "right", "h", "l":
		// Navigate between error recovery options (0: retry, 1: regenerate, 2: cancel)
		if msg.String() == "left" || msg.String() == "h" {
			m.confirmCursor = (m.confirmCursor - 1 + 3) % 3
		} else {
			m.confirmCursor = (m.confirmCursor + 1) % 3
		}

	case "enter":
		switch m.confirmCursor {
		case 0: // Retry - clear error and retry current step
			m.err = nil
			m.confirmCursor = 0 // Reset for next use
			switch m.step {
			case stepTestConnection:
				// Retry connection test
				m.testInProgress = true
				return m, m.testConnection()
			case stepGenerateKey:
				// Retry session creation
				return m, m.createBootstrapSession()
			default:
				// For other steps, just clear the error and let user continue
				return m, nil
			}

		case 1: // Regenerate - create new temporary key
			m.err = nil
			m.confirmCursor = 0 // Reset for next use
			// Go back to key generation step with new session
			m.step = stepGenerateKey
			return m, m.generateNewSession()

		case 2: // Cancel - go back to accounts list
			if m.session != nil {
				// Log the abort
				_ = logAction("BOOTSTRAP_FAILED", fmt.Sprintf("%s@%s, reason: aborted by user",
					m.session.PendingAccount.Username, m.session.PendingAccount.Hostname))
				bootstrap.UnregisterSession(m.session.ID)
				_ = m.session.Delete()
			}
			return m, func() tea.Msg { return backToListMsg{} }
		}

	case "esc":
		// Same as cancel
		if m.session != nil {
			// Log the abort
			_ = logAction("BOOTSTRAP_FAILED", fmt.Sprintf("%s@%s, reason: aborted by user",
				m.session.PendingAccount.Username, m.session.PendingAccount.Hostname))
			bootstrap.UnregisterSession(m.session.ID)
			_ = m.session.Delete()
		}
		return m, func() tea.Msg { return backToListMsg{} }
	}

	return m, nil
}

// handleGenerateKeyKeys handles input during the key generation/display step.
func (m *bootstrapModel) handleGenerateKeyKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "c":
		// Copy command to clipboard
		if m.session != nil {
			command := m.session.GetBootstrapCommand()
			if err := clipboard.WriteAll(command); err == nil {
				m.commandCopied = true
			}
		}
		return m, nil

	case "enter":
		// Advance to next step (wait for confirmation)
		m.step = stepWaitConfirm
		return m, nil
	}

	return m, nil
}

// handleWaitConfirmKeys handles input during the confirmation waiting step.
func (m *bootstrapModel) handleWaitConfirmKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "left", "right", "h", "l":
		m.confirmCursor = 1 - m.confirmCursor // Toggle between 0 and 1

	case "enter":
		if m.confirmCursor == 0 { // "Yes, I installed it"
			m.step = stepVerifyHostKey
			return m, m.retrieveHostKey()
		} else { // "Generate new command"
			// Generate new temporary key and go back to key generation step
			m.step = stepGenerateKey
			return m, m.generateNewSession()
		}
	}

	return m, nil
}

// handleTestConnectionKeys handles input during connection testing step.
func (m *bootstrapModel) handleTestConnectionKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Only handle input if test is not in progress and there's an error
	if m.testInProgress {
		return m, nil
	}

	switch msg.String() {
	case "r":
		// Retry connection test
		if !m.testInProgress {
			m.err = nil // Clear previous error
			m.testInProgress = true
			return m, m.testConnection()
		}

	case "tab", "b":
		// Go back to waiting for confirmation step
		m.err = nil // Clear error
		m.step = stepWaitConfirm
		m.confirmCursor = 0 // Reset to "Yes, I installed it"
	}

	return m, nil
}

// handleSelectKeysKeys handles input during key selection.
func (m *bootstrapModel) handleSelectKeysKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if len(m.availableKeys) > 0 {
			m.keysCursor = (m.keysCursor - 1 + len(m.availableKeys)) % len(m.availableKeys)
		}

	case "down", "j":
		if len(m.availableKeys) > 0 {
			m.keysCursor = (m.keysCursor + 1) % len(m.availableKeys)
		}

	case " ":
		// Toggle selection of current item (using same pattern as assign_keys.go)
		if m.keysCursor < len(m.availableKeys) {
			keyID := m.availableKeys[m.keysCursor].ID
			if _, ok := m.selectedKeys[keyID]; ok {
				delete(m.selectedKeys, keyID)
			} else {
				m.selectedKeys[keyID] = struct{}{}
			}
		}

	case "enter", "tab":
		m.step = stepConfirmDeploy
		m.confirmCursor = 0
		return m, nil
	}

	return m, nil
}

// handleConfirmDeployKeys handles input during final confirmation.
func (m *bootstrapModel) handleConfirmDeployKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "left", "right", "h", "l":
		m.confirmCursor = 1 - m.confirmCursor // Toggle between 0 and 1

	case "enter":
		if m.confirmCursor == 0 { // "Deploy"
			m.step = stepDeploying
			return m, m.executeDeployment()
		} else { // "Back"
			// Only allow going back to key selection if connection was successful
			if m.connectionTested {
				m.step = stepSelectKeys
			}
		}

	case "tab":
		// Only allow going back to key selection if connection was successful
		if m.connectionTested {
			m.step = stepSelectKeys
		}
	}

	return m, nil
}

// View renders the bootstrap workflow UI.
func (m *bootstrapModel) View() string {
	if m.err != nil {
		return m.viewError()
	}

	if m.session != nil {
		bootstrap.UnregisterSession(m.session.ID)
		_ = m.session.Delete()
	}
	switch m.step {
	case stepGenerateKey:
		return m.viewGenerateKey()
	case stepWaitConfirm:
		return m.viewWaitConfirm()
	case stepVerifyHostKey:
		return m.viewVerifyHostKey()
	case stepTestConnection:
		return m.viewTestConnection()
	case stepSelectKeys:
		return m.viewSelectKeys()
	case stepConfirmDeploy:
		return m.viewConfirmDeploy()
	case stepDeploying:
		return m.viewDeploying()
	case stepComplete:
		return m.viewComplete()
	}

	return "Unknown step"
}

// viewGenerateKey renders the key generation step.
func (m *bootstrapModel) viewGenerateKey() string {
	var content []string

	content = append(content, titleStyle.Render("🚀 "+i18n.T("bootstrap.title")))
	content = append(content, "")

	if m.err != nil {
		content = append(content, errorStyle.Render(i18n.T("bootstrap.error_prefix")+m.err.Error()))
		content = append(content, "")
	}

	content = append(content, i18n.T("bootstrap.step1_description"))
	content = append(content, "")

	// Show the command to paste
	command := m.session.GetBootstrapCommand()

	commandBox := dialogBoxStyle.
		BorderForeground(colorHighlight).
		Width(80).
		Render(command)

	content = append(content, commandBox)
	content = append(content, "")

	// Show copy status or help
	if m.commandCopied {
		content = append(content, successStyle.Render(i18n.T("bootstrap.step1_copied")))
	} else {
		content = append(content, helpStyle.Render(i18n.T("bootstrap.step1_copy_hint")))
	}

	// Main pane using shared dialog style
	mainContent := dialogBoxStyle.
		BorderForeground(colorSubtle).
		Width(90).
		Render(lipgloss.JoinVertical(lipgloss.Left, content...))

	// Help footer using shared help style with background
	helpFooterStyle := helpStyle.
		Background(lipgloss.Color("236")).
		Padding(0, 1).
		Italic(true)

	helpFooter := helpFooterStyle.Render(i18n.T("bootstrap.step1_help"))

	return lipgloss.JoinVertical(lipgloss.Left, mainContent, "", helpFooter)
}

// viewWaitConfirm renders the confirmation waiting step.
func (m *bootstrapModel) viewWaitConfirm() string {
	var content []string

	content = append(content, titleStyle.Render("🔄 "+i18n.T("bootstrap.confirm_title")))
	content = append(content, "")

	content = append(content, i18n.T("bootstrap.confirm_description"))
	content = append(content, "")

	// Yes/No buttons using modern button styles
	var yesButton, noButton string
	if m.confirmCursor == 0 {
		yesButton = activeButtonStyle.Render(i18n.T("bootstrap.confirm_yes"))
		noButton = buttonStyle.Render(i18n.T("bootstrap.confirm_no"))
	} else {
		yesButton = buttonStyle.Render(i18n.T("bootstrap.confirm_yes"))
		noButton = activeButtonStyle.Render(i18n.T("bootstrap.confirm_no"))
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Left, yesButton, "  ", noButton)
	content = append(content, buttons)

	// Main pane using shared dialog style
	mainContent := dialogBoxStyle.
		BorderForeground(colorSubtle).
		Width(70).
		Render(lipgloss.JoinVertical(lipgloss.Left, content...))

	// Help footer using shared help style with background
	helpFooterStyle := helpStyle.
		Background(lipgloss.Color("236")).
		Padding(0, 1).
		Italic(true)

	helpFooter := helpFooterStyle.Render(i18n.T("bootstrap.confirm_help"))

	return lipgloss.JoinVertical(lipgloss.Left, mainContent, "", helpFooter)
}

// viewTestConnection renders the connection testing step.
func (m *bootstrapModel) viewTestConnection() string {
	var content []string

	content = append(content, titleStyle.Render("🔌 "+i18n.T("bootstrap.testing_title")))
	content = append(content, "")

	if m.testInProgress {
		content = append(content, i18n.T("bootstrap.testing_progress"))
	} else {
		if m.err != nil {
			content = append(content, errorStyle.Render(i18n.T("bootstrap.testing_failed_prefix")+m.err.Error()))
		} else {
			content = append(content, successStyle.Render(i18n.T("bootstrap.testing_success")))
		}
	}

	// Main pane using shared dialog style
	mainContent := dialogBoxStyle.
		BorderForeground(colorSubtle).
		Width(70).
		Render(lipgloss.JoinVertical(lipgloss.Left, content...))

	// Help footer using shared help style with background
	helpFooterStyle := helpStyle.
		Background(lipgloss.Color("236")).
		Padding(0, 1).
		Italic(true)

	var helpText string
	if m.err != nil {
		helpText = i18n.T("bootstrap.testing_retry_help")
	} else {
		helpText = "Connection test completed"
	}
	helpFooter := helpFooterStyle.Render(helpText)

	return lipgloss.JoinVertical(lipgloss.Left, mainContent, "", helpFooter)
}

// viewSelectKeys renders the key selection step.
func (m *bootstrapModel) viewSelectKeys() string {
	var content []string

	content = append(content, titleStyle.Render("🔑 "+i18n.T("bootstrap.select_keys_title")))
	content = append(content, "")

	content = append(content, i18n.T("bootstrap.select_keys_description"))
	content = append(content, "")

	// Show Global Keys section first (non-selectable, informational)
	if len(m.globalKeys) > 0 {
		content = append(content, titleStyle.Render("Global Keys (automatically deployed):"))
		for _, key := range m.globalKeys {
			item := fmt.Sprintf("  ✓ %s (%s)", key.Comment, key.Algorithm)
			content = append(content, inactiveItemStyle.Render(item))
		}
		content = append(content, "")
	}

	// Show User-selectable Keys section
	if len(m.availableKeys) > 0 {
		content = append(content, titleStyle.Render("Select Additional Keys:"))

		var listItems []string
		for i, key := range m.availableKeys {
			// Determine checkbox state
			checked := i18n.T("assign_keys.checkmark_unchecked")
			if _, ok := m.selectedKeys[key.ID]; ok {
				checked = i18n.T("assign_keys.checkmark_checked")
			}

			// Cursor indicator
			cursor := "  "
			if m.keysCursor == i {
				cursor = "▸ "
			}

			// Format the item
			item := fmt.Sprintf("%s%s %s (%s)", cursor, checked, key.Comment, key.Algorithm)

			// Style based on selection
			if _, ok := m.selectedKeys[key.ID]; ok {
				listItems = append(listItems, selectedItemStyle.Render(item))
			} else {
				listItems = append(listItems, itemStyle.Render(item))
			}
		}

		content = append(content, lipgloss.JoinVertical(lipgloss.Left, listItems...))
	} else {
		content = append(content, titleStyle.Render("Select Additional Keys:"))
		content = append(content, inactiveItemStyle.Render("  (No additional keys available)"))
	}

	// Main pane using shared dialog style
	mainContent := dialogBoxStyle.
		BorderForeground(colorSubtle).
		Width(80).
		Render(lipgloss.JoinVertical(lipgloss.Left, content...))

	// Help footer using shared help style with background
	helpFooterStyle := helpStyle.
		Background(lipgloss.Color("236")).
		Padding(0, 1).
		Italic(true)

	helpFooter := helpFooterStyle.Render(i18n.T("bootstrap.select_keys_help"))

	return lipgloss.JoinVertical(lipgloss.Left, mainContent, "", helpFooter)
}

// viewConfirmDeploy renders the final confirmation step.
func (m *bootstrapModel) viewConfirmDeploy() string {
	var content []string

	content = append(content, titleStyle.Render("✅ "+i18n.T("bootstrap.confirm_deploy_title")))
	content = append(content, "")

	// Show what will be deployed
	username := ""
	hostname := ""

	// Get values from session first, then fallback to pending values
	if m.session != nil {
		username = m.session.PendingAccount.Username
		hostname = m.session.PendingAccount.Hostname
	}

	// Use fallback if values are still empty
	if username == "" {
		username = m.pendingUsername
	}
	if hostname == "" {
		hostname = m.pendingHostname
	}

	// Ensure we have valid values
	if username == "" {
		username = "unknown"
	}
	if hostname == "" {
		hostname = "unknown"
	}

	content = append(content, fmt.Sprintf("Will create account: %s@%s", username, hostname))

	// Count selected keys plus global keys
	selectedCount := len(m.selectedKeys)
	globalCount := len(m.globalKeys)
	totalKeys := selectedCount + globalCount

	content = append(content, fmt.Sprintf("Will deploy %d selected keys", totalKeys))
	content = append(content, "Will deploy Keymaster System Key for management")
	content = append(content, i18n.T("bootstrap.will_replace_temp_key"))
	content = append(content, "")

	// Deploy/Back buttons using modern button styles
	var deployButton, backButton string
	if m.confirmCursor == 0 {
		deployButton = activeButtonStyle.Render(i18n.T("bootstrap.deploy"))
		backButton = buttonStyle.Render(i18n.T("bootstrap.back"))
	} else {
		deployButton = buttonStyle.Render(i18n.T("bootstrap.deploy"))
		backButton = activeButtonStyle.Render(i18n.T("bootstrap.back"))
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Left, deployButton, "  ", backButton)
	content = append(content, buttons)

	// Main pane using shared dialog style
	mainContent := dialogBoxStyle.
		BorderForeground(colorSubtle).
		Width(80).
		Render(lipgloss.JoinVertical(lipgloss.Left, content...))

	// Help footer using shared help style with background
	helpFooterStyle := helpStyle.
		Background(lipgloss.Color("236")).
		Padding(0, 1).
		Italic(true)

	helpFooter := helpFooterStyle.Render(i18n.T("bootstrap.confirm_deploy_help"))

	return lipgloss.JoinVertical(lipgloss.Left, mainContent, "", helpFooter)
}

// viewDeploying renders the deployment progress step.
func (m *bootstrapModel) viewDeploying() string {
	var content []string

	content = append(content, titleStyle.Render("🚀 "+i18n.T("bootstrap.deploying_title")))
	content = append(content, "")

	content = append(content, i18n.T("bootstrap.deploying_progress"))
	content = append(content, "")

	// Simple progress indicator
	content = append(content, "⏳ Deploying...")

	// Main pane using shared dialog style
	mainContent := dialogBoxStyle.
		BorderForeground(colorSubtle).
		Width(60).
		Render(lipgloss.JoinVertical(lipgloss.Left, content...))

	// Help footer using shared help style with background
	helpFooterStyle := helpStyle.
		Background(lipgloss.Color("236")).
		Padding(0, 1).
		Italic(true)

	helpFooter := helpFooterStyle.Render(i18n.T("bootstrap.deploying_help"))

	return lipgloss.JoinVertical(lipgloss.Left, mainContent, "", helpFooter)
}

// viewComplete renders the completion step.
func (m *bootstrapModel) viewComplete() string {
	var content []string

	if m.deploymentResult == nil {
		content = append(content, titleStyle.Render("✅ "+i18n.T("bootstrap.success_title")))
		content = append(content, "")
		content = append(content, successStyle.Render(fmt.Sprintf(i18n.T("bootstrap.success_message"),
			m.session.PendingAccount.Username, m.session.PendingAccount.Hostname)))
	} else {
		content = append(content, titleStyle.Render("❌ "+i18n.T("bootstrap.failed_title")))
		content = append(content, "")
		content = append(content, errorStyle.Render(fmt.Sprintf(i18n.T("bootstrap.failed_message"), m.deploymentResult.Error())))
	}

	// Main pane using shared dialog style
	mainContent := dialogBoxStyle.
		BorderForeground(colorSubtle).
		Width(70).
		Render(lipgloss.JoinVertical(lipgloss.Left, content...))

	// Help footer using shared help style with background
	helpFooterStyle := helpStyle.
		Background(lipgloss.Color("236")).
		Padding(0, 1).
		Italic(true)

	helpFooter := helpFooterStyle.Render(i18n.T("bootstrap.complete_help"))

	return lipgloss.JoinVertical(lipgloss.Left, mainContent, "", helpFooter)
}

// viewError renders the error state with recovery options.
func (m *bootstrapModel) viewError() string {
	var content []string

	content = append(content, titleStyle.Render("❌ "+i18n.T("bootstrap.error_title")))
	content = append(content, "")

	// Show the error message
	content = append(content, errorStyle.Render(i18n.T("bootstrap.error_message_prefix")+m.err.Error()))
	content = append(content, "")

	// Show recovery options
	content = append(content, titleStyle.Render(i18n.T("bootstrap.error_options")))
	content = append(content, "")

	// Recovery option buttons
	var retryButton, regenerateButton, cancelButton string

	switch m.confirmCursor {
	case 0: // Retry
		retryButton = activeButtonStyle.Render(i18n.T("bootstrap.error_retry"))
		regenerateButton = buttonStyle.Render(i18n.T("bootstrap.error_regenerate"))
		cancelButton = buttonStyle.Render(i18n.T("bootstrap.error_cancel"))
	case 1: // Regenerate
		retryButton = buttonStyle.Render(i18n.T("bootstrap.error_retry"))
		regenerateButton = activeButtonStyle.Render(i18n.T("bootstrap.error_regenerate"))
		cancelButton = buttonStyle.Render(i18n.T("bootstrap.error_cancel"))
	case 2: // Cancel
		retryButton = buttonStyle.Render(i18n.T("bootstrap.error_retry"))
		regenerateButton = buttonStyle.Render(i18n.T("bootstrap.error_regenerate"))
		cancelButton = activeButtonStyle.Render(i18n.T("bootstrap.error_cancel"))
	}

	buttonRow := lipgloss.JoinHorizontal(lipgloss.Left,
		retryButton, "  ",
		regenerateButton, "  ",
		cancelButton,
	)
	content = append(content, buttonRow)

	// Main pane using shared dialog style
	mainContent := dialogBoxStyle.
		BorderForeground(colorError).
		Width(80).
		Render(lipgloss.JoinVertical(lipgloss.Left, content...))

	// Help footer using shared help style with background
	helpFooterStyle := helpStyle.
		Background(lipgloss.Color("236")).
		Padding(0, 1).
		Italic(true)

	helpFooter := helpFooterStyle.Render(i18n.T("bootstrap.error_help"))

	return lipgloss.JoinVertical(lipgloss.Left, mainContent, "", helpFooter)
}

// handleVerifyHostKeyKeys handles input during host key verification step.
func (m *bootstrapModel) handleVerifyHostKeyKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "c":
		// Copy verify command to clipboard
		if m.currentVerifyCommand != "" {
			if err := clipboard.WriteAll(m.currentVerifyCommand); err == nil {
				m.verifyCommandCopied = true
			}
		}
		return m, nil

	case "left", "right", "h", "l":
		m.confirmCursor = 1 - m.confirmCursor // Toggle between 0 and 1

	case "enter":
		if m.confirmCursor == 0 { // "Accept"
			m.hostKeyVerified = true
			m.step = stepTestConnection
			m.testInProgress = true
			return m, m.testConnection()
		} else { // "Reject"
			// Go back to key generation step for new session
			m.step = stepGenerateKey
			return m, m.generateNewSession()
		}

	case "r":
		// Retry host key retrieval
		m.hostKeyRetrieved = false
		m.hostKey = ""
		m.err = nil
		return m, m.retrieveHostKey()

	case "tab":
		// Go back to waiting for confirmation step
		m.step = stepWaitConfirm
		m.confirmCursor = 0
	}

	return m, nil
}

// viewVerifyHostKey renders the host key verification step with proper styling.
func (m *bootstrapModel) viewVerifyHostKey() string {
	var content []string

	content = append(content, titleStyle.Render("🔐 "+i18n.T("bootstrap.verify_hostkey_title")))
	content = append(content, "")

	if !m.hostKeyRetrieved {
		content = append(content, "🔄 "+i18n.T("bootstrap.verify_hostkey_retrieving"))

		// Main pane using shared dialog style
		mainContent := dialogBoxStyle.
			BorderForeground(colorSubtle).
			Width(70).
			Render(lipgloss.JoinVertical(lipgloss.Left, content...))

		// Help footer
		helpFooterStyle := helpStyle.
			Background(lipgloss.Color("236")).
			Padding(0, 1).
			Italic(true)

		helpFooter := helpFooterStyle.Render(i18n.T("bootstrap.verify_hostkey_retrieving"))
		return lipgloss.JoinVertical(lipgloss.Left, mainContent, "", helpFooter)
	}

	if m.err != nil {
		content = append(content, errorStyle.Render(fmt.Sprintf(i18n.T("bootstrap.verify_hostkey_error_retrieving"), m.err.Error())))
		content = append(content, "")

		// Main pane
		mainContent := dialogBoxStyle.
			BorderForeground(colorError).
			Width(80).
			Render(lipgloss.JoinVertical(lipgloss.Left, content...))

		// Help footer
		helpFooterStyle := helpStyle.
			Background(lipgloss.Color("236")).
			Padding(0, 1).
			Italic(true)

		helpFooter := helpFooterStyle.Render(i18n.T("bootstrap.verify_hostkey_error_retry_help"))
		return lipgloss.JoinVertical(lipgloss.Left, mainContent, "", helpFooter)
	}

	// Show host key details
	content = append(content, i18n.T("bootstrap.verify_hostkey_server_key"))
	content = append(content, "")

	// Extract key type and generate fingerprints
	parts := strings.Fields(strings.TrimSpace(m.hostKey))
	if len(parts) >= 2 {
		keyType := parts[0]
		keyData := parts[1]

		// Decode the base64 key data for fingerprint calculation
		keyBytes, err := base64.StdEncoding.DecodeString(keyData)
		if err == nil {
			// Generate SHA256 fingerprint (modern)
			sha256Hash := sha256.Sum256(keyBytes)
			sha256Fingerprint := base64.StdEncoding.EncodeToString(sha256Hash[:])
			sha256Fingerprint = strings.TrimRight(sha256Fingerprint, "=")

			// Generate MD5 fingerprint (legacy, but still commonly used)
			md5Hash := md5.Sum(keyBytes)
			md5Fingerprint := ""
			for i, b := range md5Hash {
				if i > 0 {
					md5Fingerprint += ":"
				}
				md5Fingerprint += fmt.Sprintf("%02x", b)
			}

			content = append(content, fmt.Sprintf("🔑 %s: %s", i18n.T("bootstrap.verify_hostkey_type_label"), keyType))
			content = append(content, "")
			content = append(content, fmt.Sprintf("🔒 SHA256: %s", sha256Fingerprint))
			content = append(content, fmt.Sprintf("🔒 MD5:    %s", md5Fingerprint))
		} else {
			content = append(content, fmt.Sprintf("🔑 %s: %s", i18n.T("bootstrap.verify_hostkey_type_label"), keyType))
			content = append(content, i18n.T("bootstrap.verify_hostkey_fingerprint_error"))
		}
	} else {
		content = append(content, i18n.T("bootstrap.verify_hostkey_invalid_format"))
	}

	content = append(content, "")
	content = append(content, i18n.T("bootstrap.verify_hostkey_warning"))
	content = append(content, "")
	content = append(content, i18n.T("bootstrap.verify_hostkey_check_command"))
	content = append(content, "")

	// Generate specific ssh-keygen command based on key type
	sshKeygenCmd := "ssh-keygen -lf /etc/ssh/ssh_host_*_key.pub" // fallback
	if len(parts) >= 1 {
		keyType := parts[0]
		// Map SSH key types to their corresponding host key files
		switch keyType {
		case "ssh-rsa":
			sshKeygenCmd = "ssh-keygen -lf /etc/ssh/ssh_host_rsa_key.pub"
		case "ssh-dss":
			sshKeygenCmd = "ssh-keygen -lf /etc/ssh/ssh_host_dsa_key.pub"
		case "ecdsa-sha2-nistp256", "ecdsa-sha2-nistp384", "ecdsa-sha2-nistp521":
			sshKeygenCmd = "ssh-keygen -lf /etc/ssh/ssh_host_ecdsa_key.pub"
		case "ssh-ed25519":
			sshKeygenCmd = "ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub"
		default:
			sshKeygenCmd = "ssh-keygen -lf /etc/ssh/ssh_host_*_key.pub"
		}
	}

	// Store the command for copying
	m.currentVerifyCommand = sshKeygenCmd

	// Style the command with highlighting but no box
	styledCommand := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("220")).
		Padding(0, 1).
		Render(sshKeygenCmd)

	content = append(content, ""+styledCommand)
	content = append(content, "")

	// Show copy status or help for verify command (indented)
	if m.verifyCommandCopied {
		content = append(content, ""+successStyle.Render(i18n.T("bootstrap.verify_hostkey_copied")))
	} else {
		content = append(content, ""+helpStyle.Render(i18n.T("bootstrap.verify_hostkey_copy_hint")))
	}
	content = append(content, "")

	// Accept/Reject buttons using modern button styles
	var acceptButton, rejectButton string
	if m.confirmCursor == 0 {
		acceptButton = activeButtonStyle.Render(i18n.T("bootstrap.verify_hostkey_accept"))
		rejectButton = buttonStyle.Render(i18n.T("bootstrap.verify_hostkey_reject"))
	} else {
		acceptButton = buttonStyle.Render(i18n.T("bootstrap.verify_hostkey_accept"))
		rejectButton = activeButtonStyle.Render(i18n.T("bootstrap.verify_hostkey_reject"))
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Left, acceptButton, "  ", rejectButton)
	content = append(content, buttons)

	// Main pane using shared dialog style
	mainContent := dialogBoxStyle.
		BorderForeground(colorSubtle).
		Width(70).
		Render(lipgloss.JoinVertical(lipgloss.Left, content...))

	// Help footer
	helpFooterStyle := helpStyle.
		Background(lipgloss.Color("236")).
		Padding(0, 1).
		Italic(true)

	helpFooter := helpFooterStyle.Render(i18n.T("bootstrap.verify_hostkey_help"))
	return lipgloss.JoinVertical(lipgloss.Left, mainContent, "", helpFooter)
}

// Helper methods for async operations

// testConnection tests if the temporary key works by attempting an SSH connection.
func (m *bootstrapModel) testConnection() tea.Cmd {
	return func() tea.Msg {
		if m.session == nil || m.session.TempKeyPair == nil {
			return connectionTestMsg{success: false, err: fmt.Errorf("no temporary key pair available")}
		}

		// Parse the temporary private key for SSH connection
		signer, err := ssh.ParsePrivateKey(m.session.TempKeyPair.GetPrivateKeyPEM())
		if err != nil {
			return connectionTestMsg{success: false, err: fmt.Errorf("failed to parse temporary private key: %w", err)}
		}

		// Create host key callback using the verified host key
		hostKeyCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			presentedKey := string(ssh.MarshalAuthorizedKey(key))
			// Verify against the manually verified host key
			if strings.TrimSpace(presentedKey) != strings.TrimSpace(m.hostKey) {
				return fmt.Errorf("host key mismatch: server presented different key than verified")
			}
			return nil
		}

		// Create SSH client configuration
		config := &ssh.ClientConfig{
			User: m.session.PendingAccount.Username,
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
			HostKeyCallback: hostKeyCallback,
			Timeout:         10 * time.Second,
		}

		// Attempt to connect to the remote host
		conn, err := ssh.Dial("tcp", m.session.PendingAccount.Hostname+":22", config)
		if err != nil {
			return connectionTestMsg{success: false, err: fmt.Errorf("failed to connect to %s: %w", m.session.PendingAccount.Hostname, err)}
		}
		defer func() { _ = conn.Close() }()

		// Test a simple command to ensure the connection works
		session, err := conn.NewSession()
		if err != nil {
			return connectionTestMsg{success: false, err: fmt.Errorf("failed to create SSH session: %w", err)}
		}
		defer func() { _ = session.Close() }()

		// Run a simple command to verify access
		if err := session.Run("echo 'test'"); err != nil {
			return connectionTestMsg{success: false, err: fmt.Errorf("failed to run test command: %w", err)}
		}

		return connectionTestMsg{success: true, err: nil}
	}
}

// loadAvailableKeys loads the available public keys for selection.
// System keys are automatically deployed and should not be selectable.
func (m *bootstrapModel) loadAvailableKeys() tea.Cmd {
	return func() tea.Msg {
		km := ui.DefaultKeyManager()
		if km == nil {
			return fmt.Errorf("no key manager available")
		}
		allKeys, err := km.GetAllPublicKeys()
		if err != nil {
			return err
		}

		// Get system key to filter it out from selectable keys
		systemKey, err := db.GetActiveSystemKey()
		var systemKeyData string
		if err == nil && systemKey != nil {
			systemKeyData = systemKey.PublicKey
		}

		// Separate keys into user-selectable and global keys
		var userSelectableKeys []model.PublicKey
		var globalKeys []model.PublicKey

		for _, key := range allKeys {
			// Skip if this is a system key (by comparing key data)
			if systemKeyData != "" && strings.Contains(key.KeyData, systemKeyData) {
				continue
			}
			// Skip if this looks like a system key comment
			if strings.Contains(key.Comment, "Keymaster System Key") {
				continue
			}

			// Separate global and non-global keys
			if key.IsGlobal {
				globalKeys = append(globalKeys, key)
			} else {
				userSelectableKeys = append(userSelectableKeys, key)
			}
		}

		// Store separated keys
		m.availableKeys = userSelectableKeys
		m.globalKeys = globalKeys

		return stepCompleteMsg{}
	}
}

// generateNewSession creates a new bootstrap session with fresh keys.
func (m *bootstrapModel) generateNewSession() tea.Cmd {
	return func() tea.Msg {
		// Cleanup old session
		if m.session != nil {
			bootstrap.UnregisterSession(m.session.ID)
			_ = m.session.Delete()
		}

		// Create new session using pending account data
		session, err := bootstrap.NewBootstrapSession(
			m.pendingUsername,
			m.pendingHostname,
			m.pendingLabel,
			m.pendingTags,
		)
		if err != nil {
			return err
		}

		bootstrap.RegisterSession(session)
		if err := session.Save(); err != nil {
			session.Cleanup()
			return err
		}

		return sessionCreatedMsg{session: session}
	}
}

// executeDeployment performs the final atomic deployment.
func (m *bootstrapModel) executeDeployment() tea.Cmd {
	return func() tea.Msg {
		// 1. Create account in database
		accountData := model.Account{
			Username: m.session.PendingAccount.Username,
			Hostname: m.session.PendingAccount.Hostname,
			Label:    m.session.PendingAccount.Label,
			Tags:     m.session.PendingAccount.Tags,
			IsActive: true,
		}

		var accountID int
		var err error
		mgr := ui.DefaultAccountManager()
		if mgr == nil {
			err = fmt.Errorf("no account manager configured")
		} else {
			accountID, err = mgr.AddAccount(accountData.Username, accountData.Hostname,
				accountData.Label, accountData.Tags)
		}
		if err != nil {
			// Log the failure
			_ = logAction("BOOTSTRAP_FAILED", fmt.Sprintf("%s@%s, reason: failed to create account: %v",
				accountData.Username, accountData.Hostname, err))
			return deploymentCompleteMsg{account: accountData, err: fmt.Errorf("failed to create account: %w", err)}
		}

		accountData.ID = accountID

		// 2. Assign selected keys to the account
		selectedKeyIDs := make([]int, 0, len(m.selectedKeys))
		for keyID := range m.selectedKeys {
			selectedKeyIDs = append(selectedKeyIDs, keyID)
		}

		// Add global keys (they will be automatically included by GenerateKeysContent)
		// but we still need to track them for assignment
		for _, key := range m.globalKeys {
			selectedKeyIDs = append(selectedKeyIDs, key.ID)
		}

		// Assign keys to account in database
		km := ui.DefaultKeyManager()
		if km == nil {
			_ = logAction("BOOTSTRAP_FAILED", fmt.Sprintf("%s@%s, reason: no key manager available",
				accountData.Username, accountData.Hostname))
			mgr := ui.DefaultAccountManager()
			if mgr != nil {
				_ = mgr.DeleteAccount(accountID)
			}
			return deploymentCompleteMsg{account: accountData, err: fmt.Errorf("no key manager available")}
		}
		for _, keyID := range selectedKeyIDs {
			if err := km.AssignKeyToAccount(keyID, accountID); err != nil {
				// Log the failure
				_ = logAction("BOOTSTRAP_FAILED", fmt.Sprintf("%s@%s, reason: failed to assign key: %v",
					accountData.Username, accountData.Hostname, err))
				// Cleanup: delete the account if key assignment fails
				mgr := ui.DefaultAccountManager()
				if mgr != nil {
					_ = mgr.DeleteAccount(accountID)
				}
				return deploymentCompleteMsg{account: accountData, err: fmt.Errorf("failed to assign key %d to account: %w", keyID, err)}
			}
		}

		// 3. Generate the authorized_keys content using the existing system
		content, err := deploy.GenerateKeysContent(accountID)
		if err != nil {
			// Log the failure
			_ = logAction("BOOTSTRAP_FAILED", fmt.Sprintf("%s@%s, reason: failed to generate keys content: %v",
				accountData.Username, accountData.Hostname, err))
			// Cleanup: delete the account if content generation fails
			mgr := ui.DefaultAccountManager()
			if mgr != nil {
				_ = mgr.DeleteAccount(accountID)
			}
			return deploymentCompleteMsg{account: accountData, err: fmt.Errorf("failed to generate keys content: %w", err)}
		}

		// 4. Deploy to remote host via SSH using the verified host key
		tempPrivateKey := string(m.session.TempKeyPair.GetPrivateKeyPEM())
		deployer, err := deploy.NewBootstrapDeployerWithExpectedKey(
			accountData.Hostname,
			accountData.Username,
			tempPrivateKey,
			m.hostKey,
		)
		if err != nil {
			// Log the failure
			_ = logAction("BOOTSTRAP_FAILED", fmt.Sprintf("%s@%s, reason: failed to connect: %v",
				accountData.Username, accountData.Hostname, err))
			// Cleanup: delete the account if deployer creation fails
			mgr := ui.DefaultAccountManager()
			if mgr != nil {
				_ = mgr.DeleteAccount(accountID)
			}
			return deploymentCompleteMsg{account: accountData, err: fmt.Errorf("failed to create bootstrap deployer: %w", err)}
		}
		defer deployer.Close()

		if err := deployer.DeployAuthorizedKeys(content); err != nil {
			// Log the failure
			_ = logAction("BOOTSTRAP_FAILED", fmt.Sprintf("%s@%s, reason: deployment failed: %v",
				accountData.Username, accountData.Hostname, err))
			// Cleanup: delete the account if SSH deployment fails
			mgr := ui.DefaultAccountManager()
			if mgr != nil {
				_ = mgr.DeleteAccount(accountID)
			}
			return deploymentCompleteMsg{account: accountData, err: fmt.Errorf("failed to deploy keys to remote host: %w", err)}
		}

		// 5. Update account with current system key serial
		systemKey, err := db.GetActiveSystemKey()
		if err == nil && systemKey != nil {
			if err := db.UpdateAccountSerial(accountID, systemKey.Serial); err != nil {
				fmt.Printf("Warning: failed to update account serial for account %d: %v\n", accountID, err)
			}
		}

		// 6. Log successful bootstrap
		keyCount := len(selectedKeyIDs)
		_ = logAction("BOOTSTRAP_HOST", fmt.Sprintf("account: %s@%s, keys_deployed: %d",
			accountData.Username, accountData.Hostname, keyCount))

		// 7. Cleanup bootstrap session
		bootstrap.UnregisterSession(m.session.ID)
		_ = m.session.Delete()

		return deploymentCompleteMsg{account: accountData, err: nil}
	}
}

// retrieveHostKey fetches the host key from the target server for verification.
func (m *bootstrapModel) retrieveHostKey() tea.Cmd {
	return func() tea.Msg {
		if m.session == nil {
			return hostKeyRetrievedMsg{hostKey: "", err: fmt.Errorf("no session available")}
		}

		// Use the deploy package's GetRemoteHostKey function
		hostKey, err := deploy.GetRemoteHostKey(m.session.PendingAccount.Hostname)
		if err != nil {
			return hostKeyRetrievedMsg{hostKey: "", err: fmt.Errorf("failed to retrieve host key: %w", err)}
		}

		// Convert to authorized_keys format
		hostKeyString := string(ssh.MarshalAuthorizedKey(hostKey))
		return hostKeyRetrievedMsg{hostKey: hostKeyString, err: nil}
	}
}

// helper min removed; inline comparisons preferred

// advanceStep advances to the next step in the workflow.
func (m *bootstrapModel) advanceStep() (tea.Model, tea.Cmd) {
	switch m.step {
	case stepGenerateKey:
		m.step = stepWaitConfirm
	case stepTestConnection:
		if m.connectionTested {
			m.step = stepSelectKeys
			return m, m.loadAvailableKeys()
		}
	}
	return m, nil
}
