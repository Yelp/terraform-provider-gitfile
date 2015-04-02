package main

import (
	"github.com/hashicorp/terraform/plugin"
	"terraform-provider-gitfile/gitfile"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: gitfile.Provider,
	})
}
