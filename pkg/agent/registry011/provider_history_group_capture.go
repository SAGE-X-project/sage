package registry011

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"time"
)

// Byte-engine DATA, never permission, a retained member, S or a protected
// result. The complete owner must bind the actual connection to the original
// attempt BEFORE dial, then independently retain/replay these bytes before
// advancing order.complete. No production constructor accepts a connection.
type providerHistoryCaptureData struct {
	sample                  *ProviderTransportSample
	requestWritten          int
	responseRead            int
	knownFailure, uncertain bool
	failureStage            string
}

// Closed BYTE primitive envelope only. It covers one original request copy,
// two request-part copies and comparisons, one bounded TLS identity, one wire
// read (including a rejected extra byte) and one aggregate HTTP/RPC codec.
// It is not the full T/C/S call table, a CPU/RSS bound or a production profile.
func providerHistoryCaptureUpper(requestBytes uint64, limits ProviderCaptureLimits, certificates providerTLSBounds) (candidateWorkVector, bool) {
	var work candidateWorkVector
	if !limits.valid() || !certificates.valid() || requestBytes == 0 || requestBytes > 16<<10 || requestBytes >= uint64(limits.WireBytes) {
		return work, false
	}
	response := uint64(limits.WireBytes) - requestBytes
	codec, a := providerUpperProduct(response, response)
	codec, b := providerUpperSum(codec, 1)
	request, c := providerUpperProduct(requestBytes, 4)
	// Request copy events sum to 2*r+3; the two-sided comparisons
	// sum to 2*r+2; read
	// contributes response+2, including its rejected-operation sentinel.
	linear, d := providerUpperSum(request, response, 7)
	count, e := providerUpperProduct(uint64(certificates.chains), uint64(certificates.certificates))
	der, f := providerUpperProduct(uint64(certificates.certificateBytes), 2)
	perCertificate, g := providerUpperSum(der, 2, 32*32+1, 67)
	tls, h := providerUpperProduct(count, perCertificate)
	units, i := providerUpperSum(codec, linear, tls)
	wire, j := providerUpperSum(uint64(limits.WireBytes), 1)
	if !a || !b || !c || !d || !e || !f || !g || !h || !i || !j {
		return work, false
	}
	work[1], work[2], work[10] = wire, response, units
	return work, true
}

// Private byte primitive. Its caller serializes the original meter and has
// already irreversibly consumed the concrete dial/attempt. This function does
// not dial, select an endpoint, issue an attempt, sample a qualified clock or
// make S possible. Normal owned TLS fixtures may exercise only this primitive.
// Known I/O/framing failures keep their actual prefix and healthy observer;
// budget/TLS provenance uncertainty poisons it and cannot be repaired here.
func providerHistoryCaptureBytes(ctx context.Context, conn *tls.Conn, limits ProviderCaptureLimits, request *providerIntegrationRequest, meter *providerWorkMeter, certificates providerTLSBounds) (*providerHistoryCaptureData, error) {
	if ctx == nil || conn == nil || request == nil || meter == nil || !limits.valid() || !certificates.valid() ||
		len(request.wire) == 0 || len(request.wire) > 16<<10 || len(request.head) == 0 || len(request.head) > 512 ||
		len(request.body) == 0 || len(request.body) > 16<<10 || len(request.wire) != len(request.head)+len(request.body) ||
		len(request.wire) >= limits.WireBytes {
		return nil, errProviderIntegration
	}
	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline || ctx.Err() != nil {
		return nil, errProviderIntegration
	}
	upper, ok := providerHistoryCaptureUpper(uint64(len(request.wire)), limits, certificates)
	if !ok || !meter.preflightWork(upper) ||
		!providerReplayTextEqual(request.wire[:len(request.head)], request.head, meter) ||
		!providerReplayTextEqual(request.wire[len(request.head):], request.body, meter) {
		return nil, errProviderIntegration
	}
	state := conn.ConnectionState()
	chains, err := providerFirstOperationTLSIdentity(state, certificates, meter)
	if err != nil {
		return nil, errProviderIntegration
	}
	wire := providerReplayBytes(request.wire, meter)
	head, body := providerReplayBytes(request.head, meter), providerReplayBytes(request.body, meter)
	if len(wire) != len(request.wire) || len(head) != len(request.head) || len(body) != len(request.body) || conn.SetDeadline(deadline) != nil {
		meter.refuse()
		return nil, errProviderIntegration
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	started := time.Now() // Diagnostic only, never an original clock sample.
	sample := &ProviderTransportSample{Endpoint: SepoliaRPCURL, Method: request.method, HTTPVersion: "HTTP/1.1", RequestHead: head, RequestBody: body,
		TLSVersion: state.Version, CipherSuite: state.CipherSuite, ServerName: state.ServerName, VerifiedChains: chains, Started: started}
	data := &providerHistoryCaptureData{sample: sample}
	written, writeErr := conn.Write(wire)
	data.requestWritten, sample.WireBytes = written, written
	var sent candidateWorkVector
	sent[1] = uint64(written)
	if written < 0 || written > len(wire) || !meter.add(sent) {
		return data.refuse(meter)
	}
	if writeErr != nil || written != len(wire) {
		return data.failed("write")
	}
	remaining := limits.WireBytes - len(wire)
	// Read one whole bounded original before parsing, so even an ordinary
	// EOF/cancellation/invalid reply retains and charges its actual bytes.
	raw, readErr := io.ReadAll(io.LimitReader(conn, int64(remaining)+1))
	data.responseRead, sample.ResponseWire, sample.Finished = len(raw), raw, time.Now()
	sample.WireBytes += len(raw)
	var received candidateWorkVector
	received[1] = uint64(len(raw))
	if !meter.add(received) || !meter.event(providerReadEvent, uint64(len(raw))) {
		return data.refuse(meter)
	}
	if readErr != nil || ctx.Err() != nil || len(raw) == 0 || len(raw) > remaining {
		return data.failed("read")
	}
	// One aggregate codec includes HTTP framing and JSON-RPC validation.
	// It is charged BEFORE the parser and on its actual original input.
	if !meter.event(providerCodecEvent, uint64(len(raw))) {
		return data.refuse(meter)
	}
	responseHead, headers, responseBody, result, parseErr := readProviderHTTP(bufio.NewReader(bytes.NewReader(raw)), limits)
	if parseErr != nil || ctx.Err() != nil {
		return data.failed("framing")
	}
	sample.ResponseHead, sample.Headers, sample.ResponseBody, sample.Result, sample.Complete = responseHead, headers, responseBody, result, true
	return data, nil
}

func (d *providerHistoryCaptureData) failed(stage string) (*providerHistoryCaptureData, error) {
	d.failureStage = stage
	d.knownFailure = true
	d.sample.CaptureError = errProviderIntegration.Error()
	if d.sample.Finished.IsZero() {
		d.sample.Finished = time.Now()
	}
	return d, errProviderIntegration
}

func (d *providerHistoryCaptureData) refuse(meter *providerWorkMeter) (*providerHistoryCaptureData, error) {
	meter.refuse()
	d.uncertain = true
	d.sample.CaptureError = errProviderIntegration.Error()
	d.sample.Finished = time.Now()
	return d, errProviderIntegration
}
