package registry011

import (
	"context"
	"crypto/tls"
	"testing"
)

// A finite counter subject plus a normal owned TLS fixture. This is not a
// selected store, operating principal, consumed ticket or execution permit.
func providerFirstCaptureMeter(t *testing.T, request *providerIntegrationRequest, limits ProviderCaptureLimits, certificates providerTLSBounds) (*providerWorkMeter, candidateWorkVector) {
	t.Helper()
	upper, ok := providerFirstOperationCaptureUpper(uint64(len(request.wire)), limits, certificates)
	if !ok {
		t.Fatal("closed live primitive upper")
	}
	c := &providerIntegration{store: &providerCustodyStore{config: providerCustodyConfig{bounds: providerIntegrationBounds{firstAdmission: &providerFirstAdmissionBounds{}}}}}
	meter := newProviderWorkMeter(c, providerOperationReservation{execution: upper})
	c.store.activeMeter.Store(meter)
	return meter, upper
}

func TestProviderFirstOperationCaptureMeasuresActualWireWithoutSpendingItsUpper(t *testing.T) {
	request, err := newProviderIntegrationRequest("eth_getBlockByNumber", []any{"finalized", false})
	if err != nil {
		t.Fatal(err)
	}
	limits, certificates := providerTestLimits(), providerTLSBounds{1, 1, 8 << 10}
	meter, upper := providerFirstCaptureMeter(t, request, limits, certificates)
	conn := providerIntegrationOwnedTLS(t, request.body, `{"jsonrpc":"2.0","id":1,"result":"normal owned result"}`)
	state := conn.ConnectionState()
	ctx, cancel := context.WithTimeout(context.Background(), limits.Timeout)
	defer cancel()
	sample, err := providerFirstOperationCapture(ctx, conn, limits, request, meter, certificates)
	if err != nil || sample == nil || !sample.Complete || sample.CaptureError != "" || string(sample.RequestHead) != request.head || string(sample.RequestBody) != request.body || string(sample.Result) != `"normal owned result"` || sample.WireBytes != len(request.wire)+len(sample.ResponseWire) {
		t.Fatal("normal owned fixed HTTP/RPC primitive", err)
	}
	var certificateUnits uint64
	for _, chain := range state.VerifiedChains {
		for _, certificate := range chain {
			certificateUnits += 2*uint64(len(certificate.Raw)) + 2 + 32*32 + 1 + 67
		}
	}
	response, requested := uint64(len(sample.ResponseWire)), uint64(len(request.wire))
	units := response*response + 1 + response + 4*requested + 6 + certificateUnits
	if meter.actual[1] != uint64(sample.WireBytes) || meter.actual[2] != response || meter.actual[10] != units || meter.actual == upper || meter.actual[0] != 0 || meter.actual[8] != 0 || meter.actual[9] != 0 || meter.failed {
		t.Fatal("upper substituted for actual or the closed codec omitted work", meter.actual, units)
	}
	for i, n := range meter.actual {
		if n > upper[i] {
			t.Fatal("closed primitive exceeded its derived upper", i)
		}
	}
}

func TestProviderFirstOperationCaptureUpperRefusesUnsupportedFiniteShapes(t *testing.T) {
	for _, mode := range []string{"limits", "certificates", "empty", "request-cap", "wire"} {
		t.Run(mode, func(t *testing.T) {
			limits, certificates, request := providerTestLimits(), providerTLSBounds{1, 1, 8 << 10}, uint64(200)
			switch mode {
			case "limits":
				limits.Timeout = 0
			case "certificates":
				certificates.chains = 0
			case "empty":
				request = 0
			case "request-cap":
				request = 16<<10 + 1
			case "wire":
				request = uint64(limits.WireBytes)
			}
			if upper, ok := providerFirstOperationCaptureUpper(request, limits, certificates); ok || upper != (candidateWorkVector{}) {
				t.Fatal("unsupported closed primitive upper", mode)
			}
		})
	}
}

