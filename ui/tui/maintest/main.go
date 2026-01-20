package main

import (
	"fmt"
	"os"

	tui "github.com/toeirei/keymaster/ui/tui"
)

func main() {
	if err := tui.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
