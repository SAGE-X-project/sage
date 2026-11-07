package registry011

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

func historyRecoveryCompletedMembersFixture(t *testing.T, count, completed int) (*providerHistoryGroupOwner, *providerHistoryRecoveryOwner, map[string]providerIntegrationObjectRef) {
	t.Helper()
	o, d := historyRecoveryGroupFixture(t, count)
	historyRetainNormalPrefix(t, o, d, completed)
	r, want := historyRecoveryGroupStart(t, o)
	if r.pending != nil || r.confirmed.frame.phase != "consume" || r.completed != completed {
		t.Fatal("genuine completed original spools without S")
	}
	return o, r, want
}

func historyRecoveryCompletedMembersGroup(t *testing.T, r *providerHistoryRecoveryOwner, pass int, want map[string]providerIntegrationObjectRef) {
	t.Helper()
	historyRecoveryGroupInitial(t, r, pass, want)
	if out, err := r.replayGroupModelCandidate(context.Background(), pass); err != nil || out == nil || !out.selectedLocked(r) {
		t.Fatal("actual selected T/C context before independent member payloads", err)
	}
}

func TestProviderHistoryRecoveryParsesCompletedMemberOriginalsWithoutSettlementTwice(t *testing.T) {
	for _, completed := range []int{0, 1, 3, providerHistoryGroupMembers} {
		t.Run(strconv.Itoa(completed), func(t *testing.T) {
			o, r, want := historyRecoveryCompletedMembersFixture(t, max(1, completed), completed)
			if c, err := r.beginCatalogCandidate(context.Background()); err != nil || c == nil {
				t.Fatal("original C0 before payloads", err)
			}
			old, reads, held, pin, ack, confirmed := o.meter.actual, o.store.clock.read, o.store.stored, o.pin, o.last, r.confirmed
			original := providerHistoryProvisionedOriginalBytes(t, r.config)
			var total uint64
			for i := range completed {
				total += r.members[i].capture.responseBytes + r.members[i].capture.tlsBytes
			}
			var first *providerHistoryRecoveryCompletedMembers
			for pass := range 2 {
				historyRecoveryCompletedMembersGroup(t, r, pass, want)
				files, bytes, work := r.inputFiles, r.inputBytes, r.meter.actual
				out, err := r.replayCompletedMemberPayloadsCandidate(context.Background(), pass)
				if err != nil || out == nil || !out.selectedLocked(r) || out.count != completed ||
					r.inputFiles != files+uint64(2*completed) || r.inputBytes != bytes+total || r.meter.actual[10] <= work[10] ||
					out.group.physical != confirmed || out.group.ledger.phase != "consumed" || r.pending != nil {
					t.Fatal("actual completed original pairs were skipped or promoted to S", pass, err)
				}
				for i := range completed {
					if out.results[i] != r.members[i].result {
						t.Fatal("actual HTTP/RPC/TLS context differs from original completion", pass, i)
					}
				}
				if pass == 0 {
					first = out
				} else if first == out || first.group == out.group || first.group.memberTicket == out.group.memberTicket ||
					first.group.memberConsumption == out.group.memberConsumption || first.results != out.results {
					t.Fatal("completed member passes share mutable T/C models or disagree")
				}
				if out, err := r.replayTailPayloadsCandidate(context.Background(), pass); err != nil || out == nil {
					t.Fatal("final tail proof for otherwise-valid original prefix", err)
				}
			}
			if out, err := r.finishCatalogCandidate(context.Background()); err != nil || out == nil || !out.selectedLocked(r) {
				t.Fatal("completed payload proofs did not connect to final original catalog", err)
			}
			if o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held || o.pin != pin || o.last != ack || r.confirmed != confirmed ||
				o.store.activeMeter.Load() != nil || !reflect.DeepEqual(original, providerHistoryProvisionedOriginalBytes(t, r.config)) {
				t.Fatal("payload reconstruction changed original execution, hold, clock, ACK or files")
			}
		})
	}
}

