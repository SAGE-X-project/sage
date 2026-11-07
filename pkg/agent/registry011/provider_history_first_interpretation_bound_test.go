package registry011

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"testing"
	"time"
)

func historyFirstInterpretationCapture() ProviderCaptureLimits {
	return ProviderCaptureLimits{Timeout: time.Second, HeaderBytes: 8192, BodyBytes: 131072, WireBytes: 262144, JSONDepth: 64}
}

func TestProviderHistoryFirstInterpretationMatchesIndependentRows(t *testing.T) {
	for _, row := range []struct {
		object, headers, body, wire uint64
		want                        [9][2]uint64
	}{
		{4096, 1024, 4096, 8192, [9][2]uint64{
			{425984, 38806466870}, {27308, 792348454}, {21504, 118518809}, {12288, 50343942}, {24608, 100672797},
			{0, 3298540249629}, {0, 67117627}, {8192, 403117431}, {519884, 3338878835559},
		}},
		{1048576, 8192, 131072, 262144, [9][2]uint64{
			{2883584, 2263465922870}, {196609, 48468051749}, {663552, 120326823961}, {393216, 51540000774}, {786464, 103079478557},
			{0, 3298540503581}, {0, 67117627}, {8192, 403117431}, {4931617, 5885891016550},
		}},
		{33554432, 65536, 8388608, 16777216, [9][2]uint64{
			{67895296, 2251864256352566}, {11206657, 195469509379877}, {42008576, 492585538617369}, {25165824, 211106257698822}, {50331680, 422212481844509},
			{0, 3298557018653}, {0, 67117627}, {8192, 403117431}, {196616225, 3576537071146854},
		}},
	} {
		capture := ProviderCaptureLimits{Timeout: time.Second, HeaderBytes: int(row.headers), BodyBytes: int(row.body), WireBytes: int(row.wire), JSONDepth: 64}
		got, ok := providerHistoryFirstPInterpretationUpper(row.object, capture)
		values := [...]candidateWorkVector{got.records, got.base64, got.http, got.rpc, got.header, got.comparisons, got.initialRoot, got.request, got.total}
		if !ok {
			t.Fatal("finite independent first-P interpretation row", row)
		}
		for i, value := range values {
			if value != (candidateWorkVector{2: row.want[i][0], 10: row.want[i][1]}) {
				t.Fatal("independent first-P field vector", i, value, row.want[i])
			}
		}
	}
	for _, row := range []struct{ groups, requests, decoded, logical uint64 }{
		{1, 1, 69042638, 82402474231700}, {1, 32, 69042638, 82402474231700},
		{2, 3, 138085276, 164804948463400}, {3, 96, 207127914, 247207422695100},
	} {
		got, ok := providerHistoryCollectionRepeatedFirstPInterpretationUpper(row.groups, row.requests, 1<<20, historyFirstInterpretationCapture())
		if !ok || got != (candidateWorkVector{2: row.decoded, 10: row.logical}) {
			t.Fatal("original fourteen-model repetition changed", row, got)
		}
	}
}

func TestProviderHistoryFirstInterpretationFundsTypedCandidateRecords(t *testing.T) {
	policy, context, anchor, fields := admissionTestIntentFixture(t, nil)
	upper, ok := providerHistoryFirstPInterpretationUpper(1<<20, historyFirstInterpretationCapture())
	if !ok {
		t.Fatal("finite record parsing quote")
	}
	m := providerHistoryContentUnitMeter(upper.records)
	parsedPolicy, err := parseProviderPolicyCandidateObserved(providerReplayPhaseBytes(policy.raw, m), 64<<10, m)
	if err != nil || parsedPolicy == nil || *parsedPolicy != *policy {
		t.Fatal("original policy/subtree parse", err)
	}
	parsedAnchor, err := parseCandidateEnrollmentRootObserved(context, anchor.CanonicalBytes(), 64<<10, m)
	if err != nil || parsedAnchor == nil || *parsedAnchor != *anchor {
		t.Fatal("original enrollment parse", err)
	}
	intentRaw := providerPolicyTestBytes(t, fields)
	intent, err := parseCandidateAdmissionIntentObserved(policy, context, anchor, intentRaw, 64<<10, m)
	if err != nil || intent == nil {
		t.Fatal("original admission parse", err)
	}
	_, expectedIntent, ticketFields := admissionTestTicketFixture(t)
	if intent.digest != expectedIntent.digest {
		t.Fatal("shared deterministic candidate context precursor")
	}
	ticketRaw := providerPolicyTestBytes(t, ticketFields)
	ticket, err := parseCandidateAdmissionTicketObserved(policy, intent, 0, ticketRaw, 64<<10, m)
	if err != nil || ticket == nil {
		t.Fatal("original ticket/segment parse", err)
	}
	expectedTicket, consumedFields := admissionTestConsumptionFixture(t)
	if expectedTicket.digest != ticket.digest {
		t.Fatal("shared deterministic consumption precursor")
	}
	consumedRaw := providerPolicyTestBytes(t, consumedFields)
	consumed, err := parseCandidateConsumptionObserved(ticket, consumedRaw, 64<<10, m)
	if err != nil || consumed == nil {
		t.Fatal("original consumption/policy parse", err)
	}
	settledRaw := providerPolicyTestBytes(t, ledgerTestSettlementObject(ticket, "complete", 20))
	settled, err := parseCandidateAdmissionSettlementObserved(ticket, settledRaw, 64<<10, m)
	if err != nil || settled == nil || settled.outcome != "complete" || m.failed {
		t.Fatal("original settlement/policy parse", err)
	}
	// This subject checks the six typed parsers, without pretending to have
	// executed the extra first-P exchange/binding/caller codecs or its owner.
	wantDecoded := uint64(5*len(policy.raw) + len(anchor.raw) + len(intentRaw) + len(ticketRaw) + len(consumedRaw) + len(settledRaw))
	if m.actual[2] != wantDecoded || m.actual[10] > upper.records[10] {
		t.Fatal("typed record decoder multiplicity changed", m.actual, wantDecoded)
	}
}

