package registry011

import "context"

const providerHistoryRecoveryInventoryDomain = "sage-provider-custody-objects-v2"

// Each pass starts from the sealed initial inventory, reads actual originals
// and owns fresh bounded reference state. It shares the one recovery observer
// and cumulative file/byte limits; starting pass two never renews that row.
// This byte/reference closure is not the complete initial semantic model.
type providerHistoryRecoveryInventoryWalk struct {
	owner       *providerHistoryRecoveryInventoryWalk
	recovery    *providerHistoryRecoveryOwner
	descriptor  *providerHistoryRecoveryInitialDescriptor
	pass        int
	next        string
	remaining   uint64
	current     *providerIntegrationInventory
	index       int
	seen        []string
	scanBinding *providerIntegrationObject
	frames      [4]*providerHistoryRecoveryInventoryFrame
	bytes       uint64
	complete    bool
}

type providerHistoryRecoveryInventoryRecord struct {
	sequence              uint64
	digest, previous, raw string
	objects               int
	bytes                 uint64
}

type providerHistoryRecoveryInventorySummary struct {
	root                        string
	inventories, objects, bytes uint64
}

func (*providerHistoryRecoveryInventoryWalk) UnmarshalJSON([]byte) error {
	return errProviderIntegration
}

// No map/model/cached inventory or caller digest can start a walk. A second
// pass requires the exact first pass to have closed its full reference chain.
func (r *providerHistoryRecoveryOwner) newInventoryWalkCandidate(ctx context.Context, pass int) (*providerHistoryRecoveryInventoryWalk, error) {
	if r == nil || r.store == nil {
		return nil, errProviderIntegration
	}
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if !r.activeLocked(r.meter) || pass < 0 || pass >= len(r.inventoryWalks) || r.inventoryWalks[pass] != nil || !r.initialDescriptorShapeLocked(r.group) {
		return nil, errProviderIntegration
	}
	if pass == 1 {
		first := r.inventoryWalks[0]
		if first == nil || !first.selectedLocked(r) || !first.complete || first.remaining != 0 || first.current != nil {
			return nil, errProviderIntegration
		}
	}
	accepted := false
	defer func() {
		if !accepted {
			r.stopLocked(true)
		}
	}()
	n := r.initialDescriptor
	maximum, ok := providerUpperProduct(n.main.sequence, uint64(r.config.bounds.maximumEntries))
	if !ok || n.objectCount == 0 || n.objectCount > maximum || !r.inputReadyLocked(ctx) || !r.meter.event(providerCopyEvent, n.objectCount) {
		return nil, errProviderIntegration
	}
	w := &providerHistoryRecoveryInventoryWalk{recovery: r, descriptor: n, pass: pass, next: n.objects, remaining: n.main.sequence, seen: make([]string, 0, int(n.objectCount))}
	w.owner = w
	if !r.inputReadyLocked(ctx) {
		return nil, errProviderIntegration
	}
	r.inventoryWalks[pass] = w
	accepted = true
	return w, nil
}

// Scalar original-pointer/finite-shape validation only. Actual key comparisons,
// decoding, hashing, copying and names use the explicit recovery observer.
func (w *providerHistoryRecoveryInventoryWalk) selectedLocked(r *providerHistoryRecoveryOwner) bool {
	return w != nil && w.owner == w && r != nil && w.recovery == r && w.pass >= 0 && w.pass < len(r.inventoryWalks) && r.inventoryWalks[w.pass] == w &&
		w.descriptor != nil && w.descriptor == r.initialDescriptor && r.initialDescriptorShapeLocked(r.group) &&
		w.remaining <= w.descriptor.main.sequence && w.index >= 0 && len(w.seen) <= int(w.descriptor.objectCount) && cap(w.seen) == int(w.descriptor.objectCount) &&
		w.bytes <= r.config.bounds.maximumStoredBytes && lowerHash(w.next) &&
		(w.current == nil && w.index == 0 || w.current != nil && w.remaining < uint64(len(w.frames)) && w.index < len(w.current.entries) && len(w.current.entries) <= r.config.bounds.maximumEntries)
}

