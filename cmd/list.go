package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/f-marschall/apim-kura/internal/azure"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List subscription keys from Azure API Management",
	Long: `List retrieves and displays subscriptions from an Azure API Management
instance directly in the terminal. Keys are hidden unless --show-keys is specified.

Example:
  kura list --resource-group mygroup --apim-name myapim
  kura list --resource-group mygroup --apim-name myapim --subscription mysubid
  kura list --resource-group mygroup --apim-name myapim --product-id myproduct
  kura list --resource-group mygroup --apim-name myapim --show-keys`,
	RunE: runList,
}

var (
	listResourceGroup string
	listAPIMName      string
	listSubscription  string
	listProductID     string
	listShowKeys      bool
)

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVarP(&listResourceGroup, "resource-group", "g", "", "Azure resource group name (required)")
	listCmd.Flags().StringVarP(&listAPIMName, "apim-name", "a", "", "Azure API Management instance name (required)")
	listCmd.Flags().StringVarP(&listSubscription, "subscription", "s", "", "Azure subscription ID")
	listCmd.Flags().StringVarP(&listProductID, "product-id", "p", "", "Filter by product ID")
	listCmd.Flags().BoolVar(&listShowKeys, "show-keys", false, "Display primary and secondary subscription keys")

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

	type ownerLookup struct {
		user azure.UserInfo
		err  error
	}
	owners := make(map[string]ownerLookup)

	for i, sub := range subs {
		ownerName := "not assigned"
		if sub.Properties.OwnerID != "" {
			lookup, ok := owners[sub.Properties.OwnerID]
			if !ok {
				lookup.user, lookup.err = client.GetOwner(ctx, sub.Properties.OwnerID)
				owners[sub.Properties.OwnerID] = lookup
				if lookup.err != nil {
					fmt.Fprintf(os.Stderr, "Warning: unable to resolve owner %q: %v\n", sub.Properties.OwnerID, lookup.err)
				}
			}
			if lookup.err != nil {
				ownerName = "unresolved"
			} else {
				ownerName = lookup.user.DisplayName()
				if ownerName == "" {
					ownerName = "unavailable"
				}
			}
		}
		printSubscription(os.Stdout, i+1, sub, ownerName, listShowKeys)
	}

	fmt.Println("\n────────────────────────────────────────────────────────────────")
	return nil
}

func printSubscription(w io.Writer, index int, sub azure.SubscriptionInfo, ownerName string, showKeys bool) {
	fmt.Fprintf(w, "\n[%d] %s\n", index, sub.Properties.DisplayName)
	fmt.Fprintf(w, "    ID:               %s\n", sub.ID)
	fmt.Fprintf(w, "    Name:             %s\n", sub.Name)
	fmt.Fprintf(w, "    Type:             %s\n", sub.Type)
	fmt.Fprintf(w, "    Scope:            %s\n", sub.Properties.Scope)
	fmt.Fprintf(w, "    State:            %s\n", sub.Properties.State)
	fmt.Fprintf(w, "    Owner ID:         %s\n", sub.Properties.OwnerID)
	fmt.Fprintf(w, "    Owner Display Name: %s\n", ownerName)
	fmt.Fprintf(w, "    Created:          %s\n", sub.Properties.CreatedDate)
	fmt.Fprintf(w, "    Start Date:       %s\n", sub.Properties.StartDate)
	fmt.Fprintf(w, "    End Date:         %s\n", sub.Properties.EndDate)
	fmt.Fprintf(w, "    Expiration Date:  %s\n", sub.Properties.ExpirationDate)
	fmt.Fprintf(w, "    Notification Date:%s\n", sub.Properties.NotificationDate)
	fmt.Fprintf(w, "    State Comment:    %s\n", sub.Properties.StateComment)
	fmt.Fprintf(w, "    Allow Tracing:    %t\n", sub.Properties.AllowTracing)
	if showKeys {
		fmt.Fprintf(w, "    Primary Key:      %s\n", sub.Properties.PrimaryKey)
		fmt.Fprintf(w, "    Secondary Key:    %s\n", sub.Properties.SecondaryKey)
	}
}
