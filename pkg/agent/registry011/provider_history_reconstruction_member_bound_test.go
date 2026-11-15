package registry011

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func historyReconstructionMemberMethods(c ProviderCaptureLimits) providerMethodBounds {
	l := providerMethodLimit{c.BodyBytes, c.WireBytes}
	return providerMethodBounds{l, l, l, l, l, l, l, l}
}

func TestProviderHistoryReconstructionMembersMatchOriginalEventLedger(t *testing.T) {
	for _, row := range []struct {
		name                                                          string
		groups, requests, metadata, maximum, policy, head, body, wire uint64
		chains, certificates                                          int
		failed                                                        bool
		calls                                                         [6]uint64
		parts                                                         [8][2]uint64
		normal                                                        [2]uint64
	}{
		{"small", 1, 1, 4096, 65536, 1024, 1024, 4096, 16384, 1, 1, false, [6]uint64{2, 2, 2, 0, 0, 4}, [8][2]uint64{
			{131080, 17180673396}, {4194304, 155569023660}, {262144, 34360524816}, {118784, 1548538144},
			{0, 1151396}, {4096, 4198408}, {0, 0}, {0, 76},
		}, [2]uint64{4710408, 208664109896}},
		{"medium", 3, 65, 65536, 2097152, 8192, 8192, 131072, 1048576, 2, 4, true, [6]uint64{6, 130, 64, 2, 2, 76}, [8][2]uint64{
			{393240, 51590778972}, {79691776, 2955811449540}, {12845056, 1683665715984}, {284196864, 225632962164512},
			{0, 729753540}, {81920, 671170580}, {32776, 1105692492}, {0, 39456226},
		}, [2]uint64{377241632, 230326576181846}},
		{"maximum", 16, 512, 1048576, 33554432, 65536, 65536, 8388608, 16777216, 8, 16, true, [6]uint64{32, 1024, 64, 2, 2, 102}, [8][2]uint64{
			{2097280, 279177353024}, {106954752, 3967010103330}, {71434240, 9363243012360}, {45789347840, 536918365254939280},
			{0, 71183413682}, {2359296, 154621182024}, {32776, 1577567772}, {0, 629280226},
		}, [2]uint64{45972226184, 536932202696851698}},
	} {
		t.Run(row.name, func(t *testing.T) {
			capture := ProviderCaptureLimits{time.Second, int(row.head), int(row.body), int(row.wire), 64}
			methods, certificates := historyReconstructionMemberMethods(capture), providerTLSBounds{row.chains, row.certificates, 8192}
			before, selected, original := capture, methods, certificates
			got, ok := providerHistoryReconstructionMembersUpper(row.groups, row.requests, row.metadata, row.maximum, row.policy, capture, methods, certificates, row.failed)
			if !ok || capture != before || methods != selected || certificates != original ||
				[6]uint64{got.packParses, got.settled, got.completed, got.attempts, got.failureParses, got.contexts} != row.calls ||
				got.normal != (candidateWorkVector{2: row.normal[0], 10: row.normal[1]}) || got.envelope != got.normal || got.incoming != (candidateWorkVector{}) {
				t.Fatal("fresh original member contexts, full guards or shared pack scope changed", got, ok)
			}
			for i, part := range [...]candidateWorkVector{got.packs, got.context, got.tls, got.response, got.custody, got.policy, got.failure, got.tail} {
				if part != (candidateWorkVector{2: row.parts[i][0], 10: row.parts[i][1]}) {
					t.Fatal("independently enumerated original event component", i, part, row.parts[i])
				}
			}
		})
	}
}

func TestProviderHistoryReconstructionMembersRefuseUnselectedBounds(t *testing.T) {
	for _, mode := range []string{"groups-zero", "groups-large", "requests-low", "requests-large", "metadata-zero", "metadata-large", "object-zero", "object-small", "object-large", "policy-zero", "policy-large", "timeout", "header", "body", "wire", "depth", "method-zero", "method-wide", "tls-zero", "chains", "certificate-size", "overflow"} {
		t.Run(mode, func(t *testing.T) {
			g, r, d, p, k := uint64(3), uint64(65), uint64(65536), uint64(2<<20), uint64(8192)
			capture := historyFirstInterpretationCapture()
			methods, certificates := historyReconstructionMemberMethods(capture), providerTLSBounds{1, 1, 8192}
			switch mode {
			case "groups-zero":
				g = 0
			case "groups-large":
				g = 1 << 20
			case "requests-low":
				r = 2
			case "requests-large":
				r = 97
			case "metadata-zero":
				d = 0
			case "metadata-large":
				d = 1<<20 + 1
			case "object-zero":
				p = 0
			case "object-small":
				p = uint64(len(providerHistoryOriginalPackMagic) + 4)
			case "object-large":
				p = 32<<20 + 1
			case "policy-zero":
				k = 0
			case "policy-large":
				k = 64<<10 + 1
			case "timeout":
				capture.Timeout = 0
			case "header":
				capture.HeaderBytes = 0
			case "body":
				capture.BodyBytes = 0
			case "wire":
				capture.WireBytes = 0
			case "depth":
				capture.JSONDepth = 0
			case "method-zero":
				methods.scalar.body = 0
			case "method-wide":
				methods.scalar.wire = capture.WireBytes + 1
			case "tls-zero":
				certificates = providerTLSBounds{}
			case "chains":
				certificates.chains = 9
			case "certificate-size":
				certificates.certificateBytes = 64<<10 + 1
			case "overflow":
				g, r, p = (1<<20)/3, ((1<<20)/3)*32, 32<<20
				capture.BodyBytes, capture.WireBytes = 8<<20, 16<<20
				methods = historyReconstructionMemberMethods(capture)
			}
			if got, ok := providerHistoryReconstructionMembersUpper(g, r, d, p, k, capture, methods, certificates, true); ok || got != (providerHistoryReconstructionMemberCost{}) {
				t.Fatal("invalid or overflowing selection returned partial member work", mode, got, ok)
			}
		})
	}
}