func (w *providerHistoryRecoveryInventoryWalk) inputPlanLocked(r *providerHistoryRecoveryOwner, object bool) (*retentionDisk, uint64, bool) {
	if !w.selectedLocked(r) || w.complete {
		return nil, 0, false
	}
	if !object {
		return r.evidence, 64 << 10, w.current == nil && w.remaining > 0
	}
	if w.current == nil || w.index >= len(w.current.entries) {
		return nil, 0, false
	}
	ref := w.current.entries[w.index]
	_, known := providerIntegrationDomain(ref.kind)
	return r.evidence, ref.bytes, known && lowerHash(ref.digest) && ref.bytes > 0 && ref.bytes <= r.config.bounds.maximumObjectBytes
}

func (w *providerHistoryRecoveryInventoryWalk) inputNameLocked(r *providerHistoryRecoveryOwner, object bool) string {
	if !object {
		return providerReplayJoin([]string{"inventory-", w.next[2:]}, r.meter)
	}
	ref := w.current.entries[w.index]
	return providerReplayJoin([]string{"object-", ref.kind, "-", ref.digest[2:]}, r.meter)
}

func (w *providerHistoryRecoveryInventoryWalk) nextInventoryCandidate(ctx context.Context) (*providerHistoryRecoveryInventoryRecord, error) {
	if w == nil || w.recovery == nil || w.recovery.store == nil {
		return nil, errProviderIntegration
	}
	r := w.recovery
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if !r.activeLocked(r.meter) || !w.selectedLocked(r) || w.complete || w.current != nil || w.remaining == 0 {
		return nil, errProviderIntegration
	}
	accepted := false
	defer func() {
		if !accepted {
			r.stopLocked(true)
		}
	}()
	out, err := r.readSelectedInputLocked(ctx, providerHistoryRecoveryInputSelection{inventory: w})
	if err != nil || out == nil || out.state != providerHistoryRecoveryWholeInput || !providerReplayTextEqual(providerReplayDigest(providerHistoryRecoveryInventoryDomain, out.raw, r.meter), w.next, r.meter) {
		return nil, errProviderIntegration
	}
	// Only an anchored actual hash permits interpreting its predecessor. The
	// strict decoder and canonical byte comparison precede adopting any ref.
	fields := contentObject(providerReplayJSON(out.raw, r.meter), "format", "previous", "entries")
	previous := contentString(fields["previous"])
	if fields == nil || !lowerHash(previous) {
		return nil, errProviderIntegration
	}
	b := r.config.bounds
	inventory, err := parseProviderFrontierInventoryObserved(out.raw, previous, b.maximumEntries, b.maximumObjectBytes, min(b.maximumStoredBytes, 64<<20), r.meter)
	if err != nil || inventory == nil || !candidateCapacity(0, uint64(len(w.seen)), uint64(len(inventory.entries)), w.descriptor.objectCount) ||
		!candidateCapacity(0, w.bytes, inventory.payloadBytes, b.maximumStoredBytes) {
		return nil, errProviderIntegration
	}
	canonical := providerInventoryBytesObserved(providerHistoryRecoveryInventoryDomain, previous, inventory.entries, r.meter)
	if len(canonical) == 0 || !providerReplayEqual(canonical, out.raw, r.meter) || !r.meter.event(providerCopyEvent, uint64(len(inventory.entries))+1) || !r.inputReadyLocked(ctx) {
		return nil, errProviderIntegration
	}
	frame := &providerHistoryRecoveryInventoryFrame{walk: w, sequence: w.remaining, inventory: inventory, objects: make([]providerIntegrationObject, 0, len(inventory.entries))}
	frame.owner = frame
	record := &providerHistoryRecoveryInventoryRecord{sequence: w.remaining, digest: w.next, previous: previous, raw: inventory.raw, objects: len(inventory.entries), bytes: inventory.payloadBytes}
	w.frames[w.remaining-1] = frame
	w.current, w.index, w.remaining = inventory, 0, w.remaining-1
	accepted = true
	return record, nil
}

