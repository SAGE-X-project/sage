package registry011

// The original-file reader owns a new replay and new maps. It is one component
// of the private scan body, not a completed scan, retained receipt or authority.
// The surrounding scanner still owes bounded chunk reservations, both complete
// catalogs, deferred lineage closure and the retained command/receipt chain.
// Callers hold the original store mutex and its already claimed body meter.
type providerScanReplayReader struct {
	owner     *providerScanReplayReader
	control   *providerScanControl
	view      *providerCustodyStore
	replay    *providerFrontierReplay
	meter     *providerWorkMeter
	target    providerScanTarget
	selected  providerFrontierHighwater
	attempt   providerScanAttempt
	nextSetup uint64
	positions map[string]providerCustodyPosition
	failed    bool
}

func (*providerScanReplayReader) UnmarshalJSON([]byte) error { return errProviderIntegration }

func newProviderScanReplayReader(c *providerScanControl, meter *providerWorkMeter, rawTarget []byte) (*providerScanReplayReader, error) {
	r, err := newProviderFrontierScanReplay(c, meter)
	if err != nil {
		return nil, errProviderIntegration
	}
	s := c.store
	f := s.frontierFiles
	if f == nil || !f.valid() || !f.initialized || f.pending != nil || f.pin.ordinal < f.pin.sequence {
		return nil, errProviderIntegration
	}
	x, err := newProviderScanCodec(s.config, meter)
	if err != nil {
		return nil, errProviderIntegration
	}
	target, err := x.parseTarget(rawTarget)
	if err != nil || x.hash("sage-provider-scan-target-v1", rawTarget) != c.prepared.target || target.setup != f.pin.setup || target.custody != f.pin.commit || target.frontier != f.pin.frontier || f.pin.ordinal-f.pin.sequence > uint64(s.config.bounds.maximumSetupRecoveries) {
		return nil, errProviderIntegration
	}
	// These immutable handles/configuration belong to the same selected
	// candidate. No objects, bindings, operations, indexes or setup maps are
	// imported from its provisional coordinator state.
	v := &providerCustodyStore{custody: s.custody, evidence: s.evidence, config: s.config, clock: s.clock, tip: s.config.anchor, objects: make(map[string]providerIntegrationObjectRef), bindings: make(map[string]string), operations: make(map[string]bool), setupCommits: make(map[string]string)}
	v.owner = v
	v.activeMeter.Store(meter)
	i := &providerScanReplayReader{control: c, view: v, replay: r, meter: meter, target: target, selected: f.pin, attempt: *c.prepared, positions: make(map[string]providerCustodyPosition)}
	i.owner = i
	if !i.resident(0, 0, 4*s.config.bounds.maximumMetadataPerCommit+64<<10) {
		return nil, errProviderIntegration
	}
	a, e := v.read(v.custody, "ANCHOR", int64(v.config.bounds.maximumMetadataPerCommit))
	b, e2 := v.read(v.evidence, "ANCHOR", int64(v.config.bounds.maximumMetadataPerCommit))
	m := contentObject(providerReplayJSON(a, meter), "format", "namespace", "instance_key", "profile", "initial_objects_root")
	empty, e3 := newProviderFrontierInventoryObserved("", nil, v.config.bounds.maximumEntries, 64<<20, meter)
	if e != nil || e2 != nil || e3 != nil || m == nil || !providerReplayEqual(a, b, meter) || providerReplayDigest("sage-provider-custody-anchor-v1", a, meter) != v.config.anchor || m["format"] != "sage-provider-custody-anchor-v1" || m["namespace"] != v.config.identity.Namespace || m["instance_key"] != v.config.identity.Instance || m["profile"] != v.config.profile || m["initial_objects_root"] != empty.digest {
		return nil, errProviderIntegration
	}
	v.objectsRoot, v.retainedRaw = empty.digest, uint64(2*len(a))
	return i, nil
}

func (i *providerScanReplayReader) valid() bool {
	return i != nil && i.owner == i && !i.failed && i.control.valid() && i.control.prepared != nil && *i.control.prepared == i.attempt && i.control.store.frontierFiles != nil && i.control.store.frontierFiles.valid() && i.control.store.frontierFiles.pin == i.selected && i.meter != nil && i.meter.owner == i.meter && i.meter.scan == i.control && !i.meter.failed && !i.meter.closed && i.control.store.activeMeter.Load() == i.meter && i.view != nil && i.view.activeMeter.Load() == i.meter
}

