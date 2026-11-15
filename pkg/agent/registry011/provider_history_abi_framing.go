package registry011

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// These private DATA variants observe the same canonical framing as the
// existing reader. The ABI type graph is the embedded, separately pinned
// Registry ABI, never a schema supplied by an RPC response. This is not a
// native decoder/allocation profile or a complete command/Source certificate.
func replayCallFramingObserved(method abi.Method, data []byte, meter *providerWorkMeter) bool {
	if meter == nil {
		return replayCallFraming(method, data)
	}
	maximum := replayABICallMaximum(method.Name)
	if maximum == 0 || len(data) > maximum || len(method.Inputs) > 128 || len(data) < 4+32*len(method.Inputs) {
		return false
	}
	n, ok := replayABIFieldsObserved(data[4:], method.Inputs, meter)
	return ok && n == len(data)-4
}

func replayABICallMaximum(name string) int {
	switch name {
	case "revealClaim":
		return 148804
	case "activate", "deactivate":
		return 164
	case "addKey":
		return 1028
	case "revokeKey":
		return 260
	case "updateServices":
		return 38116
	case "authorizeOperator", "revokeOperator":
		return 292
	}
	return 0
}

func replayABIZeroObserved(raw []byte, meter *providerWorkMeter) bool {
	if meter == nil || len(raw) > 32 || !meter.event(providerCompareEvent, uint64(2*len(raw))) {
		return false
	}
	for _, c := range raw {
		if c != 0 {
			return false
		}
	}
	return true
}

func replayABIWordObserved(raw []byte, meter *providerWorkMeter) (int, bool) {
	if meter == nil {
		return replayABIWord(raw)
	}
	if len(raw) < 32 || !replayABIZeroObserved(raw[:24], meter) || !meter.event(providerCodecEvent, 8) {
		return 0, false
	}
	n := binary.BigEndian.Uint64(raw[24:32])
	if n > 148804 {
		return 0, false
	}
	return int(n), true
}

func replayABIFieldsObserved(data []byte, fields abi.Arguments, meter *providerWorkMeter) (int, bool) {
	if meter == nil {
		return replayABIFields(data, fields)
	}
	return replayABIFieldsDepthObserved(data, fields, 0, meter)
}

func replayABIFieldsDepthObserved(data []byte, fields abi.Arguments, depth int, meter *providerWorkMeter) (int, bool) {
	if meter == nil || depth < 0 || depth > 32 || len(fields) > 128 || len(data) > 148804 || !meter.preflightWork(candidateWorkVector{}) {
		return 0, false
	}
	head := 32 * len(fields)
	if len(data) < head {
		return 0, false
	}
	cursor := head
	for i, f := range fields {
		word := data[i*32 : i*32+32]
		switch f.Type.T {
		case abi.StringTy, abi.BytesTy, abi.TupleTy, abi.SliceTy:
			offset, ok := replayABIWordObserved(word, meter)
			if !ok || offset != cursor {
				return 0, false
			}
			n, ok := replayABIDynamicObserved(data[cursor:], f, depth+1, meter)
			if !ok || n > len(data)-cursor {
				return 0, false
			}
			cursor += n
		case abi.AddressTy:
			if !replayABIZeroObserved(word[:12], meter) {
				return 0, false
			}
		case abi.UintTy:
			if f.Type.Size != 64 || !replayABIZeroObserved(word[:24], meter) {
				return 0, false
			}
		case abi.BoolTy:
			if !replayABIZeroObserved(word[:31], meter) || !meter.event(providerCompareEvent, 1) || word[31] > 1 {
				return 0, false
			}
		case abi.FixedBytesTy:
			if f.Type.Size != 32 {
				return 0, false
			}
		default:
			return 0, false
		}
	}
	return cursor, meter.preflightWork(candidateWorkVector{})
}
