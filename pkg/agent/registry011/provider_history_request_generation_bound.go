package registry011

// Request generation BEFORE nextTable's final table construction. All eight
// phases retain their closed builders; the class/record/table component is
// separate. Numeric DATA does not bind a cursor, source or operating profile.
func providerHistoryRequestGenerationUpper(members, effects uint64) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	control, ok := providerHistoryLedgerControlUpper(members)
	if !ok || effects > 10*16384 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	// Reuse the larger complete request-copy row for the same constructor.
	// Its params codec/normalization/equality are conservatively overreserved.
	// Each generated request also appends one observed list member.
	// Number tags encode uint64 then copy at most18 bytes; trace options have
	// one encode8 event. Direct spine lookup compares both66-byte hashes.
	tag := a.sum(8*8+1, 18, 1)
	options, spine := uint64(8*8+1), uint64(2*66+1)
	// rpcAgent is at most64 ASCII bytes. historyItem's two-word head is
	// largest: selector4 +head64 +length32 +paddedAgent64 =164 bytes.
	// Observe selector hash, calldata copy/encode and hex-string copy. Native
	// Keccak, rpcWord, hex/UTF8/format CPU and backing RAM remain separate.
	const data uint64 = 4 + 64 + 32 + 64
	registry := a.sum(uint64(len("historyItem(string,uint64)")), 1, data, 1,
		a.product(data, data), 1, 2, a.product(2, data), 1)
	member := a.sum(control.requestCopy[10], 2, tag, options, spine, registry)
	var work candidateWorkVector
	work[2] = a.product(members, control.requestCopy[2])
	// Effects selection scans ALL pending effects, even when only its last
	// member remains to request. Each has one spine and one block-hash check.
	work[10] = a.sum(a.product(members, member), a.product(effects, 266))
	if a.failed {
		return empty, false
	}
	return work, true
}

// Every original nextTable invocation comprises planning, admit and S-boundary
// selection, but excludes request-pack table reconstruction. Those differences
// are exactly16G+28J calls and16R+28E generated members. Final table construction
// is ALREADY priced in request control and is not added here a second time.
func providerHistoryCollectionRepeatedRequestGenerationUpper(groups, requests, effects uint64) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	calls, callsOK := providerHistoryCollectionRequestControlCallsBound(groups, requests)
	member, memberOK := providerHistoryRequestGenerationUpper(1, 0)
	scan, scanOK := providerHistoryRequestGenerationUpper(1, effects)
	if !callsOK || !memberOK || !scanOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	selections := a.difference(calls.tableBuilds, calls.packParses)
	generated := a.difference(calls.tableMembers, calls.originals)
	var work candidateWorkVector
	for coordinate := range work {
		base := a.difference(scan[coordinate], member[coordinate])
		work[coordinate] = a.sum(a.product(selections, base), a.product(generated, member[coordinate]))
	}
	if a.failed {
		return empty, false
	}
	return work, true
}

// One disjoint explicit-event component. A completed original operation must
// pass its full normal fence; the first unfinished operation retains the whole
// original envelope, and its caller must abort before a later operation starts.
// A numeric pair is not proof of those source postconditions or permission to
// narrow any incoming preflight. Every admitted role keeps its OWN envelope.
type providerHistoryControlPair struct{ normal, envelope candidateWorkVector }