func TestProviderHistoryFirstInterpretationCountsEncodedBase64Bytes(t *testing.T) {
	for _, row := range []struct{ maximum, encoded, logical uint64 }{
		{1, 2, 12}, {2, 3, 22}, {3, 4, 36}, {16384, 21846, 745726867}, {8388608, 11184811, 195468763653010},
	} {
		got, ok := providerHistoryBase64ControlUpper(row.maximum)
		if !ok || got != (candidateWorkVector{2: row.encoded, 10: row.logical}) {
			t.Fatal("independent raw URL-base64 size/coordinate", row, got)
		}
	}
	for _, count := range []int{1, 2, 3, 1024, 16384} {
		raw := bytes.Repeat([]byte{0x61}, count)
		encoded := base64.RawURLEncoding.EncodeToString(raw)
		upper, ok := providerHistoryBase64ControlUpper(uint64(count))
		if !ok {
			t.Fatal("finite base64 quote")
		}
		m := providerHistoryContentUnitMeter(upper)
		got, valid := providerCanonicalBase64Observed(encoded, count, m)
		n, e := uint64(count), uint64(len(encoded))
		if !valid || !bytes.Equal(got, raw) || m.failed || m.actual != (candidateWorkVector{2: e, 10: e*e + n*n + 2*e + 3}) {
			t.Fatal("encoded input was charged as decoded output or spelling changed", count, m.actual)
		}
	}
}

func TestProviderHistoryFirstInterpretationParsesFinalizedHeaderWithFieldOrderPreserved(t *testing.T) {
	response := []byte(historyFirstFinalizedResponse(t))
	result, err := decodeRPCResultObserved(response, 1, RPCResponseLimits{131072, 64}, nil)
	if err != nil {
		t.Fatal("normal original finalized envelope", err)
	}
	before := bytes.Clone(result)
	upper, ok := providerHistoryFirstPHeaderControlUpper(131072)
	if !ok {
		t.Fatal("finite header quote")
	}
	m := providerHistoryContentUnitMeter(upper)
	got, err := providerHistoryFirstFinalizedHeaderObserved(result, RPCResponseLimits{131072, 64}, m)
	if err != nil || got != historyFirstFinalizedHeader() || !bytes.Equal(result, before) || m.failed || m.actual[2] > upper[2] || m.actual[10] > upper[10] {
		t.Fatal("original field/token/quantity header parsing exceeded quote", err, m.actual)
	}
}

