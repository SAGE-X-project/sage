package registry011

// Explicit request/table byte events, not native decoding, allocation or a
// complete collection reservation. Original parsing includes its caller copy.
// tableBuilt excludes grammar-specific nextTable work before construction.
type providerHistoryRequestControlCost struct {
	original, tableBase, tableMember, tableBuilt, tableEqual, packParsed candidateWorkVector
}

func providerHistoryRequestControlUpper(members, maximum uint64) (providerHistoryRequestControlCost, bool) {
	var empty providerHistoryRequestControlCost
	control, controlOK := providerHistoryLedgerControlUpper(members)
	pack, packOK := providerHistoryOriginalPackControlUpper(members, maximum)
	if !controlOK || !packOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	const params, body, head, table uint64 = 8192, 16384, 512, providerHistoryGroupTableBytes
	var out providerHistoryRequestControlCost
	// The copy validator already bounds the same constructor, normalization
	// and final wire equality. Keep its extra field checks conservatively;
	// add the original body decode, head scan and both explicit wire copies.
	out.original = control.requestCopy
	out.original[2] = a.sum(out.original[2], body)
	out.original[10] = a.sum(out.original[10], body, 1, head, 5,
		a.product(body, body), 1, body, head, 1)
	// The independent capture selector may decode params AGAIN. Reserve its
	// actual V4 phase copy even though inert legacy UNIT meters omit it.
	for coordinate := range out.tableMember {
		out.tableMember[coordinate] = a.sum(control.requestCopy[coordinate], a.product(2, control.requestClass[coordinate]))
	}
	out.tableBase[10] = a.sum(a.product(table, table), 1, table, 1,
		table, uint64(len(providerHistoryGroupTableDomain)), 2)
	for coordinate := range out.tableBuilt {
		out.tableBuilt[coordinate] = a.sum(out.tableBase[coordinate], a.product(members, out.tableMember[coordinate]))
	}
	// Every unused slot is compared too. Seven request-field comparisons
	// retain the same intermediate body+head bound as the copy validator.
	requestEqual := a.sum(2*32+1, 2*params+1, 2*body+1, 2*head+1,
		body, head, body, 1, 2*133)
	out.tableEqual[10] = a.sum(2*table+1, 133, a.product(providerHistoryGroupMembers, requestEqual))
	for coordinate := range out.packParsed {
		out.packParsed[coordinate] = a.sum(pack.parsed[coordinate],
			a.product(members, out.original[coordinate]), out.tableBuilt[coordinate])
	}
	out.packParsed[10] = a.sum(out.packParsed[10], 133)
	if a.failed {
		return empty, false
	}
	return out, true
}

// Successful ORIGINAL call topology only. SAME-C member equivalence validates
// and copies its existing model; it does NOT parse/transition T/C again. Those
// ledger roots/copies are already counted separately. These rows concern the
// extra record operations outside ledger-root validation and TLS contexts.
type providerHistoryRequestControlCalls struct {
	packBuilds, packParses, originals, tableBuilds, tableMembers, tableEquals uint64
	ticketOps, consumptionOps, settlementOps, transitionHashes, resultHashes  uint64
}

