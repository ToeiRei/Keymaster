// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package main

import (
	log "github.com/charmbracelet/log"

	"github.com/toeirei/keymaster/internal/core/db"
	"github.com/toeirei/keymaster/internal/i18n"
)

func main() {
	dsn := "file:debprobe?mode=memory&cache=shared"
	i18n.Init("en")
	if _, err := db.New("sqlite", dsn); err != nil {
		panic(err)
	}

	mgr := db.DefaultAccountManager()
	if mgr == nil {
		panic("no account manager available")
	}
	_, err := mgr.AddAccount("user1", "host1.com", "prod-web-1", "")
	if err != nil {
		panic(err)
	}
	_, err = mgr.AddAccount("user2", "host2.com", "", "")
	if err != nil {
		panic(err)
	}
	id, err := mgr.AddAccount("user3", "host3.com", "inactive-host", "")
	if err != nil {
		panic(err)
	}
	_ = db.ToggleAccountStatus(id)

	accs, err := db.GetAllActiveAccounts()
	if err != nil {
		panic(err)
	}
	log.Infof("active accounts: %d", len(accs))
	for _, a := range accs {
		log.Infof("account: %+v", a)
	}

	all, err := db.GetAllAccounts()
	if err != nil {
		panic(err)
	}
	log.Infof("all accounts: %d", len(all))
	for _, a := range all {
		log.Infof("all account: %+v", a)
	}

	// Direct SQL probe removed — use package-level helpers above.
}
