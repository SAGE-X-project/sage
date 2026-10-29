package registry011

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func providerHistoryGroupStartFixture(t *testing.T, funded bool) *providerCustodyStore {
	t.Helper()
	s := providerHistoryProvisionedInspectionFixture(t, providerScanFixtureSelection{historyProgram: true, historyGroupInit: funded})
	first, err := newProviderFirstOperationCandidate(s)
	if err != nil {
		t.Fatal("original first owner", err)
	}
	connection := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, providerFirstOperationResponse())
	if _, err := first.executeCandidateWithConnection(context.Background(), connection); err != nil || !s.history.startReadyLocked() {
		t.Fatal("original first exchange handoff", err)
	}
	return s
}

func TestProviderHistoryGroupOwnerInitializesIndependentModelsFromOriginals(t *testing.T) {
	s := providerHistoryGroupStartFixture(t, true)
	p, stored, raw, reads, pin := s.history, s.stored, s.retainedRaw, s.clock.read, s.frontierFiles.pin
	before := providerHistoryProvisionedOriginalBytes(t, s.config)
	o, err := newProviderHistoryGroupOwnerCandidate(s)
	if err != nil || o == nil {
		t.Fatal("funded original initialization", err)
	}
	initial := providerIntegrationJSON(map[string]any{
		"format": "sage-provider-history-initial-model-v1", "program": p.bounds.program, "profile": s.config.profile, "cost_revision": p.bounds.costs,
		"initial_frontier": pin.frontier, "first_exchange": p.start.exchange, "budget": s.frontierReplay.current.budget, "inventory": s.frontierReplay.current.objects,
	})
	want := providerCandidateHash("sage-provider-history-initial-model-v1", initial)
	if p.group != o || o.parent != p || o.start != p.start || o.phase != "prepared" || o.failed ||
		o.coordinator == nil || o.custodian == nil || o.coordinator == o.custodian || o.coordinator.owner != o.coordinator || o.custodian.owner != o.custodian ||
		o.coordinator.head != o.custodian.head || o.coordinator.initial != want || o.custodian.initial != want ||
		o.coordinator.stage != 0 || o.custodian.stage != 0 || o.coordinator.ticket != nil || o.custodian.ticket != nil {
		t.Fatal("initial models were shared or detached from retained originals")
	}
	m := o.meter
	upper, upperOK := providerHistoryGroupInitializationUpper(s.config.bounds)
	for coordinate, actual := range m.actual {
		if !upperOK || actual > upper[coordinate] {
			t.Fatal("initialization exceeded its prospective logical envelope", coordinate)
		}
	}
	if m.historyGroup != o || m.integration != nil || m.scan != nil || m.parent != nil || m.clocks != 2 || m.historyClockClaims != 2 ||
		m.upper != p.bounds.future.work || m.maximumClocks != p.bounds.future.clocks || m.actual[0] != 0 || m.actual[9] != 0 || m.actual[2] == 0 || m.failed || m.closed ||
		s.activeMeter.Load() != nil || s.clock.read != reads+2 || s.stored != stored || s.retainedRaw != raw || s.frontierFiles.pin != pin || s.sequence != 4 || s.last != nil ||
		!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
		t.Fatal("initialization renewed allowance, mutated originals or manufactured an ACK")
	}
	actual, clocks := m.actual, s.clock.read
	if again, err := newProviderHistoryGroupOwnerCandidate(s); err == nil || again != nil || p.group != o || m.actual != actual || s.clock.read != clocks {
		t.Fatal("the first bridge initialized a second controller")
	}
}