func TestProviderHistoryReconstructionMembersKeepOriginalFailureAndTLSSizes(t *testing.T) {
	capture, certificates := historyFirstInterpretationCapture(), providerTLSBounds{1, 1, 8192}
	methods := historyReconstructionMemberMethods(capture)
	for _, slot := range []*providerMethodLimit{&methods.scalar, &methods.header, &methods.code, &methods.state, &methods.block, &methods.receipts, &methods.gas, &methods.execution} {
		slot.body, slot.wire = 4096, 65536
	}
	minimum := uint64(len(providerHistoryFailureMagic) + 4 + providerHistoryFailureHeaderBytes + 65536 + 1)
	before, original := capture, methods
	got, ok := providerHistoryReconstructionMembersUpper(1, 1, 8192, minimum, 8192, capture, methods, certificates, true)
	if !ok || got.failureMinimum != minimum || capture != before || methods != original ||
		got.failureParses != 2 || got.contexts != 10 || got.tls[2] != 6*65536 || got.incoming != (candidateWorkVector{}) {
		t.Fatal("active method ceiling was replaced by the larger global wire or contexts were omitted", got, ok)
	}
	if zero, ok := providerHistoryReconstructionMembersUpper(1, 1, 8192, minimum-1, 8192, capture, methods, certificates, true); ok || zero != (providerHistoryReconstructionMemberCost{}) {
		t.Fatal("failure full header and rejected wire byte were shortened to fit")
	}
	// This is a scalar boundary check, not a constructed hostile packet. The
	// opaque pack rejects oversized TLS before a TLS caller copy can occur.
	if !providerHistoryPackOriginalSize("tls", 64<<10) || providerHistoryPackOriginalSize("tls", 64<<10+1) || !providerHistoryPackOriginalSize("response", 32<<20) {
		t.Fatal("original pack entry size fence no longer supports the TLS child")
	}
}

// Stdlib lengths of the selected original framing, not another production
// observer. This fixture has only scalar requests: one params decode per
// ticket rebuild, two ticket rebuilds per complete T/C result context.
func historyReconstructionResponseDecoded(t *testing.T, wire string) uint64 {
	t.Helper()
	head := bytes.Index([]byte(wire), []byte("\r\n\r\n")) + 4
	if head < 4 {
		t.Fatal("original response framing")
	}
	body := []byte(wire[head:])
	var rpc map[string]json.RawMessage
	if json.Unmarshal(body, &rpc) != nil || len(rpc["jsonrpc"]) == 0 {
		t.Fatal("original response JSON")
	}
	return uint64(head + len(wire) + 2*len(body) + len(rpc["jsonrpc"]))
}

func historyReconstructionMemberDecoded(t *testing.T, ticket *providerHistoryGroupTicket, inputs []providerHistoryOriginalInput) (uint64, uint64) {
	t.Helper()
	var params, members uint64
	for i := range ticket.table.count {
		r := &ticket.table.members[i].request
		if r.method != "eth_chainId" && r.method != "web3_clientVersion" || r.params != "[]" || ticket.table.members[i].class != "scalar" {
			t.Fatal("closed raw-length oracle prerequisite")
		}
		params += uint64(len(r.params))
	}
	if len(inputs)%2 != 0 || len(inputs)/2 > ticket.table.count {
		t.Fatal("original prefix range")
	}
	for i := 0; i < len(inputs)/2; i++ {
		members += uint64(len(inputs[2*i+1].raw)) + historyReconstructionResponseDecoded(t, inputs[2*i].raw)
	}
	return 2 * params, members
}