// Bound the additional reader maps together with the independently rebuilt
// ledger/index state, both retained schema copies and parsing scratch BEFORE
// growth. These are profile uppers, never measured physical RAM qualification.
func (i *providerScanReplayReader) resident(objects, payload, scratch uint64) bool {
	v, b := i.view, i.view.config.bounds
	count, a := providerUpperSum(uint64(len(v.objects)), objects)
	retained, c := providerUpperSum(v.retainedRaw, payload)
	// Three worst-case global object maps coexist: references, wire bindings
	// and semantically required references. The incoming fresh-object slice
	// has its own entries; it is not covered by those three map counts.
	objectEntries, d := providerUpperProduct(count, 3)
	setups, setupOK := providerUpperSum(uint64(len(v.setups)), 1)
	setupEntries, setupMulOK := providerUpperProduct(setups, 2)
	positions, positionOK := providerUpperSum(uint64(len(i.positions)), 2)
	operations, operationOK := providerUpperSum(uint64(len(v.operations)), 1)
	catalog, catalogOK := providerUpperProduct(b.scan.maximumCatalogEntries, 2)
	entries, entriesOK := providerUpperSum(objectEntries, objects, setupEntries, positions, operations, catalog, 8)
	reader, e := providerUpperProduct(entries, b.scan.filenameEntryOverhead)
	reader, f := providerUpperProduct(reader, 2)
	// Include all finite catalog names reserved for the eventual full scan.
	names, g := providerUpperProduct(b.scan.maximumCatalogNameBytes, 2)
	reader, h := providerUpperSum(reader, names)
	if !a || !c || !d || !setupOK || !setupMulOK || !positionOK || !operationOK || !catalogOK || !entriesOK || !e || !f || !g || !h || count > uint64(b.maximumObjects) {
		return false
	}
	tickets, segments := uint64(0), uint64(0)
	if i.replay.body.ledger != nil {
		tickets, segments = uint64(len(i.replay.body.ledger.tickets)), uint64(len(i.replay.body.ledger.segments))
	}
	base, ok := b.continuation.residentUpper(tickets+1, v.sequence+1, segments+1, count, retained, scratch)
	total, sumOK := providerUpperSum(base, reader)
	return ok && sumOK && total <= b.scan.maximumResidentBytes
}

// Load only structural setup closure here. expected_tip remains an unresolved
// main-stream obligation; its presence or hash is never treated as proof that
// the expected main record exists. The typed boundary consumes it later.
func (i *providerScanReplayReader) setup() error {
	if !i.valid() || i.nextSetup > i.selected.ordinal-i.selected.sequence {
		return errProviderIntegration
	}
	i.failed = true
	v, meter, n := i.view, i.meter, i.nextSetup
	if !i.resident(0, 3*v.config.bounds.maximumMetadataPerCommit, 4*v.config.bounds.maximumMetadataPerCommit+64<<10) {
		return errProviderIntegration
	}
	claim, a := v.read(v.custody, providerCustodyName("setup-intent", n), int64(v.config.bounds.maximumMetadataPerCommit))
	local, b := v.read(v.evidence, providerCustodyName("setup-claim", n), int64(v.config.bounds.maximumMetadataPerCommit))
	commit, c := v.read(v.custody, providerCustodyName("setup-commit", n), int64(v.config.bounds.maximumMetadataPerCommit))
	m, ok := providerReplayJSON(claim, meter).(map[string]any)
	clock, clockOK := providerStoredClock(m["clock"])
	previous := v.config.anchor
	if n != 0 {
		previous = v.setups[len(v.setups)-1].commit
	}
	config := v.config
	config.attempt, config.expectedTip, config.expectedSetupTip = int(n), contentString(m["expected_tip"]), previous
	if a != nil || b != nil || c != nil || !ok || !clockOK || !lowerHash(config.expectedTip) || !candidateLedgerClockAdvance(i.control.prepared.clock, clock) || n == 0 && (clock != i.control.prepared.clock || config.expectedTip != config.anchor) || n != 0 && !candidateLedgerClockAdvance(v.setups[len(v.setups)-1].clock, clock) || !providerReplayEqual(claim, local, meter) || !providerReplayEqual(claim, providerCustodySetupDiagnosticObserved(config, clock, meter), meter) || !providerReplayEqual(commit, providerSetupCommitObserved(claim, meter), meter) {
		return errProviderIntegration
	}
	tip := providerReplayDigest("sage-provider-authority-setup-commit-v1", commit, meter)
	digest := providerReplayDigest("sage-provider-authority-setup-v1", claim, meter)
	if tip == "" || digest == "" || n == i.selected.ordinal-i.selected.sequence && tip != i.target.setup {
		return errProviderIntegration
	}
	if n == 0 {
		v.setupOrigin = ClockSample(clock)
		i.positions[v.config.anchor] = providerCustodyPosition{clock: clock}
	}
	v.setups = append(v.setups, providerSetupReference{tip, digest, config.expectedTip, int(n), clock})
	v.setupCommits[tip], v.setupTip = digest, tip
	v.retainedRaw += uint64(len(claim) + len(local) + len(commit))
	i.nextSetup++
	i.failed = false
	return nil
}

