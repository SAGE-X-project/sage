package registry011

import (
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// Only the eight already adopted mutation methods can yield a command. The
// fixed order makes the actual selector comparisons reproducible without a
// map iteration or a sender-controlled ABI. Nil preserves the legacy lookup.
func replayABIMethodObserved(contract abi.ABI, data []byte, meter *providerWorkMeter) (*abi.Method, bool) {
	if len(data) < 4 {
		return nil, false
	}
	if meter == nil {
		method, err := contract.MethodById(data[:4])
		return method, err == nil
	}
	for _, name := range [...]string{"revealClaim", "activate", "deactivate", "addKey", "revokeKey", "updateServices", "authorizeOperator", "revokeOperator"} {
		if !meter.event(providerCopyEvent, uint64(reflect.TypeFor[abi.Method]().Size())) {
			return nil, false
		}
		method, present := contract.Methods[name]
		if present && providerReplayEqual(method.ID, data[:4], meter) {
			return &method, true
		}
		if !meter.preflightWork(candidateWorkVector{}) {
			return nil, false
		}
	}
	return nil, false
}

// Callers must first pass the pinned ABI framing under this same meter. The
// admitted offsets/counts bound Unpack's allocations; re-packing the unchanged
// values cannot exceed those original bytes. A bound is never recorded as the
// actual encoder output, and no partial decoded values escape after refusal.
func replayABICanonicalObserved(fields abi.Arguments, data []byte, meter *providerWorkMeter) ([]any, error) {
	if meter != nil && (len(data) > 148804 || len(fields) > 128 || !meter.event(providerCodecEvent, uint64(len(data)))) {
		return nil, ErrReplayUnavailable
	}
	values, err := fields.Unpack(data)
	if err != nil {
		return nil, ErrReplay
	}
	if meter != nil && !meter.preflight(providerEncodeEvent, uint64(len(data))) {
		return nil, ErrReplayUnavailable
	}
	packed, err := fields.Pack(values...)
	if meter != nil {
		if len(packed) > len(data) {
			meter.refuse()
			return nil, ErrReplay
		}
		if !meter.event(providerEncodeEvent, uint64(len(packed))) {
			return nil, ErrReplayUnavailable
		}
	}
	if err != nil || !providerReplayEqual(packed, data, meter) {
		return nil, ErrReplay
	}
	return values, nil
}

func replayABINonIndexedObserved(fields abi.Arguments, meter *providerWorkMeter) (abi.Arguments, bool) {
	if meter == nil {
		return fields.NonIndexed(), true
	}
	if len(fields) > 128 {
		return nil, false
	}
	count := 0
	for _, field := range fields {
		if !field.Indexed {
			count++
		}
	}
	out, ok := replayABIArgumentStorageObserved(count, meter)
	if !ok {
		return nil, false
	}
	i := 0
	for _, field := range fields {
		if !field.Indexed {
			if !meter.event(providerCopyEvent, uint64(reflect.TypeFor[abi.Argument]().Size())) {
				return nil, false
			}
			out[i] = field
			i++
		}
	}
	return out, true
}
