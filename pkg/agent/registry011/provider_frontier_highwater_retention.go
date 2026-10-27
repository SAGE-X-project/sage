package registry011

import "bytes"

// Derive exact high-water byte charges from independently retained originals.
// Historical setup HEAD copies were physically charged by their own setup
// claim; their original lengths enter the common ledger once at the next main
// boundary. Only the NEW main high-water contributes new physical store bytes.
// No selected-principal, live-pin, atomicity or ACK claim follows from this data.
func (r *providerFrontierReplay) highwaterRetention(s *providerCustodyStore, after providerFrontierHighwater, setup providerSetupReference, intent, commit, frontier []byte) (providerFrontierRetention, error) {
	if r == nil || s == nil || setup.ordinal < 0 || setup.ordinal >= len(s.setups) || r.setupOrdinal > setup.ordinal || r.current.sequence != s.sequence {
		return providerFrontierRetention{}, errProviderIntegration
	}
	priorSetup := r.setupOrdinal
	if priorSetup < 0 {
		if s.sequence != 0 || r.current.sequence != 0 {
			return providerFrontierRetention{}, errProviderIntegration
		}
		priorSetup = 0
	}
	ordinal, ok := providerUpperSum(s.sequence, uint64(priorSetup))
	if !ok {
		return providerFrontierRetention{}, errProviderIntegration
	}
	before, raw, err := s.originalHighwater(ordinal)
	if err != nil || before.sequence != s.sequence || before.commit != s.tip || before.setup != s.setups[priorSetup].commit {
		return providerFrontierRetention{}, errProviderIntegration
	}
	if s.sequence == 0 {
		if before.ordinal != 0 || before.frontier != "" || before.previous != "" || before.fence != 1 {
			return providerFrontierRetention{}, errProviderIntegration
		}
	} else {
		previous, err := r.current.bytes()
		if err != nil || before.frontier != providerCandidateHash("sage-provider-custody-frontier-v1", previous) || before.setup != r.current.setup || before.fence != r.current.fence {
			return providerFrontierRetention{}, errProviderIntegration
		}
	}
	var priorBytes uint64
	if r.setupOrdinal < 0 {
		priorBytes, ok = providerUpperProduct(2, uint64(len(raw)))
		if !ok {
			return providerFrontierRetention{}, errProviderIntegration
		}
	}
	for attempt := priorSetup + 1; attempt <= setup.ordinal; attempt++ {
		ordinal, ok := providerUpperSum(s.sequence, uint64(attempt))
		if !ok {
			return providerFrontierRetention{}, errProviderIntegration
		}
		next, raw, err := s.originalHighwater(ordinal)
		claim, a := s.read(s.custody, providerCustodyName("setup-intent", uint64(attempt)), int64(s.config.bounds.maximumMetadataPerCommit))
		local, b := s.read(s.evidence, providerCustodyName("setup-claim", uint64(attempt)), int64(s.config.bounds.maximumMetadataPerCommit))
		completed, c := s.read(s.custody, providerCustodyName("setup-commit", uint64(attempt)), int64(s.config.bounds.maximumMetadataPerCommit))
		if err != nil || a != nil || b != nil || c != nil || !bytes.Equal(claim, local) || !checkProviderHighwaterSetup(before, next, s.config, claim, completed) || next.setup != s.setups[attempt].commit {
			return providerFrontierRetention{}, errProviderIntegration
		}
		copyBytes, aOK := providerUpperProduct(2, uint64(len(raw)))
		priorBytes, ok = providerUpperSum(priorBytes, copyBytes)
		if !aOK || !ok || priorBytes > s.config.bounds.continuation.maximumMetadataBytes {
			return providerFrontierRetention{}, errProviderIntegration
		}
		before = next
	}
	if before.setup != setup.commit || !checkProviderHighwaterMain(before, after, s.config, intent, commit, frontier) {
		return providerFrontierRetention{}, errProviderIntegration
	}
	newRaw, err := after.bytes(s.config)
	cost, ok := providerHighwaterFileCost(uint64(len(newRaw)), s.config.bounds)
	metadata, sumOK := providerUpperSum(cost.metadata, priorBytes)
	if err != nil || !ok || !sumOK || metadata > s.config.bounds.continuation.maximumMetadataBytes {
		return providerFrontierRetention{}, errProviderIntegration
	}
	cost.metadata = metadata
	return cost, nil
}

func (s *providerCustodyStore) originalHighwater(ordinal uint64) (providerFrontierHighwater, []byte, error) {
	maximum, ok := providerUpperSum(uint64(s.config.bounds.maximumCommits), uint64(s.config.bounds.maximumSetupRecoveries))
	if !ok || ordinal > maximum {
		return providerFrontierHighwater{}, nil, errProviderIntegration
	}
	raw, err := s.read(s.custody, providerCustodyName("highwater", ordinal), candidateLedgerMetadataLimit)
	if err != nil {
		return providerFrontierHighwater{}, nil, errProviderIntegration
	}
	h, err := parseProviderFrontierHighwater(raw, s.config)
	if err != nil || h.ordinal != ordinal {
		return providerFrontierHighwater{}, nil, errProviderIntegration
	}
	return h, raw, nil
}