func providerHistoryCollectionRequestControlCallsBound(groups, requests uint64) (providerHistoryRequestControlCalls, bool) {
	var empty providerHistoryRequestControlCalls
	if _, ok := providerHistoryCollectionRootCallsBound(groups, requests); !ok {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	tri := a.product(groups, groups-1) / 2
	extra := a.difference(requests, groups)
	full, tail := extra/31, extra%31
	weighted := a.sum(tri, a.product(31, a.difference(a.product(full, groups), a.product(full, a.sum(full, 1))/2)))
	if tail != 0 {
		weighted = a.sum(weighted, a.product(tail, a.difference(a.difference(groups, full), 1)))
	}
	// Current T parser counts: own2 + C3 + dispatch2 + S3. Each prior
	// successful group is reconstructed fourteen more times per later group.
	parses := a.sum(a.product(10, groups), a.product(14, tri))
	// Two planning tables, eleven admit checks, three S-boundary checks and
	// ten original-pack parsers. Do not add another writer-selection table.
	builds := a.sum(a.product(26, groups), a.product(42, tri))
	members := a.sum(a.product(26, requests), a.product(42, weighted))
	equals := a.sum(a.product(15, groups), a.product(28, tri))
	// Constructor + parsers + extra transition validator: T1+10+11,
	// C1+7+8, S1+2+3. S transition means constructor plus fixed equality.
	tickets := a.sum(a.product(22, groups), a.product(28, tri))
	consumptions := a.sum(a.product(16, groups), a.product(28, tri))
	settlements := a.sum(a.product(6, groups), a.product(28, tri))
	originals := a.sum(a.product(10, requests), a.product(14, weighted))
	transitions := a.sum(a.product(22, groups), a.product(42, tri))
	results := a.sum(a.product(3, requests), a.product(14, weighted))
	if a.failed {
		return empty, false
	}
	return providerHistoryRequestControlCalls{groups, parses, originals, builds, members, equals,
		tickets, consumptions, settlements, transitions, results}, true
}

// Compose disjoint raw-pack, request, table and outside-record subgraphs.
// Do not multiply the COMPLETE packParsed row as well: its request/table
// portions are already expanded here. Native CPU/RAM, source-specific grammar,
// remaining raw/model checks, failure/final/recovery and factory stay separate.
func providerHistoryCollectionRepeatedRequestControlUpper(groups, requests, maximum uint64) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	calls, callsOK := providerHistoryCollectionRequestControlCallsBound(groups, requests)
	one, oneOK := providerHistoryRequestControlUpper(providerHistoryGroupMembers, maximum)
	pack, packOK := providerHistoryOriginalPackControlUpper(providerHistoryGroupMembers, maximum)
	record, recordOK := providerHistoryRecordControlUpper(providerHistoryGroupMembers)
	if !callsOK || !oneOK || !packOK || !recordOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var work candidateWorkVector
	for coordinate := range work {
		work[coordinate] = a.sum(a.product(calls.packBuilds, pack.built[coordinate]),
			a.product(calls.packParses, pack.parsed[coordinate]), a.product(calls.originals, one.original[coordinate]),
			a.product(calls.tableBuilds, one.tableBase[coordinate]), a.product(calls.tableMembers, one.tableMember[coordinate]),
			a.product(calls.tableEquals, one.tableEqual[coordinate]), a.product(calls.ticketOps, record.ticket[coordinate]),
			a.product(calls.consumptionOps, record.consumption[coordinate]), a.product(calls.settlementOps, record.settlement[coordinate]))
	}
	// Pack/table binding, each transition predecessor, each S result binding.
	comparisons := a.sum(calls.packParses, calls.transitionHashes, calls.resultHashes)
	work[10] = a.sum(work[10], a.product(133, comparisons))
	if a.failed {
		return empty, false
	}
	return work, true
}

// CONTROL subgraphs of one flat leaf admission atom. A coordinator atom
// starts AFTER both planning prefix replays. A replay atom includes its whole
// request pack, every HTTP original, rebuilt table, T parser and DISTINCT T
// validation in admit, through full table equality and admitted-root success.
// Only that successful postcondition qualifies normal; an unfinished atom
// retains envelope. Numeric rows establish no source provenance or authority.
//
// Ledger copies/roots/cursors and source nextTable generation are separate K
// at their actual positions. So are object/inventory/frame/file/native work,
// cleanup, clocks, RAM and complete failure/final/factory composition. Neither
// normal nor this component alone is a sufficient whole incoming allowance.
type providerHistoryAdmissionClusterControlCost struct{ normal, envelope candidateWorkVector }

