package registry011

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"strconv"
	"testing"
)

// Pure finite DATA construction only: none of these fabricated diagnostic
// fields selects a TLS connection, live permit, C owner, Source or S ACK.
func historyResultDataFixture(t *testing.T, count int) (*providerHistoryGroupTicket, *providerHistoryGroupConsumption, []providerHistoryOriginalInput, string, providerTLSBounds) {
	t.Helper()
	ticket := historyRecordTicket(t, count, 1, admissionTestDigest("e"))
	consumption, err := newProviderHistoryGroupConsumption(ticket, admissionTestDigest("f"), admissionTestDigest("1"))
	if err != nil {
		t.Fatal(err)
	}
	roots, bounds := admissionTestDigest("2"), providerTLSBounds{1, 1, 8 << 10}
	inputs := make([]providerHistoryOriginalInput, 0, 2*count)
	for i := range count {
		r := &ticket.table.members[i].request
		body := `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`
		response := "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: " + strconv.Itoa(len(body)) + "\r\nConnection: close\r\n\r\n" + body
		sample := &ProviderTransportSample{Endpoint: SepoliaRPCURL, Method: r.method, HTTPVersion: "HTTP/1.1", RequestHead: []byte(r.head), RequestBody: []byte(r.body), Complete: true, ServerName: "rpc.sepolia.ethpandaops.io", TLSVersion: tls.VersionTLS13, CipherSuite: tls.TLS_AES_128_GCM_SHA256, VerifiedChains: [][]string{{admissionTestDigest("3")}}}
		metadata, err := newProviderHistoryTLSOriginalObserved(ticket, consumption, i, sample, roots, bounds, nil)
		if err != nil {
			t.Fatal(err)
		}
		inputs = append(inputs, providerHistoryOriginalInput{"response", response}, providerHistoryOriginalInput{"tls", metadata})
	}
	return ticket, consumption, inputs, roots, bounds
}

func TestProviderHistoryResultsBindExactOriginalsToEveryOrdinal(t *testing.T) {
	for _, count := range []int{1, 3, providerHistoryGroupMembers} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			ticket, consumption, inputs, roots, bounds := historyResultDataFixture(t, count)
			pack, err := newProviderHistoryOriginalPackObserved(ticket.table.stage, ticket.table.digest, "results", 0, count, inputs, providerHistoryOriginalPackBytes, nil)
			if err != nil {
				t.Fatal(err)
			}
			meter := historyGroupCodecMeter()
			parsed, results, err := parseProviderHistoryResultPackObserved([]byte(pack.raw), pack.digest, ticket, consumption, count, roots, bounds, providerHistoryOriginalPackBytes, meter)
			if err != nil || parsed == nil || parsed.raw != pack.raw || parsed.digest != pack.digest || meter.failed || meter.actual[2] == 0 || meter.actual[10] == 0 || meter.actual[0] != 0 || meter.actual[1] != 0 || meter.actual[9] != 0 || meter.clocks != 0 {
				t.Fatal("original pack framing and DATA replay", err)
			}
			seen := map[string]bool{}
			for i := range count {
				if !lowerHash(results[i].digest) || seen[results[i].digest] || results[i].result != `"0xaa36a7"` || results[i].response != `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}` {
					t.Fatal("even identical response bytes must bind distinct original member ordinals", i)
				}
				seen[results[i].digest] = true
			}
			for i := count; i < len(results); i++ {
				if results[i] != (providerHistoryMemberResult{}) {
					t.Fatal("unused result tail contains producer data", i)
				}
			}
		})
	}
}

