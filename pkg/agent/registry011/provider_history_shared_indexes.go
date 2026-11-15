package registry011

import (
	"reflect"

	"github.com/ethereum/go-ethereum/core/types"
)

// These are explicit logical map-key/value observations. Go map hashing,
// buckets, growth and allocator/backing RAM still need native qualification.
// Lifetime tombstones are never pruned at a window boundary.
func replaySharedIndexCloneObserved(s replayStream, maximumClaims, maximumIdentities int, meter *providerWorkMeter) (replayStream, bool) {
	if meter == nil || maximumClaims < 1 || maximumClaims > 1<<20 || maximumIdentities < 1 || maximumIdentities > 1<<20 || len(s.claims) > maximumClaims || len(s.created) > maximumIdentities || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[replayStream]().Size())) {
		return replayStream{}, false
	}
	state, ok := replayStateCloneObserved(s.state, meter)
	if !ok {
		return replayStream{}, false
	}
	// Match the existing semantic-window clone's empty-array normalization.
	if state.record.Keys == nil {
		state.record.Keys = []replayKey{}
	}
	if state.record.Services == nil {
		state.record.Services = []replayService{}
	}
	if state.grants == nil {
		state.grants = []replayGrant{}
	}
	claimSlot := uint64(reflect.TypeFor[replayClaimID]().Size() + reflect.TypeFor[replayClaimState]().Size())
	createdSlot := uint64(reflect.TypeFor[string]().Size() + reflect.TypeFor[bool]().Size())
	if !meter.event(providerCopyEvent, uint64(len(s.claims))*claimSlot+uint64(len(s.created))*createdSlot) {
		return replayStream{}, false
	}
	out := replayStream{state: state, selected: s.selected, claims: make(map[replayClaimID]replayClaimState, len(s.claims)), created: make(map[string]bool, len(s.created))}
	for key, value := range s.claims {
		actor, ok := replayStateTextObserved(key.actor, 42, meter)
		if !ok || !meter.event(providerCompareEvent, uint64(len(actor)+len(key.digest))) || !meter.event(providerCopyEvent, claimSlot) {
			return replayStream{}, false
		}
		out.claims[replayClaimID{actor: actor, digest: key.digest}] = value
	}
	for agent, created := range s.created {
		owned, ok := replayStateTextObserved(agent, 64, meter)
		if !ok || !meter.event(providerCompareEvent, uint64(len(owned))) || !meter.event(providerCopyEvent, createdSlot) {
			return replayStream{}, false
		}
		out.created[owned] = created
	}
	return out, meter.preflightWork(candidateWorkVector{})
}

// The caller has already derived this commitment from its actual command.
// Consuming another subject's create still updates the same global indexes.
func (r *replayRun) consumeCreateObserved(c replayCommand, digest [32]byte, h *types.Header, meter *providerWorkMeter) error {
	if r == nil || meter == nil || r.budget == nil || r.budget.observer != meter || r.claims == nil || r.created == nil || h == nil || h.Number == nil || !h.Number.IsUint64() || len(r.claims) > 1<<20 || len(r.created) > 1<<20 {
		return ErrReplayUnavailable
	}
	actor, a := replayStateTextObserved(c.actor, 42, meter)
	agent, b := replayStateTextObserved(c.agent, 64, meter)
	if !a || !b || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[replayClaimID]().Size())) || !meter.event(providerCompareEvent, uint64(len(actor)+len(digest))) || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[replayClaimState]().Size())) {
		return ErrReplayUnavailable
	}
	id := replayClaimID{actor: actor, digest: digest}
	claim, present := r.claims[id]
	if !meter.event(providerCompareEvent, uint64(len(agent))) {
		return ErrReplayUnavailable
	}
	if r.created[agent] || !present || claim.consumed || h.Number.Uint64() <= claim.block || h.Number.Uint64()-claim.block > 256 {
		return ErrReplay
	}
	if r.createdLimit > 0 && len(r.created) >= r.createdLimit {
		return ErrReplayUnavailable
	}
	// Fund both writes before changing either index. This is not rollback of
	// a later apply/proof/digest failure; that entire window must be discarded.
	bytes := uint64(reflect.TypeFor[replayClaimID]().Size() + reflect.TypeFor[replayClaimState]().Size() + reflect.TypeFor[string]().Size() + reflect.TypeFor[bool]().Size())
	if !meter.event(providerCopyEvent, bytes) {
		return ErrReplayUnavailable
	}
	claim.consumed = true
	r.claims[id], r.created[agent] = claim, true
	return nil
}

func (r *replayRun) appendStepObserved(step HistoryContentStep, meter *providerWorkMeter) bool {
	if r == nil || meter == nil || r.budget == nil || r.budget.observer != meter || len(r.steps) >= 4096 || !meter.event(providerCopyEvent, uint64(len(r.steps)+1)*uint64(reflect.TypeFor[HistoryContentStep]().Size())) {
		return false
	}
	out := make([]HistoryContentStep, len(r.steps)+1)
	for i, previous := range r.steps {
		if !meter.event(providerCopyEvent, uint64(reflect.TypeFor[HistoryContentStep]().Size())) {
			return false
		}
		out[i] = previous
	}
	if !meter.event(providerCopyEvent, uint64(reflect.TypeFor[HistoryContentStep]().Size())) {
		return false
	}
	out[len(r.steps)] = step
	r.steps = out
	return true
}

// Queue strings own exact bytes, including embedded JSON punctuation. They
// are not state/schema strings and must not inherit a 2048-byte ASCII bound.
func replayExpectedTextObserved(raw []byte, meter *providerWorkMeter) (string, bool) {
	if meter == nil || len(raw) == 0 || len(raw) > 262144 || !meter.event(providerCopyEvent, uint64(len(raw))) {
		return "", false
	}
	return string(raw), true
}
