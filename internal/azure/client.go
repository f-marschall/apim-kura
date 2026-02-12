package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/apimanagement/armapimanagement"
)

// Client provides methods for interacting with Azure API Management
type Client struct {
	subscriptionID string
	resourceGroup  string
	apimName       string
	credential     *azidentity.AzureCLICredential
	clientFactory  *armapimanagement.ClientFactory
}

// SubscriptionInfo mirrors the Azure REST API SubscriptionContract schema.
type SubscriptionInfo struct {
	ID         string                     `json:"id"`
	Name       string                     `json:"name"`
	Type       string                     `json:"type"`
	Properties SubscriptionInfoProperties `json:"properties"`
}

// SubscriptionInfoProperties holds the properties of a SubscriptionContract.
type SubscriptionInfoProperties struct {
	OwnerID          string `json:"ownerId,omitempty"`
	Scope            string `json:"scope"`
	DisplayName      string `json:"displayName"`
	State            string `json:"state"`
	CreatedDate      string `json:"createdDate,omitempty"`
	StartDate        string `json:"startDate,omitempty"`
	EndDate          string `json:"endDate,omitempty"`
	ExpirationDate   string `json:"expirationDate,omitempty"`
	NotificationDate string `json:"notificationDate,omitempty"`
	PrimaryKey       string `json:"primaryKey"`
	SecondaryKey     string `json:"secondaryKey"`
	StateComment     string `json:"stateComment,omitempty"`
	AllowTracing     bool   `json:"allowTracing"`
}

// NewClient creates a new Azure API Management client using Azure CLI credentials
func NewClient(ctx context.Context, subscriptionID, resourceGroup, apimName string) (*Client, error) {
	// If no subscription ID provided, resolve it from Azure CLI
	if subscriptionID == "" {
		id, err := resolveSubscriptionID()
		if err != nil {
			return nil, fmt.Errorf("no subscription ID provided and failed to resolve from Azure CLI: %w", err)
		}
		subscriptionID = id
	}

	// Use Azure CLI credentials
	cred, err := azidentity.NewAzureCLICredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with Azure CLI: %w", err)
	}

	// Create the client factory
	clientFactory, err := armapimanagement.NewClientFactory(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure API Management client factory: %w", err)
	}

	return &Client{
		subscriptionID: subscriptionID,
		resourceGroup:  resourceGroup,
		apimName:       apimName,
		credential:     cred,
		clientFactory:  clientFactory,
	}, nil
}

// SubscriptionID returns the Azure subscription ID used by this client.
func (c *Client) SubscriptionID() string {
	return c.subscriptionID
}

// ListSubscriptions returns APIM subscriptions including their secret keys.
// If productID is non-empty, only subscriptions scoped to that product are returned.
func (c *Client) ListSubscriptions(ctx context.Context, productID string) ([]SubscriptionInfo, error) {
	subClient := c.clientFactory.NewSubscriptionClient()

	// Build a page iterator depending on whether we filter by product.
	type page struct {
		Value []*armapimanagement.SubscriptionContract
	}
	var nextPage func() (page, bool, error)

	if productID != "" {
		prodPager := c.clientFactory.NewProductSubscriptionsClient().NewListPager(c.resourceGroup, c.apimName, productID, nil)
		nextPage = func() (page, bool, error) {
			if !prodPager.More() {
				return page{}, false, nil
			}
			p, err := prodPager.NextPage(ctx)
			return page{Value: p.Value}, true, err
		}
	} else {
		allPager := subClient.NewListPager(c.resourceGroup, c.apimName, nil)
		nextPage = func() (page, bool, error) {
			if !allPager.More() {
				return page{}, false, nil
			}
			p, err := allPager.NextPage(ctx)
			return page{Value: p.Value}, true, err
		}
	}

	var results []SubscriptionInfo
	for {
		p, more, err := nextPage()
		if err != nil {
			return nil, fmt.Errorf("failed to list subscriptions: %w", err)
		}
		if !more {
			break
		}

		for _, sub := range p.Value {
			if sub == nil || sub.Properties == nil {
				continue
			}

			info := SubscriptionInfo{
				ID:   deref(sub.ID),
				Name: deref(sub.Name),
				Type: deref(sub.Type),
				Properties: SubscriptionInfoProperties{
					OwnerID:      deref(sub.Properties.OwnerID),
					Scope:        deref(sub.Properties.Scope),
					DisplayName:  deref(sub.Properties.DisplayName),
					State:        string(*sub.Properties.State),
					StateComment: deref(sub.Properties.StateComment),
				},
			}

			if sub.Properties.AllowTracing != nil {
				info.Properties.AllowTracing = *sub.Properties.AllowTracing
			}
			if sub.Properties.CreatedDate != nil {
				info.Properties.CreatedDate = sub.Properties.CreatedDate.Format("2006-01-02T15:04:05Z")
			}
			if sub.Properties.StartDate != nil {
				info.Properties.StartDate = sub.Properties.StartDate.Format("2006-01-02T15:04:05Z")
			}
			if sub.Properties.EndDate != nil {
				info.Properties.EndDate = sub.Properties.EndDate.Format("2006-01-02T15:04:05Z")
			}
			if sub.Properties.ExpirationDate != nil {
				info.Properties.ExpirationDate = sub.Properties.ExpirationDate.Format("2006-01-02T15:04:05Z")
			}
			if sub.Properties.NotificationDate != nil {
				info.Properties.NotificationDate = sub.Properties.NotificationDate.Format("2006-01-02T15:04:05Z")
			}

			secrets, err := subClient.ListSecrets(ctx, c.resourceGroup, c.apimName, deref(sub.Name), nil)
			if err != nil {
				return nil, fmt.Errorf("failed to get secrets for subscription %s: %w", deref(sub.Name), err)
			}
			info.Properties.PrimaryKey = deref(secrets.PrimaryKey)
			info.Properties.SecondaryKey = deref(secrets.SecondaryKey)

			results = append(results, info)
		}
	}

	return results, nil
}

