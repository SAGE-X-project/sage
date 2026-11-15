package registry011

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestProviderHistoryResultBytesMatchIndependentNumericalRows(t *testing.T) {
	for _, row := range []struct{ entries, maximum, built, parsed uint64 }{
		{1, 4096, 4295053415, 8590082496},
		{1, 1048576, 4300275815, 8594260416},
		{64, 1048576, 4300276004, 8594268858},
		{64, 33554432, 4462805284, 8724292282},
	} {
		got, ok := providerHistoryOriginalPackControlUpper(row.entries, row.maximum)
		want := providerHistoryPackControlCost{candidateWorkVector{10: row.built}, candidateWorkVector{2: 65540, 10: row.parsed}}
		if !ok || got != want {
			t.Fatal("independent opaque-pack row", row, got, want)
		}
	}
	for _, row := range []struct {
		header, body, wire    int
		decoded, http, member uint64
	}{
		{512, 4096, 8192, 20992, 117725721, 184998472},
		{65536, 8388608, 16777216, 42008576, 492585538617369, 492585656197192},
	} {
		capture := ProviderCaptureLimits{time.Second, row.header, row.body, row.wire, 32}
		got, ok := providerHistoryResponseControlUpper(capture)
		want := providerHistoryResponseControlCost{candidateWorkVector{2: row.decoded, 10: row.http}, candidateWorkVector{2: row.decoded, 10: row.member}}
		if !ok || got != want {
			t.Fatal("independent response/member row", row, got, want)
		}
	}
	capture := ProviderCaptureLimits{time.Second, 512, 4096, 8192, 32}
	for _, row := range []struct{ groups, requests, decoded, logical uint64 }{
		{1, 1, 432660, 65397716670},
		{1, 32, 53461508, 2060033011166},
		{2, 33, 64216144, 2328629807304},
	} {
		got, ok := providerHistoryCollectionRepeatedResultBytesUpper(row.groups, row.requests, 1048576, capture)
		if !ok || got != (candidateWorkVector{2: row.decoded, 10: row.logical}) {
			t.Fatal("independent current-build/prior-parse/member composition", row, got)
		}
	}
}

func TestProviderHistoryPackControlCostsFundOpaqueOriginalRoundTrips(t *testing.T) {
	for _, purpose := range []string{"requests", "results"} {
		for _, count := range []int{1, providerHistoryGroupMembers} {
			original, inputs := historyOriginalPackFixture(t, purpose, 0, count)
			before := *original
			beforeInputs := append([]providerHistoryOriginalInput(nil), inputs...)
			maximum := uint64(1 << 20)
			if count == 1 {
				maximum = 4096 // Descriptor preflight is STILL 64KiB, not this cap.
			}
			upper, ok := providerHistoryOriginalPackControlUpper(uint64(len(inputs)), maximum)
			if !ok {
				t.Fatal("finite original pack quote")
			}
			built := providerHistoryContentUnitMeter(upper.built)
			got, err := newProviderHistoryOriginalPackObserved(original.stage, original.table, purpose, 0, count, inputs, maximum, built)
			if err != nil || got == nil || *got != before || built.failed {
				t.Fatal("bounded opaque construction changed original bytes", purpose, count, err)
			}
			parsed := providerHistoryContentUnitMeter(upper.parsed)
			checked, err := parseProviderHistoryOriginalPackObserved(providerReplayBytes(original.raw, parsed), original.digest, original.stage, original.table, purpose, 0, count, maximum, parsed)
			if err != nil || checked == nil || *checked != before || parsed.failed || *original != before {
				t.Fatal("bounded opaque parsing changed original bytes", purpose, count, err)
			}
			for i, input := range inputs {
				kind, raw, found := checked.original(i)
				if !found || kind != input.kind || raw != input.raw || input != beforeInputs[i] {
					t.Fatal("original payload was reencoded or replaced", purpose, count, i)
				}
			}
			for _, row := range []struct {
				meter *providerWorkMeter
				upper candidateWorkVector
			}{{built, upper.built}, {parsed, upper.parsed}} {
				for i, actual := range row.meter.actual {
					if actual > row.upper[i] {
						t.Fatal("opaque-pack event exceeds quote", i)
					}
				}
			}
		}
	}
}

