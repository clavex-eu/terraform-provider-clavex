package main

import (
	"context"
	"flag"
	"log"

	"github.com/clavex-eu/terraform-provider-clavex/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// version is stamped at build time: -ldflags "-X main.version=x.y.z"
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run provider in debug mode (for delve / TF_REATTACH_PROVIDERS)")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/clavex-eu/clavex",
		Debug:   debug,
	}

	if err := providerserver.Serve(context.Background(), provider.New(version), opts); err != nil {
		log.Fatal(err.Error())
	}
}
