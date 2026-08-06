package spinnaker

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Bonial-International-GmbH/terraform-provider-spinnaker/spinnaker/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"server": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "URL for Gate",
				DefaultFunc: schema.EnvDefaultFunc("GATE_URL", nil),
			},
			"config": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Path to Gate config file",
				DefaultFunc: schema.EnvDefaultFunc("SPINNAKER_CONFIG_PATH", nil),
			},
			"ignore_cert_errors": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Ignore certificate errors from Gate",
				Default:     false,
			},
			"default_headers": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Headers to be passed to the gate endpoint by the client on each request",
				Default:     "",
			},
			"oauth2_client_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "OAuth2 client ID. Enables authentication via the OAuth2 client credentials flow together with oauth2_client_secret",
				DefaultFunc: schema.EnvDefaultFunc("SPINNAKER_OAUTH2_CLIENT_ID", nil),
			},
			"oauth2_client_secret": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "OAuth2 client secret. Enables authentication via the OAuth2 client credentials flow together with oauth2_client_id",
				DefaultFunc: schema.EnvDefaultFunc("SPINNAKER_OAUTH2_CLIENT_SECRET", nil),
			},
			"oauth2_token_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "OAuth2 token endpoint. Defaults to the /oauth2/token endpoint of the Gate URL",
				DefaultFunc: schema.EnvDefaultFunc("SPINNAKER_OAUTH2_TOKEN_URL", nil),
			},
			"oauth2_scope": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Space separated list of OAuth2 scopes to request. Defaults to requesting no scopes",
				DefaultFunc: schema.EnvDefaultFunc("SPINNAKER_OAUTH2_SCOPE", nil),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"spinnaker_application":              resourceApplication(),
			"spinnaker_pipeline":                 resourcePipeline(),
			"spinnaker_pipeline_template":        resourcePipelineTemplate(),
			"spinnaker_pipeline_template_config": resourcePipelineTemplateConfig(),
			"spinnaker_pipeline_template_v2":     resourcePipelineTemplateV2(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"spinnaker_pipeline": datasourcePipeline(),
		},
		ConfigureFunc: providerConfigureFunc,
	}
}

type clientConfig struct {
	gateEndpoint     string
	defaultHeaders   string
	configLocation   string
	ignoreCertErrors bool
	oauth2           *api.OAuth2Config

	once   sync.Once
	client *api.Client
	err    error
}

// Client lazily initializes an *api.Client on the first call and returns it.
// Subsequent calls return the same client instance. Returns an error if client
// initialization fails.
func (c *clientConfig) Client() (*api.Client, error) {
	c.once.Do(func() {
		c.client, c.err = api.NewClient(api.Config{
			Endpoint:         c.gateEndpoint,
			DefaultHeaders:   c.defaultHeaders,
			ConfigLocation:   c.configLocation,
			IgnoreCertErrors: c.ignoreCertErrors,
			OAuth2:           c.oauth2,
		})
	})

	return c.client, c.err
}

func providerConfigureFunc(data *schema.ResourceData) (interface{}, error) {
	oauth2Config, err := oauth2ConfigFrom(data)
	if err != nil {
		return nil, err
	}

	c := &clientConfig{
		gateEndpoint:     data.Get("server").(string),
		defaultHeaders:   data.Get("default_headers").(string),
		configLocation:   data.Get("config").(string),
		ignoreCertErrors: data.Get("ignore_cert_errors").(bool),
		oauth2:           oauth2Config,
	}

	return c, nil
}

// oauth2ConfigFrom builds the OAuth2 client credentials configuration from the
// provider configuration. It returns nil if OAuth2 is not configured, which
// leaves authentication to the Gate config file.
func oauth2ConfigFrom(data *schema.ResourceData) (*api.OAuth2Config, error) {
	clientID := data.Get("oauth2_client_id").(string)
	clientSecret := data.Get("oauth2_client_secret").(string)

	switch {
	case clientID == "" && clientSecret == "":
		return nil, nil
	case clientID == "":
		return nil, fmt.Errorf("oauth2_client_id must be set together with oauth2_client_secret")
	case clientSecret == "":
		return nil, fmt.Errorf("oauth2_client_secret must be set together with oauth2_client_id")
	}

	return &api.OAuth2Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     data.Get("oauth2_token_url").(string),
		Scopes:       strings.Fields(data.Get("oauth2_scope").(string)),
	}, nil
}