func TestProviderHistoryResponseControlCostsFundNormalHTTPFraming(t *testing.T) {
	capture := ProviderCaptureLimits{time.Second, 512, 4096, 8192, 32}
	upper, ok := providerHistoryResponseControlUpper(capture)
	if !ok {
		t.Fatal("finite in-memory response quote")
	}
	for _, body := range []string{`{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`, `{"jsonrpc":"\u0032\u002e\u0030","id":1,"result":"0xaa36a7"}`} {
		for _, chunked := range []bool{false, true} {
			head := "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n"
			wire := head + "Content-Length: " + strconv.Itoa(len(body)) + "\r\n\r\n" + body
			if chunked {
				wire = head + "Transfer-Encoding: chunked\r\n\r\n" + strconv.FormatInt(int64(len(body)), 16) + "\r\n" + body + "\r\n0\r\n\r\n"
			}
			before := wire
			m := providerHistoryContentUnitMeter(upper.http)
			gotHead, fields, gotBody, result, err := readProviderHTTPObserved(bufio.NewReader(strings.NewReader(wire)), capture, m, uint64(len(wire)))
			if err != nil || len(gotHead) == 0 || len(fields) == 0 || string(gotBody) != body || string(result) != `"0xaa36a7"` || m.failed || wire != before {
				t.Fatal("normal complete in-memory HTTP/RPC body changed", chunked, err)
			}
			for i, actual := range m.actual {
				if actual > upper.http[i] {
					t.Fatal("HTTP/RPC explicit event exceeds quote", i)
				}
			}
		}
	}
}

func TestProviderHistoryResultBytesComposeWithIndependentTLSAndContextRows(t *testing.T) {
	for _, count := range []int{1, providerHistoryGroupMembers} {
		ticket, consumption, inputs, roots, certificates := historyResultDataFixture(t, count)
		ordinal := count - 1
		capture := ticket.table.members[ordinal].capture
		response, responseOK := providerHistoryResponseControlUpper(capture)
		tlsCost, tlsOK := providerHistoryTLSControlUpper(uint64(count), certificates)
		if !responseOK || !tlsOK {
			t.Fatal("finite independent member components")
		}
		var work candidateWorkVector
		for i := range work {
			var ok bool
			work[i], ok = providerUpperSum(response.member[i], tlsCost.parsed[i])
			if !ok {
				t.Fatal("finite member component sum")
			}
		}
		m := providerHistoryContentUnitMeter(work)
		got, err := parseProviderHistoryMemberOriginalsObserved(inputs[2*ordinal].raw, inputs[2*ordinal+1].raw, ticket, consumption, ordinal, roots, certificates, m)
		if err != nil || !lowerHash(got.digest) || got.result != `"0xaa36a7"` || got.response != `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}` || m.failed {
			t.Fatal("full original member parser was unfunded", count, err)
		}
		for i, actual := range m.actual {
			if actual > work[i] {
				t.Fatal("full member event exceeds composed quote", count, i)
			}
		}
	}
	// One normal full pack connects descriptor, own context and every member.
	// The observation subject is inert DATA, not a transport or Source owner.
	ticket, consumption, inputs, roots, certificates := historyResultDataFixture(t, 3)
	pack, err := newProviderHistoryOriginalPackObserved(ticket.table.stage, ticket.table.digest, "results", 0, 3, inputs, 1<<20, nil)
	if err != nil {
		t.Fatal("normal full-pack precursor", err)
	}
	packCost, packOK := providerHistoryOriginalPackControlUpper(6, 1<<20)
	record, recordOK := providerHistoryRecordControlUpper(3)
	tlsCost, tlsOK := providerHistoryTLSControlUpper(3, certificates)
	response, responseOK := providerHistoryResponseControlUpper(ticket.table.members[0].capture)
	if !packOK || !recordOK || !tlsOK || !responseOK {
		t.Fatal("finite full-pack component quotes")
	}
	var work candidateWorkVector
	for i := range work {
		member, a := providerUpperSum(tlsCost.parsed[i], response.member[i])
		members, b := providerUpperProduct(3, member)
		total, c := providerUpperSum(packCost.parsed[i], record.context[i], members)
		if !a || !b || !c {
			t.Fatal("finite full-pack composition")
		}
		work[i] = total
	}
	m := providerHistoryContentUnitMeter(work)
	checked, results, err := parseProviderHistoryResultPackObserved(providerReplayBytes(pack.raw, m), pack.digest, ticket, consumption, 3, roots, certificates, 1<<20, m)
	if err != nil || checked == nil || checked == pack || checked.raw != pack.raw || m.failed {
		t.Fatal("full observed pack parser was unfunded", err)
	}
	for i := range 3 {
		if !lowerHash(results[i].digest) || results[i].result != `"0xaa36a7"` {
			t.Fatal("full original member binding missing", i)
		}
	}
	for i, actual := range m.actual {
		if actual > work[i] {
			t.Fatal("full pack explicit event exceeds quote", i)
		}
	}
}

