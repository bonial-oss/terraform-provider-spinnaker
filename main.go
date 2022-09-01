package main

import (
	"github.com/Bonial-International-GmbH/terraform-provider-spinnaker/spinnaker"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() *schema.Provider {
			return spinnaker.Provider()
		},
	})
}
