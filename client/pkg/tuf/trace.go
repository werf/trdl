package tuf

import (
	"net/http"
	"net/http/httptrace"
	"time"

	"github.com/werf/trdl/client/internal/logger"
)

type TracingTransport struct {
	Transport http.RoundTripper
	Logger    logger.Logger
}

func (t *TracingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	log := t.Logger.With("sorce", "tuf-client")
	startTime := time.Now()

	log.Debug("Request started",
		"method", req.Method,
		"url", req.URL.String(),
	)

	trace := &httptrace.ClientTrace{
		DNSDone: func(info httptrace.DNSDoneInfo) {
			log.Debug("DNS lookup done", "host", info.Addrs)
		},
		ConnectDone: func(network, addr string, err error) {
			if err != nil {
				log.Debug("Failed to connect",
					"host", addr,
					"error", err,
				)
				return
			}
			log.Debug("Connected", "address", addr)
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	resp, err := t.Transport.RoundTrip(req)
	if err != nil {
		log.Debug("Failed to send request",
			"url", req.URL.String(),
			"error", err,
		)
		return nil, err
	}

	log.Debug("Request completed",
		"status", resp.Status,
		"duration", time.Since(startTime).Seconds(),
	)

	return resp, nil
}
