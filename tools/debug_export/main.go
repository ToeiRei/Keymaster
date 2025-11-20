package main

import (
	"fmt"

	"database/sql"

	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/i18n"
)

func main() {
	dsn := "file:debprobe?mode=memory&cache=shared"
	i18n.Init("en")
	if err := db.InitDB("sqlite", dsn); err != nil {
		panic(err)
	}

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

	// Direct SQL probe
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		panic(err)
	}
	rows, err := sqlDB.Query("SELECT id, username, hostname, is_active FROM accounts ORDER BY id")
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	fmt.Println("Direct SQL rows:")
	for rows.Next() {
		var id int
		var user, host string
		var isActive int
		if err := rows.Scan(&id, &user, &host, &isActive); err != nil {
			panic(err)
		}
		fmt.Printf("id=%d user=%s host=%s is_active=%d\n", id, user, host, isActive)
	}
}