// Read every new original from the V2 inventory; no producer's object map or
// supplied raw object collection is accepted. Repeated transitive object reads
// during applyWithHighwater use the SAME observer as these initial reads.
func (i *providerScanReplayReader) main() error {
	if !i.valid() || i.nextSetup != i.selected.ordinal-i.selected.sequence+1 || i.view.sequence >= i.selected.sequence {
		return errProviderIntegration
	}
	i.failed = true
	v, meter := i.view, i.meter
	n, maximum := v.sequence+1, v.config.bounds.maximumMetadataPerCommit
	if !i.resident(0, 0, 5*maximum+64<<10) {
		return errProviderIntegration
	}
	intent, a := v.read(v.custody, providerCustodyName("intent", n), int64(maximum))
	commit, b := v.read(v.custody, providerCustodyName("commit", n), int64(maximum))
	pending, c := v.read(v.evidence, providerCustodyName("pending", n), int64(maximum))
	frontier, d := v.read(v.custody, providerCustodyName("frontier", n), int64(maximum))
	f, parseErr := parseProviderFrontierRecordObserved(frontier, v.config.identity, meter)
	if a != nil || b != nil || c != nil || d != nil || parseErr != nil || f.sequence != n || !providerReplayEqual(intent, pending, meter) || !providerReplayEqual(commit, providerFrontierCommitObserved(intent, meter), meter) {
		return errProviderIntegration
	}
	// Reserve the maximum incoming entries and their maps/slice BEFORE JSON
	// constructs any inventory entries. Raw object payload is not allocated
	// by this parse; its separately bounded exact upper is checked below.
	incoming := min(v.config.bounds.maximumEntries, v.config.bounds.maximumObjects-len(v.objects))
	if incoming <= 0 || !i.resident(uint64(incoming), 64<<10, 5*maximum+64<<10) {
		return errProviderIntegration
	}
	invRaw, err := v.read(v.evidence, "inventory-"+f.objects[2:], 64<<10)
	inventory, invErr := parseProviderFrontierInventoryObserved(invRaw, v.objectsRoot, incoming, v.config.bounds.maximumObjectBytes, min(v.config.bounds.maximumStoredBytes, 64<<20), meter)
	if err != nil || invErr != nil || inventory.digest != f.objects || !i.resident(uint64(len(inventory.entries)), inventory.payloadBytes, inventory.payloadBytes+5*maximum+64<<10) {
		return errProviderIntegration
	}
	fresh := make([]providerIntegrationObject, 0, len(inventory.entries))
	v.requiredReferences = make(map[string]bool)
	for _, ref := range inventory.entries {
		key, ok := providerReplayRefKey(ref, meter)
		if _, duplicate := v.objects[key]; !ok || duplicate {
			return errProviderIntegration
		}
		v.objects[key] = ref
		raw, err := v.object(ref.kind, ref.digest)
		// Inventory verification alone is not semantic use by typed replay.
		delete(v.requiredReferences, key)
		if err != nil {
			return errProviderIntegration
		}
		retained := providerReplayString(raw, meter)
		if retained == "" {
			return errProviderIntegration
		}
		fresh = append(fresh, providerIntegrationObject{kind: ref.kind, digest: ref.digest, raw: retained})
		if ref.kind == "wire-binding" {
			m := contentObject(providerReplayJSON(raw, meter), "format", "exchange_digest", "request_wire_digest", "response_wire_digest", "tls_digest")
			exchange := contentString(m["exchange_digest"])
			if m == nil || m["format"] != "sage-provider-wire-binding-v1" || !providerCandidateDigest(exchange) || !providerCandidateDigest(m["request_wire_digest"]) || !providerCandidateDigest(m["response_wire_digest"]) || !providerCandidateDigest(m["tls_digest"]) || v.bindings[exchange] != "" {
				return errProviderIntegration
			}
			v.bindings[exchange] = ref.digest
		}
	}
	setup := -1
	for ordinal, value := range v.setups {
		if value.commit == f.setup {
			setup = ordinal
		}
	}
	if setup < 0 {
		return errProviderIntegration
	}
	after, _, err := v.originalHighwaterObserved(n+uint64(setup), meter)
	cost, applyErr := i.replay.applyWithHighwater(v, inventory, fresh, intent, commit, frontier, after)
	if err != nil || applyErr != nil || !i.resident(0, cost.metadata+cost.payload, 5*maximum+64<<10) {
		return errProviderIntegration
	}
	m := providerFrontierIntentObjectObserved(intent, meter)
	operation := contentString(m["operation"])
	if m == nil || v.operations[operation] {
		return errProviderIntegration
	}
	prior := i.positions[v.tip]
	clock := i.replay.current.clock
	prior.next = &clock
	i.positions[v.tip] = prior
	v.sequence, v.tip, v.enrollment, v.objectsRoot = n, f.commit, f.enrollment, f.objects
	v.operations[operation] = true
	i.positions[v.tip] = providerCustodyPosition{sequence: n, clock: clock}
	v.retainedRaw += cost.metadata + cost.payload
	v.requiredReferences = nil
	i.failed = false
	return nil
}

