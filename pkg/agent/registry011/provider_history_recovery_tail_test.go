package registry011

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func historyRecoveryPendingHeadFixture(t *testing.T) (*providerHistoryGroupOwner, *providerHistoryRecoveryOwner, map[string]providerIntegrationObjectRef) {
	t.Helper()
	o, d := historyRecoveryGroupFixture(t, 1)
	historyRetainNormalPrefix(t, o, d, 1)
	disk, reached := o.store.custody, false
	syncFile := disk.syncFile
	disk.syncFile = func(file *os.File) error {
		if err := syncFile(file); err != nil {
			return err
		}
		c := o.store.frontierFiles.historyPending
		if c != nil && c.frame.phase == "settle" && filepath.Base(file.Name()) == providerCustodyName("HEAD.pending", c.frame.after.ordinal) {
			reached = true
			return errProviderIntegration // Owned temporary-HEAD sync uncertainty.
		}
		return nil
	}
	_, err := o.retainSettlementCandidate(context.Background(), d)
	disk.syncFile = syncFile
	r := o.parent.recovery
	if err == nil || !reached || !o.failed || r.pending == nil || r.pending.claim.renamed || r.confirmed.frame.phase != "consume" {
		t.Fatal("genuine complete temporary HEAD before rename and without ACK", err)
	}
	r, want := historyRecoveryGroupStart(t, o)
	return o, r, want
}

func historyRecoveryTailPrecursors(t *testing.T, r *providerHistoryRecoveryOwner, pass int, want map[string]providerIntegrationObjectRef) {
	t.Helper()
	historyRecoveryCompletedMembersGroup(t, r, pass, want)
	if out, err := r.replayCompletedMemberPayloadsCandidate(context.Background(), pass); err != nil || out == nil || !out.selectedLocked(r) {
		t.Fatal("genuine completed-member proof before tail inspection", err)
	}
}

