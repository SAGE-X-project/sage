package registry011

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

// These are inert grammar DATA scenarios, not reconstructed later S facts.
// Only the separate genuine first-P test below proves original reader seeding.
// No returned candidate table is authority for the internal collection entry.
func historyCollectionPlanFixture(t *testing.T, blocks, window, content uint64) (*providerHistoryCollectionCursor, providerIntegrationBounds, ProviderCaptureLimits) {
	t.Helper()
	return historyCollectionPlanTimeFixture(t, blocks, window, content, 0)
}

func historyCollectionPlanTimeFixture(t *testing.T, blocks, window, content, timestamp uint64) (*providerHistoryCollectionCursor, providerIntegrationBounds, ProviderCaptureLimits) {
	t.Helper()
	in := historyCollectionInput(t)
	in.blocks, in.maximumSelected, in.windowBlocks, in.contentBlocks = blocks, sepoliaDeploymentBlock+blocks-1, window, content
	in.native, in.effects, in.versions, in.presentStops = blocks, 64, 64, 0
	selected := historyFirstFinalizedHeader()
	selected.Number = in.maximumSelected
	if timestamp != 0 {
		selected.Timestamp = timestamp
	}
	if blocks == 1 {
		selected = in.journal.Deployment
	}
	m := historyCollectionManifest(t, in)
	c, err := newProviderHistoryCollectionCursorObserved(admissionTestDigest("a"), m, selected, providerReplayUnitMeter())
	if err != nil || c == nil || !c.valid() {
		t.Fatal("inert explicit plan fixture", err)
	}
	b, capture := providerFrontierBoundsFixture()
	capture.Timeout = 10 * time.Second
	return c, b, capture
}

func historyCollectionPlan(t *testing.T, c *providerHistoryCollectionCursor, b providerIntegrationBounds, capture ProviderCaptureLimits) *providerHistoryGroupTable {
	t.Helper()
	table, err := c.nextTableObserved(1, *b.methods, capture, providerReplayUnitMeter())
	if err != nil || table == nil {
		t.Fatal("complete inert request table", err)
	}
	return table
}

func TestProviderHistoryCollectionPlanStartsWithExactWindowIdentityAndPinnedCodes(t *testing.T) {
	for _, count := range []uint64{1, 13} {
		t.Run(map[uint64]string{1: "deployment-only", 13: "ancestry-first"}[count], func(t *testing.T) {
			c, b, capture := historyCollectionPlanFixture(t, count, 4, 2)
			table := historyCollectionPlan(t, c, b, capture)
			want := []string{"eth_chainId", "eth_getBlockByNumber", "eth_call", "web3_clientVersion", "eth_getBlockByNumber"}
			if table.count != 18 || table.purpose != "identity" || c.phase != "open" || c.native != 0 || c.traces != 0 {
				t.Fatal("window start omitted a code read or advanced unacknowledged DATA")
			}
			for i, method := range want {
				if table.members[i].request.method != method {
					t.Fatal("window identity order", i)
				}
			}
			if table.members[1].request.params != `["0x0",false]` || table.members[4].request.params != `["finalized",false]` {
				t.Fatal("window identity selected different header requests")
			}
			for i, code := range c.selection.input.codes {
				var params []json.RawMessage
				var address string
				var block struct {
					Hash      string `json:"blockHash"`
					Canonical bool   `json:"requireCanonical"`
				}
				request := table.members[5+i].request
				if request.method != "eth_getCode" || table.members[5+i].class != "code" || json.Unmarshal([]byte(request.params), &params) != nil || len(params) != 2 ||
					json.Unmarshal(params[0], &address) != nil || json.Unmarshal(params[1], &block) != nil || address != code.Address || block.Hash != c.selection.selected.Hash || !block.Canonical {
					t.Fatal("code request moved from its original context/selected hash", i)
				}
			}
			historyCheckPlannedRegistryCall(t, table.members[2].request, "registryId", "", 0, c.selection.selected.Hash)
			original := table.digest
			table.members[0].request.body = "inert returned DATA"
			if fresh := historyCollectionPlan(t, c, b, capture); fresh == table || fresh.digest != original {
				t.Fatal("returned table mutated the source cursor")
			}
		})
	}
}

