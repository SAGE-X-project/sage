package registry011

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestProviderHistoryReconstructionMessagesMatchOriginalEventLedger(t *testing.T) {
	for _, row := range []struct {
		name                                        string
		object, body, policy, objects, header, wire uint64
		parts                                       [7][2]uint64
		normal, incoming                            [2]uint64
	}{
		{"small", 4096, 4096, 1024, 4, 1024, 16384, [7][2]uint64{
			{0, 116544}, {245760, 5643051714}, {26624, 102779158}, {54616, 1584696908},
			{24576, 100687884}, {59392, 639690802}, {0, 6597078245650},
		}, [2]uint64{410968, 6605149268660}, [2]uint64{21846, 1099511627777}},
		{"medium", 1 << 20, 131072, 8192, 64, 8192, 1 << 20, [7][2]uint64{
			{0, 10956744}, {245760, 5643051714}, {6307840, 6597208195350}, {393218, 96936103498},
			{786432, 103080001548}, {2899968, 2302237950002}, {0, 6597080842514},
		}, [2]uint64{10633218, 15702197101370}, [2]uint64{1048576, 1099511627777}},
		{"maximum", 32 << 20, 8 << 20, 65536, 1024, 65536, 16 << 20, [7][2]uint64{
			{0, 340375624}, {245760, 5643051714}, {201457664, 6755408165339414}, {22413314, 390939018759754},
			{50331648, 422212515397644}, {84017152, 985171077234738}, {0, 6597178884370},
		}, [2]uint64{358465538, 8560333939043258}, [2]uint64{33554432, 1125899906842625}},
	} {
		t.Run(row.name, func(t *testing.T) {
			limits := providerHistoryLegacyControlLimits{65536, row.object, row.objects, 64 << 20, 256, 4, 32}
			capture := ProviderCaptureLimits{Timeout: time.Second, HeaderBytes: int(row.header), BodyBytes: int(row.body), WireBytes: int(row.wire), JSONDepth: 64}
			before, original := limits, capture
			got, ok := providerHistoryReconstructionMessageUpper(limits, capture, row.policy)
			if !ok || limits != before || capture != original ||
				got.normal != (candidateWorkVector{2: row.normal[0], 10: row.normal[1]}) ||
				got.incoming != (candidateWorkVector{2: row.incoming[0], 10: row.incoming[1]}) ||
				got.envelope != (candidateWorkVector{2: row.normal[0] + row.incoming[0], 10: row.normal[1] + row.incoming[1]}) ||
				[4]uint64{got.requestCalls, got.exchangeCalls, got.bindingCandidates, got.objectLookups} != [4]uint64{6, 2, 4, 24} {
				t.Fatal("two original message passes lost a request, rejection prefix or original guard", got, ok)
			}
			for i, part := range [...]candidateWorkVector{got.lookups, got.requests, got.records, got.base64, got.rpc, got.http, got.transport} {
				if part != (candidateWorkVector{2: row.parts[i][0], 10: row.parts[i][1]}) {
					t.Fatal("independently enumerated original message event row", i, part, row.parts[i])
				}
			}
		})
	}
}

func TestProviderHistoryReconstructionMessagesRefuseUnselectedInputs(t *testing.T) {
	for _, mode := range []string{"metadata-zero", "metadata-large", "object-zero", "object-large", "first-objects", "first-raw", "entries", "attempts", "chunks-zero", "chunks-overflow", "policy-zero", "policy-large", "timeout", "body-zero", "body-large", "header-zero", "wire-zero", "depth"} {
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
			case "header-zero":
				capture.HeaderBytes = 0
			case "wire-zero":
				capture.WireBytes = 0
			case "depth":
				capture.JSONDepth = 0
			}
			if got, ok := providerHistoryReconstructionMessageUpper(limits, capture, policy); ok || got != (providerHistoryReconstructionMessageCost{}) {
				t.Fatal("unselected original input returned partial message costs", mode, got, ok)
			}
		})
	}
}