func TestProviderHistoryResultsReplayAnOwnedNormalTLSExchange(t *testing.T) {
	ticket, consumption, _, roots, bounds := historyResultDataFixture(t, 1)
	member := &ticket.table.members[0]
	conn := providerIntegrationOwnedTLS(t, member.request.body, `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`)
	meter, _ := providerFirstCaptureMeter(t, &member.request, member.capture, bounds)
	ctx, cancel := context.WithTimeout(context.Background(), member.capture.Timeout)
	defer cancel()
	sample, err := providerFirstOperationCapture(ctx, conn, member.capture, &member.request, meter, bounds)
	if err != nil || sample == nil || !sample.Complete {
		t.Fatal("normal owned TLS byte capture is mandatory", err)
	}
	// This closed normal primitive is only the origin of unit DATA here. It
	// is not passed off as the unfinished history owner's future permit.
	replay := historyGroupCodecMeter()
	metadata, err := newProviderHistoryTLSOriginalObserved(ticket, consumption, 0, sample, roots, bounds, replay)
	if err != nil {
		t.Fatal(err)
	}
	pack, err := newProviderHistoryOriginalPackObserved(ticket.table.stage, ticket.table.digest, "results", 0, 1,
		[]providerHistoryOriginalInput{{"response", string(sample.ResponseWire)}, {"tls", metadata}}, providerHistoryOriginalPackBytes, replay)
	if err != nil {
		t.Fatal(err)
	}
	_, result, err := parseProviderHistoryResultPackObserved([]byte(pack.raw), pack.digest, ticket, consumption, 1, roots, bounds, providerHistoryOriginalPackBytes, replay)
	if err != nil || result[0].response != string(sample.ResponseBody) || result[0].result != string(sample.Result) || !lowerHash(result[0].digest) || replay.failed || replay.actual[0] != 0 || replay.actual[1] != 0 {
		t.Fatal("retained exact response and TLS DATA differ from the owned normal exchange", err)
	}
}

func TestProviderHistoryResultsRefuseDifferentBindingsAndIncompleteOriginals(t *testing.T) {
	for _, mode := range []string{"ticket", "consumption", "root", "ordinal", "request", "cipher", "empty-chain", "unknown-field", "prefix-count", "missing-response", "truncated-response", "changed-pack-digest"} {
		t.Run(mode, func(t *testing.T) {
			ticket, consumption, inputs, roots, bounds := historyResultDataFixture(t, 2)
			count := 2
			switch mode {
			case "prefix-count":
				count = 1
			case "missing-response":
				inputs[0].raw = ""
			case "truncated-response":
				inputs[0].raw = inputs[0].raw[:len(inputs[0].raw)-1]
			case "changed-pack-digest":
			default:
				var fields map[string]any
				if err := json.Unmarshal([]byte(inputs[1].raw), &fields); err != nil {
					t.Fatal(err)
				}
				switch mode {
				case "ticket", "consumption", "request":
					fields[mode] = admissionTestDigest("9")
				case "root":
					fields["roots_artifact"] = admissionTestDigest("9")
				case "ordinal":
					fields["ordinal"] = "1"
				case "cipher":
					fields["cipher_suite"] = "0"
				case "empty-chain":
					fields["verified_chains"] = []any{}
				case "unknown-field":
					fields["inert_extra_unit_field"] = "inert"
				}
				inputs[1].raw = string(providerReplayCanonical(fields, 64<<10, nil))
			}
			pack, err := newProviderHistoryOriginalPackObserved(ticket.table.stage, ticket.table.digest, "results", 0, 2, inputs, providerHistoryOriginalPackBytes, nil)
			if mode == "missing-response" {
				if err == nil || pack != nil {
					t.Fatal("missing original entered pack")
				}
				return
			}
			if err != nil {
				t.Fatal("bounded inert unit input pack precursor", err)
			}
			expected := pack.digest
			if mode == "changed-pack-digest" {
				expected = admissionTestDigest("9")
			}
			parsed, results, err := parseProviderHistoryResultPackObserved([]byte(pack.raw), expected, ticket, consumption, count, roots, bounds, providerHistoryOriginalPackBytes, historyGroupCodecMeter())
			if err == nil || parsed != nil || results != ([providerHistoryGroupMembers]providerHistoryMemberResult{}) {
				t.Fatal("different original context or incomplete response returned successful DATA")
			}
		})
	}
}

func TestProviderHistoryResultsKeepObserverRefusalSeparateFromPureData(t *testing.T) {
	ticket, consumption, inputs, roots, bounds := historyResultDataFixture(t, 1)
	pack, err := newProviderHistoryOriginalPackObserved(ticket.table.stage, ticket.table.digest, "results", 0, 1, inputs, providerHistoryOriginalPackBytes, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"closed", "copied", "work-exhausted", "decode-exhausted"} {
		t.Run(mode, func(t *testing.T) {
			meter := historyGroupCodecMeter()
			switch mode {
			case "closed":
				meter.closed = true
			case "copied":
				meter = &providerWorkMeter{owner: meter, integration: meter.integration, upper: meter.upper}
			case "work-exhausted":
				meter.upper[10] = 0
			case "decode-exhausted":
				meter.upper[2] = 0
			}
			if parsed, results, err := parseProviderHistoryResultPackObserved([]byte(pack.raw), pack.digest, ticket, consumption, 1, roots, bounds, providerHistoryOriginalPackBytes, meter); err == nil || parsed != nil || results != ([providerHistoryGroupMembers]providerHistoryMemberResult{}) {
				t.Fatal("unfunded observer fell back to pure replay")
			}
		})
	}
}

