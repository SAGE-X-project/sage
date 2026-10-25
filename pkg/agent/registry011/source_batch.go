package registry011

import (
	"context"
	"strconv"
)

// SourceClock samples the explicitly configured trusted local clock. It must
// not obtain its wall time, uncertainty or epoch from RPC or peer messages.
type SourceClock func() ClockSample

// SourceBatchEvidence is a complete bounded collection, not a Source-ready
// credential. ReplayJournal must independently replay it before committing;
// historical code-at-call, the provider, clock, external checkpoint retention,
// post-commit freshness and publication still require separate qualification.
type SourceBatchEvidence struct {
	Batch              ReplayBatch
	Started, Completed ClockSample
	Exchanges          []NodeExchange
}

// SepoliaEvidenceDigests identifies the exact bundled manifest and deployment
// ABI bytes. Digests identify inputs; they do not enroll a Source or establish
// the honesty of an RPC, historical code, a clock or a storage implementation.
func SepoliaEvidenceDigests() (codeIdentity, abiIdentity string) {
	return nodeHex(sha256Digest(nodeIdentityJSON)), nodeHex(sha256Digest(nodeABIJSON))
}

// CollectReplayBatch reads a complete consecutive block interval following the
// exact externally retained cursor. The first interval includes deployment.
// It retains the full selected-finalized-to-cursor parent path, even during
// backfill, and reads state/history at the interval's one exact stop block.
//
// This API explicitly uses ReadReplayBlockWithRootGas. It does not fall back to
// another tracer/provider, skip ancestry, acknowledge a checkpoint, activate a
// generation or authorize messages. A large initial gap may exceed the limits;
// failure returns no partial batch and never expands the supplied budgets.
func (c *ReadRPC) CollectReplayBatch(ctx context.Context, config JournalConfig, expected JournalCheckpoint, limits SourceReadLimits, clock SourceClock) (*SourceBatchEvidence, error) {
	if ctx == nil || clock == nil {
		return nil, ErrConfiguration
	}
	raw, checks, err := journalConfiguration(config)
	if err != nil {
		return nil, err
	}
	identity, _, err := nodeIdentityManifest(nodeIdentityJSON, nodeABIJSON)
	code, abi := SepoliaEvidenceDigests()
	if err != nil || config.CodeIdentity != code || config.ABIIdentity != abi || config.Deployment.Number != sepoliaDeploymentBlock || config.Deployment.Hash != identity.Deployment.BlockHash || limits.Node.Replay != config.Replay {
		return nil, ErrConfiguration
	}
	if expected.Sequence >= uint64(config.MaxBatches) || !lowerHash(expected.Digest) || !validHeader(expected.Anchor) || expected.Anchor.Number < config.Deployment.Number {
		return nil, ErrJournal
	}
	if expected.Sequence == 0 && (expected.Version != 0 || expected.Anchor != config.Deployment || expected.Digest != journalDigest(raw)) {
		return nil, ErrJournal
	}
	client, child, cancel, err := newSourceAcquisition(ctx, c, limits)
	if err != nil {
		return nil, err
	}
	defer cancel()
	// Reuse the per-node limit validator without issuing a request or treating
	// this configuration reference as an authenticated node observation.
	validation := &PinnedAgentReads{client, SepoliaRegistryAddress, config.Agent, config.Deployment}
	_, finish, err := validation.nodeStart(child, limits.Node)
	if err != nil {
		return nil, err
	}
	finish()
	start := clock()
	if start.ClockID != config.Observation.ClockID || start.Epoch == "" || start.WallMS < 0 || start.MonotonicMS < 0 || start.UncertaintyMS < 0 || start.UncertaintyMS > config.Observation.MaxClockUncertaintyMS {
		return nil, ErrClock
	}
	reads, err := client.PinAgentReads(child, SepoliaRegistryAddress, config.Agent)
	if err != nil {
		return nil, err
	}
	batch, err := collectSourceBatch(child, reads, config, expected, limits, checks, identity)
	if err != nil {
		return nil, err
	}
	completed := clock()
	if err := checks.CheckReadWindow(start, completed, batch.SelectedFinalized); err != nil {
		return nil, err
	}
	if err := child.Err(); err != nil {
		return nil, err
	}
	return &SourceBatchEvidence{batch, start, completed, client.acquisition.exchanges}, nil
}

