package registry011

import (
	"math/big"
	"reflect"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// This preserves the pinned existing gas-floor rule. It neither guesses a
// fork nor provides a missing execution witness. The separate original gas
// witness and strict settlement equations remain mandatory.
func replayGasFloorObserved(tx *types.Transaction, meter *providerWorkMeter) (uint64, error) {
	if tx == nil {
		return 0, ErrReplayUnavailable
	}
	if meter == nil {
		return replayGasFloor(tx)
	}
	from, err := nodeSenderObserved(tx, meter)
	if err != nil {
		return 0, err
	}
	return replayGasFloorDerivedObserved(tx, from, meter)
}

// Private arithmetic DATA after the enclosing reader recovered from. Direct
// UNIT inputs can exercise the fixed rule; they provide no sender credential.
// The actual reader always obtains from through nodeSenderObserved above.
func replayGasFloorDerivedObserved(tx *types.Transaction, from common.Address, meter *providerWorkMeter) (uint64, error) {
	if tx == nil || meter == nil || !meter.preflightWork(candidateWorkVector{}) {
		return 0, ErrReplayUnavailable
	}
	to, ok := nodeTransactionToObserved(tx, meter)
	if !ok {
		return 0, ErrReplayUnavailable
	}
	base := uint64(12000)
	if to == nil {
		base += 12000
	} else {
		if !meter.event(providerCompareEvent, 40) {
			return 0, ErrReplayUnavailable
		}
		if *to != from {
			base += 3000
			value, valid := nodeTransactionValueObserved(tx, meter)
			if !valid {
				return 0, ErrReplayUnavailable
			}
			if value.Sign() != 0 {
				base += 6000
			}
		}
	}
	bytes := uint64(len(tx.Data()))
	if bytes > 64<<20 {
		return 0, ErrReplayUnavailable
	}
	for _, entry := range tx.AccessList() {
		if !meter.event(providerCompareEvent, 0) || !candidateCapacity(0, bytes, 20+32*uint64(len(entry.StorageKeys)), 64<<20) {
			return 0, ErrReplayUnavailable
		}
		bytes += 20 + 32*uint64(len(entry.StorageKeys))
	}
	return base + 64*bytes, nil
}

func nodeTransactionToObserved(tx *types.Transaction, meter *providerWorkMeter) (*common.Address, bool) {
	if tx == nil || meter == nil || !meter.preflight(providerCopyEvent, 20) {
		return nil, false
	}
	to := tx.To()
	if to != nil && !meter.event(providerCopyEvent, 20) {
		return nil, false
	}
	return to, true
}

func nodeTransactionValueObserved(tx *types.Transaction, meter *providerWorkMeter) (*big.Int, bool) {
	// The private bound follows the actual original input route. Round its
	// magnitude-byte ceiling to nat words before the native copied accessor.
	word := uint64(strconv.IntSize / 8)
	structure := uint64(reflect.TypeFor[big.Int]().Size())
	if tx == nil || meter == nil {
		return nil, false
	}
	bound, eligible := providerHistoryNativeInputBoundObserved(meter)
	if !eligible || !meter.preflight(providerCopyEvent, bound.value) {
		return nil, false
	}
	value := tx.Value()
	n := structure + uint64(len(value.Bits()))*word
	if n > bound.value || !meter.event(providerCopyEvent, n) {
		meter.refuse()
		return nil, false
	}
	return value, true
}
