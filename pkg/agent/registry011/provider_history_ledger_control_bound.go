package registry011

// Private explicit 2/10 event costs for the original closed request/record
// routes. A ledger row excludes its cursor commitment, which is composed
// separately. Native JSON/format/hash CPU, allocation/backing RAM, all other
// T/C/S constructors/parsers, failure/final and factory authority are separate.
type providerHistoryLedgerControlCost struct {
	requestCopy, requestClass, ticket, consumption, ledger candidateWorkVector
}

func providerHistoryLedgerControlUpper(members uint64) (providerHistoryLedgerControlCost, bool) {
	var empty providerHistoryLedgerControlCost
	if members == 0 || members > providerHistoryGroupMembers {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	quadratic := func(n uint64) uint64 { return a.sum(a.product(n, n), 1) }
	const params, body, head, metadata uint64 = 8192, 16384, 512, 8192
	// The reconstructed body and head exist BEFORE equality to input.wire.
	// Their intermediate concatenation can be larger than the input's16KiB.
	wire := a.sum(body, head)
	// All eleven selected method names fit32bytes; fixed classes fit9bytes.
	// Preserve each of the seven independent reconstructed-field comparisons.
	equal := a.sum(2*32+1, 2*params+1, 2*body+1, 2*head+1, wire, body, 1, 2*133)
	// Number normalization precedes the builder's512-node/depth guard. Its
	// decoded finite JSON tree has at most8192nodes from the raw-byte cap.
	// The closed builder can enter its513th compare before refusing.
	walk := a.sum(2*513, params, 513)
	requestUnits := a.sum(params, 1, quadratic(params), a.product(2, params), walk,
		quadratic(params), quadratic(body), quadratic(head), head, 1, body, 1, wire, 1,
		params, 1, body, 1, head, 1, wire, 1, params, uint64(len("sage-provider-params-v1")), 2, equal)
	classUnits := a.sum(params, 1, quadratic(params))
	memberUnits := a.sum(requestUnits, classUnits, equal, 19)
	const table uint64 = providerHistoryGroupTableBytes
	tableUnits := a.sum(quadratic(table), table, 1, 2*table, 1,
		table, uint64(len(providerHistoryGroupTableDomain)), 2, 133)
	ticketUnits := a.sum(a.product(members, memberUnits), tableUnits, quadratic(metadata), metadata, 1,
		metadata, uint64(len(providerHistoryGroupTicketDomain)), 2, 2*metadata, 1, 133)
	ticketDecoded := a.product(members, 2*params)
	// Consumption validation reconstructs a ticket and then its own record.
	consumptionBody := a.sum(quadratic(metadata), metadata, 1,
		metadata, uint64(len(providerHistoryGroupConsumptionDomain)), 2, 133, 2*metadata, 1, 133)
	ledgerEncoding := a.sum(quadratic(metadata), metadata, uint64(len(providerHistoryGroupRootDomain)), 2, 133)
	// A consumed ledger validates its ticket directly and AGAIN inside C.
	// Reserve that complete control row for every ledger-root invocation;
	// constructor/advance/prepared/settled branches need a subset of it.
	ledgerUnits := a.sum(ledgerEncoding, a.product(2, ticketUnits), consumptionBody, 133)
	var out providerHistoryLedgerControlCost
	out.requestCopy[2], out.requestCopy[10] = params, requestUnits
	out.requestClass[2], out.requestClass[10] = params, classUnits
	out.ticket[2], out.ticket[10] = ticketDecoded, ticketUnits
	out.consumption[2], out.consumption[10] = ticketDecoded, a.sum(ticketUnits, consumptionBody)
	out.ledger[2], out.ledger[10] = a.product(2, ticketDecoded), ledgerUnits
	if a.failed {
		return empty, false
	}
	return out, true
}

// Covers the control portion associated with all original successful ledger
// roots, conservatively at32members. Cursor work and ticket/consumption/S
// construction/parsing OUTSIDE ledger validation are not funded by this row.
// Prospective G/R must still be bound by the original production compiler.
func providerHistoryCollectionRepeatedLedgerControlUpper(groups, requests uint64) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	calls, callsOK := providerHistoryCollectionRootCallsBound(groups, requests)
	one, oneOK := providerHistoryLedgerControlUpper(providerHistoryGroupMembers)
	if !callsOK || !oneOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var work candidateWorkVector
	for coordinate, units := range one.ledger {
		work[coordinate] = a.product(calls.ledgerRoots, units)
	}
	if a.failed {
		return empty, false
	}
	return work, true
}

