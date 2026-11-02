package registry011

import "reflect"

// Prospective original-source maxima, not observed row lengths, reported
// actual work or authority. The complete factory must separately bind them.
type providerHistoryReadRetentionInput struct {
	groups, requests, firstObjects, catalogEntries uint64
	firstRaw, metadata, object, wire               uint64
}

type providerHistoryReadRetentionCost struct {
	capacity, maximumRows, recorders, recordedRows uint64
	refreshedRows, rootCalls, rootRows             uint64
	recordedBytes, refreshedBytes                  uint64
	initialization, recording, commitments         candidateWorkVector
	refresh, comparisons, total                    candidateWorkVector
}

// Retained ordered read DATA and its later SAME-C refresh only. File read
// events already belong to the read-byte component. This row adds distinct
// raw hashes, owned names/descriptors, rolling metadata roots and their direct
// comparisons. Context/claim/ancestor reflection, model/cursor roots, physical
// backing/RAM/native work, failure/final and factory qualification are separate.
func providerHistoryReadRetentionUpper(in providerHistoryReadRetentionInput) (providerHistoryReadRetentionCost, bool) {
	var empty providerHistoryReadRetentionCost
	reads, ok := providerHistoryCollectionPrefixReadBound(in.groups, in.requests, in.firstObjects)
	if !ok || in.catalogEntries == 0 || in.catalogEntries > 1<<20 || in.firstRaw == 0 || in.firstRaw > 64<<20 ||
		in.metadata == 0 || in.metadata > 1<<20 || in.object == 0 || in.object > 32<<20 || in.wire == 0 || in.wire > 16<<20 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryReadRetentionCost
	// Every recorded pass is H(g)+19: full healthy prefix followed by the
	// CURRENT T10/C9 reads. No catalog read is inside the recording interval.
	sumH := reads.healthyPrefixes / 12
	memberH := a.difference(reads.memberReplay, a.product(54, in.requests)) / 2
	out.capacity = a.sum(in.catalogEntries, in.firstObjects, 128)
	out.maximumRows = a.sum(33, in.firstObjects, a.product(23, in.groups-1),
		a.product(2, min(a.product(32, in.groups-1), in.requests-1)))
	if a.failed || out.maximumRows > out.capacity {
		return empty, false
	}
	out.recorders = a.product(2, in.groups)
	out.recordedRows = a.product(2, a.sum(sumH, a.product(19, in.groups)))
	out.refreshedRows = a.product(2, a.sum(memberH, a.product(19, in.requests)))
	// Each role records+seals one root per group; each role refreshes its
	// original root before AND after every member's fresh raw comparison.
	out.rootCalls = a.product(4, a.sum(in.groups, in.requests))
	out.rootRows = a.product(2, a.sum(out.recordedRows, out.refreshedRows))

	// The independently derived raw H(g) envelope. D retains original read
	// maxima. Current T/C adds one P request pack and18 D-sized originals;
	// ticket/consumption have their existing8KiB successful record ceiling.
	d := max(in.metadata, uint64(candidateLedgerMetadataLimit))
	u := a.sum(a.product(5, d), in.firstRaw)
	v := a.sum(a.product(21, d), a.product(2, in.object))
	l := a.sum(in.wire, 64<<10)
	tri := a.product(in.groups, in.groups-1) / 2
	extra := a.difference(in.requests, in.groups)
	full, tail := extra/31, extra%31
	weighted := a.sum(tri, a.product(31, a.difference(a.product(full, in.groups), a.product(full, a.sum(full, 1))/2)))
	if tail != 0 {
		weighted = a.sum(weighted, a.product(tail, a.difference(a.difference(in.groups, full), 1)))
	}
	q, r := in.requests/in.groups, in.requests%in.groups
	minimumSquare := a.sum(a.product(in.groups, a.product(q, q)), a.product(r, a.sum(a.product(2, q), 1)))
	cross := a.difference(a.product(in.requests, in.requests), minimumSquare) / 2
	sumBytes := a.sum(a.product(in.groups, u), a.product(v, tri), a.product(l, weighted))
	memberBytes := a.sum(a.product(u, in.requests), a.product(v, weighted), a.product(l, cross))
	current := a.sum(in.object, a.product(18, d))
	out.recordedBytes = a.product(2, a.sum(sumBytes, a.product(in.groups, current)))
	out.refreshedBytes = a.product(2, a.sum(memberBytes, a.product(in.requests, current)))

	out.initialization[10] = a.product(out.recorders, uint64(reflect.TypeFor[providerHistoryReplayReads]().Size())+1)
	// recordLocked: rawSHA(r+1), two name copies2(n+1), descriptor129.
	// n<=255, hence r+642. Fresh replay adds rawSHA and digest equality133.
	out.recording[10] = a.sum(out.recordedBytes, a.product(642, out.recordedRows))
	out.refresh[10] = a.sum(out.refreshedBytes, a.product(134, out.refreshedRows))
	const h uint64 = candidateLedgerMetadataLimit
	base := h*h + 1 + h + uint64(len("sage-provider-history-replay-reads-v1")) + 2
	row := h*h + 1 + h + uint64(len("sage-provider-history-replay-read-v1")) + 2
	out.commitments[10] = a.sum(a.product(out.rootCalls, base), a.product(out.rootRows, row))
	// Seal: root and model comparisons for2G roles. Member: two root
	// comparisons before refresh and one afterwards for each of2R roles.
	out.comparisons[10] = a.sum(a.product(532, in.groups), a.product(798, in.requests))
	out.total[10] = a.sum(out.initialization[10], out.recording[10], out.commitments[10], out.refresh[10], out.comparisons[10])
	if a.failed {
		return empty, false
	}
	return out, true
}
