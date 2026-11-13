package registry011

// Both original initial DATA models, excluding the separately composed index,
// request/exchange, first-finalized selection and file-input rows. The initial
// root boundary is later in group reconstruction; cursor initialization/root
// multiplicities already belong to reconstructionCursor and are NOT repeated.
// This prices decoded/logical work, not native resources or live authority.
type providerHistoryReconstructionInitialCost struct {
	profiles, policies, records, ledger candidateWorkVector
	materials, frontiers, seed          candidateWorkVector
	retention, control                  candidateWorkVector
	body, boundary                      candidateWorkVector
	profileCalls, policyDecodes         uint64
	semanticLookups, frameLookups       uint64
	frontierEncodes, inventoryPairs     uint64
	normal, incoming, envelope          candidateWorkVector
}

func providerHistoryReconstructionInitialUpper(l providerHistoryLegacyControlLimits, setupRecoveries, policyBytes uint64) (providerHistoryReconstructionInitialCost, bool) {
	var empty providerHistoryReconstructionInitialCost
	if setupRecoveries > 64 || policyBytes == 0 || policyBytes > 64<<10 {
		return empty, false
	}
	if _, ok := providerHistoryLegacyControlUpper(l); !ok {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	q := func(n uint64) uint64 { return a.sum(a.product(n, n), 1) }
	hash := func(n uint64, domain string) uint64 { return a.sum(n, uint64(len(domain)), 2) }
	const h, page, commit, anchor uint64 = 8192, 64 << 10, 1 << 20, 4096
	d, p, k := l.metadata, l.objectBytes, policyBytes
	f, record := min(d, page), min(p, page)
	o := min(l.firstObjects, a.product(4, l.entries))
	raw := min(l.firstRaw, a.product(o, p))
	scan := a.product(o, 2*26+2*66+2)
	out := providerHistoryReconstructionInitialCost{
		profileCalls: 14, policyDecodes: 26, semanticLookups: 130, frameLookups: 12, frontierEncodes: 30,
	}
	// Per pass: fresh frontier and scan-codec constructors, then one seed and
	// four retained highwaters. Each highwater independently rebuilds profile.
	// The SAME completed initializer already checked that profile bytes<=D.
	out.profiles[10] = a.product(out.profileCalls, a.sum(q(d), hash(d, "sage-provider-store-profile-v6")))
	// Thirteen independent policy copy/decodes per pass: two full parsers,
	// intent/T/C/S, seeded ledger, four deriveFrontier, seed deadline and
	// initialCapacity. Both full parsers retain TWO64KiB subtree encoders;
	// intent's separate limits encode/hash has its own64KiB preflight.
	policyParser := a.sum(a.product(2, q(page)), k, 1, hash(k, "sage-provider-policy-v1"),
		hash(page, "sage-registry-instance-v1"), hash(page, "sage-provider-origin-v1"))
	out.policies[2] = a.product(out.policyDecodes, k)
	out.policies[10] = a.product(2, a.sum(a.product(13, a.sum(q(k), k, 1)), a.product(2, policyParser), q(page), page, 1))
	// Four root parsers, intent/T/C/S and four fresh object constructors. The
	// latter decode/hash/string again. Six untyped peeks (budget, admission,
	// old C/S and body C/S) precede their typed guards and therefore retain P.
	// The setup-commit helper also decodes its selected claim before encoding.
	domains := uint64(5*len("sage-provider-enrollment-v2") + len("sage-provider-admission-intent-v1") +
		2*len("sage-provider-budget-ticket-v2") + 2*len("sage-provider-budget-consumption-v1") + 2*len("sage-provider-budget-settlement-v2"))
	out.records[2] = a.product(2, a.sum(a.product(12, record), a.product(7, p)))
	out.records[10] = a.product(2, a.sum(a.product(12, a.sum(q(record), a.product(2, record), 3)), domains,
		a.product(7, q(p)), q(h), hash(p, "sage-provider-authority-setup-v1"), p, h, 1,
		q(anchor), hash(anchor, "sage-provider-budget-segment-v1")))
	// Three empty nodes then ticket/C/S each create one node and one budget.
	// Continuing noteProviderRecord independently copies/hashes three nodes;
	// initial append's earlier note runs before continuation and has no event.
	out.ledger[10] = a.product(2, a.sum(a.product(9, q(h)), a.product(12, h+1),
		a.product(9, hash(h, "sage-provider-ledger-node-v1")), a.product(3, hash(h, "sage-provider-budget-root-v2"))))
	// 65 ordinary lookups/pass: oldMaterials30, roots4, index anchor/mutation
	// exact checks12 and body19 (including scan binding and six ledger nodes).
	// Each marks use with copy(1); six historical frame lookups do not. An
	// expected generated index/budget may reach8192 even when P is smaller.
	exact := max(p, h)
	nodeCheck := a.sum(h, 1, hash(h, "sage-provider-ledger-node-v1"), p, h, 1)
	out.materials[10] = a.product(2, a.sum(a.product(65, a.sum(scan, p, 3)),
		a.product(6, a.sum(scan, p, 1)), a.product(22, a.sum(p, a.product(2, exact), 2)),
		a.product(3, a.sum(p, 2*h, 2)), a.product(6, nodeCheck), record, 1,
		a.product(3, a.sum(p, 2*anchor, 2))))
	// Four apply intents plus two more inside each retention check. Fifteen
	// frontier encodes/pass: previous3, derived4, advance3, seed1, retention4.
	// Seven frontier hashes, eight commit hashes, and the four independently
	// reconstructed custody commits/intent canonicals preserve original guards.
	out.frontiers[2] = a.product(2, a.sum(a.product(12, d), a.product(4, f)))
	out.frontiers[10] = a.product(2, a.sum(a.product(12, q(d)), a.product(4, q(f)),
		a.product(15, q(page)), a.product(5, q(h)), a.product(4, q(commit)), a.product(4, q(d)),
		a.product(7, hash(page, "sage-provider-custody-frontier-v1")),
		a.product(8, hash(d, "sage-provider-custody-commit-v2")),
		a.product(4, hash(d, "sage-provider-custody-intent-v2")),
		a.product(8, a.sum(page, f, 1)), a.product(4, a.sum(commit, d, 1)), a.product(4, 2*d+1),
		a.product(4, a.sum(a.product(2, d), f, 3))))
	// The binding is decoded once and rebuilt before equality; its deadline
	// policy decode is in policies. Seed also builds target and compares two
	// original highwater/target byte values. Highwater/profile is above.
	binding := min(p, d, h)
	out.seed[2] = a.product(2, binding)
	out.seed[10] = a.product(2, a.sum(q(binding), a.product(2, q(d)), 2*h+1,
		hash(h, "sage-provider-scan-target-v1"), 6*h+4))
	// Each retained inventory rehashes every object and independently sorts
	// its own keys. R/F selects bounded insertion sort, not library sort.
	// Across4pages sum n(n-1)/2<=min(O(O-1)/2,O(E-1)/2). Adjacent duplicate
	// checks add at most O, each with two key copies and one comparison.
	out.inventoryPairs = a.product(2, min(a.product(o, o-1)/2, a.product(o, l.entries-1)/2))
	const key uint64 = 26 + 1 + 66
	pairs := a.sum(out.inventoryPairs, a.product(2, o))
	domain := uint64(len("sage-provider-guardian-admission-binding-v1"))
	pageWork := a.sum(q(page), hash(page, "sage-provider-custody-objects-v2"), 5*page, 4)
	out.retention[10] = a.sum(a.product(2, a.sum(a.product(2, raw), a.product(o, domain+3), a.product(4, pageWork))),
		a.product(pairs, 4*key+3))
	// Array copies/use checks, seven terminal digests, two scanState's four
	// root texts, four setup searches and independent historical read counts.
	// Read-count derivation's empty-lock I/O is ALREADY owned by files, not here.
	out.control[10] = a.product(2, a.sum(o, 4, a.product(2, o), 7*133, 8*133,
		a.product(a.product(4, a.sum(setupRecoveries, 1)), 133), a.product(29, o), 54))
	for coordinate := range out.body {
		out.body[coordinate] = a.sum(out.profiles[coordinate], out.policies[coordinate], out.records[coordinate], out.ledger[coordinate],
			out.materials[coordinate], out.frontiers[coordinate], out.seed[coordinate], out.retention[coordinate], out.control[coordinate])
	}
	// Later group construction separately verifies the last frontier and
	// constructs the common initial root. Cursor constructor/copies and ledger
	// root calls remain solely in their existing cursor/ledger rows.
	out.boundary[10] = a.product(2, a.sum(q(page), hash(page, "sage-provider-custody-frontier-v1"), 133,
		q(h), hash(h, "sage-provider-history-initial-model-v1"), 265, 133))
	out.incoming[2] = max(p, d, k, page)
	out.incoming[10] = max(q(out.incoming[2]), q(commit), a.sum(scan, p, 3), a.sum(p, 2*exact, 2))
	for coordinate := range out.normal {
		out.normal[coordinate] = a.sum(out.body[coordinate], out.boundary[coordinate])
		out.envelope[coordinate] = a.sum(out.normal[coordinate], out.incoming[coordinate])
	}
	if a.failed || !providerHistoryReadWorkDomain(out.envelope) {
		return empty, false
	}
	return out, true
}
