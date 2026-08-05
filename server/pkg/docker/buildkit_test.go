package docker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/werf/trdl/server/pkg/mac_signing"
	"github.com/werf/trdl/server/pkg/secrets"
)

func TestResolveBuildkitdAddress_EmptyKeepsExecPath(t *testing.T) {
	t.Setenv(buildkitdAddressEnv, "")

	address, err := resolveBuildkitdAddress(context.Background(), "")

	require.NoError(t, err)
	assert.Empty(t, address)
}

func TestResolveBuildkitdAddress_ConfiguredWinsOverEnv(t *testing.T) {
	t.Setenv(buildkitdAddressEnv, "tcp://from-env:1234")

	address, err := resolveBuildkitdAddress(context.Background(), "  unix:///run/buildkit/buildkitd.sock  ")

	require.NoError(t, err)
	assert.Equal(t, "unix:///run/buildkit/buildkitd.sock", address)
}

func TestResolveBuildkitdAddress_EnvFallback(t *testing.T) {
	t.Setenv(buildkitdAddressEnv, "tcp://from-env:1234")

	address, err := resolveBuildkitdAddress(context.Background(), "")

	require.NoError(t, err)
	assert.Equal(t, "tcp://from-env:1234", address)
}

func TestResolveBuildkitdAddress_UnsupportedSchemeRejected(t *testing.T) {
	t.Setenv(buildkitdAddressEnv, "")

	address, err := resolveBuildkitdAddress(context.Background(), "ssh://buildhost")

	require.Error(t, err)
	assert.Empty(t, address)
}

func TestValidateBuildkitdAddress(t *testing.T) {
	require.NoError(t, ValidateBuildkitdAddress(context.Background(), ""))
	require.NoError(t, ValidateBuildkitdAddress(context.Background(), "   "))
	require.NoError(t, ValidateBuildkitdAddress(context.Background(), "unix:///run/buildkit/buildkitd.sock"))
	require.NoError(t, ValidateBuildkitdAddress(context.Background(), "tcp://buildkitd:1234"))
	require.NoError(t, ValidateBuildkitdAddress(context.Background(), "docker-container://buildkitd"))
	require.NoError(t, ValidateBuildkitdAddress(context.Background(), "kube-pod://buildkitd-0?namespace=trdl-build"))

	require.Error(t, ValidateBuildkitdAddress(context.Background(), "buildkitd:1234"))
	require.Error(t, ValidateBuildkitdAddress(context.Background(), "ssh://buildhost"))
	require.Error(t, ValidateBuildkitdAddress(context.Background(), "podman-container://buildkitd"))
}

func TestBuildkitFrontendAttrs(t *testing.T) {
	attrs := buildkitFrontendAttrs(".trdl/Dockerfile", "http://buildkit-session/xyz")

	assert.Equal(t, map[string]string{
		"filename":           ".trdl/Dockerfile",
		"context":            "http://buildkit-session/xyz",
		"no-cache":           "",
		"image-resolve-mode": "pull",
	}, attrs)
}

func TestBuildkitSecretsData(t *testing.T) {
	data := buildkitSecretsData([]secrets.Secret{
		{Id: "MY_TOKEN", Data: []byte("token-value")},
	}, &mac_signing.Credentials{
		Certificate:  "cert-data",
		Password:     "cert-password",
		NotaryKeyID:  "key-id",
		NotaryKey:    "key-data",
		NotaryIssuer: "issuer-id",
	})

	identityName := mac_signing.MacSigningCertificateName
	assert.Equal(t, map[string][]byte{
		"MY_TOKEN":                      []byte("token-value"),
		identityName + "_cert":          []byte("cert-data"),
		identityName + "_password":      []byte("cert-password"),
		identityName + "_notary_key_id": []byte("key-id"),
		identityName + "_notary_key":    []byte("key-data"),
		identityName + "_notary_issuer": []byte("issuer-id"),
	}, data)
}

func TestBuildkitSecretsData_NoCredentials(t *testing.T) {
	data := buildkitSecretsData([]secrets.Secret{{Id: "MY_TOKEN", Data: []byte("token-value")}}, nil)
	assert.Equal(t, map[string][]byte{"MY_TOKEN": []byte("token-value")}, data)
}