func TestProviderHistoryResultsReplayASelectedCompletedPrefixAsDataOnly(t *testing.T) {
	ticket, consumption, inputs, roots, bounds := historyResultDataFixture(t, 3)
	pack, err := newProviderHistoryOriginalPackObserved(ticket.table.stage, ticket.table.digest, "results", 0, 2, inputs[:4], providerHistoryOriginalPackBytes, nil)
	if err != nil {
		t.Fatal(err)
	}
	parsed, result, err := parseProviderHistoryResultPackObserved([]byte(pack.raw), pack.digest, ticket, consumption, 2, roots, bounds, providerHistoryOriginalPackBytes, historyGroupCodecMeter())
	if err != nil || parsed == nil || parsed.count != 2 || !lowerHash(result[0].digest) || !lowerHash(result[1].digest) || result[2] != (providerHistoryMemberResult{}) {
		t.Fatal("completed prefix DATA is distinct from all-member successful S", err)
	}
	// A complete settlement still requires the entire T count. Two valid
	// original results do not make the third, unsent member complete.
	if s, err := newProviderHistoryGroupSettlement(ticket, consumption, admissionTestDigest("4"), admissionTestDigest("5"), "complete", "", []string{result[0].digest, result[1].digest}); err == nil || s != nil {
		t.Fatal("partial original prefix became a successful complete S")
	}
}

func TestProviderHistoryTLSDataConstructionRefusesInvalidFiniteInputs(t *testing.T) {
	for _, mode := range []string{"ticket", "consumption", "ordinal", "sample", "incomplete", "capture-error", "endpoint", "method", "request-head", "request-body", "server-name", "http-version", "roots", "bounds", "tls-version", "cipher", "no-chains", "too-many-chains", "empty-chain", "too-many-certificates", "bad-digest", "observer"} {
		t.Run(mode, func(t *testing.T) {
			ticket, consumption, _, roots, bounds := historyResultDataFixture(t, 1)
			request := &ticket.table.members[0].request
			sample := &ProviderTransportSample{Endpoint: SepoliaRPCURL, Method: request.method, HTTPVersion: "HTTP/1.1", RequestHead: []byte(request.head), RequestBody: []byte(request.body), Complete: true, ServerName: "rpc.sepolia.ethpandaops.io", TLSVersion: tls.VersionTLS13, CipherSuite: tls.TLS_AES_128_GCM_SHA256, VerifiedChains: [][]string{{admissionTestDigest("3")}}}
			meter, ordinal := historyGroupCodecMeter(), 0
			switch mode {
			case "ticket":
				ticket = nil
			case "consumption":
				consumption = nil
			case "ordinal":
				ordinal = 1
			case "sample":
				sample = nil
			case "incomplete":
				sample.Complete = false
			case "capture-error":
				sample.CaptureError = "inert"
			case "endpoint":
				sample.Endpoint = "inert-unit-data"
			case "method":
				sample.Method = "inert-unit-data"
			case "request-head":
				sample.RequestHead = nil
			case "request-body":
				sample.RequestBody = nil
			case "server-name":
				sample.ServerName = "inert-unit-data"
			case "http-version":
				sample.HTTPVersion = "inert-unit-data"
			case "roots":
				roots = "inert-unit-data"
			case "bounds":
				bounds.chains = 0
			case "tls-version":
				sample.TLSVersion = tls.VersionTLS11
			case "cipher":
				sample.CipherSuite = 0
			case "no-chains":
				sample.VerifiedChains = nil
			case "too-many-chains":
				sample.VerifiedChains = [][]string{{admissionTestDigest("3")}, {admissionTestDigest("4")}}
			case "empty-chain":
				sample.VerifiedChains = [][]string{nil}
			case "too-many-certificates":
				sample.VerifiedChains = [][]string{{admissionTestDigest("3"), admissionTestDigest("4")}}
			case "bad-digest":
				sample.VerifiedChains = [][]string{{"inert-unit-data"}}
			case "observer":
				meter.upper[10] = 0
			}
			if raw, err := newProviderHistoryTLSOriginalObserved(ticket, consumption, ordinal, sample, roots, bounds, meter); err == nil || raw != "" {
				t.Fatal("invalid finite TLS DATA returned a record")
			}
		})
	}
}

