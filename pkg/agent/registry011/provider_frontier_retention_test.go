package registry011

import "testing"

// This is inert byte-accounting data. It is not a semantically admitted E0
// inventory, independent custody, selected operating identity or work permit.
func providerRetentionFixture(t *testing.T) (ProviderCandidateIdentity, *providerIntegrationInventory, []providerIntegrationObject, []byte, []byte, []byte, providerIntegrationBounds) {
	t.Helper()
	link, root, _, clock, bounds := providerFrontierIntentFixture(t)
	entry, err := newProviderIntegrationObject("enrollment", root.CanonicalBytes(), bounds.maximumObjectBytes)
	if err != nil {
		t.Fatal(err)
	}
	data, err := newProviderIntegrationObject("response", []byte("finite normal fixture"), bounds.maximumObjectBytes)
	if err != nil {
		t.Fatal(err)
	}
	fresh := []providerIntegrationObject{entry, data}
	inventory, err := newProviderFrontierInventory(link.objects, fresh, bounds.maximumEntries, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	intent, err := providerFrontierIntent(link, root, inventory, "enroll", admissionTestDigest("a"), clock, bounds)
	if err != nil {
		t.Fatal(err)
	}
	commit := providerFrontierCommit(intent)
	f := providerFrontierFixture(t)
	f.commit, f.objects = providerCandidateHash("sage-provider-custody-commit-v2", commit), inventory.digest
	frontier, err := f.bytes()
	if err != nil {
		t.Fatal(err)
	}
	return link.identity, inventory, fresh, intent, commit, frontier, bounds
}

func TestProviderFrontierRetentionIncludesEveryNewFileWithoutDoubleChargingLedger(t *testing.T) {
	id, inventory, fresh, intent, commit, frontier, bounds := providerRetentionFixture(t)
	cost, err := providerFrontierRetentionCost(id, inventory, fresh, intent, commit, frontier, bounds)
	meta := uint64(len(fresh[0].raw) + len(inventory.raw) + 2*len(intent) + len(commit) + len(frontier))
	plain := uint64(len(fresh[1].raw))
	if err != nil || cost.payload != plain || cost.metadata != meta || cost.objects != 2 || cost.stored != meta+plain+7*bounds.fileOverheadBytes {
		t.Fatal("exact original-byte/new-file accounting", err, cost)
	}
	for _, kind := range []string{"ticket", "consumption", "settlement", "ledger-node", "budget", "ledger-page", "ledger-seal", "index-empty", "index-mutation"} {
		if providerFrontierRetentionClass(kind) != 0 {
			t.Fatal("typed ledger/index bytes charged twice", kind)
		}
	}
	for _, kind := range []string{"request", "params", "response", "tls"} {
		if providerFrontierRetentionClass(kind) != 1 {
			t.Fatal("plaintext payload omitted", kind)
		}
	}
	for _, kind := range []string{"setup", "setup-commit", "enrollment", "admission", "exchange", "wire-binding", "bootstrap", "pin-input", "pin", "segment", "batch", "checkpoint", "verification-receipt"} {
		if providerFrontierRetentionClass(kind) != 2 {
			t.Fatal("additional typed metadata omitted", kind)
		}
	}
	for _, kind := range []string{"frontier", "unknown", ""} {
		if providerFrontierRetentionClass(kind) != -1 {
			t.Fatal("unknown/self frontier became a free inventory kind")
		}
	}
}

func TestProviderFrontierRetentionRejectsCrossedOrUnboundedDelta(t *testing.T) {
	id, inventory, fresh, intent, commit, frontier, bounds := providerRetentionFixture(t)
	for _, name := range []string{"identity", "inventory", "count", "raw", "digest", "kind", "intent", "commit", "frontier", "profile", "resident", "store"} {
		t.Run(name, func(t *testing.T) {
			identity, inv, objects, i, c, f, b := id, *inventory, append([]providerIntegrationObject(nil), fresh...), intent, commit, frontier, bounds
			private := *bounds.continuation
			b.continuation = &private
			switch name {
			case "identity":
				identity.Instance = admissionTestDigest("f")
			case "inventory":
				inv.digest = admissionTestDigest("f")
			case "count":
				objects = objects[:1]
			case "raw":
				objects[0].raw = ""
			case "digest":
				objects[1].digest = admissionTestDigest("f")
			case "kind":
				objects[1].kind = "unknown"
			case "intent":
				i = []byte(`{}`)
			case "commit":
				c = []byte(`{}`)
			case "frontier":
				f = []byte(`{}`)
			case "profile":
				b.continuation = nil
			case "resident":
				private.maximumResidentBytes = 1
			case "store":
				b.maximumStoredBytes = 1
			}
			if _, err := providerFrontierRetentionCost(identity, &inv, objects, i, c, f, b); err == nil {
				t.Fatal("invalid new-delta byte accounting")
			}
		})
	}
}