// This checks only the deferred setup/main DATA relationship. A true result
// does not imply catalog EOF, a bounded complete scan, an S receipt or an ACK.
func (i *providerScanReplayReader) lineage() bool {
	if !i.valid() || i.nextSetup != i.selected.ordinal-i.selected.sequence+1 || i.view.sequence != i.selected.sequence || i.view.tip != i.target.custody || i.view.objectsRoot != i.target.objects || i.view.enrollment != i.target.enrollment {
		return false
	}
	var previous uint64
	for ordinal, setup := range i.view.setups {
		position, exists := i.positions[setup.expectedTip]
		if !exists || setup.ordinal != ordinal || position.sequence < previous || !candidateLedgerClockAdvance(position.clock, setup.clock) || position.next != nil && !candidateLedgerClockAdvance(setup.clock, *position.next) {
			return false
		}
		previous = position.sequence
	}
	f := i.replay.current
	if f.admission != i.target.admission || f.budget != i.target.budget || f.checkpoint != i.target.checkpoint {
		return false
	}
	// Consume any setup tail at the final main position too; a later setup
	// must not be silently skipped because no subsequent main record exists.
	i.failed = true
	v, meter := i.view, i.meter
	ordinal := max(0, i.replay.setupOrdinal)
	h, _, err := v.originalHighwaterObserved(v.sequence+uint64(ordinal), meter)
	if err != nil || h.commit != v.tip || h.setup != v.setups[ordinal].commit || h.sequence != v.sequence {
		return false
	}
	if v.sequence == 0 {
		if h.ordinal != 0 || h.previous != "" || h.frontier != "" {
			return false
		}
	} else {
		raw, err := f.bytesObserved(meter)
		if err != nil || h.frontier != providerReplayDigest("sage-provider-custody-frontier-v1", raw, meter) {
			return false
		}
	}
	for next := ordinal + 1; next < len(v.setups); next++ {
		after, _, err := v.originalHighwaterObserved(v.sequence+uint64(next), meter)
		claim, a := v.read(v.custody, providerCustodyName("setup-intent", uint64(next)), int64(v.config.bounds.maximumMetadataPerCommit))
		commit, b := v.read(v.custody, providerCustodyName("setup-commit", uint64(next)), int64(v.config.bounds.maximumMetadataPerCommit))
		if err != nil || a != nil || b != nil || !checkProviderHighwaterSetupObserved(h, after, v.config, claim, commit, meter) || after.setup != v.setups[next].commit {
			return false
		}
		h = after
	}
	head, err := h.bytesObserved(v.config, meter)
	current, readErr := v.read(v.custody, "HEAD", int64(v.config.bounds.maximumMetadataPerCommit))
	if err != nil || readErr != nil || h != i.selected || h.setup != i.target.setup || h.frontier != i.target.frontier || !providerReplayEqual(head, current, meter) {
		return false
	}
	i.failed = false
	return true
}