func TestProviderHistoryResultBytesKeepGenuineDecodeShortagePrefixes(t *testing.T) {
	pack, _ := historyOriginalPackFixture(t, "results", 0, 1)
	before := *pack
	upper, ok := providerHistoryOriginalPackControlUpper(2, 1<<20)
	if !ok {
		t.Fatal("finite pack quote")
	}
	valid := providerHistoryContentUnitMeter(upper.parsed)
	if p, err := parseProviderHistoryOriginalPackObserved(providerReplayBytes(pack.raw, valid), pack.digest, pack.stage, pack.table, "results", 0, 1, 1<<20, valid); err != nil || p == nil || valid.failed {
		t.Fatal("valid pack parser precursor", err)
	}
	raw := []byte(pack.raw)
	headerBytes := uint64(binary.BigEndian.Uint32(raw[len(providerHistoryOriginalPackMagic):]))
	short := upper.parsed
	short[2] = 4 + headerBytes - 1
	m := providerHistoryContentUnitMeter(short)
	if p, err := parseProviderHistoryOriginalPackObserved(providerReplayBytes(pack.raw, m), pack.digest, pack.stage, pack.table, "results", 0, 1, 1<<20, m); err == nil || p != nil || !m.failed || m.actual[2] != 4 || m.event(providerCopyEvent, 0) || *pack != before {
		t.Fatal("descriptor decode did not retain its successful four-byte prefix")
	}
	body := `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`
	wire := []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: " + strconv.Itoa(len(body)) + "\r\n\r\n" + body)
	beforeWire := append([]byte(nil), wire...)
	capture := ProviderCaptureLimits{time.Second, 512, 4096, 8192, 32}
	response, ok := providerHistoryResponseControlUpper(capture)
	if !ok {
		t.Fatal("finite HTTP quote")
	}
	valid = providerHistoryContentUnitMeter(response.http)
	head, _, _, _, err := readProviderHTTPObserved(bufio.NewReader(bytes.NewReader(wire)), capture, valid, uint64(len(wire)))
	if err != nil || len(head) == 0 || valid.failed {
		t.Fatal("valid normal HTTP precursor", err)
	}
	short = response.http
	prefix := uint64(len(head) + len(wire))
	short[2] = prefix + uint64(len(body)) - 1
	m = providerHistoryContentUnitMeter(short)
	gotHead, fields, gotBody, result, err := readProviderHTTPObserved(bufio.NewReader(bytes.NewReader(wire)), capture, m, uint64(len(wire)))
	if err == nil || gotHead != nil || fields != nil || gotBody != nil || result != nil || !m.failed || m.actual[2] != prefix || m.event(providerCopyEvent, 0) || !bytes.Equal(wire, beforeWire) {
		t.Fatal("RPC decode shortage did not retain header/wire codec prefix")
	}
}

func TestProviderHistoryResultByteBoundsRefuseInvalidAndOverflowingInputs(t *testing.T) {
	for _, row := range [][2]uint64{{0, 1048576}, {65, 1048576}, {1, 0}, {1, 43}, {1, 33554433}, {^uint64(0), ^uint64(0)}} {
		if got, ok := providerHistoryOriginalPackControlUpper(row[0], row[1]); ok || got != (providerHistoryPackControlCost{}) {
			t.Fatal("invalid opaque-pack bounds left a quote")
		}
	}
	capture := ProviderCaptureLimits{time.Second, 512, 4096, 8192, 32}
	for _, bad := range []ProviderCaptureLimits{{}, {time.Second, -1, 4096, 8192, 32}, {time.Second, 65537, 4096, 8192, 32}, {time.Second, 512, 0, 8192, 32}, {time.Second, 512, 8388609, 16777216, 32}, {time.Second, 512, 4096, 512, 32}, {time.Second, 512, 4096, 16777217, 32}, {time.Second, 512, 4096, 8192, 129}} {
		if got, ok := providerHistoryResponseControlUpper(bad); ok || got != (providerHistoryResponseControlCost{}) {
			t.Fatal("invalid response bounds left a quote")
		}
		if got, ok := providerHistoryCollectionRepeatedResultBytesUpper(1, 1, 1<<20, bad); ok || got != (candidateWorkVector{}) {
			t.Fatal("invalid response bounds left repeated work")
		}
	}
	for _, row := range [][3]uint64{{0, 1, 1048576}, {1, 0, 1048576}, {1, 33, 1048576}, {1, 1, 0}, {1, 1, 43}, {1, 1, 33554433}} {
		if got, ok := providerHistoryCollectionRepeatedResultBytesUpper(row[0], row[1], row[2], capture); ok || got != (candidateWorkVector{}) {
			t.Fatal("invalid result composition left partial work")
		}
	}
	maximum := ProviderCaptureLimits{time.Second, 65536, 8388608, 16777216, 128}
	if count, ok := providerHistoryCollectionResultContextCallsBound(100, 3200); !ok || count.tlsParses != 2432000 {
		t.Fatal("finite repeated-result count precursor")
	}
	if _, ok := providerHistoryOriginalPackControlUpper(64, 33554432); !ok {
		t.Fatal("finite pack precursor")
	}
	if _, ok := providerHistoryResponseControlUpper(maximum); !ok {
		t.Fatal("finite response precursor")
	}
	if got, ok := providerHistoryCollectionRepeatedResultBytesUpper(100, 3200, 33554432, maximum); ok || got != (candidateWorkVector{}) {
		t.Fatal("later logical overflow leaked an earlier finite decoded row")
	}
}
