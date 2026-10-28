package registry011

import "testing"

func TestProviderFrontierRetentionRequiresCanonicalOriginalBytes(t *testing.T) {
	id, inventory, fresh, intent, commit, frontier, bounds := providerRetentionFixture(t)
	if !checkProviderFrontierRetention(id, inventory, intent, commit, frontier) || checkProviderFrontierRetention(id, nil, intent, commit, frontier) {
		t.Fatal("exact original closure did not distinguish an absent inventory")
	}
	// Whitespace is finite inert parser data, never written as an operating head.
	if _, err := providerFrontierRetentionCost(id, inventory, fresh, intent, commit, append(append([]byte(nil), frontier...), '\n'), bounds); err == nil {
		t.Fatal("noncanonical frontier acquired a retention quote")
	}
	// contentJSON itself rejects noncanonical input. In particular, a commit
	// cannot be derived from it; do not describe a nil derivation as a matching
	// commit. Test the closed predicate and complete quote with unchanged roots.
	changed := append(append([]byte(nil), intent...), '\n')
	if providerFrontierCommit(changed) != nil || checkProviderFrontierRetention(id, inventory, changed, commit, frontier) {
		t.Fatal("noncanonical intent acquired a matching reference closure")
	}
	if _, err := providerFrontierRetentionCost(id, inventory, fresh, changed, commit, frontier, bounds); err == nil {
		t.Fatal("noncanonical intent acquired a retention quote")
	}
}

func TestProviderFrontierHighwaterRequiresOriginalPriorState(t *testing.T) {
	for _, mode := range []string{"prior-tip", "prior-frontier", "missing-next", "ordinal-cap", "ordinal-overflow"} {
		t.Run(mode, func(t *testing.T) {
			u, s, heads := providerHighwaterRetentionFixture(t)
			r, err := newProviderFrontierReplay(s.config)
			if err != nil {
				t.Fatal(err)
			}
			first := u.records[0]
			if mode == "ordinal-overflow" {
				// Finite impossible DATA state must refuse before an original read.
				s.sequence, r.current.sequence, r.setupOrdinal = ^uint64(0), ^uint64(0), 1
				s.setups = append(s.setups, s.setups[0])
				setup := s.setups[1]
				setup.ordinal = 1
				if _, err := r.highwaterRetention(s, heads[1], setup, first.intent, first.commit, first.frontier); err == nil {
					t.Fatal("overflow wrapped an original high-water ordinal")
				}
				return
			}
			if mode == "ordinal-cap" {
				if _, _, err := s.originalHighwater(uint64(s.config.bounds.maximumCommits+s.config.bounds.maximumSetupRecoveries) + 1); err == nil {
					t.Fatal("unbounded high-water original read")
				}
				return
			}
			if mode == "prior-tip" {
				s.tip = admissionTestDigest("f")
				if _, err := r.highwaterRetention(s, heads[1], s.setups[0], first.intent, first.commit, first.frontier); err == nil {
					t.Fatal("prior immutable head did not bind current commit")
				}
				return
			}
			for _, object := range first.fresh {
				s.objects[object.kind+":"+object.digest] = providerIntegrationObjectRef{object.kind, object.digest, uint64(len(object.raw))}
			}
			if _, err := r.applyWithHighwater(s, first.inventory, first.fresh, first.intent, first.commit, first.frontier, heads[1]); err != nil {
				t.Fatal(err)
			}
			s.sequence, s.tip, s.enrollment, s.objectsRoot = r.current.sequence, r.current.commit, r.current.enrollment, r.current.objects
			if mode == "prior-frontier" {
				r.current.fence++ // Independent DATA view disagrees with unchanged H1.
			} else {
				s.sequence, r.current.sequence = 2, 2 // No H2 original exists in fixture.
			}
			next := u.records[1]
			if _, err := r.highwaterRetention(s, heads[2], s.setups[0], next.intent, next.commit, next.frontier); err == nil {
				t.Fatal("missing or mismatched prior original became a quote")
			}
		})
	}
}

