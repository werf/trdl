package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type smokeLogger struct{ t *testing.T }

func (l smokeLogger) Info(msg string, args ...interface{}) {
	l.t.Log(append([]interface{}{msg}, args...)...)
}
func (l smokeLogger) Error(msg string, args ...interface{}) {
	l.t.Log(append([]interface{}{msg}, args...)...)
}

type nopCloseBuffer struct{ bytes.Buffer }

func (b *nopCloseBuffer) Close() error { return nil }

// Requires a running buildkitd reachable at TRDL_SMOKE_BUILDKITD_ADDRESS.
func TestBuildkitSmoke(t *testing.T) {
	address := os.Getenv("TRDL_SMOKE_BUILDKITD_ADDRESS")
	if address == "" {
		t.Skip("TRDL_SMOKE_BUILDKITD_ADDRESS is not set")
	}

	dockerfile := `FROM alpine:3.20 AS builder
RUN mkdir -p /result && echo -n hello-from-buildkit > /result/artifact.txt
RUN --mount=type=secret,id=MY_SECRET cp /run/secrets/MY_SECRET /result/secret.txt
FROM scratch
COPY --from=builder /result /result/
`

	var contextBuf bytes.Buffer
	tw := tar.NewWriter(&contextBuf)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: ".trdl/Dockerfile", Mode: 0o644, Size: int64(len(dockerfile))}))
	_, err := tw.Write([]byte(dockerfile))
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	out := &nopCloseBuffer{}
	err = buildWithBuildkit(
		context.Background(),
		address,
		".trdl/Dockerfile",
		map[string][]byte{"MY_SECRET": []byte("secret-value")},
		io.NopCloser(&contextBuf),
		out,
		smokeLogger{t},
	)
	require.NoError(t, err)

	files := map[string]string{}
	tr := tar.NewReader(&out.Buffer)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		if hdr.Typeflag == tar.TypeReg {
			data, err := io.ReadAll(tr)
			require.NoError(t, err)
			files[hdr.Name] = string(data)
		}
	}

	assert.Equal(t, "hello-from-buildkit", files["result/artifact.txt"])
	assert.Equal(t, "secret-value", files["result/secret.txt"])
}