func TestProviderHistoryTLSOriginalParserRefusesNoncanonicalAndUnboundData(t *testing.T) {
	for _, mode := range []string{"empty", "bounds", "roots", "context", "noncanonical", "chain-type", "chain-empty", "chain-count", "certificate-type", "certificate-count", "digest"} {
		t.Run(mode, func(t *testing.T) {
			ticket, consumption, inputs, roots, bounds := historyResultDataFixture(t, 1)
			raw := []byte(inputs[1].raw)
			var fields map[string]any
			if err := json.Unmarshal(raw, &fields); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "empty":
				raw = nil
			case "bounds":
				bounds.chains = 0
			case "roots":
				roots = "inert-unit-data"
			case "context":
				ticket = nil
			case "noncanonical":
				raw = append([]byte("\n"), raw...)
			case "chain-type":
				fields["verified_chains"] = "inert-unit-data"
			case "chain-empty":
				fields["verified_chains"] = []any{}
			case "chain-count":
				fields["verified_chains"] = []any{[]any{admissionTestDigest("3")}, []any{admissionTestDigest("4")}}
			case "certificate-type":
				fields["verified_chains"] = []any{"inert-unit-data"}
			case "certificate-count":
				fields["verified_chains"] = []any{[]any{admissionTestDigest("3"), admissionTestDigest("4")}}
			case "digest":
				fields["verified_chains"] = []any{[]any{"inert-unit-data"}}
			}
			if mode == "chain-type" || mode == "chain-empty" || mode == "chain-count" || mode == "certificate-type" || mode == "certificate-count" || mode == "digest" {
				raw = providerReplayCanonical(fields, 64<<10, nil)
			}
			if err := parseProviderHistoryTLSOriginalObserved(raw, ticket, consumption, 0, roots, bounds, historyGroupCodecMeter()); err == nil {
				t.Fatal("unbound or noncanonical TLS DATA was accepted")
			}
		})
	}
}

