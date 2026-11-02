package registry011

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

// All altered responses below are inert in-memory DATA. No responder, external
// RPC, transport bypass, production hold or operating profile is constructed.
func historyCollectionBoundaryFixture(t *testing.T, blocks, width uint64) (*providerHistoryCollectionCursor, []FinalizedHeader, *providerMethodBounds) {
	t.Helper()
	in := historyCollectionInput(t)
	in.blocks, in.maximumSelected, in.windowBlocks = blocks, sepoliaDeploymentBlock+blocks-1, width
	in.native, in.effects, in.versions, in.presentStops = blocks, 64, 64, 0
	headers := make([]FinalizedHeader, int(blocks))
	headers[0] = in.journal.Deployment
	for i := 1; i < len(headers); i++ {
		headers[i] = FinalizedHeader{fmt.Sprintf("0x%064x", i+1), headers[i-1].Hash, sepoliaDeploymentBlock + uint64(i), headers[0].Timestamp + uint64(i)}
	}
	c, err := newProviderHistoryCollectionCursorObserved(admissionTestDigest("a"), historyCollectionManifest(t, in), headers[len(headers)-1], providerReplayUnitMeter())
	if err != nil || !c.initial() || c.commitmentObserved(providerReplayUnitMeter()) == "" {
		t.Fatal("explicit inert selected interval", err)
	}
	b, _ := providerFrontierBoundsFixture()
	return c, headers, b.methods
}

func historyCollectionBoundaryResult(t *testing.T, raw []byte) providerHistoryMemberResult {
	t.Helper()
	return providerHistoryMemberResult{providerCandidateHash("unit-boundary-result", raw), "", string(raw)}
}

func historyCollectionBoundaryHeader(t *testing.T, h FinalizedHeader) providerHistoryMemberResult {
	t.Helper()
	raw, err := json.Marshal(map[string]string{"hash": h.Hash, "parentHash": h.ParentHash, "number": fmt.Sprintf("0x%x", h.Number), "timestamp": fmt.Sprintf("0x%x", h.Timestamp)})
	if err != nil {
		t.Fatal(err)
	}
	return historyCollectionBoundaryResult(t, raw)
}

func historyCollectionBoundaryPlan(t *testing.T, c *providerHistoryCollectionCursor, methods *providerMethodBounds) *providerHistoryGroupTable {
	t.Helper()
	table, err := c.nextTableObserved(c.completed+1, *methods, c.selection.input.capture, providerReplayUnitMeter())
	if err != nil {
		t.Fatal("exact next DATA table", err)
	}
	return table
}

func historyCollectionBoundaryResults(t *testing.T, c *providerHistoryCollectionCursor, headers []FinalizedHeader, table *providerHistoryGroupTable) []providerHistoryMemberResult {
	t.Helper()
	results := make([]providerHistoryMemberResult, table.count)
	if c.phase == "ancestry" {
		for i := range results {
			results[i] = historyCollectionBoundaryHeader(t, headers[c.next-sepoliaDeploymentBlock-uint64(i)])
		}
		return results
	}
	recorded := nodeRecorded(t)
	read := func(key string) providerHistoryMemberResult {
		var envelope struct{ Result json.RawMessage }
		if json.Unmarshal(recorded[key], &envelope) != nil || len(envelope.Result) == 0 {
			t.Fatal("normal recorded public response", key)
		}
		return historyCollectionBoundaryResult(t, envelope.Result)
	}
	switch c.phase {
	case "open":
		results[0], results[1], results[2], results[3], results[4] = read("chain"), read("genesis"), read("identity-at-anchor"), read("client"), historyCollectionBoundaryHeader(t, c.selection.selected)
		for i, code := range c.selection.input.codes {
			results[5+i] = read("code-" + code.Name)
		}
	case "close":
		results[0], results[1], results[2] = historyCollectionBoundaryHeader(t, headers[c.low-sepoliaDeploymentBlock]), historyCollectionBoundaryHeader(t, c.selection.selected), historyCollectionBoundaryHeader(t, c.selection.selected)
		results[3], results[4], results[5], results[6] = read("chain"), read("genesis"), read("identity-at-anchor"), read("client")
	default:
		for i := range results {
			results[i] = historyCollectionBoundaryResult(t, []byte(`"inert unsupported content"`))
		}
	}
	return results
}

