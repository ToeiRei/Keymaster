// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy

import (
	"errors"
	"strings"

	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/sshkey"
)

// AuditAccountStrict performs a strict audit by comparing the full normalized
// remote authorized_keys file with the expected desired state.
func AuditAccountStrict(account model.Account) error {
	// 1. An account with serial 0 has never been deployed. This is a known state, not a drift.
	if account.Serial == 0 {
		return errors.New(i18n.T("audit.error_not_deployed"))
	}

	// 2. Get the system key the database thinks is on the host.
	connectKey, err := db.GetSystemKeyBySerial(account.Serial)
	if err != nil {
		return errors.New(i18n.T("audit.error_get_serial_key", account.Serial, err))
	}
	if connectKey == nil {
		return errors.New(i18n.T("audit.error_no_serial_key", account.Serial))
	}

	// 3. Attempt to connect with that key.
	deployer, err := NewDeployer(account.Hostname, account.Username, connectKey.PrivateKey, "")
	if err != nil {
		return errors.New(i18n.T("audit.error_connection_failed", account.Serial, err))
	}
	defer deployer.Close()

	// 4. Read the content of the remote authorized_keys file.
	remoteContentBytes, err := deployer.GetAuthorizedKeys()
	if err != nil {
		return errors.New(i18n.T("audit.error_read_remote_file", err))
	}

	// 5. Generate the expected content for this account.
	expectedContent, err := GenerateKeysContent(account.ID)
	if err != nil {
		return errors.New(i18n.T("audit.error_generate_expected", err))
	}

	// 6. Normalize both for canonical comparison.
	normalize := func(s string) string {
		s = strings.ReplaceAll(s, "\r\n", "\n")
		s = strings.TrimSpace(s)
		return s
	}
	if normalize(string(remoteContentBytes)) != normalize(expectedContent) {
		return errors.New(i18n.T("audit.error_drift_detected"))
	}
	return nil
}

// AuditAccountSerial performs a lightweight audit by checking only the
// Keymaster header serial number on the remote host against the account's last
// deployed serial recorded in the database.
func AuditAccountSerial(account model.Account) error {
	if account.Serial == 0 {
		return errors.New(i18n.T("audit.error_not_deployed"))
	}

	connectKey, err := db.GetSystemKeyBySerial(account.Serial)
	if err != nil {
		return errors.New(i18n.T("audit.error_get_serial_key", account.Serial, err))
	}
	if connectKey == nil {
		return errors.New(i18n.T("audit.error_no_serial_key", account.Serial))
	}

	deployer, err := NewDeployer(account.Hostname, account.Username, connectKey.PrivateKey, "")
	if err != nil {
		return errors.New(i18n.T("audit.error_connection_failed", account.Serial, err))
	}
	defer deployer.Close()

	remoteContentBytes, err := deployer.GetAuthorizedKeys()
	if err != nil {
		return errors.New(i18n.T("audit.error_read_remote_file", err))
	}

	// Parse first non-empty header line and extract serial.
	lines := strings.Split(strings.ReplaceAll(string(remoteContentBytes), "\r\n", "\n"), "\n")
	if len(lines) == 0 {
		return errors.New(i18n.T("audit.error_drift_detected"))
	}
	var header string
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		header = ln
		break
	}
	if header == "" {
		return errors.New(i18n.T("audit.error_drift_detected"))
	}
	serial, err := sshkey.ParseSerial(header)
	if err != nil || serial != account.Serial {
		return errors.New(i18n.T("audit.error_drift_detected"))
	}
	return nil
}
