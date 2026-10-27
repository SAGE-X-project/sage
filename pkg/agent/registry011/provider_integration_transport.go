package registry011

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net"
	"time"
)

// This is the only production dial for the new gated path. There is no
// configurable endpoint, TLS root override, proxy, redirect or retry loop.
func providerIntegrationDial(ctx context.Context, limits ProviderCaptureLimits) (*tls.Conn, error) {
	if ctx == nil || !limits.valid() || ctx.Err() != nil {
		return nil, errProviderIntegration
	}
	dialer := tls.Dialer{NetDialer: &net.Dialer{Timeout: limits.Timeout}, Config: &tls.Config{MinVersion: tls.VersionTLS12, ServerName: "rpc.sepolia.ethpandaops.io", NextProtos: []string{"http/1.1"}}}
	connection, err := dialer.DialContext(ctx, "tcp", "rpc.sepolia.ethpandaops.io:443")
	if err != nil {
		return nil, errProviderIntegration
	}
	conn, ok := connection.(*tls.Conn)
	if !ok {
		_ = connection.Close()
		return nil, errProviderIntegration
	}
	return conn, nil
}

// The caller has already consumed and irreversibly used its private permit.
// Owned normal TLS fixtures may call this private byte-preserving primitive;
// no exported Agent constructor accepts a connection or transport callback.
func providerIntegrationCapture(ctx context.Context, conn *tls.Conn, limits ProviderCaptureLimits, request *providerIntegrationRequest, meter *providerWorkMeter) (*ProviderTransportSample, error) {
	return providerIntegrationCaptureWithBounds(ctx, conn, limits, request, meter, providerTLSBounds{8, 16, 64 << 10})
}

func providerIntegrationCaptureWithBounds(ctx context.Context, conn *tls.Conn, limits ProviderCaptureLimits, request *providerIntegrationRequest, meter *providerWorkMeter, certificates providerTLSBounds) (*ProviderTransportSample, error) {
	if ctx == nil || conn == nil || request == nil || meter == nil || !limits.valid() || !certificates.valid() || len(request.wire) >= limits.WireBytes || request.wire != request.head+request.body {
		return nil, errProviderIntegration
	}
	state := conn.ConnectionState()
	if !providerIntegrationCertificates(state, certificates, meter) {
		return nil, errProviderIntegration
	}
	chains, err := providerTLSIdentity(state)
	deadline, ok := ctx.Deadline()
	if err != nil || !ok || conn.SetDeadline(deadline) != nil || ctx.Err() != nil {
		return nil, errProviderIntegration
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	started := time.Now()
	if n, err := conn.Write([]byte(request.wire)); err != nil || n != len(request.wire) {
		return nil, errProviderIntegration
	}
	remaining := limits.WireBytes - len(request.wire)
	wire := &io.LimitedReader{R: conn, N: int64(remaining) + 1}
	var raw bytes.Buffer
	reader := bufio.NewReaderSize(io.TeeReader(wire, &raw), min(limits.HeaderBytes+1, 4096))
	head, headers, body, result, readErr := readProviderHTTP(reader, limits)
	consumed := remaining + 1 - int(wire.N)
	sample := &ProviderTransportSample{Endpoint: SepoliaRPCURL, Method: request.method, HTTPVersion: "HTTP/1.1", RequestHead: []byte(request.head), RequestBody: []byte(request.body), ResponseHead: head, ResponseBody: body, ResponseWire: raw.Bytes(), Headers: headers, TLSVersion: state.Version, CipherSuite: state.CipherSuite, ServerName: state.ServerName, VerifiedChains: chains, WireBytes: len(request.wire) + consumed, Started: started, Finished: time.Now()}
	if readErr != nil || ctx.Err() != nil || consumed > remaining {
		meter.refuse()
		sample.CaptureError = errProviderIntegration.Error()
		return sample, errProviderIntegration
	}
	var work candidateWorkVector
	work[1] = uint64(sample.WireBytes)
	if !meter.add(work) || !meter.event(providerCodecEvent, uint64(len(head)+len(body))) || !meter.event(providerReadEvent, uint64(consumed)) {
		return sample, errProviderIntegration
	}
	sample.Result, sample.Complete = result, true
	return sample, nil
}

// Check the already verified TLS state separately so finite certificate data
// can exercise shape and quota refusals without a hostile TLS peer. This is
// still only the fixed primitive's input validation, never origin verification
// or permission to dispatch; the real connection and providerTLSIdentity remain
// mandatory above, and every observed DER input has the same metered event.
func providerIntegrationCertificates(state tls.ConnectionState, bounds providerTLSBounds, meter *providerWorkMeter) bool {
	if !bounds.valid() || meter == nil || len(state.VerifiedChains) > bounds.chains {
		return false
	}
	for _, chain := range state.VerifiedChains {
		if len(chain) > bounds.certificates {
			return false
		}
		for _, certificate := range chain {
			if certificate == nil || len(certificate.Raw) > bounds.certificateBytes {
				return false
			}
			if !meter.event(providerTLSEvent, uint64(len(certificate.Raw))) {
				return false
			}
		}
	}
	return true
}
