package registry011

import (
	"bytes"
	"encoding/binary"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// CM-02 preflight BEFORE generic ABI decoding/allocation. Only the pinned
// method shapes are supported, with canonical packed tails, scalar/count caps,
// aligned offsets, exact zero padding and no overlapping/trailing regions.
// These are framing/scalar checks; targets and crypto follow separately.
func replayCallFraming(method abi.Method, data []byte) bool {
	maximum := map[string]int{"revealClaim": 148804, "activate": 164, "deactivate": 164, "addKey": 1028, "revokeKey": 260, "updateServices": 38116, "authorizeOperator": 292, "revokeOperator": 292}[method.Name]
	if maximum == 0 || len(data) > maximum || len(data) < 4+32*len(method.Inputs) {
		return false
	}
	n, ok := replayABIFields(data[4:], method.Inputs)
	return ok && n == len(data)-4
}
func replayABIWord(raw []byte) (int, bool) {
	if len(raw) < 32 || !bytes.Equal(raw[:24], make([]byte, 24)) {
		return 0, false
	}
	n := binary.BigEndian.Uint64(raw[24:32])
	return int(n), n <= 148804
}
func replayABIFields(data []byte, fields abi.Arguments) (int, bool) {
	head := 32 * len(fields)
	if len(data) < head {
		return 0, false
	}
	cursor := head
	for i, f := range fields {
		word := data[i*32 : i*32+32]
		if f.Type.T == abi.StringTy || f.Type.T == abi.BytesTy || f.Type.T == abi.TupleTy || f.Type.T == abi.SliceTy {
			offset, ok := replayABIWord(word)
			if !ok || offset != cursor {
				return 0, false
			}
			n, ok := replayABIDynamic(data[cursor:], f)
			if !ok {
				return 0, false
			}
			cursor += n
		} else {
			switch f.Type.T {
			case abi.AddressTy:
				if !bytes.Equal(word[:12], make([]byte, 12)) {
					return 0, false
				}
			case abi.UintTy:
				if f.Type.Size != 64 || !bytes.Equal(word[:24], make([]byte, 24)) {
					return 0, false
				}
			case abi.BoolTy:
				if !bytes.Equal(word[:31], make([]byte, 31)) || word[31] > 1 {
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
	}
	return cursor, true
}
func replayABIDynamic(data []byte, f abi.Argument) (int, bool) {
	switch f.Type.T {
	case abi.TupleTy:
		fields := make(abi.Arguments, len(f.Type.TupleElems))
		for i, typ := range f.Type.TupleElems {
			fields[i] = abi.Argument{Name: f.Type.TupleRawNames[i], Type: *typ}
		}
		return replayABIFields(data, fields)
	case abi.SliceTy:
		count, ok := replayABIWord(data)
		maximum := 0
		if f.Name == "keys" {
			maximum = 128
		}
		if f.Name == "services" {
			maximum = 16
		}
		if !ok || maximum == 0 || count > maximum || len(data) < 32+count*32 {
			return 0, false
		}
		fields := make(abi.Arguments, count)
		for i := range fields {
			fields[i] = abi.Argument{Type: *f.Type.Elem}
		}
		n, ok := replayABIFields(data[32:], fields)
		return 32 + n, ok
	case abi.StringTy, abi.BytesTy:
		n, ok := replayABIWord(data)
		maximum := map[string]int{"agentId": 64, "name": 32, "alg": 24, "key": 65, "signer": 189, "proof": 65, "serviceType": 64, "uri": 2048, "scope": 15, "did": len("did:sage:"+"eip155:11155111:"+SepoliaRegistryAddress+":") + 64, "operation": 18}[f.Name]
		if !ok || maximum == 0 || n > maximum {
			return 0, false
		}
		padded := (n + 31) / 32 * 32
		if len(data) < 32+padded || !bytes.Equal(data[32+n:32+padded], make([]byte, padded-n)) {
			return 0, false
		}
		if f.Type.T == abi.StringTy {
			for _, c := range data[32 : 32+n] {
				if c > 127 {
					return 0, false
				}
			}
		}
		return 32 + padded, true
	}
	return 0, false
}
