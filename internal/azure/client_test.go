package azure

import "testing"

func TestUserIDFromOwnerID(t *testing.T) {
	tests := []struct {
		name    string
		ownerID string
		want    string
		wantErr bool
	}{
		{name: "bare user ID", ownerID: "user-123", want: "user-123"},
		{name: "APIM user resource ID", ownerID: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.ApiManagement/service/apim/users/user-123", want: "user-123"},
		{name: "users path", ownerID: "/users/user-123/", want: "user-123"},
		{name: "empty owner ID", ownerID: "", wantErr: true},
		{name: "non-user resource ID", ownerID: "/subscriptions/sub/providers/Microsoft.ApiManagement/service/apim/products/product-123", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := userIDFromOwnerID(tt.ownerID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("userIDFromOwnerID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("userIDFromOwnerID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUserInfoDisplayName(t *testing.T) {
	user := UserInfo{FirstName: "Ada", LastName: "Lovelace"}
	if got := user.DisplayName(); got != "Ada Lovelace" {
		t.Errorf("DisplayName() = %q, want %q", got, "Ada Lovelace")
	}

	user = UserInfo{FirstName: "Ada "}
	if got := user.DisplayName(); got != "Ada" {
		t.Errorf("DisplayName() with a missing last name = %q, want %q", got, "Ada")
	}
}
