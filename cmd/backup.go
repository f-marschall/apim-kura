package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/f-marschall/apim-kura/internal/azure"
	"github.com/f-marschall/apim-kura/internal/backup"
	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup subscription keys from Azure API Management",
	Long: `Backup retrieves subscription keys from an Azure API Management instance
and saves them to a local backup directory or file.

By default, backups are stored under: backup/<resource-group>/<apim-name>[/<product-id>]
Use --output to save to a custom file path instead.

Example:
  kura backup --resource-group mygroup --apim-name myapim
  kura backup --resource-group mygroup --apim-name myapim --product-id myproduct
  kura backup -g mygroup -a myapim --output ./my-backup.json`,
	RunE: runBackup,
}

var (
	backupResourceGroup string
	backupAPIMName      string
	backupSubscription  string
	backupProductID     string
	backupOutput        string
)

func init() {
	rootCmd.AddCommand(backupCmd)

	// Local flags for the backup command
	backupCmd.Flags().StringVarP(&backupResourceGroup, "resource-group", "g", "", "Azure resource group name (required)")
	backupCmd.Flags().StringVarP(&backupAPIMName, "apim-name", "a", "", "Azure API Management instance name (required)")
	backupCmd.Flags().StringVarP(&backupSubscription, "subscription", "s", "", "Azure subscription ID")
	backupCmd.Flags().StringVarP(&backupProductID, "product-id", "p", "", "Azure APIM product ID (optional, scopes backup to a product)")
	backupCmd.Flags().StringVarP(&backupOutput, "output", "o", "", "Output file path (if not specified, defaults to backup folder structure)")

	// Mark required flags
	backupCmd.MarkFlagRequired("resource-group")
	backupCmd.MarkFlagRequired("apim-name")
}

func runBackup(cmd *cobra.Command, args []string) error {
	fmt.Printf("Backing up subscription keys from APIM instance: %s\n", backupAPIMName)
	fmt.Printf("Resource Group: %s\n", backupResourceGroup)

	if backupSubscription != "" {
		fmt.Printf("Subscription ID: %s\n", backupSubscription)
	}
	if backupProductID != "" {
		fmt.Printf("Product ID: %s\n", backupProductID)
	}

	// Determine output file path
	var filePath string
	if backupOutput != "" {
		filePath = backupOutput
		fmt.Printf("Output file: %s\n", filePath)
	} else {
		// Create backup directory structure
		backupDir, err := backup.EnsureBackupDir(backupResourceGroup, backupAPIMName, backupProductID)
		if err != nil {
			return fmt.Errorf("failed to create backup directory: %w", err)
		}
		filePath = filepath.Join(backupDir, "subscriptions.json")
		fmt.Printf("Backup directory: %s\n", backupDir)
	}

	// Authenticate with Azure CLI
	ctx := context.Background()
	fmt.Println("\nAuthenticating with Azure CLI...")

	client, err := azure.NewClient(ctx, backupSubscription, backupResourceGroup, backupAPIMName)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	fmt.Println("\nFetching subscriptions...")
	subs, err := client.ListSubscriptions(ctx, backupProductID)

	fmt.Printf("\nFound %d subscription(s)\n", len(subs))

	prettyJSON, err := json.MarshalIndent(subs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal subscriptions to JSON: %w", err)
	}

	// Ensure parent directories exist if using custom output path
	if backupOutput != "" {
		dir := filepath.Dir(filePath)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
		}
	}

	if err := os.WriteFile(filePath, prettyJSON, 0644); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}
	fmt.Printf("Backup saved to: %s\n", filePath)

	fmt.Println("Backup completed successfully")
	return nil
}
