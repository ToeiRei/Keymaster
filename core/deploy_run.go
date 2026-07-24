// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package core

import (
	"math/rand"
	"strings"
	"time"

	"github.com/toeirei/keymaster/core/model"
	"github.com/toeirei/keymaster/core/state"
	"github.com/toeirei/keymaster/ui/i18n"
)

// RunDeploymentForAccount handles the deployment logic for a single account.
func RunDeploymentForAccount(account model.Account, isTUI bool) error {
	var connectKey *model.SystemKey
	var err error

	kr := DefaultKeyReader()
	if kr == nil {
		return i18n.NewError("deploy.error_no_bootstrap_key")
	}
	if account.Serial == 0 {
		connectKey, err = kr.GetActiveSystemKey()
		if err != nil {
			return i18n.WrapError(err, "deploy.error_get_bootstrap_key")
		}
		if connectKey == nil {
			if isTUI {
				return i18n.NewError("deploy.error_no_bootstrap_key_tui")
			}
			return i18n.NewError("deploy.error_no_bootstrap_key")
		}
	} else {
		connectKey, err = kr.GetSystemKeyBySerial(account.Serial)
		if err != nil {
			return i18n.WrapError(err, "deploy.error_get_serial_key", account.Serial)
		}
		if connectKey == nil {
			if isTUI {
				return i18n.NewError("deploy.error_no_serial_key_tui", account.Serial, account.String())
			}
			return i18n.NewError("deploy.error_no_serial_key", account.Serial)
		}
	}

	content, err := GenerateKeysContent(account.ID)
	if err != nil {
		return err
	}
	activeKey, err := kr.GetActiveSystemKey()
	if err != nil || activeKey == nil {
		return i18n.NewError("deploy.error_get_active_key_for_serial")
	}

	passphrase := state.PasswordCache.Get()
	defer func() {
		for i := range passphrase {
			passphrase[i] = 0
		}
	}()
	deployer, err := NewDeployerFactory(account.Hostname, account.Username, SystemKeyToSecret(connectKey), passphrase)
	if err != nil {
		if isTUI {
			return i18n.WrapError(err, "deploy.error_connection_failed_tui", account.String())
		}
		return i18n.WrapError(err, "deploy.error_connection_failed")
	}
	defer deployer.Close()
	state.PasswordCache.Clear()

	if err := deployer.DeployAuthorizedKeys(content); err != nil {
		return i18n.WrapError(err, "deploy.error_deployment_failed")
	}

	updater := DefaultAccountSerialUpdater()
	if updater == nil {
		return i18n.NewError("deploy.error_get_active_key_for_serial")
	}
	for i := 0; i < 5; i++ {
		if err = updater.UpdateAccountSerial(account.ID, activeKey.Serial); err == nil || !strings.Contains(err.Error(), "database is locked") {
			break
		}
		time.Sleep(time.Duration(50+rand.Intn(100)) * time.Millisecond)
	}
	return err
}