func historyRecoveryAttemptTailFixture(t *testing.T, mode string) (*providerHistoryGroupOwner, *providerHistoryRecoveryOwner, map[string]providerIntegrationObjectRef) {
	t.Helper()
	o, d := historyRecoveryGroupFixture(t, 1)
	r := o.parent.recovery
	if mode == "uncaptured" || mode == "pending-failure" {
		disk, reached := o.store.evidence, false
		syncFile := disk.syncFile
		if mode == "pending-failure" {
			disk.syncFile = func(file *os.File) error {
				if err := syncFile(file); err != nil {
					return err
				}
				c := o.store.frontierFiles.historyPending
				if c != nil && c.frame.phase == "settle" && c.failure.kind == "failure" && filepath.Base(file.Name()) == providerHistoryStoredName(c.inventory.entries[0], nil) {
					reached = true
					return errProviderIntegration
				}
				return nil
			}
		}
		member := d.coordinator.ticket.table.members[0]
		conn := providerIntegrationOwnedTLS(t, member.request.body, `{}`)
		out, err := o.retainMemberCandidateWithConnection(context.Background(), d, 0, conn)
		disk.syncFile = syncFile
		if err == nil || out != nil || !o.failed || r.memberAttempt == nil || r.memberCapture != nil ||
			mode == "uncaptured" && (!r.confirmed.failedS || r.pending != nil) ||
			mode == "pending-failure" && (!reached || r.pending == nil || !r.pending.failedS || r.confirmed.frame.phase != "consume") {
			t.Fatal("genuine failed S without a prepared member spool", err)
		}
		r, want := historyRecoveryGroupStart(t, o)
		return o, r, want
	}
	part, written := "response", 0
	if strings.HasPrefix(mode, "tls") || mode == "cancelled-pair" {
		part, written = "tls", 1
	}
	data := *d
	data.owner, data.meter = &data, nil
	name := data.originalNameLocked(0, part)
	if name == "" {
		t.Fatal("bounded unit filename DATA before the original attempt")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	disk, reached := o.store.evidence, false
	syncFile := disk.syncFile
	disk.syncFile = func(file *os.File) error {
		if err := syncFile(file); err != nil {
			return err
		}
		if filepath.Base(file.Name()) == name {
			reached = true
			if mode == "cancelled-pair" {
				cancel()
				return nil
			}
			return errProviderIntegration
		}
		return nil
	}
	member := d.coordinator.ticket.table.members[0]
	conn := providerIntegrationOwnedTLS(t, member.request.body, `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`)
	_, err := o.retainMemberCandidateWithConnection(ctx, d, 0, conn)
	disk.syncFile = syncFile
	if mode == "cancelled-pair" {
		written = 2
	}
	if err == nil || !reached || !o.failed || r.memberAttempt == nil || r.memberCapture == nil || r.completed != 0 || r.pending != nil ||
		d.attempt == nil || d.attempt.files != written || r.confirmed.frame.phase != "consume" {
		t.Fatal("genuine incomplete original capture without a member completion or S", mode, err)
	}
	if strings.HasSuffix(mode, "-empty") || strings.HasSuffix(mode, "-short") {
		path := filepath.Join(r.config.evidenceDirectory, name)
		raw, err := os.ReadFile(path)
		if err != nil || len(raw) < 2 {
			t.Fatal("original uncertain write before owned partial scenario", err)
		}
		if strings.HasSuffix(mode, "-empty") {
			raw = nil
		} else {
			raw = raw[:len(raw)/2]
		}
		if os.WriteFile(path, raw, 0600) != nil {
			t.Fatal("owned uncertain write partial prefix")
		}
	}
	r, want := historyRecoveryGroupStart(t, o)
	return o, r, want
}

func TestProviderHistoryRecoveryInspectsAllPendingSlotsPastFirstMissingInput(t *testing.T) {
	for _, mode := range []string{"consumed", "pending-admission", "pending-consumption", "pending", "partial", "head-full", "head-prefix", "head-empty", "head-missing", "object-prefix"} {
		t.Run(mode, func(t *testing.T) {
			var o *providerHistoryGroupOwner
			var r *providerHistoryRecoveryOwner
			var want map[string]providerIntegrationObjectRef
			if strings.HasPrefix(mode, "head-") {
				o, r, want = historyRecoveryPendingHeadFixture(t)
				path := filepath.Join(r.config.custodyDirectory, providerCustodyName("HEAD.pending", r.pending.pin.ordinal))
				if mode == "head-missing" {
					if os.Remove(path) != nil {
						t.Fatal("owned absent temporary HEAD")
					}
				} else if mode != "head-full" {
					raw := []byte(r.pending.head)
					if mode == "head-empty" {
						raw = nil
					} else {
						raw = raw[:len(raw)/2]
					}
					if os.WriteFile(path, raw, 0600) != nil {
						t.Fatal("owned temporary HEAD partial prefix")
					}
				}
			} else {
				fixture := mode
				if mode == "object-prefix" {
					fixture = "pending-admission"
				}
				o, r, want = historyRecoveryCatalogFixture(t, fixture)
				if mode == "object-prefix" {
					path := filepath.Join(r.config.evidenceDirectory, providerHistoryStoredName(r.pending.inventory.entries[0], nil))
					raw, err := os.ReadFile(path)
					if err != nil || len(raw) < 2 || os.WriteFile(path, raw[:len(raw)/2], 0600) != nil {
						t.Fatal("genuine first uncertain object write, now a short prefix", err)
					}
				}
			}
			if _, err := r.beginCatalogCandidate(context.Background()); err != nil {
				t.Fatal("original scenario C0 before payloads", err)
			}
			old, reads, held, pin, ack, confirmed, pending := o.meter.actual, o.store.clock.read, o.store.stored, o.pin, o.last, r.confirmed, r.pending
			before := providerHistoryProvisionedOriginalBytes(t, r.config)
			var first *providerHistoryRecoveryTailPayloads
			for pass := range 2 {
				historyRecoveryTailPrecursors(t, r, pass, want)
				files, work := r.inputFiles, r.meter.actual
				out, err := r.replayTailPayloadsCandidate(context.Background(), pass)
				expected := 0
				if pending != nil {
					expected = 7 + pending.inventory.count
				}
				if err != nil || out == nil || !out.selectedLocked(r) || out.data.count != expected || r.inputFiles != files+uint64(expected) ||
					r.meter.actual[10] <= work[10] || out.data.unacknowledged != (pending != nil) || out.data.unverifiedPartial != (mode == "object-prefix") {
					t.Fatal("not all selected pending slots were actually observed", mode, pass, err)
				}
				incomplete := mode == "pending-admission" || mode == "pending-consumption" || mode == "partial" || mode == "head-prefix" || mode == "head-empty" || mode == "object-prefix"
				if out.data.incomplete != incomplete {
					t.Fatal("partial or unauthenticated tail became complete DATA", mode)
				}
				if pending != nil {
					last := out.data.entries[out.data.count-1]
					if last.kind != providerHistoryRecoveryHeadPending || !last.optional {
						t.Fatal("temporary HEAD was skipped after a missing earlier slot")
					}
				}
				if pass == 0 {
					first = out
				} else if first == out || first.group == out.group || first.data != out.data {
					t.Fatal("actual tail passes share mutable state or disagree")
				}
			}
			if out, err := r.finishCatalogCandidate(context.Background()); err != nil || out == nil || !out.selectedLocked(r) {
				t.Fatal("tail DATA did not connect to original final closure", err)
			}
			if o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held || o.pin != pin || o.last != ack || r.confirmed != confirmed || r.pending != pending ||
				o.store.activeMeter.Load() != nil || !reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, r.config)) {
				t.Fatal("tail observation promoted a pending ACK, refunded work or changed originals")
			}
		})
	}
}

