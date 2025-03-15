package tuf

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"strings"
	"time"
)

const (
	eventStart              = "START"
	eventDNSStart           = "DNS_START"
	eventDNSDone            = "DNS_DONE"
	eventConnectStart       = "CONNECT_START"
	eventConnectDone        = "CONNECT_DONE"
	eventConnectError       = "CONNECT_ERROR"
	eventTLSStart           = "TLS_START"
	eventTLSDone            = "TLS_DONE"
	eventTLSError           = "TLS_ERROR"
	eventResponseFirstByte  = "RESPONSE_FIRST_BYTE"
	eventRequestError       = "REQUEST_ERROR"
	eventRequestDone        = "REQUEST_DONE"
)

type TracingTransport struct {
	Transport http.RoundTripper
}

func (t *TracingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	startTime := time.Now()

	logBase := func(indent int, event, format string, args ...interface{}) {
		indentStr := strings.Repeat(" ", 2 * indent)
		fmt.Printf("%s [%.2fs] %s: %s\n",
			indentStr, time.Since(startTime).Seconds(), event, fmt.Sprintf(format, args...))
	}
	
	logMain := func(event, format string, args ...interface{}) {
		logBase(1, event, format, args...)
	}
	
	logStep := func(event, format string, args ...interface{}) {
		logBase(2, event, format, args...)
	}

	logMain(eventStart, "Request to %s %s", req.Method, req.URL.String())

	trace := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			logStep(eventDNSStart, "Lookup %s", info.Host)
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			logStep(eventDNSDone, "Resolved %+v", info.Addrs)
		},
		ConnectStart: func(network, addr string) {
			logStep(eventConnectStart, "Connecting to %s (%s)", addr, network)
		},
		ConnectDone: func(network, addr string, err error) {
			if err != nil {
				logStep(eventConnectError, "Failed to %s: %v", addr, err)
			} else {
				logStep(eventConnectDone, "Connected to %s", addr)
			}
		},
		TLSHandshakeStart: func() {
			logStep(eventTLSStart, "Starting handshake")
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			if err != nil {
				logStep(eventTLSError, "Handshake failed: %v", err)
			} else {
				logStep(eventTLSDone, "Handshake completed")
			}
		},
		GotFirstResponseByte: func() {
			logStep(eventResponseFirstByte, "Received first byte")
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	resp, err := t.Transport.RoundTrip(req)
	if err != nil {
		logMain(eventRequestError, "Request failed: %v", err)
		return nil, err
	}

	logMain(eventRequestDone, "Completed with status %s (Total: %.2fs)", resp.Status, time.Since(startTime).Seconds())
	
	return resp, nil
}