package registry011

import (
	"reflect"
	"testing"
)

func TestProviderHistoryMissingRecoveryCheckpointCannotPrepareOrMatchOriginalData(t *testing.T) {
	for _, r := range []*providerHistoryRecoveryOwner{nil, {}} {
		if r.prepareInitialCheckpointLocked(nil, providerHistoryCustodyScope{}, admissionTestDigest("a")) != nil ||
			r.prepareWriteCheckpointLocked(nil) || r.pendingCheckpointShapeLocked(nil) ||
			r.pendingCheckpointDataMatchesLocked(nil) || r.checkpointSelectionLocked(nil) {
			t.Fatal("absent original checkpoint owner supplied preparation, data equality or selection")
		}
	}
}

func TestProviderHistoryFailedOriginalMeterCannotAcknowledgePendingCheckpointData(t *testing.T) {
	// The original writer really fails at its first evidence sync. Its sealed
	// pending checkpoint survives, and only the separate original read-only
	// inspection lane is active. No failed allowance or writer is restored.
	o, r := historyRecoveryPendingInputFixture(t)
	s, m := o.store, o.meter
	s.mu.Lock()
	defer s.mu.Unlock()
	claim, pending, confirmed := s.frontierFiles.historyPending, r.pending, r.confirmed
	if claim == nil || pending == nil || claim.checkpoint != pending || confirmed != r.initial ||
		!r.pendingCheckpointShapeLocked(claim) || !o.failed || !m.failed || !m.closed || !r.active ||
		r.failed || r.closed || r.meter.failed || r.meter.closed || s.activeMeter.Load() != nil {
		t.Fatal("actual failed writer, intact pending shape and distinct read-only observer required")
	}
	work, inspection, reads, files, bytes := m.actual, r.meter.actual, s.clock.read, r.inputFiles, r.inputBytes
	pin, ack, held := o.pin, o.last, s.stored
	originals := providerHistoryProvisionedOriginalBytes(t, s.config)
	if r.pendingCheckpointDataMatchesLocked(claim) || !r.pendingCheckpointShapeLocked(claim) ||
		m.actual != work || r.meter.actual != inspection || s.clock.read != reads || r.inputFiles != files || r.inputBytes != bytes ||
		r.pending != pending || r.confirmed != confirmed || o.pin != pin || o.last != ack || s.stored != held ||
		!m.failed || !m.closed || !r.active || r.failed || r.closed || r.meter.failed || r.meter.closed ||
		s.activeMeter.Load() != nil || !reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, s.config)) {
		t.Fatal("failed execution observer acknowledged sealed DATA, renewed a role or changed originals")
	}
}