func TestProviderHistoryGroupOwnerRefusesChangedRootAndLockIdentity(t *testing.T) {
	for _, mode := range []string{"custody-root", "custody-lock", "evidence-root", "evidence-lock"} {
		t.Run(mode, func(t *testing.T) {
			s := providerHistoryGroupStartFixture(t, true)
			other := t.TempDir()
			alternate := filepath.Join(other, "unit-identity")
			if err := os.WriteFile(alternate, nil, 0600); err != nil {
				t.Fatal(err)
			}
			rootInfo, rootErr := os.Stat(other)
			lockInfo, lockErr := os.Stat(alternate)
			if rootErr != nil || lockErr != nil {
				t.Fatal("isolated unit identity", rootErr, lockErr)
			}
			// Inert unit fault in selected metadata only. No product path,
			// lock, original file, epoch or host permission is replaced.
			switch mode {
			case "custody-root":
				s.custody.directoryInfo = rootInfo
			case "custody-lock":
				s.custody.lockInfo = lockInfo
			case "evidence-root":
				s.evidence.directoryInfo = rootInfo
			case "evidence-lock":
				s.evidence.lockInfo = lockInfo
			}
			before, reads, held := providerHistoryProvisionedOriginalBytes(t, s.config), s.clock.read, s.stored
			if got, err := newProviderHistoryGroupOwnerCandidate(s); err == nil || got != nil {
				t.Fatal("changed root or lock identity initialized models", mode)
			}
			failed := s.history.group
			if failed == nil || !failed.failed || !failed.meter.failed || !failed.meter.closed || failed.meter.actual[10] == 0 ||
				failed.meter.clocks != 0 || s.clock.read != reads || failed.coordinator != nil || failed.custodian != nil ||
				!s.failed || !s.history.failed || !s.clock.historyRevoked || s.activeMeter.Load() != nil || s.stored != held ||
				!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
				t.Fatal("identity refusal read originals, renewed capacity or returned prepared models", mode)
			}
		})
	}
}

func TestProviderHistoryGroupInitializationEnvelopeRejectsArithmeticOverflow(t *testing.T) {
	for _, n := range []uint64{1 << 32, 1 << 63, ^uint64(0)} {
		if upper, ok := providerHistoryGroupInitializationUpper(providerIntegrationBounds{maximumMetadataPerCommit: n}); ok || upper != (candidateWorkVector{}) {
			t.Fatal("unrepresentable initialization envelope", n)
		}
	}
	upper, ok := providerHistoryGroupInitializationUpper(providerIntegrationBounds{maximumMetadataPerCommit: 64 << 10})
	if !ok || upper[2] != 128<<10 || upper[10] == 0 || upper[0] != 0 || upper[9] != 0 {
		t.Fatal("initialization envelope introduced requests or writes")
	}
}

func TestProviderHistoryGroupOwnerQuarantinesUnavailableNativeCallBudget(t *testing.T) {
	for _, allowed := range []uint64{0, 1} {
		t.Run([]string{"begin", "end"}[allowed], func(t *testing.T) {
			s := providerHistoryGroupStartFixture(t, true)
			reads, held, before := s.clock.read, s.stored, providerHistoryProvisionedOriginalBytes(t, s.config)
			// Inert unit exhaustion of the existing parent ceiling. The
			// backend, original epoch and selected native origin are intact.
			s.history.clockCeiling = reads + allowed
			if o, err := newProviderHistoryGroupOwnerCandidate(s); err == nil || o != nil {
				t.Fatal("unavailable original native call returned a controller")
			}
			failed := s.history.group
			if failed == nil || !failed.failed || !failed.meter.failed || !failed.meter.closed ||
				failed.meter.clocks != allowed+1 || failed.meter.historyClockClaims != allowed+1 || s.clock.read != reads+allowed ||
				(allowed == 0) != (failed.meter.actual[2] == 0) || failed.coordinator != nil || failed.custodian != nil ||
				!s.failed || !s.history.failed || !s.clock.historyRevoked || s.activeMeter.Load() != nil || s.stored != held ||
				!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
				t.Fatal("native-call refusal renewed capacity or retained an unqualified model", allowed)
			}
		})
	}
}

