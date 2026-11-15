package registry011

import "strconv"

const providerHistoryCustodyIntentDomain = "sage-provider-history-custody-intent-v1"
const providerHistoryCustodyCommitDomain = "sage-provider-history-custody-commit-v1"
const providerHistoryCustodyFrontierDomain = "sage-provider-history-custody-frontier-v1"
const providerHistoryCustodyFrameBytes = 8 << 10

// A closed history frame is retained DATA only. A real writer must separately
// reconstruct its objects and both logical models before the existing physical
// HEAD CAS. Neither a parsed frame nor this scope grants an ACK or a request.
type providerHistoryCustodyScope struct {
	initial  providerFrontierHighwater
	exchange string
	origin   CandidateClockDiagnostic
}

type providerHistoryCustodyFrame struct {
	scope                        providerHistoryCustodyScope
	before, after                providerFrontierHighwater
	stage                        uint64
	phase                        string
	previousInventory, inventory string
	previousModel, model         string
	operation                    string
	clock                        CandidateClockDiagnostic
	intent, commit, frontier     string
}

func (*providerHistoryCustodyFrame) UnmarshalJSON([]byte) error { return errProviderIntegration }

// Preserve the existing policy clock-ID wire bound before any frame encode.
// Diagnostic equality alone does not establish an ASCII, bounded identifier.
func providerHistoryClockIdentifierValid(id string) bool {
	if len(id) == 0 || len(id) > 256 {
		return false
	}
	for i := range len(id) {
		if id[i] < 33 || id[i] > 126 {
			return false
		}
	}
	return true
}

func newProviderHistoryCustodyFrameObserved(config providerCustodyConfig, scope providerHistoryCustodyScope, before providerFrontierHighwater, stage uint64, phase, previousInventory, inventory, previousModel, model, operation string, clock CandidateClockDiagnostic, meter *providerWorkMeter) (*providerHistoryCustodyFrame, error) {
	b := config.bounds.history
	if b == nil || !b.valid(config.bounds) || !providerHistoryClockIdentifierValid(scope.origin.ClockID) || !lowerHash(scope.origin.Epoch) ||
		!scope.initial.validObserved(config, meter) || !before.validObserved(config, meter) ||
		scope.initial.sequence != 4 || scope.initial.identity != before.identity || scope.initial.fence != before.fence ||
		scope.initial.profile != before.profile || scope.initial.setup != before.setup || !lowerHash(scope.exchange) ||
		stage == 0 || stage > b.stages || !lowerHash(previousInventory) || !lowerHash(inventory) || previousInventory == inventory ||
		!lowerHash(previousModel) || !lowerHash(model) || previousModel == model || !lowerHash(operation) ||
		!candidateLedgerClockAdvance(scope.origin, clock) || uint64(clock.MonotonicMS-scope.origin.MonotonicMS) > b.durationMS {
		return nil, errProviderIntegration
	}
	var offset uint64
	switch phase {
	case "admit":
	case "consume":
		offset = 1
	case "settle":
		offset = 2
	default:
		return nil, errProviderIntegration
	}
	prior, a := providerUpperProduct(stage-1, 3)
	sequence, c := providerUpperSum(scope.initial.sequence, prior, offset)
	ordinal, d := providerUpperSum(scope.initial.ordinal, prior, offset)
	if !a || !c || !d || before.sequence != sequence || before.ordinal != ordinal ||
		stage == 1 && phase == "admit" && before != scope.initial ||
		before.sequence >= uint64(config.bounds.maximumCommits) || before.ordinal == ^uint64(0) {
		return nil, errProviderIntegration
	}
	initialRaw, err := scope.initial.bytesObserved(config, meter)
	beforeRaw, beforeErr := before.bytesObserved(config, meter)
	if err != nil || beforeErr != nil {
		return nil, errProviderIntegration
	}
	initial := providerReplayDigest("sage-provider-custody-highwater-v1", initialRaw, meter)
	previous := providerReplayDigest("sage-provider-custody-highwater-v1", beforeRaw, meter)
	if initial == "" || previous == "" {
		return nil, errProviderIntegration
	}
	// Inner T/C/S bind their PREVIOUS committed inventory. The new inventory
	// may include that entry; only this outer intent names its resulting root.
	// The outer intent/commit/frontier/highwater are separate metadata files,
	// excluded from that object inventory and charged independently by a writer.
	// No object below includes its own containing inventory or after-pin hash.
	maximum := min(uint64(providerHistoryCustodyFrameBytes), config.bounds.maximumMetadataPerCommit)
	intent := providerReplayCanonical(map[string]any{
		"format": providerHistoryCustodyIntentDomain, "program": b.program, "profile": config.profile, "cost_revision": b.costs,
		"initial_highwater": initial, "first_exchange": scope.exchange,
		"stage": strconv.FormatUint(stage, 10), "phase": phase, "sequence": strconv.FormatUint(sequence+1, 10),
		"previous_highwater": previous, "previous_commit": before.commit, "previous_frontier": before.frontier, "setup_head": before.setup,
		"previous_inventory": previousInventory, "inventory": inventory, "previous_model": previousModel, "model": model,
		"operation": operation, "clock": providerDiagnosticClockObject(clock),
	}, maximum, meter)
	if len(intent) == 0 {
		return nil, errProviderIntegration
	}
	intentDigest := providerReplayDigest(providerHistoryCustodyIntentDomain, intent, meter)
	commit := providerReplayCanonical(map[string]any{"format": providerHistoryCustodyCommitDomain, "intent": intentDigest}, maximum, meter)
	if intentDigest == "" || len(commit) == 0 {
		return nil, errProviderIntegration
	}
	commitDigest := providerReplayDigest(providerHistoryCustodyCommitDomain, commit, meter)
	frontier := providerReplayCanonical(map[string]any{
		"format": providerHistoryCustodyFrontierDomain, "program": b.program, "profile": config.profile, "cost_revision": b.costs,
		"initial_highwater": initial, "stage": strconv.FormatUint(stage, 10), "phase": phase, "sequence": strconv.FormatUint(sequence+1, 10),
		"previous_frontier": before.frontier, "commit": commitDigest, "inventory": inventory, "model": model,
		"clock": providerDiagnosticClockObject(clock),
	}, maximum, meter)
	frontierDigest := providerReplayDigest(providerHistoryCustodyFrontierDomain, frontier, meter)
	if commitDigest == "" || len(frontier) == 0 || frontierDigest == "" {
		return nil, errProviderIntegration
	}
	after := before
	after.previous, after.commit, after.frontier = previous, commitDigest, frontierDigest
	after.ordinal, after.sequence = before.ordinal+1, before.sequence+1
	if !checkProviderHighwaterPredecessorObserved(before, after, config, meter) {
		return nil, errProviderIntegration
	}
	f := &providerHistoryCustodyFrame{scope: scope, before: before, after: after, stage: stage, phase: phase,
		previousInventory: previousInventory, inventory: inventory, previousModel: previousModel, model: model, operation: operation, clock: clock}
	f.intent, f.commit, f.frontier = providerReplayString(intent, meter), providerReplayString(commit, meter), providerReplayString(frontier, meter)
	if f.intent == "" || f.commit == "" || f.frontier == "" {
		return nil, errProviderIntegration
	}
	return f, nil
}

