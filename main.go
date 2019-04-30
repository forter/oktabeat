package main

import (
	"os"

	"github.com/forter/oktabeat/cmd"

	_ "github.com/forter/oktabeat/include"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
