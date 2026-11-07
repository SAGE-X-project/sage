package registry011

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
)

func replayCommandBinaryObserved(value any, size int, meter *providerWorkMeter) ([]byte, bool) {
	s, ok := value.(string)
	if meter == nil || !ok || size < 1 || size > 65 || len(s) != (size*8+5)/6 || !meter.event(providerCodecEvent, uint64(len(s))) {
		return nil, false
	}
	raw, err := base64.RawURLEncoding.Strict().DecodeString(s)
	if err != nil || len(raw) != size || !meter.preflight(providerEncodeEvent, uint64(len(s))) {
		return nil, false
	}
	canonical := base64.RawURLEncoding.EncodeToString(raw)
	if !meter.event(providerEncodeEvent, uint64(len(canonical))) || !providerReplayTextEqual(canonical, s, meter) {
		return nil, false
	}
	return raw, true
}

func replayCommandNameObserved(value string, meter *providerWorkMeter) bool {
	return meter != nil && len(value) <= 32 && meter.event(providerCompareEvent, uint64(len(value))) && replayName(value)
}

func replayCommandURIObserved(value string, meter *providerWorkMeter) bool {
	// The existing closed HTTPS/IP/percent grammar is a bounded parser
	// invocation. Its native CPU/allocator profile is not certified here.
	return meter != nil && len(value) <= 2048 && meter.event(providerCodecEvent, uint64(len(value))) && replayURI(value)
}

func replayCommandUintObserved(value string, meter *providerWorkMeter) (uint64, bool) {
	if meter == nil || len(value) == 0 || len(value) > 20 || !meter.event(providerCodecEvent, uint64(len(value))) {
		return 0, false
	}
	n, err := strconv.ParseUint(value, 10, 64)
	if err != nil || !meter.preflight(providerEncodeEvent, 20) {
		return 0, false
	}
	canonical := strconv.FormatUint(n, 10)
	return n, meter.event(providerEncodeEvent, uint64(len(canonical))) && providerReplayTextEqual(value, canonical, meter)
}

func replayCommandSecondsObserved(value any, meter *providerWorkMeter) (uint64, bool) {
	v, ok := value.(json.Number)
	if !ok {
		return 0, false
	}
	n, ok := replayCommandUintObserved(string(v), meter)
	return n, ok && n <= 1<<53-1
}

func replayCommandAddressObserved(value string, meter *providerWorkMeter) (common.Address, bool) {
	if meter == nil || !replayContentAddressObserved(value, meter) || !meter.event(providerCodecEvent, 40) {
		return common.Address{}, false
	}
	raw, err := hex.DecodeString(value[2:])
	if err != nil || !meter.event(providerCopyEvent, 20) {
		return common.Address{}, false
	}
	return common.BytesToAddress(raw), true
}

func replayCommandArgumentsObserved(operation string, value any, meter *providerWorkMeter) (map[string]any, bool) {
	if meter == nil {
		return nil, false
	}
	var names []string
	found := false
	for _, name := range [...]string{"create", "activate", "deactivate", "add-key", "revoke-key", "update-services", "authorize-operator", "revoke-operator"} {
		if providerReplayTextEqual(operation, name, meter) {
			found = true
			break
		}
	}
	if !found || !meter.preflightWork(candidateWorkVector{}) {
		return nil, false
	}
	switch operation {
	case "create":
		names = []string{"controller", "keys", "services", "salt"}
	case "add-key":
		names = []string{"key"}
	case "revoke-key":
		names = []string{"name"}
	case "update-services":
		names = []string{"services"}
	case "authorize-operator", "revoke-operator":
		names = []string{"operator", "scope"}
	}
	args := replayCommandObjectObserved(value, names, meter)
	return args, args != nil
}

func replayCommandGrantObserved(args map[string]any, meter *providerWorkMeter) bool {
	if _, ok := replayCommandAddressObserved(contentString(args["operator"]), meter); !ok {
		return false
	}
	for _, scope := range [...]string{"activate", "add-key", "revoke-key", "update-services", "deactivate"} {
		if providerReplayTextEqual(contentString(args["scope"]), scope, meter) {
			return true
		}
	}
	return false
}
