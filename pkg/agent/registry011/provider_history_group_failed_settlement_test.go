package registry011

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func historyFailureRoots(t *testing.T, o *providerHistoryGroupOwner) string {
	t.Helper()
	v, ok := providerReplayJSON([]byte(o.parent.policy.raw), nil).(map[string]any)
	if !ok {
		t.Fatal("original policy DATA")
	}
	p, ok := v["provider"].(map[string]any)
	if !ok {
		t.Fatal("original provider policy DATA")
	}
	roots := contentString(p["tls_roots_artifact"])
	if !lowerHash(roots) {
		t.Fatal("original policy roots")
	}
	return roots
}

func historyRetainNormalPrefix(t *testing.T, o *providerHistoryGroupOwner, d *providerHistoryGroupDispatch, count int) {
	t.Helper()
	for ordinal := range count {
		member := d.coordinator.ticket.table.members[ordinal]
		response := `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`
		if member.request.method == "web3_clientVersion" {
			response = `{"jsonrpc":"2.0","id":1,"result":"owned-node"}`
		}
		conn := providerIntegrationOwnedTLS(t, member.request.body, response)
		if original, err := o.retainMemberCandidateWithConnection(context.Background(), d, ordinal, conn); err != nil || original == nil {
			t.Fatal("genuine successful original prefix", ordinal, err)
		}
	}
}

func TestProviderHistoryFailedCaptureRetainsTerminalSForOriginalPrefixes(t *testing.T) {
	for _, ordinal := range []int{0, 1, 2} {
		t.Run(strconv.Itoa(ordinal), func(t *testing.T) {
			o, d := historyMemberDispatchFixture(t, 3)
			historyRetainNormalPrefix(t, o, d, ordinal)
			s, m := o.store, o.meter
			pin, ack, held, requests, reads, retained := o.pin, o.last, s.stored, m.actual[0], s.clock.read, o.retention
			window, clockStart := o.windowStart, o.clockStart
			member := d.coordinator.ticket.table.members[ordinal]
			// A normal owned TLS server returns an ordinary invalid RPC object.
			// This is not an external peer, attack payload or reproduction tool.
			conn := providerIntegrationOwnedTLS(t, member.request.body, `{}`)
			original, err := o.retainMemberCandidateWithConnection(context.Background(), d, ordinal, conn)
			prefix, _ := providerHistoryMemberPrefix(ordinal + 1)
			all, _ := providerHistoryMemberPrefix(3)
			if err == nil || original != nil || o.last == ack || o.last == nil || o.last.claim == nil || !o.last.claim.used || !o.last.claim.renamed ||
				o.last.claim.failure.kind != "failure" || o.last.claim.dispatch != d || o.last.claim.consumed != ack ||
				o.pin.sequence != pin.sequence+1 || o.pin != s.frontierFiles.pin || o.last.pin != o.pin || s.frontierFiles.historyPending != nil || s.activeMeter.Load() != nil ||
				!o.failed || !s.failed || !m.failed || !m.closed || !o.parent.failed || !s.clock.historyRevoked || s.stored != held ||
				o.coordinator == o.custodian || !o.coordinator.terminalDataValidObserved(nil) || !o.custodian.terminalDataValidObserved(nil) ||
				o.coordinator.valid() || o.custodian.valid() || o.coordinator.head != o.custodian.head ||
				o.order.state != providerHistoryOrderFailed || o.order.consumed != all || o.order.sent != prefix || o.order.next != ordinal || d.retained != ordinal ||
				d.attempt == nil || !d.attempt.used || d.attempt.connection != nil || d.attempt.files != 0 || d.attempt.original != nil ||
				m.actual[0] != requests+1 || s.clock.read != reads+3 || o.retention.physical <= retained.physical || o.windowStart != window || o.clockStart != clockStart {
				t.Fatal("known failure became success, lost its prefix, renewed allowance or skipped retained S", err)
			}
			c := o.last.claim
			recovery := o.parent.recovery
			if recovery.completed != ordinal || recovery.sent != prefix || recovery.memberAttempt == nil || recovery.memberAttempt.attempt != d.attempt ||
				recovery.memberAttempt.ordinal != ordinal || recovery.memberCapture != nil || recovery.memberOriginal != nil || recovery.members[ordinal] != nil || !recovery.memberSelectionLocked(o) {
				t.Fatal("terminal failure lost its sealed successful prefix or failed attempt")
			}
			checkpoint := o.parent.recovery.confirmed
			if checkpoint == nil || checkpoint != c.checkpoint || checkpoint.ack != o.last || checkpoint.previous.ack != ack || !checkpoint.failedS ||
				checkpoint.frame != *c.frame || checkpoint.inventory != *c.inventory || o.parent.recovery.pending != nil || !o.parent.recovery.checkpointSelectionLocked(o) {
				t.Fatal("failed S lost its original terminal checkpoint")
			}
			if (ordinal == 0 && c.count != 2) || (ordinal > 0 && c.count != 3) || c.inventory.previous != ack.claim.inventory.digest ||
				c.frame.previousModel != ack.claim.coordinator.head || c.frame.previousInventory != ack.claim.inventory.digest {
				t.Fatal("failed S changed C or manufactured an empty results pack")
			}
			raw, err := os.ReadFile(filepath.Join(s.config.evidenceDirectory, providerHistoryStoredName(c.failure, nil)))
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := parseProviderHistoryFailureOriginalObserved(raw, c.failure.digest, d.coordinator.ticket, d.coordinator.consumption, ordinal, historyFailureRoots(t, o), o.parent.tls, s.config.bounds.maximumObjectBytes, nil)
			if err != nil || parsed.stage != "framing" || parsed.written != len(member.request.wire) || parsed.received == 0 || parsed.raw != string(raw) {
				t.Fatal("retained failure DATA lost its exact partial bytes or original binding", err)
			}
			after, afterReads := m.actual, s.clock.read
			if out, err := o.retainMemberCandidateWithConnection(context.Background(), d, ordinal, nil); err == nil || out != nil || m.actual != after || s.clock.read != afterReads {
				t.Fatal("terminal S permitted a repeated attempt")
			}
			if out, err := o.retainSettlementCandidate(context.Background(), d); err == nil || out != nil || m.actual != after || s.clock.read != afterReads {
				t.Fatal("failed S was upgraded to successful completion")
			}
		})
	}
}

