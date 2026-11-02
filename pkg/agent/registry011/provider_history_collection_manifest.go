package registry011

import (
	"strconv"
	"time"
)

// The manifest is immutable collection DATA, not the missing whole-source cost
// certificate. It has no Source, RPC, clock, signing or dispatch operation. The
// eventual production compiler must qualify all identities and fund every
// source, semantic, failure, final-page and resident row before opening H0.
type providerHistoryCollectionInput struct {
	journal                                                          JournalConfig
	codes                                                            [13]nodeCodeIdentity
	capture                                                          ProviderCaptureLimits
	encoding, gas                                                    string
	maximumSelected, blocks, native, effects, versions, presentStops uint64
	windowBlocks, contentBlocks                                      uint64
	firstRead                                                        providerHistoryFirstPReadBounds
	traceTimeout                                                     time.Duration
}

type providerHistoryCollectionManifest struct {
	owner                   *providerHistoryCollectionManifest
	input, sealed           providerHistoryCollectionInput
	raw, digest             string
	counts                  providerHistoryCollectionCounts
	sealedRaw, sealedDigest string
	sealedCounts            providerHistoryCollectionCounts
}

type providerHistoryCollectionCounts struct {
	ancestryWindows, contentWindows, contentGroups uint64
	requests, groups, groupClocks                  uint64
}

func (*providerHistoryCollectionManifest) UnmarshalJSON([]byte) error { return ErrConfiguration }

// Construction is an explicit pre-H0 DATA operation. It is intentionally not a
// providerHistoryProgramDerivation factory. A positive request/group count does
// not fund eleven work coordinates, RAM, elapsed time or native window brackets.
func newProviderHistoryCollectionManifestCandidate(in providerHistoryCollectionInput) (*providerHistoryCollectionManifest, error) {
	journal, _, err := journalConfiguration(in.journal)
	identity, _, identityErr := nodeIdentityManifest(nodeIdentityJSON, nodeABIJSON)
	if err != nil || identityErr != nil || !in.capture.valid() ||
		in.journal.Deployment.Number != sepoliaDeploymentBlock || in.journal.Deployment.Hash != identity.Deployment.BlockHash ||
		in.maximumSelected < sepoliaDeploymentBlock || in.blocks == 0 || in.blocks != in.maximumSelected-sepoliaDeploymentBlock+1 ||
		in.native > in.blocks || in.windowBlocks == 0 || in.windowBlocks > 4096 || in.windowBlocks > uint64(in.journal.Replay.MaxBlocks) ||
		in.contentBlocks == 0 || in.contentBlocks > 10 || in.contentBlocks > uint64(in.journal.Replay.MaxBlocks) ||
		in.blocks > 1<<20 || in.blocks-1 > in.journal.Observation.MaxAncestryHeaders || in.versions > 1<<20 || !in.firstRead.valid() ||
		in.traceTimeout <= 0 || in.traceTimeout > in.capture.Timeout {
		return nil, ErrConfiguration
	}
	switch in.encoding {
	case "debug-raw-rlp-v1", EthereumJSONBlocks:
	default:
		return nil, ErrConfiguration
	}
	switch in.gas {
	case SepoliaAmsterdamGasProfile, SepoliaAmsterdamNethermindGasProfile, SepoliaAmsterdamRootRegularGasProfile, SepoliaAmsterdamRootExecutionGasProfile:
	default:
		return nil, ErrConfiguration
	}
	for i, expected := range identity.Contracts {
		if in.codes[i] != expected {
			return nil, ErrConfiguration
		}
	}
	counts, ok := providerHistoryCollectionMaximumCounts(in)
	if !ok || in.presentStops > counts.contentWindows || counts.contentWindows > uint64(in.journal.MaxBatches) {
		return nil, ErrConfiguration
	}
	// Shared claim/created indexes and whole subject versions are distinct.
	// Bound declared fanout by the original per-window replay capacities, not
	// by a claim count that may stay one across many other-subject commands.
	effects, effectsOK := providerUpperProduct(counts.contentWindows, uint64(in.journal.Replay.MaxTransactions))
	versions, versionsOK := providerUpperProduct(counts.contentWindows, uint64(in.journal.Replay.History.MaxItems))
	if !effectsOK || !versionsOK || in.effects > effects || in.versions > versions {
		return nil, ErrConfiguration
	}
	raw := providerReplayCanonical(in.object(string(journal)), candidateLedgerMetadataLimit, nil)
	digest := providerReplayDigest("sage-provider-history-collection-manifest-v1", raw, nil)
	if len(raw) == 0 || digest == "" {
		return nil, ErrConfiguration
	}
	m := &providerHistoryCollectionManifest{input: in, sealed: in, raw: string(raw), digest: digest, counts: counts}
	m.sealedRaw, m.sealedDigest, m.sealedCounts = m.raw, m.digest, counts
	m.owner = m
	return m, nil
}

// The same closed fields are encoded during each independent original replay.
// A manifest digest alone cannot stand in for interpreting a changed input.
func (in providerHistoryCollectionInput) object(journal string) map[string]any {
	return map[string]any{
		"format": "sage-provider-history-collection-manifest-v1", "journal": journal, "codes": in.codes,
		"capture": in.capture, "block_encoding": in.encoding, "gas_profile": in.gas,
		"maximum_selected_height": in.maximumSelected, "maximum_inclusive_blocks": in.blocks,
		"maximum_native_blocks": in.native, "maximum_effect_transactions": in.effects,
		"maximum_history_versions": in.versions, "maximum_present_stops": in.presentStops,
		"window_blocks": in.windowBlocks, "content_blocks_per_group": in.contentBlocks,
		"rpc_members_per_group": providerHistoryGroupMembers,
		"trace_timeout_ns":      strconv.FormatInt(int64(in.traceTimeout), 10),
		"first_original_read":   in.firstRead.object(),
	}
}

