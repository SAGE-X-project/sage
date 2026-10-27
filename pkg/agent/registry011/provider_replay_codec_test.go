package registry011

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func providerReplayUnitMeter() *providerWorkMeter {
	// A private arithmetic subject only. It cannot enter a store, restore an
	// inspection or pass the separately required live scanner-owner checks.
	m := &providerWorkMeter{integration: &providerIntegration{}, upper: candidateWorkVector{0, 0, 1 << 30, 0, 0, 0, 0, 0, 0, 0, 1 << 60}}
	m.owner = m
	return m
}

func TestProviderReplayPrimitiveObservationPreservesExactBytesAndDomain(t *testing.T) {
	raw := []byte(`{"value":"ordinary data"}`)
	m := providerReplayUnitMeter()
	if providerReplayJSON(raw, m) == nil || m.actual[2] != uint64(len(raw)) {
		t.Fatal("decoder invocation was not observed")
	}
	before := m.actual[10]
	if providerReplayDigest("test-domain", raw, m) != providerCandidateHash("test-domain", raw) || m.actual[10]-before != uint64(len("test-domain")+1+len(raw)+1) {
		t.Fatal("domain NUL or actual hash invocation omitted")
	}
	sum := sha256.Sum256(raw)
	before = m.actual[10]
	if providerReplayDigest("", raw, m) != "0x"+hex.EncodeToString(sum[:]) || m.actual[10]-before != uint64(len(raw)+1) {
		t.Fatal("ordinary content hash acquired a domain prefix")
	}
	before = m.actual[10]
	if providerReplayString(raw, m) != string(raw) || m.actual[10]-before != uint64(len(raw)+1) {
		t.Fatal("retained string copy omitted")
	}
	encoded := providerReplayMarshal(map[string]string{"value": "ordinary data"}, 4096, m)
	if !bytes.Equal(encoded, raw) || m.actual[2] != uint64(len(raw)) {
		t.Fatal("standard closed encoding changed bytes or counted decoding")
	}
	for _, mode := range []string{"decode", "domain-hash", "plain-hash", "copy", "encode"} {
		zero := providerReplayUnitMeter()
		zero.upper = candidateWorkVector{}
		switch mode {
		case "decode":
			if providerReplayJSON(raw, zero) != nil {
				t.Fatal(mode)
			}
		case "domain-hash":
			if providerReplayDigest("test-domain", raw, zero) != "" {
				t.Fatal(mode)
			}
		case "plain-hash":
			if providerReplayDigest("", raw, zero) != "" {
				t.Fatal(mode)
			}
		case "copy":
			if providerReplayString(raw, zero) != "" {
				t.Fatal(mode)
			}
		case "encode":
			if providerReplayMarshal(map[string]string{"value": "ordinary data"}, 4096, zero) != nil {
				t.Fatal(mode)
			}
		}
		if !zero.failed || zero.actual != (candidateWorkVector{}) {
			t.Fatal("unfunded primitive ran or left its meter reusable", mode)
		}
	}
	for _, limit := range []uint64{0, (64 << 10) + 1} {
		if providerReplayMarshal(nil, limit, nil) != nil {
			t.Fatal("unsupported finite encoding bound")
		}
	}
	if providerReplayMarshal(make(chan int), 4096, nil) != nil || providerReplayMarshal(map[string]string{"value": "ordinary data"}, 1, nil) != nil {
		t.Fatal("invalid or oversized encoding")
	}
}

func TestProviderReplayLimitsPreserveTheExistingPublicPolicyBound(t *testing.T) {
	policy, context, anchor, object := admissionTestIntentFixture(t, nil)
	limits := contentJSON([]byte(policy.raw)).(map[string]any)["limits"].(map[string]any)
	want := providerPolicyTestBytes(t, limits)
	if raw := providerReplayMarshal(limits, 64<<10, nil); !bytes.Equal(raw, want) {
		t.Fatal("the existing policy bound was confused with ledger-node size")
	}
	m := providerReplayUnitMeter()
	if raw := providerReplayMarshal(limits, 64<<10, m); !bytes.Equal(raw, want) || m.actual[2] != 0 || m.actual[10] != uint64(len(want)*len(want)+1) {
		t.Fatal("finite policy encoding upper replaced its actual byte metric")
	}
	raw := providerPolicyTestBytes(t, object)
	pure, err := ParseCandidateAdmissionIntent(policy, context, anchor, raw, 64<<10)
	if err != nil {
		t.Fatal("existing public intent parser rejected its valid policy", err)
	}
	observed, err := parseCandidateAdmissionIntentObserved(policy, context, anchor, raw, 64<<10, providerReplayUnitMeter())
	if err != nil || pure.digest != observed.digest || !bytes.Equal(pure.CanonicalBytes(), observed.CanonicalBytes()) {
		t.Fatal("observed replay changed the public intent contract", err)
	}
}

