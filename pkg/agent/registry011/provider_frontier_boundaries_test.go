package registry011

import "testing"

func TestProviderFrontierInvalidRecordCannotEncodeANewHead(t *testing.T) {
	if raw, err := (providerFrontierRecord{}).bytes(); err == nil || raw != nil {
		t.Fatal("zero record acquired a serialized frontier")
	}
	if _, err := parseProviderFrontierRecord(nil, ProviderCandidateIdentity{}); err == nil {
		t.Fatal("absent original became a record")
	}
}

func TestProviderFrontierRetentionChecksValidButInsufficientResourceBounds(t *testing.T) {
	id, inventory, fresh, intent, commit, frontier, bounds := providerRetentionFixture(t)
	for _, mode := range []string{"resident", "metadata", "payload", "physical"} {
		t.Run(mode, func(t *testing.T) {
			b, private := bounds, *bounds.continuation
			b.continuation = &private
			switch mode {
			case "resident":
				private.maximumResidentBytes = 1
			case "metadata":
				private.maximumMetadataBytes, private.pageMetadataBytes = 1, 1
				b.ledger.MaximumMetadataBytes = 1
			case "payload":
				private.maximumPayloadBytes, private.pagePayloadBytes = 1, 1
				b.ledger.MaximumPayloadBytes = 1
			case "physical":
				b.maximumStoredBytes, b.maximumObjectBytes = 4096, 4096
				b.setupLive, b.setupRetained = 4096, 4096
			}
			if !b.valid() {
				t.Fatal("resource refusal must use structurally valid inert bounds")
			}
			if _, err := providerFrontierRetentionCost(id, inventory, fresh, intent, commit, frontier, b); err == nil {
				t.Fatal("insufficient complete retention was admitted")
			}
		})
	}
}

// Only normal original files in this owned unit fixture are written. The
// reconstructed main/setup/high-water values are inert accounting data: no
// selected principal, actual HEAD replacement, work ACK or transport exists.
func TestProviderFrontierInterleavedSetupChargesOldCopiesOnceAndOnlyNewAllocation(t *testing.T) {
	u := providerFrontierUnitFixture(t)
	s, config := u.s, u.s.config
	h0 := providerFrontierHighwater{identity: config.identity, fence: 1, profile: config.profile, commit: config.anchor, setup: s.setupTip}
	raw0, err := h0.bytes(config)
	if err != nil || s.custody.write(providerCustodyName("highwater", 0), raw0) != nil {
		t.Fatal("normal initial original", err)
	}
	h1 := h0
	h1.ordinal, h1.sequence, h1.previous = 1, 1, providerCandidateHash("sage-provider-custody-highwater-v1", raw0)
	h1.commit, h1.frontier = s.tip, providerCandidateHash("sage-provider-custody-frontier-v1", u.records[0].frontier)
	raw1, err := h1.bytes(config)
	if err != nil || s.custody.write(providerCustodyName("highwater", 1), raw1) != nil {
		t.Fatal("normal first main original", err)
	}
	prior := *u.r // Inert replay data; no store/mutex/clock owner is copied.
	sample := u.clock
	sample.WallMS++
	sample.MonotonicMS++
	setupConfig := config
	setupConfig.attempt, setupConfig.expectedTip, setupConfig.expectedSetupTip = 1, s.tip, s.setupTip
	claim := providerCustodySetup(setupConfig, sample)
	completed := providerSetupCommit(claim)
	setup := providerSetupReference{commit: providerCandidateHash("sage-provider-authority-setup-commit-v1", completed), claim: providerCandidateHash("sage-provider-authority-setup-v1", claim), expectedTip: s.tip, ordinal: 1, clock: providerClockDiagnostic(sample)}
	if s.custody.write(providerCustodyName("setup-intent", 1), claim) != nil || s.evidence.write(providerCustodyName("setup-claim", 1), claim) != nil || s.custody.write(providerCustodyName("setup-commit", 1), completed) != nil {
		t.Fatal("normal bounded setup originals")
	}
	h2 := h1
	h2.ordinal, h2.previous, h2.setup = 2, providerCandidateHash("sage-provider-custody-highwater-v1", raw1), setup.commit
	raw2, err := h2.bytes(config)
	if err != nil || !checkProviderHighwaterSetup(h1, h2, config, claim, completed) || s.custody.write(providerCustodyName("highwater", 2), raw2) != nil {
		t.Fatal("normal setup relation", err)
	}
	s.setups, s.setupTip = append(s.setups, setup), setup.commit
	u.firstTicket(t, 2)
	record := u.records[1]
	h3 := h2
	h3.ordinal, h3.sequence, h3.previous = 3, 2, providerCandidateHash("sage-provider-custody-highwater-v1", raw2)
	h3.commit, h3.frontier = s.tip, providerCandidateHash("sage-provider-custody-frontier-v1", record.frontier)
	raw3, err := h3.bytes(config)
	view := &providerCustodyStore{config: config, custody: s.custody, evidence: s.evidence, sequence: h1.sequence, tip: h1.commit, setups: s.setups, setupTip: setup.commit}
	cost, costErr := prior.highwaterRetention(view, h3, setup, record.intent, record.commit, record.frontier)
	if err != nil || costErr != nil || cost.metadata != uint64(2*len(raw2)+2*len(raw3)) || cost.stored != uint64(2*len(raw3))+2*config.bounds.fileOverheadBytes || prior.setupOrdinal != 0 || prior.current.setup != h1.setup || u.r.body.intent.clock != providerClockDiagnostic(u.clock) {
		t.Fatal("setup bytes were lost, doubled, physically reallocated or renewed I", err, costErr, cost)
	}
	// Repeating a data quote is not a second debit, pin adoption or permission.
	again, err := prior.highwaterRetention(view, h3, setup, record.intent, record.commit, record.frontier)
	if err != nil || again != cost {
		t.Fatal("read-only quote changed retained accounting", err)
	}
	// Only independent DATA references change; every original file stays intact.
	wrongSetup := setup
	wrongSetup.commit = admissionTestDigest("f")
	if _, err := prior.highwaterRetention(view, h3, wrongSetup, record.intent, record.commit, record.frontier); err == nil {
		t.Fatal("quote used another selected setup head")
	}
	saved := view.setups[1]
	view.setups[1].commit = admissionTestDigest("f")
	if _, err := prior.highwaterRetention(view, h3, setup, record.intent, record.commit, record.frontier); err == nil {
		t.Fatal("cached setup replaced its unchanged high-water original")
	}
	view.setups[1] = saved
}