func TestProviderHistoryGroupOwnerRefusesUnfundedInitializationBeforeObservation(t *testing.T) {
	s := providerHistoryGroupStartFixture(t, false)
	reads, held, before := s.clock.read, s.stored, providerHistoryProvisionedOriginalBytes(t, s.config)
	o, err := newProviderHistoryGroupOwnerCandidate(s)
	failed := s.history.group
	if err != errProviderIntegration || o != nil || failed == nil || !failed.failed || failed.phase != "failed" || !s.failed || !s.history.failed || !s.clock.historyRevoked ||
		failed.meter.actual != (candidateWorkVector{}) || failed.meter.clocks != 0 || !failed.meter.closed || !failed.meter.failed ||
		s.activeMeter.Load() != nil || s.clock.read != reads || s.stored != held || !reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
		t.Fatal("unfunded initialization reached original I/O or refunded its hold", err)
	}
	if _, err := newProviderHistoryGroupOwnerCandidate(s); err == nil || s.history.group != failed || s.stored != held {
		t.Fatal("failed initialization reset its live owner")
	}
}

func TestProviderHistoryGroupOwnerQuarantinesUnavailableOriginals(t *testing.T) {
	for _, mode := range []string{"different-head", "missing-frontier"} {
		t.Run(mode, func(t *testing.T) {
			s := providerHistoryGroupStartFixture(t, true)
			if mode == "different-head" {
				if err := os.WriteFile(filepath.Join(s.config.custodyDirectory, "HEAD"), []byte("unit fixture: mismatched original"), 0600); err != nil {
					t.Fatal(err)
				}
			} else if err := os.Remove(filepath.Join(s.config.custodyDirectory, providerCustodyName("frontier", 4))); err != nil {
				t.Fatal(err)
			}
			before, held := providerHistoryProvisionedOriginalBytes(t, s.config), s.stored
			if o, err := newProviderHistoryGroupOwnerCandidate(s); err != errProviderIntegration || o != nil || s.history.group == nil || !s.history.group.failed ||
				!s.history.failed || !s.failed || !s.clock.historyRevoked || s.activeMeter.Load() != nil || s.stored != held || s.sequence != 4 || s.last != nil ||
				!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
				t.Fatal("unavailable original supplied a model, rewrite or new hold", err)
			}
		})
	}
}

func TestProviderHistoryGroupOwnerIdleAndCopiesCannotClaimClockOrPhase(t *testing.T) {
	var absent *providerCustodyStore
	if o, err := newProviderHistoryGroupOwnerCandidate(absent); err == nil || o != nil {
		t.Fatal("absent store initialized a group owner")
	}
	s := providerHistoryGroupStartFixture(t, true)
	o, err := newProviderHistoryGroupOwnerCandidate(s)
	if err != nil {
		t.Fatal(err)
	}
	m, reads := o.meter, s.clock.read
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeMeter.Store(m)
	if _, err := s.samplePrimitive(); err == nil || s.clock.read != reads || o.activeLocked(m) || m.hasOwner() {
		t.Fatal("idle group obtained an initialization or execution role")
	}
	s.activeMeter.Store(nil)
	copy := *o
	copy.phase = "initializing"
	if copy.activeLocked(m) || copy.fits(m) {
		t.Fatal("copied controller retained the original role")
	}
	other := newProviderWorkMeter(nil, providerOperationReservation{work: m.reservation, execution: m.upper})
	other.historyGroup, other.authority, other.maximumClocks = o, true, m.maximumClocks
	s.activeMeter.Store(other)
	if other.hasOwner() || s.history.clockRoleLocked(other) {
		t.Fatal("replacement observer borrowed the original initialization")
	}
	s.activeMeter.Store(nil)
	if json.Unmarshal([]byte(`{}`), &providerHistoryGroupOwner{}) == nil || json.Unmarshal([]byte(`{}`), &providerHistoryCustodyFrame{}) == nil {
		t.Fatal("JSON reconstructed live ownership")
	}
}
