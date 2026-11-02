package registry011

import (
	"encoding/binary"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
)

// Only the pinned commitClaim(bytes32)/ClaimCommitted schema is admitted.
// Its uint256 event height is encoded from the already-decoded uint64 header;
// the separate mutation-only selector/uint64 framing helpers do not apply.
// Native ABI initialization/map/allocator/getter costs remain unqualified.
func (r *replayRun) commitObserved(l replayLog, h *types.Header, event abi.Event) error {
	if r == nil || r.budget == nil || r.budget.observer == nil || r.claims == nil || l.frame == nil || l.log == nil || h == nil || h.Number == nil || !h.Number.IsUint64() || len(l.frame.Input) != 36 || len(l.log.Topics) != 3 || len(l.log.Data) != 32 {
		return ErrReplay
	}
	meter := r.budget.observer
	if !meter.event(providerCopyEvent, uint64(reflect.TypeFor[abi.Method]().Size())) {
		return ErrReplayUnavailable
	}
	method, present := r.contract.Methods["commitClaim"]
	if !present || len(method.ID) != 4 || len(method.Inputs) != 1 || method.Inputs[0].Type.T != abi.FixedBytesTy || method.Inputs[0].Type.Size != 32 || !providerReplayTextEqual(method.Name, "commitClaim", meter) || !providerReplayEqual(l.frame.Input[:4], method.ID, meter) {
		return ErrReplay
	}
	actor, indexed := replayAddressTopicObserved(l.frame.From, meter)
	if !indexed || !providerReplayEqual(l.log.Topics[1][:], actor[:], meter) || !meter.event(providerCopyEvent, 32) {
		return ErrReplay
	}
	digest := [32]byte(l.frame.Input[4:])
	if !providerReplayEqual(l.log.Topics[2][:], digest[:], meter) || !providerReplayTextEqual(event.Name, "ClaimCommitted", meter) {
		return ErrReplay
	}
	fields, selected := replayABINonIndexedObserved(event.Inputs, meter)
	if !selected || len(fields) != 1 || fields[0].Type.T != abi.UintTy || fields[0].Type.Size != 256 || !meter.event(providerCopyEvent, 32) {
		return ErrReplay
	}
	var expected [32]byte
	if !meter.event(providerEncodeEvent, 32) {
		return ErrReplayUnavailable
	}
	binary.BigEndian.PutUint64(expected[24:], h.Number.Uint64())
	if !providerReplayEqual(l.log.Data, expected[:], meter) || !meter.event(providerCopyEvent, uint64(len(l.frame.From))) || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[replayClaimID]().Size())) {
		return ErrReplay
	}
	id := replayClaimID{actor: strings.Clone(l.frame.From), digest: digest}
	if !meter.event(providerCompareEvent, uint64(len(id.actor)+len(id.digest))) {
		return ErrReplayUnavailable
	}
	if _, duplicate := r.claims[id]; duplicate {
		return ErrReplay
	}
	if r.claimLimit > 0 && len(r.claims) >= r.claimLimit {
		return ErrReplayUnavailable
	}
	if !meter.event(providerCopyEvent, uint64(reflect.TypeFor[replayClaimID]().Size()+reflect.TypeFor[replayClaimState]().Size())) {
		return ErrReplayUnavailable
	}
	r.claims[id] = replayClaimState{block: h.Number.Uint64()}
	return nil
}
