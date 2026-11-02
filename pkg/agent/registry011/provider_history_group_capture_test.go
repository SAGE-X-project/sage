package registry011

import (
	"context"
	"crypto/tls"
	"testing"
)

// Arithmetic observer for the private BYTE primitive only. It owns no store,
// T/C/attempt/order or origin selection and cannot make a retained S/Source.
func historyCaptureUnitMeter(limits ProviderCaptureLimits) *providerWorkMeter {
	m := providerReplayUnitMeter()
	m.upper[1] = uint64(limits.WireBytes) + 1
	return m
}

func TestProviderHistoryCaptureUpperIncludesTheRejectedWireSentinel(t *testing.T) {
	request := historyGroupSimple(t)
	_, limits := historyGroupBounds()
	bounds := providerTLSBounds{2, 4, 8 << 10}
	u, ok := providerHistoryCaptureUpper(uint64(len(request.wire)), limits, bounds)
	if !ok || u[0] != 0 || u[1] != uint64(limits.WireBytes)+1 || u[2] != uint64(limits.WireBytes-len(request.wire)) || u[9] != 0 || u[10] == 0 {
		t.Fatal("primitive quote omitted rejected bytes or invented an attempt/storage")
	}
	for _, mode := range []string{"empty-request", "large-request", "request-at-wire", "bad-limits", "bad-certificates"} {
		t.Run(mode, func(t *testing.T) {
			n, l, c := uint64(len(request.wire)), limits, bounds
			switch mode {
			case "empty-request":
				n = 0
			case "large-request":
				n = 16<<10 + 1
			case "request-at-wire":
				n = uint64(l.WireBytes)
			case "bad-limits":
				l.Timeout = 0
			case "bad-certificates":
				c.chains = 0
			}
			if got, valid := providerHistoryCaptureUpper(n, l, c); valid || got != (candidateWorkVector{}) {
				t.Fatal("invalid primitive quote")
			}
		})
	}
}

func TestProviderHistoryCaptureBytesRetainsANormalOwnedExchange(t *testing.T) {
	request := historyGroupSimple(t)
	_, limits := historyGroupBounds()
	bounds := providerTLSBounds{2, 4, 8 << 10}
	m := historyCaptureUnitMeter(limits)
	response := `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`
	conn := providerIntegrationOwnedTLS(t, request.body, response)
	ctx, cancel := context.WithTimeout(context.Background(), limits.Timeout)
	defer cancel()
	data, err := providerHistoryCaptureBytes(ctx, conn, limits, request, m, bounds)
	upper, ok := providerHistoryCaptureUpper(uint64(len(request.wire)), limits, bounds)
	if err != nil || data == nil || data.sample == nil || !data.sample.Complete || data.knownFailure || data.uncertain ||
		data.requestWritten != len(request.wire) || data.responseRead != len(data.sample.ResponseWire) ||
		string(data.sample.RequestHead) != request.head || string(data.sample.RequestBody) != request.body ||
		string(data.sample.ResponseBody) != response || string(data.sample.Result) != `"0xaa36a7"` ||
		m.actual[1] != uint64(len(request.wire)+len(data.sample.ResponseWire)) || m.actual[2] != uint64(len(data.sample.ResponseWire)) ||
		m.actual[0] != 0 || m.failed || m.closed || !ok {
		t.Fatal("normal byte primitive changed originals, poisoned an observer or invented authority", err)
	}
	for i, actual := range m.actual {
		if actual > upper[i] {
			t.Fatal("actual closed primitive exceeds its preflight", i, actual, upper[i])
		}
	}
}

func TestProviderHistoryCaptureRefusesBeforeWritingWithoutItsWholeEnvelope(t *testing.T) {
	request := historyGroupSimple(t)
	_, limits := historyGroupBounds()
	bounds := providerTLSBounds{2, 4, 8 << 10}
	// The peer is an ordinary owned server. Refused scenarios never write
	// malformed data or exercise a hostile responder at runtime.
	conn := providerIntegrationOwnedTLS(t, request.body, `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`)
	for _, mode := range []string{"nil-context", "no-deadline", "cancelled", "nil-connection", "nil-request", "nil-meter", "inconsistent-request", "wire-one-short", "decoded-one-short", "units-one-short"} {
		t.Run(mode, func(t *testing.T) {
			m := historyCaptureUnitMeter(limits)
			ctx, cancel := context.WithTimeout(context.Background(), limits.Timeout)
			defer cancel()
			input, connection, observer := request, conn, m
			upper, ok := providerHistoryCaptureUpper(uint64(len(request.wire)), limits, bounds)
			if !ok {
				t.Fatal("positive primitive quote")
			}
			switch mode {
			case "nil-context":
				ctx = nil
			case "no-deadline":
				ctx = context.Background()
			case "cancelled":
				cancel()
			case "nil-connection":
				connection = nil
			case "nil-request":
				input = nil
			case "nil-meter":
				observer = nil
			case "inconsistent-request":
				copy := *request
				copy.wire = "x" + copy.wire[1:]
				input = &copy
			case "wire-one-short":
				m.upper[1] = upper[1] - 1
			case "decoded-one-short":
				m.upper[2] = upper[2] - 1
			case "units-one-short":
				m.upper[10] = upper[10] - 1
			}
			if data, err := providerHistoryCaptureBytes(ctx, connection, limits, input, observer, bounds); err == nil || data != nil || m.actual[1] != 0 || m.actual[2] != 0 {
				t.Fatal("unfunded or detached byte input reached wire/codec")
			}
		})
	}
	// The same normal connection remains usable after every no-effect refusal.
	m := historyCaptureUnitMeter(limits)
	ctx, cancel := context.WithTimeout(context.Background(), limits.Timeout)
	defer cancel()
	if data, err := providerHistoryCaptureBytes(ctx, conn, limits, request, m, bounds); err != nil || data == nil || !data.sample.Complete {
		t.Fatal("refused inputs had a hidden connection effect", err)
	}
}

func TestProviderHistoryCaptureKnownAndUncertainFailureDataStaySeparate(t *testing.T) {
	// Finite unit DATA only: no socket, invalid TLS peer, hostile payload or
	// attack reproduction. Connected partial-I/O and S tests remain required.
	known := &providerHistoryCaptureData{sample: &ProviderTransportSample{}, requestWritten: 7, responseRead: 11}
	if got, err := known.failed("read"); got != known || err == nil || !got.knownFailure || got.failureStage != "read" || got.uncertain || got.sample.Complete || got.requestWritten != 7 || got.responseRead != 11 || got.sample.CaptureError == "" {
		t.Fatal("known prefix failure became success or lost its actuals")
	}
	m := providerReplayUnitMeter()
	uncertain := &providerHistoryCaptureData{sample: &ProviderTransportSample{}, requestWritten: 13}
	if got, err := uncertain.refuse(m); got != uncertain || err == nil || got.knownFailure || !got.uncertain || !m.failed || got.sample.Complete || got.requestWritten != 13 {
		t.Fatal("observer uncertainty became a reusable known failure")
	}
	// Shape-only invalid TLS state is unit DATA, never a zero tls.Conn.
	if chains, err := providerFirstOperationTLSIdentity(tls.ConnectionState{}, providerTLSBounds{2, 4, 8 << 10}, providerReplayUnitMeter()); err == nil || chains != nil {
		t.Fatal("unverified TLS DATA supplied connection identity")
	}
}
