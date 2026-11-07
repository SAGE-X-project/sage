package registry011

import "reflect"

// Successful decoded-command/claim work only. ABI framing/canonical calls,
// complete commitObserved, state/proofs and outer RecordChanged are separate.
// Numeric DATA grants no original input, profile, reservation or execution.
type providerHistoryCommandWork struct {
	tree, tuple, generation, consumer, claim candidateWorkVector
	linear, pool, actualWindow, headroom     candidateWorkVector
	window, total                            candidateWorkVector
	windowOpens, commandSlots, rawPool       uint64
}

// Fixed closed schema atoms, independent of window or original-role counts.
// These logical targets grant no source, observer or reservation authority.
func providerHistoryCommandShapeUpper() (providerHistoryCommandWork, bool) {
	var empty providerHistoryCommandWork
	var a providerHistoryReadArithmetic
	mapSize := uint64(reflect.TypeFor[map[string]any]().Size())
	materialMap := uint64(reflect.TypeFor[map[[32]byte]bool]().Size())
	text := uint64(reflect.TypeFor[string]().Size())
	value := uint64(reflect.TypeFor[any]().Size())
	field := uint64(reflect.TypeFor[reflect.StructField]().Size())
	reflected := uint64(reflect.TypeFor[reflect.Value]().Size())
	abiKey := uint64(reflect.TypeFor[replayABIKey]().Size())
	abiService := uint64(reflect.TypeFor[replayABIService]().Size())
	key := uint64(reflect.TypeFor[replayKey]().Size())
	service := uint64(reflect.TypeFor[replayService]().Size())
	var out providerHistoryCommandWork
	// Closed create graph dominates all eight command and claim graphs:
	// 2311 visited nodes INCLUDING keys;274 maps,1083 entries,2 arrays,
	// 144 array elements,1907 strings/94431 bytes and128 numbers of<=20.
	// The tuple's65-byte binary ceiling encodes to87 bytes, not86.
	out.tree[2] = 2560
	out.tree[10] = a.sum(a.product(274, mapSize), a.product(1083, text), a.product(1371, value), 247818)
	//128 seven-field key tuples and16 three-field services, including
	// original reflection fields/tags, string/byte clones and array stores.
	out.tuple[10] = a.sum(a.product(1888, reflected), a.product(944, field),
		a.product(384, abiKey), a.product(48, abiService), 162402)
	// Closed generated maps/arrays, scalar encoders, operation/caller checks
	// and DID construction. Generated command Marshal is in tree/pool below.
	out.generation[10] = a.sum(a.product(274, mapSize), a.product(1083, text),
		a.product(1371, value), 2001596)
	// Strict consumer schema/scalars, material uniqueness, services x keys,
	// ABI parameter arrays/conversion. Raw JSON and final Pack are pooled.
	// Controller/salt and operator alternatives are conservatively enveloped;
	// expiry20 is allowed even though a successful safe integer needs<=16.
	out.consumer[2] = 68455
	out.consumer[10] = a.sum(materialMap, a.product(10, value), a.product(256, key),
		a.product(256, abiKey), a.product(32, service), a.product(32, abiService), 72310923)
	// Claim preconditions and six-field object only. Its native tree, JCS,
	// domain buffer/copies and hash are in tree/pool, not another invocation.
	out.claim[10] = a.sum(mapSize, a.product(6, text), a.product(6, value), 353)
	for i := range out.linear {
		out.linear[i] = a.sum(a.product(3, out.tree[i]), out.tuple[i],
			out.generation[i], out.consumer[i], out.claim[i])
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

// One numeric window's existing successful actual and incoming parts.
// Whole-program role multiplicity is applied only by its separate wrapper.
func providerHistoryCommandWindowUpper(logs, evidence uint64) (providerHistoryCommandWork, bool) {
	var empty providerHistoryCommandWork
	if logs == 0 || logs > 65536 || evidence == 0 || evidence > 64<<20 {
		return empty, false
	}
	out, ok := providerHistoryCommandShapeUpper()
	if !ok {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	// For each successful original calldata D, generated graph nodes+bytes
	// <=D+512: each key fragment fits its ABI tuple/offset and each service
	// fragment its tuple/offset; the fixed remainder fits512 extra bytes.
	// Both command/claim JSON therefore have R<=6*D+3136 and R<=262144.
	// Unique accepted frames share the SAME original replayBudget, hence
	// sum(D)<=evidence and sum(R)<=6*evidence+3136*logs, not logs*max(R).
	out.rawPool = a.sum(a.product(6, evidence), a.product(3136, logs))
	out.pool[2] = a.product(2, out.rawPool)
	// Generated encode + consumer2codec/encode + claim encode give at most
	// 5*262144*sum(R). Raw copy/comparisons + claim buffer/hash add8*sum(R).
	// Consumer Pack/final original comparison add(148804+2)*sum(D).
	// Per event constants:15 plus3*18 adopted claim domain bytes.
	out.pool[10] = a.sum(a.product(1310728, out.rawPool), a.product(148806, evidence), a.product(69, logs))
	// Consumer encode preflight is<=6*262144+64. Reserve one unconsumed
	// headroom per original window IN ADDITION to its entire actual upper.
	// This funds every command preflight without reporting it as actual or
	// renewing/refunding a meter. Native CPU/allocations remain unqualified.
	out.headroom[10] = 2474102493185
	out.windowOpens = 1
	out.commandSlots = a.product(out.windowOpens, logs)
	for i := range out.window {
		out.actualWindow[i] = a.sum(a.product(logs, out.linear[i]), out.pool[i])
		out.window[i] = a.sum(out.actualWindow[i], out.headroom[i])
		out.total[i] = a.product(out.windowOpens, out.window[i])
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

// Keep the original successful whole-program values and validation unchanged.
func providerHistoryCommandUpper(l providerHistoryCollectionSemanticLimits, logs, evidence uint64) (providerHistoryCommandWork, bool) {
	var empty providerHistoryCommandWork
	calls, ok := providerHistoryCollectionSemanticCallsBound(l)
	if !ok {
		return empty, false
	}
	out, ok := providerHistoryCommandWindowUpper(logs, evidence)
	if !ok {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	out.windowOpens = calls.windowOpens
	out.commandSlots = a.product(out.windowOpens, logs)
	for c := range out.total {
		out.total[c] = a.product(out.windowOpens, out.window[c])
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

// One complete conservative refusing generator/consumer/claim prefix only.
// The closed producer graph bounds raw output BEFORE the consumer's later
// 262144-byte guard. It is not valid to apply that successful guard to the
// earlier generation encoder. No large graph is created by this scalar row.
// Source-route, fixed graph, first-abort and original incoming scope still
// require an independent proof before this numeric DATA can be composed.
func providerHistoryCommandRefusalUpper() (providerHistoryControlPair, bool) {
	var empty providerHistoryControlPair
	shape, ok := providerHistoryCommandShapeUpper()
	if !ok {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	const raw uint64 = 6*(2311+94431+2560) + 64 // nodes,strings AND Number bytes.
	const calldata uint64 = 148804
	var out providerHistoryControlPair
	out.normal = shape.linear
	out.normal[2] = a.sum(out.normal[2], a.product(2, raw))
	out.normal[10] = a.sum(out.normal[10], a.product(5, a.product(raw, raw)), a.product(8, raw), a.product(calldata, calldata), a.product(2, calldata), 69)
	out.envelope = out.normal
	const incoming uint64 = 6*262144 + 64
	out.envelope[10] = a.sum(out.normal[10], a.product(incoming, incoming), 1)
	if a.failed {
		return empty, false
	}
	return out, true
}
