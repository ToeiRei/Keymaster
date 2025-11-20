//go:build tools_probe
// +build tools_probe

package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"

    "github.com/uptrace/bun"
    // _ "modernc.org/sqlite"
)

type AccountModel struct {
    bun.BaseModel `bun:"table:accounts"`
    ID       int    `bun:"id,pk,autoincrement"`
    Username string `bun:"username"`
    Hostname string `bun:"hostname"`
}

func main() {
    dsn := "file:probe?mode=memory&cache=shared"
    db, err := sql.Open("sqlite", dsn)
    if err != nil {
        log.Fatalf("open: %v", err)
    }
    if _, err := db.Exec(`CREATE TABLE accounts (
        id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        username TEXT NOT NULL,
        hostname TEXT NOT NULL
    )`); err != nil {
        log.Fatalf("create table: %v", err)
    }
    bdb := bun.NewDB(db, sqlitedialect.New())
    ctx := context.Background()
    am := &AccountModel{Username: "u1", Hostname: "h1"}
    if _, err := bdb.NewInsert().Model(am).Exec(ctx); err != nil {
        log.Fatalf("insert err: %v", err)
    }
    fmt.Printf("Inserted ID: %d\n", am.ID)

    var res []AccountModel
    if err := bdb.NewSelect().Model(&res).Scan(ctx); err != nil {
        log.Fatalf("select err: %v", err)
    }
    fmt.Printf("Rows: %d\n", len(res))
    for _, r := range res {
        fmt.Printf("row: %+v\n", r)
    }
}
package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"

    "github.com/uptrace/bun"
    "github.com/uptrace/bun/dialect/sqlitedialect"
    _ "modernc.org/sqlite"
    // _ "modernc.org/sqlite"

type AccountModel struct {
    bun.BaseModel ` + "`bun:\"table:accounts\"`" + `
    ID       int    ` + "`bun:\"id,pk,autoincrement\"`" + `
    Username string ` + "`bun:\"username\"`" + `
    Hostname string ` + "`bun:\"hostname\"`" + `
}

func main() {
    dsn := "file:probe?mode=memory&cache=shared"
    db, err := sql.Open("sqlite", dsn)
    if err != nil {
        log.Fatalf("open: %v", err)
    }
    if _, err := db.Exec(`CREATE TABLE accounts (
        id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        username TEXT NOT NULL,
        hostname TEXT NOT NULL
    )`); err != nil {
        log.Fatalf("create table: %v", err)
    }
    bdb := bun.NewDB(db, sqlitedialect.New())
    ctx := context.Background()
    am := &AccountModel{Username: "u1", Hostname: "h1"}
    if _, err := bdb.NewInsert().Model(am).Exec(ctx); err != nil {
        log.Fatalf("insert err: %v", err)
    }
    fmt.Printf("Inserted ID: %d\n", am.ID)

    var res []AccountModel
    if err := bdb.NewSelect().Model(&res).Scan(ctx); err != nil {
        log.Fatalf("select err: %v", err)
    }
    fmt.Printf("Rows: %d\n", len(res))
    for _, r := range res {
        fmt.Printf("row: %+v\n", r)
    }
}
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"
)
    // _ "modernc.org/sqlite"
type AccountModel struct {
	bun.BaseModel ` + "`bun:\"table:accounts\"`" + `
	ID       int    ` + "`bun:\"id,pk,autoincrement\"`" + `
	Username string ` + "`bun:\"username\"`" + `
	Hostname string ` + "`bun:\"hostname\"`" + `
}

func main() {
	dsn := "file:probe?mode=memory&cache=shared"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		log.Fatalf("open: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE accounts (
		id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL,
		hostname TEXT NOT NULL
	)`); err != nil {
		log.Fatalf("create table: %v", err)
	}
	bdb := bun.NewDB(db, sqlitedialect.New())
	ctx := context.Background()
	am := &AccountModel{Username: "u1", Hostname: "h1"}
	if _, err := bdb.NewInsert().Model(am).Exec(ctx); err != nil {
		log.Fatalf("insert err: %v", err)
	}
	fmt.Printf("Inserted ID: %d\n", am.ID)

	var res []AccountModel
	if err := bdb.NewSelect().Model(&res).Scan(ctx); err != nil {
		log.Fatalf("select err: %v", err)
	}
	fmt.Printf("Rows: %d\n", len(res))
	for _, r := range res {
		fmt.Printf("row: %+v\n", r)
	}
}