func providerHistoryAdmissionClusterControlUpper(members, maximum uint64) (providerHistoryAdmissionClusterControlCost, bool) {
	var empty providerHistoryAdmissionClusterControlCost
	planned, plannedOK := providerHistoryPlannedLedgerControlUpper(members)
	record, recordOK := providerHistoryPlannedRecordControlUpper(members)
	request, requestOK := providerHistoryRequestControlUpper(members, maximum)
	pack, packOK := providerHistoryOriginalPackControlUpper(members, maximum)
	if !plannedOK || !recordOK || !requestOK || !packOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	quadratic := func(n uint64) uint64 { return a.sum(a.product(n, n), 1) }
	const params, body, head, wire uint64 = 500, 578, 178, 756
	// Completed wire equality and the full original next-table fence constrain
	// each earlier reconstructed request too. These are postcondition costs,
	// never permission to narrow an unverified raw decoder or its preflight.
	original := planned.normal.requestCopy
	original[2] = a.sum(original[2], body)
	original[10] = a.sum(original[10], body, 1, min(wire, 512), 5, quadratic(body), body, head, 1)
	var table candidateWorkVector
	for coordinate := range table {
		table[coordinate] = a.product(members, a.sum(planned.normal.requestCopy[coordinate], a.product(2, planned.normal.requestClass[coordinate])))
	}
	table[10] = a.sum(table[10], quadratic(planned.tableBytes), planned.tableBytes, 1,
		planned.tableBytes, uint64(len(providerHistoryGroupTableDomain)), 2)
	requestEqual := a.sum(2*32+1, 2*params+1, 2*body+1, 2*head+1, 2*wire+1, 2*133)
	var equal candidateWorkVector
	equal[10] = a.sum(a.product(2, planned.tableBytes), 1, 133, a.product(providerHistoryGroupMembers, requestEqual))
	// A successful request descriptor has seven fixed fields, first=0 and
	// exactly m request rows. Even over-reserving EVERY decimal at20 bytes,
	// the ASCII outer object is below512 and each row below192. Hashes have
	// 66 bytes and ordinals at most2. No escaped/ignored/trailing data survives
	// canonical descriptor, full wire and full next-table equality together.
	header := a.sum(512, a.product(192, members))
	magic, domain := uint64(len(providerHistoryOriginalPackMagic)), uint64(len(providerHistoryOriginalPackDomain))
	raw := min(maximum, a.sum(header, magic, 4, a.product(wire, members)))
	var normalPack providerHistoryPackControlCost
	normalPack.built[10] = a.sum(quadratic(header), a.product(5, raw), header, a.product(3, members), magic, domain, 22)
	normalPack.parsed[2] = a.sum(header, 4)
	normalPack.parsed[10] = a.sum(a.product(2, quadratic(header)), a.product(4, raw), a.product(2, header), a.product(134, members), a.product(3, magic), domain, 157)
	var out providerHistoryAdmissionClusterControlCost
	for coordinate := range out.normal {
		normalReplay := a.sum(normalPack.parsed[coordinate], a.product(members, original[coordinate]), table[coordinate], a.product(2, record.normal.ticket[coordinate]), equal[coordinate])
		normalCoordinator := a.sum(normalPack.built[coordinate], a.product(2, record.normal.ticket[coordinate]), equal[coordinate])
		envelopeReplay := a.sum(pack.parsed[coordinate], a.product(members, request.original[coordinate]), request.tableBuilt[coordinate], a.product(2, record.envelope.ticket[coordinate]), request.tableEqual[coordinate])
		envelopeCoordinator := a.sum(pack.built[coordinate], a.product(2, record.envelope.ticket[coordinate]), request.tableEqual[coordinate])
		out.normal[coordinate], out.envelope[coordinate] = max(normalReplay, normalCoordinator), max(envelopeReplay, envelopeCoordinator)
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

// One original actual-accounting scope with at most atoms flat leaf atoms.
// All prior atoms must have completed the full original fence; after the
// first unfinished atom the actual selected root must abort before another
// atom starts. Both T operations and every assigned sibling are in each row.
// Source-derived counts and external K/deferred incoming bounds are separate.
// EACH I/T upper still needs its OWN complete component; this is not one
// global headroom to replace SUM of full irrevocable stage/member uppers.
func providerHistoryCollectionPointwiseAdmissionClusterControlUpper(atoms, maximum uint64) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	if atoms == 0 {
		return empty, false
	}
	one, ok := providerHistoryAdmissionClusterControlUpper(providerHistoryGroupMembers, maximum)
	if !ok {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out candidateWorkVector
	for coordinate := range out {
		out[coordinate] = a.sum(a.product(atoms-1, one.normal[coordinate]), one.envelope[coordinate])
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

// Flat admission atoms in one SAME original T window. Both planning prefixes
// precede the direct coordinator leaf. Own advancement follows S inside this
// window; no same-C member pass admits T again. This count grants no allowance.
func providerHistoryCollectionStageAdmissionAtomsBound(prior uint64, advance bool) (uint64, bool) {
	const maximumGroups uint64 = (1 << 20) / 3
	if prior >= maximumGroups || advance && prior+1 >= maximumGroups {
		return 0, false
	}
	var a providerHistoryReadArithmetic
	atoms := a.sum(11, a.product(12, prior))
	if advance {
		atoms = a.sum(atoms, 2, a.product(2, prior))
	}
	if a.failed {
		return 0, false
	}
	return atoms, true
}

// Successful whole topology only. This is NOT the sum of each original full
// stage allowance: every admitted stage must retain its OWN generic envelope.
func providerHistoryCollectionWholeAdmissionAtomsBound(groups, requests uint64) (uint64, bool) {
	if _, ok := providerHistoryCollectionRootCallsBound(groups, requests); !ok {
		return 0, false
	}
	var a providerHistoryReadArithmetic
	tri := a.product(groups, groups-1) / 2
	atoms := a.sum(a.product(11, groups), a.product(14, tri))
	if a.failed {
		return 0, false
	}
	return atoms, true
}

// Last and nonfinal windows have different maxima. Fewer optional groups or
// an interrupted prefix cannot add atoms. Every actual stage still needs its
// complete source-bound K/native/memory/failure rows and completed-leaf fence.
func providerHistoryCollectionUniformAdmissionAtomsBound(groups, requests uint64) (uint64, bool) {
	if _, ok := providerHistoryCollectionRootCallsBound(groups, requests); !ok {
		return 0, false
	}
	last, ok := providerHistoryCollectionStageAdmissionAtomsBound(groups-1, false)
	if !ok {
		return 0, false
	}
	if groups == 1 {
		return last, true
	}
	nonfinal, ok := providerHistoryCollectionStageAdmissionAtomsBound(groups-2, true)
	if !ok {
		return 0, false
	}
	return max(last, nonfinal), true
}

// Only three disjoint original comparison sites: reconstructed request-table
// digest binding, T/C/S transition predecessor, and successful S result binding.
// Other ledger/record/TLS/pack/internal/member hashes retain their prior rows or
// separate K. These counts do not authenticate a source, state or private hash.
type providerHistoryRoleBindingCalls struct{ packTables, predecessors, results uint64 }

type providerHistoryRoleBindingCounts struct {
	before, successClosure, failedClosure, advance providerHistoryRoleBindingCalls
}

func providerHistoryCollectionRoleBindingCallsBound(prior, pastMembers, members uint64, advancing bool) (providerHistoryRoleBindingCounts, bool) {
	var empty providerHistoryRoleBindingCounts
	const maximumGroups uint64 = (1 << 20) / 3
	if prior >= maximumGroups || advancing && prior+1 >= maximumGroups || members == 0 || members > providerHistoryGroupMembers {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	maximumPast := a.product(prior, providerHistoryGroupMembers)
	if a.failed || pastMembers < prior || pastMembers > maximumPast {
		return empty, false
	}
	out := providerHistoryRoleBindingCounts{
		before:         providerHistoryRoleBindingCalls{a.sum(7, a.product(9, prior)), a.sum(13, a.product(27, prior)), a.product(9, pastMembers)},
		successClosure: providerHistoryRoleBindingCalls{a.sum(3, a.product(3, prior)), a.sum(9, a.product(9, prior)), a.product(3, a.sum(pastMembers, members))},
		failedClosure:  providerHistoryRoleBindingCalls{a.sum(3, a.product(3, prior)), a.sum(9, a.product(9, prior)), a.product(3, pastMembers)},
	}
	if advancing {
		out.advance = providerHistoryRoleBindingCalls{a.sum(2, a.product(2, prior)), a.sum(6, a.product(6, prior)), a.product(2, a.sum(pastMembers, members))}
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

// Each assigned original comparison has TWO source-bounded66-byte hashes and
// one compare event, hence N=E=133 logical units. This width must hold BEFORE
// the event: request-pack shape validates expected table and the constructor
// returns a fresh digest; T/C validation precedes their predecessor comparisons.
// Successful l.validObserved implies head==fresh66-byte digest.
//
// S.previous is compared BEFORE settle's reconstruction checks, so the original
// caller's new/parseSettlement constructor must already have lowerHash-checked
// its expected previous. Likewise the original raw member parser must supply
// result.digest66 BEFORE settle compares it; the later boundary guard is NOT
// that earlier fence. Arbitrary damaged private S/result fields are not covered.
// Numeric calls alone establish none of these source or ownership conditions.
func (c providerHistoryRoleBindingCalls) upper() (candidateWorkVector, bool) {
	var empty candidateWorkVector
	var a providerHistoryReadArithmetic
	calls := a.sum(c.packTables, c.predecessors, c.results)
	var out candidateWorkVector
	out[10] = a.product(133, calls)
	if a.failed {
		return empty, false
	}
	return out, true
}

// A conditional comparison-only component, not a complete stage allowance.
// The source factory must bind all original field/reader fences and uniform
// historical member bounds before H0. I/P/M have none of THESE comparison sites,
// not zero hash work. The request pack's INTERNAL digest comparisons remain in
// its old raw-pack row; only its final rebuilt-table binding is assigned here.
// The admission atom deliberately excluded that external binding when unrolled.
// Native hashing/comparison CPU, all other K, incoming/deferred/final/recovery,
// full eleven-dimensional feasibility, RAM/windows and authority stay separate.
type providerHistoryRoleBindingParts struct {
	initial, before, preparation, member, closure, advance candidateWorkVector
}

func providerHistoryCollectionRoleBindingPartsUpper(prior, members uint64, advancing bool) (providerHistoryRoleBindingParts, bool) {
	var empty providerHistoryRoleBindingParts
	past, pastOK := providerUpperProduct(prior, providerHistoryGroupMembers)
	calls, callsOK := providerHistoryCollectionRoleBindingCallsBound(prior, past, members, advancing)
	if !pastOK || !callsOK {
		return empty, false
	}
	before, beforeOK := calls.before.upper()
	success, successOK := calls.successClosure.upper()
	failed, failedOK := calls.failedClosure.upper()
	if !beforeOK || !successOK || !failedOK {
		return empty, false
	}
	out := providerHistoryRoleBindingParts{before: before}
	for coordinate := range out.closure {
		out.closure[coordinate] = max(success[coordinate], failed[coordinate])
	}
	if advancing {
		advance, ok := calls.advance.upper()
		if !ok {
			return empty, false
		}
		out.advance = advance
	}
	return out, true
}