func TestProviderHistoryFailedSettlementUncertaintyPreservesCWithoutFailureACK(t *testing.T) {
	for _, mode := range []string{"failure-sync", "pre-cas-original", "post-cas-original", "post-cas-cancel"} {
		t.Run(mode, func(t *testing.T) {
			o, d := historyMemberDispatchFixture(t, 1)
			s, m := o.store, o.meter
			pin, ack, held, retained, rawBefore := o.pin, o.last, s.stored, o.retention, s.retainedRaw
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			disk := s.custody
			if mode == "failure-sync" {
				disk = s.evidence
			}
			syncFile := disk.syncFile
			reached := false
			disk.syncFile = func(file *os.File) error {
				if err := syncFile(file); err != nil {
					return err
				}
				c := s.frontierFiles.historyPending
				if reached || c == nil || c.failure.kind != "failure" {
					return nil
				}
				checkpoint := o.parent.recovery.pending
				if checkpoint == nil || checkpoint != c.checkpoint || checkpoint.claim != c || checkpoint.previous != o.parent.recovery.confirmed || checkpoint.previous.ack != ack ||
					checkpoint.ack == ack || !checkpoint.failedS || checkpoint.frame != *c.frame || checkpoint.inventory != *c.inventory {
					t.Fatal("physical write preceded the original pending checkpoint")
				}
				if mode == "failure-sync" {
					if filepath.Base(file.Name()) != providerHistoryStoredName(c.failure, nil) {
						return nil
					}
					reached = true
					return errProviderIntegration
				}
				if mode == "pre-cas-original" {
					if filepath.Base(file.Name()) != providerCustodyName("HEAD.pending", c.frame.after.ordinal) {
						return nil
					}
				} else {
					info, err := file.Stat()
					if err != nil {
						return err
					}
					if !info.IsDir() {
						return nil
					}
					head, err := os.ReadFile(filepath.Join(s.config.custodyDirectory, "HEAD"))
					if err != nil || string(head) != c.highwater {
						return nil
					}
				}
				reached = true
				if mode == "post-cas-cancel" {
					cancel()
					return nil
				}
				return os.WriteFile(filepath.Join(s.config.evidenceDirectory, providerHistoryStoredName(c.failure, nil)), []byte("inert owned failure-record mismatch"), 0600)
			}
			member := d.coordinator.ticket.table.members[0]
			conn := providerIntegrationOwnedTLS(t, member.request.body, `{}`)
			out, err := o.retainMemberCandidateWithConnection(ctx, d, 0, conn)
			c := s.frontierFiles.historyPending
			if err == nil || out != nil || !reached || c == nil || c.used || o.last != ack || o.pin != pin || s.frontierFiles.pin != pin ||
				!o.failed || !s.failed || !m.failed || !m.closed || s.stored != held || s.retainedRaw != rawBefore || o.retention.physical <= retained.physical ||
				o.order.state != providerHistoryOrderFailed || o.order.next != 0 || o.order.sent != 1 || s.activeMeter.Load() != nil {
				t.Fatal("uncertain failure custody issued an ACK, refunded or completed the failed attempt", mode, err)
			}
			r := o.parent.recovery
			if r.confirmed == nil || r.confirmed.ack != ack || r.confirmed.pin != pin || r.pending == nil || r.pending != c.checkpoint ||
				r.pending.previous != r.confirmed || r.pending.ack == ack || r.pending.pin != c.frame.after || !r.pending.failedS || !r.checkpointSelectionLocked(o) {
				t.Fatal("uncertain failed S replaced the last confirmed C checkpoint")
			}
			post := mode == "post-cas-original" || mode == "post-cas-cancel"
			if c.renamed != post {
				t.Fatal("failed S confused physical CAS with logical failure ACK")
			}
		})
	}
}

