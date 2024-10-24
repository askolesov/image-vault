package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/askolesov/image-vault/pkg/vault"
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
			return initLibrary(cmd)
		},
	}

	return res
}

func ensureLibraryInitialized(cmd *cobra.Command) error {
	cfgExists, err := util.IsConfigExists(DefaultConfigFile)
	if err != nil {
		return err
	}
	if !cfgExists {
		cmd.Println("The current directory is not initialized as an image-vault library.")
		cmd.Print("Do you want to initialize it now? (y/N): ")
		var response string
		_, err := fmt.Scanln(&response)
		if err != nil {
			return err
		}
		if strings.ToLower(response) == "y" {
			err := initLibrary(cmd)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("cannot continue without initialization")
		}
	}
	return nil
}

func initLibrary(cmd *cobra.Command) error {
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
		_, err := fmt.Scanln(&response)
		if err != nil {
			return err
		}
		if strings.ToLower(response) != "y" {
			return fmt.Errorf("initialization cancelled")
		}
	}

	// Write default config to file
	err = util.WriteDefaultConfigToFile(DefaultConfigFile)
	if err != nil {
		return err
	}

	cmd.Printf("Created %s\n", DefaultConfigFile)

	return nil
}
