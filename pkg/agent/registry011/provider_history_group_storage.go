package registry011

import "strconv"

const providerHistoryInventoryDomain = "sage-provider-history-inventory-v1"

// History references have a separate closed namespace. In particular a binary
// pack is never smuggled through the legacy JSON-object or class-zero path.
type providerHistoryStoredObject struct {
	kind, digest, raw string
}

type providerHistoryStoredReference struct {
	kind, digest string
	bytes        uint64
}

type providerHistoryInventory struct {
	previous, raw, digest string
	count                 int
	entries               [8]providerHistoryStoredReference
}

type providerHistoryStorageCharge struct {
	physical, raw, payload, metadata, objects, files uint64
}

func (*providerHistoryInventory) UnmarshalJSON([]byte) error { return errProviderIntegration }

func providerHistoryStoredDomain(kind string) (string, bool) {
	switch kind {
	case "requests", "results":
		return providerHistoryOriginalPackDomain, true
	case "ticket":
		return providerHistoryGroupTicketDomain, true
	case "consumption":
		return providerHistoryGroupConsumptionDomain, true
	case "settlement":
		return providerHistoryGroupSettlementDomain, true
	}
	return "", false
}

func newProviderHistoryStoredObjectObserved(kind, raw string, maximum uint64, meter *providerWorkMeter) (providerHistoryStoredObject, error) {
	domain, known := providerHistoryStoredDomain(kind)
	if !known || len(raw) == 0 || maximum == 0 || maximum > providerHistoryOriginalPackBytes || uint64(len(raw)) > maximum ||
		kind != "requests" && kind != "results" && len(raw) > candidateLedgerMetadataLimit {
		return providerHistoryStoredObject{}, errProviderIntegration
	}
	digest := providerReplayDigest(domain, providerReplayBytes(raw, meter), meter)
	if digest == "" {
		return providerHistoryStoredObject{}, errProviderIntegration
	}
	return providerHistoryStoredObject{kind, digest, raw}, nil
}

func providerHistoryStoredName(ref providerHistoryStoredReference, meter *providerWorkMeter) string {
	if _, known := providerHistoryStoredDomain(ref.kind); !known || !lowerHash(ref.digest) || ref.bytes == 0 || ref.bytes > providerHistoryOriginalPackBytes {
		return ""
	}
	return providerReplayJoin([]string{"history-object-", ref.kind, "-", ref.digest[2:]}, meter)
}

func providerHistoryInventoryName(digest string, meter *providerWorkMeter) string {
	if !lowerHash(digest) {
		return ""
	}
	return providerReplayJoin([]string{"history-inventory-", digest[2:]}, meter)
}

func newProviderHistoryInventoryObserved(previous string, objects []providerHistoryStoredObject, maximum uint64, meter *providerWorkMeter) (*providerHistoryInventory, error) {
	if !lowerHash(previous) || len(objects) == 0 || len(objects) > 8 || maximum == 0 || maximum > candidateLedgerMetadataLimit {
		return nil, errProviderIntegration
	}
	// Preflight every scalar and the aggregate before hashing or building maps.
	fresh := uint64(0)
	previousKind := ""
	for _, object := range objects {
		if _, known := providerHistoryStoredDomain(object.kind); !known || !lowerHash(object.digest) || len(object.raw) == 0 || len(object.raw) > providerHistoryOriginalPackBytes ||
			!candidateCapacity(0, fresh, uint64(len(object.raw)), 64<<20) ||
			previousKind != "" && previousKind >= object.kind {
			return nil, errProviderIntegration
		}
		fresh += uint64(len(object.raw))
		previousKind = object.kind
	}
	v := &providerHistoryInventory{previous: previous, count: len(objects)}
	rows := make([]map[string]string, len(objects))
	for i, object := range objects {
		rebuilt, err := newProviderHistoryStoredObjectObserved(object.kind, object.raw, providerHistoryOriginalPackBytes, meter)
		if err != nil || !providerReplayTextEqual(rebuilt.digest, object.digest, meter) {
			return nil, errProviderIntegration
		}
		v.entries[i] = providerHistoryStoredReference{object.kind, object.digest, uint64(len(object.raw))}
		rows[i] = map[string]string{"kind": object.kind, "digest": object.digest, "bytes": strconv.FormatUint(uint64(len(object.raw)), 10)}
	}
	raw := providerReplayCanonical(map[string]any{"format": providerHistoryInventoryDomain, "previous": previous, "entries": rows}, maximum, meter)
	if len(raw) == 0 {
		return nil, errProviderIntegration
	}
	v.raw, v.digest = providerReplayString(raw, meter), providerReplayDigest(providerHistoryInventoryDomain, raw, meter)
	if v.raw == "" || v.digest == "" {
		return nil, errProviderIntegration
	}
	return v, nil
}

