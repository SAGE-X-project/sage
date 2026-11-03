package registry011

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// Parse only the same two fixed embedded ABI definitions as the existing API.
// Each is one observed native codec invocation on its actual immutable input,
// including internal type/method/event construction. Logical codec weights are
// not measured CPU instructions or an allocator/RAM certificate. The complete
// program must still qualify the pinned native implementation and live set.
func replayABIsObserved(meter *providerWorkMeter) (abi.ABI, abi.ABI, error) {
	if meter == nil {
		return replayABIs()
	}
	contract, err := replayABIObserved(replayContractABI, meter)
	if err != nil {
		return abi.ABI{}, abi.ABI{}, err
	}
	outcomes, err := replayABIObserved(replayOutcomesABI, meter)
	if err != nil || !meter.preflightWork(candidateWorkVector{}) {
		return abi.ABI{}, abi.ABI{}, ErrReplayUnavailable
	}
	return contract, outcomes, nil
}

// Both fixed definitions pass through the same observed copy/parse boundary.
// Keeping native parsing here also permits small invalid-ABI UNIT cases
// without modifying either embedded definition or adding a live input route.
func replayABIObserved(raw string, meter *providerWorkMeter) (abi.ABI, error) {
	if meter == nil {
		return abi.ABI{}, ErrReplayUnavailable
	}
	owned := providerReplayBytes(raw, meter)
	if len(owned) != len(raw) || !meter.event(providerCodecEvent, uint64(len(owned))) {
		return abi.ABI{}, ErrReplayUnavailable
	}
	return abi.JSON(bytes.NewReader(owned))
}