// Scalar-only identity checks are safe under the original meter mutex. Actual
// replay must separately charge any interpretation of these retained bytes.
func (m *providerHistoryCollectionManifest) validSnapshot() bool {
	return m != nil && m.owner == m && m.input == m.sealed && m.raw != "" && lowerHash(m.digest) &&
		m.raw == m.sealedRaw && m.digest == m.sealedDigest && m.counts == m.sealedCounts
}

func (p *providerHistoryProgramOwner) collectionMatchesLocked() bool {
	if p == nil {
		return false
	}
	m := p.bounds.collection
	if m == nil {
		return p.collectionInput == (providerHistoryCollectionInput{}) && p.collectionRaw == "" && p.collectionDigest == "" && p.collectionCounts == (providerHistoryCollectionCounts{})
	}
	return m.validSnapshot() && m.input == p.collectionInput && m.raw == p.collectionRaw && m.digest == p.collectionDigest && m.counts == p.collectionCounts && m.input.capture == p.config.capture
}

// The selected original first P cannot narrow or renew the admitted program.
// At deployment itself every header field must equal the pinned deployment.
func (m *providerHistoryCollectionManifest) selectedObserved(h FinalizedHeader, meter *providerWorkMeter) bool {
	if !m.validSnapshot() || meter == nil || !validHeader(h) || h.Number < m.input.journal.Deployment.Number ||
		h.Number > m.input.maximumSelected || h.Number-m.input.journal.Deployment.Number >= m.input.blocks ||
		h.Timestamp < m.input.journal.Deployment.Timestamp ||
		!meter.event(providerCompareEvent, uint64(len(h.Hash)+len(h.ParentHash)+len(m.input.journal.Deployment.Hash)+len(m.input.journal.Deployment.ParentHash))) {
		return false
	}
	return h.Number != m.input.journal.Deployment.Number || h == m.input.journal.Deployment
}

// These are prospective maxima for the grammar, never the exact successful
// transcript or a per-window renewal of T/V. First P, final inspection/pages,
// new window/semantic roles and failed execution are deliberately not counted.
func providerHistoryCollectionMaximumCounts(in providerHistoryCollectionInput) (providerHistoryCollectionCounts, bool) {
	var out providerHistoryCollectionCounts
	if in.blocks == 0 || in.native > in.blocks || in.windowBlocks == 0 || in.windowBlocks > 4096 || in.contentBlocks == 0 || in.contentBlocks > 10 {
		return out, false
	}
	ceil := func(n, d uint64) uint64 {
		if n == 0 {
			return 0
		}
		return 1 + (n-1)/d
	}
	out.ancestryWindows, out.contentWindows = ceil(in.blocks-1, in.windowBlocks), ceil(in.blocks, in.windowBlocks)
	if in.presentStops > out.contentWindows {
		return providerHistoryCollectionCounts{}, false
	}
	full, a := providerUpperProduct(in.blocks/in.windowBlocks, ceil(in.windowBlocks, in.contentBlocks))
	if !a {
		return providerHistoryCollectionCounts{}, false
	}
	out.contentGroups, a = providerUpperSum(full, ceil(in.blocks%in.windowBlocks, in.contentBlocks))
	if !a {
		return providerHistoryCollectionCounts{}, false
	}
	windows, a := providerUpperSum(out.ancestryWindows, out.contentWindows)
	if !a {
		return providerHistoryCollectionCounts{}, false
	}
	boundaryRequests, b := providerUpperProduct(12+uint64(len(in.codes)), windows)
	blockRequests, c := providerUpperProduct(2, in.blocks)
	out.requests, a = providerUpperSum(in.blocks-1, blockRequests, in.native, boundaryRequests, out.contentWindows, in.presentStops, in.effects, in.versions)
	if !a || !b || !c {
		return providerHistoryCollectionCounts{}, false
	}
	// K=13 is fixed by the original deployment manifest. Start and end each
	// fit a nonempty table; no assumption is made for a future larger K.
	boundaryGroups, d := providerUpperProduct(2, windows)
	ancestryGroups := uint64(0)
	if in.blocks > 1 {
		ancestryGroups, a = providerUpperSum(ceil(in.blocks-1, providerHistoryGroupMembers), out.ancestryWindows-1)
		if !a {
			return providerHistoryCollectionCounts{}, false
		}
	}
	traces, versions := uint64(0), uint64(0)
	if in.effects > 0 {
		traces, a = providerUpperSum(ceil(in.effects, providerHistoryGroupMembers), out.contentGroups-1)
		if !a {
			return providerHistoryCollectionCounts{}, false
		}
		traces = min(traces, in.effects)
	}
	if in.versions > 0 {
		versions, a = providerUpperSum(ceil(in.versions, providerHistoryGroupMembers), out.contentWindows-1)
		if !a {
			return providerHistoryCollectionCounts{}, false
		}
		versions = min(versions, in.versions)
	}
	out.groups, a = providerUpperSum(boundaryGroups, ancestryGroups, out.contentGroups, out.contentWindows, in.presentStops, traces, versions)
	if !a || !d {
		return providerHistoryCollectionCounts{}, false
	}
	rpcClocks, e := providerUpperProduct(2, out.requests)
	groupClocks, f := providerUpperProduct(10, out.groups)
	out.groupClocks, a = providerUpperSum(rpcClocks, groupClocks)
	if !a || !e || !f {
		return providerHistoryCollectionCounts{}, false
	}
	return out, true
}