func TestProviderHistoryRecoveryCompletedMembersRefuseDetachedPrematureAndRepeatedSelection(t *testing.T) {
	for _, detached := range []*providerHistoryRecoveryOwner{nil, {}} {
		if out, err := detached.replayCompletedMemberPayloadsCandidate(context.Background(), 0); err == nil || out != nil {
			t.Fatal("detached owner accepted a completed member proof")
		}
	}
	_, r, want := historyRecoveryCompletedMembersFixture(t, 1, 1)
	work, files := r.meter.actual, r.inputFiles
	if out, err := r.replayCompletedMemberPayloadsCandidate(context.Background(), 0); err == nil || out != nil || r.meter.actual != work || r.inputFiles != files || !r.active {
		t.Fatal("member proof before C0 reached observation")
	}
	if _, err := r.beginCatalogCandidate(context.Background()); err != nil {
		t.Fatal("genuine C0", err)
	}
	historyRecoveryCompletedMembersGroup(t, r, 0, want)
	for _, mode := range []string{"negative-pass", "excess-pass", "second-before-first", "nil-group", "copied-group", "incomplete-group", "nil-c0"} {
		t.Run(mode, func(t *testing.T) {
			g, c0, pass := r.groupModels[0], r.catalogBefore, 0
			switch mode {
			case "negative-pass":
				pass = -1
			case "excess-pass":
				pass = 2
			case "second-before-first":
				pass = 1
			case "nil-group":
				r.groupModels[0] = nil
			case "copied-group":
				copy := *g
				r.groupModels[0] = &copy // owner remains the original.
			case "incomplete-group":
				g.complete = false
			case "nil-c0":
				r.catalogBefore = nil
			}
			work, files, bytes := r.meter.actual, r.inputFiles, r.inputBytes
			out, err := r.replayCompletedMemberPayloadsCandidate(context.Background(), pass)
			r.groupModels[0], r.catalogBefore, g.complete = g, c0, true
			if err == nil || out != nil || r.meter.actual != work || r.inputFiles != files || r.inputBytes != bytes || !r.active || r.closed {
				t.Fatal("unselected member context reached observation or closed the original lane", mode, err)
			}
		})
	}
	historyRecoveryCompletedMembersGroup(t, r, 1, want)
	work = r.meter.actual
	if out, err := r.finishCatalogCandidate(context.Background()); err == nil || out != nil || r.meter.actual != work || !r.active {
		t.Fatal("catalog closed without actual completed-member proof")
	}
	first, err := r.replayCompletedMemberPayloadsCandidate(context.Background(), 0)
	if err != nil || first == nil || !first.selectedLocked(r) {
		t.Fatal("positive control after detached selection refusals", err)
	}
	copy := *first
	copy.owner = &copy
	if copy.selectedLocked(r) || json.Unmarshal([]byte(`{}`), &providerHistoryRecoveryCompletedMembers{}) == nil {
		t.Fatal("copied or serialized DATA manufactured an original member proof")
	}
	work, files = r.meter.actual, r.inputFiles
	if out, err := r.replayCompletedMemberPayloadsCandidate(context.Background(), 0); err == nil || out != nil || r.meter.actual != work || r.inputFiles != files || !r.active {
		t.Fatal("repeated member proof renewed the row")
	}
	if _, err := r.replayCompletedMemberPayloadsCandidate(context.Background(), 1); err != nil {
		t.Fatal("independent second pass", err)
	}
	for pass := range 2 {
		if _, err := r.replayTailPayloadsCandidate(context.Background(), pass); err != nil {
			t.Fatal("tail proofs before final original closure", err)
		}
	}
	if out, err := r.finishCatalogCandidate(context.Background()); err != nil || out == nil || !out.selectedLocked(r) {
		t.Fatal("otherwise-valid full closure after both actual member proofs", err)
	}
}

func TestProviderHistoryRecoveryRejectsChangedOrIncompleteCompletedSpoolsBeforeSettlement(t *testing.T) {
	for _, ordinal := range []int{0, 1} {
		for _, mode := range []string{"response-empty", "tls-empty", "response-missing", "tls-missing", "response-changed", "tls-changed"} {
			t.Run(strconv.Itoa(ordinal)+"-"+mode, func(t *testing.T) {
				o, r, want := historyRecoveryCompletedMembersFixture(t, 2, 2)
				intent := r.members[ordinal].capture.intent
				name := intent.responseName
				if mode == "tls-empty" || mode == "tls-missing" || mode == "tls-changed" {
					name = intent.tlsName
				}
				path := filepath.Join(r.config.evidenceDirectory, name)
				if raw, err := os.ReadFile(path); err != nil || len(raw) == 0 {
					t.Fatal("genuine completed original before owned unit file scenario", err)
				}
				if mode == "response-missing" || mode == "tls-missing" {
					if os.Remove(path) != nil {
						t.Fatal("owned missing completed spool")
					}
				} else {
					var data []byte
					if mode == "response-changed" || mode == "tls-changed" {
						data = []byte("owned inert unit DATA")
					}
					if os.WriteFile(path, data, 0600) != nil {
						t.Fatal("owned empty or changed completed spool")
					}
				}
				// Capture C0 after this owned scenario. A metadata mismatch must
				// not mask missing/partial/content rejection by the payload reader.
				if _, err := r.beginCatalogCandidate(context.Background()); err != nil {
					t.Fatal("scenario C0", err)
				}
				historyRecoveryCompletedMembersGroup(t, r, 0, want)
				old, reads, held, ack, pin := o.meter.actual, o.store.clock.read, o.store.stored, o.last, o.pin
				before := providerHistoryProvisionedOriginalBytes(t, r.config)
				files := r.inputFiles
				if out, err := r.replayCompletedMemberPayloadsCandidate(context.Background(), 0); err == nil || out != nil || r.completedMemberModels[0] != nil ||
					r.inputFiles <= files || r.active || !r.closed || !r.failed || r.catalogAfter != nil || r.catalogResult != nil ||
					o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held || o.last != ack || o.pin != pin ||
					!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, r.config)) {
					t.Fatal("unverified completed spool escaped as a selected proof or changed original authority", err)
				}
			})
		}
	}
}

