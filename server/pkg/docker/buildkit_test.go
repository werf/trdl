package docker

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/djherbis/buffer"
	"github.com/djherbis/nio/v3"
	"github.com/moby/buildkit/session/auth"
	bksecrets "github.com/moby/buildkit/session/secrets"
	"github.com/moby/buildkit/session/upload"
	"github.com/moby/buildkit/session/upload/uploadprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/werf/logboek"
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

func TestBuildkitSessionAttachables_ServeContextSecretsAndRegistryAuth(t *testing.T) {
	attachables := buildkitSessionAttachables(
		context.Background(),
		uploadprovider.New(),
		map[string][]byte{"MY_TOKEN": []byte("token-value")},
	)

	var servesContext, servesSecrets, servesRegistryAuth bool
	for _, attachable := range attachables {
		if _, ok := attachable.(upload.UploadServer); ok {
			servesContext = true
		}
		if _, ok := attachable.(bksecrets.SecretsServer); ok {
			servesSecrets = true
		}
		if _, ok := attachable.(auth.AuthServer); ok {
			servesRegistryAuth = true
		}
	}

	assert.True(t, servesContext, "build context is not served to buildkitd, every build would fail to resolve its context")
	assert.True(t, servesSecrets, "build secrets are not served to buildkitd, secret mounts would be empty")
	assert.True(t, servesRegistryAuth, "registry credentials are not served to buildkitd, private base images would fail to pull")
}

func TestValidateBuildkitdAddress_RejectsSchemeWithoutEndpoint(t *testing.T) {
	ctx := context.Background()

	require.Error(t, ValidateBuildkitdAddress(ctx, "unix://"))
	require.Error(t, ValidateBuildkitdAddress(ctx, "tcp://   "))
	require.NoError(t, ValidateBuildkitdAddress(ctx, "unix:///run/buildkit/buildkitd.sock"))
}

type recordingLogger struct {
	mu    sync.Mutex
	lines []string
}

func (l *recordingLogger) Info(msg string, _ ...interface{}) {
	// A logger slow enough that an unsynchronized wait would return before the
	// last lines are logged.
	time.Sleep(30 * time.Millisecond)

	l.mu.Lock()
	defer l.mu.Unlock()
	l.lines = append(l.lines, msg)
}

func (l *recordingLogger) Error(msg string, _ ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lines = append(l.lines, msg)
}

func TestLogWriter_WaitReturnsOnlyAfterEveryLineIsLogged(t *testing.T) {
	logger := &recordingLogger{}
	writer, waitForLogs := logWriter(logger)

	for _, line := range []string{"first", "second", "third"} {
		_, err := io.WriteString(writer, line+"\n")
		require.NoError(t, err)
	}
	waitForLogs()

	logger.mu.Lock()
	defer logger.mu.Unlock()
	assert.Equal(t, []string{"first", "second", "third"}, logger.lines)
}

func TestLogWriter_OversizedLineDoesNotBlockTheBuild(t *testing.T) {
	logger := &recordingLogger{}
	writer, waitForLogs := logWriter(logger)

	writeErr := make(chan error, 1)
	go func() {
		if _, err := writer.Write(append(bytes.Repeat([]byte("x"), maxLogLineSize+1), '\n')); err != nil {
			writeErr <- err
			return
		}
		_, err := io.WriteString(writer, "line after the oversized one\n")
		writeErr <- err
	}()

	select {
	case err := <-writeErr:
		require.NoError(t, err)
	case <-time.After(30 * time.Second):
		t.Fatal("the build is blocked writing its output after an oversized log line")
	}
	waitForLogs()

	logger.mu.Lock()
	defer logger.mu.Unlock()
	assert.Contains(t, strings.Join(logger.lines, "\n"), "unable to read build output",
		"dropping the rest of the build output has to be reported")
}

func TestMacSigningSecrets_PasswordlessCertificateIsStillServed(t *testing.T) {
	credentials := &mac_signing.Credentials{
		Certificate:  "cert-data",
		NotaryKeyID:  "key-id",
		NotaryKey:    "key-data",
		NotaryIssuer: "issuer-id",
	}
	passwordId := mac_signing.MacSigningCertificateName + "_password"

	// The generated Dockerfile mounts the password unconditionally and reads it
	// under `set -e`, so both build paths have to serve it even when it is empty.
	data := buildkitSecretsData(nil, credentials)
	if assert.Contains(t, data, passwordId) {
		assert.Empty(t, data[passwordId])
	}

	assert.Contains(t, GetMacSigningCommandMounts(credentials), "id="+passwordId)
}

func TestNewBuilder_ReportsUnusedBuildxSettingsInBuildkitMode(t *testing.T) {
	t.Setenv(buildkitdAddressEnv, "tcp://buildkitd:1234")
	logger := &recordingLogger{}

	// The address from the environment cannot be rejected at configure time, so
	// the operator has to learn from the build log why the driver is unused.
	builder, err := NewBuilder(context.Background(), &NewBuilderOpts{
		BuildId:                 "42",
		DockerfilePathInContext: ".trdl/Dockerfile",
		BuildxDriver:            "kubernetes",
		Logger:                  logger,
	})

	require.NoError(t, err)
	assert.Equal(t, "tcp://buildkitd:1234", builder.buildkitdAddress)

	logger.mu.Lock()
	defer logger.mu.Unlock()
	assert.Contains(t, strings.Join(logger.lines, "\n"), "buildx driver settings are not used")
}

func TestBuild_UnblocksContextProducerWhenBuildFails(t *testing.T) {
	// A one-byte pipe buffer puts the producer in the state a real context reaches
	// once it outgrows the buffer: blocked on write until the build drains it.
	contextReader, contextWriter := nio.Pipe(buffer.New(1))
	_, tarWriter := nio.Pipe(buffer.New(1))

	producerErr := make(chan error, 1)
	go func() {
		_, err := contextWriter.Write(make([]byte, 4096))
		producerErr <- err
	}()

	builder := &Builder{
		buildkitdAddress: "unix:///nonexistent/trdl-buildkitd.sock",
		dockerfilePath:   ".trdl/Dockerfile",
		logger:           smokeLogger{t},
	}

	ctx, cancel := context.WithTimeout(logboek.NewContext(context.Background(), logboek.DefaultLogger()), 30*time.Second)
	defer cancel()

	require.Error(t, builder.Build(ctx, contextReader, tarWriter))

	select {
	case err := <-producerErr:
		assert.ErrorIs(t, err, io.ErrClosedPipe)
	case <-time.After(10 * time.Second):
		t.Fatal("build context producer is still blocked after the build failed")
	}
}
