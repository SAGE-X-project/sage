package registry011

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"testing"
)

func TestProviderReplayRPCEnvelopeCountsThreeActualJSONPasses(t *testing.T) {
	body := []byte(`{"jsonrpc":"2.0","id":1,"result":{"value":"ordinary"}}`)
	limits := RPCResponseLimits{4096, 16}
	plain, err := DecodeRPCResult(body, 1, limits)
	meter := providerReplayUnitMeter()
	got, observedErr := decodeRPCResultObserved(body, 1, limits, meter)
	if err != nil || observedErr != nil || !bytes.Equal(plain, got) || meter.actual[2] != uint64(2*len(body)+len(`"2.0"`)) {
		t.Fatal("RPC token/envelope/version passes changed data or were omitted", err, observedErr)
	}
	short := providerReplayUnitMeter()
	short.upper[2] = uint64(len(body))
	if result, err := decodeRPCResultObserved(body, 1, limits, short); err == nil || result != nil || !short.failed || short.actual[2] != uint64(len(body)) {
		t.Fatal("second JSON pass reused first-pass allowance")
	}
	zero := providerReplayUnitMeter()
	zero.upper = candidateWorkVector{}
	if result, err := decodeRPCResultObserved(body, 1, limits, zero); err == nil || result != nil || !zero.failed || zero.actual != (candidateWorkVector{}) {
		t.Fatal("unfunded RPC validation returned result bytes")
	}
}

func TestProviderReplayRequestBuilderPreservesEveryWirePart(t *testing.T) {
	params := []any{"finalized", false}
	plain, err := newProviderIntegrationRequest("eth_getBlockByNumber", params)
	meter := providerReplayUnitMeter()
	got, observedErr := newProviderIntegrationRequestObserved("eth_getBlockByNumber", params, meter)
	if err != nil || observedErr != nil || *got != *plain || !got.firstFinalizedObserved(meter) || meter.actual[2] != 0 || meter.actual[10] == 0 {
		t.Fatal("observed request changed exact parts or counted a nonexistent decode", err, observedErr)
	}
	zero := providerReplayUnitMeter()
	zero.upper = candidateWorkVector{}
	if request, err := newProviderIntegrationRequestObserved("eth_getBlockByNumber", params, zero); err == nil || request != nil || !zero.failed {
		t.Fatal("unfunded request produced immutable wire data")
	}
	zero = providerReplayUnitMeter()
	zero.upper = candidateWorkVector{}
	if plain.firstFinalizedObserved(zero) || !zero.failed {
		t.Fatal("unfunded first-request validation returned true")
	}
	copy := *plain
	copy.wire = "ordinary different bytes"
	if copy.firstFinalizedObserved(providerReplayUnitMeter()) {
		t.Fatal("wire mismatch ignored")
	}
}

func TestProviderReplayBase64CountsDecodeEncodeAndComparison(t *testing.T) {
	raw := []byte("ordinary inert data")
	encoded := base64.RawURLEncoding.EncodeToString(raw)
	meter := providerReplayUnitMeter()
	got, ok := providerCanonicalBase64Observed(encoded, 4096, meter)
	if !ok || !bytes.Equal(raw, got) || meter.actual[2] != uint64(len(encoded)) || meter.actual[10] != uint64(len(encoded)*len(encoded)+1+len(raw)*len(raw)+1+2*len(encoded)+1) {
		t.Fatal("base64 invocation accounting or exact data changed")
	}
	zero := providerReplayUnitMeter()
	zero.upper = candidateWorkVector{}
	if got, ok := providerCanonicalBase64Observed(encoded, 4096, zero); ok || got != nil || !zero.failed {
		t.Fatal("unfunded base64 decode ran")
	}
}

func TestProviderReplayHTTPReadsOrdinaryMemoryWireUnderOneObserver(t *testing.T) {
	body := []byte(`{"jsonrpc":"2.0","id":1,"result":"ordinary"}`)
	wire := []byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", len(body), body))
	limits := providerTestLimits()
	meter := providerReplayUnitMeter()
	head, headers, got, result, err := readProviderHTTPObserved(bufio.NewReader(bytes.NewReader(wire)), limits, meter, uint64(len(wire)))
	if err != nil || len(headers) != 3 || !bytes.Equal(got, body) || string(result) != `"ordinary"` || meter.actual[2] != uint64(len(head)+len(wire)+2*len(body)+len(`"2.0"`)) {
		t.Fatal("HTTP framing/header/nested RPC passes omitted or data changed", err)
	}
	for _, allowance := range []uint64{0, 2, 32} {
		short := providerReplayUnitMeter()
		short.upper[10] = allowance
		if _, _, _, _, err := readProviderHTTPObserved(bufio.NewReader(bytes.NewReader(wire)), limits, short, uint64(len(wire))); err == nil || !short.failed {
			t.Fatal("finite exhausted header work yielded a response", allowance)
		}
	}
	if _, _, _, _, err := readProviderHTTPObserved(bufio.NewReader(bytes.NewReader(wire)), limits, meter, 0); err == nil {
		t.Fatal("observed framing accepted unknown original input size")
	}
}

func TestProviderReplayObservedExchangeReadsNormalRetainedClosure(t *testing.T) {
	_, view, replay, settlement := providerResultReplayUnit(t)
	meter := providerReplayUnitMeter()
	replay.meter = meter
	view.activeMeter.Store(meter)
	defer view.activeMeter.Store(nil)
	// The fixture first retains one ordinary owned TLS exchange. This later
	// independent DATA parser does no network call or operating selection.
	if err := replay.checkExchange(view, settlement); err != nil || meter.failed || meter.actual[2] == 0 || meter.actual[10] == 0 {
		t.Fatal("observed exchange closure rejected normal originals", err)
	}
	zero := providerReplayUnitMeter()
	zero.upper = candidateWorkVector{}
	replay.meter = zero
	view.activeMeter.Store(zero)
	if err := replay.checkExchange(view, settlement); err == nil || !zero.failed {
		t.Fatal("unfunded exchange returned successful closure")
	}
}
