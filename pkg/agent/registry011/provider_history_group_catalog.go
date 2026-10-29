package registry011

// Derive the allowed first bridge catalog from its selected inventory hash,
// original legacy inventory chain, and the already replayed setup/main prefix.
// Directory enumeration is only a comparison against this independently
// derived set. An unknown observed name is never adopted as an allowed name.
func (o *providerHistoryGroupOwner) prepareAdmissionCatalogLocked(c *providerHistoryWriteClaim) bool {
	if c == nil || c.owner != c || c.group != o || c.frame == nil || c.inventory == nil || c.count <= 0 || c.count > len(c.fresh) || c.inventory.count != c.count ||
		o == nil || o.phase != "preparing-admit" && o.phase != "preparing-consume" || !o.activeLocked(o.meter) || o.start.main.sequence != 4 ||
		o.phase == "preparing-admit" && o.pin != o.start.main || o.phase == "preparing-consume" && !o.admissionShapeLocked() ||
		o.pin.ordinal > uint64(o.store.config.bounds.maximumCommits+o.store.config.bounds.maximumSetupRecoveries) ||
		len(o.store.objects) > o.store.config.bounds.maximumObjects || len(o.store.setups) == 0 || len(o.store.setups) > o.store.config.bounds.maximumSetupRecoveries+1 {
		return false
	}
	s, m := o.store, o.meter
	before, ok := o.initialHistoryCatalogLocked()
	if !ok {
		return false
	}
	if o.phase == "preparing-consume" && !o.extendHistoryCatalogLocked(before, o.admission.claim) {
		return false
	}
	var after [2]map[string]bool
	for i := range after {
		after[i] = make(map[string]bool, len(before[i])+9)
		for name := range before[i] {
			if !m.event(providerCopyEvent, uint64(len(name))) {
				return false
			}
			after[i][name] = true
		}
	}
	addAfter := func(root int, name string) bool {
		if name == "" || len(name) > 255 || after[root][name] || uint64(len(after[root])) >= o.parent.bounds.catalogEntries {
			return false
		}
		after[root][name] = true
		return true
	}
	for i := range c.count {
		if !addAfter(1, providerHistoryStoredName(c.inventory.entries[i], m)) {
			return false
		}
	}
	if !addAfter(1, providerHistoryInventoryName(c.inventory.digest, m)) || !addAfter(1, s.name("history-pending", c.frame.after.sequence)) ||
		!addAfter(0, s.name("intent", c.frame.after.sequence)) || !addAfter(0, s.name("commit", c.frame.after.sequence)) ||
		!addAfter(0, s.name("frontier", c.frame.after.sequence)) || !addAfter(0, s.name("highwater", c.frame.after.ordinal)) {
		return false
	}
	c.beforeCatalog, c.afterCatalog = before, after
	return true
}

func (o *providerHistoryGroupOwner) exactAdmissionCatalogLocked(c *providerHistoryWriteClaim, after bool) bool {
	if c == nil || c.owner != c || c.group != o || o == nil || !o.activeLocked(o.meter) {
		return false
	}
	want := c.beforeCatalog
	if after {
		want = c.afterCatalog
	}
	pending := ""
	if after && !c.renamed {
		pending = o.store.name("HEAD.pending", c.frame.after.ordinal)
	}
	return o.exactHistoryCatalogLocked(want, pending)
}

func (o *providerHistoryGroupOwner) exactHistoryCatalogLocked(want [2]map[string]bool, pendingName string) bool {
	if o == nil || !o.activeLocked(o.meter) {
		return false
	}
	var total, names uint64
	for root, disk := range []*retentionDisk{o.store.custody, o.store.evidence} {
		if want[root] == nil {
			return false
		}
		var identity candidateWorkVector
		identity[10] = 4
		if !o.meter.preflightWork(identity) {
			return false
		}
		for range 4 {
			if !o.meter.event(providerReadEvent, 0) {
				return false
			}
		}
		actual, err := providerDirectoryNamesObserved(disk, int(o.parent.bounds.catalogEntries), o.meter)
		pending := ""
		if root == 0 {
			pending = pendingName
		}
		count := len(want[root])
		if pending != "" {
			count++
		}
		if err != nil || len(actual) != count || pending != "" && !actual[pending] {
			return false
		}
		for name := range actual {
			if !want[root][name] && name != pending || !candidateCapacity(0, total, 1, o.parent.bounds.catalogEntries) ||
				!candidateCapacity(0, names, uint64(len(name)), o.parent.bounds.catalogNameBytes) {
				return false
			}
			total++
			names += uint64(len(name))
		}
	}
	return true
}

