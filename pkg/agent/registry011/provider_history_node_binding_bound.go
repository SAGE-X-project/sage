package registry011

type providerHistoryNodeSignatureCost struct {
	signing, sender, room       candidateWorkVector
	preimageBytes, preimagePool uint64
}

// Alternative original signing-hash and sender rows for one complete
// transaction pass. Canonical transaction bytes share E/T; V is their
// independently derived value ceiling. Native curve/CPU/RAM and complete
// caller/window/failure costs remain separate. DATA creates no authority.
func providerHistoryNodeSignatureUpper(evidence, transactions, valueBytes uint64) (providerHistoryNodeSignatureCost, bool) {
	if evidence == 0 || evidence > 64<<20 || transactions == 0 || transactions > 16384 || valueBytes == 0 || valueBytes > evidence {
		return providerHistoryNodeSignatureCost{}, false
	}
	// Pinned supported tuples remove V/R/S; the protected legacy tuple adds
	// only the fixed chain and two zeros.256 also covers every RLP prefix.
	m, pool := valueBytes+256, evidence+256*transactions
	signing := (m+2)*pool + 5*transactions
	const maximum = uint64(64<<20) + 256
	return providerHistoryNodeSignatureCost{
		signing:       candidateWorkVector{10: signing},
		sender:        candidateWorkVector{2: 97 * transactions, 10: signing + 13788*transactions},
		room:          candidateWorkVector{10: maximum*maximum + 1},
		preimageBytes: m, preimagePool: pool,
	}, true
}

type providerHistoryJSONBindingCost struct {
	binding, room               candidateWorkVector
	sourceBytes, canonicalBytes uint64
}

// Successful closed Header/Transaction/Withdrawal binding, excluding the
// typed projection's separately observed hashes. Present ASCII-hex/null/list
// values have a compact spelling no longer than their original JSON. At most
// 32 absent null/yParity fields add at most32 bytes each. Arbitrary marshalers,
// number-valued JSON graphs and producer-reported output lengths are excluded.
func providerHistoryJSONBindingUpper(sourceBytes uint64) (providerHistoryJSONBindingCost, bool) {
	if sourceBytes < 4 || sourceBytes > 64<<20 {
		return providerHistoryJSONBindingCost{}, false
	}
	// Every intermediate is below2^55 in this original finite domain. The
	// tighter actual-output bound does not change the original preflight.
	s, j := sourceBytes, sourceBytes+1024
	maximum := 2*sourceBytes + 65536
	return providerHistoryJSONBindingCost{
		binding:     candidateWorkVector{2: s + 2*j, 10: s*s + 3*j*j + 5*s + 6*j + 8490},
		room:        candidateWorkVector{10: maximum*maximum + 1},
		sourceBytes: s, canonicalBytes: j,
	}, true
}
