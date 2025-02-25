package tuf

import (
	"fmt"
	"net/http"
	"net/http/httptrace"
)

type TracingTransport struct {
	Transport http.RoundTripper
}

func (t *TracingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	prefix := "[TRACE tuf client]"
	fmt.Printf("%s Making request to: %s\n", prefix, req.URL.String())
	trace := &httptrace.ClientTrace{
		DNSDone: func(info httptrace.DNSDoneInfo) {
			fmt.Printf("%s Finished DNS lookup: %v\n", prefix, info.Addrs)
		},
		ConnectDone: func(network, addr string, err error) {
			if err != nil {
				fmt.Printf("%s Connection failed: %s over %s, error: %v\n", prefix, addr, network, err)
			} else {
				fmt.Printf("%s Connected to %s over %s\n", prefix, addr, network)
			}
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	return t.Transport.RoundTrip(req)
}