// Reconstruct the authenticated initial namespace. Enumerated names never
// select authority, and neither historical claim's cached catalog is reused.
func (o *providerHistoryGroupOwner) initialHistoryCatalogLocked() ([2]map[string]bool, bool) {
	if o == nil || o.start == nil || o.store == nil || !o.activeLocked(o.meter) || o.start.main.sequence != 4 ||
		o.start.main.ordinal > uint64(o.store.config.bounds.maximumCommits+o.store.config.bounds.maximumSetupRecoveries) ||
		len(o.store.objects) > o.store.config.bounds.maximumObjects || len(o.store.setups) == 0 || len(o.store.setups) > o.store.config.bounds.maximumSetupRecoveries+1 {
		return [2]map[string]bool{}, false
	}
	s, m, initial := o.store, o.meter, o.start.main
	var before [2]map[string]bool
	for i := range before {
		before[i] = make(map[string]bool)
	}
	add := func(root int, name string) bool {
		if name == "" || len(name) > 255 || before[root][name] || uint64(len(before[root])) >= o.parent.bounds.catalogEntries {
			return false
		}
		before[root][name] = true
		return true
	}
	for i := range before {
		if !add(i, "ANCHOR") || !add(i, "AUTHORITY.LOCK") {
			return [2]map[string]bool{}, false
		}
	}
	if !add(0, "HEAD") {
		return [2]map[string]bool{}, false
	}
	for i := uint64(0); i <= initial.ordinal; i++ {
		if !add(0, s.name("highwater", i)) {
			return [2]map[string]bool{}, false
		}
	}
	for i := uint64(1); i <= initial.sequence; i++ {
		for _, kind := range []string{"intent", "commit", "frontier"} {
			if !add(0, s.name(kind, i)) {
				return [2]map[string]bool{}, false
			}
		}
		if !add(1, s.name("pending", i)) {
			return [2]map[string]bool{}, false
		}
	}
	for i, setup := range s.setups {
		if setup.ordinal != i || !add(0, s.name("setup-intent", uint64(i))) || !add(0, s.name("setup-commit", uint64(i))) || !add(1, s.name("setup-claim", uint64(i))) {
			return [2]map[string]bool{}, false
		}
	}
	// The pre-I scan has its own already authenticated control prefix and
	// receipts. Its selected first-admission snapshot, not today's directory,
	// fixes the permitted names and the final original chunk commitment.
	first := o.parent.firstAdmission
	if first == nil || first.body != s.scanBody || first.control != s.scanControl || first.control == nil ||
		first.control.owner != first.control || first.control.pin != first.scanPin || first.control.prepared == nil || *first.control.prepared != first.attempt ||
		first.body == nil || first.records == 0 || first.records > s.config.bounds.scan.maximumChunks ||
		first.mainPin.ordinal != first.records-1 || !lowerHash(first.lastChunk) || !first.control.readPrefix() {
		return [2]map[string]bool{}, false
	}
	codec, err := newProviderScanCodec(s.config, m)
	if err != nil {
		return [2]map[string]bool{}, false
	}
	highwater, a := codec.highwater(first.scanPin)
	head, b := codec.head(first.scanPin)
	attempt, e := codec.attempt(first.attempt)
	// First admission deliberately discarded the closed inspection caches.
	// Reconstruct only receipt context DATA from the selected live snapshot;
	// this neither restores the old body nor replays its clock or authority.
	binding := providerScanAdmissionBinding{target: first.attempt.target,
		head: codec.hash(providerScanHeadDomain, head), highwater: codec.hash(providerScanHighwaterDomain, highwater),
		attempt: codec.hash(providerScanIntentDomain, attempt)}
	if a != nil || b != nil || e != nil || !lowerHash(binding.head) || !lowerHash(binding.highwater) || !lowerHash(binding.target) || binding.attempt != first.scanPin.intent {
		return [2]map[string]bool{}, false
	}
	for root := range before {
		if !add(root, "SCAN-HEAD") {
			return [2]map[string]bool{}, false
		}
		for i := uint64(0); i <= first.scanPin.ordinal; i++ {
			if !add(root, s.name("scan-highwater", i)) {
				return [2]map[string]bool{}, false
			}
		}
	}
	for i := uint64(1); i <= first.scanPin.ordinal; i++ {
		if !add(0, s.name("scan-intent", i)) || !add(0, s.name("scan-commit", i)) || !add(1, s.name("scan-claim", i)) {
			return [2]map[string]bool{}, false
		}
	}
	data := &providerScanReceipts{head: binding.head, highwater: binding.highwater, attempt: binding.attempt}
	digest, wantCursor := first.lastChunk, (providerScanCursor{"done", 0})
	for remaining := first.records; remaining > 0; remaining-- {
		_, chunk, readErr := readProviderAdmissionReceipt(s, codec, binding, "chunk", digest)
		index, indexOK := providerCandidateUint(chunk["index"], false)
		start, startOK := providerScanReceiptCursor(chunk["cursor_start"])
		end, endOK := providerScanReceiptCursor(chunk["cursor_end"])
		previous := contentString(chunk["previous_chunk"])
		if readErr != nil || !indexOK || index != remaining-1 || !startOK || !endOK || end != wantCursor || !providerScanReceiptStep(start, end) ||
			!providerIndexNullableDigest(chunk["previous_chunk"]) || (index == 0) != (previous == "") || index == 0 && start != (providerScanCursor{"setup", 0}) {
			return [2]map[string]bool{}, false
		}
		commandDigest := contentString(chunk["command_digest"])
		command, _, commandErr := readProviderAdmissionReceipt(s, codec, binding, "command", commandDigest)
		expected, encodeErr := data.command(codec, binding.target, index, previous, start)
		if commandErr != nil || encodeErr != nil || !codec.equal(command, expected) {
			return [2]map[string]bool{}, false
		}
		for _, pair := range []struct{ kind, digest string }{{"chunk", digest}, {"command", commandDigest}} {
			name := providerScanReceiptName(pair.kind, pair.digest, m)
			for root := range before {
				if !add(root, name) {
					return [2]map[string]bool{}, false
				}
			}
		}
		digest, wantCursor = previous, start
	}
	seen := make(map[string]bool)
	cursor := o.start.objects
	for range initial.sequence {
		if !lowerHash(cursor) || !add(1, s.inventoryName(cursor)) {
			return [2]map[string]bool{}, false
		}
		raw, err := s.read(s.evidence, s.inventoryName(cursor), 64<<10)
		if err != nil {
			return [2]map[string]bool{}, false
		}
		fields := contentObject(providerReplayJSON(raw, m), "format", "previous", "entries")
		previous := contentString(fields["previous"])
		inventory, err := parseProviderFrontierInventoryObserved(raw, previous, s.config.bounds.maximumEntries, s.config.bounds.maximumObjectBytes, 64<<20, m)
		if err != nil || !providerReplayTextEqual(inventory.digest, cursor, m) {
			return [2]map[string]bool{}, false
		}
		for _, ref := range inventory.entries {
			key, ok := providerReplayRefKey(ref, m)
			known, exists := s.objects[key]
			if !ok || !exists || known != ref || seen[key] || !add(1, s.objectName(ref.kind, ref.digest)) {
				return [2]map[string]bool{}, false
			}
			seen[key] = true
		}
		cursor = previous
	}
	empty, err := newProviderFrontierInventoryObserved("", nil, s.config.bounds.maximumEntries, 64<<20, m)
	if err != nil || !providerReplayTextEqual(cursor, empty.digest, m) || len(seen) != len(s.objects) {
		return [2]map[string]bool{}, false
	}
	return before, true
}

