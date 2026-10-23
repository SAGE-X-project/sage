package registry011

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type rpcRoundTrip func(*http.Request) (*http.Response, error)

func (f rpcRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func testReadRPC(t *testing.T, f rpcRoundTrip) *ReadRPC {
	t.Helper()
	c, err := NewSepoliaReadRPC(RPCTransportLimits{time.Second, 2 << 20, 16})
	if err != nil {
		t.Fatal(err)
	}
	c.client.Transport = f
	t.Cleanup(c.Close)
	return c
}
func jsonResponse(body string) *http.Response {
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), ContentLength: -1}
}
func TestReadRPCConfiguration(t *testing.T) {
	for name, limits := range map[string]RPCTransportLimits{
		"missing timeout": {0, 100, 2}, "negative timeout": {-1, 100, 2}, "missing bytes": {1, 0, 2},
		"negative bytes": {1, -1, 2}, "unsupported bytes": {1, (8 << 20) + 1, 2}, "missing depth": {1, 100, 0}, "unsupported depth": {1, 100, 129},
		"unsupported timeout": {5*time.Minute + 1, 100, 2},
	} {
		t.Run(name, func(t *testing.T) {
			c, err := NewSepoliaReadRPC(limits)
			if c != nil || !errors.Is(err, ErrConfiguration) {
				t.Fatalf("%v %v", c, err)
			}
		})
	}
	c, err := NewSepoliaReadRPC(RPCTransportLimits{time.Second, 8 << 20, 128})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	tr, ok := c.client.Transport.(*http.Transport)
	if !ok || tr.Proxy != nil || tr.TLSClientConfig.InsecureSkipVerify || !tr.DisableCompression || tr.MaxResponseHeaderBytes != 8<<20 || c.client.Jar != nil || c.client.Timeout != time.Second {
		t.Fatal("transport policy changed")
	}
	if c.client.CheckRedirect(&http.Request{}, nil) != http.ErrUseLastResponse {
		t.Fatal("redirect permitted")
	}
	var absent *ReadRPC
	absent.Close()
	(&ReadRPC{}).Close()
	for _, client := range []*ReadRPC{nil, {}} {
		if _, err := client.call(context.Background(), "eth_chainId", []any{}); !errors.Is(err, ErrConfiguration) {
			t.Fatal(err)
		}
	}
	if _, err := c.call(nil, "eth_chainId", []any{}); !errors.Is(err, ErrConfiguration) { //nolint:staticcheck // Verify refusal of an explicitly missing context.
		t.Fatal(err)
	}
}
func TestReadRPCTransportRefusals(t *testing.T) {
	for _, name := range []string{"redirect", "http error", "wrong media", "invalid media", "compressed", "advertised oversize", "stream oversize", "read failure", "dial failure", "cancelled", "wrong envelope"} {
		t.Run(name, func(t *testing.T) {
			called := 0
			c := testReadRPC(t, func(req *http.Request) (*http.Response, error) {
				called++
				if req.URL.String() != SepoliaRPCURL || req.Method != "POST" || req.Header.Get("Content-Type") != "application/json" {
					t.Fatal("request policy")
				}
				response := jsonResponse(`{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`)
				switch name {
				case "redirect":
					response.StatusCode = 302
					response.Header.Set("Location", "https://invalid.example/")
				case "http error":
					response.StatusCode = 503
				case "wrong media":
					response.Header.Set("Content-Type", "text/plain")
				case "invalid media":
					response.Header.Set("Content-Type", ";")
				case "compressed":
					response.Header.Set("Content-Encoding", "gzip")
				case "advertised oversize":
					response.ContentLength = int64(cappedTestBytes + 1)
				case "stream oversize":
					response.Body = io.NopCloser(strings.NewReader(strings.Repeat(" ", cappedTestBytes+1)))
				case "read failure":
					response.Body = io.NopCloser(errorReader{})
				case "dial failure":
					return nil, errors.New("unavailable")
				case "cancelled":
					return nil, req.Context().Err()
				case "wrong envelope":
					response.Body = io.NopCloser(strings.NewReader(`{"jsonrpc":"2.0","id":2,"result":null}`))
				}
				return response, nil
			})
			c.limits.MaxBytes = cappedTestBytes
			ctx := context.Background()
			if name == "cancelled" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			result, err := c.call(ctx, "eth_chainId", []any{})
			if result != nil || err == nil || called > 1 {
				t.Fatalf("%q %v calls%d", result, err, called)
			}
		})
	}
}

const cappedTestBytes = 128

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

// Only a normal owned TLS server is used at runtime. All refusal scenarios use
// inert RoundTripper units, without redirect servers or malformed network peers.
func TestReadRPCOwnedTLSRuntime(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.Header.Get("Accept") != "application/json" {
			t.Error("request")
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`)
	}))
	defer server.Close()
	c, err := NewSepoliaReadRPC(RPCTransportLimits{time.Second, 512, 8})
	if err != nil {
		t.Fatal(err)
	}
	// This unexported test seam routes only the positive exchange to the owned
	// server, retaining its standard certificate-verifying TLS client.
	local := server.Client()
	c.client.Transport = rpcRoundTrip(func(r *http.Request) (*http.Response, error) {
		clone := r.Clone(r.Context())
		u := *clone.URL
		clone.URL = &u
		clone.URL.Host = strings.TrimPrefix(server.URL, "https://")
		clone.Host = clone.URL.Host
		return local.Transport.RoundTrip(clone)
	})
	defer c.Close()
	result, err := c.call(context.Background(), "eth_chainId", []any{})
	if err != nil || string(result) != `"0xaa36a7"` {
		t.Fatalf("%s %v", result, err)
	}
}