func historyReconstructionOriginalSpool(t *testing.T, r *providerHistoryRecoveryOwner, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(r.config.evidenceDirectory, name))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal("independent owned-original length oracle", err)
	}
	return string(raw)
}

func TestProviderHistoryReconstructionMembersContainFreshPackAndStandaloneOriginals(t *testing.T) {
	for _, count := range []int{1, 3, providerHistoryGroupMembers} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			methods, capture := historyGroupBounds()
			quote, ok := providerHistoryReconstructionMembersUpper(1, uint64(count), 8192, 1<<20, 8192, capture, methods, providerTLSBounds{1, 1, 8192}, false)
			if !ok {
				t.Fatal("original component quote")
			}
			var allowance candidateWorkVector
			for c := range allowance {
				allowance[c] = quote.packs[c] + quote.context[c] + quote.tls[c] + quote.response[c]
			}
			meter := providerHistoryContentUnitMeter(allowance)
			var decoded uint64
			for pass := range 2 {
				ticket, consumption, inputs, roots, certificates := historyResultDataFixture(t, count)
				pack, err := newProviderHistoryOriginalPackObserved(ticket.table.stage, ticket.table.digest, "results", 0, count, inputs, 1<<20, nil)
				if err != nil || pack == nil {
					t.Fatal("fresh original pack precursor", pass, err)
				}
				contextDecoded, memberDecoded := historyReconstructionMemberDecoded(t, ticket, inputs)
				header := binary.BigEndian.Uint32([]byte(pack.raw[len(providerHistoryOriginalPackMagic) : len(providerHistoryOriginalPackMagic)+4]))
				decoded += 4 + uint64(header) + uint64(1+count)*contextDecoded + 2*memberDecoded
				parsed, results, err := parseProviderHistoryResultPackObserved(providerReplayBytes(pack.raw, meter), pack.digest, ticket, consumption, count, roots, certificates, 1<<20, meter)
				if err != nil || parsed == nil {
					t.Fatal("fresh shared-context pack parse", pass, err)
				}
				for i := range count {
					result, err := parseProviderHistoryMemberOriginalsObserved(inputs[2*i].raw, inputs[2*i+1].raw, ticket, consumption, i, roots, certificates, meter)
					if err != nil || result != results[i] {
						t.Fatal("independent standalone member context", pass, i, err)
					}
				}
			}
			if meter.failed || meter.actual[2] != decoded || meter.clocks != 0 || meter.authority || meter.upper != allowance {
				t.Fatal("shared/standalone decoded oracle or inert observer domain changed", meter.actual, decoded)
			}
		})
	}
}

func TestProviderHistoryReconstructionMembersContainActualCompletedReadersTwice(t *testing.T) {
	for _, count := range []int{0, 1, 3} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			o, r, refs := historyRecoveryCompletedMembersFixture(t, max(1, count), count)
			if _, err := r.beginCatalogCandidate(context.Background()); err != nil {
				t.Fatal("original catalog precursor", err)
			}
			old, reads, held, pin, ack := o.meter.actual, o.store.clock.read, o.store.stored, o.pin, o.last
			original := providerHistoryProvisionedOriginalBytes(t, r.config)
			tls, a := providerHistoryTLSControlUpper(32, r.parent.tls)
			response, b := providerHistoryResponseControlUpper(r.config.capture)
			if !a || !b {
				t.Fatal("separate original component rows")
			}
			var actual, allowed uint64
			for pass := range 2 {
				historyRecoveryCompletedMembersGroup(t, r, pass, refs)
				g := r.groupModels[pass]
				inputs := make([]providerHistoryOriginalInput, 0, 2*count)
				for i := range count {
					n := r.members[i].capture.intent
					inputs = append(inputs, providerHistoryOriginalInput{"response", historyReconstructionOriginalSpool(t, r, n.responseName)}, providerHistoryOriginalInput{"tls", historyReconstructionOriginalSpool(t, r, n.tlsName)})
				}
				contextDecoded, memberDecoded := historyReconstructionMemberDecoded(t, g.memberTicket, inputs)
				decoded := uint64(count)*contextDecoded + memberDecoded
				before, files, rawBytes := r.meter.actual, r.inputFiles, r.inputBytes
				out, err := r.replayCompletedMemberPayloadsCandidate(context.Background(), pass)
				if err != nil || out == nil || !out.selectedLocked(r) || out.count != count || r.inputFiles != files+uint64(2*count) {
					t.Fatal("actual fresh completed original path", pass, err)
				}
				actual += r.meter.actual[10] - before[10]
				k := uint64(len(g.initial.frontier.body.policy.raw))
				if count > 0 {
					decoded += k
					allowed += k*k + k + 2
				}
				// Exact source subset of the already separate I/O row; neither
				// whole initial/group work nor result-pack headroom is borrowed.
				n := uint64(2 * count)
				allowed += 3*(r.inputBytes-rawBytes) + n*(18+767+r.parent.bounds.catalogEntries*511+1) + uint64(count+2)
				for i := range count {
					capture := r.members[i].capture
					wire, diagnostic := capture.responseBytes, capture.tlsBytes
					allowed += tls.parsed[10] + response.member[10] + 83298 + 2*(wire+diagnostic) + 470
					if out.results[i] != r.members[i].result {
						t.Fatal("original completed digest changed", pass, i)
					}
				}
				if r.meter.actual[2]-before[2] != decoded {
					t.Fatal("whole completed original decoded oracle", pass, r.meter.actual[2]-before[2], decoded)
				}
				if out, err := r.replayTailPayloadsCandidate(context.Background(), pass); err != nil || out == nil {
					t.Fatal("complete current pass tail", err)
				}
			}
			if actual > allowed || r.meter.failed || o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held ||
				o.pin != pin || o.last != ack || o.store.activeMeter.Load() != nil || !reflect.DeepEqual(original, providerHistoryProvisionedOriginalBytes(t, r.config)) {
				t.Fatal("completed original reader exceeded its own rows or changed producer state", actual, allowed)
			}
		})
	}
}

