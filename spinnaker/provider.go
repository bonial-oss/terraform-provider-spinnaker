package spinnaker

import (
	"os"
	"sync"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	gate "github.com/spinnaker/spin/cmd/gateclient"
	"github.com/spinnaker/spin/cmd/output"
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

	once   sync.Once
	client *gate.GatewayClient
	err    error
}

// Client lazily initializes a *gate.GatewayClient on the first call and
// returns it. Subsequent calls return the same client instance. Returns an
// error if client initialization fails.
func (c *clientConfig) Client() (*gate.GatewayClient, error) {
	c.once.Do(func() {
		c.client, c.err = gate.NewGateClient(
			output.NewUI(true, false, output.MarshalToJson, os.Stdout, os.Stderr),
			c.gateEndpoint,
			c.defaultHeaders,
			c.configLocation,
			c.ignoreCertErrors,
		)
	})

	return c.client, c.err
}

func providerConfigureFunc(data *schema.ResourceData) (interface{}, error) {
	c := &clientConfig{
		gateEndpoint:     data.Get("server").(string),
		defaultHeaders:   data.Get("default_headers").(string),
		configLocation:   data.Get("config").(string),
		ignoreCertErrors: data.Get("ignore_cert_errors").(bool),
	}

	return c, nil
}
