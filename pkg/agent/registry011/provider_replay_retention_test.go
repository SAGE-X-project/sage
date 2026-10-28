package registry011

import (
	"bytes"
	"testing"
)

func TestProviderReplaySetupAndProfileObserveExactOrdinaryData(t *testing.T) {
	u := providerFrontierUnitFixture(t)
	config := u.s.config
	clock := providerClockDiagnostic(u.clock)
	want := providerCustodySetupDiagnostic(config, clock)
	meter := providerReplayUnitMeter()
	claim := providerCustodySetupDiagnosticObserved(config, clock, meter)
	if !bytes.Equal(claim, want) || meter.actual[2] != 0 || meter.actual[10] == 0 {
		t.Fatal("observed setup changed bytes or counted a nonexistent decode")
	}
	before := meter.actual[2]
	commit := providerSetupCommitObserved(claim, meter)
	if !bytes.Equal(commit, providerSetupCommit(want)) || meter.actual[2]-before != uint64(len(claim)) || providerIntegrationProfileObserved(config.bounds, config.capture, meter) != config.profile {
		t.Fatal("setup commit/profile omitted repeated actual work")
	}
	for _, subject := range []string{"setup", "commit", "profile", "join"} {
		zero := providerReplayUnitMeter()
		zero.upper = candidateWorkVector{}
		switch subject {
		case "setup":
			if providerCustodySetupDiagnosticObserved(config, clock, zero) != nil {
				t.Fatal(subject)
			}
		case "commit":
			if providerSetupCommitObserved(claim, zero) != nil {
				t.Fatal(subject)
			}
		case "profile":
			if providerIntegrationProfileObserved(config.bounds, config.capture, zero) != "" {
				t.Fatal(subject)
			}
		case "join":
			if providerReplayJoin([]string{"ordinary", "data"}, zero) != "" {
				t.Fatal(subject)
			}
		}
		if !zero.failed || zero.actual != (candidateWorkVector{}) {
			t.Fatal("unfunded primitive ran", subject)
		}
	}
}

func TestProviderReplayHighwaterAndRetentionPreserveExactClosure(t *testing.T) {
	u, s, heads := providerHighwaterRetentionFixture(t)
	meter := providerReplayUnitMeter()
	raw, err := heads[0].bytesObserved(s.config, meter)
	plain, plainErr := heads[0].bytes(s.config)
	if err != nil || plainErr != nil || !bytes.Equal(raw, plain) {
		t.Fatal("high-water encoder changed bytes", err, plainErr)
	}
	before := meter.actual[2]
	parsed, err := parseProviderFrontierHighwaterObserved(raw, s.config, meter)
	if err != nil || parsed != heads[0] || meter.actual[2]-before != uint64(len(raw)) {
		t.Fatal("high-water repeated parse omitted actual input", err)
	}
	record := u.records[0]
	if !checkProviderHighwaterMainObserved(heads[0], heads[1], s.config, record.intent, record.commit, record.frontier, meter) {
		t.Fatal("observed high-water main relation changed")
	}
	want, err := providerFrontierRetentionCost(s.config.identity, record.inventory, record.fresh, record.intent, record.commit, record.frontier, s.config.bounds)
	got, observedErr := providerFrontierRetentionCostObserved(s.config.identity, record.inventory, record.fresh, record.intent, record.commit, record.frontier, s.config.bounds, meter)
	if err != nil || observedErr != nil || got != want {
		t.Fatal("observed exact retention changed accounting", err, observedErr)
	}
	for _, subject := range []string{"parse", "relation", "retention"} {
		zero := providerReplayUnitMeter()
		zero.upper = candidateWorkVector{}
		switch subject {
		case "parse":
			if _, err := parseProviderFrontierHighwaterObserved(raw, s.config, zero); err == nil {
				t.Fatal(subject)
			}
		case "relation":
			if checkProviderHighwaterMainObserved(heads[0], heads[1], s.config, record.intent, record.commit, record.frontier, zero) {
				t.Fatal(subject)
			}
		case "retention":
			if _, err := providerFrontierRetentionCostObserved(s.config.identity, record.inventory, record.fresh, record.intent, record.commit, record.frontier, s.config.bounds, zero); err == nil {
				t.Fatal(subject)
			}
		}
		if !zero.failed {
			t.Fatal("exhaustion was hidden", subject)
		}
	}
}

func TestProviderReplayObservedRetainedOriginalsReachFirstTicket(t *testing.T) {
	u, s, heads := providerHighwaterRetentionFixture(t)
	meter := providerReplayUnitMeter()
	r, err := newProviderFrontierReplayObserved(s.config, meter)
	if err != nil {
		t.Fatal(err)
	}
	// Only this finite independent DATA replay is the unit subject. No
	// selected principal, scan command chain, live clock or ACK is created.
	r.meter, r.body.meter, r.indexes.meter = meter, meter, meter
	s.activeMeter.Store(meter)
	defer s.activeMeter.Store(nil)
	for n, record := range u.records {
		for _, object := range record.fresh {
			s.objects[object.kind+":"+object.digest] = providerIntegrationObjectRef{object.kind, object.digest, uint64(len(object.raw))}
		}
		if _, err := r.applyWithHighwater(s, record.inventory, record.fresh, record.intent, record.commit, record.frontier, heads[n+1]); err != nil || meter.failed {
			t.Fatal("ordinary retained originals could not reconstruct", n, err)
		}
		s.sequence, s.tip, s.enrollment, s.objectsRoot = r.current.sequence, r.current.commit, r.current.enrollment, r.current.objects
	}
	if r.body.ledger == nil || r.body.ledger.meter != meter || r.indexes.meter != meter || r.current != u.r.current || meter.actual[2] == 0 {
		t.Fatal("independent model changed data or lost its one observer")
	}
}
