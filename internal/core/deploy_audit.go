// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	"github.com/toeirei/keymaster/internal/core/model"
	"github.com/toeirei/keymaster/internal/core/sshkey"
	"github.com/toeirei/keymaster/internal/core/state"
	"github.com/toeirei/keymaster/internal/i18n"
)

// AuditAccountStrict performs a strict audit by comparing the full normalized
// remote authorized_keys file with the expected desired state.
func AuditAccountStrict(account model.Account) error {
	if account.Serial == 0 {
		return errors.New(i18n.T("audit.error_not_deployed"))
	}

	kr := DefaultKeyReader()
	if kr == nil {
		return errors.New(i18n.T("audit.error_no_serial_key", account.Serial))
	}
	connectKey, err := kr.GetSystemKeyBySerial(account.Serial)
	if err != nil {
		return errors.New(i18n.T("audit.error_get_serial_key", account.Serial, err))
	}
	if connectKey == nil {
		return errors.New(i18n.T("audit.error_no_serial_key", account.Serial))
	}

	passphrase := state.PasswordCache.Get()
	defer func() {
		for i := range passphrase {
			passphrase[i] = 0
		}
	}()

	deployer, err := NewDeployerFactory(account.Hostname, account.Username, SystemKeyToSecret(connectKey), passphrase)
	if err != nil {
		return fmt.Errorf(i18n.T("audit.error_connection_failed"), account.Serial, err)
	}
	defer deployer.Close()
	state.PasswordCache.Clear()

	remoteContentBytes, err := deployer.GetAuthorizedKeys()
	if err != nil {
		return errors.New(i18n.T("audit.error_read_remote_file", err))
	}

	expectedContent, err := GenerateKeysContent(account.ID)
	if err != nil {
		return errors.New(i18n.T("audit.error_generate_expected", err))
	}

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

	kr := DefaultKeyReader()
	if kr == nil {
		return errors.New(i18n.T("audit.error_no_serial_key", account.Serial))
	}
	connectKey, err := kr.GetSystemKeyBySerial(account.Serial)
	if err != nil {
		return errors.New(i18n.T("audit.error_get_serial_key", account.Serial, err))
	}
	if connectKey == nil {
		return errors.New(i18n.T("audit.error_no_serial_key", account.Serial))
	}

	passphrase := state.PasswordCache.Get()
	defer func() {
		for i := range passphrase {
			passphrase[i] = 0
		}
	}()

	deployer, err := NewDeployerFactory(account.Hostname, account.Username, SystemKeyToSecret(connectKey), passphrase)
	if err != nil {
		return fmt.Errorf(i18n.T("audit.error_connection_failed"), account.Serial, err)
	}
	defer deployer.Close()
	state.PasswordCache.Clear()

	remoteContentBytes, err := deployer.GetAuthorizedKeys()
	if err != nil {
		return errors.New(i18n.T("audit.error_read_remote_file", err))
	}

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

// HashAuthorizedKeysContent normalizes raw authorized_keys content and returns
// a SHA256 hex fingerprint. Normalization mirrors what we use when
// constructing authorized_keys to make comparisons robust across platforms.
func HashAuthorizedKeysContent(raw []byte) string {
	s := string(raw)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	norm := strings.Join(lines, "\n")
	sum := sha256.Sum256([]byte(norm))
	return fmt.Sprintf("%x", sum[:])
}
