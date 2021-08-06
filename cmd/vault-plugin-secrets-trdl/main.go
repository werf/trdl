package main

import (
	"fmt"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/plugin"

	trdl "github.com/werf/vault-plugin-secrets-trdl"
)

func main() {
	logFile, err := os.OpenFile("trdl.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o666)
	if err != nil {
		panic(fmt.Sprintf("failed to open trdl.log file: %s", err))
	}

	hclog.DefaultOptions = &hclog.LoggerOptions{
		Level:           hclog.Trace,
		IncludeLocation: true,
		Output:          logFile,
	}

	apiClientMeta := &api.PluginAPIClientMeta{}
	flags := apiClientMeta.FlagSet()
	_ = flags.Parse(os.Args[1:]) // Ignore command, strictly parse flags

	tlsConfig := apiClientMeta.GetTLSConfig()
	tlsProviderFunc := api.VaultPluginTLSProvider(tlsConfig)

	if err := plugin.Serve(&plugin.ServeOpts{
		Logger:             hclog.Default(),
		BackendFactoryFunc: trdl.Factory,
		TLSProviderFunc:    tlsProviderFunc,
	}); err != nil {
		hclog.L().Error("plugin shutting down", "error", err)
		os.Exit(1)
	}
}