func TestProviderHistoryRecoveryInspectsUnfinishedMemberSpoolsWithoutCompletion(t *testing.T) {
	for _, mode := range []string{"uncaptured", "pending-failure", "response-sync", "response-empty", "response-short", "tls-sync", "tls-empty", "tls-short", "cancelled-pair"} {
		t.Run(mode, func(t *testing.T) {
			o, r, want := historyRecoveryAttemptTailFixture(t, mode)
			if _, err := r.beginCatalogCandidate(context.Background()); err != nil {
				t.Fatal("original attempted-member C0", err)
			}
			old, reads, held, ack, pin, confirmed, pending := o.meter.actual, o.store.clock.read, o.store.stored, o.last, o.pin, r.confirmed, r.pending
			before := providerHistoryProvisionedOriginalBytes(t, r.config)
			for pass := range 2 {
				historyRecoveryTailPrecursors(t, r, pass, want)
				files := r.inputFiles
				out, err := r.replayTailPayloadsCandidate(context.Background(), pass)
				pair := mode == "tls-sync" || mode == "cancelled-pair"
				partial := mode != "uncaptured" && !pair
				expected := 2
				if pending != nil {
					expected += 7 + pending.inventory.count
				}
				if err != nil || out == nil || !out.selectedLocked(r) || out.data.count != expected || r.inputFiles != files+uint64(expected) || !out.data.unacknowledged ||
					out.data.incomplete != partial || out.data.unverifiedPartial != strings.HasSuffix(mode, "-short") || (out.data.attemptResult != "") != pair ||
					r.completed != 0 || r.members[0] != nil || r.pending != pending || r.confirmed != confirmed {
					t.Fatal("unfinished original capture became a member completion or its prefix was authenticated", mode, pass, err)
				}
			}
			if out, err := r.finishCatalogCandidate(context.Background()); err != nil || out == nil || !out.selectedLocked(r) {
				t.Fatal("original incomplete capture DATA closure", err)
			}
			if o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held || o.last != ack || o.pin != pin ||
				!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, r.config)) {
				t.Fatal("inspection completed an attempted member or changed original effects")
			}
		})
	}
}

func TestProviderHistoryRecoveryTailRejectsWholeChangedAndUnplannedPayloads(t *testing.T) {
	for _, mode := range []string{"object", "inventory", "temporary-head", "unprepared-spool", "previously-written-response"} {
		t.Run(mode, func(t *testing.T) {
			var r *providerHistoryRecoveryOwner
			var want map[string]providerIntegrationObjectRef
			var path string
			var raw []byte
			switch mode {
			case "temporary-head":
				_, r, want = historyRecoveryPendingHeadFixture(t)
				path = filepath.Join(r.config.custodyDirectory, providerCustodyName("HEAD.pending", r.pending.pin.ordinal))
				raw = []byte(strings.Repeat("x", len(r.pending.head)))
			case "unprepared-spool":
				_, r, want = historyRecoveryAttemptTailFixture(t, "uncaptured")
				path, raw = filepath.Join(r.config.evidenceDirectory, r.memberAttempt.responseName), []byte("owned unexpected spool DATA")
			case "previously-written-response":
				_, r, want = historyRecoveryAttemptTailFixture(t, "tls-sync")
				path = filepath.Join(r.config.evidenceDirectory, r.memberAttempt.responseName)
			default:
				_, r, want = historyRecoveryCatalogFixture(t, "pending-admission")
				if mode == "inventory" {
					path = filepath.Join(r.config.evidenceDirectory, providerHistoryInventoryName(r.pending.inventory.digest, nil))
					raw = []byte(strings.Repeat("x", len(r.pending.inventory.raw)))
				} else {
					ref := r.pending.inventory.entries[0]
					path, raw = filepath.Join(r.config.evidenceDirectory, providerHistoryStoredName(ref, nil)), []byte(strings.Repeat("x", int(ref.bytes)))
				}
			}
			if os.WriteFile(path, raw, 0600) != nil {
				t.Fatal("owned inert whole-content/absence boundary scenario")
			}
			if _, err := r.beginCatalogCandidate(context.Background()); err != nil {
				t.Fatal("scenario C0 before actual payload models", err)
			}
			historyRecoveryTailPrecursors(t, r, 0, want)
			files := r.inputFiles
			if out, err := r.replayTailPayloadsCandidate(context.Background(), 0); err == nil || out != nil || r.tailModels[0] != nil ||
				r.tailSeals[0] != nil || r.inputFiles <= files || r.active || !r.failed || !r.closed || r.catalogAfter != nil {
				t.Fatal("first-missing short circuit hid a changed or impossible later payload", mode, err)
			}
		})
	}
}

