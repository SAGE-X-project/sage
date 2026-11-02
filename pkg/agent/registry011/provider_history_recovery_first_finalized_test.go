package registry011

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func historyFirstFinalizedHeader() FinalizedHeader {
	return FinalizedHeader{admissionTestDigest("a"), admissionTestDigest("b"), sepoliaDeploymentBlock + 12, 1760000000}
}

func historyFirstFinalizedResponse(t *testing.T) string {
	t.Helper()
	h := historyFirstFinalizedHeader()
	return string(replayTestJSON(t, map[string]any{"jsonrpc": "2.0", "id": 1, "result": map[string]string{
		"hash": h.Hash, "parentHash": h.ParentHash, "number": "0x" + strconv.FormatUint(h.Number, 16), "timestamp": "0x" + strconv.FormatUint(h.Timestamp, 16),
	}}))
}

// Genuine original P/owned TLS/group initialization, with the isolated UNIT
// profile selected before H0. No external RPC, hostile responder or Source.
func historyFirstFinalizedFixture(t *testing.T, response string) (*providerHistoryGroupOwner, *providerHistoryRecoveryOwner, map[string]providerIntegrationObjectRef) {
	return historyFirstFinalizedFixtureWithManifest(t, response, nil)
}

func historyFirstFinalizedFixtureWithManifest(t *testing.T, response string, manifest *providerHistoryCollectionManifest) (*providerHistoryGroupOwner, *providerHistoryRecoveryOwner, map[string]providerIntegrationObjectRef) {
	t.Helper()
	s := historyFirstFinalizedStore(t, response, manifest)
	o, err := newProviderHistoryGroupOwnerCandidate(s)
	if err != nil {
		t.Fatal("genuine original bridge", err)
	}
	r, refs := historyRecoveryGroupStart(t, o)
	return o, r, refs
}

func historyFirstFinalizedStore(t *testing.T, response string, manifest *providerHistoryCollectionManifest) *providerCustodyStore {
	t.Helper()
	_, capture := providerFrontierBoundsFixture()
	capture.Timeout = 10 * time.Second
	if manifest != nil {
		// Carry the exact original manifest selection into H0.
		capture = manifest.input.capture
	}
	s := providerHistoryProvisionedInspectionFixture(t, providerScanFixtureSelection{
		capture: &capture, pin: true, pinLargeObject: true, historyProgram: true, historyGroupWrite: true,
		historyGroupConsume: true, historyGroupDispatch: true, historyRecoveryNamespace: true, historyRecoveryLegacy: true,
		historyCollection: manifest,
	})
	first, err := newProviderFirstOperationCandidate(s)
	if err != nil {
		t.Fatal("genuine first operation", err)
	}
	connection := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, response)
	if _, err := first.executeCandidateWithConnection(context.Background(), connection); err != nil {
		t.Fatal("ordinary original request", err)
	}
	return s
}

func TestProviderHistoryFirstFinalizedReconstructsBothOriginalPassesWithoutProducerCaches(t *testing.T) {
	o, r, refs := historyFirstFinalizedFixture(t, historyFirstFinalizedResponse(t))
	s := o.store
	old, reads, held, pin, ack := o.meter.actual, s.clock.read, s.stored, o.pin, o.last
	before := providerHistoryProvisionedOriginalBytes(t, s.config)
	o.parent.firstIntegration.firstResult = `"ordinary obsolete diagnostic"`
	for pass := range 2 {
		historyRecoveryModelPrefix(t, r, pass, refs)
		m, err := r.replayInitialModelCandidate(context.Background(), pass)
		if err != nil || m == nil {
			t.Fatal("fresh full original model", pass, err)
		}
		work, files, rawBytes := r.meter.actual, r.inputFiles, r.inputBytes
		out, err := m.selectFirstFinalizedCandidate(context.Background())
		if err != nil || out != historyFirstFinalizedHeader() || m.firstFinalized != out || !m.finalizedSelected ||
			r.meter.actual[2] <= work[2] || r.meter.actual[10] <= work[10] || r.inputFiles != files || r.inputBytes != rawBytes {
			t.Fatal("selection borrowed cached DATA or was not observed", pass, err)
		}
		out.Hash = "inert returned DATA"
		if m.firstFinalized != historyFirstFinalizedHeader() {
			t.Fatal("returned DATA changed original selection")
		}
		work = r.meter.actual
		if again, err := m.selectFirstFinalizedCandidate(context.Background()); err == nil || again != (FinalizedHeader{}) || r.meter.actual != work || !r.active {
			t.Fatal("repeated selection spent work or reset the row")
		}
	}
	if o.meter.actual != old || s.clock.read != reads || s.stored != held || o.pin != pin || o.last != ack ||
		s.activeMeter.Load() != nil || s.objects != nil || s.frontierReplay != nil || s.bindings != nil || !o.failed || !o.meter.closed ||
		!r.active || r.failed || r.closed || !reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
		t.Fatal("DATA selection changed originals, old execution, hold or native time")
	}
}

