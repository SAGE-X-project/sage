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
	contractRaw := providerReplayBytes(replayContractABI, meter)
	if len(contractRaw) != len(replayContractABI) || !meter.event(providerCodecEvent, uint64(len(contractRaw))) {
		return abi.ABI{}, abi.ABI{}, ErrReplayUnavailable
	}
	contract, err := abi.JSON(bytes.NewReader(contractRaw))
	if err != nil {
		return abi.ABI{}, abi.ABI{}, err
	}
	outcomesRaw := providerReplayBytes(replayOutcomesABI, meter)
	if len(outcomesRaw) != len(replayOutcomesABI) || !meter.event(providerCodecEvent, uint64(len(outcomesRaw))) {
		return abi.ABI{}, abi.ABI{}, ErrReplayUnavailable
	}
	outcomes, err := abi.JSON(bytes.NewReader(outcomesRaw))
	if err != nil || !meter.preflightWork(candidateWorkVector{}) {
		return abi.ABI{}, abi.ABI{}, ErrReplayUnavailable
	}
	return contract, outcomes, nil
}