func TestProviderHistoryRecoveryTailSelectionAndFinalSealRejectCopiesAndChangedData(t *testing.T) {
	_, r, want := historyRecoveryCatalogFixture(t, "pending-admission")
	work := r.meter.actual
	if out, err := r.replayTailPayloadsCandidate(context.Background(), 0); err == nil || out != nil || r.meter.actual != work || !r.active {
		t.Fatal("tail without C0 and actual semantic/member prefixes")
	}
	if _, err := r.beginCatalogCandidate(context.Background()); err != nil {
		t.Fatal("genuine C0", err)
	}
	for pass := range 2 {
		historyRecoveryTailPrecursors(t, r, pass, want)
	}
	for _, pass := range []int{-1, 2, 1} {
		work, files := r.meter.actual, r.inputFiles
		if out, err := r.replayTailPayloadsCandidate(context.Background(), pass); err == nil || out != nil || r.meter.actual != work || r.inputFiles != files || !r.active {
			t.Fatal("invalid pass/order changed observation", pass)
		}
	}
	work = r.meter.actual
	if out, err := r.finishCatalogCandidate(context.Background()); err == nil || out != nil || r.meter.actual != work || !r.active {
		t.Fatal("final closure skipped actual tail observations")
	}
	first, err := r.replayTailPayloadsCandidate(context.Background(), 0)
	if err != nil || first == nil || !first.selectedLocked(r) {
		t.Fatal("otherwise-valid original tail", err)
	}
	copy := *first
	copy.owner = &copy
	if copy.selectedLocked(r) || json.Unmarshal([]byte(`{}`), &providerHistoryRecoveryTailPayloads{}) == nil || json.Unmarshal([]byte(`{}`), &providerHistoryRecoveryTailData{}) == nil {
		t.Fatal("copy/JSON manufactured a tail observation")
	}
	work, files := r.meter.actual, r.inputFiles
	if out, err := r.replayTailPayloadsCandidate(context.Background(), 0); err == nil || out != nil || r.meter.actual != work || r.inputFiles != files || !r.active {
		t.Fatal("repeated tail inspection reset its row")
	}
	if _, err := r.replayTailPayloadsCandidate(context.Background(), 1); err != nil {
		t.Fatal("independent second tail pass", err)
	}
	if out, err := r.finishCatalogCandidate(context.Background()); err != nil || out == nil || !out.selectedLocked(r) {
		t.Fatal("positive final closure after isolated selection refusals", err)
	}
}