func TestProviderHistoryResultsBindDifferentOriginalMethodsWithoutInterchangingTLS(t *testing.T) {
	original := historyRecordTicket(t, 2, 1, admissionTestDigest("e"))
	bounds, capture := historyGroupBounds()
	requests := []*providerIntegrationRequest{historyGroupSimple(t), historyGroupRequest(t, "web3_clientVersion", []any{})}
	table, err := newProviderHistoryGroupTable(original.program, "identity", 1, requests, bounds, capture)
	if err != nil {
		t.Fatal(err)
	}
	ticket, err := newProviderHistoryGroupTicket(original.program, original.profile, original.costs, original.previous, original.inventory, table, original.upper)
	if err != nil {
		t.Fatal(err)
	}
	consumption, err := newProviderHistoryGroupConsumption(ticket, admissionTestDigest("f"), admissionTestDigest("1"))
	if err != nil {
		t.Fatal(err)
	}
	roots, tlsBounds := admissionTestDigest("2"), providerTLSBounds{1, 1, 8 << 10}
	var inputs []providerHistoryOriginalInput
	for i, r := range requests {
		result := `"0xaa36a7"`
		if i == 1 {
			result = `"normal-client-version"`
		}
		body := `{"jsonrpc":"2.0","id":1,"result":` + result + `}`
		response := "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: " + strconv.Itoa(len(body)) + "\r\nConnection: close\r\n\r\n" + body
		sample := &ProviderTransportSample{Endpoint: SepoliaRPCURL, Method: r.method, HTTPVersion: "HTTP/1.1", RequestHead: []byte(r.head), RequestBody: []byte(r.body), Complete: true, ServerName: "rpc.sepolia.ethpandaops.io", TLSVersion: tls.VersionTLS13, CipherSuite: tls.TLS_AES_128_GCM_SHA256, VerifiedChains: [][]string{{admissionTestDigest("3")}}}
		metadata, err := newProviderHistoryTLSOriginalObserved(ticket, consumption, i, sample, roots, tlsBounds, nil)
		if err != nil {
			t.Fatal(err)
		}
		inputs = append(inputs, providerHistoryOriginalInput{"response", response}, providerHistoryOriginalInput{"tls", metadata})
	}
	pack, err := newProviderHistoryOriginalPackObserved(1, table.digest, "results", 0, 2, inputs, providerHistoryOriginalPackBytes, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, results, err := parseProviderHistoryResultPackObserved([]byte(pack.raw), pack.digest, ticket, consumption, 2, roots, tlsBounds, providerHistoryOriginalPackBytes, historyGroupCodecMeter())
	if err != nil || results[0].result != `"0xaa36a7"` || results[1].result != `"normal-client-version"` || results[0].digest == results[1].digest {
		t.Fatal("different original method/result bindings", err)
	}
	// Inert finite DATA only. No peer or runtime response is modified.
	inputs[1], inputs[3] = inputs[3], inputs[1]
	changed, err := newProviderHistoryOriginalPackObserved(1, table.digest, "results", 0, 2, inputs, providerHistoryOriginalPackBytes, nil)
	if err != nil {
		t.Fatal(err)
	}
	if p, result, err := parseProviderHistoryResultPackObserved([]byte(changed.raw), changed.digest, ticket, consumption, 2, roots, tlsBounds, providerHistoryOriginalPackBytes, historyGroupCodecMeter()); err == nil || p != nil || result != ([providerHistoryGroupMembers]providerHistoryMemberResult{}) {
		t.Fatal("another original method/ordinal TLS binding was accepted as this member")
	}
}

func TestProviderHistoryTLSDataAlwaysChargesItsOriginalRequestCopiesAndComparisons(t *testing.T) {
	ticket, consumption, _, roots, bounds := historyResultDataFixture(t, 1)
	request := &ticket.table.members[0].request
	control := historyGroupCodecMeter()
	if !providerHistoryResultContextObserved(ticket, consumption, 0, control) || control.failed {
		t.Fatal("independently funded T/C DATA precursor")
	}
	baseline := control.actual
	head, body := uint64(len(request.head)), uint64(len(request.body))
	for _, mode := range []string{"head-mismatch", "body-mismatch", "head-copy-quota", "head-compare-quota"} {
		t.Run(mode, func(t *testing.T) {
			sample := &ProviderTransportSample{Endpoint: SepoliaRPCURL, Method: request.method, HTTPVersion: "HTTP/1.1", RequestHead: []byte(request.head), RequestBody: []byte(request.body), Complete: true, ServerName: "rpc.sepolia.ethpandaops.io", TLSVersion: tls.VersionTLS13, CipherSuite: tls.TLS_AES_128_GCM_SHA256, VerifiedChains: [][]string{{admissionTestDigest("3")}}}
			meter, expected := historyGroupCodecMeter(), baseline
			switch mode {
			case "head-mismatch":
				sample.RequestHead[0] = 'X' // Inert finite DATA, never written.
				expected[10] += (head + 1) + (2*head + 1)
			case "body-mismatch":
				sample.RequestBody[0] = 'X'
				expected[10] += (head + 1) + (2*head + 1) + (body + 1) + (2*body + 1)
			case "head-copy-quota":
				meter.upper[10] = head
			case "head-compare-quota":
				meter.upper[10] = (head + 1) + 2*head
			}
			if mode == "head-copy-quota" || mode == "head-compare-quota" {
				// Test this byte primitive in isolation. Its caller's earlier
				// context encoder has a larger preflight ceiling; pretending
				// a small constructor-wide cap reached this later copy would
				// incorrectly substitute an early refusal for the target.
				expected = candidateWorkVector{}
				if mode == "head-compare-quota" {
					expected[10] = head + 1
				}
				if providerHistoryRequestBytesMatchObserved(sample, request, meter) || !meter.failed || meter.actual != expected {
					t.Fatal("unfunded original byte primitive skipped its copy/comparison boundary", mode, meter.actual, expected)
				}
				return
			}
			if raw, err := newProviderHistoryTLSOriginalObserved(ticket, consumption, 0, sample, roots, bounds, meter); err == nil || raw != "" || meter.actual != expected || meter.failed {
				t.Fatal("new history DATA omitted an original copy/comparison or confused semantic refusal with budget failure", mode, meter.actual, expected)
			}
		})
	}
}
