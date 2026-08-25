package docker

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/remotecommand"
)

var execParameterCodec = runtime.NewParameterCodec(coreScheme)

// dial reaches buildkitd through the API server's pods/exec channel rather than
// over the network, so the plugin needs no route to the builder pod itself. The
// address is ignored: the BuildKit client is constructed with an empty one and
// every connection comes from here.
func (b *kubernetesBuilder) dial(ctx context.Context, _ string) (net.Conn, error) {
	request := b.restClient.
		Post().
		Namespace(b.namespace).
		Resource("pods").
		Name(b.podName).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: buildkitContainerName,
			Command:   []string{"buildctl", "dial-stdio"},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
		}, execParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(b.restConfig, "POST", request.URL())
	if err != nil {
		return nil, fmt.Errorf("unable to set up an exec stream to %s/%s: %w", b.namespace, b.podName, err)
	}

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()

	go func() {
		logPipe, waitForLogs := logWriter(b.logger)
		defer waitForLogs()

		streamErr := executor.StreamWithContext(ctx, remotecommand.StreamOptions{
			Stdin:  stdinReader,
			Stdout: stdoutWriter,
			Stderr: logPipe,
		})

		// Closing both ends with the stream error fails a caller blocked on the
		// connection instead of leaving it waiting for bytes that cannot arrive.
		stdoutWriter.CloseWithError(streamErr)
		stdinReader.CloseWithError(streamErr)
	}()

	return &execConn{stdin: stdinWriter, stdout: stdoutReader}, nil
}

var _ net.Conn = (*execConn)(nil)

// execConn adapts one exec stream to net.Conn. The BuildKit client half-closes
// the connection when it finishes sending, so CloseWrite and CloseRead have to
// be real: without them the gRPC transport waits for an EOF that never comes.
type execConn struct {
	stdin  *io.PipeWriter
	stdout *io.PipeReader

	closedMu     sync.Mutex
	stdinClosed  bool
	stdoutClosed bool
}

func (c *execConn) Read(p []byte) (int, error) {
	return c.stdout.Read(p)
}

func (c *execConn) Write(p []byte) (int, error) {
	return c.stdin.Write(p)
}

func (c *execConn) CloseWrite() error {
	c.closedMu.Lock()
	c.stdinClosed = true
	c.closedMu.Unlock()

	return c.stdin.Close()
}

func (c *execConn) CloseRead() error {
	c.closedMu.Lock()
	c.stdoutClosed = true
	c.closedMu.Unlock()

	return c.stdout.Close()
}

func (c *execConn) Close() error {
	c.closedMu.Lock()
	stdinClosed, stdoutClosed := c.stdinClosed, c.stdoutClosed
	c.closedMu.Unlock()

	var err error
	if !stdinClosed {
		err = c.CloseWrite()
	}
	if !stdoutClosed {
		if closeErr := c.CloseRead(); err == nil {
			err = closeErr
		}
	}

	return err
}

func (c *execConn) LocalAddr() net.Addr  { return execAddr("local") }
func (c *execConn) RemoteAddr() net.Addr { return execAddr("remote") }

// The stream carries no deadline of its own; canceling the build context is
// what ends it.
func (c *execConn) SetDeadline(time.Time) error      { return nil }
func (c *execConn) SetReadDeadline(time.Time) error  { return nil }
func (c *execConn) SetWriteDeadline(time.Time) error { return nil }

type execAddr string

func (a execAddr) Network() string { return "pods/exec" }
func (a execAddr) String() string  { return string(a) }
