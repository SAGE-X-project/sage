package registry011

import (
	"encoding/json"
	"testing"
)

func TestProviderHistoryClockRequiresOneNewMeteredEventAndExactClaim(t *testing.T) {
	s, p := providerHistoryParentBoundScalarFixture(t)
	m := s.activeMeter.Load()
	before := s.clock.read
	if p.issueClockClaimLocked(m) != nil {
		t.Fatal("previous sample's event minted another call")
	}
	if !s.event(providerClockEvent, 0) {
		t.Fatal("one new event")
	}
	claim := p.issueClockClaimLocked(m)
	if claim == nil || claim.used || p.issueClockClaimLocked(m) != nil {
		t.Fatal("one event issued multiple native claims")
	}
	copy := *claim
	if _, err := s.clock.sampleForHistory(p, &copy); err == nil || s.clock.read != before || claim.used {
		t.Fatal("copied claim consumed the original")
	}
	if _, err := s.clock.sampleForHistory(p, nil); err == nil || s.clock.read != before {
		t.Fatal("parent pointer alone spent future clock hold")
	}
	if _, err := s.clock.sampleForHistory(p, claim); err != nil || !claim.used || s.clock.read != before+1 {
		t.Fatal("one exact previously metered native call", err)
	}
	if _, err := s.clock.sampleForHistory(p, claim); err == nil || s.clock.read != before+1 || p.issueClockClaimLocked(m) != nil {
		t.Fatal("used claim/event was restored")
	}
	var decoded providerHistoryClockClaim
	if json.Unmarshal([]byte(`{}`), &decoded) == nil {
		t.Fatal("JSON reconstructed a native claim")
	}
}

func TestProviderHistoryClockIdleAndDoneCannotSpendOrCharge(t *testing.T) {
	s, p := providerHistoryParentBoundScalarFixture(t)
	m := s.activeMeter.Load()
	beforeRead, beforeClocks, beforeWork := s.clock.read, m.clocks, m.actual
	s.activeMeter.Store(nil)
	if _, err := s.samplePrimitive(); err == nil {
		t.Fatal("idle store sampled the future lifetime")
	}
	if _, err := s.sample(); err == nil || p.issueClockClaimLocked(m) != nil || s.clock.read != beforeRead || m.clocks != beforeClocks || m.actual != beforeWork {
		t.Fatal("idle refusal touched a clock or event")
	}
	// An inert DONE role has no private execution authority. This tests only
	// early refusal; no fake ledger, original readback or dispatch is accepted.
	o := &providerFirstOperationOwner{store: s, state: providerFirstOperationDone}
	o.owner, s.firstOperation, p.startupClaimed = o, o, true
	m.firstOperation, o.execution = o, m
	s.activeMeter.Store(m)
	if _, err := s.samplePrimitive(); err == nil || s.clock.read != beforeRead || m.clocks != beforeClocks || m.actual != beforeWork {
		t.Fatal("DONE role consumed a future event or sample")
	}
}

func TestProviderHistoryClockRegistersOnlyMatchedInspectionPhases(t *testing.T) {
	s, p := providerHistoryParentBoundScalarFixture(t)
	s.activeMeter.Store(nil)
	c := &providerScanControl{store: s, failed: true}
	c.owner, s.scanControl = c, c
	upper := providerScanUpper{work: candidateWorkVector{0, 0, 1, 0, 0, 0, 0, 0, 0, 1, 100}, retained: 2, clocks: 4}
	if !p.claimInspectionLocked(providerScanUpper{}, upper, upper) {
		t.Fatal("one irreversibly held P+B prefix")
	}
	c.held = p.inspection
	m := &providerWorkMeter{scan: c, upper: upper.work, maximumClocks: upper.clocks}
	m.owner = m
	if !p.bindInspectionMeterLocked(m, false) || p.bindInspectionMeterLocked(m, false) {
		t.Fatal("preparation observer was replaced or repeated")
	}
	s.activeMeter.Store(m)
	if !p.clockRoleLocked(m) {
		t.Fatal("exact matched preparation not selected")
	}
	c.failed = false
	if p.clockRoleLocked(m) {
		t.Fatal("completed preparation retained clock role")
	}
	body := &providerScanBody{control: c, active: true}
	body.owner = body
	bm := &providerWorkMeter{scan: c, scanBody: body, upper: upper.work, maximumClocks: upper.clocks}
	bm.owner, body.meter, s.scanBody = bm, bm, body
	if !p.bindInspectionMeterLocked(bm, true) || p.bindInspectionMeterLocked(bm, true) {
		t.Fatal("body observer was replaced or repeated")
	}
	s.activeMeter.Store(bm)
	if !p.clockRoleLocked(bm) {
		t.Fatal("exact matched active body not selected")
	}
	body.active = false
	if p.clockRoleLocked(bm) {
		t.Fatal("idle body still owned a clock role")
	}
	f := &providerFirstAdmissionOwner{store: s, meter: bm, state: providerFirstAdmissionStarted}
	s.firstAdmissionOwner, bm.firstAdmission = f, f
	if !p.clockRoleLocked(bm) {
		t.Fatal("same started F observer was lost")
	}
	f.state = providerFirstAdmissionRetained
	if p.clockRoleLocked(bm) {
		t.Fatal("retained F observer kept its role")
	}
}

func TestProviderHistoryClockFirstPUsesOnlyItsPreheldLiveObserver(t *testing.T) {
	s, p := providerHistoryParentBoundScalarFixture(t)
	p.startupClaimed = true
	c := &providerIntegration{store: s}
	m := newProviderWorkMeter(c, providerOperationReservation{})
	o := &providerFirstOperationOwner{store: s, integration: c, execution: m, state: providerFirstOperationPrepared}
	o.owner, s.firstOperation, m.firstOperation = o, o, o
	s.activeMeter.Store(m)
	if !p.clockRoleLocked(m) {
		t.Fatal("matched fixed first-P observer was lost")
	}
	for _, state := range []int{providerFirstOperationDone, providerFirstOperationFailed} {
		o.state = state
		if p.clockRoleLocked(m) {
			t.Fatal("terminal first P retained clock role")
		}
	}
	o.state, p.startupClaimed = providerFirstOperationPrepared, false
	if p.clockRoleLocked(m) {
		t.Fatal("unheld first P acquired a clock role")
	}
}