func TestProviderHistoryFirstFinalizedRequiresOriginalCompleteOrderedModelsBeforeWork(t *testing.T) {
	_, r, refs := historyFirstFinalizedFixture(t, historyFirstFinalizedResponse(t))
	for pass := range 2 {
		historyRecoveryModelPrefix(t, r, pass, refs)
		if _, err := r.replayInitialModelCandidate(context.Background(), pass); err != nil {
			t.Fatal(err)
		}
	}
	first, second := r.initialModels[0], r.initialModels[1]
	for _, mode := range []string{"nil", "copy", "incomplete", "second-before-first"} {
		t.Run(mode, func(t *testing.T) {
			work, files, bytes := r.meter.actual, r.inputFiles, r.inputBytes
			candidate := first
			switch mode {
			case "nil":
				candidate = nil
			case "copy":
				copyModel := *first
				copyModel.owner = &copyModel
				candidate = &copyModel
			case "incomplete":
				first.complete = false
			case "second-before-first":
				candidate = second
			}
			out, err := candidate.selectFirstFinalizedCandidate(context.Background())
			first.complete = true
			if err == nil || out != (FinalizedHeader{}) || r.meter.actual != work || r.inputFiles != files || r.inputBytes != bytes ||
				first.finalizedSelected || second.finalizedSelected || !r.active || r.failed || r.closed {
				t.Fatal("unselected model performed work or manufactured selection", mode)
			}
		})
	}
	if _, err := first.selectFirstFinalizedCandidate(context.Background()); err != nil {
		t.Fatal("positive first after precondition refusals", err)
	}
	if _, err := second.selectFirstFinalizedCandidate(context.Background()); err != nil {
		t.Fatal("positive independent second", err)
	}
}

func TestProviderHistoryFirstFinalizedEnteredFailureClosesOnlyReadOnlyLane(t *testing.T) {
	for _, mode := range []string{"legacy-transport-result", "nil-context", "cancelled", "changed-original-bytes"} {
		t.Run(mode, func(t *testing.T) {
			response := historyFirstFinalizedResponse(t)
			if mode == "legacy-transport-result" {
				response = providerFirstOperationResponse()
			}
			o, r, refs := historyFirstFinalizedFixture(t, response)
			historyRecoveryModelPrefix(t, r, 0, refs)
			m, err := r.replayInitialModelCandidate(context.Background(), 0)
			if err != nil {
				t.Fatal("complete transport model must precede header refusal", err)
			}
			old, reads, held, pin, ack := o.meter.actual, o.store.clock.read, o.store.stored, o.pin, o.last
			before := providerHistoryProvisionedOriginalBytes(t, o.store.config)
			ctx := context.Background()
			switch mode {
			case "nil-context":
				ctx = nil
			case "cancelled":
				cancelled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = cancelled
			case "changed-original-bytes":
				// Alter only private intermediate DATA, never a file or traffic.
				found := false
				for _, frame := range m.prefix.source.frames {
					for i := range frame.objects {
						if frame.objects[i].kind == "exchange" {
							frame.objects[i].raw += " "
							found = true
						}
					}
				}
				if !found {
					t.Fatal("genuine original exchange precursor missing")
				}
			}
			out, err := m.selectFirstFinalizedCandidate(ctx)
			if err == nil || out != (FinalizedHeader{}) || m.finalizedSelected || !r.failed || !r.closed || r.active ||
				!r.meter.closed || !r.meter.failed || o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held || o.pin != pin || o.last != ack ||
				!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, o.store.config)) {
				t.Fatal("entered refusal restored old execution or published selection", mode, err)
			}
		})
	}
}

// In-memory parser cases only; no malformed traffic or adversarial server.
func TestProviderHistoryFirstFinalizedHeaderDataPreservesStrictQuantitiesAndBounds(t *testing.T) {
	h := historyFirstFinalizedHeader()
	for _, mode := range []string{"normal", "ordinary-field-order", "extra-case-alias", "before-deployment", "noncanonical-number", "missing-time", "case-alias-only", "bad-hash", "large-time", "wrong-type", "duplicate", "trailing", "invalid-utf8", "nil-meter", "closed-meter", "no-budget", "oversized", "invalid-depth"} {
		t.Run(mode, func(t *testing.T) {
			value := map[string]string{"hash": h.Hash, "parentHash": h.ParentHash, "number": "0x" + strconv.FormatUint(h.Number, 16), "timestamp": "0x" + strconv.FormatUint(h.Timestamp, 16)}
			switch mode {
			case "before-deployment":
				value["number"] = "0x1"
			case "noncanonical-number":
				value["number"] = "0x00"
			case "missing-time":
				delete(value, "timestamp")
			case "case-alias-only":
				delete(value, "hash")
				value["Hash"] = h.Hash
			case "extra-case-alias":
				value["Hash"] = admissionTestDigest("c")
			case "bad-hash":
				value["hash"] = "bad"
			case "large-time":
				value["timestamp"] = "0x20000000000000"
			}
			raw, m := replayTestJSON(t, value), providerReplayUnitMeter()
			limits := RPCResponseLimits{64 << 10, 32}
			switch mode {
			case "ordinary-field-order":
				raw = []byte(fmt.Sprintf(`{"timestamp":%q,"hash":%q,"number":%q,"parentHash":%q}`, value["timestamp"], h.Hash, value["number"], h.ParentHash))
			case "wrong-type":
				raw = []byte(`"0x1"`)
			case "duplicate":
				raw = append(raw[:len(raw)-1], []byte(fmt.Sprintf(`,"hash":%q}`, h.Hash))...)
			case "trailing":
				raw = append(raw, []byte(` false`)...)
			case "invalid-utf8":
				raw = []byte{255}
			case "nil-meter":
				m = nil
			case "closed-meter":
				m.closed = true
			case "no-budget":
				m.upper = candidateWorkVector{}
			case "oversized":
				raw = make([]byte, (64<<10)+1)
			case "invalid-depth":
				limits.MaxDepth = 0
			}
			out, err := providerHistoryFirstFinalizedHeaderObserved(raw, limits, m)
			if mode == "normal" || mode == "ordinary-field-order" || mode == "extra-case-alias" {
				if err != nil || out != h || m.actual[2] == 0 || m.actual[10] == 0 {
					t.Fatal("normal header DATA was not parsed and observed", err)
				}
			} else if err == nil || out != (FinalizedHeader{}) {
				t.Fatal("invalid DATA became a header", mode)
			}
		})
	}
}

