package registry011

import "testing"

// These are fixed UNIT variations of read-only completion DATA after the
// existing normal owned exchange has finished. Only local cursor/semantic
// value copies change; shared arrays and maps are never mutated. This does
// not retry a driver, restore a failed owner, reset a debit or sample a clock.
func historyCollectionCompletionUnitCases(t *testing.T, group *providerHistoryGroupOwner, settled *providerHistorySettledResults) {
	t.Helper()
	for _, mode := range []string{"completed-ordinal", "coordinator-incomplete", "custodian-incomplete"} {
		t.Run(mode, func(t *testing.T) {
			s := group.store
			s.mu.Lock()
			defer s.mu.Unlock()
			stage := group.coordinator.stage
			if done, valid := group.collectionSettlementStateLocked(settled, stage); !done || !valid {
				t.Fatal("normal completed DATA precursor is missing")
			}
			left, right := group.coordinator.collection, group.custodian.collection
			work, clocks, pin, held := group.meter.actual, s.clock.read, group.pin, s.stored
			original, ceiling, future := group.parent.original, group.parent.clockCeiling, group.parent.bounds.future
			defer func() {
				group.coordinator.collection, group.custodian.collection = left, right
				if done, valid := group.collectionSettlementStateLocked(settled, stage); !done || !valid ||
					group.meter.actual != work || s.clock.read != clocks || group.pin != pin || s.stored != held ||
					group.parent.original != original || group.parent.clockCeiling != ceiling || group.parent.bounds.future != future ||
					group.failed || group.parent.failed || s.failed || s.activeMeter.Load() != nil {
					t.Error("read-only UNIT check changed original completion or execution resources")
				}
			}()
			leftCopy, rightCopy := *left, *right
			leftSemantic, rightSemantic := *left.semantic, *right.semantic
			leftCopy.owner, rightCopy.owner = &leftCopy, &rightCopy
			leftSemantic.owner, rightSemantic.owner = &leftSemantic, &rightSemantic
			leftCopy.semantic, rightCopy.semantic = &leftSemantic, &rightSemantic
			group.coordinator.collection, group.custodian.collection = &leftCopy, &rightCopy
			switch mode {
			case "completed-ordinal":
				leftCopy.completed--
			case "coordinator-incomplete":
				leftSemantic.windows = 0
			case "custodian-incomplete":
				rightSemantic.windows = 0
			}
			if !group.successfulSettlementShapeLocked() || !leftCopy.valid() || !rightCopy.valid() {
				t.Fatal("UNIT input did not reach completion-specific consistency checks")
			}
			if done, valid := group.collectionSettlementStateLocked(settled, stage); done || valid {
				t.Fatal("inconsistent reader DATA was accepted as a complete collection")
			}
		})
	}
}
