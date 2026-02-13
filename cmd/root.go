package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	Version = "dev"
)

var rootCmd = &cobra.Command{
	Use:   "kura",
	Short: "Kura - Azure API Management subscription key backup and restore tool",
	Long: `Kura is a CLI tool for backing up and restoring subscription keys
from Azure API Management instances.

It provides simple commands to export subscription keys to a file
and restore them from a backup file.`,
	Version: Version,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kura.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