func TestProviderHistoryReconstructionMembersContainUnfinishedOriginalAlternativesTwice(t *testing.T) {
	for _, mode := range []string{"uncaptured", "response-short", "tls-short", "cancelled-pair"} {
		t.Run(mode, func(t *testing.T) {
			o, r, refs := historyRecoveryAttemptTailFixture(t, mode)
			if _, err := r.beginCatalogCandidate(context.Background()); err != nil {
				t.Fatal("original catalog precursor", err)
			}
			old, reads, held, pin, ack := o.meter.actual, o.store.clock.read, o.store.stored, o.pin, o.last
			original := providerHistoryProvisionedOriginalBytes(t, r.config)
			tls, a := providerHistoryTLSControlUpper(32, r.parent.tls)
			response, b := providerHistoryResponseControlUpper(r.config.capture)
			if !a || !b {
				t.Fatal("separate original member byte rows")
			}
			for pass := range 2 {
				historyRecoveryTailPrecursors(t, r, pass, refs)
				g, capture := r.groupModels[pass], r.memberCapture
				var decoded uint64
				k := uint64(len(g.initial.frontier.body.policy.raw))
				if capture != nil {
					decoded = k
					wire := historyReconstructionOriginalSpool(t, r, r.memberAttempt.responseName)
					diagnostic := historyReconstructionOriginalSpool(t, r, r.memberAttempt.tlsName)
					if uint64(len(wire)) == capture.responseBytes && wire != "" {
						decoded += historyReconstructionResponseDecoded(t, wire)
						if uint64(len(diagnostic)) == capture.tlsBytes && diagnostic != "" {
							decoded += uint64(len(diagnostic))
							for i := range g.memberTicket.table.count {
								request := &g.memberTicket.table.members[i].request
								if request.method != "eth_chainId" || request.params != "[]" {
									t.Fatal("closed attempt context oracle")
								}
								decoded += 2 * uint64(len(request.params))
							}
						}
					}
				}
				before, files, raw := r.meter.actual, r.inputFiles, r.inputBytes
				out, err := r.replayTailPayloadsCandidate(context.Background(), pass)
				if err != nil || out == nil || !out.selectedLocked(r) || out.data.count != 2 || r.inputFiles != files+2 || r.meter.actual[2]-before[2] != decoded {
					t.Fatal("actual absent/partial/full attempt original path", mode, pass, err, r.meter.actual[2]-before[2], decoded)
				}
				wire, diagnostic := uint64(r.config.capture.WireBytes), min(r.config.bounds.maximumObjectBytes, uint64(64<<10))
				// The exact two-spool subset of the existing I/O and tail rows:
				// no pending-object/metadata or settled-pack slack is used.
				io := 3*(r.inputBytes-raw) + 2*(18+767+r.parent.bounds.catalogEntries*511+1)
				tail := wire + diagnostic + 310
				member := uint64(83298) + tls.parsed[10] + response.member[10] + wire + diagnostic + 69
				policy := k*k + k + 2
				if r.meter.actual[10]-before[10] > io+tail+member+policy || r.meter.failed ||
					mode != "cancelled-pair" && out.data.attemptResult != "" || mode == "cancelled-pair" && out.data.attemptResult == "" {
					t.Fatal("unfinished original path borrowed unrelated work or manufactured completion", mode, pass)
				}
			}
			if o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held || o.pin != pin || o.last != ack ||
				o.store.activeMeter.Load() != nil || !reflect.DeepEqual(original, providerHistoryProvisionedOriginalBytes(t, r.config)) {
				t.Fatal("tail DATA changed original producer, clock, storage, ACK or bytes")
			}
		})
	}
}
