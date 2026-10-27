package registry011

import "testing"

// These normal owned original files plus reconstructed records are DATA-only
// fixtures: no HEAD CAS, selected principal, operating ACK or dispatch occurs.
func providerHighwaterRetentionFixture(t *testing.T) (*providerFrontierUnit, *providerCustodyStore, []providerFrontierHighwater) {
	t.Helper()
	u := providerFrontierUnitFixture(t)
	u.firstTicket(t, 1)
	config := u.s.config
	h := providerFrontierHighwater{identity: config.identity, fence: 1, profile: config.profile, commit: config.anchor, setup: u.s.setups[0].commit}
	heads := []providerFrontierHighwater{h}
	for i := range u.records {
		raw, err := h.bytes(config)
		if err != nil || u.s.custody.write(providerCustodyName("highwater", h.ordinal), raw) != nil {
			t.Fatal("normal immutable high-water original", err)
		}
		f, err := parseProviderFrontierRecord(u.records[i].frontier, config.identity)
		if err != nil {
			t.Fatal(err)
		}
		h.ordinal, h.sequence, h.previous = h.ordinal+1, f.sequence, providerCandidateHash("sage-provider-custody-highwater-v1", raw)
		h.commit, h.frontier = f.commit, providerCandidateHash("sage-provider-custody-frontier-v1", u.records[i].frontier)
		heads = append(heads, h)
	}
	empty, err := newProviderFrontierInventory("", nil, config.bounds.maximumEntries, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	s := &providerCustodyStore{config: config, custody: u.s.custody, evidence: u.s.evidence, tip: config.anchor, objectsRoot: empty.digest, setupOrigin: u.clock, setupTip: u.s.setupTip, setups: u.s.setups, objects: make(map[string]providerIntegrationObjectRef), operations: make(map[string]bool), bindings: make(map[string]string)}
	s.owner = s
	return u, s, heads
}

func TestProviderFrontierHighwaterSharesOneTypedRetentionDebit(t *testing.T) {
	u, s, heads := providerHighwaterRetentionFixture(t)
	r, err := newProviderFrontierReplay(s.config)
	if err != nil {
		t.Fatal(err)
	}
	var metadata uint64
	for i, record := range u.records {
		for _, object := range record.fresh {
			s.objects[object.kind+":"+object.digest] = providerIntegrationObjectRef{object.kind, object.digest, uint64(len(object.raw))}
		}
		base, err := providerFrontierRetentionCost(s.config.identity, record.inventory, record.fresh, record.intent, record.commit, record.frontier, s.config.bounds)
		if err != nil {
			t.Fatal(err)
		}
		hRaw, err := heads[i+1].bytes(s.config)
		if err != nil {
			t.Fatal(err)
		}
		extra := uint64(2 * len(hRaw))
		if i == 0 {
			initial, _ := heads[0].bytes(s.config)
			extra += uint64(2 * len(initial))
			setupBytes, ok := r.setupBytes(s, 0)
			if !ok {
				t.Fatal("initial original setup bytes")
			}
			extra += setupBytes
		}
		cost, err := r.applyWithHighwater(s, record.inventory, record.fresh, record.intent, record.commit, record.frontier, heads[i+1])
		if err != nil || !r.highwater || cost.metadata != base.metadata+extra || cost.payload != base.payload || cost.stored != base.stored+uint64(2*len(hRaw))+2*s.config.bounds.fileOverheadBytes {
			t.Fatal("high-water was omitted, separately debited or doubled", i, err, cost, base, extra)
		}
		metadata += cost.metadata
		if i == 0 && r.preI.metadata != cost.metadata {
			t.Fatal("actual H0/E0 bytes were not retained for first I")
		}
		s.sequence, s.tip, s.enrollment, s.objectsRoot = r.current.sequence, r.current.commit, r.current.enrollment, r.current.objects
	}
	if r.body.ledger == nil || r.body.ledger.current.privateMetadata < metadata || r.body.ledger.current.privatePayload == 0 {
		t.Fatal("common ledger lost prior high-water or request bytes")
	}
}

func TestProviderFrontierHighwaterCannotBeDroppedFromTypedReplay(t *testing.T) {
	u, s, heads := providerHighwaterRetentionFixture(t)
	r, err := newProviderFrontierReplay(s.config)
	if err != nil {
		t.Fatal(err)
	}
	first := u.records[0]
	for _, object := range first.fresh {
		s.objects[object.kind+":"+object.digest] = providerIntegrationObjectRef{object.kind, object.digest, uint64(len(object.raw))}
	}
	if _, err := r.applyWithHighwater(s, first.inventory, first.fresh, first.intent, first.commit, first.frontier, heads[1]); err != nil {
		t.Fatal(err)
	}
	s.sequence, s.tip, s.enrollment, s.objectsRoot = r.current.sequence, r.current.commit, r.current.enrollment, r.current.objects
	next := u.records[1]
	if _, err := r.apply(s, next.inventory, next.fresh, next.intent, next.commit, next.frontier); err == nil || !r.failed {
		t.Fatal("physical high-water costs disappeared from a live prefix")
	}
}

func TestProviderFrontierHighwaterRejectsInertMismatchedSuccessor(t *testing.T) {
	u, s, heads := providerHighwaterRetentionFixture(t)
	r, err := newProviderFrontierReplay(s.config)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ordinal", "setup", "previous", "commit"} {
		t.Run(name, func(t *testing.T) {
			after := heads[1]
			switch name {
			case "ordinal":
				after.ordinal++
			case "setup":
				after.setup = admissionTestDigest("f")
			case "previous":
				after.previous = admissionTestDigest("f")
			case "commit":
				after.commit = admissionTestDigest("f")
			}
			first := u.records[0]
			if _, err := r.highwaterRetention(s, after, s.setups[0], first.intent, first.commit, first.frontier); err == nil {
				t.Fatal("invalid successor acquired a data retention quote")
			}
		})
	}
}