func TestProviderHistoryRecoveryCompletedMemberClosureRejectsChangedResultData(t *testing.T) {
	for _, coherent := range []bool{false, true} {
		t.Run(strconv.FormatBool(coherent), func(t *testing.T) {
			_, r, want := historyRecoveryCompletedMembersFixture(t, 1, 1)
			if _, err := r.beginCatalogCandidate(context.Background()); err != nil {
				t.Fatal("genuine C0", err)
			}
			historyRecoveryCatalogModels(t, r, want)
			changed := providerReplayDigest("", []byte("changed owned unit result DATA"), nil)
			if !lowerHash(changed) || changed == r.members[0].result {
				t.Fatal("different but well-formed result digest required")
			}
			r.completedMemberModels[0].results[0] = changed
			if coherent {
				r.completedMemberModels[1].results[0] = r.completedMemberModels[0].results[0]
			}
			if out, err := r.finishCatalogCandidate(context.Background()); err == nil || out != nil || r.catalogAfter != nil || r.catalogResult != nil || r.active || !r.failed || !r.closed {
				t.Fatal("equal forged intermediate result DATA replaced original completion", err)
			}
		})
	}
}

func TestProviderHistoryRecoveryCompletedMembersRefuseChangedFreshContextBeforeSpoolReads(t *testing.T) {
	for _, mode := range []string{"nil-ticket", "nil-consumption", "wrong-checkpoint", "stage", "class", "request", "capture"} {
		t.Run(mode, func(t *testing.T) {
			_, r, want := historyRecoveryCompletedMembersFixture(t, 1, 1)
			if _, err := r.beginCatalogCandidate(context.Background()); err != nil {
				t.Fatal("genuine C0", err)
			}
			historyRecoveryCompletedMembersGroup(t, r, 0, want)
			g := r.groupModels[0]
			switch mode {
			case "nil-ticket":
				g.memberTicket = nil
			case "nil-consumption":
				g.memberConsumption = nil
			case "wrong-checkpoint":
				g.memberCheckpoint = r.initial
			case "stage":
				g.memberTicket.table.stage++
			case "class":
				g.memberTicket.table.members[0].class = "other owned class DATA"
			case "request":
				g.memberTicket.table.members[0].request.wire += "other owned request DATA"
			case "capture":
				g.memberTicket.table.members[0].capture.WireBytes++
			}
			files, bytes := r.inputFiles, r.inputBytes
			if out, err := r.replayCompletedMemberPayloadsCandidate(context.Background(), 0); err == nil || out != nil || r.completedMemberModels[0] != nil ||
				r.inputFiles != files || r.inputBytes != bytes || r.active || !r.failed || !r.closed {
				t.Fatal("modified fresh DATA supplied another original spool context", mode, err)
			}
		})
	}
}

func TestProviderHistoryRecoveryCompletedMemberFilesUseOriginalRemainingAllowance(t *testing.T) {
	for _, short := range []bool{false, true} {
		t.Run(strconv.FormatBool(short), func(t *testing.T) {
			_, r, want := historyRecoveryCompletedMembersFixture(t, 2, 2)
			if _, err := r.beginCatalogCandidate(context.Background()); err != nil {
				t.Fatal("genuine C0", err)
			}
			historyRecoveryCompletedMembersGroup(t, r, 0, want)
			bounds := r.bounds
			remaining := uint64(4)
			if short {
				remaining--
			}
			if bounds.maximumFiles != 1024 || r.inputFiles >= bounds.maximumFiles-remaining {
				t.Fatal("original pre-H0 row and genuine remaining-input precursor")
			}
			for r.inputFiles < bounds.maximumFiles-remaining {
				if out, err := r.readCheckpointInputCandidate(context.Background(), r.confirmed, providerHistoryRecoveryHead, 0); err != nil || out == nil || out.state != providerHistoryRecoveryWholeInput {
					t.Fatal("spend actual original HEAD/EOF input; never edit the row or counters", err)
				}
			}
			files, bytes, work := r.inputFiles, r.inputBytes, r.meter.actual
			out, err := r.replayCompletedMemberPayloadsCandidate(context.Background(), 0)
			if short {
				if err == nil || out != nil || r.completedMemberModels[0] != nil || r.inputFiles != files || r.inputBytes != bytes || r.meter.actual != work ||
					r.active || !r.closed || !r.failed {
					t.Fatal("insufficient original allowance reached a completed spool", err)
				}
			} else if err != nil || out == nil || !out.selectedLocked(r) || r.inputFiles != files+4 || r.inputFiles != bounds.maximumFiles || r.inputBytes <= bytes ||
				!r.active || r.failed || r.closed {
				t.Fatal("exact four-input allowance did not complete its one-pass payload witness", err)
			}
			if r.bounds != bounds {
				t.Fatal("member proof changed the original selected allowance")
			}
		})
	}
}

