package main

import (
	"os"

	log "github.com/hashicorp/go-hclog"
	"github.com/openbao/openbao/api/v2"
	"github.com/openbao/openbao/sdk/v2/plugin"

	"github.com/weftspun/service-bao-sqlite-fdb/backend"
)

func main() {
	meta := &api.PluginAPIClientMeta{}
	if err := meta.FlagSet().Parse(os.Args[1:]); err != nil {
		log.Default().Error("plugin flags", "error", err)
		os.Exit(1)
	}
	tlsProvider := api.VaultPluginTLSProvider(meta.GetTLSConfig())

	if err := plugin.Serve(&plugin.ServeOpts{
		BackendFactoryFunc: backend.Factory,
		TLSProviderFunc:    tlsProvider,
	}); err != nil {
		log.Default().Error("plugin serve", "error", err)
		os.Exit(1)
	}
}
