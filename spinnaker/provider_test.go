package spinnaker

import (
	"context"
	"os"
	"testing"

	"github.com/Bonial-International-GmbH/terraform-provider-spinnaker/spinnaker/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

var (
	testAccProviders map[string]*schema.Provider
	testAccProvider  *schema.Provider
)

func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]*schema.Provider{
		"spinnaker": testAccProvider,
	}
}

func testAccPreCheck(t *testing.T) {
	if os.Getenv("GATE_URL") == "" {
		t.Fatal("GATE_URL must be set for acceptance tests")
	}
	err := testAccProvider.Configure(context.Background(), terraform.NewResourceConfigRaw(nil))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ = Provider()
}

func TestOAuth2ConfigFrom(t *testing.T) {
	tests := []struct {
		name        string
		raw         map[string]interface{}
		expected    *api.OAuth2Config
		expectedErr string
	}{
		{
			name:     "oauth2 disabled if neither client id nor secret are set",
			raw:      map[string]interface{}{},
			expected: nil,
		},
		{
			name: "client id and secret enable oauth2",
			raw: map[string]interface{}{
				"oauth2_client_id":     "the-client-id",
				"oauth2_client_secret": "the-client-secret",
			},
			expected: &api.OAuth2Config{
				ClientID:     "the-client-id",
				ClientSecret: "the-client-secret",
				Scopes:       []string{},
			},
		},
		{
			name: "token url and scopes are picked up",
			raw: map[string]interface{}{
				"oauth2_client_id":     "the-client-id",
				"oauth2_client_secret": "the-client-secret",
				"oauth2_token_url":     "https://auth.example.com/oauth2/token",
				"oauth2_scope":         "scope1 scope2",
			},
			expected: &api.OAuth2Config{
				ClientID:     "the-client-id",
				ClientSecret: "the-client-secret",
				TokenURL:     "https://auth.example.com/oauth2/token",
				Scopes:       []string{"scope1", "scope2"},
			},
		},
		{
			name:        "client id without secret",
			raw:         map[string]interface{}{"oauth2_client_id": "the-client-id"},
			expectedErr: "oauth2_client_secret must be set together with oauth2_client_id",
		},
		{
			name:        "client secret without id",
			raw:         map[string]interface{}{"oauth2_client_secret": "the-client-secret"},
			expectedErr: "oauth2_client_id must be set together with oauth2_client_secret",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := schema.TestResourceDataRaw(t, Provider().Schema, test.raw)

			oauth2Config, err := oauth2ConfigFrom(data)

			if test.expectedErr != "" {
				require.EqualError(t, err, test.expectedErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.expected, oauth2Config)
		})
	}
}