func TestProviderHistoryFirstInterpretationKeepsBothEnvelopeParses(t *testing.T) {
	capture := historyFirstInterpretationCapture()
	upper, ok := providerHistoryFirstPInterpretationUpper(1<<20, capture)
	if !ok {
		t.Fatal("finite original HTTP/RPC quote")
	}
	body := []byte(historyFirstFinalizedResponse(t))
	wire := []byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", len(body), body))
	before := bytes.Clone(wire)
	// One observer retains all real earlier work. Exchange response bytes
	// come from their independent Base64 input, not the HTTP parser's result.
	m := providerHistoryContentUnitMeter(upper.total)
	response, valid := providerCanonicalBase64Observed(base64.RawURLEncoding.EncodeToString(body), capture.BodyBytes, m)
	if !valid || !bytes.Equal(response, body) || m.failed {
		t.Fatal("normal independent exchange-response precursor")
	}
	afterBase64 := m.actual
	_, headers, parsed, ignoredResult, err := readProviderHTTPObserved(bufio.NewReader(bytes.NewReader(wire)), capture, m, uint64(len(wire)))
	if err != nil || len(headers) == 0 || !bytes.Equal(parsed, body) || len(ignoredResult) == 0 || m.failed {
		t.Fatal("normal original HTTP envelope parse", err)
	}
	afterHTTP := m.actual
	if !providerReplayEqual(parsed, response, m) {
		t.Fatal("actual caller body comparison")
	}
	afterComparison := m.actual
	if afterComparison[10]-afterHTTP[10] != uint64(len(parsed)+len(response)+1) {
		t.Fatal("caller body equality was omitted or counted more than once")
	}
	for coordinate := range afterHTTP {
		if afterComparison[coordinate] < afterHTTP[coordinate] || coordinate != 10 && afterComparison[coordinate] != afterHTTP[coordinate] {
			t.Fatal("caller comparison erased prior HTTP work or decoded bytes", coordinate)
		}
	}
	result, err := decodeRPCResultObserved(response, 1, RPCResponseLimits{capture.BodyBytes, capture.JSONDepth}, m)
	if err != nil || !bytes.Equal(result, ignoredResult) || m.failed || m.actual[2] == afterComparison[2] {
		t.Fatal("separate caller envelope parse was replaced by cached result", err)
	}
	afterRPC := m.actual
	header, err := providerHistoryFirstFinalizedHeaderObserved(result, RPCResponseLimits{capture.BodyBytes, capture.JSONDepth}, m)
	if err != nil || header != historyFirstFinalizedHeader() || m.failed || !bytes.Equal(wire, before) {
		t.Fatal("original response/header DATA changed", err)
	}
	// Independently check each real increment against its own component row;
	// a large total allowance cannot hide one parser or comparison crossing it.
	for _, row := range []struct {
		before, after, upper candidateWorkVector
	}{{candidateWorkVector{}, afterBase64, upper.base64}, {afterBase64, afterHTTP, upper.http},
		{afterHTTP, afterComparison, upper.comparisons}, {afterComparison, afterRPC, upper.rpc}, {afterRPC, m.actual, upper.header}} {
		for coordinate, actual := range row.after {
			if actual < row.before[coordinate] || actual-row.before[coordinate] > row.upper[coordinate] {
				t.Fatal("original response stage escaped its own coordinate", coordinate)
			}
		}
	}
}

func TestProviderHistoryFirstInterpretationFundsFixedRequestAndInitialRoot(t *testing.T) {
	upper, ok := providerHistoryFirstPInterpretationUpper(1<<20, historyFirstInterpretationCapture())
	if !ok {
		t.Fatal("finite fixed request/root quote")
	}
	m := providerHistoryContentUnitMeter(upper.request)
	request, err := newProviderIntegrationRequestObserved("eth_getBlockByNumber", []any{"finalized", false}, m)
	if err != nil || request == nil || request.method != "eth_getBlockByNumber" || request.params != `["finalized",false]` || m.failed {
		t.Fatal("fixed first-P constructor exceeded existing request envelope", err)
	}
	manifest := historyCollectionManifest(t, historyCollectionInput(t))
	selected := historyFirstFinalizedHeader()
	bounds := providerHistoryProgramBounds{program: admissionTestDigest("a"), costs: admissionTestDigest("b"), collection: manifest}
	rootMeter := providerHistoryContentUnitMeter(upper.initialRoot)
	if !manifest.selectedObserved(selected, rootMeter) {
		t.Fatal("original caller selected-header check")
	}
	root := providerHistoryInitialRootObserved(bounds, admissionTestDigest("c"), providerFrontierHighwater{frontier: admissionTestDigest("d")},
		admissionTestDigest("e"), admissionTestDigest("f"), admissionTestDigest("1"), &selected, rootMeter)
	if !lowerHash(root) || rootMeter.failed || rootMeter.actual[2] != 0 || rootMeter.actual[10] > upper.initialRoot[10] || m.actual[10] > upper.request[10] {
		t.Fatal("initial commitment exceeded its original preflight/check envelope", rootMeter.actual)
	}
}