func (p providerHistoryControlPair) pointwiseUpper(calls uint64) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	if calls == 0 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out candidateWorkVector
	for coordinate, normal := range p.normal {
		if normal > p.envelope[coordinate] {
			return empty, false
		}
		out[coordinate] = a.sum(a.product(calls-1, normal), p.envelope[coordinate])
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

// This nextTable component includes request GENERATION and its final table
// construction, but not the separately reconstructed table inside a pack/T
// validation. The equality component covers only equality outside the already
// priced admission atom. All historic states must have the original selected
// effects/maximum bounds before H0. No reported count or private table proves
// that selection. Arbitrary raw decoding retains its old full envelope.
type providerHistoryPlannedNextTableControlCost struct {
	generated, equality providerHistoryControlPair
}

func providerHistoryPlannedNextTableControlUpper(members, effects, maximum uint64) (providerHistoryPlannedNextTableControlCost, bool) {
	var empty providerHistoryPlannedNextTableControlCost
	planned, plannedOK := providerHistoryPlannedLedgerControlUpper(members)
	request, requestOK := providerHistoryRequestControlUpper(members, maximum)
	generation, generationOK := providerHistoryRequestGenerationUpper(members, effects)
	if !plannedOK || !requestOK || !generationOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryPlannedNextTableControlCost
	for coordinate := range generation {
		// Replace ONLY successful generated request-copy work. Keep the same
		// original tags/options/spine/ABI/pending-effect scan overhead. The
		// unchanged complete generation row remains in the envelope.
		overhead := a.difference(generation[coordinate], a.product(members, planned.envelope.requestCopy[coordinate]))
		normalGeneration := a.sum(overhead, a.product(members, planned.normal.requestCopy[coordinate]))
		tableMembers := a.product(members, a.sum(planned.normal.requestCopy[coordinate], a.product(2, planned.normal.requestClass[coordinate])))
		out.generated.normal[coordinate] = a.sum(normalGeneration, tableMembers)
		out.generated.envelope[coordinate] = a.sum(generation[coordinate], request.tableBuilt[coordinate])
	}
	// Exactly the same generated planned-table shape as the closed admission
	// atom, including both capture-class decodes. That atom's OWN equality is
	// not added here; equality below is priced only at residual source sites.
	table := planned.tableBytes
	out.generated.normal[10] = a.sum(out.generated.normal[10], a.product(table, table), 1,
		table, 1, table, uint64(len(providerHistoryGroupTableDomain)), 2)
	const params, body, head, wire uint64 = 500, 578, 178, 756
	requestEqual := a.sum(2*32+1, 2*params+1, 2*body+1, 2*head+1, 2*wire+1, 2*133)
	out.equality.normal[10] = a.sum(a.product(2, table), 1, 133, a.product(providerHistoryGroupMembers, requestEqual))
	out.equality.envelope = request.tableEqual
	if a.failed {
		return empty, false
	}
	return out, true
}

// Source counts OUTSIDE the prior ledger/admission-control components and
// OUTSIDE TLS/result-context validations. All ticket constructor/parser and
// transition-validator calls are already paired inside admission atoms.
// Hash-binding comparisons, native work and boundary application remain K.
type providerHistoryRoleRequestControlCalls struct {
	nextTables, tableEquals, consumptions, settlements uint64
}

type providerHistoryRoleRequestControlCounts struct {
	before, successClosure, failedClosure, advance providerHistoryRoleRequestControlCalls
}

func providerHistoryCollectionRoleRequestControlCallsBound(prior uint64, advancing bool) (providerHistoryRoleRequestControlCounts, bool) {
	var empty providerHistoryRoleRequestControlCounts
	const maximumGroups uint64 = (1 << 20) / 3
	if prior >= maximumGroups || advancing && prior+1 >= maximumGroups {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	out := providerHistoryRoleRequestControlCounts{
		before: providerHistoryRoleRequestControlCalls{
			a.sum(10, a.product(18, prior)), a.sum(1, a.product(9, prior)),
			a.sum(10, a.product(18, prior)), a.product(18, prior)},
		successClosure: providerHistoryRoleRequestControlCalls{
			a.sum(6, a.product(6, prior)), a.sum(3, a.product(3, prior)),
			a.sum(6, a.product(6, prior)), a.sum(6, a.product(6, prior))},
		failedClosure: providerHistoryRoleRequestControlCalls{
			a.sum(3, a.product(6, prior)), a.product(3, prior),
			a.sum(6, a.product(6, prior)), a.sum(6, a.product(6, prior))},
	}
	if advancing {
		out.advance = providerHistoryRoleRequestControlCalls{
			a.sum(4, a.product(4, prior)), a.sum(2, a.product(2, prior)),
			a.sum(4, a.product(4, prior)), a.sum(4, a.product(4, prior))}
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

func (c providerHistoryRoleRequestControlCalls) upper(next providerHistoryPlannedNextTableControlCost, record providerHistoryPlannedRecordControlCost) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	rows := [4]struct {
		calls uint64
		cost  providerHistoryControlPair
	}{
		{c.nextTables, next.generated}, {c.tableEquals, next.equality},
		{c.consumptions, providerHistoryControlPair{record.normal.consumption, record.envelope.consumption}},
		{c.settlements, providerHistoryControlPair{record.normal.settlement, record.envelope.settlement}},
	}
	var a providerHistoryReadArithmetic
	var out candidateWorkVector
	for _, row := range rows {
		if row.calls == 0 {
			// Only a source-proven absence skips its independent envelope.
			continue
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

// Disjoint per-role explicit-event work, not a complete row or source witness.
// I/P/M have no calls in THIS component, not zero complete work. New tables in
// S boundary construction precede state advancement; own A therefore uses the
// old successful T and states0..prior, never the after-state's next table.
// The uniform32 rows require original selected bounds at EVERY historic state.
// Success and known-failed S are priced separately and then coordinate-MAXed.
// The factory must still compose all other K/incoming/deferred work, clocks,
// native CPU/backing peak RAM, windows/failure/final/recovery and full stages.
type providerHistoryRoleRequestControlParts struct {
	initial, before, preparation, member, closure, advance candidateWorkVector
}

func providerHistoryCollectionRoleRequestControlPartsUpper(prior, effects, maximum uint64, advancing bool) (providerHistoryRoleRequestControlParts, bool) {
	var empty providerHistoryRoleRequestControlParts
	calls, callsOK := providerHistoryCollectionRoleRequestControlCallsBound(prior, advancing)
	next, nextOK := providerHistoryPlannedNextTableControlUpper(providerHistoryGroupMembers, effects, maximum)
	record, recordOK := providerHistoryPlannedRecordControlUpper(providerHistoryGroupMembers)
	if !callsOK || !nextOK || !recordOK {
		return empty, false
	}
	before, beforeOK := calls.before.upper(next, record)
	success, successOK := calls.successClosure.upper(next, record)
	failed, failedOK := calls.failedClosure.upper(next, record)
	if !beforeOK || !successOK || !failedOK {
		return empty, false
	}
	out := providerHistoryRoleRequestControlParts{before: before}
	for coordinate := range out.closure {
		out.closure[coordinate] = max(success[coordinate], failed[coordinate])
	}
	if advancing {
		advance, ok := calls.advance.upper(next, record)
		if !ok {
			return empty, false
		}
		out.advance = advance
	}
	return out, true
}
