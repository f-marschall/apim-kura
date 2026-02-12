package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/f-marschall/apim-kura/internal/azure"
	"github.com/spf13/cobra"
)

var compareCmd = &cobra.Command{
	Use:   "compare <file_a> <file_b>",
	Short: "Compare subscription keys in two backup files",
	Long: `Compare loads two backup JSON files and checks if all subscription keys
from the first file exist in the second file with the same attributes.

Master subscriptions are excluded from the comparison. All subscription
attributes must match for a key to be considered equivalent.

Example:
  kura compare before.json after.json
  kura compare -a file1.json -b file2.json`,
	Args: cobra.MaximumNArgs(2),
	RunE: runCompare,
}

var (
	compareFileA string
	compareFileB string
)

func init() {
	rootCmd.AddCommand(compareCmd)

	compareCmd.Flags().StringVarP(&compareFileA, "a", "a", "", "First backup file path")
	compareCmd.Flags().StringVarP(&compareFileB, "b", "b", "", "Second backup file path")
}

func runCompare(cmd *cobra.Command, args []string) error {
	var fileA, fileB string

	// Use positional arguments if provided
	if len(args) == 2 {
		fileA = args[0]
		fileB = args[1]
	} else if len(args) == 1 {
		return fmt.Errorf("expected 2 files, got 1")
	} else {
		// Fall back to flags
		if compareFileA == "" || compareFileB == "" {
			return fmt.Errorf("must provide either two positional arguments or both -a and -b flags")
		}
		fileA = compareFileA
		fileB = compareFileB
	}

	fmt.Printf("Comparing backup files:\n")
	fmt.Printf("  File A: %s\n", fileA)
	fmt.Printf("  File B: %s\n", fileB)

	// Load file A
	subsA, err := loadBackupFile(fileA)
	if err != nil {
		return fmt.Errorf("failed to load file A: %w", err)
	}

	// Load file B
	subsB, err := loadBackupFile(fileB)
	if err != nil {
		return fmt.Errorf("failed to load file B: %w", err)
	}

	// Filter out master subscriptions
	subsA = filterOutMaster(subsA)
	subsB = filterOutMaster(subsB)

	fmt.Printf("\nFile A: %d subscription(s) (master excluded)\n", len(subsA))
	fmt.Printf("File B: %d subscription(s) (master excluded)\n", len(subsB))

	// Compare: check if each key in A exists in B with same attributes
	var matched, missing, mismatch int
	for _, subA := range subsA {
		found := false
		for _, subB := range subsB {
			if subA.Properties.PrimaryKey == subB.Properties.PrimaryKey &&
				subA.Properties.SecondaryKey == subB.Properties.SecondaryKey {
				// Found matching keys, check all attributes
				if attributesEqual(&subA, &subB) {
					fmt.Printf("  [OK]   %s\n", subA.Properties.DisplayName)
					matched++
				} else {
					fmt.Printf("  [DIFF] %s (keys match, attributes differ)\n", subA.Properties.DisplayName)
					printAttributeDifferences(&subA, &subB)
					mismatch++
				}
				found = true
				break
			}
		}

		if !found {
			fmt.Printf("  [MISS] %s (primaryKey=%s)\n", subA.Properties.DisplayName, subA.Properties.PrimaryKey)
			missing++
		}
	}

	fmt.Printf("\nComparison complete: %d matched, %d mismatched, %d missing (out of %d total)\n", matched, mismatch, missing, len(subsA))
	if missing > 0 || mismatch > 0 {
		return fmt.Errorf("%d key(s) missing or attributes differ", missing+mismatch)
	}
	return nil
}

func loadBackupFile(filePath string) ([]azure.SubscriptionInfo, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var subs []azure.SubscriptionInfo
	if err := json.Unmarshal(data, &subs); err != nil {
		return nil, err
	}

	return subs, nil
}

func filterOutMaster(subs []azure.SubscriptionInfo) []azure.SubscriptionInfo {
	var filtered []azure.SubscriptionInfo
	for _, sub := range subs {
		if sub.Name != "master" {
			filtered = append(filtered, sub)
		}
	}
	return filtered
}

func attributesEqual(subA, subB *azure.SubscriptionInfo) bool {
	propsA := &subA.Properties
	propsB := &subB.Properties

	return propsA.DisplayName == propsB.DisplayName &&
		propsA.Scope == propsB.Scope &&
		propsA.State == propsB.State &&
		propsA.OwnerID == propsB.OwnerID &&
		propsA.PrimaryKey == propsB.PrimaryKey &&
		propsA.SecondaryKey == propsB.SecondaryKey &&
		propsA.AllowTracing == propsB.AllowTracing &&
		propsA.StartDate == propsB.StartDate &&
		propsA.EndDate == propsB.EndDate &&
		propsA.ExpirationDate == propsB.ExpirationDate &&
		propsA.NotificationDate == propsB.NotificationDate &&
		propsA.StateComment == propsB.StateComment
}

func printAttributeDifferences(subA, subB *azure.SubscriptionInfo) {
	propsA := &subA.Properties
	propsB := &subB.Properties

	if propsA.DisplayName != propsB.DisplayName {
		fmt.Printf("      displayName: %q != %q\n", propsA.DisplayName, propsB.DisplayName)
	}
	if propsA.Scope != propsB.Scope {
		fmt.Printf("      scope: %q != %q\n", propsA.Scope, propsB.Scope)
	}
	if propsA.State != propsB.State {
		fmt.Printf("      state: %q != %q\n", propsA.State, propsB.State)
	}
	if propsA.OwnerID != propsB.OwnerID {
		fmt.Printf("      ownerId: %q != %q\n", propsA.OwnerID, propsB.OwnerID)
	}
	if propsA.AllowTracing != propsB.AllowTracing {
		fmt.Printf("      allowTracing: %v != %v\n", propsA.AllowTracing, propsB.AllowTracing)
	}
	if propsA.CreatedDate != propsB.CreatedDate {
		fmt.Printf("      createdDate: %q != %q\n", propsA.CreatedDate, propsB.CreatedDate)
	}
	if propsA.StartDate != propsB.StartDate {
		fmt.Printf("      startDate: %q != %q\n", propsA.StartDate, propsB.StartDate)
	}
	if propsA.EndDate != propsB.EndDate {
		fmt.Printf("      endDate: %q != %q\n", propsA.EndDate, propsB.EndDate)
	}
	if propsA.ExpirationDate != propsB.ExpirationDate {
		fmt.Printf("      expirationDate: %q != %q\n", propsA.ExpirationDate, propsB.ExpirationDate)
	}
	if propsA.NotificationDate != propsB.NotificationDate {
		fmt.Printf("      notificationDate: %q != %q\n", propsA.NotificationDate, propsB.NotificationDate)
	}
	if propsA.StateComment != propsB.StateComment {
		fmt.Printf("      stateComment: %q != %q\n", propsA.StateComment, propsB.StateComment)
	}
}
