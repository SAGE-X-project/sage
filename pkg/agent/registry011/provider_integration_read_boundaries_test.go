package registry011

import (
	"strconv"
	"testing"
)

// This independent view reads only this test's normal owned originals. It has
// no live owner, ACK, dispatch permit, operating principal or recovery power.
func providerReadDataView(t *testing.T, s *providerCustodyStore) *providerCustodyStore {
	t.Helper()
	empty, err := newProviderIntegrationInventory("", nil, s.config.bounds.maximumEntries, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	return &providerCustodyStore{config: s.config, custody: s.custody, evidence: s.evidence, clock: s.clock, tip: s.config.anchor, objectsRoot: empty.digest, setupOrigin: s.setupOrigin, objects: make(map[string]providerIntegrationObjectRef), operations: make(map[string]bool), bindings: make(map[string]string)}
}

func TestProviderIntegrationSetupDataRequiresTheOriginalClockAndEnvelope(t *testing.T) {
	s, _, _, _ := providerStoreFixture(t)
	raw, err := s.read(s.custody, providerCustodyName("setup-intent", 0), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	// This finite clock value is parser input, never installed in an owner.
	now := s.setupOrigin
	if !providerReadDataView(t, s).checkSetup(raw, 0, now) {
		t.Fatal("normal original setup did not parse")
	}
	for _, mode := range []string{"shape", "digest", "clock-id", "epoch", "future-wall", "future-mono", "uncertainty", "work", "attempt", "noninteger"} {
		t.Run(mode, func(t *testing.T) {
			now := s.setupOrigin
			m := contentJSON(raw).(map[string]any)
			c := m["clock"].(map[string]any)
			switch mode {
			case "shape":
				m["extra"] = true
			case "digest":
				m["expected_tip"] = "invalid"
			case "clock-id":
				c["id"] = admissionTestDigest("f")
			case "epoch":
				c["epoch"] = admissionTestDigest("f")
			case "future-wall":
				now.WallMS--
			case "future-mono":
				c["monotonic_ms"] = strconv.FormatInt(now.MonotonicMS+1, 10)
			case "uncertainty":
				c["uncertainty_ms"] = "99"
			case "work":
				m["work"].(map[string]any)["requests"] = "1"
			case "attempt":
				m["attempt"] = "1"
			case "noninteger":
				c["wall_ms"] = "bad"
			}
			if providerReadDataView(t, s).checkSetup(providerIntegrationJSON(m), 0, now) {
				t.Fatal("changed setup data passed", mode)
			}
		})
	}
}

func TestProviderIntegrationIntentDataCannotChangeIdentityClockOrPriorRoot(t *testing.T) {
	_, s := providerPendingIntegrationUnit(t)
	raw, err := s.read(s.custody, providerCustodyName("intent", 1), 64<<10)
	if err != nil || providerReadDataView(t, s).checkIntent(raw, 1) == nil {
		t.Fatal("normal initial intent data", err)
	}
	for _, mode := range []string{"shape", "sequence", "identity", "previous", "duplicate", "kind", "initial-root", "later-root", "metadata", "clock-id", "clock-number"} {
		t.Run(mode, func(t *testing.T) {
			view := providerReadDataView(t, s)
			m := contentJSON(raw).(map[string]any)
			switch mode {
			case "shape":
				m["extra"] = true
			case "sequence":
				m["sequence"] = "2"
			case "identity":
				m["instance_key"] = admissionTestDigest("f")
			case "previous":
				m["previous_commit"] = admissionTestDigest("f")
			case "duplicate":
				view.operations[contentString(m["operation"])] = true
			case "kind":
				m["kind"] = "unknown"
			case "initial-root":
				m["expected_enrollment_root"] = admissionTestDigest("f")
			case "later-root":
				view.enrollment = admissionTestDigest("f")
			case "metadata":
				m["metadata_reservation"].(map[string]any)["bytes_upper"] = "1"
			case "clock-id":
				m["clock"].(map[string]any)["id"] = admissionTestDigest("f")
			case "clock-number":
				m["clock"].(map[string]any)["monotonic_ms"] = "bad"
			}
			if view.checkIntent(providerIntegrationJSON(m), 1) != nil {
				t.Fatal("changed intent data passed", mode)
			}
		})
	}
}

func TestProviderIntegrationInventoryReadRejectsMissingOrUnfundedOriginals(t *testing.T) {
	_, s := providerPendingIntegrationUnit(t)
	raw, err := s.read(s.custody, providerCustodyName("intent", 1), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	digest := contentString(contentJSON(raw).(map[string]any)["objects_root"])
	view := providerReadDataView(t, s)
	names, err := providerDirectoryNames(s.evidence, s.config.bounds.maximumObjects+2*s.config.bounds.maximumCommits+16)
	if err != nil || view.loadInventory(digest, names) != nil || len(view.newReferences) == 0 || view.stored == 0 {
		t.Fatal("normal bounded inventory read", err)
	}
	ref := view.newReferences[0]
	for _, mode := range []string{"inventory-name", "object-name", "duplicate", "objects-cap", "object-bytes", "store-cap", "physical-cap"} {
		t.Run(mode, func(t *testing.T) {
			v := providerReadDataView(t, s)
			selected, err := providerDirectoryNames(s.evidence, s.config.bounds.maximumObjects+2*s.config.bounds.maximumCommits+16)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "inventory-name":
				delete(selected, "inventory-"+digest[2:])
			case "object-name":
				delete(selected, providerObjectName(ref.kind, ref.digest))
			case "duplicate":
				v.objects[providerIntegrationRefKey(ref)] = ref
			case "objects-cap":
				v.config.bounds.maximumObjects = len(view.newReferences) - 1
			case "object-bytes":
				v.config.bounds.maximumObjectBytes = 1
			case "store-cap":
				v.config.bounds.maximumStoredBytes = 1
			case "physical-cap":
				v.stored = v.config.bounds.maximumStoredBytes
			}
			if v.loadInventory(digest, selected) == nil {
				t.Fatal("missing/unfunded original accepted", mode)
			}
		})
	}
	if view.loadInventory(digest, map[string]bool{}) == nil {
		t.Fatal("missing original inventory name accepted")
	}
	if !s.valid() || s.sequence != 2 || s.replay.ledger == nil {
		t.Fatal("independent parser data changed the live owner")
	}
}

func TestProviderIntegrationPrefixDataCannotTruncateAnExistingCompletePrefix(t *testing.T) {
	_, s := providerPendingIntegrationUnit(t)
	now, err := s.sample()
	if err != nil {
		t.Fatal(err)
	}
	view := providerReadDataView(t, s)
	if view.loadPrefix(now) != nil || view.sequence != s.sequence || view.tip != s.tip || view.enrollment != s.enrollment {
		t.Fatal("independent original-prefix read disagrees with normal fixture")
	}
	short := providerReadDataView(t, s)
	short.config.bounds.maximumCommits = 1
	if short.loadPrefix(now) == nil {
		t.Fatal("end-of-cap was misreported as the end of the retained history")
	}
	closed := providerReadDataView(t, s)
	if s.close() != nil {
		t.Fatal("normal owned store close")
	}
	if closed.loadPrefix(now) == nil {
		t.Fatal("closed original root produced a complete prefix")
	}
}
