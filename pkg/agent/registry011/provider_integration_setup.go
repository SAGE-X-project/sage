package registry011

import (
	"bytes"
	"math"
)

type providerSetupReference struct {
	commit, claim, expectedTip string
	ordinal                    int
	clock                      CandidateClockDiagnostic
}

type providerCustodyPosition struct {
	sequence uint64
	clock    CandidateClockDiagnostic
	next     *CandidateClockDiagnostic
}

func providerSetupCommit(claim []byte) []byte {
	m, ok := contentJSON(claim).(map[string]any)
	if !ok || m["format"] != "sage-provider-authority-setup-v1" {
		return nil
	}
	return providerIntegrationJSON(map[string]any{"format": "sage-provider-authority-setup-commit-v1", "anchor": m["anchor"], "namespace": m["namespace"], "instance_key": m["instance_key"], "attempt": m["attempt"], "previous_setup_commit": m["previous_setup_commit"], "operation": m["operation"], "claim_digest": providerCandidateHash("sage-provider-authority-setup-v1", claim)})
}

// A setup charge is an independently retained prefix, not a single local file.
// Its high-water is separate from later custody commits: no self/future E0/I
// reference is necessary. Partial writes retain intent/charge and quarantine.
func (s *providerCustodyStore) writeSetup(clock ClockSample, anchors uint64) error {
	claim := providerCustodySetup(s.config, clock)
	commit := providerSetupCommit(claim)
	b := s.config.bounds
	bytesUpper, ok := providerUpperSum(anchors, uint64(2*len(claim)+len(commit)), 7*b.fileOverheadBytes)
	if !ok || bytesUpper > b.setupRetained || uint64(len(claim)) > b.maximumMetadataPerCommit || uint64(len(commit)) > b.maximumMetadataPerCommit {
		return errProviderIntegration
	}
	n := uint64(s.config.attempt)
	if s.write(s.custody, providerCustodyName("setup-intent", n), claim) != nil || s.write(s.evidence, providerCustodyName("setup-claim", n), claim) != nil || s.write(s.custody, providerCustodyName("setup-commit", n), commit) != nil {
		return errProviderIntegration
	}
	return nil // This alone is not an ACK; loadSetupPrefix must agree completely.
}

func (s *providerCustodyStore) loadSetupPrefix(custodyNames, localNames map[string]bool, now ClockSample) error {
	tip := s.config.anchor
	s.setupCommits = make(map[string]string)
	s.setups = nil
	for n := 0; n <= s.config.attempt; n++ {
		intentName, commitName, claimName := providerCustodyName("setup-intent", uint64(n)), providerCustodyName("setup-commit", uint64(n)), providerCustodyName("setup-claim", uint64(n))
		if !custodyNames[intentName] || !custodyNames[commitName] || !localNames[claimName] || n == s.config.attempt && tip != s.config.expectedSetupTip {
			return errProviderIntegration
		}
		intent, a := s.read(s.custody, intentName, int64(s.config.bounds.maximumMetadataPerCommit))
		claim, b := s.read(s.evidence, claimName, int64(s.config.bounds.maximumMetadataPerCommit))
		commit, c := s.read(s.custody, commitName, int64(s.config.bounds.maximumMetadataPerCommit))
		if a != nil || b != nil || c != nil || !bytes.Equal(intent, claim) || !s.checkSetup(claim, n, now) || contentJSON(claim).(map[string]any)["previous_setup_commit"] != tip || !bytes.Equal(commit, providerSetupCommit(claim)) {
			return errProviderIntegration
		}
		tip = providerCandidateHash("sage-provider-authority-setup-commit-v1", commit)
		s.setupCommits[tip] = providerCandidateHash("sage-provider-authority-setup-v1", claim)
		m := contentJSON(claim).(map[string]any)
		sample, valid := providerStoredClock(m["clock"])
		if !valid || len(s.setups) != 0 && !candidateLedgerClockAdvance(s.setups[len(s.setups)-1].clock, sample) {
			return errProviderIntegration
		}
		s.setups = append(s.setups, providerSetupReference{tip, s.setupCommits[tip], contentString(m["expected_tip"]), n, sample})
		s.retainedRaw += uint64(len(intent) + len(claim) + len(commit))
		delete(custodyNames, intentName)
		delete(custodyNames, commitName)
		delete(localNames, claimName)
	}
	s.setupTip = tip
	return nil
}

// Select the setup that belonged to this main-stream boundary. A later reopen
// at a T/C tip must not retroactively replace the setup of an earlier E0.
// All interleaving is checked against the complete main prefix before ACK.
func (s *providerCustodyStore) setupForBoundary(tip string) (providerSetupReference, bool) {
	var selected providerSetupReference
	found := false
	for _, setup := range s.setups {
		if setup.expectedTip == tip {
			selected, found = setup, true
		}
	}
	return selected, found
}

func (s *providerCustodyStore) checkSetupLineage(positions map[string]providerCustodyPosition) bool {
	var previous uint64
	for ordinal, setup := range s.setups {
		position, exists := positions[setup.expectedTip]
		if !exists || setup.ordinal != ordinal || position.sequence < previous || !candidateLedgerClockAdvance(position.clock, setup.clock) || position.next != nil && !candidateLedgerClockAdvance(setup.clock, *position.next) {
			return false
		}
		previous = position.sequence
	}
	return len(s.setups) != 0 && s.setups[len(s.setups)-1].expectedTip == s.config.expectedTip
}

func providerStoredClock(value any) (CandidateClockDiagnostic, bool) {
	m := contentObject(value, "id", "epoch", "wall_ms", "monotonic_ms", "uncertainty_ms")
	if m == nil {
		return CandidateClockDiagnostic{}, false
	}
	var values [3]int64
	for i, name := range []string{"wall_ms", "monotonic_ms", "uncertainty_ms"} {
		value, valid := providerCandidateUint(m[name], false)
		if !valid || value > math.MaxInt64 {
			return CandidateClockDiagnostic{}, false
		}
		values[i] = int64(value)
	}
	clock := CandidateClockDiagnostic{ClockID: contentString(m["id"]), Epoch: contentString(m["epoch"]), WallMS: values[0], MonotonicMS: values[1], UncertaintyMS: values[2]}
	return clock, clock.ClockID != "" && clock.Epoch != "" && candidateLedgerClockAdvance(clock, clock)
}