// Finite DATA only. Fabricated TLS diagnostics here never enter a live owner
// or prove an actual dial/partial-write failure; live coverage is above.
func historyFailureDataFixture(t *testing.T, stage string) (*providerHistoryGroupTicket, *providerHistoryGroupConsumption, *providerHistoryCaptureData, string, providerTLSBounds) {
	t.Helper()
	ticket, consumption, _, roots, bounds := historyResultDataFixture(t, 1)
	r := ticket.table.members[0].request
	data := &providerHistoryCaptureData{knownFailure: true, failureStage: stage}
	if stage == "dial" {
		return ticket, consumption, data, roots, bounds
	}
	data.sample = &ProviderTransportSample{Endpoint: SepoliaRPCURL, Method: r.method, HTTPVersion: "HTTP/1.1", ServerName: "rpc.sepolia.ethpandaops.io", TLSVersion: tls.VersionTLS13, CipherSuite: tls.TLS_AES_128_GCM_SHA256,
		VerifiedChains: [][]string{{admissionTestDigest("3")}}, RequestHead: []byte(r.head), RequestBody: []byte(r.body)}
	data.requestWritten = len(r.wire)
	switch stage {
	case "write":
		data.requestWritten = 5
	case "read":
		data.sample.ResponseWire = []byte("HTTP/1.1")
	default:
		data.sample.ResponseWire = []byte("finite invalid reply DATA")
	}
	data.responseRead = len(data.sample.ResponseWire)
	return ticket, consumption, data, roots, bounds
}

func TestProviderHistoryFailureOriginalPreservesDialAndPartialByteData(t *testing.T) {
	for _, stage := range []string{"dial", "write", "read", "framing"} {
		t.Run(stage, func(t *testing.T) {
			ticket, consumption, data, roots, bounds := historyFailureDataFixture(t, stage)
			meter := historyGroupCodecMeter()
			original, err := newProviderHistoryFailureOriginalObserved(ticket, consumption, 0, data, roots, bounds, providerHistoryOriginalPackBytes, meter)
			if err != nil || original == nil || meter.failed {
				t.Fatal("bounded finite failure DATA", err)
			}
			parsed, err := parseProviderHistoryFailureOriginalObserved([]byte(original.raw), original.digest, ticket, consumption, 0, roots, bounds, providerHistoryOriginalPackBytes, meter)
			metadata, payload, ok := providerHistoryFailureLayout(original.raw)
			if err != nil || parsed == original || *parsed != *original || !ok || metadata+payload != uint64(len(original.raw)) || payload != uint64(data.requestWritten+data.responseRead) ||
				meter.failed || meter.actual[0] != 0 || meter.actual[1] != 0 || meter.clocks != 0 {
				t.Fatal("failure representation changed exact prefix bytes or manufactured an effect", stage, err)
			}
			if json.Unmarshal([]byte(`{}`), original) == nil {
				t.Fatal("serialized DATA manufactured a live failure record")
			}
		})
	}
}

func TestProviderHistoryFailureOriginalRejectsChangedFramingAndContexts(t *testing.T) {
	ticket, consumption, data, roots, bounds := historyFailureDataFixture(t, "read")
	original, err := newProviderHistoryFailureOriginalObserved(ticket, consumption, 0, data, roots, bounds, providerHistoryOriginalPackBytes, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"magic", "length", "header", "payload", "truncated", "extra", "digest", "ordinal", "roots", "too-small"} {
		t.Run(mode, func(t *testing.T) {
			raw := []byte(original.raw)
			digest, selectedRoots, ordinal, maximum := original.digest, roots, 0, uint64(providerHistoryOriginalPackBytes)
			switch mode {
			case "magic":
				raw[0] ^= 1
			case "length":
				raw[len(providerHistoryFailureMagic)] ^= 0x40
			case "header":
				raw[len(providerHistoryFailureMagic)+4] ^= 1
			case "payload":
				raw[len(raw)-1] ^= 1
			case "truncated":
				raw = raw[:len(raw)-1]
			case "extra":
				raw = append(raw, 0)
			case "digest":
				digest = admissionTestDigest("f")
			case "ordinal":
				ordinal = 1
			case "roots":
				selectedRoots = admissionTestDigest("e")
			case "too-small":
				maximum = uint64(len(raw) - 1)
			}
			if got, err := parseProviderHistoryFailureOriginalObserved(raw, digest, ticket, consumption, ordinal, selectedRoots, bounds, maximum, historyGroupCodecMeter()); err == nil || got != nil {
				t.Fatal("changed failure DATA was accepted", mode)
			}
		})
	}
}

