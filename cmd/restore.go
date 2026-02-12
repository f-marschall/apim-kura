package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/f-marschall/apim-kura/internal/azure"
	"github.com/spf13/cobra"
)

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore subscription keys to Azure API Management",
	Long: `Restore reads a backup file and restores subscription keys
to an Azure API Management instance.

Example:
  kura restore --resource-group mygroup --apim-name myapim --input backup/mygroup/myapim/subscriptions.json
  kura restore -g mygroup -a myapim -i backup/mygroup/myapim/myproduct/subscriptions.json --dry-run`,
	RunE: runRestore,
}

var (
	restoreResourceGroup string
	restoreAPIMName      string
	restoreSubscription  string
	restoreInput         string
	restoreDryRun        bool
)

func init() {
	rootCmd.AddCommand(restoreCmd)

	// Local flags for the restore command
	restoreCmd.Flags().StringVarP(&restoreResourceGroup, "resource-group", "g", "", "Azure resource group name (required)")
	restoreCmd.Flags().StringVarP(&restoreAPIMName, "apim-name", "a", "", "Azure API Management instance name (required)")
	restoreCmd.Flags().StringVarP(&restoreSubscription, "subscription", "s", "", "Azure subscription ID")
	restoreCmd.Flags().StringVarP(&restoreInput, "input", "i", "", "Backup file path to restore from (required)")
	restoreCmd.Flags().BoolVar(&restoreDryRun, "dry-run", false, "Preview changes without applying them")

	// Mark required flags
	restoreCmd.MarkFlagRequired("resource-group")
	restoreCmd.MarkFlagRequired("apim-name")
	restoreCmd.MarkFlagRequired("input")
}

// extractScopeSuffix extracts the scope suffix after the APIM service name.
// For example, given a scope like:
//
//	/subscriptions/.../service/<apim>/products/<productID>
//
// it returns "products/<productID>".
// For instance-level scopes (ending with /service/<apim> or /service/<apim>/)
// it returns an empty string.
func extractScopeSuffix(scope string) string {
	const marker = "/service/"
	idx := strings.LastIndex(scope, marker)
	if idx == -1 {
		return ""
	}
	// Skip past "/service/<apim-name>"
	rest := scope[idx+len(marker):]
	slashIdx := strings.Index(rest, "/")
	if slashIdx == -1 {
		return ""
	}
	suffix := rest[slashIdx+1:]
	// Trim trailing slash
	suffix = strings.TrimRight(suffix, "/")
	return suffix
}

// buildScopeFromSuffix constructs a full APIM scope resource ID from a suffix.
// If suffix is empty, the scope is the APIM instance itself.
func buildScopeFromSuffix(azureSubscriptionID, resourceGroup, apimName, suffix string) string {
	base := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ApiManagement/service/%s",
		azureSubscriptionID, resourceGroup, apimName,
	)
	if suffix == "" {
		return base
	}
	return base + "/" + suffix
}

func runRestore(cmd *cobra.Command, args []string) error {
	fmt.Printf("Restoring subscription keys to APIM instance: %s\n", restoreAPIMName)
	fmt.Printf("Resource Group: %s\n", restoreResourceGroup)
	fmt.Printf("Input file: %s\n", restoreInput)

	if restoreSubscription != "" {
		fmt.Printf("Subscription ID: %s\n", restoreSubscription)
	}

	if restoreDryRun {
		fmt.Println("\nRunning in DRY-RUN mode. No changes will be applied.")
	}

	// 1. Read and parse the backup file.
	data, err := os.ReadFile(restoreInput)
	if err != nil {
		return fmt.Errorf("failed to read input file %s: %w", restoreInput, err)
	}

	var subs []azure.SubscriptionInfo
	if err := json.Unmarshal(data, &subs); err != nil {
		return fmt.Errorf("failed to parse input file: %w", err)
	}

	if len(subs) == 0 {
		fmt.Println("No subscriptions found in input file. Nothing to restore.")
		return nil
	}
	fmt.Printf("\nFound %d subscription(s) to restore\n", len(subs))

	// 2. Authenticate to Azure.
	ctx := context.Background()
	fmt.Println("\nAuthenticating with Azure CLI...")

	client, err := azure.NewClient(ctx, restoreSubscription, restoreResourceGroup, restoreAPIMName)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	fmt.Println("Successfully authenticated with Azure CLI")

	// Resolve the Azure subscription ID so we can rebuild scopes.
	azureSubID := client.SubscriptionID()

	// 3. Restore each subscription.
	var restored, failed int
	for _, sub := range subs {
		sid := sub.Name // The subscription entity ID (GUID).
		displayName := sub.Properties.DisplayName

		// Determine the target scope.
		// Extract the scope suffix from the backup and rebuild for the target environment.
		scopeSuffix := extractScopeSuffix(sub.Properties.Scope)
		scope := buildScopeFromSuffix(azureSubID, restoreResourceGroup, restoreAPIMName, scopeSuffix)

		opts := &azure.CreateSubscriptionOptions{
			PrimaryKey:   sub.Properties.PrimaryKey,
			SecondaryKey: sub.Properties.SecondaryKey,
			State:        sub.Properties.State,
		}
		if sub.Properties.OwnerID != "" {
			opts.OwnerID = sub.Properties.OwnerID
		}
		allowTracing := sub.Properties.AllowTracing
		opts.AllowTracing = &allowTracing

		scopeLabel := scopeSuffix
		if scopeLabel == "" {
			scopeLabel = "(instance)"
		}

		if restoreDryRun {
			fmt.Printf("  [DRY-RUN] Would restore: %s (sid=%s, scope=%s)\n", displayName, sid, scopeLabel)
			restored++
			continue
		}

		fmt.Printf("  Restoring: %s (sid=%s, scope=%s)...\n", displayName, sid, scopeLabel)
		_, err := client.CreateSubscription(ctx, sid, scope, displayName, opts)
		if err != nil {
			fmt.Printf("  [FAIL] %s: %v\n", displayName, err)
			failed++
			continue
		}
		fmt.Printf("  [OK]   %s\n", displayName)
		restored++
	}

	// 4. Summary.
	fmt.Printf("\nRestore complete: %d succeeded, %d failed (out of %d total)\n", restored, failed, len(subs))
	if failed > 0 {
		return fmt.Errorf("%d subscription(s) failed to restore", failed)
	}
	return nil
}
