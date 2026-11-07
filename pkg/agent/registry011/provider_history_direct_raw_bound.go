package registry011

// Original prospective source maxima, not observed smaller DATA or a profile.
type providerHistoryDirectRawInput struct {
	groups, requests, metadata, object, wire, body uint64
}

const (
	providerHistoryRawHealthy = iota
	providerHistoryRawPrior
	providerHistoryRawAdmission
	providerHistoryRawConsumption
	providerHistoryRawSettlement
	providerHistoryRawOuterConsumption
	providerHistoryRawAnchor
	providerHistoryRawWriter
	providerHistoryRawAdvance
	providerHistoryRawOutput
	providerHistoryRawCapture
	providerHistoryRawCurrentSpool
	providerHistoryRawArchivedSpool
	providerHistoryRawCurrentResult
	providerHistoryRawArchivedResult
	providerHistoryRawPolicy
	providerHistoryRawCheckpoint
	providerHistoryRawArchive
	providerHistoryRawWrapper
	providerHistoryRawMemberHead
	providerHistoryRawRows
)

type providerHistoryDirectRawCost struct {
	healthy, priorStages, archivedMembers uint64
	currentMembers, currentResults        uint64
	nonfirst, policyCalls, wrapperHashes  uint64
	rows                                  [providerHistoryRawRows]candidateWorkVector
	total                                 candidateWorkVector
}

// Successful direct raw copies/hashes/comparisons and policy-getter decoding.
// Actual I/O, filenames, object/frame/inventory/T/C/S parsers, ledger/cursor
// roots, typed metadata and request-field comparisons are separate components.
// Native CPU/RAM, failure/recovery/final work and the complete source-bound
// factory remain mandatory: a positive result grants no execution authority.
func providerHistoryDirectRawUpper(in providerHistoryDirectRawInput) (providerHistoryDirectRawCost, bool) {
	var empty providerHistoryDirectRawCost
	maximum, ok := providerUpperProduct(in.groups, providerHistoryGroupMembers)
	if !ok || in.groups == 0 || in.groups > (1<<20)/3 || in.requests < in.groups || in.requests > maximum ||
		in.metadata == 0 || in.metadata > 1<<20 || in.object == 0 || in.object > providerHistoryOriginalPackBytes ||
		in.wire == 0 || in.wire > 16<<20 || in.body == 0 || in.body > 8<<20 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryDirectRawCost
	g, r := in.groups, in.requests
	j := a.product(g, g-1) / 2
	extra := a.difference(r, g)
	full, tail := extra/31, extra%31
	e := a.sum(j, a.product(31, a.difference(a.product(full, g), a.product(full, a.sum(full, 1))/2)))
	if tail != 0 {
		e = a.sum(e, a.product(tail, a.difference(a.difference(g, full), 1)))
	}
	// Optional groups can be absent. Pack32-member groups for current spool
	// squares and nonfirst-member counts, rather than using exactly G groups.
	squares := a.sum(a.product(r/32, 1024), a.product(r%32, r%32))
	out.healthy = a.difference(a.product(14, g), 2)
	out.priorStages = a.product(14, j)
	out.archivedMembers = a.product(14, e)
	out.currentMembers = a.sum(a.product(2, squares), a.product(3, r))
	out.currentResults = a.sum(a.product(2, squares), r)
	out.nonfirst = a.difference(r, a.sum(r, 31)/32)
	out.policyCalls = a.sum(r, out.nonfirst, a.product(3, g), out.priorStages)
	out.wrapperHashes = a.sum(a.difference(a.product(24, g), 1), a.product(3, r), out.nonfirst)
	const h, p uint64 = candidateLedgerMetadataLimit, 64 << 10
	i, l := min(in.metadata, h), a.sum(in.wire, 64<<10)

	// Full healthy boundary, including its chain-copy/count events. The
	// initial model's raw controls are already in the legacy component.
	out.rows[providerHistoryRawHealthy][10] = a.sum(a.product(5*h+271, out.healthy), a.product(15, out.priorStages))
	// Retained checkpoint raw readback is DISTINCT from frame parser's own
	// reconstructed bytes, inventory parsing and object text/digest checks.
	// Every prior S also compares its parsed digest with the frame operation,
	// separately from the ledger's predecessor and member-result bindings.
	out.rows[providerHistoryRawPrior][10] = a.sum(a.product(a.sum(14*h, a.product(3, i), 277), a.product(3, out.priorStages)),
		a.product(133, out.priorStages))
	out.rows[providerHistoryRawAdmission][10] = a.product(11*h+273, a.product(10, g))
	out.rows[providerHistoryRawConsumption][10] = a.product(10*h+139, a.product(7, g))
	out.rows[providerHistoryRawSettlement][10] = a.product(a.sum(5*h, a.product(2, in.object), 270), a.product(2, g))
	out.rows[providerHistoryRawOuterConsumption][10] = a.product(3*h+2, a.product(2, g))
	out.rows[providerHistoryRawAnchor][10] = a.product(7*h+4, a.sum(g, a.product(2, r)))
	// All three physical writers: expected old pins, raw file-input copies
	// and claim.highwater string copies. A reused before-pin is copied ONCE.
	out.rows[providerHistoryRawWriter][10] = a.product(a.sum(a.product(2, in.object), 45*h, a.product(3, i), 41), g)
	out.rows[providerHistoryRawAdvance][10] = a.product(3*h+2, g-1)
	// The result view contains digest66, the HTTP BODY and the JSON result.
	// Both variable fields have the body ceiling; each has two copy events.
	out.rows[providerHistoryRawOutput][10] = a.sum(a.product(a.sum(a.product(2, in.object), 2), g),
		a.product(a.sum(a.product(4, in.body), 138), r))
	out.rows[providerHistoryRawCapture][10] = a.product(a.sum(a.product(3, l), 6), r)
	out.rows[providerHistoryRawCurrentSpool][10] = a.product(a.sum(a.product(2, l), 270), out.currentMembers)
	out.rows[providerHistoryRawArchivedSpool][10] = a.product(a.sum(a.product(4, l), 272), out.archivedMembers)
	// Direct spool/archived result bindings are NOT the existing separate
	// result.digest-to-settlement binding inside ledger.settleOriginals.
	out.rows[providerHistoryRawCurrentResult][10] = a.product(133, out.currentResults)
	out.rows[providerHistoryRawArchivedResult][10] = a.product(133, out.archivedMembers)
	out.rows[providerHistoryRawPolicy][2] = a.product(p, out.policyCalls)
	out.rows[providerHistoryRawPolicy][10] = a.product(p*p+p+2, out.policyCalls)
	// Normal checkpoint clones+final matches, T descriptor, then the single
	// initial checkpoint/model and both original descriptor variable strings.
	out.rows[providerHistoryRawCheckpoint][10] = a.sum(a.product(a.product(3, g), a.sum(12*h, a.product(3, i), 10)),
		a.product(77, g), h, a.product(2, in.metadata), 70)
	out.rows[providerHistoryRawArchive][10] = a.sum(a.product(67, r), a.product(4097, a.difference(a.product(2, g), 1)))
	out.rows[providerHistoryRawWrapper][10] = a.product(133, out.wrapperHashes)
	out.rows[providerHistoryRawMemberHead][10] = a.product(3*h+2, a.product(2, r))
	for _, row := range out.rows {
		for coordinate, n := range row {
			out.total[coordinate] = a.sum(out.total[coordinate], n)
		}
	}
	if a.failed {
		return empty, false
	}
	return out, true
}