// CreateSubscriptionOptions holds optional parameters for creating a subscription.
type CreateSubscriptionOptions struct {
	PrimaryKey   string
	SecondaryKey string
	State        string
	OwnerID      string
	AllowTracing *bool
}

// CreateSubscription creates (or updates) an APIM subscription key.
// sid is the subscription entity identifier (e.g. a GUID).
// scope is the full resource ID of the product or API the subscription is scoped to.
// displayName is the human-readable name for the subscription.
func (c *Client) CreateSubscription(ctx context.Context, sid, scope, displayName string, opts *CreateSubscriptionOptions) (*SubscriptionInfo, error) {
	if opts == nil {
		opts = &CreateSubscriptionOptions{}
	}

	params := armapimanagement.SubscriptionCreateParameters{
		Properties: &armapimanagement.SubscriptionCreateParameterProperties{
			Scope:       &scope,
			DisplayName: &displayName,
		},
	}

	if opts.PrimaryKey != "" {
		params.Properties.PrimaryKey = &opts.PrimaryKey
	}
	if opts.SecondaryKey != "" {
		params.Properties.SecondaryKey = &opts.SecondaryKey
	}
	if opts.OwnerID != "" {
		params.Properties.OwnerID = &opts.OwnerID
	}
	if opts.AllowTracing != nil {
		params.Properties.AllowTracing = opts.AllowTracing
	}
	if opts.State != "" {
		state := armapimanagement.SubscriptionState(opts.State)
		params.Properties.State = &state
	}

	subClient := c.clientFactory.NewSubscriptionClient()

	resp, err := subClient.CreateOrUpdate(ctx, c.resourceGroup, c.apimName, sid, params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription %s: %w", sid, err)
	}

	sub := resp.SubscriptionContract
	info := SubscriptionInfo{
		ID:   deref(sub.ID),
		Name: deref(sub.Name),
		Type: deref(sub.Type),
		Properties: SubscriptionInfoProperties{
			OwnerID:     deref(sub.Properties.OwnerID),
			Scope:       deref(sub.Properties.Scope),
			DisplayName: deref(sub.Properties.DisplayName),
			State:       string(*sub.Properties.State),
		},
	}

	if sub.Properties.AllowTracing != nil {
		info.Properties.AllowTracing = *sub.Properties.AllowTracing
	}
	if sub.Properties.CreatedDate != nil {
		info.Properties.CreatedDate = sub.Properties.CreatedDate.Format("2006-01-02T15:04:05Z")
	}
	if sub.Properties.StartDate != nil {
		info.Properties.StartDate = sub.Properties.StartDate.Format("2006-01-02T15:04:05Z")
	}

	// Fetch the secrets since CreateOrUpdate does not return them.
	secrets, err := subClient.ListSecrets(ctx, c.resourceGroup, c.apimName, sid, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get secrets for subscription %s: %w", sid, err)
	}
	info.Properties.PrimaryKey = deref(secrets.PrimaryKey)
	info.Properties.SecondaryKey = deref(secrets.SecondaryKey)

	return &info, nil
}

// DeleteSubscription deletes an APIM subscription by its ID.
func (c *Client) DeleteSubscription(ctx context.Context, sid string) error {
	subClient := c.clientFactory.NewSubscriptionClient()
	_, err := subClient.Delete(ctx, c.resourceGroup, c.apimName, sid, "*", nil)
	if err != nil {
		return fmt.Errorf("failed to delete subscription %s: %w", sid, err)
	}
	return nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// resolveSubscriptionID gets the current subscription ID from Azure CLI.
func resolveSubscriptionID() (string, error) {
	out, err := exec.Command("az", "account", "show", "--query", "id", "-o", "tsv").Output()
	if err != nil {
		// Fallback: try JSON output
		out2, err2 := exec.Command("az", "account", "show", "-o", "json").Output()
		if err2 != nil {
			return "", fmt.Errorf("failed to run 'az account show': %w", err)
		}
		var acc struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(out2, &acc); err != nil {
			return "", fmt.Errorf("failed to parse 'az account show' output: %w", err)
		}
		if acc.ID == "" {
			return "", fmt.Errorf("no subscription ID found in 'az account show' output")
		}
		return acc.ID, nil
	}

	id := string(out)
	// Trim whitespace/newlines
	for len(id) > 0 && (id[len(id)-1] == '\n' || id[len(id)-1] == '\r' || id[len(id)-1] == ' ') {
		id = id[:len(id)-1]
	}
	if id == "" {
		return "", fmt.Errorf("empty subscription ID from 'az account show'")
	}
	return id, nil
}