// The selected objects must themselves come from independent retained reads.
// This exact-byte parser neither chooses them nor grants a writer/ACK role.
func parseProviderHistoryInventoryObserved(raw []byte, previous, expected string, objects []providerHistoryStoredObject, maximum uint64, meter *providerWorkMeter) (*providerHistoryInventory, error) {
	if len(raw) == 0 || uint64(len(raw)) > maximum || !lowerHash(expected) {
		return nil, errProviderIntegration
	}
	v, err := newProviderHistoryInventoryObserved(previous, objects, maximum, meter)
	if err != nil || !providerReplayTextEqual(v.digest, expected, meter) || !providerReplayEqual(raw, providerReplayBytes(v.raw, meter), meter) {
		return nil, errProviderIntegration
	}
	return v, nil
}

// This is the actual finite delta's physical/raw allocation, not the whole
// program certificate. HEAD replacement is charged again without refunding
// its previous file. Inventory objects exclude the outer acyclic frame.
func providerHistoryStorageChargeFor(inventory *providerHistoryInventory, fresh []providerHistoryStoredObject, frame *providerHistoryCustodyFrame, highwaterBytes uint64, overhead uint64) (providerHistoryStorageCharge, bool) {
	var c providerHistoryStorageCharge
	if inventory == nil || frame == nil || inventory.count != len(fresh) || len(fresh) == 0 || len(fresh) > 8 ||
		len(inventory.raw) == 0 || len(inventory.raw) > candidateLedgerMetadataLimit || highwaterBytes == 0 || highwaterBytes > candidateLedgerMetadataLimit || overhead == 0 {
		return c, false
	}
	for _, object := range fresh {
		if len(object.raw) == 0 || len(object.raw) > providerHistoryOriginalPackBytes {
			return providerHistoryStorageCharge{}, false
		}
		n := uint64(len(object.raw))
		var ok bool
		if object.kind == "requests" || object.kind == "results" {
			c.payload, ok = providerUpperSum(c.payload, n)
		} else if _, known := providerHistoryStoredDomain(object.kind); known {
			c.metadata, ok = providerUpperSum(c.metadata, n)
		}
		if !ok || !candidateCapacity(0, c.raw, n, 64<<20) {
			return providerHistoryStorageCharge{}, false
		}
		c.raw += n
	}
	outer, ok := providerUpperSum(uint64(len(inventory.raw)), uint64(len(frame.intent)), uint64(len(frame.intent)), uint64(len(frame.commit)), uint64(len(frame.frontier)), highwaterBytes, highwaterBytes)
	metadata, metaOK := providerUpperSum(c.metadata, outer)
	raw, rawOK := providerUpperSum(c.raw, outer)
	// One inventory, two intents, commit/frontier, immutable highwater and HEAD.
	files := uint64(len(fresh)) + 7
	allocation, allocOK := providerUpperProduct(files, overhead)
	physical, physicalOK := providerUpperSum(raw, allocation)
	if !ok || !metaOK || !rawOK || !allocOK || !physicalOK {
		return providerHistoryStorageCharge{}, false
	}
	c.objects, c.files, c.metadata, c.raw, c.physical = uint64(len(fresh)), files, metadata, raw, physical
	return c, true
}

func (c providerHistoryStorageCharge) add(next providerHistoryStorageCharge, maximum providerHistoryGroupTotals) (providerHistoryStorageCharge, bool) {
	values := [6]uint64{c.physical, c.raw, c.payload, c.metadata, c.objects, c.files}
	more := [6]uint64{next.physical, next.raw, next.payload, next.metadata, next.objects, next.files}
	limits := [6]uint64{maximum.physical, maximum.retainedRaw, maximum.payload, maximum.metadata, maximum.objects, maximum.files}
	for i, n := range more {
		if !candidateCapacity(0, values[i], n, limits[i]) {
			return providerHistoryStorageCharge{}, false
		}
		values[i] += n
	}
	return providerHistoryStorageCharge{values[0], values[1], values[2], values[3], values[4], values[5]}, true
}