func historyCollectionBoundaryStep(t *testing.T, c *providerHistoryCollectionCursor, headers []FinalizedHeader, methods *providerMethodBounds) *providerHistoryCollectionCursor {
	t.Helper()
	table := historyCollectionBoundaryPlan(t, c, methods)
	next, err := c.advanceBoundaryObserved(context.Background(), table, historyCollectionBoundaryResults(t, c, headers, table), methods, c.semantic.observer)
	if err != nil || next == nil || next == c || next.completed != c.completed+1 {
		t.Fatal("normal original-shaped DATA boundary", c.mode, c.phase, err)
	}
	return next
}

func TestProviderHistoryCollectionBoundariesCoverEveryParentAndOriginalWindow(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		blocks, width, groups uint64
		requests              int
	}{{"deployment-only", 1, 1, 0, 0}, {"three-windows", 13, 4, 9, 87}, {"split-32-and-short-tail", 71, 64, 7, 120}} {
		t.Run(tc.name, func(t *testing.T) {
			c, headers, methods := historyCollectionBoundaryFixture(t, tc.blocks, tc.width)
			selected, requests := c.selection.selected, 0
			for c.mode == "ancestry" {
				table := historyCollectionBoundaryPlan(t, c, methods)
				requests += table.count
				before := c.commitmentObserved(providerReplayUnitMeter())
				next := historyCollectionBoundaryStep(t, c, headers, methods)
				if c.commitmentObserved(providerReplayUnitMeter()) != before || &next.spine[0] == &c.spine[0] {
					t.Fatal("candidate transition changed the previous independent prefix")
				}
				c = next
			}
			if c.completed != tc.groups || requests != tc.requests || len(c.spine) != len(headers) || c.phase != "open" || c.next != sepoliaDeploymentBlock ||
				c.low != sepoliaDeploymentBlock || c.high != min(selected.Number, sepoliaDeploymentBlock+tc.width-1) || c.selection.selected != selected {
				t.Fatal("handwritten ancestry/window count or fixed selected header changed", c.completed, requests)
			}
			for i, h := range c.spine {
				if h != headers[len(headers)-1-i] {
					t.Fatal("full descending spine skipped or replaced a parent", i)
				}
			}
			c = historyCollectionBoundaryStep(t, c, headers, methods)
			if c.phase != "content" || c.completed != tc.groups+1 {
				t.Fatal("content did not require its own identity and all code reads")
			}
			table := historyCollectionBoundaryPlan(t, c, methods)
			before := c.commitmentObserved(providerReplayUnitMeter())
			if next, err := c.advanceBoundaryObserved(context.Background(), table, historyCollectionBoundaryResults(t, c, headers, table), methods, c.semantic.observer); err == nil || next != nil || c.commitmentObserved(providerReplayUnitMeter()) != before {
				t.Fatal("invalid canonical content manufactured a successful prefix")
			}
		})
	}
}

func TestProviderHistoryCollectionOpeningRefusesIdentityCodeAndTableChanges(t *testing.T) {
	for _, mode := range []string{"chain", "genesis", "registry", "client-empty", "client-large", "finality-regression", "finality-same-height", "code-empty", "code-mismatch", "last-code", "result-digest", "result-empty", "result-large", "missing-member", "table-purpose", "table-stage", "table-request", "unused-tail", "nil-methods", "nil-meter", "quota"} {
		t.Run(mode, func(t *testing.T) {
			c, headers, methods := historyCollectionBoundaryFixture(t, 13, 4)
			table := historyCollectionBoundaryPlan(t, c, methods)
			results := historyCollectionBoundaryResults(t, c, headers, table)
			if next, err := c.advanceBoundaryObserved(context.Background(), table, results, methods, c.semantic.observer); err != nil || next.phase != "ancestry" {
				t.Fatal("normal precursor", err)
			}
			meter := c.semantic.observer
			switch mode {
			case "chain":
				results[0] = historyCollectionBoundaryResult(t, []byte(`"0x1"`))
			case "genesis":
				h := FinalizedHeader{admissionTestDigest("f"), admissionTestDigest("0"), 0, 0}
				results[1] = historyCollectionBoundaryHeader(t, h)
			case "registry":
				results[2] = historyCollectionBoundaryResult(t, []byte(`"0x"`))
			case "client-empty":
				results[3] = historyCollectionBoundaryResult(t, []byte(`""`))
			case "client-large":
				raw, _ := json.Marshal(fmt.Sprintf("%0257d", 1))
				results[3] = historyCollectionBoundaryResult(t, raw)
			case "finality-regression", "finality-same-height":
				h := c.selection.selected
				if mode == "finality-regression" {
					h.Number--
				} else {
					h.Hash = admissionTestDigest("f")
				}
				results[4] = historyCollectionBoundaryHeader(t, h)
			case "code-empty":
				results[5] = historyCollectionBoundaryResult(t, []byte(`"0x"`))
			case "code-mismatch":
				results[5] = historyCollectionBoundaryResult(t, []byte(`"0x00"`))
			case "last-code":
				results[17] = historyCollectionBoundaryResult(t, []byte(`"0x00"`))
			case "result-digest":
				results[0].digest = ""
			case "result-empty":
				results[0].result = ""
			case "result-large":
				results[0].result = fmt.Sprintf("%0*d", table.members[0].capture.BodyBytes+1, 1)
			case "missing-member":
				results = results[:len(results)-1]
			case "table-purpose":
				table.purpose = "content"
			case "table-stage":
				table.stage++
			case "table-request":
				table.members[0].request = *historyGroupRequest(t, "web3_clientVersion", []any{})
			case "unused-tail":
				table.members[31].class = "scalar"
			case "nil-methods":
				methods = nil
			case "nil-meter":
				meter = nil
			case "quota":
				meter.upper[10] = 1
			}
			before := c.commitmentObserved(providerReplayUnitMeter())
			if next, err := c.advanceBoundaryObserved(context.Background(), table, results, methods, meter); err == nil || next != nil || c.commitmentObserved(providerReplayUnitMeter()) != before || c.completed != 0 || c.client != "" {
				t.Fatal("invalid normal-shaped DATA changed its original selection", mode)
			}
		})
	}
}