func TestProviderFrontierHighwaterRetentionRejectsAbsentAndCrossedState(t *testing.T) {
	u, s, heads := providerHighwaterRetentionFixture(t)
	r, err := newProviderFrontierReplay(s.config)
	if err != nil {
		t.Fatal(err)
	}
	record := u.records[0]
	for _, mode := range []string{"nil-replay", "nil-store", "negative-setup", "missing-setup", "setup-backwards", "wrong-sequence", "negative-prior-with-main", "wrong-tip", "wrong-setup"} {
		t.Run(mode, func(t *testing.T) {
			copy := *r
			selected, view, setup := &copy, s, s.setups[0]
			savedTip, savedSetup := s.tip, s.setups[0]
			defer func() { s.tip, s.setups[0] = savedTip, savedSetup }()
			switch mode {
			case "nil-replay":
				selected = nil
			case "nil-store":
				view = nil
			case "negative-setup":
				setup.ordinal = -1
			case "missing-setup":
				setup.ordinal = len(s.setups)
			case "setup-backwards":
				copy.setupOrdinal = 1
			case "wrong-sequence":
				copy.current.sequence = 1
			case "negative-prior-with-main":
				copy.current.sequence, s.sequence = 1, 1
				defer func() { s.sequence = 0 }()
			case "wrong-tip":
				s.tip = admissionTestDigest("f")
			case "wrong-setup":
				s.setups[0].commit = admissionTestDigest("f")
			}
			if _, err := selected.highwaterRetention(view, heads[1], setup, record.intent, record.commit, record.frontier); err == nil {
				t.Fatal("crossed or absent predecessor acquired a retention quote")
			}
		})
	}
	if _, _, err := s.originalHighwater(uint64(s.config.bounds.maximumCommits) + uint64(s.config.bounds.maximumSetupRecoveries) + 1); err == nil {
		t.Fatal("out-of-bound ordinal was read")
	}
	if _, _, err := s.originalHighwater(2); err == nil {
		t.Fatal("missing original was inferred from another head")
	}
}
