package registry011

// Base64's observer decodes ENCODED input bytes, then encodes decoded raw
// bytes and compares the exact unpadded spelling. Numeric DATA is not a parser
// result or authority. The original maximum is at most8MiB.
func providerHistoryBase64ControlUpper(maximum uint64) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	if maximum == 0 || maximum > 8<<20 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	tail := [...]uint64{0, 2, 3}
	encoded := a.sum(a.product(maximum/3, 4), tail[maximum%3])
	var work candidateWorkVector
	work[2] = encoded
	work[10] = a.sum(a.product(encoded, encoded), 1, a.product(maximum, maximum), 1, 2*encoded+1)
	if a.failed {
		return empty, false
	}
	return work, true
}

// Successful first-P field/token parsing, not finality authentication. Six
// bounded raw/field codec rows conservatively cover the header parser; both
// canonical uint64 quantities separately keep their16-digit codec and encoder
// preflights. Native decoder/UTF8/token-map/hex costs remain separate.
func providerHistoryFirstPHeaderControlUpper(maximum uint64) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	if maximum == 0 || maximum > 8<<20 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var work candidateWorkVector
	work[2] = a.sum(a.product(6, maximum), 32)
	work[10] = a.sum(a.product(6, a.sum(a.product(maximum, maximum), 1)), a.product(2, maximum), 4*(16*16+1), 275)
	if a.failed {
		return empty, false
	}
	return work, true
}

type providerHistoryFirstPInterpretationCost struct {
	records, base64, http, rpc, header, comparisons candidateWorkVector
	initialRoot, request, total                     candidateWorkVector
}

