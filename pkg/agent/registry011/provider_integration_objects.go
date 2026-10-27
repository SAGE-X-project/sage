package registry011

import (
	"errors"
	"sort"
	"strconv"

	"github.com/sage-x-project/sage/pkg/agent/internal/rfc8785"
)

var errProviderIntegration = errors.New("registry011: connected provider integration unavailable")

// Only closed owned metadata uses this encoder. Original RPC bytes are never
// rewritten. Failure returns nil, which every object/schema boundary refuses.
func providerIntegrationJSON(value any) []byte {
	raw, err := rfc8785.Marshal(value)
	if err != nil {
		return nil
	}
	return raw
}

// All types in this file are private integration data. An object, inventory or
// matching digest is not a selected store, custody ACK or dispatch permission.
// The connected store must validate the entire retained closure and transition.
type providerIntegrationObject struct {
	kind, digest, raw string
}

type providerIntegrationObjectRef struct {
	kind, digest string
	bytes        uint64
}

type providerIntegrationInventory struct {
	raw, digest, previous string
	entries               []providerIntegrationObjectRef
	payloadBytes          uint64
}

func (*providerIntegrationInventory) UnmarshalJSON([]byte) error { return errProviderIntegration }

func providerIntegrationDomain(kind string) (string, bool) {
	switch kind {
	case "request", "params", "response", "tls":
		return "", true // Ordinary content SHA-256; kind stays bound by inventory.
	case "setup":
		return "sage-provider-authority-setup-v1", true
	case "setup-commit":
		return "sage-provider-authority-setup-commit-v1", true
	case "guardian-origin":
		return providerGuardianOriginDomain, true
	case "guardian-admission-binding":
		return "sage-provider-guardian-admission-binding-v1", true
	case "scan-admission-binding-v5":
		return "sage-provider-scan-admission-binding-v2", true
	case "enrollment":
		return "sage-provider-enrollment-v2", true
	case "admission":
		return "sage-provider-admission-intent-v1", true
	case "scan-admission-binding":
		return providerScanAdmissionBindingDomain, true
	case "ticket":
		return "sage-provider-budget-ticket-v2", true
	case "consumption":
		return "sage-provider-budget-consumption-v1", true
	case "settlement":
		return "sage-provider-budget-settlement-v2", true
	case "ledger-node":
		return "sage-provider-ledger-node-v1", true
	case "budget":
		return "sage-provider-budget-root-v2", true
	case "exchange":
		return "sage-provider-exchange-v2", true
	case "wire-binding":
		return "sage-provider-wire-binding-v1", true
	case "bootstrap":
		return "sage-provider-start-v2", true
	case "pin-input":
		return "sage-provider-owned-pin-input-v1", true
	case "pin":
		return "sage-provider-owned-pin-v1", true
	case "ledger-page":
		return "sage-provider-ledger-page-v1", true
	case "ledger-seal":
		return "sage-provider-ledger-seal-v1", true
	case "index-empty":
		return "sage-provider-index-empty-v1", true
	case "index-mutation":
		return "sage-provider-index-mutation-v1", true
	case "verification-receipt":
		return "sage-provider-verification-receipt-v1", true
	case "frontier":
		return "sage-provider-custody-frontier-v1", true
	case "segment":
		return "sage-provider-segment-v2", true
	case "batch":
		return "sage-provider-batch-v2", true
	case "checkpoint":
		return "sage-provider-checkpoint-v2", true
	}
	return "", false
}

func providerIntegrationObjectDigest(kind string, raw []byte) (string, bool) {
	return providerIntegrationObjectDigestObserved(kind, raw, nil)
}

func providerIntegrationObjectDigestObserved(kind string, raw []byte, meter *providerWorkMeter) (string, bool) {
	domain, ok := providerIntegrationDomain(kind)
	if !ok {
		return "", false
	}
	digest := providerReplayDigest(domain, raw, meter)
	return digest, digest != ""
}

// A format check here is deliberately weaker than semantic validation. Only
// the connected transition builder may introduce these objects into custody.
func newProviderIntegrationObject(kind string, raw []byte, maximum uint64) (providerIntegrationObject, error) {
	return newProviderIntegrationObjectObserved(kind, raw, maximum, nil)
}

func newProviderIntegrationObjectObserved(kind string, raw []byte, maximum uint64, meter *providerWorkMeter) (providerIntegrationObject, error) {
	if maximum == 0 || maximum > 32<<20 || len(raw) == 0 || uint64(len(raw)) > maximum {
		return providerIntegrationObject{}, errProviderIntegration
	}
	if kind == "request" && len(raw) > 16<<10 || kind == "params" && len(raw) > 8<<10 {
		return providerIntegrationObject{}, errProviderIntegration
	}
	domain, ok := providerIntegrationDomain(kind)
	if !ok {
		return providerIntegrationObject{}, errProviderIntegration
	}
	if domain != "" {
		m, ok := providerReplayJSON(raw, meter).(map[string]any)
		if !ok || m["format"] != domain {
			return providerIntegrationObject{}, errProviderIntegration
		}
	}
	digest, valid := providerIntegrationObjectDigestObserved(kind, raw, meter)
	retained := providerReplayString(raw, meter)
	if !valid || retained == "" {
		return providerIntegrationObject{}, errProviderIntegration
	}
	return providerIntegrationObject{kind, digest, retained}, nil
}

