package registry011

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"time"
)

// SepoliaRPCURL is the sole endpoint approved for this observation client.
const SepoliaRPCURL = "https://rpc.sepolia.ethpandaops.io"

// SepoliaRegistryAddress is the exact instance selected for these reads.
const SepoliaRegistryAddress = "0x9361223865e641f9941f4640d8fe34faad54bf23"
const sepoliaGenesis = "0x25a5cc106eea7138acab33231d7160d69cb777ee0c2c553fcddf5138993e6dd9"

var ErrRPCTransport = errors.New("registry011: RPC transport unavailable")
var ErrRPCState = errors.New("registry011: unusable block-pinned read")

// RPCTransportLimits must be supplied explicitly. Passing tests does not qualify
// these values as an operating policy. MaxBytes is supported up to 8 MiB; request timeout up to five minutes.
type RPCTransportLimits struct {
	RequestTimeout time.Duration
	MaxBytes       int
	MaxDepth       int
}

// ReadRPC has no arbitrary-method, signing, transaction or fallback API. The
// production constructor owns its transport: no environment proxy, cookie jar,
// caller transport, alternate endpoint or disabled certificate verification.
type ReadRPC struct {
	client      *http.Client
	limits      RPCTransportLimits
	acquisition *rpcAcquisition
}

func NewSepoliaReadRPC(limits RPCTransportLimits) (*ReadRPC, error) {
	if limits.RequestTimeout <= 0 || limits.RequestTimeout > 5*time.Minute || limits.MaxBytes <= 0 || limits.MaxBytes > 8<<20 || limits.MaxDepth <= 0 || limits.MaxDepth > 128 {
		return nil, ErrConfiguration
	}
	transport := &http.Transport{
		DialContext:            (&net.Dialer{Timeout: limits.RequestTimeout}).DialContext,
		TLSClientConfig:        &tls.Config{MinVersion: tls.VersionTLS12},
		TLSHandshakeTimeout:    limits.RequestTimeout,
		ResponseHeaderTimeout:  limits.RequestTimeout,
		MaxResponseHeaderBytes: int64(limits.MaxBytes),
		DisableCompression:     true,
		ForceAttemptHTTP2:      true,
	}
	return &ReadRPC{client: &http.Client{
		Transport:     transport,
		Timeout:       limits.RequestTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}, limits: limits}, nil
}

// Close releases idle connections; it does not invalidate retained observations.
func (c *ReadRPC) Close() {
	if c != nil && c.client != nil {
		c.client.CloseIdleConnections()
	}
}

func (c *ReadRPC) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if c == nil {
		return nil, ErrConfiguration
	}
	_, result, err := c.callEnvelope(ctx, method, params, c.limits.MaxBytes)
	return result, err
}

// callEnvelope applies the smaller remaining acquisition budget while streaming,
// before JSON decoding. It retains the exact validated response for audit.
func (c *ReadRPC) callEnvelope(ctx context.Context, method string, params any, maximum int) (json.RawMessage, json.RawMessage, error) {
	if c == nil || c.client == nil || ctx == nil || maximum <= 0 {
		return nil, nil, ErrConfiguration
	}
	maximum = min(maximum, c.limits.MaxBytes)
	if c.acquisition != nil {
		child, allowed, release, err := c.acquisition.admit(ctx, maximum)
		if err != nil {
			return nil, nil, err
		}
		defer release()
		ctx, maximum = child, allowed
	}
	// Only internal fixed methods reach this boundary. IDs are scoped to a single
	// request/response exchange; no batched or asynchronous demultiplexing exists.
	request, err := json.Marshal(struct {
		Version string `json:"jsonrpc"`
		ID      uint64 `json:"id"`
		Method  string `json:"method"`
		Params  any    `json:"params"`
	}{"2.0", 1, method, params})
	if err != nil {
		return nil, nil, ErrRPCTransport
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, SepoliaRPCURL, bytes.NewReader(request))
	if err != nil {
		return nil, nil, ErrRPCTransport
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	response, err := c.client.Do(req)
	if err != nil {
		return nil, nil, ErrRPCTransport
	}
	// The complete bounded body is checked below; closing only releases transport resources.
	defer func() { _ = response.Body.Close() }()
	contentType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	encoding := response.Header.Get("Content-Encoding")
	if err != nil || contentType != "application/json" || response.StatusCode != http.StatusOK ||
		(encoding != "" && encoding != "identity") || response.ContentLength > int64(maximum) {
		return nil, nil, ErrRPCTransport
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, int64(maximum)+1))
	if c.acquisition != nil {
		c.acquisition.consume(len(body))
	}
	if err != nil || len(body) > maximum {
		return nil, nil, ErrRPCTransport
	}
	result, err := DecodeRPCResult(body, 1, RPCResponseLimits{maximum, c.limits.MaxDepth})
	if err != nil {
		return nil, nil, err
	}
	if c.acquisition != nil {
		if err := c.acquisition.retain(method, params, body); err != nil {
			return nil, nil, err
		}
	}
	return body, result, nil
}
