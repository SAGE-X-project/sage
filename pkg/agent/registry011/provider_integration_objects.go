package registry011

import (
	"crypto/sha256"
	"encoding/hex"
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
	case "enrollment":
		return "sage-provider-enrollment-v2", true
	case "admission":
		return "sage-provider-admission-intent-v1", true
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
	domain, ok := providerIntegrationDomain(kind)
	if !ok {
		return "", false
	}
	if domain != "" {
		return providerCandidateHash(domain, raw), true
	}
	sum := sha256.Sum256(raw)
	return "0x" + hex.EncodeToString(sum[:]), true
}

// A format check here is deliberately weaker than semantic validation. Only
// the connected transition builder may introduce these objects into custody.
func newProviderIntegrationObject(kind string, raw []byte, maximum uint64) (providerIntegrationObject, error) {
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
		m, ok := contentJSON(raw).(map[string]any)
		if !ok || m["format"] != domain {
			return providerIntegrationObject{}, errProviderIntegration
		}
	}
	digest, _ := providerIntegrationObjectDigest(kind, raw)
	return providerIntegrationObject{kind, digest, string(raw)}, nil
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
		digest, ok := providerIntegrationObjectDigest(object.kind, []byte(object.raw))
		if !ok || digest != object.digest {
			return nil, errProviderIntegration
		}
		total += uint64(len(object.raw))
		entries = append(entries, providerIntegrationObjectRef{object.kind, object.digest, uint64(len(object.raw))})
	}
	sort.Slice(entries, func(a, b int) bool {
		return providerIntegrationRefKey(entries[a]) < providerIntegrationRefKey(entries[b])
	})
	for n := 1; n < len(entries); n++ {
		if providerIntegrationRefKey(entries[n-1]) == providerIntegrationRefKey(entries[n]) {
			return nil, errProviderIntegration
		}
	}
	raw := providerInventoryBytes(domain, previous, entries)
	if len(raw) > 64<<10 {
		return nil, errProviderIntegration
	}
	return &providerIntegrationInventory{string(raw), providerCandidateHash(domain, raw), previous, entries, total}, nil
}

func providerIntegrationRefKey(ref providerIntegrationObjectRef) string {
	return ref.kind + ":" + ref.digest
}

func providerIntegrationInventoryBytes(previous string, entries []providerIntegrationObjectRef) []byte {
	return providerInventoryBytes("sage-provider-custody-objects-v1", previous, entries)
}

func providerInventoryBytes(domain, previous string, entries []providerIntegrationObjectRef) []byte {
	var parent any
	if previous != "" {
		parent = previous
	}
	items := make([]map[string]string, 0, len(entries))
	for _, e := range entries {
		items = append(items, map[string]string{"kind": e.kind, "digest": e.digest, "bytes": strconv.FormatUint(e.bytes, 10)})
	}
	raw := providerIntegrationJSON(map[string]any{"format": domain, "previous": parent, "entries": items})
	return raw
}

func parseProviderIntegrationInventory(raw []byte, expectedPrevious string, maximumEntries int, maximumObject, maximumPayload uint64) (*providerIntegrationInventory, error) {
	return parseProviderInventory(1, raw, expectedPrevious, maximumEntries, maximumObject, maximumPayload)
}

func parseProviderFrontierInventory(raw []byte, expectedPrevious string, maximumEntries int, maximumObject, maximumPayload uint64) (*providerIntegrationInventory, error) {
	return parseProviderInventory(2, raw, expectedPrevious, maximumEntries, maximumObject, maximumPayload)
}

func parseProviderInventory(version int, raw []byte, expectedPrevious string, maximumEntries int, maximumObject, maximumPayload uint64) (*providerIntegrationInventory, error) {
	domain, cap, valid := providerInventoryDomain(version)
	if !valid || maximumEntries <= 0 || maximumEntries > cap || maximumObject == 0 || maximumObject > 32<<20 || maximumPayload == 0 || maximumPayload > 64<<20 || len(raw) == 0 || len(raw) > 64<<10 || expectedPrevious != "" && !lowerHash(expectedPrevious) {
		return nil, errProviderIntegration
	}
	m := contentObject(contentJSON(raw), "format", "previous", "entries")
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
		if len(entries) > 0 && providerIntegrationRefKey(entries[len(entries)-1]) >= providerIntegrationRefKey(entry) {
			return nil, errProviderIntegration
		}
		total += size
		entries = append(entries, entry)
	}
	return &providerIntegrationInventory{string(raw), providerCandidateHash(domain, raw), expectedPrevious, entries, total}, nil
}
