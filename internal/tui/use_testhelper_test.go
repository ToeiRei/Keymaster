package tui

import "testing"

// Ensure the package test helper is compiled and used so linters don't mark it unused.
func Test_UseInitTestDBT(t *testing.T) {
	initTestDBT(t)
}