func TestProviderReplayTypedParsersObserveEveryRepeatedPolicyDecode(t *testing.T) {
	policy, context, anchor, intentObject := admissionTestIntentFixture(t, nil)
	intentRaw := providerPolicyTestBytes(t, intentObject)
	ticketPolicy, intent, ticketObject := admissionTestTicketFixture(t)
	ticketRaw := providerPolicyTestBytes(t, ticketObject)
	ticket := admissionTestTicket(t, ticketPolicy, intent, 0, ticketObject)
	consumptionTicket, consumptionObject := admissionTestConsumptionFixture(t)
	consumptionRaw := providerPolicyTestBytes(t, consumptionObject)
	settlementRaw := providerPolicyTestBytes(t, ledgerTestSettlementObject(ticket, "failed", 20))
	cases := []struct {
		name    string
		decoded uint64
		parse   func(*providerWorkMeter) (string, error)
	}{
		{"enrollment", uint64(len(anchor.raw)), func(m *providerWorkMeter) (string, error) {
			v, err := parseCandidateEnrollmentRootObserved(context, []byte(anchor.raw), 64<<10, m)
			if err != nil {
				return "", err
			}
			return v.digest, nil
		}},
		{"intent", uint64(len(intentRaw) + len(policy.raw)), func(m *providerWorkMeter) (string, error) {
			v, err := parseCandidateAdmissionIntentObserved(policy, context, anchor, intentRaw, 64<<10, m)
			if err != nil {
				return "", err
			}
			return v.digest, nil
		}},
		{"ticket", uint64(len(ticketRaw) + len(ticketPolicy.raw)), func(m *providerWorkMeter) (string, error) {
			v, err := parseCandidateAdmissionTicketObserved(ticketPolicy, intent, 0, ticketRaw, 64<<10, m)
			if err != nil {
				return "", err
			}
			return v.digest, nil
		}},
		{"consumption", uint64(len(consumptionRaw) + len(consumptionTicket.policy)), func(m *providerWorkMeter) (string, error) {
			v, err := parseCandidateConsumptionObserved(consumptionTicket, consumptionRaw, 64<<10, m)
			if err != nil {
				return "", err
			}
			return v.digest, nil
		}},
		{"settlement", uint64(len(settlementRaw) + len(ticket.policy)), func(m *providerWorkMeter) (string, error) {
			v, err := parseCandidateAdmissionSettlementObserved(ticket, settlementRaw, 64<<10, m)
			if err != nil {
				return "", err
			}
			return v.digest, nil
		}},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			pure, err := item.parse(nil)
			if err != nil || !lowerHash(pure) {
				t.Fatal("normal pure DATA fixture", err)
			}
			m := providerReplayUnitMeter()
			first, err := item.parse(m)
			if err != nil || first != pure || m.actual[2] != item.decoded {
				t.Fatal("observed parser omitted an actual input or changed bytes", err)
			}
			work := m.actual
			second, err := item.parse(m)
			if err != nil || second != pure {
				t.Fatal("normal repeated parse", err)
			}
			for index, n := range work {
				if m.actual[index] != 2*n {
					t.Fatal("repeated parse reused a diagnostic counter", index)
				}
			}
			short := providerReplayUnitMeter()
			short.upper[2] = item.decoded - 1
			if value, err := item.parse(short); err == nil || value != "" || !short.failed {
				t.Fatal("one decoded byte short returned a parsed record")
			}
		})
	}
}

func TestProviderReplayScanReadsObserveEachActualDecodeAndDomainSeparator(t *testing.T) {
	u := providerFrontierUnitFixture(t)
	s := u.s
	c := &providerScanControl{store: s}
	c.owner = c
	m := &providerWorkMeter{scan: c, upper: candidateWorkVector{0, 0, 1 << 30, 0, 0, 0, 0, 0, 0, 0, 1 << 60}}
	m.owner = m
	s.activeMeter.Store(m)
	defer s.activeMeter.Store(nil)
	raw, err := s.object("enrollment", u.root.digest)
	if err != nil || !bytes.Equal(raw, u.root.CanonicalBytes()) {
		t.Fatal("ordinary retained object DATA", err)
	}
	n := uint64(len(raw))
	// Object digest validation hashes only. The subsequent format check is
	// one actual decode; the former pre-observation must not count it again.
	wantUnits := n*n + 1 + (n + 1) + uint64(len("sage-provider-enrollment-v2")+1) + n + 1
	if m.actual[2] != n || m.actual[10] != wantUnits {
		t.Fatal("retained object reread/parse/hash invocation omitted", m.actual, wantUnits)
	}
	before := m.actual
	if _, err := s.object("enrollment", u.root.digest); err != nil {
		t.Fatal(err)
	}
	for index, count := range before {
		if m.actual[index] != 2*count {
			t.Fatal("second original read borrowed an earlier measurement")
		}
	}
}

func TestProviderReplayScanOwnerCannotBorrowOrRestoreAMeter(t *testing.T) {
	if _, err := newProviderFrontierScanReplay(nil, nil); err == nil {
		t.Fatal("missing claimed owner")
	}
	s := providerScanStoreFixture(t)
	if _, err := newProviderFrontierScanReplay(s.scanControl, providerReplayUnitMeter()); err == nil {
		t.Fatal("unclaimed scan used an unrelated arithmetic meter")
	}
	raw, a := providerScanInitialTargetFixture(t, s)
	if err := s.scanControl.claim(a); err != nil {
		t.Fatal(err)
	}
	m := &providerWorkMeter{scan: s.scanControl, upper: a.body.work, maximumClocks: a.body.clocks}
	m.owner = m
	if _, err := newProviderFrontierScanReplay(s.scanControl, m); err == nil {
		t.Fatal("scan used an inactive meter")
	}
	if s.frontierReplay.meter != nil || s.frontierReplay.body.meter != nil {
		t.Fatal("producer retained its expired setup meter")
	}
	if _, err := s.scanControl.inspectInitial(raw); err != nil {
		t.Fatal("normal one-use scan", err)
	}
	i := s.initialInspection
	if i.replay.meter != i.meter || i.replay.body.meter != i.meter {
		t.Fatal("independent scan did not retain its own invocation observer")
	}
	if _, err := i.replay.applyWithHighwater(s, nil, nil, nil, nil, nil, providerFrontierHighwater{}); err == nil {
		t.Fatal("inactive body meter was silently replaced")
	}
	if err := i.replay.body.apply(s, nil); err == nil {
		t.Fatal("inactive typed body silently ran without its observer")
	}
}
