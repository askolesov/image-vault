package main

import (
	"fmt"
	"os"

	"github.com/askolesov/image-vault/pkg/command"
)

func main() {
	if err := command.GetRootCommand().Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