// Each node adds only new object references to an exact retained predecessor.
// A root alone proves neither presence nor absence. The store must retain and
// validate every predecessor node and referenced immutable object. Empty nodes
// are permitted only for the explicit initial inventory anchor.
func newProviderIntegrationInventory(previous string, objects []providerIntegrationObject, maximumEntries int, maximumPayload uint64) (*providerIntegrationInventory, error) {
	return newProviderInventory(1, previous, objects, maximumEntries, maximumPayload)
}

func newProviderFrontierInventory(previous string, objects []providerIntegrationObject, maximumEntries int, maximumPayload uint64) (*providerIntegrationInventory, error) {
	return newProviderInventory(2, previous, objects, maximumEntries, maximumPayload)
}

func newProviderFrontierInventoryObserved(previous string, objects []providerIntegrationObject, maximumEntries int, maximumPayload uint64, meter *providerWorkMeter) (*providerIntegrationInventory, error) {
	return newProviderInventoryObserved(2, previous, objects, maximumEntries, maximumPayload, meter)
}

func providerInventoryDomain(version int) (string, int, bool) {
	switch version {
	case 1:
		return "sage-provider-custody-objects-v1", 64, true
	case 2:
		return "sage-provider-custody-objects-v2", 256, true
	}
	return "", 0, false
}

func newProviderInventory(version int, previous string, objects []providerIntegrationObject, maximumEntries int, maximumPayload uint64) (*providerIntegrationInventory, error) {
	return newProviderInventoryObserved(version, previous, objects, maximumEntries, maximumPayload, nil)
}

func newProviderInventoryObserved(version int, previous string, objects []providerIntegrationObject, maximumEntries int, maximumPayload uint64, meter *providerWorkMeter) (*providerIntegrationInventory, error) {
	domain, cap, valid := providerInventoryDomain(version)
	if !valid || maximumEntries <= 0 || maximumEntries > cap || maximumPayload == 0 || maximumPayload > 64<<20 || previous != "" && !lowerHash(previous) || len(objects) > maximumEntries || len(objects) == 0 && previous != "" {
		return nil, errProviderIntegration
	}
	entries := make([]providerIntegrationObjectRef, 0, len(objects))
	var total uint64
	for _, object := range objects {
		if len(object.raw) == 0 || len(object.raw) > 32<<20 || !candidateCapacity(0, total, uint64(len(object.raw)), maximumPayload) {
			return nil, errProviderIntegration
		}
		digest, ok := providerIntegrationObjectDigestObserved(object.kind, providerReplayPhaseBytes(object.raw, meter), meter)
		if !ok || digest != object.digest {
			return nil, errProviderIntegration
		}
		total += uint64(len(object.raw))
		entries = append(entries, providerIntegrationObjectRef{object.kind, object.digest, uint64(len(object.raw))})
	}
	if !providerInventoryOrderObserved(entries, meter) {
		return nil, errProviderIntegration
	}
	for n := 1; n < len(entries); n++ {
		left, aOK := providerReplayRefKey(entries[n-1], meter)
		right, bOK := providerReplayRefKey(entries[n], meter)
		if !aOK || !bOK || meter != nil && !meter.event(providerCompareEvent, uint64(len(left)+len(right))) || left == right {
			return nil, errProviderIntegration
		}
	}
	raw := providerInventoryBytesObserved(domain, previous, entries, meter)
	if len(raw) == 0 || len(raw) > 64<<10 {
		return nil, errProviderIntegration
	}
	digest := providerReplayDigest(domain, raw, meter)
	retained := providerReplayString(raw, meter)
	if digest == "" || retained == "" {
		return nil, errProviderIntegration
	}
	return &providerIntegrationInventory{retained, digest, previous, entries, total}, nil
}

// The first finishing phase has a concrete quadratic comparison bound. It
// does not depend on the Go library's sorting implementation: at most
// n*(n-1)/2 comparisons, each with two observed key copies. Earlier phases
// keep their existing sort and event contract. Neither path returns authority.
func providerInventoryOrderObserved(entries []providerIntegrationObjectRef, meter *providerWorkMeter) bool {
	if meter != nil && (meter.firstAdmission != nil || meter.firstOperation != nil) {
		if len(entries) > 256 {
			return false
		}
		for n := 1; n < len(entries); n++ {
			for k := n; k > 0; k-- {
				left, aOK := providerReplayRefKey(entries[k-1], meter)
				right, bOK := providerReplayRefKey(entries[k], meter)
				if !aOK || !bOK || !meter.event(providerCompareEvent, uint64(len(left)+len(right))) {
					return false
				}
				if left <= right {
					break
				}
				entries[k-1], entries[k] = entries[k], entries[k-1]
			}
		}
		return !meter.failed
	}
	sort.Slice(entries, func(a, b int) bool {
		left, aOK := providerReplayRefKey(entries[a], meter)
		right, bOK := providerReplayRefKey(entries[b], meter)
		return aOK && bOK && (meter == nil || meter.event(providerCompareEvent, uint64(len(left)+len(right)))) && left < right
	})
	return meter == nil || !meter.failed
}