func TestProviderFrontierOldMaterialReferencesCannotUseCachedDataAlone(t *testing.T) {
	for _, mode := range []string{"index", "anchor", "no-anchor", "invalid-ledger", "no-intent", "admission", "budget", "node", "missing-ticket", "missing-ticket-state", "ticket", "unknown-consumption", "unknown-settlement", "unknown-kind"} {
		t.Run(mode, func(t *testing.T) {
			u := providerFrontierUnitFixture(t)
			u.firstTicket(t, 1)
			r, s, kind, operation := u.r, u.s, "admit", u.ledger.lastTicket.digest
			if r.checkOldMaterials(s, kind, operation) != nil {
				t.Fatal("normal retained originals do not match their DATA view")
			}
			switch mode {
			case "index":
				delete(s.objects, "index-mutation:"+r.indexes.heads[0])
				delete(s.objects, "index-empty:"+r.indexes.heads[0])
			case "anchor":
				delete(s.objects, "enrollment:"+r.body.anchor.digest)
			case "no-anchor":
				r.body.anchor = nil
			case "invalid-ledger":
				r.body.ledger.owner = nil
			case "no-intent":
				r.body.intent = nil
			case "admission":
				delete(s.objects, "admission:"+r.body.intent.digest)
			case "budget":
				delete(s.objects, "budget:"+r.body.ledger.current.digest)
			case "node":
				delete(s.objects, "ledger-node:"+r.body.ledger.current.heads[0])
			case "missing-ticket":
				r.body.ledger.lastTicket = nil
			case "missing-ticket-state":
				delete(r.body.ledger.tickets, r.body.ledger.lastTicket.digest)
			case "ticket":
				delete(s.objects, "ticket:"+r.body.ledger.lastTicket.digest)
			case "unknown-consumption":
				kind, operation = "consume", admissionTestDigest("f")
			case "unknown-settlement":
				kind, operation = "settle", admissionTestDigest("f")
			case "unknown-kind":
				// Old-material lookup is not transition authorization. Its finite
				// default does no additional operation lookup; typed apply rejects it.
				if r.checkOldMaterials(s, "unknown", admissionTestDigest("f")) != nil {
					t.Fatal("unrelated DATA lookup unexpectedly required a ticket")
				}
				return
			}
			if r.checkOldMaterials(s, kind, operation) == nil {
				t.Fatal("cached state replaced a required retained original", mode)
			}
		})
	}
}

func TestProviderFrontierHighwaterQuoteRejectsInsufficientInertMetadataProfile(t *testing.T) {
	u := providerFrontierUnitFixture(t)
	config, private := u.s.config, *u.s.config.bounds.continuation
	private.maximumMetadataBytes, private.pageMetadataBytes = 1, 1
	config.bounds.continuation, config.bounds.ledger.MaximumMetadataBytes = &private, 1
	config.profile = providerIntegrationProfile(config.bounds, config.capture)
	if !config.bounds.valid() {
		t.Fatal("finite insufficient profile must be structurally valid")
	}
	// These are only high-water quote inputs, not an admitted prefix or a
	// selected profile. No constructor/ACK/dispatch accepts this DATA view.
	s := &providerCustodyStore{config: config, custody: u.s.custody, evidence: u.s.evidence, tip: config.anchor, setups: u.s.setups}
	r := &providerFrontierReplay{setupOrdinal: -1}
	h0 := providerFrontierHighwater{identity: config.identity, fence: 1, profile: config.profile, commit: config.anchor, setup: s.setups[0].commit}
	raw, err := h0.bytes(config)
	if err != nil || s.custody.write(providerCustodyName("highwater", 0), raw) != nil {
		t.Fatal("normal immutable quote input", err)
	}
	record := u.records[0]
	f, err := parseProviderFrontierRecord(record.frontier, config.identity)
	if err != nil {
		t.Fatal(err)
	}
	h1 := h0
	h1.ordinal, h1.sequence, h1.previous = 1, 1, providerCandidateHash("sage-provider-custody-highwater-v1", raw)
	h1.commit, h1.frontier = f.commit, providerCandidateHash("sage-provider-custody-frontier-v1", record.frontier)
	if !checkProviderHighwaterMain(h0, h1, config, record.intent, record.commit, record.frontier) {
		t.Fatal("finite data must reach the common metadata-cap check")
	}
	if _, err := r.highwaterRetention(s, h1, s.setups[0], record.intent, record.commit, record.frontier); err == nil {
		t.Fatal("both original head copies escaped the selected metadata cap")
	}
}