// Hand-derived byte domains: two raw JSON scans plus four quoted-string
// decodes, followed by TWO independent hexadecimal codec/encode operations.
// No production meter/vector/quantity helper computes these expected values.
func historyFirstFinalizedParserCosts(t *testing.T) ([]byte, uint64, uint64, uint64, uint64) {
	t.Helper()
	h := historyFirstFinalizedHeader()
	number, stamp := strconv.FormatUint(h.Number, 16), strconv.FormatUint(h.Timestamp, 16)
	raw := replayTestJSON(t, map[string]string{"hash": h.Hash, "parentHash": h.ParentHash, "number": "0x" + number, "timestamp": "0x" + stamp})
	n, d, e := uint64(len(raw)), uint64(len(number)), uint64(len(stamp))
	quotedHash, quotedNumber, quotedStamp := uint64(len(h.Hash)+2), d+4, e+4
	decoded := 2*n + 2*quotedHash + quotedNumber + quotedStamp
	units := 2*(n+1) + 2*(n*n+1) + 2*(quotedHash*quotedHash+1) + quotedNumber*quotedNumber + 1 + quotedStamp*quotedStamp + 1
	return raw, decoded, units, d, e
}

func TestProviderHistoryFirstFinalizedChargesBothQuantityCodecPaths(t *testing.T) {
	raw, decoded, units, d, e := historyFirstFinalizedParserCosts(t)
	m := providerReplayUnitMeter()
	out, err := providerHistoryFirstFinalizedHeaderObserved(raw, RPCResponseLimits{64 << 10, 32}, m)
	// Each successful quantity: compare d+2, decode d, encode d,
	// canonical compare 2*d. The final header comparison is a separate event.
	units += 2*d*d + 3*d + 6 + 2*e*e + 3*e + 6 + 132 + d + e + 5
	want := candidateWorkVector{2: decoded + d + e, 10: units}
	if err != nil || out != historyFirstFinalizedHeader() || m.actual != want {
		t.Fatal("JSON observation replaced or double-counted a separate quantity codec", err, m.actual, want)
	}
}

func TestProviderHistoryFirstFinalizedRefusesQuantityWorkBeforeCrossingQuota(t *testing.T) {
	for _, mode := range []string{"number-codec", "timestamp-codec", "number-encode-preflight"} {
		t.Run(mode, func(t *testing.T) {
			raw, decoded, units, d, e := historyFirstFinalizedParserCosts(t)
			m := providerReplayUnitMeter()
			expectedDecoded, expectedUnits := decoded, units+d+3
			switch mode {
			case "number-codec":
				m.upper[2] = decoded + d - 1
			case "timestamp-codec":
				m.upper[2] = decoded + d + e - 1
				expectedDecoded, expectedUnits = decoded+d, units+(2*d*d+3*d+6)+e+3
			case "number-encode-preflight":
				// JSON and number decoding fit, but the existing fixed 16-digit
				// prospective encode preflight needs 257 logical units, not 256.
				expectedDecoded, expectedUnits = decoded+d, units+d+3+d*d+1
				m.upper[10] = expectedUnits + 256
			}
			out, err := providerHistoryFirstFinalizedHeaderObserved(raw, RPCResponseLimits{64 << 10, 32}, m)
			if err == nil || out != (FinalizedHeader{}) || !m.failed || m.actual[2] != expectedDecoded || m.actual[10] != expectedUnits {
				t.Fatal("quota refusal did not occur at the separate quantity work after the complete JSON precursor", mode, err, m.actual, expectedDecoded, expectedUnits)
			}
			before := m.actual
			if _, err := providerHistoryFirstFinalizedHeaderObserved(raw, RPCResponseLimits{64 << 10, 32}, m); err == nil || m.actual != before {
				t.Fatal("exhausted DATA observer was renewed by another parse")
			}
		})
	}
}
