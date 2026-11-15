package registry011

import "testing"

func TestProviderScanInitialClaimRequiresItsOriginalCollection(t *testing.T) {
	s := providerScanStoreFixture(t)
	before, reads, held := s.stored, s.clock.read, s.scanControl.held
	target, err := s.scanControl.claimInitialTargetCandidate()
	if err == nil || target != nil || s.stored != before || s.clock.read != reads || s.scanControl.held != held ||
		s.scanControl.prepared != nil || s.failed || s.scanControl.failed || s.activeMeter.Load() != nil {
		t.Fatal("missing collection entered original preparation or acquired a target", err)
	}
}

func TestProviderScanClaimRefusalsPreserveTheOriginalCapacityBoundary(t *testing.T) {
	for _, mode := range []string{"retained-capacity", "deadline"} {
		t.Run(mode, func(t *testing.T) {
			s := providerScanStoreFixture(t)
			a := providerScanAttemptFixture(t, s, providerScanHighwater{})
			if mode == "retained-capacity" {
				// A numeric boundary in this private UNIT only. The selected
				// capacity is unchanged and no artificial file is written.
				s.stored = s.config.bounds.maximumStoredBytes
			} else {
				a.deadline--
			}
			before, reads, held := s.stored, s.clock.read, s.scanControl.held
			if err := s.scanControl.claim(a); err == nil || s.scanControl.prepared != nil || s.activeMeter.Load() != nil {
				t.Fatal("insufficient capacity or wrong original deadline produced a successful preparation", err)
			}
			if mode == "retained-capacity" {
				if s.stored != before || s.clock.read != reads || s.scanControl.held != held || s.failed || s.scanControl.failed {
					t.Fatal("pre-claim capacity refusal consumed a hold or observed originals")
				}
			} else {
				addition, ok := providerScanUpperSum(a.preparation, a.body)
				if !ok || s.stored != before+addition.retained || s.scanControl.held != addition || !s.failed || !s.scanControl.failed || s.clock.read != reads {
					t.Fatal("post-claim deadline refusal refunded the irreversible hold or sampled a clock")
				}
			}
		})
	}
}
