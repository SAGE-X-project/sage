package registry011

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProviderHistoryCollectionBoundaryRequiresLiveCallerContext(t *testing.T) {
	for _, mode := range []string{"nil", "canceled"} {
		t.Run(mode, func(t *testing.T) {
			c, headers, methods := historyCollectionBoundaryFixture(t, 13, 4)
			table := historyCollectionBoundaryPlan(t, c, methods)
			results := historyCollectionBoundaryResults(t, c, headers, table)
			before := c.commitmentObserved(providerReplayUnitMeter())
			m := c.semantic.observer
			actual := m.actual
			var ctx context.Context
			if mode == "canceled" {
				canceled, cancel := context.WithCancel(context.Background())
				cancel()
				ctx = canceled
			}
			if next, err := c.advanceBoundaryObserved(ctx, table, results, methods, m); err == nil || next != nil || before == "" ||
				c.commitmentObserved(providerReplayUnitMeter()) != before || c.completed != 0 || c.phase != "open" || c.client != "" || m.actual != actual || m.failed {
				t.Fatal("stopped caller advanced or observed a collection boundary")
			}
		})
	}
}

func TestProviderHistoryCollectionSettlementCancellationKeepsConsumedCharge(t *testing.T) {
	for _, mode := range []string{"complete-nil", "complete-canceled", "failed-nil", "failed-canceled"} {
		t.Run(mode, func(t *testing.T) {
			c, headers, methods := historyCollectionBoundaryFixture(t, 13, 4)
			l := historyCollectionLedger(t, c)
			settlement, results := historyCollectionPendingSettlement(t, l, headers, methods)
			if strings.HasPrefix(mode, "failed-") {
				var err error
				settlement, err = newProviderHistoryGroupSettlement(l.ticket, l.consumption, l.head, admissionTestDigest("2"), "failed", admissionTestDigest("3"), nil)
				if err != nil {
					t.Fatal("valid failed settlement DATA precursor", err)
				}
				results = nil
			}
			var ctx context.Context
			if strings.HasSuffix(mode, "-canceled") {
				canceled, cancel := context.WithCancel(context.Background())
				cancel()
				ctx = canceled
			}
			m := l.collection.semantic.observer
			before, charged, actual := l.head, l.charged, m.actual
			ticket, consumption := l.ticket, l.consumption
			cursor := l.collection.commitmentObserved(providerReplayUnitMeter())
			if l.settleOriginalsObserved(ctx, settlement, results, methods, m) == nil || !l.failed || l.terminal || l.head != before || l.charged != charged ||
				l.ticket != ticket || l.consumption != consumption || l.phase != "consumed" || l.stage != 0 || m.actual != actual ||
				l.collection.commitmentObserved(providerReplayUnitMeter()) != cursor || cursor == "" {
				t.Fatal("stopped S published a new root, failure receipt, cursor or refund")
			}
		})
	}
}

// These runtime unit scenarios use the existing owned TLS/temporary-directory
// fixtures. They never contact an RPC, submit a transaction or use operator keys.
func TestProviderHistoryOriginalOperationsRejectStoppedCallersBeforeEffects(t *testing.T) {
	for _, mode := range []string{"admit-nil", "admit-canceled", "consume-nil", "consume-canceled", "dispatch-nil", "dispatch-canceled"} {
		t.Run(mode, func(t *testing.T) {
			var o *providerHistoryGroupOwner
			var ack *providerHistoryGroupAcknowledgment
			var upper providerHistoryGroupUpper
			switch {
			case strings.HasPrefix(mode, "admit-"):
				o, upper = historyAdmissionWriterFixture(t)
			case strings.HasPrefix(mode, "consume-"):
				o, ack = historyConsumptionFixture(t, 1)
			default:
				o, ack = historyDispatchReplayFixture(t, 1)
			}
			var ctx context.Context
			if strings.HasSuffix(mode, "-canceled") {
				canceled, cancel := context.WithCancel(context.Background())
				cancel()
				ctx = canceled
			}
			s, m := o.store, o.meter
			pin, last, phase, actual, reads := o.pin, o.last, o.phase, m.actual, s.clock.read
			held, retention, pending, order := s.stored, o.retention, s.frontierFiles.historyPending, o.order
			var err error
			switch {
			case strings.HasPrefix(mode, "admit-"):
				var out *providerHistoryGroupAcknowledgment
				out, err = o.retainAdmissionCandidate(ctx, "selection", []*providerIntegrationRequest{historyGroupSimple(t)}, upper)
				if out != nil {
					t.Fatal("stopped admission returned an acknowledgment")
				}
			case strings.HasPrefix(mode, "consume-"):
				var out *providerHistoryGroupAcknowledgment
				out, err = o.retainConsumptionCandidate(ctx, ack)
				if out != nil {
					t.Fatal("stopped consumption returned an acknowledgment")
				}
			default:
				var out *providerHistoryGroupDispatch
				out, err = o.prepareDispatchCandidate(ctx, ack)
				if out != nil {
					t.Fatal("stopped dispatch returned a capability")
				}
			}
			if err == nil || o.pin != pin || o.last != last || o.phase != phase || m.actual != actual || s.clock.read != reads || s.stored != held ||
				o.retention != retention || s.frontierFiles.historyPending != pending || o.order != order || o.dispatch != nil || o.failed || s.failed || m.failed || m.closed || s.activeMeter.Load() != nil {
				t.Fatal("stopped caller entered an original transaction or changed its held state")
			}
		})
	}
}

