package main

import (
	"fmt"
	"github.com/askolesov/image-vault/pkg/command"
	"os"
)

func main() {
	if err := command.GetRootCommand().Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
