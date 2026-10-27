package registry011

import (
	"testing"
	"time"
)

func providerFrontierBoundsFixture() (providerIntegrationBounds, ProviderCaptureLimits) {
	ledger := providerLedgerBoundsFixture()
	methods := providerMethodBounds{scalar: providerMethodLimit{1024, 16 << 10}, header: providerMethodLimit{16 << 10, 32 << 10}, code: providerMethodLimit{64 << 10, 128 << 10}, state: providerMethodLimit{64 << 10, 128 << 10}, block: providerMethodLimit{64 << 10, 128 << 10}, receipts: providerMethodLimit{64 << 10, 128 << 10}, gas: providerMethodLimit{16 << 10, 64 << 10}, execution: providerMethodLimit{128 << 10, 256 << 10}}
	tls := providerTLSBounds{1, 4, 8 << 10}
	store := providerIntegrationBounds{ledger: CandidateLedgerBounds{4096, 16 << 20, 32 << 20}, continuation: &ledger, methods: &methods, tls: &tls, maximumCommits: int(ledger.maximumCommits), maximumObjects: int(ledger.maximumObjects), maximumEntries: 256, maximumObjectBytes: 256 << 10, maximumStoredBytes: 64 << 20, maximumMetadataPerCommit: 16 << 10, fileOverheadBytes: 4096, maximumSetupRecoveries: 3, setupWork: candidateWorkVector{0, 0, 1 << 20, 0, 0, 0, 0, 0, 0, 1 << 20, 1 << 60}, setupLive: 2 << 20, setupRetained: 1 << 20, setupMaximumClockReads: 65536, setupMaximumDurationMS: 300000}
	return store, ProviderCaptureLimits{Timeout: 2 * time.Second, HeaderBytes: 8 << 10, BodyBytes: 128 << 10, WireBytes: 256 << 10, JSONDepth: 64}
}

func TestProviderMethodBoundsUseClosedClassesWithoutFallback(t *testing.T) {
	store, global := providerFrontierBoundsFixture()
	if !store.valid() || !store.methods.fits(global) {
		t.Fatal("explicit inert profile")
	}
	hash := admissionTestDigest("a")
	for _, item := range []struct {
		method   string
		params   []any
		expected providerMethodLimit
	}{
		{"eth_chainId", []any{}, store.methods.scalar}, {"web3_clientVersion", []any{}, store.methods.scalar},
		{"eth_getBlockByNumber", []any{"finalized", false}, store.methods.header}, {"eth_getBlockByHash", []any{hash, true}, store.methods.block},
		{"eth_getCode", []any{SepoliaRegistryAddress, "finalized"}, store.methods.code}, {"eth_call", []any{map[string]any{"to": SepoliaRegistryAddress, "data": "0x"}, "finalized"}, store.methods.state},
		{"debug_getRawBlock", []any{hash}, store.methods.block}, {"eth_getBlockReceipts", []any{hash}, store.methods.receipts}, {"debug_getRawReceipts", []any{hash}, store.methods.receipts},
		{"debug_traceBlockByHash", []any{hash, map[string]any{"tracer": "stateGasTracer", "reexec": 0}}, store.methods.gas},
		{"debug_traceTransaction", []any{hash, map[string]any{"tracer": "callTracer", "reexec": 0}}, store.methods.execution},
	} {
		t.Run(item.method, func(t *testing.T) {
			request, err := newProviderIntegrationRequest(item.method, item.params)
			if err != nil {
				t.Fatal(err)
			}
			limits, err := store.methods.capture(request, global)
			if err != nil || limits.BodyBytes != item.expected.body || limits.WireBytes != item.expected.wire || limits.Timeout != global.Timeout || limits.HeaderBytes != global.HeaderBytes {
				t.Fatal("method changed its fixed bounds", err)
			}
		})
	}
}

func TestProviderMethodBoundsRefuseUnfundedClassesAndInputs(t *testing.T) {
	store, global := providerFrontierBoundsFixture()
	for _, name := range []string{"zero", "large-body", "large-wire", "body-wire", "above-global", "small-wire"} {
		t.Run(name, func(t *testing.T) {
			copy := *store.methods
			switch name {
			case "zero":
				copy.scalar.body = 0
			case "large-body":
				copy.scalar.body = 8<<20 + 1
			case "large-wire":
				copy.scalar.wire = 16<<20 + 1
			case "body-wire":
				copy.scalar.body = copy.scalar.wire
			case "above-global":
				copy.scalar.body = global.BodyBytes + 1
				copy.scalar.wire = global.WireBytes
			case "small-wire":
				copy.scalar.wire = global.HeaderBytes
			}
			if copy.fits(global) {
				t.Fatal("unfunded profile")
			}
		})
	}
	for _, request := range []*providerIntegrationRequest{nil, {method: "unknown"}, {method: "eth_getBlockByNumber", params: `["finalized"]`}, {method: "eth_getBlockByHash", params: `["x","true"]`}, {method: "debug_traceTransaction", params: `["x",{}]`}, {method: "debug_traceBlockByHash", params: `["x",null]`}} {
		if _, err := store.methods.capture(request, global); err == nil {
			t.Fatal("unsupported request selected another class")
		}
	}
	for _, limits := range []providerTLSBounds{{}, {9, 1, 1}, {1, 17, 1}, {1, 1, 64<<10 + 1}} {
		if _, ok := limits.upper(); ok {
			t.Fatal("unbounded TLS identity")
		}
	}
}

func TestProviderFrontierReservationFundsExecutionAndRemainingSeparately(t *testing.T) {
	store, capture := providerFrontierBoundsFixture()
	primitive := providerPrimitiveBounds{capture: capture, blocks: 4, transactions: 16, frames: 64, logs: 64, proofs: 16, maximumCreatedObjects: 32}
	for _, rpc := range []bool{false, true} {
		bytes := uint64(0)
		if rpc {
			bytes = 512
		}
		reservation, err := providerEstimateFrontierOperation(store, primitive, bytes, rpc, true)
		if err != nil {
			t.Fatal(err)
		}
		for i := range reservation.work {
			if reservation.work[i] != reservation.execution[i]+reservation.remaining[i] {
				t.Fatal("remaining authority work was omitted", i)
			}
		}
		if reservation.live != reservation.retained || reservation.live == 0 || reservation.execution[8] != 1 || rpc && (reservation.execution[0] != 1 || reservation.execution[3] != 0) || !rpc && (reservation.execution[0] != 0 || reservation.execution[3] != primitive.blocks) {
			t.Fatal("wrong reservation scope")
		}
	}
}
