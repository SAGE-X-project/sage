package registry011

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

// A test-only length oracle over the completed pass's original immutable
// frames. Standard JSON/base64 decoding here never calls a production observer
// or borrows the failed producer's result, lookup cache or earlier pass.
func historyReconstructionFinalizedDecodedBytes(t *testing.T, m *providerHistoryRecoveryInitialModel) uint64 {
	t.Helper()
	if m == nil || m.recovery == nil || !m.selectedLocked(m.recovery) {
		t.Fatal("complete original initial-model precursor for decode attribution")
	}
	n := m.recovery.initialDescriptor
	entry, ok := m.frontier.body.ledger.tickets[n.ticket]
	if !ok || entry.ticket == nil || entry.ticket.policy == "" || entry.terminal == "" {
		t.Fatal("actual retained first ticket and settlement precursor")
	}
	lookup := func(kind, digest string) []byte {
		var raw []byte
		var count int
		for _, frame := range m.prefix.source.frames {
			for _, object := range frame.objects {
				if object.kind == kind && object.digest == digest {
					raw = []byte(object.raw)
					count++
				}
			}
		}
		if count != 1 || len(raw) == 0 {
			t.Fatal("one exact original frame dependency", kind, count)
		}
		return raw
	}
	exchange, settlement := lookup("exchange", n.exchange), lookup("settlement", entry.terminal)
	var x struct{ Response string }
	if json.Unmarshal(exchange, &x) != nil || x.Response == "" {
		t.Fatal("actual original exchange response field")
	}
	response, err := base64.RawURLEncoding.DecodeString(x.Response)
	if err != nil || len(response) == 0 {
		t.Fatal("actual original response bytes", err)
	}
	var envelope, fields map[string]json.RawMessage
	if json.Unmarshal(response, &envelope) != nil || len(envelope["jsonrpc"]) == 0 || len(envelope["result"]) == 0 ||
		json.Unmarshal(envelope["result"], &fields) != nil {
		t.Fatal("original successful RPC envelope and header")
	}
	// Exchange, settlement, ticket policy, encoded response, two complete
	// response decodes plus jsonrpc, two complete header decodes, four raw
	// header field decodes and both quantities' actual hexadecimal digits.
	decoded := uint64(len(exchange) + len(settlement) + len(entry.ticket.policy) + len(x.Response) +
		2*len(response) + len(envelope["jsonrpc"]) + 2*len(envelope["result"]))
	for _, name := range []string{"hash", "parentHash", "number", "timestamp"} {
		if len(fields[name]) == 0 {
			t.Fatal("actual original header field", name)
		}
		decoded += uint64(len(fields[name]))
	}
	for _, name := range []string{"number", "timestamp"} {
		var text string
		if json.Unmarshal(fields[name], &text) != nil || len(text) < 3 || text[:2] != "0x" {
			t.Fatal("actual successful original quantity", name)
		}
		decoded += uint64(len(text) - 2)
	}
	return decoded
}

func TestProviderHistoryReconstructionFinalizedMatchesOriginalLeafRows(t *testing.T) {
	for _, row := range []struct {
		object, body, policy, objects uint64
		parts                         [7][2]uint64
		normal, incoming              [2]uint64
	}{
		{4096, 4096, 1024, 4, [7][2]uint64{
			{0, 54548}, {18432, 69224530}, {16384, 806234862}, {10924, 93243174},
			{24576, 100687884}, {49216, 201345594}, {0, 76576},
		}, [2]uint64{119532, 1270867168}, [2]uint64{8192, 268435457}},
		{1 << 20, 131072, 8192, 64, [7][2]uint64{
			{0, 12655268}, {2244608, 2207747686482}, {16384, 806234862}, {349526, 95444649764},
			{786432, 103080001548}, {1572928, 206158957114}, {0, 2165536},
		}, [2]uint64{4969878, 2613252350574}, [2]uint64{1048576, 1099511627777}},
		{32 << 20, 8 << 20, 65536, 1024, [7][2]uint64{
			{0, 403796900}, {67371008, 2251816993947730}, {16384, 806234862}, {22369622, 390937527306020},
			{50331648, 422212515397644}, {100663360, 844424963689018}, {0, 67177248},
		}, [2]uint64{240752022, 3909393277549422}, [2]uint64{33554432, 1125899906842625}},
	} {
		limits := providerHistoryLegacyControlLimits{65536, row.object, row.objects, 64 << 20, 256, 4, 32}
		capture := ProviderCaptureLimits{Timeout: time.Second, HeaderBytes: 1024, BodyBytes: int(row.body), WireBytes: 16 << 20, JSONDepth: 64}
		before, original := limits, capture
		got, ok := providerHistoryReconstructionFinalizedUpper(limits, capture, row.policy)
		if !ok || limits != before || capture != original ||
			got.normal != (candidateWorkVector{2: row.normal[0], 10: row.normal[1]}) ||
			got.incoming != (candidateWorkVector{2: row.incoming[0], 10: row.incoming[1]}) ||
			got.envelope != (candidateWorkVector{2: row.normal[0] + row.incoming[0], 10: row.normal[1] + row.incoming[1]}) {
			t.Fatal("two fresh finalized selections lost original scopes or a codec guard", row, got, ok)
		}
		for i, part := range [...]candidateWorkVector{got.lookups, got.records, got.request, got.base64, got.rpc, got.header, got.comparisons} {
			if part != (candidateWorkVector{2: row.parts[i][0], 10: row.parts[i][1]}) {
				t.Fatal("original finalized selection event-ledger row", i, part, row.parts[i])
			}
		}
	}
}

