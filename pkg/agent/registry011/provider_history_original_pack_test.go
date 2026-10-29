package registry011

import (
	"encoding/binary"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

// These are bounded framing DATA fixtures, not authentic HTTP/TLS samples,
// selected storage, custody receipts or a history execution authority.
func historyOriginalPackFixture(t *testing.T, purpose string, first, count int) (*providerHistoryOriginalPack, []providerHistoryOriginalInput) {
	t.Helper()
	n := count
	if purpose == "results" {
		n *= 2
	}
	inputs := make([]providerHistoryOriginalInput, n)
	for i := range inputs {
		inputs[i] = providerHistoryOriginalInput{providerHistoryOriginalKind(purpose, i), "original\x00\xff\r\n" + strconv.Itoa(i)}
	}
	p, err := newProviderHistoryOriginalPackObserved(1, admissionTestDigest("a"), purpose, first, count, inputs, providerHistoryOriginalPackBytes, nil)
	if err != nil {
		t.Fatal(err)
	}
	return p, inputs
}

func TestProviderHistoryOriginalPacksPreserveBytesAndDistinctOrdinals(t *testing.T) {
	for _, purpose := range []string{"requests", "results"} {
		t.Run(purpose, func(t *testing.T) {
			first, count := 0, 32
			if purpose == "results" {
				first, count = 3, 2
			}
			p, inputs := historyOriginalPackFixture(t, purpose, first, count)
			// Identical content remains two separately described ordinals.
			if purpose == "requests" {
				inputs[1].raw = inputs[0].raw
			}
			m := historyGroupCodecMeter()
			observed, err := newProviderHistoryOriginalPackObserved(p.stage, p.table, purpose, first, count, inputs, providerHistoryOriginalPackBytes, m)
			if err != nil {
				t.Fatal(err)
			}
			pure, err := newProviderHistoryOriginalPackObserved(p.stage, p.table, purpose, first, count, inputs, providerHistoryOriginalPackBytes, nil)
			if err != nil || *pure != *observed {
				t.Fatal("observation changed the packed original bytes", err)
			}
			raw := []byte(observed.raw)
			parsed, err := parseProviderHistoryOriginalPackObserved(raw, observed.digest, p.stage, p.table, purpose, first, count, providerHistoryOriginalPackBytes, m)
			if err != nil || *parsed != *observed {
				t.Fatal("original pack did not reconstruct identically", err)
			}
			for i, input := range inputs {
				kind, original, ok := parsed.original(i)
				if !ok || kind != input.kind || original != input.raw {
					t.Fatal("a raw original was normalized, replaced or reordered", i)
				}
			}
			before := parsed.raw
			raw[len(raw)-1] ^= 1
			inputs[0].raw = "changed caller value"
			if parsed.raw != before || m.actual[2] == 0 || m.actual[10] == 0 || m.actual[0] != 0 || m.actual[1] != 0 || m.actual[9] != 0 || m.clocks != 0 || m.failed {
				t.Fatal("input alias or an invented external operation", m.actual)
			}
		})
	}
}

func TestProviderHistoryOriginalPackBoundsPrecedePayloadWork(t *testing.T) {
	for _, mode := range []string{"stage-zero", "stage-large", "table", "purpose", "first-negative", "count-zero", "count-large", "range", "maximum-zero", "maximum-large", "input-count", "input-kind", "empty", "request-large", "tls-large", "payload-cap"} {
		t.Run(mode, func(t *testing.T) {
			stage, table, purpose, first, count, maximum := uint64(1), admissionTestDigest("a"), "requests", 0, 1, uint64(providerHistoryOriginalPackBytes)
			inputs := []providerHistoryOriginalInput{{"request", "small original"}}
			switch mode {
			case "stage-zero":
				stage = 0
			case "stage-large":
				stage = 1<<53 + 1
			case "table":
				table = "unbound"
			case "purpose":
				purpose = "other"
			case "first-negative":
				first = -1
			case "count-zero":
				count = 0
			case "count-large":
				count = 33
			case "range":
				first, count = 31, 2
			case "maximum-zero":
				maximum = 0
			case "maximum-large":
				maximum = providerHistoryOriginalPackBytes + 1
			case "input-count":
				inputs = nil
			case "input-kind":
				inputs[0].kind = "tls"
			case "empty":
				inputs[0].raw = ""
			case "request-large":
				inputs[0].raw = strings.Repeat("x", 16<<10+1)
			case "tls-large":
				purpose, inputs = "results", []providerHistoryOriginalInput{{"response", "small"}, {"tls", strings.Repeat("x", 64<<10+1)}}
			case "payload-cap":
				maximum = uint64(len(providerHistoryOriginalPackMagic) + 4 + 1)
			}
			m := historyGroupCodecMeter()
			if p, err := newProviderHistoryOriginalPackObserved(stage, table, purpose, first, count, inputs, maximum, m); err == nil || p != nil || m.actual != (candidateWorkVector{}) {
				t.Fatal("invalid shape reached payload hash, encode or copy", mode, m.actual)
			}
		})
	}
	p, inputs := historyOriginalPackFixture(t, "requests", 0, 1)
	m := historyGroupCodecMeter()
	maximum := uint64(len(providerHistoryOriginalPackMagic) + 4 + len(inputs[0].raw))
	if got, err := newProviderHistoryOriginalPackObserved(p.stage, p.table, p.purpose, p.first, p.count, inputs, maximum, m); err == nil || got != nil || m.actual[10] == 0 || m.actual[9] != 0 {
		t.Fatal("a descriptor outside the same object cap was published")
	}
}

// Rebuild a tiny descriptor for inert parser tests. No file, live source,
// connection, signature, ACK or execution credential is produced.
func historyOriginalHeaderForTest(p *providerHistoryOriginalPack, change func(map[string]any)) []byte {
	raw := []byte(p.raw)
	start := len(providerHistoryOriginalPackMagic) + 4
	n := int(binary.BigEndian.Uint32(raw[len(providerHistoryOriginalPackMagic):start]))
	m := contentJSON(raw[start : start+n]).(map[string]any)
	change(m)
	header := providerIntegrationJSON(m)
	out := append([]byte(nil), raw[:start]...)
	binary.BigEndian.PutUint32(out[len(providerHistoryOriginalPackMagic):], uint32(len(header)))
	out = append(out, header...)
	return append(out, raw[start+n:]...)
}

func TestProviderHistoryOriginalPackRejectsChangedDescriptorsAndPayloads(t *testing.T) {
	p, _ := historyOriginalPackFixture(t, "requests", 0, 1)
	for _, mode := range []string{"magic", "header-zero", "header-large", "header-outside", "payload", "trailing", "truncated", "format", "stage", "table", "purpose", "first", "count", "entries-type", "entries-count", "ordinal", "kind", "digest", "bytes-zero", "bytes-overflow", "bytes-outside", "row-extra", "header-extra", "expected-digest", "header-whitespace"} {
		t.Run(mode, func(t *testing.T) {
			raw := []byte(p.raw)
			switch mode {
			case "magic":
				raw[0] ^= 1
			case "header-zero":
				binary.BigEndian.PutUint32(raw[len(providerHistoryOriginalPackMagic):], 0)
			case "header-large":
				binary.BigEndian.PutUint32(raw[len(providerHistoryOriginalPackMagic):], providerHistoryOriginalPackHeaderBytes+1)
			case "header-outside":
				binary.BigEndian.PutUint32(raw[len(providerHistoryOriginalPackMagic):], uint32(len(raw)))
			case "payload":
				raw[len(raw)-1] ^= 1
			case "trailing":
				raw = append(raw, 0)
			case "truncated":
				raw = raw[:len(raw)-1]
			case "expected-digest": // Otherwise valid descriptor/payload below.
			case "header-whitespace":
				start := len(providerHistoryOriginalPackMagic) + 4
				n := binary.BigEndian.Uint32(raw[len(providerHistoryOriginalPackMagic):start])
				raw = append(append(append([]byte(nil), raw[:start]...), ' '), raw[start:]...)
				binary.BigEndian.PutUint32(raw[len(providerHistoryOriginalPackMagic):], n+1)
			default:
				raw = historyOriginalHeaderForTest(p, func(m map[string]any) {
					rows := m["entries"].([]any)
					row := rows[0].(map[string]any)
					switch mode {
					case "format":
						m["format"] = "other"
					case "stage":
						m["stage"] = "2"
					case "table":
						m["table"] = admissionTestDigest("b")
					case "purpose":
						m["purpose"] = "results"
					case "first":
						m["first"] = "1"
					case "count":
						m["count"] = "2"
					case "entries-type":
						m["entries"] = "none"
					case "entries-count":
						m["entries"] = []any{}
					case "ordinal":
						row["ordinal"] = "1"
					case "kind":
						row["kind"] = "response"
					case "digest":
						row["digest"] = admissionTestDigest("b")
					case "bytes-zero":
						row["bytes"] = "0"
					case "bytes-overflow":
						row["bytes"] = "18446744073709551616"
					case "bytes-outside":
						row["bytes"] = "16384"
					case "row-extra":
						row["extra"] = "unbound"
					case "header-extra":
						m["extra"] = "unbound"
					}
				})
			}
			// Rebind the expected content hash so that a changed descriptor
			// must fail its own grammar instead of only a stale outer digest.
			expected := providerCandidateHash(providerHistoryOriginalPackDomain, raw)
			if mode == "expected-digest" {
				expected = admissionTestDigest("c")
			}
			if got, err := parseProviderHistoryOriginalPackObserved(raw, expected, p.stage, p.table, p.purpose, p.first, p.count, providerHistoryOriginalPackBytes, historyGroupCodecMeter()); err == nil || got != nil {
				t.Fatal("changed original pack returned parsed data", mode)
			}
		})
	}
}

func TestProviderHistoryRequestPackReconstructsIndependentRequests(t *testing.T) {
	bounds, capture := historyGroupBounds()
	program := admissionTestDigest("a")
	requests := []*providerIntegrationRequest{historyGroupSimple(t), historyGroupRequest(t, "debug_traceTransaction", []any{program, map[string]any{"tracer": "stateGasTracer", "reexec": 0}})}
	table, err := newProviderHistoryGroupTable(program, "content", 1, requests, bounds, capture)
	if err != nil {
		t.Fatal(err)
	}
	inputs := []providerHistoryOriginalInput{{"request", requests[0].wire}, {"request", requests[1].wire}}
	p, err := newProviderHistoryOriginalPackObserved(1, table.digest, "requests", 0, 2, inputs, providerHistoryOriginalPackBytes, nil)
	if err != nil {
		t.Fatal(err)
	}
	m := historyGroupCodecMeter()
	original, rebuilt, err := parseProviderHistoryRequestPackObserved([]byte(p.raw), p.digest, 1, program, "content", table.digest, 2, bounds, capture, providerHistoryOriginalPackBytes, m)
	if err != nil || original == p || rebuilt == table || *rebuilt != *table || rebuilt.members[0].request != *requests[0] || rebuilt.members[1].request != *requests[1] || m.failed || m.actual[2] == 0 {
		t.Fatal("request table was supplied instead of rebuilt from HTTP originals", err)
	}
	if _, _, err := parseProviderHistoryRequestPackObserved([]byte(p.raw), p.digest, 1, program, "final", table.digest, 2, bounds, capture, providerHistoryOriginalPackBytes, historyGroupCodecMeter()); err == nil {
		t.Fatal("a different purpose reused the same table identity")
	}
	if _, _, err := parseProviderHistoryRequestPackObserved([]byte(p.raw), admissionTestDigest("b"), 1, program, "content", table.digest, 2, bounds, capture, providerHistoryOriginalPackBytes, historyGroupCodecMeter()); err == nil {
		t.Fatal("unmatched original digest reconstructed a table")
	}
	framingOnly, _ := historyOriginalPackFixture(t, "requests", 0, 1)
	if _, _, err := parseProviderHistoryRequestPackObserved([]byte(framingOnly.raw), framingOnly.digest, 1, program, "content", framingOnly.table, 1, bounds, capture, providerHistoryOriginalPackBytes, historyGroupCodecMeter()); err == nil {
		t.Fatal("framing validity substituted for HTTP request reconstruction")
	}
}

func historyOriginalRequestBodyForTest(body map[string]any) ([]byte, error) {
	// Match the existing request builder's exact field order. A sorted map
	// would make even the unmodified precursor fail the final wire equality.
	return json.Marshal(struct {
		Version any `json:"jsonrpc"`
		ID      any `json:"id"`
		Method  any `json:"method"`
		Params  any `json:"params"`
	}{body["jsonrpc"], body["id"], body["method"], body["params"]})
}

func TestProviderHistoryRequestOriginalRejectsInvalidBoundedData(t *testing.T) {
	r := historyGroupSimple(t)
	base := map[string]any{"jsonrpc": "2.0", "id": 1, "method": r.method, "params": []any{}}
	baseline, err := historyOriginalRequestBodyForTest(base)
	if err != nil || string(append(providerRequestHead(len(baseline)), baseline...)) != r.wire {
		t.Fatal("negative cases need the otherwise exact original request", err)
	}
	for _, mode := range []string{"empty", "oversized", "separator", "empty-body", "json", "version", "id", "method", "method-large", "params-null", "number", "array-large", "map-large", "deep", "nodes", "header", "body-extra"} {
		t.Run(mode, func(t *testing.T) {
			wire := []byte(r.wire)
			body := map[string]any{"jsonrpc": "2.0", "id": 1, "method": r.method, "params": []any{}}
			switch mode {
			case "empty":
				wire = nil
			case "oversized":
				wire = []byte(strings.Repeat("x", 16<<10+1))
			case "separator":
				wire = []byte("no header delimiter")
			case "empty-body":
				wire = []byte(r.head)
			case "header":
				wire[0] = 'X'
			default:
				switch mode {
				case "version":
					body["jsonrpc"] = "1.0"
				case "id":
					body["id"] = 2
				case "method":
					body["method"] = "unknown_method"
				case "method-large":
					body["method"] = strings.Repeat("m", 65)
				case "params-null":
					body["params"] = nil
				case "number":
					body["params"] = []any{1}
				case "array-large":
					body["params"] = make([]bool, 17)
				case "map-large":
					v := map[string]bool{}
					for i := range 17 {
						v[strconv.Itoa(i)] = true
					}
					body["params"] = []any{v}
				case "deep":
					var v any = true
					for range 9 {
						v = []any{v}
					}
					body["params"] = v
				case "nodes":
					outer := make([]any, 16)
					for i := range outer {
						inner := make([]any, 16)
						for j := range inner {
							inner[j] = []any{true, true}
						}
						outer[i] = inner
					}
					body["params"] = outer
				case "body-extra":
					body["extra"] = true
				}
				raw, err := historyOriginalRequestBodyForTest(body)
				if err != nil {
					t.Fatal(err)
				}
				if mode == "body-extra" {
					raw = append(raw[:len(raw)-1], []byte(",\"extra\":true}")...)
				}
				if mode == "json" {
					raw = []byte("{")
				}
				wire = append(providerRequestHead(len(raw)), raw...)
			}
			if got, err := providerHistoryRequestOriginalObserved(wire, historyGroupCodecMeter()); err == nil || got != nil {
				t.Fatal("invalid raw request became a rebuilt request", mode)
			}
		})
	}
}

func TestProviderHistoryRequestNormalizationBoundsBeforeBuilder(t *testing.T) {
	array := func(n int) []any {
		v := make([]any, n)
		for i := range v {
			v[i] = true
		}
		return v
	}
	object := func(n int) map[string]any {
		v := make(map[string]any, n)
		for i := range n {
			v[strconv.Itoa(i)] = true
		}
		return v
	}
	deep := func(n int) any {
		var v any = true
		for range n {
			v = []any{v}
		}
		return v
	}
	wide := make([]any, 16)
	for i := range wide {
		inner := make([]any, 16)
		for j := range inner {
			inner[j] = []any{true, true}
		}
		wide[i] = inner
	}
	// Call the walker directly: a later request builder must not mask a
	// missing guard here. Compare work is exactly two units per visited node;
	// an out-of-range depth/node is refused before that node's first event.
	for _, tc := range []struct {
		name                           string
		value                          any
		depth, initialNodes, wantNodes int
		upper, wantWork                uint64
		wantOK, wantFailed             bool
	}{
		{"array-overflow", array(17), 0, 0, 1, 1 << 60, 2, false, false},
		{"map-overflow", object(17), 0, 0, 1, 1 << 60, 2, false, false},
		{"depth-before-event", true, 9, 0, 1, 1 << 60, 0, false, false},
		{"nodes-before-event", true, 0, 512, 513, 1 << 60, 0, false, false},
		{"depth-recursive", deep(9), 0, 0, 10, 1 << 60, 18, false, false},
		{"nodes-recursive", wide, 0, 0, 513, 1 << 60, 1024, false, false},
		{"array-at-bound", array(16), 0, 0, 17, 1 << 60, 34, true, false},
		{"map-at-bound", object(16), 0, 0, 17, 1 << 60, 34, true, false},
		{"depth-at-bound", deep(8), 0, 0, 9, 1 << 60, 18, true, false},
		{"nodes-at-bound", float64(0), 0, 511, 512, 1 << 60, 2, true, false},
		{"observer-before-event", true, 0, 0, 1, 1, 0, false, true},
		{"observer-after-parent", []any{true}, 0, 0, 2, 2, 2, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := historyGroupCodecMeter()
			m.upper[10] = tc.upper
			nodes := tc.initialNodes
			got, ok := providerHistoryRequestNumbersObserved(tc.value, tc.depth, &nodes, m)
			var want candidateWorkVector
			want[10] = tc.wantWork
			if ok != tc.wantOK || nodes != tc.wantNodes || m.actual != want || m.failed != tc.wantFailed || m.closed || m.clocks != 0 {
				t.Fatal("walker guard or observed prefix changed", ok, nodes, m.actual, m.failed)
			}
			if !ok && got != nil {
				t.Fatal("refused normalization returned partial data")
			}
			if tc.name == "nodes-at-bound" && got != int(0) {
				t.Fatal("the accepted last node was not normalized")
			}
		})
	}
}

