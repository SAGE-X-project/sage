package registry011

import (
	"strconv"
	"testing"
)

func providerHighwaterFixture(t *testing.T) (*providerFrontierUnit, providerFrontierHighwater, providerFrontierHighwater) {
	t.Helper()
	u := providerFrontierUnitFixture(t)
	config := u.s.config
	h0 := providerFrontierHighwater{identity: config.identity, fence: 1, profile: config.profile, commit: config.anchor, setup: u.r.current.setup}
	raw, err := h0.bytes(config)
	if err != nil {
		t.Fatal(err)
	}
	h1 := h0
	h1.ordinal, h1.sequence, h1.previous = 1, 1, providerCandidateHash("sage-provider-custody-highwater-v1", raw)
	h1.commit, h1.frontier = u.r.current.commit, providerCandidateHash("sage-provider-custody-frontier-v1", u.records[0].frontier)
	return u, h0, h1
}

func TestProviderHighwaterClosedDataAndMainRelationDoNotCreateAuthority(t *testing.T) {
	u, h0, h1 := providerHighwaterFixture(t)
	for _, h := range []providerFrontierHighwater{h0, h1} {
		raw, err := h.bytes(u.s.config)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := parseProviderFrontierHighwater(raw, u.s.config)
		if err != nil || parsed != h {
			t.Fatal("closed high-water data", err)
		}
		object := contentJSON(raw).(map[string]any)
		for key := range object {
			for _, mode := range []string{"missing", "wrong"} {
				t.Run(key+"/"+mode, func(t *testing.T) {
					copy := contentJSON(raw).(map[string]any)
					if mode == "missing" {
						delete(copy, key)
					} else {
						copy[key] = "invalid"
					}
					if _, err := parseProviderFrontierHighwater(providerIntegrationJSON(copy), u.s.config); err == nil {
						t.Fatal("malformed high-water data")
					}
				})
			}
		}
	}
	record := u.records[0]
	if !checkProviderHighwaterMain(h0, h1, u.s.config, record.intent, record.commit, record.frontier) {
		t.Fatal("existing commit/frontier main relation")
	}
	for _, name := range []string{"ordinal", "main-sequence", "previous", "profile", "fence", "commit", "setup", "frontier", "intent", "bytes"} {
		t.Run(name, func(t *testing.T) {
			after, intent, commit := h1, record.intent, record.commit
			switch name {
			case "ordinal":
				after.ordinal++
			case "main-sequence":
				after.sequence++
			case "previous":
				after.previous = admissionTestDigest("f")
			case "profile":
				after.profile = admissionTestDigest("f")
			case "fence":
				after.fence++
			case "commit":
				after.commit = admissionTestDigest("f")
			case "setup":
				after.setup = admissionTestDigest("f")
			case "frontier":
				after.frontier = admissionTestDigest("f")
			case "intent":
				intent = []byte(`{}`)
			case "bytes":
				commit = []byte(`{}`)
			}
			if checkProviderHighwaterMain(h0, after, u.s.config, intent, commit, record.frontier) {
				t.Fatal("changed main relation")
			}
		})
	}
}

