package registry011

import (
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// Tuple and array heads have bounded, explicitly initialized argument storage.
// Copied Type values retain the pinned ABI graph; they do not clone or accept
// a producer-owned type graph. Native backing-memory qualification is separate.
func replayABIDynamicObserved(data []byte, field abi.Argument, depth int, meter *providerWorkMeter) (int, bool) {
	if meter == nil || depth < 0 || depth > 32 || len(data) > 148804 || !meter.preflightWork(candidateWorkVector{}) {
		return 0, false
	}
	switch field.Type.T {
	case abi.TupleTy:
		count := len(field.Type.TupleElems)
		if count > 128 || count != len(field.Type.TupleRawNames) {
			return 0, false
		}
		fields, ok := replayABIArgumentStorageObserved(count, meter)
		if !ok {
			return 0, false
		}
		for i, typ := range field.Type.TupleElems {
			if typ == nil || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[abi.Argument]().Size())) {
				return 0, false
			}
			fields[i] = abi.Argument{Name: field.Type.TupleRawNames[i], Type: *typ}
		}
		return replayABIFieldsDepthObserved(data, fields, depth, meter)
	case abi.SliceTy:
		count, ok := replayABIWordObserved(data, meter)
		maximum := 0
		switch field.Name {
		case "keys":
			maximum = 128
		case "services":
			maximum = 16
		}
		if !ok || maximum == 0 || count > maximum || field.Type.Elem == nil || len(data) < 32+count*32 {
			return 0, false
		}
		fields, ok := replayABIArgumentStorageObserved(count, meter)
		if !ok {
			return 0, false
		}
		for i := range fields {
			if !meter.event(providerCopyEvent, uint64(reflect.TypeFor[abi.Argument]().Size())) {
				return 0, false
			}
			fields[i] = abi.Argument{Type: *field.Type.Elem}
		}
		n, ok := replayABIFieldsDepthObserved(data[32:], fields, depth, meter)
		if !ok {
			return 0, false
		}
		return 32 + n, true
	case abi.StringTy, abi.BytesTy:
		n, ok := replayABIWordObserved(data, meter)
		maximum := replayABIFieldMaximum(field.Name)
		if !ok || maximum == 0 || n > maximum {
			return 0, false
		}
		padded := (n + 31) / 32 * 32
		if len(data) < 32+padded || !replayABIZeroObserved(data[32+n:32+padded], meter) {
			return 0, false
		}
		if field.Type.T == abi.StringTy {
			for _, c := range data[32 : 32+n] {
				if !meter.event(providerCompareEvent, 1) || c > 127 {
					return 0, false
				}
			}
		}
		return 32 + padded, true
	}
	return 0, false
}

func replayABIArgumentStorageObserved(count int, meter *providerWorkMeter) (abi.Arguments, bool) {
	if meter == nil || count < 0 || count > 128 || !meter.event(providerCopyEvent, uint64(count)*uint64(reflect.TypeFor[abi.Argument]().Size())) {
		return nil, false
	}
	return make(abi.Arguments, count), true
}

func replayABIFieldMaximum(name string) int {
	switch name {
	case "agentId", "serviceType":
		return 64
	case "name":
		return 32
	case "alg":
		return 24
	case "key", "proof":
		return 65
	case "signer":
		return 189
	case "uri":
		return 2048
	case "scope":
		return 15
	case "did":
		return len("did:sage:eip155:11155111:"+SepoliaRegistryAddress+":") + 64
	case "operation":
		return 18
	}
	return 0
}
