package cmd

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/f-marschall/apim-kura/internal/azure"
)

func TestListKeyVisibility(t *testing.T) {
	sub := azure.SubscriptionInfo{
		ID: "subscription-id",
		Properties: azure.SubscriptionInfoProperties{
			DisplayName:  "Example subscription",
			PrimaryKey:   "primary-secret",
			SecondaryKey: "secondary-secret",
		},
	}

	flag := listCmd.Flags().Lookup("show-keys")
	if flag == nil || flag.DefValue != "false" || listShowKeys {
		t.Fatal("--show-keys must default to false")
	}
	t.Cleanup(func() {
		if err := listCmd.Flags().Set("show-keys", "false"); err != nil {
			t.Errorf("restore --show-keys: %v", err)
		}
		flag.Changed = false
	})

	for _, tt := range []struct {
		name     string
		showKeys bool
	}{
		{name: "default"},
		{name: "explicitly show keys", showKeys: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := listCmd.Flags().Set("show-keys", strconv.FormatBool(tt.showKeys)); err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			printSubscription(&output, 1, sub, "Example Owner", listShowKeys)
			for _, text := range []string{"Example subscription", "subscription-id", "Example Owner"} {
				if !strings.Contains(output.String(), text) {
					t.Errorf("output missing %q", text)
				}
			}
			for _, text := range []string{"primary-secret", "secondary-secret", "Primary Key:", "Secondary Key:"} {
				if strings.Contains(output.String(), text) != tt.showKeys {
					t.Errorf("output contains %q = %t, want %t", text, strings.Contains(output.String(), text), tt.showKeys)
				}
			}
		})
	}
}
