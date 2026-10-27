package registry011

import "testing"

func TestProviderGuardianBoundsHaveSeparateLifetimeAndClosedAuxiliaryCaps(t *testing.T) {
	config, _ := providerGuardianConfigFixture(t)
	b := config.bounds
	g := *b.guardian
	raw := contentJSON(providerIntegrationJSON(g.object()))
	parsed, ok := providerGuardianBoundsValue(raw, b)
	if !ok || parsed != g || !g.valid(b) {
		t.Fatal("explicit closed lifetime did not round trip")
	}
	used, ok := g.used(1)
	if !ok || used["guardian_file_allocations"] != "7" || used["guardian_committed_records"] != "2" {
		t.Fatal("guardian lifetime forgot pending allocation or genesis")
	}
	for _, mode := range []string{"attempts", "records", "objects", "retained", "live", "clocks", "work", "raw", "metadata", "resident", "duration", "private-records", "global-objects", "commits", "names", "name-bytes", "page", "seal-claims", "overflow", "coordinate"} {
		t.Run(mode, func(t *testing.T) {
			v, store, page := g, b, *b.continuation
			store.continuation = &page
			switch mode {
			case "attempts":
				v.attempts = 0
			case "records":
				v.claim.records = 5
			case "objects":
				v.claim.objects = 3
			case "retained":
				v.lifetime.retained = v.claim.retained
			case "live":
				v.claim.live++
			case "clocks":
				v.lifetime.clocks = 1
			case "work":
				v.lifetime.work[10] = 1
			case "raw":
				v.lifetime.raw = 1
			case "metadata":
				v.claim.metadata = 1
			case "resident":
				v.lifetime.resident = 1
			case "duration":
				v.claim.duration = 300001
			case "private-records":
				v.privateRecords = 1
			case "global-objects":
				v.objects = 1
			case "commits":
				v.commits = 1
			case "names":
				v.catalogEntries = 1
			case "name-bytes":
				v.catalogNameBytes = 1
			case "page":
				page.pageRecords = 8
			case "seal-claims":
				v.attempts = page.maximumRecords / 3
			case "overflow":
				v.claim.work[10] = ^uint64(0)
			case "coordinate":
				v.claim.work[0] = 1
			}
			if v.valid(store) {
				t.Fatal("unfunded lifetime/closing claim accepted", mode)
			}
		})
	}
	if _, ok := g.used(g.attempts + 1); ok {
		t.Fatal("lifetime attempts reset")
	}
	for _, value := range []any{nil, map[string]any{}, map[string]any{"format": "sage-provider-guardian-bounds-v1"}} {
		if _, ok := providerGuardianBoundsValue(value, b); ok {
			t.Fatal("incomplete bounds became a selected upper")
		}
	}
	for _, mode := range []string{"capacity", "scalar"} {
		t.Run(mode, func(t *testing.T) {
			value := contentJSON(providerIntegrationJSON(g.object())).(map[string]any)
			if mode == "capacity" {
				value["per_claim"].(map[string]any)["clock_reads"] = "invalid"
			} else {
				value["maximum_attempts"] = "invalid"
			}
			if _, ok := providerGuardianBoundsValue(value, b); ok {
				t.Fatal("non-numeric closed bound created a capacity", mode)
			}
		})
	}
}

func TestProviderGuardianPreflightAddsDistinctCheckedWorkStorageAndMemory(t *testing.T) {
	config, clock := providerGuardianConfigFixture(t)
	b := config.bounds
	s := &providerCustodyStore{config: config, clock: clock, firstLimits: &providerFirstAdmissionLimits{}}
	if !providerGuardianInitialFits(s) {
		t.Fatal("explicit complete O candidate did not fit")
	}
	work, physical, ok := providerGuardianBootstrapExtraUpper(b)
	extra, fits := providerGuardianFirstExtraUpper(b)
	ram, memory := providerGuardianMemoryExtra(b)
	stored, storedOK := providerGuardianInitialStored(b)
	if !ok || !fits || !memory || !storedOK || ram == 0 || stored != b.setupRetained+b.guardian.lifetime.retained || !providerGuardianStoredMatches(b, stored) || physical != 3*(candidateLedgerMetadataLimit+b.fileOverheadBytes) || work[0] != 0 || work[8] != 0 || work[2] == 0 || work[9] != 3*candidateLedgerMetadataLimit || extra[2] == 0 || extra[9] != 0 {
		t.Fatal("preflight mixed future hold, bytes, or protected work")
	}
	old := b
	old.guardian = nil
	if got, ok := providerGuardianInitialStored(old); !ok || got != old.setupRetained {
		t.Fatal("old O changed")
	}
	if _, _, ok := providerGuardianBootstrapExtraUpper(old); ok {
		t.Fatal("old profile received a guardian grant")
	}
	for _, mode := range []string{"no-guardian", "physical", "work", "clock", "used-clock", "memory"} {
		t.Run(mode, func(t *testing.T) {
			c := config
			owned := providerOwnedClock{maximum: clock.maximum} // Inert scalar preflight fixture; no clock read or lock copy.
			owned.owner = &owned
			local := &providerCustodyStore{config: c, clock: &owned, firstLimits: s.firstLimits}
			switch mode {
			case "no-guardian":
				local.config.bounds.guardian = nil
			case "physical":
				local.config.bounds.setupRetained = 1
			case "work":
				local.config.bounds.setupWork[2] = 1
			case "clock":
				owned.maximum = 1
			case "used-clock":
				owned.read = 1
			case "memory":
				page := *b.continuation
				page.maximumResidentBytes = 1
				local.config.bounds.continuation = &page
			}
			if providerGuardianInitialFits(local) {
				t.Fatal("incomplete original O was accepted", mode)
			}
		})
	}
}