func TestProviderFirstOperationCaptureRefusesBeforeNormalPeerUse(t *testing.T) {
	for _, mode := range []string{"context", "cancelled", "connection", "request", "meter", "store", "revision", "detached", "scan", "remaining", "closed", "failed", "limits", "certificates", "request-bytes", "wire-upper", "decoded-upper", "work-upper", "receive-limit", "certificate-size", "deadline", "closed-connection"} {
		t.Run(mode, func(t *testing.T) {
			request, err := newProviderIntegrationRequest("eth_getBlockByNumber", []any{"finalized", false})
			if err != nil {
				t.Fatal(err)
			}
			limits, certificates := providerTestLimits(), providerTLSBounds{1, 1, 8 << 10}
			meter, upper := providerFirstCaptureMeter(t, request, limits, certificates)
			conn := providerIntegrationOwnedTLS(t, request.body, `{"jsonrpc":"2.0","id":1,"result":"normal"}`)
			ctx, cancel := context.WithTimeout(context.Background(), limits.Timeout)
			defer cancel()
			switch mode {
			case "context":
				ctx = nil
			case "cancelled":
				cancel()
			case "connection":
				conn = nil
			case "request":
				request = nil
			case "meter":
				meter = nil
			case "store":
				meter.integration.store = nil
			case "revision":
				meter.integration.store.config.bounds.firstAdmission = nil
			case "detached":
				meter.integration.store.activeMeter.Store(nil)
			case "scan":
				meter.scan = &providerScanControl{}
			case "remaining":
				meter.authority = true
			case "closed":
				meter.closed = true
			case "failed":
				meter.failed = true
			case "limits":
				limits.Timeout = 0
			case "certificates":
				certificates.chains = 0
			case "request-bytes":
				copy := *request
				copy.wire = "X" + copy.wire[1:]
				request = &copy
			case "wire-upper":
				meter.upper[1] = upper[1] - 1
			case "decoded-upper":
				meter.upper[2] = upper[2] - 1
			case "work-upper":
				meter.upper[10] = upper[10] - 1
			case "receive-limit":
				// The peer remains normal; this receiver chose an insufficient
				// allowance and must preserve failure rather than partial DATA.
				limits.HeaderBytes, limits.BodyBytes, limits.WireBytes = 1, 1, len(request.wire)+1
			case "certificate-size":
				certificates.certificateBytes = 1
			case "deadline":
				ctx = context.Background()
			case "closed-connection":
				_ = conn.Close()
			}
			if sample, err := providerFirstOperationCapture(ctx, conn, limits, request, meter, certificates); err == nil || sample != nil {
				t.Fatal("invalid or unfunded primitive reached a complete exchange", mode)
			}
			if meter != nil && !meter.failed {
				t.Fatal("failed capture left a reusable observer", mode)
			}
		})
	}
}

func TestProviderFirstOperationTLSIdentityRefusesFiniteUnverifiedStateOrQuota(t *testing.T) {
	request, err := newProviderIntegrationRequest("eth_getBlockByNumber", []any{"finalized", false})
	if err != nil {
		t.Fatal(err)
	}
	conn := providerIntegrationOwnedTLS(t, request.body, `{"jsonrpc":"2.0","id":1,"result":"normal"}`)
	for _, mode := range []string{"unverified", "empty-chain", "nil-certificate", "empty-der", "hash-quota", "encode-quota", "copy-quota"} {
		t.Run(mode, func(t *testing.T) {
			certificates := providerTLSBounds{1, 1, 8 << 10}
			meter, _ := providerFirstCaptureMeter(t, request, providerTestLimits(), certificates)
			state := conn.ConnectionState()
			der := uint64(len(state.VerifiedChains[0][0].Raw))
			switch mode {
			case "unverified":
				state = tls.ConnectionState{}
			case "empty-chain":
				state.VerifiedChains = state.VerifiedChains[:1:1]
				state.VerifiedChains = append(state.VerifiedChains[:0:0], nil)
			case "nil-certificate":
				state.VerifiedChains = append(state.VerifiedChains[:0:0], append(state.VerifiedChains[0][:0:0], nil))
			case "empty-der":
				certificate := *state.VerifiedChains[0][0]
				certificate.Raw = nil
				state.VerifiedChains = append(state.VerifiedChains[:0:0], append(state.VerifiedChains[0][:0:0], &certificate))
			case "hash-quota":
				meter.upper[10] = der + 1
			case "encode-quota":
				meter.upper[10] = 2*der + 2 + 32*32
			case "copy-quota":
				meter.upper[10] = 2*der + 2 + 32*32 + 1 + 66
			}
			if identity, err := providerFirstOperationTLSIdentity(state, certificates, meter); err == nil || identity != nil || !meter.failed {
				t.Fatal("finite invalid TLS DATA or incomplete measurement returned an identity", mode)
			}
		})
	}
}