// The selected first-P interpretation only. Original file reads, legacy
// load/object lookups/control, cursor+semantic constructor, general callers,
// native CPU/RAM/all-eleven/failure/final/factory are separate. HTTP parsing
// here is in-memory replay, not a network request or another retained read.
func providerHistoryFirstPInterpretationUpper(objectBytes uint64, capture ProviderCaptureLimits) (providerHistoryFirstPInterpretationCost, bool) {
	var empty providerHistoryFirstPInterpretationCost
	if objectBytes == 0 || objectBytes > 32<<20 || !capture.valid() {
		return empty, false
	}
	response, responseOK := providerHistoryResponseControlUpper(capture)
	control, controlOK := providerHistoryLedgerControlUpper(1)
	request64, requestOK := providerHistoryBase64ControlUpper(16 << 10)
	response64, response64OK := providerHistoryBase64ControlUpper(uint64(capture.BodyBytes))
	header, headerOK := providerHistoryFirstPHeaderControlUpper(uint64(capture.BodyBytes))
	if !responseOK || !controlOK || !requestOK || !response64OK || !headerOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	q := func(n uint64) uint64 { return a.sum(a.product(n, n), 1) }
	const l, z, request uint64 = 64 << 10, 1 << 20, 16 << 10
	record := min(objectBytes, l)
	body, headers := uint64(capture.BodyBytes), uint64(capture.HeaderBytes)
	// Policy, enrollment, admission, T/C/S, extra admission fields, exchange,
	// wire-binding and the separately reparsed policy configuration. Policy
	// raw comes from the fixed config, independently of objectBytes. Ordinary
	// limits/segment Marshal retains its existing64KiB/4096 byte preflight.
	domains := uint64(len("sage-provider-policy-v1") + len("sage-registry-instance-v1") + len("sage-provider-origin-v1") +
		len("sage-provider-enrollment-v2") + len("sage-provider-admission-intent-v1") + len("sage-provider-budget-ticket-v2") +
		len("sage-provider-budget-consumption-v1") + len("sage-provider-budget-settlement-v2") + len("sage-provider-budget-segment-v1"))
	var out providerHistoryFirstPInterpretationCost
	out.records[2] = a.sum(6*l, a.product(6, record), a.product(2, objectBytes))
	out.records[10] = a.sum(a.product(9, q(l)), a.product(6, q(record)), a.product(2, q(objectBytes)), q(4096),
		11*l, a.product(10, record), 4096, 31, domains)
	for c := range out.base64 {
		out.base64[c] = a.sum(request64[c], response64[c])
	}
	out.http, out.header = response.http, header
	// readProviderHTTPObserved already parses the envelope once. The caller
	// ignores that result and independently decodes its original body again.
	out.rpc[2] = a.product(3, body)
	out.rpc[10] = a.sum(a.product(3, q(body)), a.product(3, body), 3)
	// Three expected digest checks; expected request wire copy+comparison,
	// params copy/hash/comparison, request-body comparison and the independent
	// parsed-HTTP-body versus exchange-response comparison. Transport and
	// both header encoders keep original1MiB preflights, not smaller raw caps.
	out.comparisons[10] = a.sum(3*133, 5*request+3, 4*8192+3, a.product(2, body), 1,
		a.product(3, q(z)), 5*z, min(headers/4, 128), 4)
	// Two selected checks: before initial-root creation and inside it. The
	// third selected check/copy belongs to newCursor and is NOT included here.
	out.initialRoot[10] = a.sum(q(candidateLedgerMetadataLimit), candidateLedgerMetadataLimit,
		uint64(len("sage-provider-history-initial-model-v1")), 2, 2*265)
	// The larger existing request-copy quote covers this fixed constructor.
	// It is separate from all nextTable generation calls and all12 lookups.
	out.request = control.requestCopy
	for c := range out.total {
		out.total[c] = a.sum(out.records[c], out.base64[c], out.http[c], out.rpc[c], out.header[c],
			out.comparisons[c], out.initialRoot[c], out.request[c])
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

func providerHistoryCollectionRepeatedFirstPInterpretationUpper(groups, requests, objectBytes uint64, capture ProviderCaptureLimits) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	calls, callsOK := providerHistoryCollectionRootCallsBound(groups, requests)
	one, oneOK := providerHistoryFirstPInterpretationUpper(objectBytes, capture)
	if !callsOK || !oneOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var work candidateWorkVector
	for c := range work {
		work[c] = a.product(calls.initialModels, one.total[c])
	}
	if a.failed {
		return empty, false
	}
	return work, true
}

// Boundary interpretation uses original result bytes from the earlier strict
// HTTP/RPC-envelope reader. That reader has already bounded the selected
// method's raw body and refused invalid UTF8, repaired surrogates and duplicate
// fields. Numeric DATA does not establish that producer route. In particular,
// a decoded short quantity or ABI value never narrows the raw JSON allowance.
// These pairs include ONE caller result-string copy, but no HTTP/envelope,
// native parser/crypto CPU, allocation/RAM, cursor, semantic or storage cost.
func providerHistoryBoundaryHeaderControlUpper(maximum uint64) (providerHistoryControlPair, bool) {
	var empty providerHistoryControlPair
	if maximum == 0 || maximum > 8<<20 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	square := a.product(maximum, maximum)
	var out providerHistoryControlPair
	// The four distinct RawMessage fields occupy disjoint original ranges:
	// sum(field lengths) <= raw, sum(field length squared) <= raw squared.
	// A successful canonical uint64 quantity contributes at most16 digits.
	out.normal[2] = a.sum(a.product(3, maximum), 32)
	out.normal[10] = a.sum(a.product(3, square), a.product(3, maximum), 1310)
	// BOTH quantities are called, even if the first one refuses. Their total
	// decoded field widths remain <= raw; a failed quantity's codec is not
	// restricted to16. Each encoder still keeps its original16 preflight.
	// No early small-width claim is used for an unsuccessful comparison.
	out.envelope[2] = max(out.normal[2], a.product(4, maximum))
	out.envelope[10] = max(out.normal[10], a.sum(a.product(4, square), a.product(5, maximum), 730))
	if a.failed {
		return empty, false
	}
	return out, true
}

// Original RPC data interpretation plus its caller's result copy. Unlike a
// header, this parser relies on the EARLIER strict envelope for UTF8 and unique
// JSON; arbitrary private raw strings without that fence are not this row.
func providerHistoryBoundaryDataControlUpper(maximum uint64) (providerHistoryControlPair, bool) {
	var empty providerHistoryControlPair
	if maximum == 0 || maximum > 8<<20 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var work candidateWorkVector
	work[2] = a.product(2, maximum)
	work[10] = a.sum(a.product(2, a.product(maximum, maximum)), a.product(2, maximum), 4)
	if a.failed {
		return empty, false
	}
	return providerHistoryControlPair{work, work}, true
}

// The four pinned Registry result forms only. Original rpcUintObserved reads
// a complete32-byte word (codec32), distinct from the replay ABI's word costs.
// Successful canonical ABI limits the inner hex text, NOT the outer JSON:
// whitespace and valid escapes still consume the full selected raw allowance.
// E retains full-size decoding before a bad tuple length/offset can refuse.
func providerHistoryBoundaryABIControlUpper(maximum uint64, hasBool bool, maxima ...uint64) (providerHistoryControlPair, bool) {
	var empty providerHistoryControlPair
	if maximum == 0 || maximum > 8<<20 || len(maxima) < 1 || len(maxima) > 2 || hasBool && len(maxima) != 1 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	fields, flag, values := uint64(len(maxima)), uint64(0), uint64(0)
	if hasBool {
		flag = 1
	}
	data := a.product(a.sum(a.product(2, fields), flag), 32)
	for _, limit := range maxima {
		if limit > 262144 {
			return empty, false
		}
		data = a.sum(data, a.product(a.sum(limit, 31)/32, 32))
		values = a.sum(values, limit)
	}
	hexBytes := min(maximum, a.product(2, data))
	textBytes := min(maximum, a.sum(a.product(2, data), 2))
	// Returned field copies cannot overlap because offset==cursor precedes
	// each copy. A partial failed tuple retains that same prefix property.
	copied := min(values, maximum/2)
	words := a.product(fields, 64)
	// Two codec32 words, one <=31-byte padding equality and one copy-event
	// overhead per field; the optional bool equality costs63 separately.
	tail := a.sum(a.product(fields, 2114), copied, a.product(flag, 63))
	square := a.product(maximum, maximum)
	var out providerHistoryControlPair
	out.normal[2] = a.sum(maximum, hexBytes, words)
	out.normal[10] = a.sum(square, a.product(hexBytes, hexBytes), maximum, textBytes, 4, tail)
	out.envelope[2] = a.sum(a.product(2, maximum), words)
	out.envelope[10] = a.sum(a.product(2, square), a.product(2, maximum), 4, tail)
	if a.failed {
		return empty, false
	}
	return out, true
}

// These are the RPC/result-interpretation sites of six original boundary
// phases. They are NOT complete boundary costs: begin/finish-window, content
// and effects replay, state/history semantics, copy/root/table, failure/raw
// control, native/backing/peak/lifetime RAM and the source factory are separate.
type providerHistoryBoundaryInterpretationCost struct {
	opening, ancestry, closing, state, grants, history providerHistoryControlPair
}

func providerHistoryBoundaryInterpretationUpper(methods providerMethodBounds, blocks uint64) (providerHistoryBoundaryInterpretationCost, bool) {
	var empty providerHistoryBoundaryInterpretationCost
	if !methods.valid() || blocks == 0 || blocks > 1<<20 {
		return empty, false
	}
	header, headerOK := providerHistoryBoundaryHeaderControlUpper(uint64(methods.header.body))
	code, codeOK := providerHistoryBoundaryDataControlUpper(uint64(methods.code.body))
	registry, registryOK := providerHistoryBoundaryABIControlUpper(uint64(methods.state.body), false, 128)
	state, stateOK := providerHistoryBoundaryABIControlUpper(uint64(methods.state.body), true, 65536)
	grants, grantsOK := providerHistoryBoundaryABIControlUpper(uint64(methods.state.body), false, 262144)
	history, historyOK := providerHistoryBoundaryABIControlUpper(uint64(methods.state.body), false, 262144, 262144)
	if !headerOK || !codeOK || !registryOK || !stateOK || !grantsOK || !historyOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	sum := func(parts ...providerHistoryControlPair) providerHistoryControlPair {
		var out providerHistoryControlPair
		for _, part := range parts {
			for coordinate := range out.normal {
				out.normal[coordinate] = a.sum(out.normal[coordinate], part.normal[coordinate])
				out.envelope[coordinate] = a.sum(out.envelope[coordinate], part.envelope[coordinate])
			}
		}
		return out
	}
	scale := func(part providerHistoryControlPair, count uint64) providerHistoryControlPair {
		var out providerHistoryControlPair
		for coordinate := range out.normal {
			out.normal[coordinate] = a.product(part.normal[coordinate], count)
			out.envelope[coordinate] = a.product(part.envelope[coordinate], count)
		}
		return out
	}
	logical := func(units uint64) providerHistoryControlPair {
		work := candidateWorkVector{10: units}
		return providerHistoryControlPair{work, work}
	}
	scalar := uint64(methods.scalar.body)
	square := a.product(scalar, scalar)
	chain := providerHistoryControlPair{
		normal: candidateWorkVector{2: scalar, 10: a.sum(square, scalar, 19)},
	}
	// chainID is compared before its value is accepted. E must retain the
	// raw-derived string width, even though successful Sepolia ID is8 bytes.
	chain.envelope = candidateWorkVector{2: scalar, 10: max(chain.normal[10], a.sum(square, a.product(2, scalar), 11))}
	clientWork := candidateWorkVector{2: scalar, 10: a.sum(square, scalar, 515)}
	client := providerHistoryControlPair{clientWork, clientWork}
	// The original constructor fixes the expected Registry URN. The value
	// limit128 and client limit256 precede their respective comparisons;
	// a prior client came from this same successful original version guard.
	registry = sum(registry, logical(a.sum(129, 128, uint64(len("eip155:11155111:"+SepoliaRegistryAddress)), 1)))
	identity := sum(chain, client, registry, logical(133)) // genesis hash
	// Code decoding can refuse before the hash, but the original loop still
	// calls Digest(nil). Include its hash row and the final64-byte equality.
	code = sum(code, logical(a.sum(uint64(methods.code.body)/2, 130)))
	var out providerHistoryBoundaryInterpretationCost
	out.opening = sum(identity, scale(header, 2), scale(code, 13))
	// Closing parses BOTH extra headers before checking either error. Its
	// spineHeader lookup independently compares two valid66-byte hashes.
	out.closing = sum(identity, scale(header, 4), logical(133))
	// A successful admitted ancestry group copies at most blocks spine slots;
	// every header has its own original-byte copy and parent comparison.
	out.ancestry = sum(scale(sum(header, logical(133)), providerHistoryGroupMembers), logical(a.sum(blocks, 1)))
	out.state, out.grants = state, grants
	out.history = scale(history, providerHistoryGroupMembers)
	if a.failed {
		return empty, false
	}
	return out, true
}

// Counts for the original closed collection grammar, not an admitted profile.
// Across all windows, ancestry has only blocks-1 header items and history has
// only versions items. A split window may add groups, never32 new items to
// every such group. All counts must bind the same immutable manifest before H0.
type providerHistoryBoundaryPhaseCaps struct {
	opening, closing, ancestry, content, effects, state, grants, history uint64
	ancestryItems, historyItems, groups                                  uint64
}

func providerHistoryCollectionBoundaryPhaseCapsBound(manifest *providerHistoryCollectionManifest) (providerHistoryBoundaryPhaseCaps, bool) {
	var empty providerHistoryBoundaryPhaseCaps
	if !manifest.validSnapshot() || manifest.input.blocks == 0 || manifest.input.blocks > 1<<20 || manifest.input.versions > 1<<20 {
		return empty, false
	}
	in := manifest.input
	counts, ok := providerHistoryCollectionMaximumCounts(in)
	if !ok || counts != manifest.counts || counts.groups == 0 || counts.groups > (1<<20)/3 {
		return empty, false
	}
	ceil := func(items uint64) uint64 {
		if items == 0 {
			return 0
		}
		return 1 + (items-1)/providerHistoryGroupMembers
	}
	var a providerHistoryReadArithmetic
	windows := a.sum(counts.ancestryWindows, counts.contentWindows)
	out := providerHistoryBoundaryPhaseCaps{
		opening: windows, closing: windows, content: counts.contentGroups,
		state: counts.contentWindows, grants: in.presentStops,
		ancestryItems: in.blocks - 1, historyItems: in.versions, groups: counts.groups,
	}
	if in.blocks > 1 {
		out.ancestry = a.sum(ceil(in.blocks-1), counts.ancestryWindows-1)
	}
	if in.effects > 0 {
		out.effects = min(in.effects, a.sum(ceil(in.effects), counts.contentGroups-1))
	}
	if in.versions > 0 {
		out.history = min(in.versions, a.sum(ceil(in.versions), counts.contentWindows-1))
	}
	total := a.sum(out.opening, out.closing, out.ancestry, out.content, out.effects, out.state, out.grants, out.history)
	if a.failed || total != out.groups {
		return empty, false
	}
	return out, true
}

type providerHistoryBoundaryNormalTier struct {
	count uint64
	work  candidateWorkVector
}

// Finite source-cost DATA. These11 marginal tiers describe a relaxation of
// original phase ordering; each coordinate may put its expensive phases first.
// It is an upper, not an actual trace or a source/owner/factory capability.
type providerHistoryBoundaryRoleModel struct {
	tiers  [11]providerHistoryBoundaryNormalTier
	groups uint64
	delta  candidateWorkVector
}

func providerHistoryCollectionBoundaryRoleModelUpper(manifest *providerHistoryCollectionManifest, methods providerMethodBounds) (providerHistoryBoundaryRoleModel, bool) {
	var empty providerHistoryBoundaryRoleModel
	caps, capsOK := providerHistoryCollectionBoundaryPhaseCapsBound(manifest)
	if !capsOK || !methods.fits(manifest.input.capture) {
		return empty, false
	}
	phase, phaseOK := providerHistoryBoundaryInterpretationUpper(methods, manifest.input.blocks)
	header, headerOK := providerHistoryBoundaryHeaderControlUpper(uint64(methods.header.body))
	history, historyOK := providerHistoryBoundaryABIControlUpper(uint64(methods.state.body), false, 262144, 262144)
	if !phaseOK || !headerOK || !historyOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryBoundaryRoleModel
	out.groups = caps.groups
	out.tiers[0] = providerHistoryBoundaryNormalTier{caps.opening, phase.opening.normal}
	out.tiers[1] = providerHistoryBoundaryNormalTier{caps.closing, phase.closing.normal}
	out.tiers[5] = providerHistoryBoundaryNormalTier{caps.state, phase.state.normal}
	out.tiers[6] = providerHistoryBoundaryNormalTier{caps.grants, phase.grants.normal}
	out.tiers[10].count = a.sum(caps.content, caps.effects) // assigned subgraph only
	ancestryItem := header.normal
	ancestryItem[10] = a.sum(ancestryItem[10], 133)
	assign := func(first int, items, groups uint64, item candidateWorkVector, overhead uint64) {
		full, rest := items/providerHistoryGroupMembers, items%providerHistoryGroupMembers
		partial := uint64(0)
		if rest != 0 {
			partial = 1
		}
		used := a.sum(full, partial)
		if a.failed || used > groups {
			a.failed = true
			return
		}
		out.tiers[first].count = full
		out.tiers[first+1].count = partial
		out.tiers[first+2].count = groups - used
		for coordinate := range item {
			out.tiers[first].work[coordinate] = a.product(providerHistoryGroupMembers, item[coordinate])
			out.tiers[first+1].work[coordinate] = a.product(rest, item[coordinate])
		}
		for index := first; index < first+3; index++ {
			out.tiers[index].work[10] = a.sum(out.tiers[index].work[10], overhead)
		}
	}
	assign(2, caps.ancestryItems, caps.ancestry, ancestryItem, a.sum(manifest.input.blocks, 1))
	assign(7, caps.historyItems, caps.history, history.normal, 0)
	// E belongs to a COMPLETE phase leaf. In particular both/all sibling
	// header calls after a bad header are included, not replaced by first-false
	// reasoning for an individual header. Current variable groups have <=32
	// members; the max-size leaf delta bounds every smaller group as well.
	for _, row := range [...]struct {
		count uint64
		pair  providerHistoryControlPair
	}{{caps.opening, phase.opening}, {caps.closing, phase.closing}, {caps.ancestry, phase.ancestry},
		{caps.state, phase.state}, {caps.grants, phase.grants}, {caps.history, phase.history}} {
		if row.count == 0 {
			continue
		}
		for coordinate, normal := range row.pair.normal {
			if normal > row.pair.envelope[coordinate] {
				return empty, false
			}
			out.delta[coordinate] = max(out.delta[coordinate], row.pair.envelope[coordinate]-normal)
		}
	}
	if a.failed || !out.valid() {
		return empty, false
	}
	return out, true
}

func (m providerHistoryBoundaryRoleModel) valid() bool {
	if m.groups == 0 || m.groups > (1<<20)/3 {
		return false
	}
	var a providerHistoryReadArithmetic
	var total uint64
	for _, tier := range m.tiers {
		total = a.sum(total, tier.count)
	}
	return !a.failed && total == m.groups
}

// U(k) takes the k largest marginal costs in each coordinate. The ancestry
// and history tiers are nonincreasing within their own item allocation, so
// sorting the fixed11 tiers conservatively solves the relaxed prefix budget.
// No loop depends on the number of groups or supplied original byte lengths.
// With weighted=true also return sum(U(j),j=0..k-1). An unused weighted result
// cannot make an otherwise finite normal-prefix calculation overflow.
func (m providerHistoryBoundaryRoleModel) rankedUpper(count uint64, weighted bool) (candidateWorkVector, candidateWorkVector, bool) {
	var empty candidateWorkVector
	if !m.valid() || count > m.groups {
		return empty, empty, false
	}
	var a providerHistoryReadArithmetic
	var normal, prefixes candidateWorkVector
	for coordinate := range normal {
		// The fixed eleven tiers use eleven bits; no selected index writes an array.
		var seen uint16
		left, rank := count, uint64(0)
		for step := 0; step < len(m.tiers) && left > 0; step++ {
			best := -1
			for index, tier := range m.tiers {
				if seen&(1<<index) == 0 && tier.count > 0 && (best < 0 || tier.work[coordinate] > m.tiers[best].work[coordinate]) {
					best = index
				}
			}
			if best < 0 || best >= len(m.tiers) {
				return empty, empty, false
			}
			seen |= 1 << best
			tier := m.tiers[best]
			take := min(left, tier.count)
			normal[coordinate] = a.sum(normal[coordinate], a.product(take, tier.work[coordinate]))
			if weighted {
				full := a.product(take, count-1-rank)
				triangle := a.product(take, take-1) / 2
				if a.failed || triangle > full {
					return empty, empty, false
				}
				prefixes[coordinate] = a.sum(prefixes[coordinate], a.product(full-triangle, tier.work[coordinate]))
			}
			left, rank = left-take, rank+take
		}
	}
	if a.failed {
		return empty, empty, false
	}
	return normal, prefixes, true
}

// These are disjoint assigned interpretation roles only. closure is the
// coordinatewise MAX of independently priced Cs/Cf; do not sum all three.
// I/P/M have no advanceBoundary invocation, not zero whole source cost.
type providerHistoryBoundaryRoleCost struct {
	initial, before, preparation, member            candidateWorkVector
	successClosure, failedClosure, closure, advance candidateWorkVector
}

func (m providerHistoryBoundaryRoleModel) roleUpper(prior uint64, advancing bool) (providerHistoryBoundaryRoleCost, bool) {
	var empty providerHistoryBoundaryRoleCost
	if !m.valid() || prior >= m.groups || advancing && prior+1 >= m.groups {
		return empty, false
	}
	past, _, pastOK := m.rankedUpper(prior, false)
	current, _, currentOK := m.rankedUpper(prior+1, false)
	if !pastOK || !currentOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	price := func(normal candidateWorkVector, copies uint64, present bool) candidateWorkVector {
		var work candidateWorkVector
		if present {
			for coordinate := range work {
				work[coordinate] = a.sum(a.product(copies, normal[coordinate]), m.delta[coordinate])
			}
		}
		return work
	}
	var out providerHistoryBoundaryRoleCost
	out.before = price(past, 9, prior > 0)
	out.successClosure = price(current, 3, true)
	out.failedClosure = price(past, 3, prior > 0)
	for coordinate := range out.closure {
		out.closure[coordinate] = max(out.successClosure[coordinate], out.failedClosure[coordinate])
	}
	if advancing {
		out.advance = price(current, 2, true)
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

// Sum each original T's complete assigned role allowance, without actual
// refunds:14*sum(U(j),j=0..G-1)+3*U(G)+(3G-2)*delta. The extra already-entered
// preparation is zero for THIS subgraph only. This is not the full future,
// all-eleven certificate, native/RAM estimate, caller or derivation factory.
func (m providerHistoryBoundaryRoleModel) futureUpper() (candidateWorkVector, bool) {
	var empty candidateWorkVector
	normal, prefixes, ok := m.rankedUpper(m.groups, true)
	if !ok {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	deltas := a.product(3, m.groups)
	if a.failed || deltas < 2 {
		return empty, false
	}
	deltas -= 2
	var work candidateWorkVector
	for coordinate := range work {
		work[coordinate] = a.sum(a.product(14, prefixes[coordinate]), a.product(3, normal[coordinate]), a.product(deltas, m.delta[coordinate]))
	}
	if a.failed {
		return empty, false
	}
	return work, true
}
