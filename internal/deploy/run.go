// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/state"
)

// RunDeploymentForAccount handles the deployment logic for a single account.
// It determines the correct system key to use for connection (either the active
// key for bootstrapping or the account's last known key), generates the
// authorized_keys content, deploys it, and updates the account's key serial
// in the database upon success. The isTUI flag controls which i18n keys are used.
func RunDeploymentForAccount(account model.Account, isTUI bool) error {
	var connectKey *model.SystemKey
	var err error

	if account.Serial == 0 {
		connectKey, err = db.GetActiveSystemKey()
		if err != nil {
			return fmt.Errorf(i18n.T("deploy.error_get_bootstrap_key"), err)
		}
		if connectKey == nil {
			if isTUI {
				return errors.New(i18n.T("deploy.error_no_bootstrap_key_tui"))
			}
			return errors.New(i18n.T("deploy.error_no_bootstrap_key"))
		}
	} else {
		connectKey, err = db.GetSystemKeyBySerial(account.Serial)
		if err != nil {
			return fmt.Errorf(i18n.T("deploy.error_get_serial_key"), account.Serial, err)
		}
		if connectKey == nil {
			if isTUI {
				return fmt.Errorf(i18n.T("deploy.error_no_serial_key_tui"), account.Serial, account.String())
			}
			return fmt.Errorf(i18n.T("deploy.error_no_serial_key"), account.Serial)
		}
	}

	content, err := GenerateKeysContent(account.ID)
	if err != nil {
		return err // This error is already i18n-ready from the generator
	}
	activeKey, err := db.GetActiveSystemKey()
	if err != nil || activeKey == nil {
		return errors.New(i18n.T("deploy.error_get_active_key_for_serial"))
	}

	// Get passphrase from the in-memory cache.
	passphrase := state.PasswordCache.Get()
	// It's critical to wipe the passphrase from memory after we're done with it.
	// We use a defer to ensure this happens even if other parts of the function fail.
	defer func() {
		if passphrase != nil {
			for i := range passphrase {
				passphrase[i] = 0
			}
		}
	}()
	deployer, err := NewDeployer(account.Hostname, account.Username, connectKey.PrivateKey, passphrase)
	if err != nil {
		if isTUI {
			return fmt.Errorf(i18n.T("deploy.error_connection_failed_tui"), account.String(), err)
		}
		return fmt.Errorf(i18n.T("deploy.error_connection_failed"), err) // For CLI
	}
	defer deployer.Close()

	if err := deployer.DeployAuthorizedKeys(content); err != nil {
		return fmt.Errorf(i18n.T("deploy.error_deployment_failed"), err)
	}

	for i := 0; i < 5; i++ { // Retry up to 5 times
		if err = db.UpdateAccountSerial(account.ID, activeKey.Serial); err == nil || !strings.Contains(err.Error(), "database is locked") {
			break
		}
		time.Sleep(time.Duration(50+rand.Intn(100)) * time.Millisecond)
	}
	return err
}
