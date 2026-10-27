package registry011

import "strconv"

// These are explicit candidate implementation ceilings, not operating defaults.
// The public ephemeral CandidateLedgerBounds remains a complete4096-record
// lifetime. Only the private connected continuation can use this other profile.
type providerLedgerBounds struct {
	maximumRecords, pageRecords, maximumPages     uint64
	maximumPayloadBytes, maximumMetadataBytes     uint64
	pagePayloadBytes, pageMetadataBytes           uint64
	maximumResidentBytes, codecMemoryMultiplier   uint64
	ticketMemoryOverhead, operationMemoryOverhead uint64
	segmentMemoryOverhead, objectMemoryOverhead   uint64
	maximumCommits, maximumObjects                uint64
}

func (b providerLedgerBounds) valid() bool {
	if b.maximumRecords == 0 || b.maximumRecords > 1<<20 || b.pageRecords == 0 || b.pageRecords > 4096 || b.pageRecords > b.maximumRecords || b.maximumPages == 0 || b.maximumPages > 1<<20 ||
		b.maximumPayloadBytes == 0 || b.maximumPayloadBytes > 16<<30 || b.maximumMetadataBytes == 0 || b.maximumMetadataBytes > 64<<30 ||
		b.pagePayloadBytes == 0 || b.pagePayloadBytes > b.maximumPayloadBytes || b.pageMetadataBytes == 0 || b.pageMetadataBytes > b.maximumMetadataBytes ||
		b.maximumResidentBytes == 0 || b.maximumResidentBytes > 16<<30 || b.codecMemoryMultiplier == 0 || b.codecMemoryMultiplier > 256 ||
		b.ticketMemoryOverhead == 0 || b.ticketMemoryOverhead > 1<<20 || b.operationMemoryOverhead == 0 || b.operationMemoryOverhead > 1<<20 || b.segmentMemoryOverhead == 0 || b.segmentMemoryOverhead > 1<<20 || b.objectMemoryOverhead == 0 || b.objectMemoryOverhead > 1<<20 ||
		b.maximumCommits == 0 || b.maximumCommits > 2<<20 || b.maximumObjects == 0 || b.maximumObjects > 16<<20 {
		return false
	}
	pages := (b.maximumRecords-1)/b.pageRecords + 1
	commits, ok := providerUpperSum(b.maximumRecords, pages, 2)
	return ok && b.maximumPages >= pages && b.maximumCommits >= commits
}

func (b providerLedgerBounds) object() map[string]string {
	values := map[string]uint64{
		"maximum_records": b.maximumRecords, "page_records": b.pageRecords, "maximum_pages": b.maximumPages,
		"maximum_payload_bytes": b.maximumPayloadBytes, "maximum_metadata_bytes": b.maximumMetadataBytes,
		"page_payload_bytes": b.pagePayloadBytes, "page_metadata_bytes": b.pageMetadataBytes,
		"maximum_resident_bytes": b.maximumResidentBytes, "codec_memory_multiplier": b.codecMemoryMultiplier,
		"ticket_memory_overhead": b.ticketMemoryOverhead, "operation_memory_overhead": b.operationMemoryOverhead,
		"segment_memory_overhead": b.segmentMemoryOverhead, "object_memory_overhead": b.objectMemoryOverhead,
		"maximum_commits": b.maximumCommits, "maximum_objects": b.maximumObjects,
	}
	m := map[string]string{"format": "sage-provider-private-ledger-profile-v1"}
	for key, value := range values {
		m[key] = strconv.FormatUint(value, 10)
	}
	return m
}

// Counts and actual retained schema bytes come from the owned typed state.
// Overhead is an explicit qualified-profile upper, never a measured allocator
// claim. Include both coordinator/custodian copies and bounded codec scratch.
func (b providerLedgerBounds) residentUpper(tickets, operations, segments, objects, retainedSchemaBytes, scratch uint64) (uint64, bool) {
	if !b.valid() || tickets > b.maximumRecords || operations > b.maximumCommits || segments > b.maximumRecords || objects > b.maximumObjects {
		return 0, false
	}
	var total uint64
	for _, item := range [][2]uint64{{tickets, b.ticketMemoryOverhead}, {operations, b.operationMemoryOverhead}, {segments, b.segmentMemoryOverhead}, {objects, b.objectMemoryOverhead}} {
		bytes, ok := providerUpperProduct(item[0], item[1])
		if !ok {
			return 0, false
		}
		total, ok = providerUpperSum(total, bytes)
		if !ok {
			return 0, false
		}
	}
	total, ok := providerUpperSum(total, retainedSchemaBytes)
	if !ok {
		return 0, false
	}
	total, ok = providerUpperProduct(total, 2)
	if !ok {
		return 0, false
	}
	temporary, valid := providerUpperProduct(scratch, b.codecMemoryMultiplier)
	if !valid {
		return 0, false
	}
	total, ok = providerUpperSum(total, temporary)
	return total, ok && total <= b.maximumResidentBytes
}