func TestProviderHistoryReconstructionMessagesKeepIndependentGuards(t *testing.T) {
	limits, capture := historyLegacyControlLimits(), historyFirstInterpretationCapture()
	limits.objectBytes = 4096
	capture.BodyBytes, capture.WireBytes = 8<<20, 16<<20
	got, ok := providerHistoryReconstructionMessageUpper(limits, capture, 1024)
	if !ok || got.requests != (candidateWorkVector{2: 245760, 10: 5643051714}) ||
		got.records[2] != 2*(3*4096+1024) || got.incoming[2] != 16<<20 ||
		got.incoming[10] != (16<<20)*(16<<20)+1 || got.bindingCandidates != 4 {
		t.Fatal("P replaced raw normalization, encoded input, wire or independent encoder guards", got, ok)
	}
	for _, coordinate := range []int{0, 1, 3, 4, 5, 6, 7, 8, 9} {
		if got.envelope[coordinate] != 0 {
			t.Fatal("message costing acquired another resource or execution role", coordinate)
		}
	}
}

// This UNIT prepares fresh arithmetic DATA from one genuine selected prefix.
// It does not install an initial model, change an original or revive a failed
// producer. Preparatory body/index work remains outside the message snapshots.
func historyReconstructionMessageModel(t *testing.T, r *providerHistoryRecoveryOwner, pass int) (*providerHistoryRecoveryInitialModel, *CandidateAdmissionSettlement) {
	t.Helper()
	f, err := newProviderFrontierReplayObserved(r.config, r.meter)
	if err != nil || f == nil || r.legacyPrefixes[pass] == nil {
		t.Fatal("fresh message model from the actual selected prefix", err)
	}
	f.meter, f.body.meter, f.indexes.meter = r.meter, r.meter, r.meter
	m := &providerHistoryRecoveryInitialModel{recovery: r, prefix: r.legacyPrefixes[pass], frontier: f}
	m.owner = m
	for i, frame := range m.prefix.source.frames {
		m.used[i] = make([]bool, len(frame.objects))
	}
	for i := range 3 {
		m.through = uint64(i + 1)
		if m.apply(m.prefix.records[i]) != nil {
			t.Fatal("actual enrollment/admission/consumption message precursor", pass, i)
		}
	}
	m.through = 4
	state, exists := f.body.ledger.tickets[r.initialDescriptor.ticket]
	if !exists || state.ticket == nil || state.consumption == nil {
		t.Fatal("original consumed first ticket for message checks")
	}
	raw, err := m.object("settlement", m.prefix.records[3].operation)
	settlement, parseErr := parseCandidateAdmissionSettlementObserved(state.ticket, raw, 64<<10, r.meter)
	if err != nil || parseErr != nil || settlement == nil {
		t.Fatal("actual original first settlement precursor", err, parseErr)
	}
	return m, settlement
}

// Independent length oracle uses standard codecs over actual selected frames,
// never a production observer, producer binding map or another model's result.
func historyReconstructionMessageDecoded(t *testing.T, m *providerHistoryRecoveryInitialModel, s *CandidateAdmissionSettlement) (uint64, uint64) {
	t.Helper()
	lookup := func(kind, digest string) []byte {
		var raw []byte
		var count int
		for _, frame := range m.prefix.source.frames {
			for _, object := range frame.objects {
				if object.kind == kind && object.digest == digest {
					raw, count = []byte(object.raw), count+1
				}
			}
		}
		if count != 1 || len(raw) == 0 {
			t.Fatal("one original message dependency", kind, count)
		}
		return raw
	}
	request := lookup("request", s.ticket.input)
	parts := bytes.SplitN(request, []byte("\r\n\r\n"), 2)
	if len(parts) != 2 {
		t.Fatal("actual original request framing")
	}
	exchange := lookup("exchange", s.retained)
	var x struct{ Request, Response string }
	if json.Unmarshal(exchange, &x) != nil {
		t.Fatal("actual exchange JSON oracle")
	}
	body, err := base64.RawURLEncoding.DecodeString(x.Response)
	var envelope map[string]json.RawMessage
	if err != nil || json.Unmarshal(body, &envelope) != nil || len(envelope["jsonrpc"]) == 0 {
		t.Fatal("actual original RPC response oracle", err)
	}
	var bindingRaw []byte
	var binding struct {
		ResponseWireDigest string `json:"response_wire_digest"`
	}
	var count int
	for _, frame := range m.prefix.source.frames {
		for _, object := range frame.objects {
			if object.kind == "wire-binding" {
				bindingRaw, count = []byte(object.raw), count+1
			}
		}
	}
	if count != 1 || json.Unmarshal(bindingRaw, &binding) != nil {
		t.Fatal("one original successful exchange binding", count)
	}
	response := lookup("response", binding.ResponseWireDigest)
	head := bytes.Index(response, []byte("\r\n\r\n")) + 4
	if head < 4 || len(x.Request) == 0 || len(x.Response) == 0 {
		t.Fatal("actual response header and both base64 fields")
	}
	// request has two body codecs. exchange has exchange/policy/binding,
	// both ENCODED base64 inputs, HTTP header/wire, and two independent RPC
	// parsers each decoding the whole body twice plus its raw jsonrpc value.
	return 2 * uint64(len(parts[1])), uint64(len(exchange) + len(m.frontier.body.policy.raw) + len(bindingRaw) +
		len(x.Request) + len(x.Response) + head + len(response) + 4*len(body) + 2*len(envelope["jsonrpc"]))
}

