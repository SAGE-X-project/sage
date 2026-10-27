package registry011

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// Concrete primitive observation for independent typed replay. A nil meter
// preserves existing pure DATA callers. Neither form creates an inspection
// owner, admission, retained acknowledgment or execution capability. A future
// owned scanner must bind one nonnil meter before calling these variants.
func providerReplayJSON(raw []byte, meter *providerWorkMeter) any {
	if meter != nil && !meter.event(providerCodecEvent, uint64(len(raw))) {
		return nil
	}
	return contentJSON(raw)
}

func providerReplayDigest(domain string, raw []byte, meter *providerWorkMeter) string {
	n := uint64(len(raw))
	if domain != "" {
		var ok bool
		n, ok = providerUpperSum(n, uint64(len(domain)), 1)
		if !ok {
			return ""
		}
	}
	if meter != nil && !meter.event(providerHashEvent, n) {
		return ""
	}
	if domain != "" {
		return providerCandidateHash(domain, raw)
	}
	sum := sha256.Sum256(raw)
	return "0x" + hex.EncodeToString(sum[:])
}

func providerReplayString(raw []byte, meter *providerWorkMeter) string {
	if meter != nil && !meter.event(providerCopyEvent, uint64(len(raw))) {
		return ""
	}
	return string(raw)
}

// This preserves the standard JSON encoder used by the existing closed
// ASCII-only limits/segment schemas. It does not replace their byte contract
// with the separate canonical metadata encoder. The upper is never actual.
func providerReplayMarshal(value any, maximum uint64, meter *providerWorkMeter) []byte {
	// Policy parsers already accept at most64KiB. That existing policy-schema
	// bound is distinct from the smaller per-ledger-node metadata ceiling.
	if maximum == 0 || maximum > 64<<10 || meter != nil && !meter.preflight(providerEncodeEvent, maximum) {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil || uint64(len(raw)) > maximum || meter != nil && !meter.event(providerEncodeEvent, uint64(len(raw))) {
		return nil
	}
	return raw
}

func providerReplayCanonical(value any, maximum uint64, meter *providerWorkMeter) []byte {
	// The existing closed page encoder permits4MiB. Store metadata callers
	// still supply their original1MiB or smaller record limit.
	if maximum == 0 || maximum > 4<<20 || meter != nil && !meter.preflight(providerEncodeEvent, maximum) {
		return nil
	}
	raw := providerIntegrationJSON(value)
	if len(raw) == 0 || uint64(len(raw)) > maximum || meter != nil && !meter.event(providerEncodeEvent, uint64(len(raw))) {
		return nil
	}
	return raw
}

func providerReplayEqual(a, b []byte, meter *providerWorkMeter) bool {
	return (meter == nil || meter.event(providerCompareEvent, uint64(len(a))+uint64(len(b)))) && bytes.Equal(a, b)
}

// Preserve the existing composite byte ordering, including prefix kinds such
// as pin/pin-input. Every actual temporary key copy is funded before creation.
func providerReplayRefKey(ref providerIntegrationObjectRef, meter *providerWorkMeter) (string, bool) {
	if meter != nil && !meter.event(providerCopyEvent, uint64(len(ref.kind)+1+len(ref.digest))) {
		return "", false
	}
	return providerIntegrationRefKey(ref), true
}

func providerReplayJoin(parts []string, meter *providerWorkMeter) string {
	var total uint64
	for _, part := range parts {
		var ok bool
		total, ok = providerUpperSum(total, uint64(len(part)))
		if !ok {
			return ""
		}
	}
	if meter != nil && !meter.event(providerCopyEvent, total) {
		return ""
	}
	return strings.Join(parts, "")
}