func TestProviderHistoryOriginalPackObservationAndViewsRefuseInvalidState(t *testing.T) {
	p, inputs := historyOriginalPackFixture(t, "requests", 0, 1)
	for _, mode := range []string{"exhausted", "closed", "copied", "decode"} {
		t.Run(mode, func(t *testing.T) {
			m := historyGroupCodecMeter()
			switch mode {
			case "exhausted":
				m.upper[10] = 0
			case "closed":
				m.closed = true
			case "copied":
				m = &providerWorkMeter{owner: m, integration: m.integration, upper: m.upper}
			case "decode":
				m.upper[2] = 0
			}
			if mode != "decode" {
				if got, err := newProviderHistoryOriginalPackObserved(p.stage, p.table, p.purpose, p.first, p.count, inputs, providerHistoryOriginalPackBytes, m); err == nil || got != nil {
					t.Fatal("failed observer built a pack")
				}
			}
			if got, err := parseProviderHistoryOriginalPackObserved([]byte(p.raw), p.digest, p.stage, p.table, p.purpose, p.first, p.count, providerHistoryOriginalPackBytes, m); err == nil || got != nil {
				t.Fatal("failed observer parsed a pack")
			}
		})
	}
	var absent *providerHistoryOriginalPack
	if _, _, ok := absent.original(0); ok {
		t.Fatal("absent original view")
	}
	for _, mode := range []string{"negative", "outside", "entries", "offset", "size"} {
		copy := *p
		index := 0
		switch mode {
		case "negative":
			index = -1
		case "outside":
			index = p.entries
		case "entries":
			copy.entries = 65
		case "offset":
			copy.records[0].offset = len(copy.raw) + 1
		case "size":
			copy.records[0].size = len(copy.raw) + 1
		}
		if _, _, ok := copy.original(index); ok {
			t.Fatal("invalid private original view", mode)
		}
	}
	if providerHistoryOriginalCopy(make([]byte, 1), "two", nil) || json.Unmarshal([]byte(`{}`), &providerHistoryOriginalPack{}) == nil {
		t.Fatal("invalid copy or JSON reconstructed pack")
	}
}
