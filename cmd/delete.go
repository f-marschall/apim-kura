package cmd

import (
	"context"
	"fmt"

	"github.com/f-marschall/apim-kura/internal/azure"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete subscription keys from Azure API Management",
	Long: `Delete removes subscription keys from an Azure API Management instance.

By default, built-in subscriptions (e.g. the master key) are preserved.
Use --all to include built-in subscriptions in the deletion.

Example:
  kura delete --resource-group mygroup --apim-name myapim
  kura delete -g mygroup -a myapim --product-id myproduct
  kura delete -g mygroup -a myapim --dry-run
  kura delete -g mygroup -a myapim --all`,
	RunE: runDelete,
}

var (
	deleteResourceGroup string
	deleteAPIMName      string
	deleteSubscription  string
	deleteProductID     string
	deleteDryRun        bool
	deleteAll           bool
)

func init() {
	rootCmd.AddCommand(deleteCmd)

	deleteCmd.Flags().StringVarP(&deleteResourceGroup, "resource-group", "g", "", "Azure resource group name (required)")
	deleteCmd.Flags().StringVarP(&deleteAPIMName, "apim-name", "a", "", "Azure API Management instance name (required)")
	deleteCmd.Flags().StringVarP(&deleteSubscription, "subscription", "s", "", "Azure subscription ID")
	deleteCmd.Flags().StringVarP(&deleteProductID, "product-id", "p", "", "Only delete subscriptions scoped to this product")
	deleteCmd.Flags().BoolVar(&deleteDryRun, "dry-run", false, "Preview deletions without applying them")
	deleteCmd.Flags().BoolVar(&deleteAll, "all", false, "Delete all subscriptions including built-in ones")

	deleteCmd.MarkFlagRequired("resource-group")
	deleteCmd.MarkFlagRequired("apim-name")
}

func runDelete(cmd *cobra.Command, args []string) error {
	fmt.Printf("Deleting subscription keys from APIM instance: %s\n", deleteAPIMName)
	fmt.Printf("Resource Group: %s\n", deleteResourceGroup)

	if deleteSubscription != "" {
		fmt.Printf("Subscription ID: %s\n", deleteSubscription)
	}

	if deleteProductID != "" {
		fmt.Printf("Product ID: %s\n", deleteProductID)
	}

	if deleteAll {
		fmt.Println("Mode: Delete ALL subscriptions (including built-in)")
	} else {
		fmt.Println("Mode: Delete all subscriptions except built-in (master)")
	}

	if deleteDryRun {
		fmt.Println("\nRunning in DRY-RUN mode. No changes will be applied.")
	}

	ctx := context.Background()
	fmt.Println("\nAuthenticating with Azure CLI...")

	client, err := azure.NewClient(ctx, deleteSubscription, deleteResourceGroup, deleteAPIMName)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	fmt.Println("Successfully authenticated with Azure CLI")

	fmt.Println("\nFetching subscriptions...")
	subs, err := client.ListSubscriptions(ctx, deleteProductID)
	if err != nil {
		return fmt.Errorf("failed to list subscriptions: %w", err)
	}

	if len(subs) == 0 {
		fmt.Println("No subscriptions found. Nothing to delete.")
		return nil
	}
	fmt.Printf("\nFound %d subscription(s)\n", len(subs))

	var deleted, skipped, failed int
	for _, sub := range subs {
		sid := sub.Name
		displayName := sub.Properties.DisplayName

		if !deleteAll && sid == "master" {
			fmt.Printf("  [SKIP] %s (built-in)\n", displayName)
			skipped++
			continue
		}

		if deleteDryRun {
			fmt.Printf("  [DRY-RUN] Would delete: %s (id=%s)\n", displayName, sid)
			deleted++
			continue
		}

		fmt.Printf("  Deleting: %s (id=%s)...\n", displayName, sid)
		if err := client.DeleteSubscription(ctx, sid); err != nil {
			fmt.Printf("  [FAIL] %s: %v\n", displayName, err)
			failed++
			continue
		}
		fmt.Printf("  [OK]   %s\n", displayName)
		deleted++
	}

	fmt.Printf("\nDelete complete: %d deleted, %d skipped, %d failed\n", deleted, skipped, failed)
	if failed > 0 {
		return fmt.Errorf("%d subscription(s) failed to delete", failed)
	}
	return nil
}