func TestProviderHistoryFirstInterpretationRefusalPreservesEarlierWork(t *testing.T) {
	policy, _, _, _ := admissionTestIntentFixture(t, nil)
	upper, ok := providerHistoryFirstPInterpretationUpper(1<<20, historyFirstInterpretationCapture())
	if !ok {
		t.Fatal("finite normal precursor quote")
	}
	short := upper.records
	short[2] = uint64(len(policy.raw) - 1)
	m := providerHistoryContentUnitMeter(short)
	if !m.event(providerCopyEvent, 1) {
		t.Fatal("successful policy caller prefix")
	}
	if got, err := parseProviderPolicyCandidateObserved([]byte(policy.raw), 64<<10, m); err == nil || got != nil || !m.failed || m.actual != (candidateWorkVector{10: 2}) || m.event(providerCodecEvent, 0) {
		t.Fatal("policy codec shortage refunded prefix or returned candidate", err, m.actual)
	}
	encoded := base64.RawURLEncoding.EncodeToString([]byte("ordinary"))
	baseUpper, bounded := providerHistoryBase64ControlUpper(8)
	if !bounded {
		t.Fatal("normal base64 precursor quote")
	}
	fullBase := providerHistoryContentUnitMeter(baseUpper)
	if raw, valid := providerCanonicalBase64Observed(encoded, 8, fullBase); !valid || string(raw) != "ordinary" || fullBase.failed {
		t.Fatal("normal original base64 must reach its decoder/encoder")
	}
	baseUpper[2] = uint64(len(encoded) - 1)
	m = providerHistoryContentUnitMeter(baseUpper)
	if !m.event(providerCopyEvent, 1) {
		t.Fatal("successful base64 caller prefix")
	}
	if raw, valid := providerCanonicalBase64Observed(encoded, 8, m); valid || raw != nil || !m.failed || m.actual != (candidateWorkVector{10: 2}) || m.event(providerEncodeEvent, 0) {
		t.Fatal("encoded-input shortage refunded prefix or leaked decoded output", m.actual)
	}
	body := []byte(historyFirstFinalizedResponse(t))
	result, err := decodeRPCResultObserved(body, 1, RPCResponseLimits{131072, 64}, nil)
	if err != nil {
		t.Fatal("normal header precursor", err)
	}
	n := uint64(len(result))
	headerUpper, bounded := providerHistoryFirstPHeaderControlUpper(131072)
	if !bounded {
		t.Fatal("normal header quote")
	}
	fullHeader := providerHistoryContentUnitMeter(headerUpper)
	if got, err := providerHistoryFirstFinalizedHeaderObserved(result, RPCResponseLimits{131072, 64}, fullHeader); err != nil || got != historyFirstFinalizedHeader() || fullHeader.failed {
		t.Fatal("normal header must reach both original codecs", err)
	}
	headerUpper[2] = 2*n - 1
	m = providerHistoryContentUnitMeter(headerUpper)
	if !m.event(providerCopyEvent, 1) {
		t.Fatal("successful header caller prefix")
	}
	if got, err := providerHistoryFirstFinalizedHeaderObserved(result, RPCResponseLimits{131072, 64}, m); err == nil || got != (FinalizedHeader{}) || !m.failed ||
		m.actual != (candidateWorkVector{2: n, 10: 2 + 2*(n+1) + n*n + 1}) || m.event(providerCodecEvent, 0) {
		t.Fatal("second original header codec refunded its first codec/comparisons", err, m.actual)
	}
}

func TestProviderHistoryFirstInterpretationRejectsInvalidAndOverflowingQuotes(t *testing.T) {
	capture := historyFirstInterpretationCapture()
	for _, n := range []uint64{0, (8 << 20) + 1, ^uint64(0)} {
		if got, ok := providerHistoryBase64ControlUpper(n); ok || got != (candidateWorkVector{}) {
			t.Fatal("invalid base64 bound leaked quote", n, got)
		}
		if got, ok := providerHistoryFirstPHeaderControlUpper(n); ok || got != (candidateWorkVector{}) {
			t.Fatal("invalid header bound leaked quote", n, got)
		}
	}
	for _, n := range []uint64{0, (32 << 20) + 1, ^uint64(0)} {
		if got, ok := providerHistoryFirstPInterpretationUpper(n, capture); ok || got != (providerHistoryFirstPInterpretationCost{}) {
			t.Fatal("invalid original object bound leaked partial fields", n, got)
		}
	}
	if got, ok := providerHistoryFirstPInterpretationUpper(1<<20, ProviderCaptureLimits{}); ok || got != (providerHistoryFirstPInterpretationCost{}) {
		t.Fatal("invalid capture leaked fields", got)
	}
	for _, row := range [][2]uint64{{0, 1}, {1, 0}, {1, 33}, {2, 1}, {(1<<20)/3 + 1, (1<<20)/3 + 1}, {349525, 11184800}} {
		// The last valid G/R has finite decoded24132128046950 and logical
		// 28801724805834942500. Its later logical overflow must erase both.
		if got, ok := providerHistoryCollectionRepeatedFirstPInterpretationUpper(row[0], row[1], 1<<20, capture); ok || got != (candidateWorkVector{}) {
			t.Fatal("invalid or overflowing original repetition leaked partial quote", row, got)
		}
	}
}