func TestProviderHistoryRecoveryCompletedMemberPayloadsRejectInvalidContextBeforeReads(t *testing.T) {
	for _, mode := range []string{"nil", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			_, r, want := historyRecoveryCompletedMembersFixture(t, 1, 1)
			if _, err := r.beginCatalogCandidate(context.Background()); err != nil {
				t.Fatal("genuine C0", err)
			}
			historyRecoveryCompletedMembersGroup(t, r, 0, want)
			var ctx context.Context
			if mode == "cancelled" {
				c, cancel := context.WithCancel(context.Background())
				cancel()
				ctx = c
			}
			files, bytes, work := r.inputFiles, r.inputBytes, r.meter.actual
			if out, err := r.replayCompletedMemberPayloadsCandidate(ctx, 0); err == nil || out != nil || r.completedMemberModels[0] != nil ||
				!r.closed || !r.failed || r.active || r.inputFiles != files || r.inputBytes != bytes || r.meter.actual != work {
				t.Fatal("invalid context reached a completed spool or published partial proof", err)
			}
		})
	}
}

func TestProviderHistoryMemberOriginalParserValidatesContextWithoutSettlementPack(t *testing.T) {
	_, r, want := historyRecoveryCompletedMembersFixture(t, 2, 2)
	if _, err := r.beginCatalogCandidate(context.Background()); err != nil {
		t.Fatal("genuine C0", err)
	}
	historyRecoveryCompletedMembersGroup(t, r, 0, want)
	g := r.groupModels[0]
	read := func(name string) string {
		raw, err := os.ReadFile(filepath.Join(r.config.evidenceDirectory, name))
		if err != nil || len(raw) == 0 {
			t.Fatal("genuine unit original", err)
		}
		return string(raw)
	}
	response, tls := read(r.members[0].capture.intent.responseName), read(r.members[0].capture.intent.tlsName)
	otherTLS := read(r.members[1].capture.intent.tlsName)
	policy := providerReplayJSON([]byte(g.initial.frontier.body.policy.raw), nil).(map[string]any)
	provider := policy["provider"].(map[string]any)
	roots := contentString(provider["tls_roots_artifact"])
	for _, mode := range []string{"nil-ticket", "nil-consumption", "negative-ordinal", "excess-ordinal", "empty-response", "invalid-http", "other-ordinal-tls", "wrong-roots"} {
		t.Run(mode, func(t *testing.T) {
			ticket, consumption, ordinal, root := g.memberTicket, g.memberConsumption, 0, roots
			wire, diagnostic := response, tls
			switch mode {
			case "nil-ticket":
				ticket = nil
			case "nil-consumption":
				consumption = nil
			case "negative-ordinal":
				ordinal = -1
			case "excess-ordinal":
				ordinal = ticket.table.count
			case "empty-response":
				wire = ""
			case "invalid-http":
				wire = "owned invalid HTTP unit frame"
			case "other-ordinal-tls":
				diagnostic = otherTLS
			case "wrong-roots":
				root = providerReplayDigest("", []byte("other owned unit roots"), nil)
			}
			if out, err := parseProviderHistoryMemberOriginalsObserved(wire, diagnostic, ticket, consumption, ordinal, root, r.parent.tls, r.meter); err == nil || out.digest != "" {
				t.Fatal("standalone semantic parser accepted invalid request/HTTP/TLS context", mode)
			}
		})
	}
	if out, err := parseProviderHistoryMemberOriginalsObserved(response, tls, g.memberTicket, g.memberConsumption, 0, roots, r.parent.tls, r.meter); err != nil || out.digest != r.members[0].result || out.response == "" || out.result == "" {
		t.Fatal("otherwise-valid standalone original pair after semantic refusals", err)
	}
}