func TestProviderHighwaterSetupKeepsMainAndUsesExactNextAttempt(t *testing.T) {
	u, _, before := providerHighwaterFixture(t)
	config := u.s.config
	sample := u.clock
	sample.WallMS++
	sample.MonotonicMS++
	copy := config
	copy.attempt, copy.expectedTip, copy.expectedSetupTip = 1, before.commit, before.setup
	claim := providerCustodySetup(copy, sample)
	commit := providerSetupCommit(claim)
	previous, err := before.bytes(config)
	if err != nil {
		t.Fatal(err)
	}
	after := before
	after.ordinal++
	after.previous = providerCandidateHash("sage-provider-custody-highwater-v1", previous)
	after.setup = providerCandidateHash("sage-provider-authority-setup-commit-v1", commit)
	if !checkProviderHighwaterSetup(before, after, config, claim, commit) {
		t.Fatal("exact next setup relation")
	}
	for _, mode := range []string{"main", "frontier", "same-setup", "attempt", "commit", "generation", "setup-count"} {
		t.Run(mode, func(t *testing.T) {
			h, localClaim, localCommit := after, claim, commit
			switch mode {
			case "main":
				h.commit = admissionTestDigest("f")
			case "frontier":
				h.frontier = admissionTestDigest("f")
			case "same-setup":
				h.setup = before.setup
			case "attempt":
				old := copy
				old.attempt = 0
				localClaim = providerCustodySetup(old, sample)
				localCommit = providerSetupCommit(localClaim)
				h.setup = providerCandidateHash("sage-provider-authority-setup-commit-v1", localCommit)
			case "commit":
				localCommit = []byte(`{}`)
			case "generation":
				h.identity.Generation++
			case "setup-count":
				h.ordinal = h.sequence + uint64(config.bounds.maximumSetupRecoveries) + 1
			}
			if checkProviderHighwaterSetup(before, h, config, localClaim, localCommit) {
				t.Fatal("setup changed the main lineage or reset its attempts")
			}
		})
	}
}

func TestProviderHighwaterMainRefusesClosedDataUnderAnotherPolicy(t *testing.T) {
	u, before, after := providerHighwaterFixture(t)
	record := u.records[0]
	m := contentJSON(record.intent).(map[string]any)
	m["policy"] = admissionTestDigest("e")
	intent := providerIntegrationJSON(m)
	commit := providerFrontierCommit(intent)
	f, err := parseProviderFrontierRecord(record.frontier, u.s.config.identity)
	if err != nil {
		t.Fatal(err)
	}
	f.commit = providerCandidateHash("sage-provider-custody-commit-v2", commit)
	frontier, err := f.bytes()
	if err != nil {
		t.Fatal(err)
	}
	after.commit, after.frontier = f.commit, providerCandidateHash("sage-provider-custody-frontier-v1", frontier)
	// These are self-consistent inert canonical data, without any file write,
	// dispatch or selected authority. Policy must still bind to the fixed context.
	if checkProviderHighwaterMain(before, after, u.s.config, intent, commit, frontier) {
		t.Fatal("high-water relation omitted the selected context policy")
	}
}

// These are finite retained DATA comparisons, without dispatch or a live role.
func TestProviderHighwaterSetupAttemptBoundaries(t *testing.T) {
	u, _, base := providerHighwaterFixture(t)
	config := u.s.config
	config.bounds.maximumSetupRecoveries = 64
	config.profile = providerIntegrationProfile(config.bounds, config.capture)
	base.profile = config.profile
	sample := u.clock
	sample.WallMS++
	sample.MonotonicMS++
	for _, attempt := range []int{1, 64} {
		t.Run(strconv.Itoa(attempt), func(t *testing.T) {
			before := base
			before.ordinal = before.sequence + uint64(attempt-1)
			previous, err := before.bytes(config)
			if err != nil {
				t.Fatal(err)
			}
			copy := config
			copy.attempt, copy.expectedTip, copy.expectedSetupTip = attempt, before.commit, before.setup
			claim := providerCustodySetup(copy, sample)
			commit := providerSetupCommit(claim)
			after := before
			after.ordinal++
			after.previous = providerCandidateHash("sage-provider-custody-highwater-v1", previous)
			after.setup = providerCandidateHash("sage-provider-authority-setup-commit-v1", commit)
			if !checkProviderHighwaterSetup(before, after, config, claim, commit) {
				t.Fatal("valid setup attempt boundary refused")
			}
			for _, invalid := range []string{"0", "65", "2147483648", "9223372036854775808", "18446744073709551615"} {
				m := contentJSON(claim).(map[string]any)
				m["attempt"] = invalid
				raw := providerIntegrationJSON(m)
				completed := providerSetupCommit(raw)
				bad := after
				bad.setup = providerCandidateHash("sage-provider-authority-setup-commit-v1", completed)
				if checkProviderHighwaterSetup(before, bad, config, raw, completed) {
					t.Fatalf("invalid setup attempt %s accepted", invalid)
				}
			}
		})
	}
}
