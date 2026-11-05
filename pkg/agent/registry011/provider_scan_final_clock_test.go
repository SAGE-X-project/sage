package registry011

import (
	"bytes"
	"os"
	"testing"
)

func TestProviderScanFinalClockRefusalKeepsWrittenOriginalsAndTheWholeHold(t *testing.T) {
	s := providerScanStoreFixture(t)
	c := s.scanControl
	a := providerScanAttemptFixture(t, s, c.pin)
	beforePin, beforeStored, beforeRaw := c.pin, s.stored, s.retainedRaw
	beforeTip, beforeSetup, beforeSequence := s.tip, s.setupTip, s.sequence
	held, heldOK := providerScanUpperSum(a.preparation, a.body)
	after, _, afterErr := providerScanSuccessor(s.config, beforePin, a)
	head, headErr := after.head(s.config)
	if !heldOK || afterErr != nil || headErr != nil {
		t.Fatal("ordinary scan records must be valid before the refusal", afterErr, headErr)
	}
	var observed *providerWorkMeter
	changedMeter := false
	originalSync := s.evidence.syncFile
	s.evidence.syncFile = func(file *os.File) error {
		// Observe this owned disk's existing preparation only. Preserve the
		// exact original synchronization and its error; never sample a clock
		// or copy, retarget, finish or revive the original meter.
		if meter := s.activeMeter.Load(); meter != nil {
			if observed != nil && observed != meter {
				changedMeter = true
			}
			observed = meter
		}
		return originalSync(file)
	}
	t.Cleanup(func() { s.evidence.syncFile = originalSync })

	// Spend all but the last read in this isolated UNIT. The original ceiling,
	// origin, policy and deadlines stay unchanged. The first sample is still
	// available; only the final sample must refuse after both HEAD writes.
	s.clock.mu.Lock()
	maximum := s.clock.maximum
	if maximum <= s.clock.read {
		s.clock.mu.Unlock()
		t.Fatal("fixture has no remaining original clock read")
	}
	s.clock.read = maximum - 1
	s.clock.mu.Unlock()

	if err := c.claim(a); err == nil || !s.failed || !c.failed || c.prepared != nil ||
		c.pin != beforePin || c.held != held || s.stored != beforeStored+held.retained ||
		s.retainedRaw != beforeRaw || s.tip != beforeTip || s.setupTip != beforeSetup ||
		s.sequence != beforeSequence || s.activeMeter.Load() != nil || s.last != nil || s.enrollment != "" {
		t.Fatal("final clock refusal refunded the hold, advanced a pin or returned usable metadata", err)
	}
	if observed == nil || changedMeter {
		t.Fatal("the original preparation meter was not retained for observation")
	}
	observed.mu.Lock()
	healthyFinalSample := observed.owner == observed && observed.scan == c &&
		observed.upper == a.preparation.work && observed.maximumClocks == a.preparation.clocks &&
		observed.clocks == 2 && !observed.failed && !observed.closed && !observed.authority
	beforeRetryWork, beforeRetryClocks := observed.actual, observed.clocks
	observed.mu.Unlock()
	if !healthyFinalSample {
		t.Fatal("an earlier readback or work refusal was mistaken for the original final clock call")
	}
	for _, disk := range []*retentionDisk{s.custody, s.evidence} {
		got, err := disk.read("SCAN-HEAD", candidateLedgerMetadataLimit)
		if err != nil || !bytes.Equal(got, head) {
			t.Fatal("refusal must occur after the ordinary HEAD was retained in both roots", err)
		}
	}
	s.clock.mu.Lock()
	reads, unchangedMaximum := s.clock.read, s.clock.maximum
	s.clock.mu.Unlock()
	if reads != maximum || unchangedMaximum != maximum {
		t.Fatal("final refusal renewed or changed the original clock ceiling")
	}
	if err := c.claim(a); err == nil || c.held != held || s.stored != beforeStored+held.retained || c.prepared != nil {
		t.Fatal("a failed final sample allowed the held preparation to be retried", err)
	}
	s.clock.mu.Lock()
	readsAfterRetry := s.clock.read
	s.clock.mu.Unlock()
	if readsAfterRetry != reads || s.activeMeter.Load() != nil {
		t.Fatal("retry observed another clock sample or retained a meter")
	}
	observed.mu.Lock()
	unchangedWork := observed.actual == beforeRetryWork && observed.clocks == beforeRetryClocks
	observed.mu.Unlock()
	if !unchangedWork {
		t.Fatal("retry changed the original failed preparation's observations")
	}
}
