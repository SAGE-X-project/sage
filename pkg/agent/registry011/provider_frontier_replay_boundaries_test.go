package registry011

import "testing"

func providerFrontierBeforeTicketData(t *testing.T) (*providerFrontierUnit, *providerCustodyStore, *providerFrontierReplay) {
	t.Helper()
	u := providerFrontierUnitFixture(t)
	u.firstTicket(t, 1)
	r, err := newProviderFrontierReplay(u.s.config)
	if err != nil {
		t.Fatal(err)
	}
	empty, err := newProviderFrontierInventory("", nil, u.s.config.bounds.maximumEntries, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	// Independent DATA view only. No live store owner, owned clock, ACK, CAS,
	// dispatch token or Source qualification is constructed from these maps.
	s := &providerCustodyStore{config: u.s.config, custody: u.s.custody, evidence: u.s.evidence, tip: u.s.config.anchor, objectsRoot: empty.digest, setupTip: u.s.setupTip, setups: u.s.setups, setupOrigin: u.s.setupOrigin, objects: make(map[string]providerIntegrationObjectRef), operations: make(map[string]bool), bindings: make(map[string]string)}
	for key, ref := range u.s.objects {
		s.objects[key] = ref
	}
	first := u.records[0]
	if _, err := r.apply(s, first.inventory, first.fresh, first.intent, first.commit, first.frontier); err != nil {
		t.Fatal("normal original E0 reconstruction", err)
	}
	s.sequence, s.tip, s.enrollment, s.objectsRoot = r.current.sequence, r.current.commit, r.current.enrollment, r.current.objects
	return u, s, r
}

func TestProviderFrontierReconstructionRejectsCrossedOrIncompleteData(t *testing.T) {
	for _, mode := range []string{"normal", "intent", "clock", "root", "prior-frontier", "commit", "frontier", "old-root", "ticket", "index", "invented-replay", "index-head", "cached-bytes"} {
		t.Run(mode, func(t *testing.T) {
			u, s, r := providerFrontierBeforeTicketData(t)
			record := u.records[1]
			inventory := *record.inventory
			before := r.current
			switch mode {
			case "intent":
				record.intent = []byte(`{}`)
			case "clock":
				value := contentJSON(record.intent).(map[string]any)
				value["clock"].(map[string]any)["epoch"] = admissionTestDigest("f")
				record.intent = providerIntegrationJSON(value)
			case "root":
				value := contentJSON(record.intent).(map[string]any)
				delete(s.objects, "enrollment:"+contentString(value["new_enrollment_root"]))
			case "prior-frontier":
				r.current.fence = 0
			case "commit":
				record.commit = []byte(`{}`)
			case "frontier":
				record.frontier = []byte(`{}`)
			case "old-root":
				delete(s.objects, "enrollment:"+r.body.anchor.digest)
			case "ticket":
				delete(s.objects, "ticket:"+u.ledger.lastTicket.digest)
			case "index":
				for _, object := range record.fresh {
					if object.kind == "index-mutation" {
						delete(s.objects, object.kind+":"+object.digest)
						break
					}
				}
			case "invented-replay", "index-head":
				f, err := parseProviderFrontierRecord(record.frontier, s.config.identity)
				if err != nil {
					t.Fatal(err)
				}
				if mode == "invented-replay" {
					f.replay, f.checkpoint = admissionTestDigest("e"), admissionTestDigest("f")
				} else {
					f.indexes[0] = admissionTestDigest("f")
				}
				record.frontier, err = f.bytes()
				if err != nil {
					t.Fatal("finite claimed record must reach typed comparison", err)
				}
			case "cached-bytes":
				inventory.payloadBytes++
			}
			if mode == "prior-frontier" {
				before = r.current // Deliberately invalid DATA, not a live pin.
			}
			_, err := r.apply(s, &inventory, record.fresh, record.intent, record.commit, record.frontier)
			if mode == "normal" {
				if err != nil || r.failed || r.current.sequence != 2 || r.body.ledger == nil || r.body.ledger == u.ledger {
					t.Fatal("normal independent reconstruction", err)
				}
			} else if err == nil || !r.failed || r.current != before {
				t.Fatal("incomplete/crossed data advanced a reusable frontier", mode, err)
			}
		})
	}
}

func TestProviderFrontierReconstructionChecksPolicyAndIndexIdentity(t *testing.T) {
	u := providerFrontierUnitFixture(t)
	for _, mode := range []string{"policy", "source"} {
		t.Run(mode, func(t *testing.T) {
			config := u.s.config
			if mode == "policy" {
				policy := *config.policy
				policy.raw = `{}`
				config.policy = &policy
			} else {
				config.identity.SourceID = ""
			}
			if _, err := newProviderFrontierReplay(config); err == nil {
				t.Fatal("unparsed policy or absent index identity constructed replay")
			}
		})
	}
}
