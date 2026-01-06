// This source code is licensed under the MIT license found in the LICENSE file.

package core

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
		return err
	}
	activeKey, err := db.GetActiveSystemKey()
	if err != nil || activeKey == nil {
		return errors.New(i18n.T("deploy.error_get_active_key_for_serial"))
	}

	passphrase := state.PasswordCache.Get()
	defer func() {
		for i := range passphrase {
			passphrase[i] = 0
		}
	}()
	deployer, err := NewDeployerFactory(account.Hostname, account.Username, connectKey.PrivateKey, passphrase)
	if err != nil {
		if isTUI {
			return fmt.Errorf(i18n.T("deploy.error_connection_failed_tui"), account.String(), err)
		}
		return fmt.Errorf(i18n.T("deploy.error_connection_failed"), err)
	}
	defer deployer.Close()
	state.PasswordCache.Clear()

	if err := deployer.DeployAuthorizedKeys(content); err != nil {
		return fmt.Errorf(i18n.T("deploy.error_deployment_failed"), err)
	}

	for i := 0; i < 5; i++ {
		if err = db.UpdateAccountSerial(account.ID, activeKey.Serial); err == nil || !strings.Contains(err.Error(), "database is locked") {
			break
		}
		time.Sleep(time.Duration(50+rand.Intn(100)) * time.Millisecond)
	}
	return err
}