func TestProviderHistoryCollectionPlanBoundsDescendingGroupsAndWindowStop(t *testing.T) {
	for _, mode := range []string{"short", "maximum"} {
		t.Run(mode, func(t *testing.T) {
			blocks, window, count, first, last := uint64(13), uint64(4), 4, `["0xb6b819",false]`, `["0xb6b816",false]`
			if mode == "maximum" {
				blocks, window, count, first, last = 71, 64, 32, `["0xb6b853",false]`, `["0xb6b834",false]`
			}
			c, b, capture := historyCollectionPlanFixture(t, blocks, window, 2)
			c.phase = "ancestry" // Inert grammar scenario; no live owner is changed.
			table := historyCollectionPlan(t, c, b, capture)
			if table.purpose != "ancestry" || table.count != count || table.members[0].request.params != first || table.members[count-1].request.params != last {
				t.Fatal("descending numeric group lost its exact finite boundary")
			}
			c.phase = "close"
			end := historyCollectionPlan(t, c, b, capture)
			if end.count != 7 || end.members[2].request.params != `["finalized",false]` || end.members[4].request.params != `["0x0",false]` {
				t.Fatal("window end omitted an independent network observation")
			}
			historyCheckPlannedRegistryCall(t, end.members[5].request, "registryId", "", 0, c.selection.selected.Hash)
			if mode == "short" && (end.members[0].request.params != `["0xb6b816",false]` || end.members[1].request.params != `["0xb6b81a",false]`) {
				t.Fatal("descending window rechecked its high header instead of its low stop")
			}
		})
	}
}

func TestProviderHistoryCollectionPlanUsesCompleteContentSetsAndExplicitGasChoice(t *testing.T) {
	for _, mode := range []string{"raw-before-fork", "json-native", "raw-state-gas", "partial-tail"} {
		t.Run(mode, func(t *testing.T) {
			stamp := sepoliaAmsterdamTime + 100
			if mode == "raw-before-fork" {
				stamp = sepoliaAmsterdamTime - 1
			}
			c, b, capture := historyCollectionPlanTimeFixture(t, 13, 10, 10, stamp)
			c.mode, c.phase, c.low, c.high, c.next = "content", "content", sepoliaDeploymentBlock, sepoliaDeploymentBlock+9, sepoliaDeploymentBlock
			c.spine = make([]FinalizedHeader, 13)
			for i := range c.spine {
				c.spine[i] = FinalizedHeader{c.selection.selected.Hash, c.selection.selected.ParentHash, c.selection.selected.Number - uint64(i), stamp}
			}
			// The cursor and semantic fold received the same selected timestamp
			// at construction; do not patch a sealed selection after that point.
			if mode == "raw-before-fork" || mode == "raw-state-gas" {
				c.selection.input.encoding = "debug-raw-rlp-v1"
			}
			if mode == "raw-state-gas" {
				c.selection.input.gas = SepoliaAmsterdamGasProfile
			}
			c.sealed = c.selection
			blocks, members := 10, 30
			if mode == "raw-before-fork" {
				members = 20
			}
			if mode == "partial-tail" {
				c.next = c.high
				blocks, members = 1, 3
			}
			table := historyCollectionPlan(t, c, b, capture)
			if table.count != members || table.purpose != "content" || c.native != 0 || c.traces != 0 {
				t.Fatal("partial content set or unacknowledged charge")
			}
			stride := members / blocks
			for i := range blocks {
				block, receipt := table.members[i*stride].request, table.members[i*stride+1].request
				wantBlock, wantReceipt := "eth_getBlockByHash", "eth_getBlockReceipts"
				if c.selection.input.encoding == "debug-raw-rlp-v1" {
					wantBlock, wantReceipt = "debug_getRawBlock", "debug_getRawReceipts"
				}
				if block.method != wantBlock || receipt.method != wantReceipt {
					t.Fatal("fixed content encoding changed", i)
				}
				if stride == 3 {
					request := table.members[i*stride+2].request
					var params []json.RawMessage
					var options struct {
						Tracer  string          `json:"tracer"`
						Timeout string          `json:"timeout"`
						Config  map[string]bool `json:"tracerConfig"`
					}
					if request.method != "debug_traceBlockByHash" || json.Unmarshal([]byte(request.params), &params) != nil || len(params) != 2 || json.Unmarshal(params[1], &options) != nil || options.Timeout != "1s" {
						t.Fatal("gas request lost the original explicit timeout")
					}
					if mode == "raw-state-gas" {
						if options.Tracer != "stateGasTracer" || options.Config != nil {
							t.Fatal("state-gas format gained a root-tracer fallback")
						}
					} else if options.Tracer != "callTracer" || !options.Config["onlyTopCall"] || options.Config["withLog"] || !options.Config["includePrecompiles"] {
						t.Fatal("root-gas format changed its exact tracer options")
					}
				}
			}
		})
	}
}