func TestProviderHistoryOriginalCancellationAroundRenameKeepsEarlierArchive(t *testing.T) {
	for _, mode := range []string{"admit-before-rename", "admit-after-rename", "consume-before-rename", "consume-after-rename", "settle-before-rename", "settle-after-rename"} {
		t.Run(mode, func(t *testing.T) {
			o := historyMultistageOwner(t, 2)
			first := historyMultistageSettle(t, o, 1)
			s, r, m := o.store, o.parent.recovery, o.meter
			archive := r.confirmed.memberArchive
			if err := o.advanceStageCandidate(context.Background(), first.ack); err != nil {
				t.Fatal("genuine first-stage archive before cancellation", err)
			}
			phase := strings.SplitN(mode, "-", 2)[0]
			var admission *providerHistoryGroupAcknowledgment
			var dispatch *providerHistoryGroupDispatch
			switch phase {
			case "consume":
				var err error
				admission, err = o.retainAdmissionCandidate(context.Background(), "identity", []*providerIntegrationRequest{historyGroupRequest(t, "web3_clientVersion", []any{})}, historyMultistageTestUpper(s.config.capture))
				if err != nil || admission == nil {
					t.Fatal("genuine second T before C cancellation", err)
				}
			case "settle":
				dispatch = historyMultistageDispatch(t, o, 2)
				historyRetainNormalPrefix(t, o, dispatch, 1)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			before, last, confirmed, held, original := o.pin, o.last, r.confirmed, s.stored, o.parent.original
			beforeRetention := o.retention
			var retainedAtCancel providerHistoryStorageCharge
			if archive == nil || archive != first.ack.claim.checkpoint.memberArchive {
				t.Fatal("actual earlier immutable member archive precursor")
			}
			oldRoot, archiveValue := first.ack.claim.checkpoint.model, *archive
			originals := providerHistoryProvisionedOriginalBytes(t, s.config)
			afterRename := strings.HasSuffix(mode, "after-rename")
			reached := false
			syncFile := s.custody.syncFile
			s.custody.syncFile = func(file *os.File) error {
				if err := syncFile(file); err != nil {
					return err
				}
				c := s.frontierFiles.historyPending
				if reached || c == nil || c.frame.stage != 2 || c.frame.phase != phase {
					return nil
				}
				if afterRename && c.renamed || !afterRename && filepath.Base(file.Name()) == s.name("HEAD.pending", c.frame.after.ordinal) {
					reached = true
					retainedAtCancel = o.retention
					cancel() // Ordinary cancellation after a successful owned sync.
				}
				return nil
			}
			var operationErr error
			switch phase {
			case "admit":
				var out *providerHistoryGroupAcknowledgment
				out, operationErr = o.retainAdmissionCandidate(ctx, "identity", []*providerIntegrationRequest{historyGroupRequest(t, "web3_clientVersion", []any{})}, historyMultistageTestUpper(s.config.capture))
				if out != nil {
					t.Fatal("canceled T acknowledged")
				}
			case "consume":
				var out *providerHistoryGroupAcknowledgment
				out, operationErr = o.retainConsumptionCandidate(ctx, admission)
				if out != nil {
					t.Fatal("canceled C acknowledged")
				}
			case "settle":
				var out *providerHistorySettledResults
				out, operationErr = o.retainSettlementCandidate(ctx, dispatch)
				if out != nil {
					t.Fatal("canceled S published results")
				}
			}
			s.custody.syncFile = syncFile
			if operationErr == nil || !reached || ctx.Err() != context.Canceled || !o.failed || !s.failed || !m.failed || !m.closed ||
				o.pin != before || o.last != last || s.frontierFiles.pin != before || r.confirmed != confirmed || s.stored != held || o.parent.original != original ||
				s.activeMeter.Load() != nil || s.frontierFiles.historyPending == nil || s.frontierFiles.historyPending.used || s.frontierFiles.historyPending.renamed != afterRename ||
				retainedAtCancel.physical <= beforeRetention.physical || o.retention != retainedAtCancel || first.ack.claim.checkpoint.model != oldRoot || *archive != archiveValue || first.ack.claim.checkpoint.memberArchive != archive ||
				!archive.shapeLocked(r, archive.checkpoint) || archive.order.state != providerHistoryOrderClosed {
				t.Fatal("cancellation published, refunded, rewound or lost the earlier archive")
			}
			after := providerHistoryProvisionedOriginalBytes(t, s.config)
			headName := filepath.Join(s.config.custodyDirectory, "HEAD")
			wantHead := originals[headName]
			if afterRename {
				wantHead = s.frontierFiles.historyPending.highwater
			}
			if after[headName] != wantHead {
				t.Fatal("cancellation confused physical HEAD with its unpublished logical pin")
			}
			for name, raw := range originals {
				if name != headName && after[name] != raw {
					t.Fatal("cancellation rewrote an earlier immutable original", name)
				}
			}
		})
	}
}
