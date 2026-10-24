package registry011

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"time"
)

// ErrProviderCapture means no complete data-only transport sample is available.
var ErrProviderCapture = errors.New("registry011: provider transport capture unavailable")

// ProviderCaptureLimits bound one qualification request, not an operating policy.
// WireBytes includes the plaintext HTTP request, response headers, body and
// transfer framing. TLS records and the platform network stack are not counted.
// This probe has no retry, arbitrary method, policy selection or signing API.
type ProviderCaptureLimits struct {
	Timeout                                      time.Duration
	HeaderBytes, BodyBytes, WireBytes, JSONDepth int
}

// ProviderHeader preserves the original name and field order. Value strips only
// surrounding optional whitespace; ResponseHead retains the exact HTTP bytes.
type ProviderHeader struct{ Name, Value string }

// ProviderTransportSample is forgeable diagnostic data, never a checked batch,
// ExchangeV1, retained checkpoint, qualified policy or Source authority. Its
// bytes can only enter those later stages after complete independent checking.
// Complete describes only this bounded HTTP sample. On read refusal, a nonnil
// sample may retain incomplete diagnostic wire bytes together with the error;
// Result is then nil. Planned requests with a failed write return no sample.
// TLS origin validation does not authenticate a backend version or execution.
type ProviderTransportSample struct {
	Endpoint, Method, HTTPVersion                        string
	RequestHead, RequestBody, ResponseHead, ResponseBody []byte
	ResponseWire                                         []byte
	Complete                                             bool
	CaptureError                                         string
	Headers                                              []ProviderHeader
	TLSVersion, CipherSuite                              uint16
	ServerName                                           string
	VerifiedChains                                       [][]string
	WireBytes                                            int
	Started, Finished                                    time.Time
	Result                                               json.RawMessage
}

func (l ProviderCaptureLimits) valid() bool {
	return l.Timeout > 0 && l.Timeout <= 5*time.Minute &&
		l.HeaderBytes > 0 && l.HeaderBytes <= 64<<10 &&
		l.BodyBytes > 0 && l.BodyBytes <= 8<<20 &&
		l.WireBytes > l.HeaderBytes && l.WireBytes <= 16<<20 &&
		l.JSONDepth > 0 && l.JSONDepth <= 128
}

// CaptureSepoliaProviderFinality issues exactly one read-only finalized-block
// request to the sole approved TLS origin. It intentionally negotiates HTTP/1.1
// to retain bounded raw response headers before net/http normalizes them.
// No proxy, redirects, custom transport, insecure TLS, retries or fallback exist.
// The system-root choice is recorded by implementation context but is not yet a
// qualified manifest root-store artifact. This does not activate any policy.
func CaptureSepoliaProviderFinality(ctx context.Context, limits ProviderCaptureLimits) (*ProviderTransportSample, error) {
	if ctx == nil || !limits.valid() {
		return nil, ErrConfiguration
	}
	ctx, cancel := context.WithTimeout(ctx, limits.Timeout)
	defer cancel()
	if ctx.Err() != nil {
		return nil, ErrProviderCapture
	}
	dialer := &tls.Dialer{NetDialer: &net.Dialer{Timeout: limits.Timeout}, Config: &tls.Config{
		MinVersion: tls.VersionTLS12, ServerName: "rpc.sepolia.ethpandaops.io", NextProtos: []string{"http/1.1"},
	}}
	connection, err := dialer.DialContext(ctx, "tcp", "rpc.sepolia.ethpandaops.io:443")
	if err != nil {
		return nil, ErrProviderCapture
	}
	defer func() { _ = connection.Close() }()
	// tls.Dialer returns a TLS connection; refusing any other implementation keeps
	// raw HTTP samples behind the certificate-verifying owned constructor.
	conn, ok := connection.(*tls.Conn)
	if !ok {
		return nil, ErrProviderCapture
	}
	return captureProviderTLS(ctx, conn, limits)
}

func captureProviderTLS(ctx context.Context, conn *tls.Conn, limits ProviderCaptureLimits) (*ProviderTransportSample, error) {
	state := conn.ConnectionState()
	chains, err := providerTLSIdentity(state)
	if err != nil {
		return nil, err
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	deadline, ok := ctx.Deadline()
	if !ok || conn.SetDeadline(deadline) != nil {
		return nil, ErrProviderCapture
	}
	started := time.Now()
	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"eth_getBlockByNumber","params":["finalized",false]}`)
	head := providerRequestHead(len(body))
	if len(head)+len(body) >= limits.WireBytes {
		return nil, ErrProviderCapture
	}
	request := append(append([]byte(nil), head...), body...)
	if n, e := conn.Write(request); e != nil || n != len(request) {
		return nil, ErrProviderCapture
	}
	// One extra byte distinguishes a response exceeding the aggregate wire cap.
	remaining := limits.WireBytes - len(request)
	wire := &io.LimitedReader{R: conn, N: int64(remaining) + 1}
	var raw bytes.Buffer
	reader := bufio.NewReaderSize(io.TeeReader(wire, &raw), min(limits.HeaderBytes+1, 4096))
	responseHead, headers, responseBody, result, readErr := readProviderHTTP(reader, limits)
	consumed := remaining + 1 - int(wire.N)
	sample := &ProviderTransportSample{
		Endpoint: SepoliaRPCURL, Method: "eth_getBlockByNumber", HTTPVersion: "HTTP/1.1",
		RequestHead: head, RequestBody: body, ResponseHead: responseHead, ResponseBody: responseBody,
		ResponseWire: raw.Bytes(), Headers: headers, TLSVersion: state.Version, CipherSuite: state.CipherSuite,
		ServerName: state.ServerName, VerifiedChains: chains, WireBytes: len(request) + consumed,
		Started: started, Finished: time.Now(),
	}
	if readErr != nil || ctx.Err() != nil || consumed > remaining {
		sample.CaptureError = ErrProviderCapture.Error()
		return sample, ErrProviderCapture
	}
	sample.Result = result
	sample.Complete = true
	return sample, nil
}

func providerTLSIdentity(state tls.ConnectionState) ([][]string, error) {
	if !state.HandshakeComplete || state.Version < tls.VersionTLS12 || state.ServerName != "rpc.sepolia.ethpandaops.io" ||
		state.NegotiatedProtocol != "" && state.NegotiatedProtocol != "http/1.1" || len(state.VerifiedChains) == 0 {
		return nil, ErrProviderCapture
	}
	out := make([][]string, len(state.VerifiedChains))
	for i, chain := range state.VerifiedChains {
		if len(chain) == 0 {
			return nil, ErrProviderCapture
		}
		for _, cert := range chain {
			if cert == nil || len(cert.Raw) == 0 {
				return nil, ErrProviderCapture
			}
			digest := sha256.Sum256(cert.Raw)
			out[i] = append(out[i], "0x"+hex.EncodeToString(digest[:]))
		}
	}
	return out, nil
}