func TestProviderHistoryCollectionRegistryCalldataMatchesPinnedABI(t *testing.T) {
	contract, _, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{"record", "activeGrants", "historyItem"} {
		t.Run(method, func(t *testing.T) {
			version := uint64(0)
			if method == "historyItem" {
				version = 33
			}
			params, err := providerHistoryRegistryParamsObserved(SepoliaRegistryAddress, "Agent_1", method, version, admissionTestDigest("a"), providerReplayUnitMeter())
			if err != nil || len(params) != 2 {
				t.Fatal("closed ABI request", err)
			}
			call := params[0].(map[string]string)
			data, err := hex.DecodeString(strings.TrimPrefix(call["data"], "0x"))
			abiMethod := contract.Methods[method]
			if err != nil || len(data) < 4 || !bytes.Equal(data[:4], abiMethod.ID) || call["to"] != SepoliaRegistryAddress {
				t.Fatal("selector/address differs from pinned ABI")
			}
			args, err := abiMethod.Inputs.Unpack(data[4:])
			want := []any{"Agent_1"}
			if version != 0 {
				want = append(want, version)
			}
			block := params[1].(map[string]any)
			if err != nil || !reflect.DeepEqual(args, want) || block["blockHash"] != admissionTestDigest("a") || block["requireCanonical"] != true {
				t.Fatal("canonical ABI arguments or same-block requirement changed", args, err)
			}
		})
	}
}

func TestProviderHistoryCollectionPlanKeepsWholeEffectAndVersionBounds(t *testing.T) {
	c, b, capture := historyCollectionPlanFixture(t, 13, 4, 2)
	c.mode, c.phase = "content", "effects"
	c.high, c.low = c.selection.selected.Number, c.selection.selected.Number-3
	for i := range 33 {
		c.effects = append(c.effects, providerHistoryCollectionEffect{block: c.selection.selected.Hash, transaction: fmt.Sprintf("0x%064x", i+1), number: c.selection.selected.Number, index: uint64(i)})
	}
	first := historyCollectionPlan(t, c, b, capture)
	if first.purpose != "effects" || first.count != 32 || !strings.Contains(first.members[0].request.params, `"onlyTopCall":false`) || !strings.Contains(first.members[0].request.params, `"withLog":true`) {
		t.Fatal("full effect trace was replaced with top-only gas DATA")
	}
	c.effectNext, c.traces = 32, 32
	if tail := historyCollectionPlan(t, c, b, capture); tail.count != 1 {
		t.Fatal("partial effect group was dropped")
	}
	c.traces = 64
	if out, err := c.nextTableObserved(1, *b.methods, capture, providerReplayUnitMeter()); err == nil || out != nil {
		t.Fatal("smaller earlier trace results refunded the whole bound")
	}
	c.traces, c.phase, c.effects, c.effectNext = 0, "history", nil, 0
	c.high, c.low = c.selection.selected.Number, c.selection.selected.Number-3
	c.historyNext, c.stopVersion = 1, 33
	first = historyCollectionPlan(t, c, b, capture)
	if first.purpose != "history" || first.count != 32 {
		t.Fatal("bounded version table")
	}
	historyCheckPlannedRegistryCall(t, first.members[0].request, "historyItem", c.selection.input.journal.Agent, 1, c.selection.selected.Hash)
	historyCheckPlannedRegistryCall(t, first.members[31].request, "historyItem", c.selection.input.journal.Agent, 32, c.selection.selected.Hash)
	c.historyNext = 33
	tail := historyCollectionPlan(t, c, b, capture)
	if tail.count != 1 {
		t.Fatal("last exact version was dropped")
	}
	historyCheckPlannedRegistryCall(t, tail.members[0].request, "historyItem", c.selection.input.journal.Agent, 33, c.selection.selected.Hash)
	c.stopVersion = 65
	if out, err := c.nextTableObserved(1, *b.methods, capture, providerReplayUnitMeter()); err == nil || out != nil {
		t.Fatal("later window renewed the whole version maximum")
	}
}

