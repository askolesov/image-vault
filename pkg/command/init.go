package command

import (
	"fmt"
	"os"
	"strings"

	v2 "github.com/askolesov/image-vault/pkg/v2"
	"github.com/spf13/cobra"
)

const (
	DefaultConfigFile = AppName + ".yaml"
)

func GetInitCmd() *cobra.Command {
	res := &cobra.Command{
		Use:   "init",
		Short: "initialize the library (create config file)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if config file already exists
			if _, err := os.Stat(DefaultConfigFile); err == nil {
				return fmt.Errorf("config file %s already exists", DefaultConfigFile)
			}

			// Check if current directory is not empty
			entries, err := os.ReadDir(".")
			if err != nil {
				return err
			}
			if len(entries) > 0 {
				cmd.Println("Warning: The current directory is not empty.")
				cmd.Print("Do you want to continue? (y/N): ")
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) != "y" {
					return fmt.Errorf("initialization cancelled")
				}
			}

			err = v2.WriteDefaultConfigToFile(DefaultConfigFile)
			if err != nil {
				return err
			}

			cmd.Printf("Created %s\n", DefaultConfigFile)

			return nil
		},
	}

	return res
}
