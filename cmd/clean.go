package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Delete the backup folder and all its contents",
	Long: `Clean removes the local backup directory and all subfolders created by
the backup command.

Example:
  kura clean`,
	RunE: runClean,
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}

func runClean(cmd *cobra.Command, args []string) error {
	dir := "backup"

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Println("No backup folder found. Nothing to clean.")
		return nil
	}

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("failed to remove backup folder: %w", err)
	}

	fmt.Println("Backup folder removed successfully.")
	return nil
}
