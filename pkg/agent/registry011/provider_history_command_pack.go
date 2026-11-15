package registry011

import (
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func replayCommandABIKeysObserved(keys []replayKey, meter *providerWorkMeter) ([]replayABIKey, bool) {
	if meter == nil || len(keys) > 128 || !meter.event(providerCopyEvent, uint64(len(keys))*uint64(reflect.TypeFor[replayABIKey]().Size())) {
		return nil, false
	}
	out := make([]replayABIKey, len(keys))
	for i, key := range keys {
		raw, ok := replayCommandBinaryObserved(key.Key, 32, meter)
		if !ok {
			return nil, false
		}
		proof, ok := replayCommandBinaryObserved(key.Proof.Value, 64, meter)
		if !ok || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[replayABIKey]().Size())) {
			return nil, false
		}
		out[i] = replayABIKey{Name: key.Name, Alg: key.Alg, Key: raw, Signer: key.Proof.Signer, Proof: proof, HasExpiry: key.Expires != nil}
		if key.Expires != nil {
			if !meter.event(providerCopyEvent, 8) {
				return nil, false
			}
			out[i].Expires = *key.Expires
		}
	}
	return out, true
}

func replayCommandABIServicesObserved(services []replayService, meter *providerWorkMeter) ([]replayABIService, bool) {
	if meter == nil || len(services) > 16 || !meter.event(providerCopyEvent, uint64(len(services))*uint64(reflect.TypeFor[replayABIService]().Size())) {
		return nil, false
	}
	out := make([]replayABIService, len(services))
	for i, service := range services {
		if !meter.event(providerCopyEvent, uint64(reflect.TypeFor[replayABIService]().Size())) {
			return nil, false
		}
		out[i] = replayABIService{Name: service.Name, ServiceType: service.Type, URI: service.URI}
	}
	return out, true
}

func replayCommandParamsObserved(count int, meter *providerWorkMeter) ([]any, bool) {
	if meter == nil || count < 2 || count > 5 || !meter.event(providerCopyEvent, uint64(count)*uint64(reflect.TypeFor[any]().Size())) {
		return nil, false
	}
	return make([]any, count), true
}

func replayCommandParamObserved(params []any, index int, value any, meter *providerWorkMeter) bool {
	if meter == nil || index < 0 || index >= len(params) || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[any]().Size())) {
		return false
	}
	params[index] = value
	return true
}

// Values come only from the closed JSON/schema parser below. The immutable
// embedded ABI and existing field/count limits independently bound Pack's
// final output. This does not qualify its complete native allocation profile.
func replayCommandPackObserved(contract abi.ABI, method string, params []any, meter *providerWorkMeter) ([]byte, bool) {
	maximum := replayABICallMaximum(method)
	if meter == nil || maximum == 0 || len(params) < 2 || len(params) > 5 || !meter.preflight(providerEncodeEvent, uint64(maximum)) {
		return nil, false
	}
	raw, err := contract.Pack(method, params...)
	if err != nil || len(raw) > maximum {
		meter.refuse()
		return nil, false
	}
	if !meter.event(providerEncodeEvent, uint64(len(raw))) {
		return nil, false
	}
	return raw, true
}
