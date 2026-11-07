package registry011

import "strings"

// These preflight limits bound the text traversed or retained by the compiler
// input copy. They are not operating storage limits or native RAM evidence.
// Existing policy/journal schemas bound raw text by64KiB and their string
// scalars by256 bytes; identities, code hashes and format names are smaller.
// The original combined directory limit remains2MiB. All input copies,
// parsing, transient allocations and their lifetimes still need funding in
// the enclosing pre-H0 compiler before any derivation can be constructed.
const (
	providerHistorySourceRawTextLimit   = 64 << 10
	providerHistorySourceFixedTextLimit = 256
)

func providerHistorySourceInputTextFits(c providerCustodyConfig) bool {
	if c.policy == nil || c.bounds.history == nil || c.bounds.history.collection == nil ||
		c.policy.raw == "" || len(c.policy.raw) > providerHistorySourceRawTextLimit ||
		len(c.custodyDirectory) > 2<<20 || len(c.evidenceDirectory) > (2<<20)-len(c.custodyDirectory) {
		return false
	}
	h, m, p, id := c.bounds.history, c.bounds.history.collection, c.policy, c.identity
	if len(m.raw) > providerHistorySourceRawTextLimit || len(m.sealedRaw) > providerHistorySourceRawTextLimit {
		return false
	}
	for _, text := range [...]string{
		c.anchor, c.expectedTip, c.expectedSetupTip, c.profile, c.expectedScanHead,
		id.Policy, id.Instance, id.Provider, id.Namespace, id.SourceID,
		p.policy, p.instance, p.provider, h.program, h.costs, m.digest, m.sealedDigest,
	} {
		if len(text) > providerHistorySourceFixedTextLimit {
			return false
		}
	}
	// Preflight both compared values before the seal/constructor checks. A
	// length check alone does not validate a seal, policy, journal or identity.
	for _, in := range [...]providerHistoryCollectionInput{m.input, m.sealed} {
		j := in.journal
		for _, text := range [...]string{
			in.encoding, in.gas, j.Registry, j.Agent, j.GenesisHash, j.CodeIdentity, j.ABIIdentity,
			j.SourceID, j.PolicyGeneration, j.Deployment.Hash, j.Deployment.ParentHash,
			j.Observation.ClockID, j.Observation.PolicyGeneration,
		} {
			if len(text) > providerHistorySourceFixedTextLimit {
				return false
			}
		}
		for _, code := range in.codes {
			if len(code.Name) > providerHistorySourceFixedTextLimit || len(code.Address) > providerHistorySourceFixedTextLimit ||
				len(code.SHA256) > providerHistorySourceFixedTextLimit {
				return false
			}
		}
	}
	return true
}

// Go strings are immutable values but a short substring can retain a much
// larger caller allocation. Copy every retained leaf, then let the original
// manifest constructor produce its own owner, seal, canonical bytes and hash.
// No old owner/seal is copied or repaired and no D/native permission is minted.
func providerHistoryCloneCollectionText(in providerHistoryCollectionInput) providerHistoryCollectionInput {
	in.encoding, in.gas = strings.Clone(in.encoding), strings.Clone(in.gas)
	j := in.journal
	j.Registry, j.Agent, j.GenesisHash = strings.Clone(j.Registry), strings.Clone(j.Agent), strings.Clone(j.GenesisHash)
	j.CodeIdentity, j.ABIIdentity = strings.Clone(j.CodeIdentity), strings.Clone(j.ABIIdentity)
	j.SourceID, j.PolicyGeneration = strings.Clone(j.SourceID), strings.Clone(j.PolicyGeneration)
	j.Deployment.Hash, j.Deployment.ParentHash = strings.Clone(j.Deployment.Hash), strings.Clone(j.Deployment.ParentHash)
	j.Observation.ClockID, j.Observation.PolicyGeneration = strings.Clone(j.Observation.ClockID), strings.Clone(j.Observation.PolicyGeneration)
	in.journal = j
	for i := range in.codes {
		in.codes[i].Name = strings.Clone(in.codes[i].Name)
		in.codes[i].Address = strings.Clone(in.codes[i].Address)
		in.codes[i].SHA256 = strings.Clone(in.codes[i].SHA256)
	}
	return in
}
