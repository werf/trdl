package main

import (
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
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

	switch isPprofEnabled := os.Getenv("VAULT_PLUGIN_SECRETS_TRDL_PPROF_ENABLED"); isPprofEnabled {
	case "", "0", "false", "FALSE", "no", "NO":
	default:
		go servePprof()
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

func servePprof() {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		hclog.L().Warn(fmt.Sprintf("can't serve pprof: %s", err))
		return
	}

	hclog.L().Info(fmt.Sprintf("pprof for PID %d will be available on http://127.0.0.1:%d/debug/pprof", os.Getpid(), listener.Addr().(*net.TCPAddr).Port))
	if err := http.Serve(listener, nil); err != nil {
		hclog.L().Warn(fmt.Sprintf("can't serve pprof: %s", err))
		return
	}
}