func TestProviderHistoryCollectionPlansSeedEachGenuineOriginalReaderIndependently(t *testing.T) {
	m := historyCollectionManifest(t, historyCollectionInput(t))
	s := historyFirstFinalizedStore(t, historyFirstFinalizedResponse(t), m)
	o, err := newProviderHistoryGroupOwnerCandidate(s)
	if err != nil || o.coordinator.collection == nil || o.custodian.collection == nil || o.coordinator.collection == o.custodian.collection {
		t.Fatal("two genuine healthy originals did not seed independent cursors", err)
	}
	a := historyCollectionPlan(t, o.coordinator.collection, s.config.bounds, s.config.capture)
	b := historyCollectionPlan(t, o.custodian.collection, s.config.bounds, s.config.capture)
	if a.digest != b.digest || a.raw != b.raw {
		t.Fatal("independent healthy first tables differ")
	}
	r, refs := historyRecoveryGroupStart(t, o)
	for pass := range 2 {
		historyRecoveryModelPrefix(t, r, pass, refs)
		initial, err := r.replayInitialModelCandidate(context.Background(), pass)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := initial.selectFirstFinalizedCandidate(context.Background()); err != nil {
			t.Fatal(err)
		}
		model, err := r.replayGroupModelCandidate(context.Background(), pass)
		if err != nil || model.ledger.collection == nil {
			t.Fatal("independent terminal original selection", pass, err)
		}
		got := historyCollectionPlan(t, model.ledger.collection, s.config.bounds, s.config.capture)
		if got.digest != a.digest || got.raw != a.raw {
			t.Fatal("terminal reader borrowed or changed the healthy first table", pass)
		}
	}
}

func TestProviderHistoryCollectionPlanRejectsIncompleteFactsAndNeverReturnsEmptyGroups(t *testing.T) {
	for _, mode := range []string{"nil", "copy", "mode", "phase", "stage-zero", "capture", "nil-meter", "closed-meter", "quota", "missing-spine", "effects-empty", "effects-index", "history-zero", "history-regression", "history-bound", "native-bound"} {
		t.Run(mode, func(t *testing.T) {
			c, b, capture := historyCollectionPlanFixture(t, 13, 4, 2)
			stage, meter := uint64(1), providerReplayUnitMeter()
			switch mode {
			case "nil":
				c = nil
			case "copy":
				copied := *c
				c = &copied
			case "mode":
				c.mode = "caller-selected"
			case "phase":
				c.phase = "complete"
			case "stage-zero":
				stage = 0
			case "capture":
				capture.Timeout--
			case "nil-meter":
				meter = nil
			case "closed-meter":
				meter.closed = true
			case "quota":
				meter.upper = candidateWorkVector{}
			case "missing-spine":
				c.phase, c.mode, c.next = "content", "content", c.low
			case "effects-empty":
				c.phase, c.mode = "effects", "content"
			case "effects-index":
				c.phase, c.mode = "effects", "content"
				c.high, c.low = c.selection.selected.Number, c.selection.selected.Number-3
				c.effects = []providerHistoryCollectionEffect{{block: c.selection.selected.Hash, transaction: admissionTestDigest("b"), number: c.selection.selected.Number, index: 0}}
				if precursor := historyCollectionPlan(t, c, b, capture); precursor.count != 1 {
					t.Fatal("valid effect precursor missing")
				}
				c.effectNext = -1
			case "history-zero", "history-regression", "history-bound":
				c.phase, c.mode, c.high = "history", "content", c.selection.selected.Number
				c.low = c.high - 3
				c.historyNext, c.stopVersion = 0, 1
				if mode == "history-regression" {
					c.historyNext, c.stopVersion = 2, 1
				}
				if mode == "history-bound" {
					c.historyNext, c.stopVersion = 1, 65
				}
			case "native-bound":
				c.phase, c.mode = "content", "content"
				c.low, c.high, c.next = c.selection.selected.Number, c.selection.selected.Number, c.selection.selected.Number
				c.selection.selected.Timestamp = sepoliaAmsterdamTime
				c.selection.input.native = 0
				c.sealed, c.spine[0] = c.selection, c.selection.selected
			}
			if out, err := c.nextTableObserved(stage, *b.methods, capture, meter); err == nil || out != nil {
				t.Fatal("incomplete facts gained a shorter or empty next group", mode)
			}
		})
	}
}

func TestProviderHistoryCollectionPlanStateAndGrantsRemainAtTheSameExactStop(t *testing.T) {
	for _, phase := range []string{"state", "grants"} {
		t.Run(phase, func(t *testing.T) {
			c, b, capture := historyCollectionPlanFixture(t, 13, 4, 2)
			c.phase, c.mode, c.high = phase, "content", c.selection.selected.Number
			c.low = c.high - 3
			table := historyCollectionPlan(t, c, b, capture)
			var params []json.RawMessage
			var block map[string]any
			if table.count != 1 || table.purpose != "state" || table.members[0].request.method != "eth_call" || json.Unmarshal([]byte(table.members[0].request.params), &params) != nil ||
				len(params) != 2 || json.Unmarshal(params[1], &block) != nil || block["blockHash"] != c.selection.selected.Hash || block["requireCanonical"] != true {
				t.Fatal("record/grants request moved to a different stop")
			}
			method := "record"
			if phase == "grants" {
				method = "activeGrants"
			}
			historyCheckPlannedRegistryCall(t, table.members[0].request, method, c.selection.input.journal.Agent, 0, c.selection.selected.Hash)
		})
	}
}