func TestProviderHistoryRecoveryTailPrivateSealsRejectCoherentIntermediateChanges(t *testing.T) {
	for _, mode := range []string{"incomplete", "digest", "unused-slot", "nil-first-seal", "nil-second-seal"} {
		t.Run(mode, func(t *testing.T) {
			o, r, want := historyRecoveryCatalogFixture(t, "pending-admission")
			if _, err := r.beginCatalogCandidate(context.Background()); err != nil {
				t.Fatal("genuine C0", err)
			}
			historyRecoveryCatalogModels(t, r, want)
			for pass := range 2 {
				if r.tailSeals[pass] == nil || r.tailSeals[pass] == &r.tailModels[pass].data || *r.tailSeals[pass] != r.tailModels[pass].data {
					t.Fatal("original private value copy must not alias returned DATA", pass)
				}
			}
			if r.tailSeals[0] == r.tailSeals[1] {
				t.Fatal("independent observations shared one private seal")
			}
			work, reads, held, pin, ack := o.meter.actual, o.store.clock.read, o.store.stored, o.pin, o.last
			for pass := range 2 {
				view := &r.tailModels[pass].data
				switch mode {
				case "incomplete":
					if !view.incomplete {
						t.Fatal("genuine incomplete pending prefix required")
					}
					view.incomplete = false
				case "digest":
					view.entries[0].digest = providerReplayDigest("", []byte("changed owned observed hash"), nil)
				case "unused-slot":
					view.entries[len(view.entries)-1].digest = providerReplayDigest("", []byte("changed owned unused slot"), nil)
				}
			}
			if mode == "nil-first-seal" {
				r.tailSeals[0] = nil
			} else if mode == "nil-second-seal" {
				r.tailSeals[1] = nil
			}
			if out, err := r.finishCatalogCandidate(context.Background()); err == nil || out != nil || r.catalogAfter != nil || r.catalogResult != nil || r.active || !r.failed || !r.closed ||
				o.meter.actual != work || o.store.clock.read != reads || o.store.stored != held || o.pin != pin || o.last != ack || !o.failed || !o.meter.failed || !o.meter.closed {
				t.Fatal("coherent returned DATA replaced original private observation seals", mode, err)
			}
		})
	}
}

func TestProviderHistoryRecoveryTailInvalidContextDoesNotReachInputs(t *testing.T) {
	for _, mode := range []string{"nil", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			_, r, want := historyRecoveryCatalogFixture(t, "pending-admission")
			if _, err := r.beginCatalogCandidate(context.Background()); err != nil {
				t.Fatal("genuine C0", err)
			}
			historyRecoveryTailPrecursors(t, r, 0, want)
			var ctx context.Context
			if mode == "cancelled" {
				c, cancel := context.WithCancel(context.Background())
				cancel()
				ctx = c
			}
			files, bytes, work := r.inputFiles, r.inputBytes, r.meter.actual
			if out, err := r.replayTailPayloadsCandidate(ctx, 0); err == nil || out != nil || r.tailModels[0] != nil || r.inputFiles != files || r.inputBytes != bytes || r.meter.actual != work ||
				r.active || !r.closed || !r.failed {
				t.Fatal("invalid context reached tail I/O or selected a partial witness", err)
			}
		})
	}
}

func TestProviderHistoryRecoveryTailUsesActualRemainingOriginalFileBudget(t *testing.T) {
	for _, short := range []bool{false, true} {
		t.Run(strconv.FormatBool(short), func(t *testing.T) {
			_, r, want := historyRecoveryPendingHeadFixture(t)
			if _, err := r.beginCatalogCandidate(context.Background()); err != nil {
				t.Fatal("genuine C0", err)
			}
			historyRecoveryTailPrecursors(t, r, 0, want)
			bounds := r.bounds
			remaining := uint64(9) // Two S objects and all seven controls.
			if short {
				remaining--
			}
			if r.pending.inventory.count != 2 || bounds.maximumFiles != 1024 || r.inputFiles >= bounds.maximumFiles-remaining {
				t.Fatal("genuine original pending S and file allowance")
			}
			for r.inputFiles < bounds.maximumFiles-remaining {
				if out, err := r.readCheckpointInputCandidate(context.Background(), r.confirmed, providerHistoryRecoveryHead, 0); err != nil || out == nil || out.state != providerHistoryRecoveryWholeInput {
					t.Fatal("actual closed HEAD/EOF spending, not counter or cap edits", err)
				}
			}
			files, bytes, work := r.inputFiles, r.inputBytes, r.meter.actual
			out, err := r.replayTailPayloadsCandidate(context.Background(), 0)
			if short {
				if err == nil || out != nil || r.tailModels[0] != nil || r.inputFiles != files || r.inputBytes != bytes || r.meter.actual != work || r.active || !r.closed || !r.failed {
					t.Fatal("insufficient original allowance reached a pending input", err)
				}
			} else if err != nil || out == nil || !out.selectedLocked(r) || r.inputFiles != files+9 || r.inputFiles != bounds.maximumFiles || out.data.count != 9 || !out.data.unacknowledged {
				t.Fatal("exact one-pass remaining allowance refused actual pending evidence", err)
			}
			if r.bounds != bounds {
				t.Fatal("tail inspection changed the original selected row")
			}
		})
	}
}
