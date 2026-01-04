package main

import (
	"fmt"

	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/i18n"
)

func main() {
	dsn := "file:debprobe?mode=memory&cache=shared"
	i18n.Init("en")
	if err := db.InitDB("sqlite", dsn); err != nil {
		panic(err)
	}

	mgr := db.DefaultAccountManager()
	if mgr == nil {
		// Fall back to legacy helpers if no manager is configured.
		_, err := db.AddAccount("user1", "host1.com", "prod-web-1", "")
		if err != nil {
			panic(err)
		}
		_, err = db.AddAccount("user2", "host2.com", "", "")
		if err != nil {
			panic(err)
		}
		id, err := db.AddAccount("user3", "host3.com", "inactive-host", "")
		if err != nil {
			panic(err)
		}
		_ = db.ToggleAccountStatus(id)
	} else {
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
	}

	accs, err := db.GetAllActiveAccounts()
	if err != nil {
		panic(err)
	}
	fmt.Printf("active accounts: %d\n", len(accs))
	for _, a := range accs {
		fmt.Printf("account: %+v\n", a)
	}

	all, err := db.GetAllAccounts()
	if err != nil {
		panic(err)
	}
	fmt.Printf("all accounts: %d\n", len(all))
	for _, a := range all {
		fmt.Printf("all account: %+v\n", a)
	}

	// Direct SQL probe removed — use package-level helpers above.
}
