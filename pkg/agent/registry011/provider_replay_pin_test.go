package registry011

import (
	"bytes"
	"testing"
)

func TestProviderReplayPinRequestsAndPartsObserveEveryRevalidation(t *testing.T) {
	selected := admissionTestDigest("a")
	plain, err := providerPinRequests(selected)
	meter := providerReplayUnitMeter()
	observed, observedErr := providerPinRequestsObserved(selected, meter)
	if err != nil || observedErr != nil {
		t.Fatal(err, observedErr)
	}
	for n := range plain {
		if *plain[n] != *observed[n] {
			t.Fatal("observed pin request changed exact wire", n)
		}
	}
	results := providerPinResultFixture(t)
	meter = providerReplayUnitMeter()
	for n, raw := range results {
		if !providerCheckPinPartObserved(n, raw, meter) {
			t.Fatal("normal pinned response refused", n)
		}
	}
	before := meter.actual
	for n, raw := range results {
		if !providerCheckPinPartObserved(n, raw, meter) {
			t.Fatal("same retained response refused on second validation", n)
		}
	}
	for coordinate, value := range before {
		if meter.actual[coordinate] != value*2 {
			t.Fatal("second validation reused first-pass work", coordinate)
		}
	}
	for n, raw := range results {
		zero := providerReplayUnitMeter()
		zero.upper = candidateWorkVector{}
		if providerCheckPinPartObserved(n, raw, zero) || !zero.failed {
			t.Fatal("unfunded pinned response parser ran", n)
		}
	}
}

func TestProviderReplayABIWordAllowanceCannotBeBorrowedFromJSON(t *testing.T) {
	raw := providerPinResultFixture(t)[3]
	plain, present, err := rpcABI(raw, false, 128)
	meter := providerReplayUnitMeter()
	observed, observedPresent, observedErr := rpcABIObserved(raw, false, meter, 128)
	if err != nil || observedErr != nil || present || observedPresent || !bytes.Equal(plain[0], observed[0]) || meter.actual[2] <= uint64(len(raw)) {
		t.Fatal("observed ABI changed bytes or omitted nested decoders", err, observedErr)
	}
	short := providerReplayUnitMeter()
	// The ordinary fixture's JSON string has no escapes: decoded hex input
	// has len(raw)-4 bytes. The next32-byte ABI word is one byte underfunded.
	beforeWord := uint64(2*len(raw) - 4)
	short.upper[2] = beforeWord + 31
	if values, _, err := rpcABIObserved(raw, false, short, 128); err == nil || values != nil || !short.failed || short.actual[2] != beforeWord {
		t.Fatal("ABI word reused JSON/hex decode allowance")
	}
}

func TestProviderReplayObservedPinSettlementCountsOriginalPolicy(t *testing.T) {
	_, ticket, _, _ := providerMeterFixture(t)
	raw := providerPolicyTestBytes(t, ledgerTestSettlementObject(ticket, "complete", 1))
	s := &providerCustodyStore{}
	meter := providerReplayUnitMeter()
	got, err := s.parsePinSettlementObserved(ticket, raw, meter)
	if err != nil || got == nil || meter.actual[2] != uint64(len(raw)+len(ticket.policy)) {
		t.Fatal("settlement and policy must be separate actual parses", err)
	}
	short := providerReplayUnitMeter()
	short.upper[2] = uint64(len(raw))
	if got, err := s.parsePinSettlementObserved(ticket, raw, short); err == nil || got != nil || !short.failed || short.actual[2] != uint64(len(raw)) {
		t.Fatal("unfunded policy parse returned a settlement")
	}
}

func TestProviderReplayPinClosureReadsOrdinaryOwnedChildOriginals(t *testing.T) {
	c, _, input, digests := providerOwnedPinChildrenUnit(t)
	view := providerReadDataView(t, c.store)
	for key, ref := range c.store.objects {
		view.objects[key] = ref
	}
	for key, binding := range c.store.bindings {
		view.bindings[key] = binding
	}
	replay := *c.store.replay
	meter := providerReplayUnitMeter()
	replay.meter = meter
	view.activeMeter.Store(meter)
	defer view.activeMeter.Store(nil)
	// The existing fixture retains only normal owned TLS responses. This
	// subsequent parser unit is not the complete catalog/independent-map
	// inspector and cannot select an actual principal, ACK or dispatch.
	agent, selected, err := replay.pinInput(view, input)
	if err != nil || agent != "finite-owned-pin" || selected != c.selected.Hash {
		t.Fatal("observed pin input changed retained binding", err)
	}
	got, err := replay.pinResults(view, digests, selected)
	if err != nil {
		t.Fatal("observed child-original closure", err)
	}
	want := providerPinResultFixture(t)
	for n := range got {
		if !bytes.Equal(got[n], want[n]) || !providerCheckPinPartObserved(n, got[n], meter) {
			t.Fatal("normal child response or repeated parser changed", n)
		}
	}
	if meter.failed || meter.actual[0] != 0 || meter.actual[1] != 0 || meter.actual[3] != 0 {
		t.Fatal("retained parser invented new transport or block work")
	}
}