func TestProviderHistoryCollectionAncestryRefusesGapsOrderAndPinnedDeploymentChanges(t *testing.T) {
	for _, mode := range []string{"number", "hash", "parent-link", "timestamp", "order", "deployment-parent", "missing", "decode"} {
		t.Run(mode, func(t *testing.T) {
			c, headers, methods := historyCollectionBoundaryFixture(t, 4, 4)
			c = historyCollectionBoundaryStep(t, c, headers, methods)
			table := historyCollectionBoundaryPlan(t, c, methods)
			results := historyCollectionBoundaryResults(t, c, headers, table)
			if next, err := c.advanceBoundaryObserved(context.Background(), table, results, methods, c.semantic.observer); err != nil || next.phase != "close" || len(next.spine) != 4 {
				t.Fatal("valid complete-parent precursor", err)
			}
			h := headers[2]
			switch mode {
			case "number":
				h.Number--
			case "hash":
				h.Hash = admissionTestDigest("f")
			case "parent-link":
				h.ParentHash = admissionTestDigest("f")
			case "timestamp":
				h.Timestamp = headers[3].Timestamp
			case "order":
				results[0], results[1] = results[1], results[0]
			case "deployment-parent":
				x := headers[0]
				x.ParentHash = admissionTestDigest("f")
				results[2] = historyCollectionBoundaryHeader(t, x)
			case "missing":
				results = results[:2]
			case "decode":
				results[0] = historyCollectionBoundaryResult(t, []byte(`{}`))
			}
			if mode == "number" || mode == "hash" || mode == "parent-link" || mode == "timestamp" {
				results[0] = historyCollectionBoundaryHeader(t, h)
			}
			before := c.commitmentObserved(providerReplayUnitMeter())
			if next, err := c.advanceBoundaryObserved(context.Background(), table, results, methods, c.semantic.observer); err == nil || next != nil || len(c.spine) != 1 || c.commitmentObserved(providerReplayUnitMeter()) != before {
				t.Fatal("partial or inconsistent parents changed the retained prefix")
			}
		})
	}
}