func collectSourceBatch(ctx context.Context, reads *PinnedAgentReads, config JournalConfig, expected JournalCheckpoint, limits SourceReadLimits, checks *ObservationChecks, identity nodeIdentity) (ReplayBatch, error) {
	var empty ReplayBatch
	selected, base := reads.header, expected.Anchor
	if selected.Number < base.Number || selected.Number-base.Number > config.Observation.MaxAncestryHeaders || selected.Number-base.Number > uint64(limits.MaxRequests) {
		return empty, ErrReplayUnavailable
	}
	begin := base.Number
	if expected.Sequence != 0 {
		if begin == ^uint64(0) {
			return empty, ErrReplayUnavailable
		}
		begin++
	}
	if selected.Number < begin {
		return empty, ErrReplayUnavailable
	}
	batch := ReplayBatch{SelectedFinalized: selected}
	work := replayBudget{limits: config.Replay}
	// Admit each parent against the remaining work before growing its array.
	// Request count and JSON bytes are independently bounded by the private RPC.
	for height := selected.Number; height > base.Number; {
		height--
		header, err := reads.rpc.header(ctx, "0x"+strconv.FormatUint(height, 16))
		if err != nil {
			return empty, err
		}
		if !work.charge(len(header.Hash) + len(header.ParentHash) + 16) {
			return empty, ErrReplayUnavailable
		}
		batch.Parents = append(batch.Parents, header)
	}
	if err := checks.CheckLineage(base, selected, batch.Parents); err != nil {
		return empty, err
	}
	for _, contract := range identity.Contracts {
		code, err := reads.CodeBytes(ctx, contract.Address)
		if err != nil {
			return empty, err
		}
		if len(code) == 0 || nodeHex(sha256Digest(code)) != "0x"+contract.SHA256 {
			return empty, ErrRPCState
		}
	}
	stop := begin + min(uint64(config.Replay.MaxBlocks)-1, selected.Number-begin)
	for height := begin; ; height++ {
		observed, err := reads.ReadReplayBlockWithRootGas(ctx, height, limits.Node)
		if err != nil {
			return empty, err
		}
		want := selected
		if height < selected.Number {
			want = batch.Parents[selected.Number-height-1]
		}
		if observed.Header != want {
			return empty, ErrAncestry
		}
		// Reuse the batch-wide content/gas work bound; per-node counters cannot
		// reset the cumulative transaction, receipt or decoded-content budget.
		if _, err := replayDecodeBlock(ctx, observed.Block, &work); err != nil {
			return empty, err
		}
		batch.Blocks = append(batch.Blocks, observed.Block)
		batch.Anchor = observed.Header
		if height == stop {
			break
		}
	}
	pinned := &PinnedAgentReads{reads.rpc, reads.registry, reads.agent, batch.Anchor}
	present, record, err := pinned.RecordBytes(ctx)
	if err != nil {
		return empty, err
	}
	var grants []byte
	// CM-08 makes activeGrants unavailable for an absent identity. Do not
	// invent a successful [] RPC result or treat that refusal as evidence.
	// Only the journal's later complete replay can validate this empty state.
	if present {
		grants, err = pinned.GrantsBytes(ctx)
		if err != nil {
			return empty, err
		}
	}
	version, err := sourceRecordVersion(config, present, record, grants)
	if err != nil || version < expected.Version {
		return empty, ErrRPCState
	}
	if version != 0 {
		batch.Record, batch.Grants = record, grants
	}
	remaining := config.Replay.History.MaxBytes - len(batch.Record) - len(batch.Grants)
	if remaining < 0 || !work.charge(len(batch.Record)+len(batch.Grants)) {
		return empty, ErrReplayUnavailable
	}
	if version != expected.Version {
		batch.History, err = pinned.HistoryRange(ctx, expected.Version, version, HistoryReadLimits{config.Replay.History.MaxItems, remaining})
		if err != nil {
			return empty, err
		}
		for _, item := range batch.History {
			if !work.charge(len(item.Entry) + len(item.Command)) {
				return empty, ErrReplayUnavailable
			}
		}
	}
	// These are additional calls in the same aggregate scope. The stop and
	// selected headers must still be canonical; finality cannot regress/change
	// at the same height. Newer finality does not refresh the original selection.
	for _, want := range []FinalizedHeader{batch.Anchor, selected} {
		current, err := reads.rpc.header(ctx, "0x"+strconv.FormatUint(want.Number, 16))
		if err != nil {
			return empty, err
		}
		if current != want {
			return empty, ErrAncestry
		}
	}
	current, err := reads.rpc.PinAgentReads(ctx, reads.registry, reads.agent)
	if err != nil {
		return empty, err
	}
	if current.header.Number < selected.Number || current.header.Number == selected.Number && current.header != selected {
		return empty, ErrAncestry
	}
	return batch, ctx.Err()
}

func sourceRecordVersion(config JournalConfig, present bool, record, grants []byte) (uint64, error) {
	if !present {
		if len(record) != 0 || len(grants) != 0 {
			return 0, ErrRPCState
		}
		return 0, nil // An empty internal state, never an authenticated absence.
	}
	if len(record) == 0 || len(record) > 65536 || len(grants) == 0 || len(grants) > 262144 {
		return 0, ErrRPCState
	}
	m := contentObject(contentJSON(record), "id", "controller", "keys", "services", "state", "version")
	version := contentString(m["version"])
	n, err := strconv.ParseUint(version, 10, 64)
	if m == nil || err != nil || n == 0 || strconv.FormatUint(n, 10) != version || contentString(m["id"]) != "did:sage:"+config.Registry+":"+config.Agent || !contentGrants(contentJSON(grants)) {
		return 0, ErrRPCState
	}
	return n, nil
}
