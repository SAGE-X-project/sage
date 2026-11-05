package registry011

import (
	"encoding/json"
	"testing"
)

// These are scalar reservation tests. They create no final reader, source
// derivation, operating profile, native clock sample or network capability.
func TestProviderHistoryFinalReservationIsDisjointBeforeFirstSample(t *testing.T) {
	if json.Unmarshal([]byte(`{}`), &providerHistoryFinalBounds{}) == nil {
		t.Fatal("serialized DATA supplied a final reservation")
	}
	s := providerHistoryParentScalarFixture(t)
	store, selected := s.config.bounds, *s.config.bounds.history
	before, ok := providerHistoryWholeHold(store)
	if !ok {
		t.Fatal("existing complete hold")
	}
	selected.final = providerHistoryFinalBounds{
		read:      providerHistoryRecoveryBounds{work: candidateWorkVector{2: 19, 10: 71}, maximumFiles: 1, maximumBytes: 1},
		elapsedMS: 1,
	}
	store.history = &selected
	// Only this test's existing synthetic snapshot changes. No production
	// derivation constructor is supplied by a numeric reservation row.
	selected.derivation.snapshot = selected
	if !selected.valid(store) {
		t.Fatal("separate finite final reservation")
	}
	after, ok := providerHistoryWholeHold(store)
	if !ok || after.retained != before.retained || after.clocks != before.clocks+2 {
		t.Fatal("final inspection borrowed storage or omitted original samples")
	}
	for coordinate := range after.work {
		want := before.work[coordinate]
		if coordinate == 2 {
			want += 19
		}
		if coordinate == 10 {
			want += 73
		}
		if after.work[coordinate] != want {
			t.Fatal("final inspection borrowed a different resource domain", coordinate)
		}
	}
	if selected.recovery != s.config.bounds.history.recovery || selected.future != s.config.bounds.history.future ||
		providerHistoryPreIUpper(store) != providerHistoryPreIUpper(s.config.bounds) ||
		store.firstAdmission != s.config.bounds.firstAdmission {
		t.Fatal("final row changed execution, failure inspection or old initial seed")
	}
	if _, exists := s.config.bounds.history.object()["successful_final_inspection"]; exists {
		t.Fatal("unselected final row changed an existing candidate commitment")
	}
	if _, exists := selected.object()["successful_final_inspection"]; !exists {
		t.Fatal("selected final row omitted from committed profile")
	}
}

func TestProviderHistorySemanticReadReservationsRemainDisjointFromExecution(t *testing.T) {
	s := providerHistoryParentScalarFixture(t)
	store, selected := s.config.bounds, *s.config.bounds.history
	active := s.activeMeter.Load()
	before, ok := providerHistoryWholeHold(store)
	if !ok {
		t.Fatal("existing original whole hold")
	}
	recovery := candidateWorkVector{3: 5, 4: 7, 5: 11, 6: 13}
	for coordinate := 3; coordinate <= 6; coordinate++ {
		selected.recovery.work[coordinate] = recovery[coordinate]
	}
	selected.final = providerHistoryFinalBounds{
		read: providerHistoryRecoveryBounds{
			work: candidateWorkVector{2: 19, 3: 17, 4: 23, 5: 29, 6: 31, 10: 71}, maximumFiles: 1, maximumBytes: 1,
		},
		elapsedMS: 1,
	}
	store.history = &selected
	selected.derivation.snapshot = selected // Existing scalar UNIT selection before H0.
	after, valid := providerHistoryWholeHold(store)
	if !selected.valid(store) || !valid || after.retained != before.retained || after.clocks != before.clocks+2 {
		t.Fatal("semantic read rows changed storage or the exact two final samples")
	}
	for coordinate, actual := range after.work {
		want := before.work[coordinate] + recovery[coordinate] + selected.final.read.work[coordinate]
		if coordinate == 10 {
			want += 2
		}
		if actual != want {
			t.Fatal("independent recovery and final rows overlapped, borrowed execution or were added more than once", coordinate)
		}
	}
	if selected.startup != s.config.bounds.history.startup || selected.future != s.config.bounds.history.future ||
		providerHistoryPreIUpper(store) != providerHistoryPreIUpper(s.config.bounds) || store.firstAdmission != s.config.bounds.firstAdmission ||
		s.history != nil || s.clock.read != 0 || s.activeMeter.Load() != active {
		t.Fatal("read reservation changed the original execution selection or created a live role")
	}
}

