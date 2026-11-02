package registry011

// Explicit byte-event costs for one closed record operation OUTSIDE the
// separately counted ledger validations. A row covers one constructor,
// validator or parser, with one bounded caller input copy; settlement also
// covers constructor plus fixed-array equality. It does not cover parser AND
// a subsequent equality, arbitrary damaged private fields or native work.
type providerHistoryRecordControlCost struct {
	ticket, consumption, settlement, context candidateWorkVector
}

func providerHistoryRecordControlUpper(members uint64) (providerHistoryRecordControlCost, bool) {
	var empty providerHistoryRecordControlCost
	control, ok := providerHistoryLedgerControlUpper(members)
	if !ok {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	const metadata uint64 = candidateLedgerMetadataLimit
	// Parser suffix: caller copy, reconstructed-record copy and raw equality.
	// 4M+3 also exceeds S's separate 4-field/32-result equality (21040).
	parseSuffix := a.sum(a.product(4, metadata), 3)
	settlementBody := a.sum(a.product(metadata, metadata), 1, metadata, 1,
		metadata, uint64(len(providerHistoryGroupSettlementDomain)), 2)
	var out providerHistoryRecordControlCost
	out.ticket, out.consumption = control.ticket, control.consumption
	out.ticket[10] = a.sum(out.ticket[10], parseSuffix)
	out.consumption[10] = a.sum(out.consumption[10], parseSuffix)
	// Result context and S construction each validate T directly, then T
	// again inside complete C validation. This is not just one T plus C body.
	for coordinate := range out.context {
		out.context[coordinate] = a.sum(control.ticket[coordinate], control.consumption[coordinate])
	}
	out.settlement = out.context
	out.settlement[10] = a.sum(out.settlement[10], settlementBody, parseSuffix)
	if a.failed {
		return empty, false
	}
	return out, true
}

// TLS diagnostic DATA construction/parsing only. Certificate verification,
// connection capture, HTTP/RPC member parsing and physical RAM are separate.
// Parsing includes one original caller text-to-byte copy before the parser.
type providerHistoryTLSControlCost struct{ original, parsed candidateWorkVector }

func providerHistoryTLSControlUpper(members uint64, certificates providerTLSBounds) (providerHistoryTLSControlCost, bool) {
	var empty providerHistoryTLSControlCost
	record, ok := providerHistoryRecordControlUpper(members)
	if !ok || !certificates.valid() {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	const metadata uint64 = 64 << 10
	quadratic := a.sum(a.product(metadata, metadata), 1)
	chainCopies := a.product(67, a.product(uint64(certificates.chains), uint64(certificates.certificates)))
	requestMatch := a.sum(a.product(3, 512+16384), 4)
	out := providerHistoryTLSControlCost{record.context, record.context}
	out.original[10] = a.sum(out.original[10], requestMatch, chainCopies, quadratic, metadata, 1)
	out.parsed[2] = a.sum(out.parsed[2], metadata)
	out.parsed[10] = a.sum(out.parsed[10], a.product(2, quadratic), a.product(3, metadata), 2)
	if a.failed {
		return empty, false
	}
	return out, true
}

// Successful-path call upper, not measured counts or a whole collection cost.
// Normal member preparation/retention reparse prior response originals; the
// three S views and all fresh successful-prefix reconstructions do so again.
type providerHistoryResultContextCalls struct {
	tlsOriginals, packParses, tlsParses, contexts uint64
}

func providerHistoryCollectionResultContextCallsBound(groups, requests uint64) (providerHistoryResultContextCalls, bool) {
	var empty providerHistoryResultContextCalls
	if _, ok := providerHistoryCollectionRootCallsBound(groups, requests); !ok {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	tri := a.product(groups, groups-1) / 2
	// One member per group, then place the remaining members in the earliest
	// groups, gives the maximum prior-prefix weight sum((G-1-j)*m_j).
	extra := a.difference(requests, groups)
	full, tail := extra/31, extra%31
	weighted := a.sum(tri, a.product(31, a.difference(a.product(full, groups), a.product(full, a.sum(full, 1))/2)))
	if tail != 0 {
		weighted = a.sum(weighted, a.product(tail, a.difference(a.difference(groups, full), 1)))
	}
	// At-most-G realization can have fewer groups and a larger sum of m^2.
	// Use packed 32-member groups, independent of the prior-weight maximum.
	squares := a.sum(a.product(requests/32, 1024), a.product(requests%32, requests%32))
	packs := a.sum(a.product(4, requests), groups, a.product(14, tri))
	tls := a.sum(a.product(2, squares), a.product(3, requests), a.product(14, weighted))
	contexts := a.sum(requests, packs, tls)
	if a.failed {
		return empty, false
	}
	return providerHistoryResultContextCalls{requests, packs, tls, contexts}, true
}

// Diagnostic TLS and ALL nested result-context validations, without counting
// the T/C context inside each TLS constructor/parser twice. Original packs,
// HTTP/RPC member replay and other record callers remain separate. Numeric
// G/R cannot authenticate a source or grant/renew a reservation/window/clock.
func providerHistoryCollectionRepeatedResultControlUpper(groups, requests uint64, certificates providerTLSBounds) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	calls, callsOK := providerHistoryCollectionResultContextCallsBound(groups, requests)
	record, recordOK := providerHistoryRecordControlUpper(providerHistoryGroupMembers)
	tls, tlsOK := providerHistoryTLSControlUpper(providerHistoryGroupMembers, certificates)
	if !callsOK || !recordOK || !tlsOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var work candidateWorkVector
	for coordinate := range work {
		work[coordinate] = a.sum(a.product(calls.tlsOriginals, tls.original[coordinate]),
			a.product(calls.tlsParses, tls.parsed[coordinate]), a.product(calls.packParses, record.context[coordinate]))
	}
	if a.failed {
		return empty, false
	}
	return work, true
}

// Planned successful CONTROL parts for a record operation outside ledger
// validation. The original-owned request grammar must be established BEFORE
// using a normal row. Incoming T/raw operations before exact next-table match
// retain the original envelope; private typed DATA alone is not qualification.
// Context/S also require the original-owned planned T/C values. No production
// caller, source witness or reservation is created by these numeric parts.
//
// Parser caller-copy/equality suffixes and S's whole metadata-body envelope
// remain at their original generic byte caps. This deliberately does not infer
// smaller raw input sizes. A normal row is entered-work DATA, not a sufficient
// standalone incoming allowance. Composite remaining/native/RAM stay separate.
type providerHistoryPlannedRecordControlCost struct {
	normal, envelope providerHistoryRecordControlCost
}

func providerHistoryPlannedRecordControlUpper(members uint64) (providerHistoryPlannedRecordControlCost, bool) {
	var empty providerHistoryPlannedRecordControlCost
	planned, plannedOK := providerHistoryPlannedLedgerControlUpper(members)
	envelope, envelopeOK := providerHistoryRecordControlUpper(members)
	if !plannedOK || !envelopeOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	const metadata uint64 = candidateLedgerMetadataLimit
	parseSuffix := a.sum(a.product(4, metadata), 3)
	settlementBody := a.sum(a.product(metadata, metadata), 1, metadata, 1,
		metadata, uint64(len(providerHistoryGroupSettlementDomain)), 2)
	out := providerHistoryPlannedRecordControlCost{envelope: envelope}
	out.normal.ticket, out.normal.consumption = planned.normal.ticket, planned.normal.consumption
	out.normal.ticket[10] = a.sum(out.normal.ticket[10], parseSuffix)
	out.normal.consumption[10] = a.sum(out.normal.consumption[10], parseSuffix)
	// Context includes T directly and AGAIN inside complete C validation.
	for coordinate := range out.normal.context {
		out.normal.context[coordinate] = a.sum(planned.normal.ticket[coordinate], planned.normal.consumption[coordinate])
	}
	out.normal.settlement = out.normal.context
	out.normal.settlement[10] = a.sum(out.normal.settlement[10], settlementBody, parseSuffix)
	if a.failed {
		return empty, false
	}
	return out, true
}

// Completed original diagnostic TLS DATA, not live TLS verification, transport
// origin evidence or a source witness. Both TLS rows already contain ONE full
// result-context validation. The separate context pair is only for other real
// call sites, never a second charge for the same TLS-internal validation.
type providerHistoryPlannedTLSControlCost struct {
	original, parsed, context providerHistoryControlPair
	normalBytes               uint64
}

func providerHistoryPlannedTLSControlUpper(members uint64, certificates providerTLSBounds) (providerHistoryPlannedTLSControlCost, bool) {
	var empty providerHistoryPlannedTLSControlCost
	record, recordOK := providerHistoryPlannedRecordControlUpper(members)
	envelope, envelopeOK := providerHistoryTLSControlUpper(members, certificates)
	if !recordOK || !envelopeOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	digests := a.product(uint64(certificates.chains), uint64(certificates.certificates))
	// The same original whole-result-pack guard reserves2048 fixed bytes for
	// all field names, punctuation, hashes66, stage20, ordinal2, official
	// endpoint/server and version/cipher strings. Each digest needs <=70bytes
	// and each chain <=4 punctuation bytes. Successful canonical equality
	// excludes extra/escaped/unbounded fields. Failed/raw work still keeps E.
	metadata := a.sum(2048, a.product(70, digests), a.product(4, uint64(certificates.chains)))
	quadratic := a.sum(a.product(metadata, metadata), 1)
	requestMatch := a.sum(a.product(3, 756), 4)
	out := providerHistoryPlannedTLSControlCost{
		original:    providerHistoryControlPair{record.normal.context, envelope.original},
		parsed:      providerHistoryControlPair{record.normal.context, envelope.parsed},
		context:     providerHistoryControlPair{record.normal.context, record.envelope.context},
		normalBytes: metadata,
	}
	out.original.normal[10] = a.sum(out.original.normal[10], requestMatch, a.product(67, digests), quadratic, metadata, 1)
	out.parsed.normal[2] = a.sum(out.parsed.normal[2], metadata)
	out.parsed.normal[10] = a.sum(out.parsed.normal[10], a.product(2, quadratic), a.product(3, metadata), 2)
	if a.failed || metadata > 64<<10 {
		return empty, false
	}
	return out, true
}

// Standalone contexts here belong to a result-pack parser or failure-original
// constructor/parser, not the contexts already INSIDE original/parsed TLS.
// Each failure parser validates context in its guard and AGAIN in its rebuilt
// constructor. The failed S writer plus two readbacks therefore have FIVE
// current failure contexts. Their format/raw/native costs remain separate K.
type providerHistoryRoleResultControlCalls struct{ originals, parsed, contexts uint64 }

type providerHistoryRoleResultControlCounts struct {
	before, preparation, member, successClosure, failedClosure, advance providerHistoryRoleResultControlCalls
}

func providerHistoryCollectionRoleResultControlCallsBound(prior, pastMembers, members, ordinal uint64, advancing bool) (providerHistoryRoleResultControlCounts, bool) {
	var empty providerHistoryRoleResultControlCounts
	const maximumGroups uint64 = (1 << 20) / 3
	if prior >= maximumGroups || advancing && prior+1 >= maximumGroups ||
		members == 0 || members > providerHistoryGroupMembers || ordinal >= members {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	maximumPast := a.product(prior, providerHistoryGroupMembers)
	if a.failed || pastMembers < prior || pastMembers > maximumPast {
		return empty, false
	}
	// A completed prior prefix has one results pack per prior S and one TLS
	// parser per prior member. It constructs no new live TLS original.
	preparationPacks, failedPacks := uint64(0), uint64(0)
	if ordinal != 0 {
		preparationPacks, failedPacks = 2, 3
	}
	out := providerHistoryRoleResultControlCounts{
		before:         providerHistoryRoleResultControlCalls{0, a.product(9, pastMembers), a.product(9, prior)},
		preparation:    providerHistoryRoleResultControlCalls{0, a.product(2, ordinal), preparationPacks},
		member:         providerHistoryRoleResultControlCalls{1, a.sum(a.product(4, ordinal), 2), a.sum(preparationPacks, 2)},
		successClosure: providerHistoryRoleResultControlCalls{0, a.product(3, a.sum(pastMembers, members)), a.sum(a.product(3, prior), 3)},
		failedClosure:  providerHistoryRoleResultControlCalls{0, a.product(3, a.sum(pastMembers, ordinal)), a.sum(a.product(3, prior), failedPacks, 5)},
	}
	if advancing {
		out.advance = providerHistoryRoleResultControlCalls{0, a.product(2, a.sum(pastMembers, members)), a.product(2, a.sum(prior, 1))}
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

func (c providerHistoryRoleResultControlCalls) upper(tls providerHistoryPlannedTLSControlCost) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	rows := [3]struct {
		calls uint64
		cost  providerHistoryControlPair
	}{{c.originals, tls.original}, {c.parsed, tls.parsed}, {c.contexts, tls.context}}
	var a providerHistoryReadArithmetic
	var out candidateWorkVector
	for _, row := range rows {
		if row.calls == 0 {
			continue // source-proven absence of THIS independent component only
		}
		work, ok := row.cost.pointwiseUpper(row.calls)
		if !ok {
			return empty, false
		}
		for coordinate := range out {
			out[coordinate] = a.sum(out[coordinate], work[coordinate])
		}
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

// Uniform prospective per-role bounds: every earlier T/C has at most32 members,
// and every current ordinal is <=members-1. ALL selected bounds and original
// planned/owned fences must be bound by the future source factory before H0.
// Numeric cardinalities alone cannot supply those facts or lower any incoming
// raw/preflight guard. M includes P; the unchanged literal remaining guard still
// retains P AGAIN as separate already-entered preparation in each full T upper.
//
// Success and failed closure are priced independently, including the five
// standalone failure contexts, then coordinate-MAXed. Final A is absent. I has
// no calls in this component. TLS/context work is disjoint from prior standalone
// C/S and ledger/admission costs. HTTP/RPC/result-pack/failure body, hashes,
// capture/native CPU/backing RAM, all other incoming/deferred/final/recovery K
// and complete full-stage/resource/window/factory qualification remain separate.
type providerHistoryRoleResultControlParts struct {
	initial, before, preparation, member, closure, advance candidateWorkVector
}

func providerHistoryCollectionRoleResultControlPartsUpper(prior, members uint64, certificates providerTLSBounds, advancing bool) (providerHistoryRoleResultControlParts, bool) {
	var empty providerHistoryRoleResultControlParts
	if members == 0 || members > providerHistoryGroupMembers {
		return empty, false
	}
	past, pastOK := providerUpperProduct(prior, providerHistoryGroupMembers)
	calls, callsOK := providerHistoryCollectionRoleResultControlCallsBound(prior, past, members, members-1, advancing)
	tls, tlsOK := providerHistoryPlannedTLSControlUpper(providerHistoryGroupMembers, certificates)
	if !pastOK || !callsOK || !tlsOK {
		return empty, false
	}
	var parts [6]candidateWorkVector
	for index, role := range [6]providerHistoryRoleResultControlCalls{
		calls.before, calls.preparation, calls.member, calls.successClosure, calls.failedClosure, calls.advance,
	} {
		if index == 5 && !advancing {
			continue
		}
		work, ok := role.upper(tls)
		if !ok {
			return empty, false
		}
		parts[index] = work
	}
	out := providerHistoryRoleResultControlParts{before: parts[0], preparation: parts[1], member: parts[2], advance: parts[5]}
	for coordinate := range out.closure {
		out.closure[coordinate] = max(parts[3][coordinate], parts[4][coordinate])
	}
	return out, true
}
