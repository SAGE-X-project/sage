package registry011

import "testing"

func providerFrontierFixture(t *testing.T) providerFrontierRecord {
	t.Helper()
	s, m, root := providerIndexFixture(t)
	return providerFrontierRecord{identity: m.intent.identity, fence: 1, sequence: 1, commit: admissionTestDigest("4"), setup: admissionTestDigest("5"), objects: admissionTestDigest("6"), enrollment: root.digest, indexes: s.heads, initial: m.intent.clock, clock: m.intent.clock, deadline: m.intent.deadline}
}

func TestProviderFrontierRecordIsClosedAcyclicData(t *testing.T) {
	f := providerFrontierFixture(t)
	raw, err := f.bytes()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parseProviderFrontierRecord(raw, f.identity)
	if err != nil || parsed != f {
		t.Fatal("frontier data roundtrip", err)
	}
	object := contentJSON(raw).(map[string]any)
	if object["previous_frontier"] != nil || object["admission_digest"] != nil || object["budget_root"] != nil || object["ledger_seal"] != nil || object["checkpoint"] != nil || object["replay_state_digest"] != nil {
		t.Fatal("initial record named future objects")
	}
	if object["custody_commit"] != f.commit {
		t.Fatal("receipt lost already existing commit")
	}
	future := contentJSON(raw).(map[string]any)
	future["admission_digest"], future["budget_root"] = admissionTestDigest("a"), admissionTestDigest("b")
	if _, err := parseProviderFrontierRecord(providerIntegrationJSON(future), f.identity); err == nil {
		t.Fatal("E0 frontier named its future admission/budget")
	}
	for key := range object {
		for _, mode := range []string{"missing", "wrong"} {
			t.Run(key+"/"+mode, func(t *testing.T) {
				copy := contentJSON(raw).(map[string]any)
				if mode == "missing" {
					delete(copy, key)
				} else {
					copy[key] = "invalid"
				}
				if _, err := parseProviderFrontierRecord(providerIntegrationJSON(copy), f.identity); err == nil {
					t.Fatal("invalid frontier field")
				}
			})
		}
	}
}

func TestProviderFrontierAdvancePreservesOriginalAdmissionAndClock(t *testing.T) {
	before := providerFrontierFixture(t)
	raw, err := before.bytes()
	if err != nil {
		t.Fatal(err)
	}
	after := before
	after.sequence, after.previous = 2, providerCandidateHash("sage-provider-custody-frontier-v1", raw)
	after.commit, after.admission, after.budget = admissionTestDigest("7"), admissionTestDigest("8"), admissionTestDigest("9")
	after.clock.WallMS++
	after.clock.MonotonicMS++
	if !checkProviderFrontierAdvance(before, after) {
		t.Fatal("acyclic next frontier")
	}
	for _, name := range []string{"sequence", "previous", "fence", "identity", "initial-clock", "deadline", "backwards-clock", "unpaired-budget", "future-checkpoint", "expired", "cleared-admission", "replaced-admission"} {
		t.Run(name, func(t *testing.T) {
			a, b := before, after
			switch name {
			case "sequence":
				b.sequence++
			case "previous":
				b.previous = admissionTestDigest("f")
			case "fence":
				b.fence++
			case "identity":
				b.identity.SourceID = admissionTestDigest("f")
			case "initial-clock":
				b.initial.WallMS++
			case "deadline":
				b.deadline++
			case "backwards-clock":
				b.clock.MonotonicMS--
			case "unpaired-budget":
				b.budget = ""
			case "future-checkpoint":
				b.checkpoint = admissionTestDigest("a")
			case "expired":
				b.clock.WallMS = int64(b.deadline) + 1
				b.clock.MonotonicMS += b.clock.WallMS - before.clock.WallMS
			case "cleared-admission":
				a = after
				b = after
				b.sequence++
				aRaw, _ := a.bytes()
				b.previous = providerCandidateHash("sage-provider-custody-frontier-v1", aRaw)
				b.admission, b.budget = "", ""
			case "replaced-admission":
				a = after
				b = after
				b.sequence++
				aRaw, _ := a.bytes()
				b.previous = providerCandidateHash("sage-provider-custody-frontier-v1", aRaw)
				b.admission = admissionTestDigest("e")
			}
			if checkProviderFrontierAdvance(a, b) {
				t.Fatal("original frontier relation changed")
			}
		})
	}
}
