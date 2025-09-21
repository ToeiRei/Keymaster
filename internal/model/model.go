package model

import "fmt"

// Account represents a user on a specific host (e.g., deploy@server-01).
// This is the core entity for which we manage access.
type Account struct {
	ID       int
	Username string
	Hostname string
	Serial   int
}

// String returns the user@host representation.
func (a Account) String() string {
	return fmt.Sprintf("%s@%s", a.Username, a.Hostname)
}