// The caller selects the exact original, acknowledged prefix; fresh directory
// entries and cached before/after maps are never promoted into this set.
func (o *providerHistoryGroupOwner) extendHistoryCatalogLocked(names [2]map[string]bool, c *providerHistoryWriteClaim) bool {
	if o == nil || !o.activeLocked(o.meter) || c == nil || c.owner != c || c.group != o || c.meter != o.meter ||
		!c.used || !c.renamed || c.frame == nil || c.inventory == nil || c.count <= 0 || c.count > len(c.fresh) || c.inventory.count != c.count {
		return false
	}
	add := func(root int, name string) bool {
		if names[root] == nil || name == "" || len(name) > 255 || names[root][name] || uint64(len(names[root])) >= o.parent.bounds.catalogEntries {
			return false
		}
		names[root][name] = true
		return true
	}
	for i := range c.count {
		if !add(1, providerHistoryStoredName(c.inventory.entries[i], o.meter)) {
			return false
		}
	}
	s := o.store
	return add(1, providerHistoryInventoryName(c.inventory.digest, o.meter)) && add(1, s.name("history-pending", c.frame.after.sequence)) &&
		add(0, s.name("intent", c.frame.after.sequence)) && add(0, s.name("commit", c.frame.after.sequence)) &&
		add(0, s.name("frontier", c.frame.after.sequence)) && add(0, s.name("highwater", c.frame.after.ordinal))
}