// Successful entered work for the exact original collection request grammar,
// plus unchanged original one-call control envelopes. A private request/table
// does not establish that grammar. BOTH original readers must independently
// reconstruct and compare the exact next-table before these normal parts can
// be used by a future complete cost derivation. This is not such a derivation.
// Normalization of arbitrary raw JSON can precede the constructor's node guard;
// it retains the wider original envelope, never the small planned-shape row.
// Cursor work, repeated/composite scopes, native CPU/RAM and authority remain
// separate. No original preflight, quota, guard or standalone result changes.
type providerHistoryPlannedLedgerControlCost struct {
	normal, envelope providerHistoryLedgerControlCost
	tableBytes       uint64
}

func providerHistoryPlannedLedgerControlUpper(members uint64) (providerHistoryPlannedLedgerControlCost, bool) {
	var empty providerHistoryPlannedLedgerControlCost
	envelope, ok := providerHistoryLedgerControlUpper(members)
	if !ok {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	quadratic := func(n uint64) uint64 { return a.sum(a.product(n, n), 1) }
	// Original templates use generated text without JSON escapes. Duration
	// text can contain UTF-8 µs; its32-byte bound counts bytes, not runes. The
	// largest params are the164-byte history calldata's330-byte hex string,
	// address42 and pinned block66. The method is conservatively32 bytes.
	// Direct and decoded template trees both fit32 visited nodes; map keys
	// are counted by the builder. Duration text is over-reserved at32 bytes.
	const params, body, head, wire uint64 = 500, 578, 178, 756
	equal := a.sum(2*32+1, 2*params+1, 2*body+1, 2*head+1, 2*wire+1, 2*133)
	walk := a.sum(4*32, params, 32) // normalization + builder + string events
	request := a.sum(params, 1, quadratic(params), walk,
		quadratic(params), quadratic(body), quadratic(head), head, 1, body, 1, wire, 1,
		params, 1, body, 1, head, 1, wire, 1, params, uint64(len("sage-provider-params-v1")), 2, equal)
	class := a.sum(params, 1, quadratic(params))
	member := a.sum(request, class, equal, 19)
	// Table objects contain request/params DIGESTS, not raw params or bodies.
	// A member is at most419 bytes;185 includes outer fields/punctuation and
	// the adjustment for the first member's missing preceding comma. Capture
	// numeric strings20 and ordinal2 conservatively exceed their valid widths.
	table := a.sum(185, a.product(420, members))
	tableUnits := a.sum(quadratic(table), table, 1, 2*table, 1,
		table, uint64(len(providerHistoryGroupTableDomain)), 2, 133)
	const ticketBytes, consumptionBytes, ledgerBytes uint64 = 1647, 505, 2270
	ticket := a.sum(a.product(members, member), tableUnits, quadratic(ticketBytes), ticketBytes, 1,
		ticketBytes, uint64(len(providerHistoryGroupTicketDomain)), 2, 2*ticketBytes, 1, 133)
	consumption := a.sum(quadratic(consumptionBytes), consumptionBytes, 1,
		consumptionBytes, uint64(len(providerHistoryGroupConsumptionDomain)), 2, 133, 2*consumptionBytes, 1, 133)
	ledger := a.sum(quadratic(ledgerBytes), ledgerBytes, uint64(len(providerHistoryGroupRootDomain)), 2, 133,
		a.product(2, ticket), consumption, 133)
	out := providerHistoryPlannedLedgerControlCost{envelope: envelope, tableBytes: table}
	out.normal.requestCopy = candidateWorkVector{2: params, 10: request}
	out.normal.requestClass = candidateWorkVector{2: params, 10: class}
	out.normal.ticket = candidateWorkVector{2: a.product(members, 2*params), 10: ticket}
	out.normal.consumption = candidateWorkVector{2: a.product(members, 2*params), 10: a.sum(ticket, consumption)}
	out.normal.ledger = candidateWorkVector{2: a.product(members, 4*params), 10: ledger}
	if a.failed {
		return empty, false
	}
	return out, true
}

// Ledger-control component for one original actual-accounting scope. Every
// internal owned ticket/C must already have passed the exact original next-
// table comparison before installation. External raw parsing and incoming
// T/C/S construction/validation retain their separate complete envelopes.
// This numeric count authenticates no source, role, state or original meter.
//
// Successful calls use the uniform planned32 normal; a first refused control
// call uses the unchanged complete32 envelope, including internal siblings.
// The original caller must stop later independent calls after that refusal.
// Cursor work, nonroot cleanup and native/backing RAM remain separate.
// Each I/T reservation still includes its OWN complete component: this row
// cannot replace SUM of full admitted uppers with one whole-actual headroom.
func providerHistoryCollectionPointwiseLedgerControlUpper(ledgerRoots uint64) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	if ledgerRoots == 0 {
		return empty, false
	}
	one, ok := providerHistoryPlannedLedgerControlUpper(providerHistoryGroupMembers)
	if !ok {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var work candidateWorkVector
	for coordinate, normal := range one.normal.ledger {
		envelope := one.envelope.ledger[coordinate]
		if normal > envelope {
			return empty, false
		}
		work[coordinate] = a.sum(a.product(ledgerRoots-1, normal), envelope)
	}
	if a.failed {
		return empty, false
	}
	return work, true
}

// These are disjoint CONTROL components plus the original owned-clock
// event/counts, not full child rows, a source witness or execution authority.
// Cursor/state roots, copies, semantic work, raw/file/native/CPU/time/RAM and
// deferred/failure/final/recovery costs remain separate source obligations.
type providerHistoryRoleControlParts struct {
	initial, before, preparation, member, closure, advance providerHistoryStageWorkCost
}

// Price one conditional role from source-derived invocation counts. A zero
// admission count means the caller proved no flat admission leaf in this
// component, not that an untrusted caller can skip a required fence. Ledger
// roots still require a nonempty original scope. Numeric DATA never proves
// the source counts, completed normal fences or first-failure closure.
func providerHistoryRoleControlPartUpper(ledgerRoots, admissionAtoms, maximum, clocks uint64) (providerHistoryStageWorkCost, bool) {
	var empty providerHistoryStageWorkCost
	ledger, ok := providerHistoryCollectionPointwiseLedgerControlUpper(ledgerRoots)
	if !ok {
		return empty, false
	}
	var admission candidateWorkVector
	if admissionAtoms != 0 {
		admission, ok = providerHistoryCollectionPointwiseAdmissionClusterControlUpper(admissionAtoms, maximum)
		if !ok {
			return empty, false
		}
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryStageWorkCost
	for coordinate := range out.work {
		out.work[coordinate] = a.sum(ledger[coordinate], admission[coordinate])
	}
	// The original closed providerClockEvent(0) has logical weight ONE. Its
	// separate counter is not an elapsed-time or native CPU measurement.
	out.work[10] = a.sum(out.work[10], clocks)
	out.clocks = clocks
	if a.failed {
		return empty, false
	}
	return out, true
}

// Split one original stage at the actual B/P/M/C/A source cuts. P and M have
// the SAME ledger-control roots, all reached during preparation before the
// remaining preflight; M has one additional owned end sample. Their other
// work is NOT equal: the transport/retention and all remaining K are separate.
// Successful and recognized failed S have the same ledger/admission CONTROL
// counts, while their distinct cursor/state-root weights stay separate.
//
// Both pointwise children retain their OWN full envelope in every role. Sum
// their disjoint components conservatively; never reuse one global headroom
// to replace the sum of full admitted stages. The complete compiler must bind
// these counts and maximum to the original immutable profile and retain all
// child sublimits. Source-count clocks2/2/2 require exact pre-H0 selection;
// they never override wider existing dispatch.remaining clock allowances.
// Any wider original incoming row must still be fully funded. No production
// caller or derivation factory is supplied.
func providerHistoryCollectionRoleControlPartsUpper(prior, maximum uint64, advancing bool) (providerHistoryRoleControlParts, bool) {
	var empty providerHistoryRoleControlParts
	const maximumGroups uint64 = (1 << 20) / 3
	if prior >= maximumGroups || advancing && prior+1 >= maximumGroups {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	calls := [6]struct{ ledger, admission, clocks uint64 }{
		{2, 0, 2},
		{a.sum(66, a.product(63, prior)), a.sum(8, a.product(9, prior)), 6},
		{10, 0, 1},
		{10, 0, 2},
		{a.sum(36, a.product(21, prior)), a.sum(3, a.product(3, prior)), 2},
		{a.sum(20, a.product(14, prior)), a.sum(2, a.product(2, prior)), 2},
	}
	if a.failed {
		return empty, false
	}
	var parts [6]providerHistoryStageWorkCost
	for index, row := range calls {
		if index == 5 && !advancing {
			continue
		}
		part, ok := providerHistoryRoleControlPartUpper(row.ledger, row.admission, maximum, row.clocks)
		if !ok {
			return empty, false
		}
		parts[index] = part
	}
	return providerHistoryRoleControlParts{initial: parts[0], before: parts[1], preparation: parts[2], member: parts[3], closure: parts[4], advance: parts[5]}, true
}
