package cmd

import (
	"context"
	"fmt"

	"github.com/f-marschall/apim-kura/internal/azure"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List subscription keys from Azure API Management",
	Long: `List retrieves and displays all subscription keys from an Azure API Management
instance directly in the terminal.

Example:
  kura list --resource-group mygroup --apim-name myapim
  kura list --resource-group mygroup --apim-name myapim --subscription mysubid
  kura list --resource-group mygroup --apim-name myapim --product-id myproduct`,
	RunE: runList,
}

var (
	listResourceGroup string
	listAPIMName      string
	listSubscription  string
	listProductID     string
)

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVarP(&listResourceGroup, "resource-group", "g", "", "Azure resource group name (required)")
	listCmd.Flags().StringVarP(&listAPIMName, "apim-name", "a", "", "Azure API Management instance name (required)")
	listCmd.Flags().StringVarP(&listSubscription, "subscription", "s", "", "Azure subscription ID")
	listCmd.Flags().StringVarP(&listProductID, "product-id", "p", "", "Filter by product ID")

	listCmd.MarkFlagRequired("resource-group")
	listCmd.MarkFlagRequired("apim-name")
}

func runList(cmd *cobra.Command, args []string) error {
	fmt.Printf("Listing subscription keys from APIM instance: %s\n", listAPIMName)
	fmt.Printf("Resource Group: %s\n", listResourceGroup)

	if listSubscription != "" {
		fmt.Printf("Subscription ID: %s\n", listSubscription)
	}
	if listProductID != "" {
		fmt.Printf("Product ID: %s\n", listProductID)
	}

	ctx := context.Background()
	fmt.Println("\nAuthenticating with Azure CLI...")

	client, err := azure.NewClient(ctx, listSubscription, listResourceGroup, listAPIMName)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	fmt.Println("Successfully authenticated with Azure CLI")

	fmt.Println("\nFetching subscriptions...")
	subs, err := client.ListSubscriptions(ctx, listProductID)
	if err != nil {
		return fmt.Errorf("failed to list subscriptions: %w", err)
	}

	if len(subs) == 0 {
		fmt.Println("No subscriptions found.")
		return nil
	}

	fmt.Printf("\nFound %d subscription(s):\n", len(subs))
	fmt.Println("────────────────────────────────────────────────────────────────")

	for i, sub := range subs {
		fmt.Printf("\n[%d] %s\n", i+1, sub.Properties.DisplayName)
		fmt.Printf("    ID:               %s\n", sub.ID)
		fmt.Printf("    Name:             %s\n", sub.Name)
		fmt.Printf("    Type:             %s\n", sub.Type)
		fmt.Printf("    Scope:            %s\n", sub.Properties.Scope)
		fmt.Printf("    State:            %s\n", sub.Properties.State)
		fmt.Printf("    Owner ID:         %s\n", sub.Properties.OwnerID)
		fmt.Printf("    Created:          %s\n", sub.Properties.CreatedDate)
		fmt.Printf("    Start Date:       %s\n", sub.Properties.StartDate)
		fmt.Printf("    End Date:         %s\n", sub.Properties.EndDate)
		fmt.Printf("    Expiration Date:  %s\n", sub.Properties.ExpirationDate)
		fmt.Printf("    Notification Date:%s\n", sub.Properties.NotificationDate)
		fmt.Printf("    State Comment:    %s\n", sub.Properties.StateComment)
		fmt.Printf("    Allow Tracing:    %t\n", sub.Properties.AllowTracing)
		fmt.Printf("    Primary Key:      %s\n", sub.Properties.PrimaryKey)
		fmt.Printf("    Secondary Key:    %s\n", sub.Properties.SecondaryKey)
	}

	fmt.Println("\n────────────────────────────────────────────────────────────────")
	return nil
}