func TestProviderHistoryCollectionWindowCloseRechecksWithoutRefreshingSelection(t *testing.T) {
	for _, mode := range []string{"normal", "newer-finalized", "stop", "selected", "finalized-regression", "chain", "genesis-drift", "registry", "client-drift"} {
		t.Run(mode, func(t *testing.T) {
			c, headers, methods := historyCollectionBoundaryFixture(t, 7, 4)
			c = historyCollectionBoundaryStep(t, c, headers, methods)
			c = historyCollectionBoundaryStep(t, c, headers, methods)
			table := historyCollectionBoundaryPlan(t, c, methods)
			results := historyCollectionBoundaryResults(t, c, headers, table)
			if next, err := c.advanceBoundaryObserved(context.Background(), table, results, methods, c.semantic.observer); err != nil || next.high != sepoliaDeploymentBlock+1 || next.low != sepoliaDeploymentBlock {
				t.Fatal("valid low-stop precursor", err)
			}
			switch mode {
			case "newer-finalized":
				h := c.selection.selected
				h.Number++
				h.Timestamp++
				h.ParentHash, h.Hash = h.Hash, admissionTestDigest("f")
				results[2] = historyCollectionBoundaryHeader(t, h)
			case "stop":
				results[0] = historyCollectionBoundaryHeader(t, headers[len(headers)-1])
			case "selected":
				h := c.selection.selected
				h.ParentHash = admissionTestDigest("f")
				results[1] = historyCollectionBoundaryHeader(t, h)
			case "finalized-regression":
				h := c.selection.selected
				h.Number--
				results[2] = historyCollectionBoundaryHeader(t, h)
			case "chain":
				results[3] = historyCollectionBoundaryResult(t, []byte(`"0x1"`))
			case "genesis-drift":
				h := c.genesis
				h.Timestamp++
				results[4] = historyCollectionBoundaryHeader(t, h)
			case "registry":
				results[5] = historyCollectionBoundaryResult(t, []byte(`"0x"`))
			case "client-drift":
				results[6] = historyCollectionBoundaryResult(t, []byte(`"different normal client"`))
			}
			before, selected := c.commitmentObserved(providerReplayUnitMeter()), c.selection.selected
			next, err := c.advanceBoundaryObserved(context.Background(), table, results, methods, c.semantic.observer)
			if mode == "normal" || mode == "newer-finalized" {
				if err != nil || next == nil || next.selection.selected != selected || next.phase != "open" || next.mode != "ancestry" || next.next != sepoliaDeploymentBlock+1 {
					t.Fatal("valid recheck refreshed selection or skipped a window", err)
				}
			} else if err == nil || next != nil {
				t.Fatal("changed window identity accepted")
			}
			if c.commitmentObserved(providerReplayUnitMeter()) != before {
				t.Fatal("window recheck mutated its source")
			}
		})
	}
}

func TestProviderHistoryCollectionStateSealsInputAndCopiesBothArrays(t *testing.T) {
	c, headers, methods := historyCollectionBoundaryFixture(t, 7, 4)
	for c.mode == "ancestry" {
		c = historyCollectionBoundaryStep(t, c, headers, methods)
	}
	c.phase = "effects" // Separate inert array-copy scenario; never a live S.
	c.effects = []providerHistoryCollectionEffect{{headers[0].Hash, admissionTestDigest("e"), headers[0].Number, 0}}
	before := c.commitmentObserved(providerReplayUnitMeter())
	copy := c.copyObserved(c.semantic.observer)
	if before == "" || copy == nil || copy == c || copy.owner != copy || &copy.spine[0] == &c.spine[0] || &copy.effects[0] == &c.effects[0] || copy.commitmentObserved(providerReplayUnitMeter()) != before {
		t.Fatal("independent arrays or exact reconstructed commitment missing")
	}
	copy.spine[0].Hash, copy.effects[0].transaction = admissionTestDigest("f"), admissionTestDigest("f")
	if c.commitmentObserved(providerReplayUnitMeter()) != before {
		t.Fatal("changing a copy changed its source")
	}
	copy = c.copyObserved(c.semantic.observer)
	copy.selection.input.traceTimeout++
	copy.sealed = copy.selection
	if copy.commitmentObserved(providerReplayUnitMeter()) != "" {
		t.Fatal("coordinated input snapshot change bypassed original manifest")
	}
	if c.copyObserved(nil) != nil || c.commitmentObserved(nil) != "" {
		t.Fatal("collection copy silently used an unobserved path")
	}
	zero := providerReplayUnitMeter()
	zero.upper[10] = 1
	if c.copyObserved(zero) != nil || !zero.failed {
		t.Fatal("refused original observer still allocated an independent state")
	}
	copy = c.copyObserved(c.semantic.observer)
	copy.completed++
	if copy.commitmentObserved(providerReplayUnitMeter()) == before {
		t.Fatal("completed group count missing from cursor commitment")
	}
	if reflect.DeepEqual(copy.effects, nil) {
		t.Fatal("required effect facts were dropped")
	}
}

func TestProviderHistoryCollectionGenericHeaderRetainsFirstPDeploymentMinimum(t *testing.T) {
	recorded := nodeRecorded(t)
	var envelope struct{ Result json.RawMessage }
	if json.Unmarshal(recorded["genesis"], &envelope) != nil {
		t.Fatal("normal genesis response")
	}
	limits := RPCResponseLimits{128 << 10, 64}
	h, err := providerHistoryRPCHeaderObserved(envelope.Result, limits, providerReplayUnitMeter())
	if err != nil || h.Number != 0 || h.Hash != sepoliaGenesis {
		t.Fatal("strict genesis DATA", err)
	}
	if _, err := providerHistoryFirstFinalizedHeaderObserved(envelope.Result, limits, providerReplayUnitMeter()); err == nil {
		t.Fatal("generic parser weakened original first-P minimum")
	}
}
