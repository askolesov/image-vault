package main

import (
	"fmt"
	"os"

	"github.com/askolesov/image-vault/internal/command"
)

func main() {
	if err := command.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