func providerIntegrationRefKey(ref providerIntegrationObjectRef) string {
	return ref.kind + ":" + ref.digest
}

func providerIntegrationInventoryBytes(previous string, entries []providerIntegrationObjectRef) []byte {
	return providerInventoryBytes("sage-provider-custody-objects-v1", previous, entries)
}

func providerInventoryBytes(domain, previous string, entries []providerIntegrationObjectRef) []byte {
	return providerInventoryBytesObserved(domain, previous, entries, nil)
}

func providerInventoryBytesObserved(domain, previous string, entries []providerIntegrationObjectRef, meter *providerWorkMeter) []byte {
	if meter != nil && (len(entries) > 256 || !meter.preflight(providerEncodeEvent, 64<<10)) {
		return nil
	}
	var parent any
	if previous != "" {
		parent = previous
	}
	items := make([]map[string]string, 0, len(entries))
	for _, e := range entries {
		items = append(items, map[string]string{"kind": e.kind, "digest": e.digest, "bytes": strconv.FormatUint(e.bytes, 10)})
	}
	raw := providerReplayCanonical(map[string]any{"format": domain, "previous": parent, "entries": items}, 64<<10, meter)
	return raw
}

func parseProviderIntegrationInventory(raw []byte, expectedPrevious string, maximumEntries int, maximumObject, maximumPayload uint64) (*providerIntegrationInventory, error) {
	return parseProviderInventory(1, raw, expectedPrevious, maximumEntries, maximumObject, maximumPayload)
}

func parseProviderFrontierInventory(raw []byte, expectedPrevious string, maximumEntries int, maximumObject, maximumPayload uint64) (*providerIntegrationInventory, error) {
	return parseProviderInventory(2, raw, expectedPrevious, maximumEntries, maximumObject, maximumPayload)
}

func parseProviderFrontierInventoryObserved(raw []byte, expectedPrevious string, maximumEntries int, maximumObject, maximumPayload uint64, meter *providerWorkMeter) (*providerIntegrationInventory, error) {
	return parseProviderInventoryObserved(2, raw, expectedPrevious, maximumEntries, maximumObject, maximumPayload, meter)
}

func parseProviderInventory(version int, raw []byte, expectedPrevious string, maximumEntries int, maximumObject, maximumPayload uint64) (*providerIntegrationInventory, error) {
	return parseProviderInventoryObserved(version, raw, expectedPrevious, maximumEntries, maximumObject, maximumPayload, nil)
}

func parseProviderInventoryObserved(version int, raw []byte, expectedPrevious string, maximumEntries int, maximumObject, maximumPayload uint64, meter *providerWorkMeter) (*providerIntegrationInventory, error) {
	domain, cap, valid := providerInventoryDomain(version)
	if !valid || maximumEntries <= 0 || maximumEntries > cap || maximumObject == 0 || maximumObject > 32<<20 || maximumPayload == 0 || maximumPayload > 64<<20 || len(raw) == 0 || len(raw) > 64<<10 || expectedPrevious != "" && !lowerHash(expectedPrevious) {
		return nil, errProviderIntegration
	}
	m := contentObject(providerReplayJSON(raw, meter), "format", "previous", "entries")
	if m == nil || m["format"] != domain || expectedPrevious == "" && m["previous"] != nil || expectedPrevious != "" && m["previous"] != expectedPrevious {
		return nil, errProviderIntegration
	}
	items, ok := m["entries"].([]any)
	if !ok || len(items) > maximumEntries || len(items) == 0 && expectedPrevious != "" {
		return nil, errProviderIntegration
	}
	entries := make([]providerIntegrationObjectRef, 0, len(items))
	var total uint64
	for _, item := range items {
		e := contentObject(item, "kind", "digest", "bytes")
		kind, digest := contentString(e["kind"]), contentString(e["digest"])
		_, known := providerIntegrationDomain(kind)
		size, valid := providerCandidateUint(e["bytes"], true)
		if e == nil || !known || !lowerHash(digest) || !valid || size > maximumObject || !candidateCapacity(0, total, size, maximumPayload) {
			return nil, errProviderIntegration
		}
		entry := providerIntegrationObjectRef{kind, digest, size}
		if len(entries) > 0 {
			left, aOK := providerReplayRefKey(entries[len(entries)-1], meter)
			right, bOK := providerReplayRefKey(entry, meter)
			if !aOK || !bOK || meter != nil && !meter.event(providerCompareEvent, uint64(len(left)+len(right))) || left >= right {
				return nil, errProviderIntegration
			}
		}
		total += size
		entries = append(entries, entry)
	}
	digest := providerReplayDigest(domain, raw, meter)
	retained := providerReplayString(raw, meter)
	if digest == "" || retained == "" {
		return nil, errProviderIntegration
	}
	return &providerIntegrationInventory{retained, digest, expectedPrevious, entries, total}, nil
}