func TestProviderHistoryReconstructionMessagesContainOriginalSelectedCalls(t *testing.T) {
	s, in, _ := historyCollectionRecordedRunStore(t)
	o, err := newProviderHistoryGroupOwnerCandidate(s)
	if err != nil || o == nil {
		t.Fatal("original bounded collection initialization", err)
	}
	r, refs := historyRecoveryGroupStart(t, o)
	bounds := s.config.bounds
	if bounds.history.collection == nil || bounds.history.collection.input.firstRead != in.firstRead || s.objects != nil {
		t.Fatal("pre-H0 collection selection and discarded producer caches")
	}
	quote, ok := providerHistoryReconstructionMessageUpper(providerHistoryLegacyControlLimits{
		bounds.maximumMetadataPerCommit, bounds.maximumObjectBytes, in.firstRead.objects, in.firstRead.raw,
		uint64(bounds.maximumEntries), bounds.scan.maximumAttempts, bounds.scan.maximumChunks,
	}, s.config.capture, uint64(len(s.config.policy.raw)))
	if !ok {
		t.Fatal("same-source initial message quote")
	}
	old, clocks, held, pin, ack := o.meter.actual, s.clock.read, s.stored, o.pin, o.last
	originals := providerHistoryProvisionedOriginalBytes(t, s.config)
	var decoded, logical uint64
	var first *providerHistoryRecoveryInitialModel
	for pass := range 2 {
		historyRecoveryModelPrefix(t, r, pass, refs)
		func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			m, settlement := historyReconstructionMessageModel(t, r, pass)
			if pass == 0 {
				first = m
			} else if first == m || first.prefix == m.prefix || first.frontier == m.frontier || first.frontier.body.ledger == m.frontier.body.ledger {
				t.Fatal("independent message arithmetic reused another prefix or model")
			}
			requestBytes, exchangeBytes := historyReconstructionMessageDecoded(t, m, settlement)
			work, files, rawBytes := r.meter.actual, r.inputFiles, r.inputBytes
			for range 3 {
				before := r.meter.actual
				if err := m.request(settlement.ticket); err != nil || r.meter.actual[2] < before[2] ||
					r.meter.actual[2]-before[2] != requestBytes {
					t.Fatal("original request codecs lost exact attribution", pass, err)
				}
			}
			before := r.meter.actual
			if err := m.exchange(settlement); err != nil || r.meter.actual[2] < before[2] || r.meter.actual[2]-before[2] != exchangeBytes ||
				r.meter.actual[10] <= work[10] || r.inputFiles != files || r.inputBytes != rawBytes {
				t.Fatal("original exchange codecs changed, performed I/O or skipped independent RPC checks", pass, err)
			}
			decoded += 3*requestBytes + exchangeBytes
			logical += r.meter.actual[10] - work[10]
		}()
	}
	if decoded == 0 || decoded > quote.normal[2] || logical > quote.normal[10] || o.meter.actual != old || s.clock.read != clocks ||
		s.stored != held || o.pin != pin || o.last != ack || !o.failed || !o.meter.closed || s.activeMeter.Load() != nil || s.objects != nil ||
		!r.active || r.closed || r.failed || r.meter.authority || r.meter.historyRecovery != r ||
		r.initialModels != ([2]*providerHistoryRecoveryInitialModel{}) || !reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, s.config)) {
		t.Fatal("message UNIT exceeded its quote, installed a model or renewed producer execution", decoded, logical)
	}
}
