package main

import (
	"fmt"
	"os"

	"github.com/rhysj6/devops-tools/cmd"
)

var version = "dev"

func main() {
	rootCmd := cmd.GetCommand(version)
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