func TestProviderHistoryCollectionTraceTimeoutIsExplicitBeforeH0(t *testing.T) {
	for _, mode := range []string{"missing", "negative", "longer-than-capture"} {
		t.Run(mode, func(t *testing.T) {
			in := historyCollectionInput(t)
			switch mode {
			case "missing":
				in.traceTimeout = 0
			case "negative":
				in.traceTimeout = -time.Nanosecond
			case "longer-than-capture":
				in.traceTimeout = in.capture.Timeout + time.Nanosecond
			}
			if out, err := newProviderHistoryCollectionManifestCandidate(in); err == nil || out != nil {
				t.Fatal("implicit trace timeout gained a new profile", mode)
			}
		})
	}
	in := historyCollectionInput(t)
	a := historyCollectionManifest(t, in)
	in.traceTimeout += time.Nanosecond
	b := historyCollectionManifest(t, in)
	if a.raw == b.raw || a.digest == b.digest {
		t.Fatal("different pre-H0 trace timeouts share a profile")
	}
}

func TestProviderHistoryCollectionCallBuilderRejectsForeignAndAmbiguousRequests(t *testing.T) {
	for _, mode := range []string{"registry", "registry-urn", "agent", "method", "block", "unexpected-version", "missing-version", "nil-meter", "closed-meter", "quota"} {
		t.Run(mode, func(t *testing.T) {
			registry, agent, method, block, version := SepoliaRegistryAddress, "Agent_1", "record", admissionTestDigest("a"), uint64(0)
			meter := providerReplayUnitMeter()
			switch mode {
			case "registry":
				registry = "0x" + strings.Repeat("0", 40)
			case "registry-urn":
				registry = "eip155:11155111:" + SepoliaRegistryAddress
			case "agent":
				agent = "foreign/agent"
			case "method":
				method = "registryId"
			case "block":
				block = "latest"
			case "unexpected-version":
				version = 1
			case "missing-version":
				method = "historyItem"
			case "nil-meter":
				meter = nil
			case "closed-meter":
				meter.closed = true
			case "quota":
				meter.upper = candidateWorkVector{}
			}
			if params, err := providerHistoryRegistryParamsObserved(registry, agent, method, version, block, meter); err == nil || params != nil {
				t.Fatal("foreign or ambiguous DATA gained fixed Registry calldata", mode)
			}
		})
	}
}

// The oracle reads the actual planned RPC request, rather than passing the
// correct address straight to the helper and missing a planner wiring error.
func historyCheckPlannedRegistryCall(t *testing.T, request providerIntegrationRequest, method, agent string, version uint64, selected string) {
	t.Helper()
	contract, _, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	var params []json.RawMessage
	var call map[string]string
	var block map[string]any
	if request.method != "eth_call" || json.Unmarshal([]byte(request.params), &params) != nil || len(params) != 2 ||
		json.Unmarshal(params[0], &call) != nil || json.Unmarshal(params[1], &block) != nil || call["to"] != SepoliaRegistryAddress ||
		block["blockHash"] != selected || block["requireCanonical"] != true {
		t.Fatal("planned RPC used a Registry URN or changed block context")
	}
	raw, err := hex.DecodeString(strings.TrimPrefix(call["data"], "0x"))
	abiMethod, exists := contract.Methods[method]
	if err != nil || !exists || len(raw) < 4 || !bytes.Equal(raw[:4], abiMethod.ID) {
		t.Fatal("planned selector differs from pinned ABI", method)
	}
	args, err := abiMethod.Inputs.Unpack(raw[4:])
	var want []any
	if agent != "" {
		want = append(want, agent)
	}
	if method == "historyItem" {
		want = append(want, version)
	}
	if err != nil || len(args) != len(want) {
		t.Fatal("planned ABI argument count", method, err)
	}
	for i := range want {
		if !reflect.DeepEqual(args[i], want[i]) {
			t.Fatal("planned ABI arguments", method, i, args[i], want[i])
		}
	}
}
