package registry011

import "testing"

func providerFrontierIntentFixture(t *testing.T) (providerFrontierLink, *CandidateEnrollmentRoot, *providerIntegrationInventory, ClockSample, providerIntegrationBounds) {
	t.Helper()
	_, m, root := providerIndexFixture(t)
	bounds, _ := providerFrontierBoundsFixture()
	empty, err := newProviderFrontierInventory("", nil, bounds.maximumEntries, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	object, err := newProviderIntegrationObject("enrollment", root.CanonicalBytes(), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := newProviderFrontierInventory(empty.digest, []providerIntegrationObject{object}, bounds.maximumEntries, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	link := providerFrontierLink{identity: root.identity, fence: root.fence, commit: admissionTestDigest("4"), setup: admissionTestDigest("5"), objects: empty.digest}
	d := m.intent.clock
	clock := ClockSample(d) // Inert unit data, never an owned-clock constructor.
	return link, root, inventory, clock, bounds
}

func TestProviderFrontierIntentAndCommitNameOnlyExistingPredecessors(t *testing.T) {
	link, root, inventory, clock, bounds := providerFrontierIntentFixture(t)
	operation := admissionTestDigest("a")
	intent, err := providerFrontierIntent(link, root, inventory, "enroll", operation, clock, bounds)
	if err != nil || !checkProviderFrontierIntent(intent, link, root, inventory, "enroll", operation, bounds) {
		t.Fatal("exact V2 candidate intent", err)
	}
	value := providerFrontierIntentObject(intent)
	if value["previous_frontier"] != nil || value["expected_enrollment_root"] != nil || value["sequence"] != "1" || value["previous_commit"] != link.commit || value["setup_head"] != link.setup {
		t.Fatal("initial intent names a future root")
	}
	commit := providerFrontierCommit(intent)
	object := contentJSON(commit).(map[string]any)
	if len(object) != 13 || object["format"] != "sage-provider-custody-commit-v2" || object["intent_digest"] != providerCandidateHash("sage-provider-custody-intent-v2", intent) || object["previous_frontier"] != nil || object["setup_head"] != link.setup {
		t.Fatal("V2 commit lost predecessor/intent identity")
	}
	if _, hasSelf := object["frontier"]; hasSelf {
		t.Fatal("commit includes its future frontier")
	}
	for key := range value {
		for _, mode := range []string{"missing", "wrong"} {
			t.Run(key+"/"+mode, func(t *testing.T) {
				copy := contentJSON(intent).(map[string]any)
				if mode == "missing" {
					delete(copy, key)
				} else {
					copy[key] = "invalid"
				}
				if checkProviderFrontierIntent(providerIntegrationJSON(copy), link, root, inventory, "enroll", operation, bounds) {
					t.Fatal("invalid exact V2 intent")
				}
			})
		}
	}
}

func TestProviderFrontierIntentRefusesWrongContextAndV1Relabeling(t *testing.T) {
	link, root, inventory, clock, bounds := providerFrontierIntentFixture(t)
	for _, mode := range []string{"frontier", "setup", "previous-objects", "sequence", "fence", "root", "inventory", "kind", "operation", "clock", "bounds"} {
		t.Run(mode, func(t *testing.T) {
			prior, after, objects, sample, profile := link, *root, inventory, clock, bounds
			kind, operation := "enroll", admissionTestDigest("a")
			switch mode {
			case "frontier":
				prior.frontier = admissionTestDigest("f")
			case "setup":
				prior.setup = "invalid"
			case "previous-objects":
				prior.objects = admissionTestDigest("f")
			case "sequence":
				prior.sequence = ^uint64(0)
			case "fence":
				after.fence++
			case "root":
				after.digest = admissionTestDigest("f")
			case "inventory":
				objects = nil
			case "kind":
				kind = "admit"
			case "operation":
				operation = "invalid"
			case "clock":
				sample.WallMS = -1
			case "bounds":
				profile.continuation = nil
			}
			if _, err := providerFrontierIntent(prior, &after, objects, kind, operation, sample, profile); err == nil {
				t.Fatal("wrong V2 context")
			}
		})
	}
	if providerFrontierCommit([]byte(`{"format":"sage-provider-custody-intent-v1"}`)) != nil {
		t.Fatal("V1 intent relabeled as V2")
	}
	if checkProviderFrontierIntent(nil, link, root, inventory, "enroll", admissionTestDigest("a"), bounds) {
		t.Fatal("missing intent")
	}
}

func TestProviderFrontierIntentRequiresTheImmediateEnrollmentSuccessor(t *testing.T) {
	link, root, inventory, clock, bounds := providerFrontierIntentFixture(t)
	link.sequence, link.frontier, link.enrollment, link.objects = 1, admissionTestDigest("b"), root.digest, inventory.digest
	value := contentJSON(root.CanonicalBytes()).(map[string]any)
	value["sequence"], value["previous_root"], value["budget_root"] = "1", root.digest, admissionTestDigest("c")
	after, err := ParseCandidateEnrollmentRoot(&ProviderCandidateContext{identity: root.identity, bound: true}, providerIntegrationJSON(value), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	object, err := newProviderIntegrationObject("enrollment", after.CanonicalBytes(), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	next, err := newProviderFrontierInventory(link.objects, []providerIntegrationObject{object}, bounds.maximumEntries, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := providerFrontierIntent(link, after, next, "admit", admissionTestDigest("d"), clock, bounds); err != nil {
		t.Fatal("immediate successor refused", err)
	}
	if _, err := providerFrontierIntent(link, root, next, "admit", admissionTestDigest("d"), clock, bounds); err == nil {
		t.Fatal("old enrollment root reused")
	}
	for _, mode := range []string{"skip", "different-parent"} {
		copy := contentJSON(after.CanonicalBytes()).(map[string]any)
		if mode == "skip" {
			copy["sequence"] = "2"
		} else {
			copy["previous_root"] = admissionTestDigest("f")
		}
		candidate, err := ParseCandidateEnrollmentRoot(&ProviderCandidateContext{identity: root.identity, bound: true}, providerIntegrationJSON(copy), 64<<10)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := providerFrontierIntent(link, candidate, next, "admit", admissionTestDigest("d"), clock, bounds); err == nil {
			t.Fatal("non-successor enrollment", mode)
		}
	}
}
