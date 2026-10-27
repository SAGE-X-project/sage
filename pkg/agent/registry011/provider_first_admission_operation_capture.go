package registry011

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"time"
)

// The closed live primitive has one HTTP codec, one wire read, three request
// copies and two native request comparisons. For each bounded certificate it
// adds TLS+hash, one 32-byte digest encode and one 66-byte identity copy. The
// explicit event metric includes no trusted clocks or future C/S retention.
func providerFirstOperationCaptureUpper(requestBytes uint64, limits ProviderCaptureLimits, certificates providerTLSBounds) (candidateWorkVector, bool) {
	var work candidateWorkVector
	if !limits.valid() || !certificates.valid() || requestBytes == 0 || requestBytes > 16<<10 || requestBytes >= uint64(limits.WireBytes) {
		return work, false
	}
	response := uint64(limits.WireBytes) - requestBytes
	quadratic, a := providerUpperProduct(response, response)
	quadratic, b := providerUpperSum(quadratic, 1)
	request, c := providerUpperProduct(requestBytes, 4)
	linear, d := providerUpperSum(response, request, 6)
	count, e := providerUpperProduct(uint64(certificates.chains), uint64(certificates.certificates))
	der, f := providerUpperProduct(uint64(certificates.certificateBytes), 2)
	perCertificate, g := providerUpperSum(der, 2, 32*32+1, 67)
	tls, h := providerUpperProduct(count, perCertificate)
	units, i := providerUpperSum(quadratic, linear, tls)
	if !a || !b || !c || !d || !e || !f || !g || !h || !i {
		return work, false
	}
	work[1], work[2], work[10] = uint64(limits.WireBytes), response, units
	return work, true
}

// This is the closed live HTTP/RPC primitive of the future first operation.
// Its one aggregate codec event covers the bounded readProviderHTTP call,
// including framing and JSON-RPC decoding. It preflights the complete wire
// ceiling BEFORE reading and records only the actual response wire afterward.
// Retained-original replay is an additional invocation with its own observer.
// This logical metric is not a measurement of CPU instructions or TLS traffic.
// Its only connected caller must already have burned the private one-use C
// permit. Calling this byte primitive does not supply that permit or an ACK.
func providerFirstOperationCapture(ctx context.Context, conn *tls.Conn, limits ProviderCaptureLimits, request *providerIntegrationRequest, meter *providerWorkMeter, certificates providerTLSBounds) (*ProviderTransportSample, error) {
	complete := false
	defer func() {
		if !complete {
			meter.refuse() // No error path can reuse this execution observer.
		}
	}()
	if ctx == nil || ctx.Err() != nil || conn == nil || request == nil || meter == nil || meter.integration == nil || meter.integration.store == nil || meter.integration.store.config.bounds.firstAdmission == nil || meter.integration.store.activeMeter.Load() != meter || meter.scan != nil || meter.authority || meter.closed || meter.failed || !limits.valid() || !certificates.valid() || len(request.wire) >= limits.WireBytes || len(request.wire) > 16<<10 || len(request.head) > 512 || len(request.body) > 16<<10 || len(request.wire) != len(request.head)+len(request.body) {
		return nil, errProviderIntegration
	}
	upper, ok := providerFirstOperationCaptureUpper(uint64(len(request.wire)), limits, certificates)
	if !ok || !meter.preflightWork(upper) || !providerReplayTextEqual(request.wire[:len(request.head)], request.head, meter) || !providerReplayTextEqual(request.wire[len(request.head):], request.body, meter) {
		return nil, errProviderIntegration
	}
	state := conn.ConnectionState()
	chains, err := providerFirstOperationTLSIdentity(state, certificates, meter)
	if err != nil {
		return nil, errProviderIntegration
	}
	deadline, hasDeadline := ctx.Deadline()
	remaining := uint64(limits.WireBytes - len(request.wire))
	if !hasDeadline || conn.SetDeadline(deadline) != nil {
		return nil, errProviderIntegration
	}
	wire := providerReplayPhaseBytes(request.wire, meter)
	if len(wire) != len(request.wire) {
		return nil, errProviderIntegration
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	started := time.Now()
	if n, err := conn.Write(wire); err != nil || n != len(wire) {
		meter.refuse()
		return nil, errProviderIntegration
	}
	limited := &io.LimitedReader{R: conn, N: int64(remaining) + 1}
	var raw bytes.Buffer
	reader := bufio.NewReaderSize(io.TeeReader(limited, &raw), min(limits.HeaderBytes+1, 4096))
	head, headers, body, result, readErr := readProviderHTTP(reader, limits)
	consumed := int(remaining) + 1 - int(limited.N)
	if readErr != nil || ctx.Err() != nil || consumed <= 0 || consumed > int(remaining) {
		meter.refuse()
		return nil, errProviderIntegration
	}
	var work candidateWorkVector
	work[1] = uint64(len(wire) + consumed)
	if !meter.add(work) || !meter.event(providerCodecEvent, uint64(consumed)) || !meter.event(providerReadEvent, uint64(consumed)) {
		return nil, errProviderIntegration
	}
	requestHead := providerReplayPhaseBytes(request.head, meter)
	requestBody := providerReplayPhaseBytes(request.body, meter)
	if meter.failed || len(requestHead) != len(request.head) || len(requestBody) != len(request.body) {
		return nil, errProviderIntegration
	}
	complete = true
	return &ProviderTransportSample{Endpoint: SepoliaRPCURL, Method: request.method, HTTPVersion: "HTTP/1.1", RequestHead: requestHead, RequestBody: requestBody, ResponseHead: head, ResponseBody: body, ResponseWire: raw.Bytes(), Result: result, Headers: headers, TLSVersion: state.Version, CipherSuite: state.CipherSuite, ServerName: state.ServerName, VerifiedChains: chains, WireBytes: len(wire) + consumed, Started: started, Finished: time.Now(), Complete: true}, nil
}

// Invalid TLS states are finite unit DATA. A zero tls.Conn is not a usable Go
// connection and must never be manufactured as a runtime refusal scenario.
// Real capture always obtains this state from its normal verified connection.
func providerFirstOperationTLSIdentity(state tls.ConnectionState, certificates providerTLSBounds, meter *providerWorkMeter) ([][]string, error) {
	complete := false
	defer func() {
		if !complete {
			meter.refuse()
		}
	}()
	if meter == nil || meter.closed || meter.failed || !state.HandshakeComplete || state.Version < tls.VersionTLS12 || state.ServerName != "rpc.sepolia.ethpandaops.io" || state.NegotiatedProtocol != "" && state.NegotiatedProtocol != "http/1.1" || len(state.VerifiedChains) == 0 || !providerIntegrationCertificates(state, certificates, meter) {
		return nil, errProviderIntegration
	}
	// Identity construction hashes each bounded original certificate and
	// encodes its 32-byte digest. Charge both separately from the TLS event.
	for _, chain := range state.VerifiedChains {
		for _, certificate := range chain {
			if !meter.event(providerHashEvent, uint64(len(certificate.Raw))) || !meter.event(providerEncodeEvent, 32) || !meter.event(providerCopyEvent, 66) {
				return nil, errProviderIntegration
			}
		}
	}
	chains, err := providerTLSIdentity(state)
	if err != nil {
		return nil, errProviderIntegration
	}
	complete = true
	return chains, nil
}