func TestProviderHistoryFinalReservationRejectsInvalidDomainsAndOverflow(t *testing.T) {
	base := providerHistoryFinalBounds{
		read:      providerHistoryRecoveryBounds{work: candidateWorkVector{2: 1, 10: 1}, maximumFiles: 1, maximumBytes: 1},
		elapsedMS: 1,
	}
	for _, mode := range []string{"files", "bytes", "decode", "work", "duration", "duration-limit", "request", "wire", "write", "late-clock-overflow"} {
		t.Run(mode, func(t *testing.T) {
			b := base
			switch mode {
			case "files":
				b.read.maximumFiles = 0
			case "bytes":
				b.read.maximumBytes = 0
			case "decode":
				b.read.work[2] = 0
			case "work":
				b.read.work[10] = 0
			case "duration":
				b.elapsedMS = 0
			case "duration-limit":
				b.elapsedMS = 300001
			case "request":
				b.read.work[0] = 1
			case "wire":
				b.read.work[1] = 1
			case "write":
				b.read.work[9] = 1
			case "late-clock-overflow":
				b.read.work[10] = ^uint64(0) - 1
			}
			if out, accepted := b.upper(); accepted || out != (providerScanUpper{}) {
				t.Fatal("invalid final row returned a complete or partial reservation", mode)
			}
		})
	}
	segment := base
	segment.read.work[8] = 1
	if out, ok := segment.upper(); ok || out != (providerScanUpper{}) {
		t.Fatal("final read acquired a segment allowance")
	}
	for _, coordinate := range []int{3, 4, 5, 6, 7} {
		b := base
		b.read.work[coordinate] = 1
		upper, ok := b.upper()
		if !ok || upper.work[coordinate] != 1 || upper.work[10] != 3 || upper.clocks != 2 {
			t.Fatal("final reservation lost its historical semantic dimension or borrowed clock work", coordinate)
		}
	}
}

func TestProviderHistoryFinalReservationRefusesParentCapacityAndSnapshotChanges(t *testing.T) {
	for _, mode := range []string{"files", "bytes", "lifetime", "clock-total", "snapshot", "sum-overflow"} {
		t.Run(mode, func(t *testing.T) {
			s := providerHistoryParentScalarFixture(t)
			store, selected := s.config.bounds, *s.config.bounds.history
			selected.final = providerHistoryFinalBounds{
				read:      providerHistoryRecoveryBounds{work: candidateWorkVector{2: 1, 10: 1}, maximumFiles: 1, maximumBytes: 1},
				elapsedMS: 1,
			}
			store.history = &selected
			switch mode {
			case "files":
				selected.final.read.maximumFiles = 2*selected.catalogEntries + 1
			case "bytes":
				selected.final.read.maximumBytes = 2*store.maximumStoredBytes + 1
			case "lifetime":
				selected.durationMS = 1
				selected.final.elapsedMS = 2
			case "clock-total":
				selected.startup.clocks = (1 << 20) - store.scan.maximumPreIClockReads - selected.future.clocks - 1
			case "sum-overflow":
				selected.recovery.work[10] = ^uint64(0)
			}
			selected.derivation.snapshot = selected
			if mode == "snapshot" {
				selected.final.read.work[10]++
			}
			if mode == "sum-overflow" {
				if out, ok := providerHistoryWholeHold(store); ok || out != (providerScanUpper{}) {
					t.Fatal("final row overflow returned a partial whole hold")
				}
			} else if selected.valid(store) {
				t.Fatal("invalid parent final reservation accepted", mode)
			}
		})
	}
}

func TestProviderHistoryFinalReservationCannotBorrowFirstAdmissionCapacity(t *testing.T) {
	s := providerHistoryParentScalarFixture(t)
	selected := *s.config.bounds.history
	selected.final = providerHistoryFinalBounds{
		read:      providerHistoryRecoveryBounds{work: candidateWorkVector{2: 1, 10: 1}, maximumFiles: 1, maximumBytes: 1},
		elapsedMS: 1,
	}
	selected.derivation.snapshot = selected
	s.config.bounds.history = &selected
	if !selected.valid(s.config.bounds) {
		t.Fatal("finite final row before original capacity check")
	}
	limits, retained, active, reads := *s.firstLimits, s.stored, s.activeMeter.Load(), s.clock.read
	if out, err := newProviderHistoryProgramBeforeOpenCandidate(s); err == nil || out != nil || s.history != nil || s.clock.history != nil ||
		*s.firstLimits != limits || s.stored != retained || s.activeMeter.Load() != active || s.clock.read != reads || s.setupOrigin != (ClockSample{}) {
		t.Fatal("final reservation borrowed original first-admission funds or sampled before refusal", err)
	}
}