func TestProviderHistoryReconstructionFinalizedRefusesUnselectedInputs(t *testing.T) {
	for _, mode := range []string{"metadata-zero", "metadata-large", "object-zero", "object-large", "first-objects", "first-raw", "entries", "attempts", "chunks-zero", "chunks-overflow", "policy-zero", "policy-large", "timeout", "body-zero", "body-large", "depth"} {
		t.Run(mode, func(t *testing.T) {
			limits, capture, policy := historyLegacyControlLimits(), historyFirstInterpretationCapture(), uint64(8192)
			switch mode {
			case "metadata-zero":
				limits.metadata = 0
			case "metadata-large":
				limits.metadata = 1<<20 + 1
			case "object-zero":
				limits.objectBytes = 0
			case "object-large":
				limits.objectBytes = 32<<20 + 1
			case "first-objects":
				limits.firstObjects = 0
			case "first-raw":
				limits.firstRaw = 0
			case "entries":
				limits.entries = 0
			case "attempts":
				limits.attempts = 0
			case "chunks-zero":
				limits.chunks = 0
			case "chunks-overflow":
				limits.chunks = ^uint64(0)
			case "policy-zero":
				policy = 0
			case "policy-large":
				policy = 64<<10 + 1
			case "timeout":
				capture.Timeout = 0
			case "body-zero":
				capture.BodyBytes = 0
			case "body-large":
				capture.BodyBytes = 8<<20 + 1
			case "depth":
				capture.JSONDepth = 0
			}
			if got, ok := providerHistoryReconstructionFinalizedUpper(limits, capture, policy); ok || got != (providerHistoryReconstructionFinalizedCost{}) {
				t.Fatal("invalid original finalized limits returned a partial row", mode, got, ok)
			}
		})
	}
}

func TestProviderHistoryReconstructionFinalizedKeepsEncodedAndOriginalGuardWidths(t *testing.T) {
	limits, capture := historyLegacyControlLimits(), historyFirstInterpretationCapture()
	limits.objectBytes = 4096
	capture.BodyBytes = 8 << 20
	got, ok := providerHistoryReconstructionFinalizedUpper(limits, capture, 1024)
	if !ok || got.incoming[2] != 11184811 || got.incoming[10] != 11184811*11184811+1 ||
		got.request != (candidateWorkVector{2: 16384, 10: 806234862}) ||
		got.records[2] != 2*(4096+4096+1024) || !providerHistoryReadWorkDomain(got.envelope) {
		t.Fatal("raw response length replaced encoded input or original request encoder guards", got, ok)
	}
	for _, coordinate := range []int{0, 1, 3, 4, 5, 6, 7, 8, 9} {
		if got.envelope[coordinate] != 0 {
			t.Fatal("finalized DATA costing acquired another resource or execution role", coordinate)
		}
	}
}

func TestProviderHistoryReconstructionFinalizedContainsBothActualOriginalSelections(t *testing.T) {
	s, in, _ := historyCollectionRecordedRunStore(t)
	o, err := newProviderHistoryGroupOwnerCandidate(s)
	if err != nil || o == nil {
		t.Fatal("original bounded collection initialization", err)
	}
	r, refs := historyRecoveryGroupStart(t, o)
	bounds, manifest := s.config.bounds, s.config.bounds.history.collection
	if manifest == nil || !manifest.validSnapshot() || manifest.input.firstRead != in.firstRead || s.objects != nil {
		t.Fatal("pre-H0 collection limits and discarded producer caches")
	}
	quote, ok := providerHistoryReconstructionFinalizedUpper(providerHistoryLegacyControlLimits{
		bounds.maximumMetadataPerCommit, bounds.maximumObjectBytes, in.firstRead.objects, in.firstRead.raw,
		uint64(bounds.maximumEntries), bounds.scan.maximumAttempts, bounds.scan.maximumChunks,
	}, s.config.capture, uint64(len(s.config.policy.raw)))
	if !ok {
		t.Fatal("same-source finalized interpretation quote")
	}
	old, reads, held, pin, ack := o.meter.actual, s.clock.read, s.stored, o.pin, o.last
	originals := providerHistoryProvisionedOriginalBytes(t, s.config)
	var decoded, logical uint64
	for pass := range 2 {
		historyRecoveryGroupInitial(t, r, pass, refs)
		m := r.initialModels[pass]
		expected := historyReconstructionFinalizedDecodedBytes(t, m)
		work, files, rawBytes := r.meter.actual, r.inputFiles, r.inputBytes
		out, err := m.selectFirstFinalizedCandidate(context.Background())
		if err != nil || out != in.journal.Deployment || !m.finalizedSelected || r.meter.actual[2] < work[2] ||
			r.meter.actual[2]-work[2] != expected || expected == 0 || r.meter.actual[10] <= work[10] || r.inputFiles != files || r.inputBytes != rawBytes {
			t.Fatal("actual first-finalized interpretation changed source, decoded stream or original I/O", pass, err)
		}
		decoded += expected
		logical += r.meter.actual[10] - work[10]
	}
	if decoded > quote.normal[2] || logical > quote.normal[10] || o.meter.actual != old || s.clock.read != reads || s.stored != held ||
		o.pin != pin || o.last != ack || !o.failed || !o.meter.closed || s.activeMeter.Load() != nil || s.objects != nil ||
		!r.active || r.closed || r.failed || !reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, s.config)) {
		t.Fatal("original finalized interpretation exceeded its row or renewed execution/resources", decoded, logical)
	}
}
