package main

import (
	"github.com/Yelp/terraform-provider-gitfile/gitfile"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: gitfile.Provider,
	})
}
