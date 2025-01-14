package docker

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/buildx/commands"
	_ "github.com/docker/buildx/driver/docker"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
	"github.com/werf/logboek"
	"github.com/werf/trdl/server/pkg/secrets"

	"path/filepath"

	cliconfig "github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/go-connections/tlsconfig"
)

const (
	isDebug = false
)

func NewBuildxCommand(dockerCli command.Cli) *cobra.Command {
	cmd := commands.NewRootCmd("", false, dockerCli)
	return cmd
}

func defaultCliOptions(ctx context.Context) []command.CLIOption {
	return []command.CLIOption{
		command.WithInputStream(os.Stdin),
		command.WithOutputStream(logboek.Context(ctx).OutStream()),
		command.WithErrorStream(logboek.Context(ctx).ErrStream()),
		command.WithContentTrust(false),
	}
}

func newDockerCli(opts []command.CLIOption) (command.Cli, error) {
	newCli, err := command.NewDockerCli(opts...)
	if err != nil {
		return nil, err
	}

	clientOpts := flags.NewClientOptions()

	dockerCertPath := os.Getenv("DOCKER_CERT_PATH")
	if dockerCertPath == "" {
		dockerCertPath = cliconfig.Dir()
	}

	clientOpts.TLS = os.Getenv("DOCKER_TLS") != ""
	clientOpts.TLSVerify = os.Getenv("DOCKER_TLS_VERIFY") != ""

	if clientOpts.TLSVerify {
		clientOpts.TLSOptions = &tlsconfig.Options{
			CAFile:   filepath.Join(dockerCertPath, flags.DefaultCaFile),
			CertFile: filepath.Join(dockerCertPath, flags.DefaultCertFile),
			KeyFile:  filepath.Join(dockerCertPath, flags.DefaultKeyFile),
		}
	}

	if isDebug {
		clientOpts.LogLevel = "debug"
	} else {
		clientOpts.LogLevel = "fatal"
	}

	if err := newCli.Initialize(clientOpts); err != nil {
		return nil, err
	}
	return newCli, nil
}

func prepareCliCmd(cmd *cobra.Command, args ...string) *cobra.Command {
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs(args)
	return cmd
}

func setCliArgs(serviceDockerfilePathInContext string, secrets []secrets.Secret) ([]string, error) {
	args := []string{
		"--file", serviceDockerfilePathInContext,
		"--pull",
		"--no-cache",
		"-o", "-",
	}

	if len(secrets) > 0 {
		if err := SetTempEnvVars(secrets); err != nil {
			return nil, fmt.Errorf("unable to set secrets")
		}
		args = append(args, GetSecretsCommandMounts(secrets)...)
	}

	args = append(args, "-")
	return args, nil
}