// Independent callers provide their own selected scope and exact BEFORE pin.
// This verifies a framing relation only; object replay, cost admission and the
// concrete HEAD readback/CAS are mandatory outside this DATA parser.
func parseProviderHistoryCustodyFrameObserved(config providerCustodyConfig, scope providerHistoryCustodyScope, before, after providerFrontierHighwater, intent, commit, frontier []byte, meter *providerWorkMeter) (*providerHistoryCustodyFrame, error) {
	maximum := min(uint64(providerHistoryCustodyFrameBytes), config.bounds.maximumMetadataPerCommit)
	for _, raw := range [][]byte{intent, commit, frontier} {
		if len(raw) == 0 || uint64(len(raw)) > maximum {
			return nil, errProviderIntegration
		}
	}
	m := contentObject(providerReplayJSON(intent, meter), "format", "program", "profile", "cost_revision", "initial_highwater", "first_exchange", "stage", "phase", "sequence", "previous_highwater", "previous_commit", "previous_frontier", "setup_head", "previous_inventory", "inventory", "previous_model", "model", "operation", "clock")
	stage, ok := providerCandidateUint(m["stage"], true)
	clock, clockOK := providerStoredClock(m["clock"])
	if m == nil || !ok || !clockOK {
		return nil, errProviderIntegration
	}
	f, err := newProviderHistoryCustodyFrameObserved(config, scope, before, stage, contentString(m["phase"]), contentString(m["previous_inventory"]), contentString(m["inventory"]), contentString(m["previous_model"]), contentString(m["model"]), contentString(m["operation"]), clock, meter)
	if err != nil || f.after != after || !providerReplayEqual(providerReplayPhaseBytes(f.intent, meter), intent, meter) ||
		!providerReplayEqual(providerReplayPhaseBytes(f.commit, meter), commit, meter) || !providerReplayEqual(providerReplayPhaseBytes(f.frontier, meter), frontier, meter) {
		return nil, errProviderIntegration
	}
	return f, nil
}