// Finite rejected DATA proves framing work is charged on the same observer,
// before the JSON header and without creating a capture or retained ACK.
func TestProviderHistoryFailureFramingConsumesOriginalObservation(t *testing.T) {
	ticket, consumption, data, roots, bounds := historyFailureDataFixture(t, "read")
	original, err := newProviderHistoryFailureOriginalObserved(ticket, consumption, 0, data, roots, bounds, providerHistoryOriginalPackBytes, nil)
	if err != nil {
		t.Fatal("genuine failure DATA precursor", err)
	}
	contextProbe := historyGroupCodecMeter()
	if !providerHistoryResultContextObserved(ticket, consumption, 0, contextProbe) || contextProbe.failed {
		t.Fatal("genuine context observation precursor")
	}
	beforeFraming := contextProbe.actual
	// Copying n magic bytes costs n+1; comparing two n-byte strings costs
	// 2*n+1. The subsequent four-byte decode costs 4 decoded bytes and17
	// logical units. The zero header length stops before any JSON decode.
	beforeDecode := beforeFraming
	beforeDecode[10] += 3*uint64(len(providerHistoryFailureMagic)) + 2
	decoded := beforeDecode
	decoded[2] += 4
	decoded[10] += 17
	for _, mode := range []string{"zero-length", "short-decode", "changed-magic"} {
		t.Run(mode, func(t *testing.T) {
			raw := []byte(original.raw)
			start := len(providerHistoryFailureMagic)
			for i := range 4 {
				raw[start+i] = 0
			}
			m := historyGroupCodecMeter()
			if mode == "short-decode" {
				m.upper[2] = beforeFraming[2] + 3
			}
			if mode == "changed-magic" {
				raw[0] ^= 1
			}
			got, err := parseProviderHistoryFailureOriginalObserved(raw, original.digest, ticket, consumption, 0, roots, bounds, providerHistoryOriginalPackBytes, m)
			if err == nil || got != nil || m.closed || m.actual[0] != 0 || m.actual[1] != 0 || m.clocks != 0 {
				t.Fatal("rejected framing produced DATA or a live effect", mode)
			}
			switch mode {
			case "zero-length":
				if m.failed || m.actual != decoded {
					t.Fatal("length decode was unobserved or JSON work occurred before length validation")
				}
			case "changed-magic":
				if m.failed || m.actual != beforeDecode {
					t.Fatal("magic mismatch skipped its comparison or reached length decoding")
				}
			case "short-decode":
				if !m.failed || m.actual != beforeDecode {
					t.Fatal("short original decode allowance was exceeded or lost its charged prefix")
				}
				after := m.actual
				m.upper[2]++
				if got, err := parseProviderHistoryFailureOriginalObserved(raw, original.digest, ticket, consumption, 0, roots, bounds, providerHistoryOriginalPackBytes, m); err == nil || got != nil || !m.failed || m.actual != after {
					t.Fatal("failed original observer was revived by a later allowance")
				}
			}
		})
	}
	for _, mode := range []string{"mismatched-owner", "closed-observer"} {
		t.Run(mode, func(t *testing.T) {
			originalMeter := historyGroupCodecMeter()
			m := originalMeter
			if mode == "mismatched-owner" {
				// Copy only DATA, never the mutex or a valid ownership role.
				m = &providerWorkMeter{owner: originalMeter, integration: originalMeter.integration, upper: originalMeter.upper, remaining: originalMeter.remaining, reservation: originalMeter.reservation}
			} else {
				m.closed = true
			}
			before := m.actual
			if got, err := parseProviderHistoryFailureOriginalObserved([]byte(original.raw), original.digest, ticket, consumption, 0, roots, bounds, providerHistoryOriginalPackBytes, m); err == nil || got != nil || m.actual != before || m.clocks != 0 {
				t.Fatal("invalid observer reached failure parsing or borrowed a nil observer", mode)
			}
		})
	}
}
