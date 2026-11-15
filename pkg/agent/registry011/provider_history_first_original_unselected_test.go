package registry011

import "testing"

// Only finite policy/identity comparison DATA is supplied. This unopened
// private store has no owner, roots, clock, retained originals or live meter.
// A valid parsed policy never grants the original first-P reading role.
func TestProviderHistoryFirstOriginalSelectionRejectsPolicyDataWithoutOriginalAuthority(t *testing.T) {
	for _, mode := range []string{"invalid-policy", "different-policy-digest", "unselected-original"} {
		t.Run(mode, func(t *testing.T) {
			policy, err := ParseProviderPolicyCandidate(providerPolicyTestBytes(t, providerPolicyTestObject()), 64<<10)
			if err != nil {
				t.Fatal("ordinary finite parsed policy DATA prerequisite", err)
			}
			switch mode {
			case "invalid-policy":
				policy.raw = "not-policy-json"
			case "different-policy-digest":
				policy.policy = admissionTestDigest("f")
			}
			m := providerReplayUnitMeter()
			store := &providerCustodyStore{config: providerCustodyConfig{policy: policy}}
			group := &providerHistoryGroupOwner{store: store, meter: m, start: &providerHistoryProgramStart{}}
			input := &providerHistoryFirstPOriginals{group: group}
			if input.active() || group.owner != nil || group.parent != nil || store.owner != nil || store.activeMeter.Load() != nil {
				t.Fatal("private comparison DATA must not have an original execution role")
			}
			before, upper, selectedPolicy := m.actual, m.upper, *policy
			header, err := input.selected(providerFrontierRecord{})
			if err == nil || header != (FinalizedHeader{}) || m.failed || m.closed || m.actual[10] <= before[10] || m.upper != upper ||
				input.active() || group.owner != nil || group.parent != nil || group.firstPReading || group.start.owner != nil || store.owner != nil ||
				store.clock != nil || store.custody != nil || store.evidence != nil || store.history != nil || store.activeMeter.Load() != nil ||
				store.stored != 0 || len(input.objects) != 0 || input.raw != 0 || *policy != selectedPolicy {
				t.Fatal("invalid or unselected policy DATA supplied a finalized header or acquired original authority", mode, err)
			}
		})
	}
}

func TestProviderHistoryFirstExchangeCannotBorrowUnselectedOriginalData(t *testing.T) {
	m := providerReplayUnitMeter()
	store := &providerCustodyStore{}
	group := &providerHistoryGroupOwner{store: store, meter: m, start: &providerHistoryProgramStart{}}
	input := &providerHistoryFirstPOriginals{group: group}
	// Even hash-shaped requested identity cannot make object() consult a
	// producer cache or treat missing original bytes as an exchange envelope.
	group.start.exchange = admissionTestDigest("a")
	if input.active() {
		t.Fatal("unselected original closure must remain inactive")
	}
	before, upper := m.actual, m.upper
	header, err := input.exchange(nil, nil, nil, nil)
	if err == nil || header != (FinalizedHeader{}) || m.failed || m.closed || m.actual[10] <= before[10] || m.upper != upper ||
		input.active() || group.owner != nil || group.parent != nil || group.firstPReading || group.start.owner != nil ||
		store.owner != nil || store.clock != nil || store.custody != nil || store.evidence != nil || store.activeMeter.Load() != nil ||
		len(input.objects) != 0 || input.raw != 0 {
		t.Fatal("unselected exchange DATA returned a header, touched original input or gained a reading role", err)
	}
}