func (w *providerHistoryRecoveryInventoryWalk) nextObjectCandidate(ctx context.Context) (*providerIntegrationObject, error) {
	if w == nil || w.recovery == nil || w.recovery.store == nil {
		return nil, errProviderIntegration
	}
	r := w.recovery
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if !r.activeLocked(r.meter) || !w.selectedLocked(r) || w.complete || w.current == nil {
		return nil, errProviderIntegration
	}
	accepted := false
	defer func() {
		if !accepted {
			r.stopLocked(true)
		}
	}()
	if !r.inputReadyLocked(ctx) {
		return nil, errProviderIntegration
	}
	ref := w.current.entries[w.index]
	key, ok := providerReplayRefKey(ref, r.meter)
	if !ok || uint64(len(w.seen)) >= w.descriptor.objectCount || !r.inputReadyLocked(ctx) {
		return nil, errProviderIntegration
	}
	// Bounded linear comparisons avoid an unobserved hash-map lookup. Each
	// pass owns independent keys; no producer/cache map selects an object.
	for _, previous := range w.seen {
		if !r.meter.event(providerCompareEvent, uint64(len(previous)+len(key))) || previous == key {
			return nil, errProviderIntegration
		}
	}
	out, err := r.readSelectedInputLocked(ctx, providerHistoryRecoveryInputSelection{inventory: w, object: true})
	if err != nil || out == nil || out.state != providerHistoryRecoveryWholeInput || uint64(len(out.raw)) != ref.bytes {
		return nil, errProviderIntegration
	}
	object, err := newProviderIntegrationObjectObserved(ref.kind, out.raw, r.config.bounds.maximumObjectBytes, r.meter)
	if err != nil || !providerReplayTextEqual(object.digest, ref.digest, r.meter) || !r.meter.event(providerCopyEvent, 1) {
		return nil, errProviderIntegration
	}
	var binding *providerIntegrationObject
	if ref.kind == providerAdmissionBindingKind(r.config.bounds) {
		if w.scanBinding != nil || !r.meter.event(providerCopyEvent, 1) {
			return nil, errProviderIntegration
		}
		// A separate value from returned DATA; its immutable raw came from
		// this exact anchored actual object, never a producer binding cache.
		copy := object
		binding = &copy
	}
	if !r.inputReadyLocked(ctx) {
		return nil, errProviderIntegration
	}
	frame := w.frames[w.remaining]
	if frame == nil || frame.owner != frame || frame.walk != w || frame.inventory != w.current || frame.complete || len(frame.objects) != w.index || cap(frame.objects) != len(w.current.entries) || !r.meter.event(providerCopyEvent, 1) || !r.inputReadyLocked(ctx) {
		return nil, errProviderIntegration
	}
	if binding != nil {
		w.scanBinding = binding
	}
	w.seen = append(w.seen, key)
	frame.objects = append(frame.objects, object)
	w.bytes += ref.bytes
	w.index++
	if w.index == len(w.current.entries) {
		frame.complete = true
		w.next, w.current, w.index = w.current.previous, nil, 0
	}
	accepted = true
	return &object, nil
}

// Four actual nonempty inventory nodes must terminate at the independently
// encoded empty root. The bootstrap does not promise an empty-root file, so
// this final comparison deliberately does not invent another file read.
func (w *providerHistoryRecoveryInventoryWalk) finishCandidate(ctx context.Context) (*providerHistoryRecoveryInventorySummary, error) {
	if w == nil || w.recovery == nil || w.recovery.store == nil {
		return nil, errProviderIntegration
	}
	r := w.recovery
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if !r.activeLocked(r.meter) || !w.selectedLocked(r) || w.complete || w.current != nil || w.remaining != 0 {
		return nil, errProviderIntegration
	}
	accepted := false
	defer func() {
		if !accepted {
			r.stopLocked(true)
		}
	}()
	if !r.inputReadyLocked(ctx) {
		return nil, errProviderIntegration
	}
	empty, err := newProviderFrontierInventoryObserved("", nil, r.config.bounds.maximumEntries, min(r.config.bounds.maximumStoredBytes, 64<<20), r.meter)
	if err != nil || empty == nil || uint64(len(w.seen)) != w.descriptor.objectCount || !providerReplayTextEqual(w.next, empty.digest, r.meter) || !r.inputReadyLocked(ctx) {
		return nil, errProviderIntegration
	}
	w.complete = true
	accepted = true
	return &providerHistoryRecoveryInventorySummary{root: w.descriptor.objects, inventories: w.descriptor.main.sequence, objects: uint64(len(w.seen)), bytes: w.bytes}, nil
}
