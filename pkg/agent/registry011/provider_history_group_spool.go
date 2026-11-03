package registry011

import "strconv"

// Original private member receipt, produced only by the serialized capture
// path after both independent retained-byte passes. It contains no response
// payload, open connection, operating authority or protected result.
type providerHistoryMemberOriginal struct {
	owner                   *providerHistoryMemberOriginal
	dispatch                *providerHistoryGroupDispatch
	attempt                 *providerHistoryDispatchAttempt
	response, tls, result   string
	responseBytes, tlsBytes uint64
}

func (*providerHistoryMemberOriginal) UnmarshalJSON([]byte) error { return errProviderIntegration }

// Closed namespace derived from original C, T stage and ordinal. Directory
// contents and a producer-supplied name never select an allowed spool file.
func (d *providerHistoryGroupDispatch) originalNameLocked(ordinal int, kind string) string {
	if d == nil || d.owner != d || d.group == nil || d.acknowledgment == nil || d.acknowledgment.claim == nil ||
		d.acknowledgment.claim.frame == nil || d.coordinator == nil || d.coordinator.ticket == nil ||
		ordinal < 0 || ordinal >= d.coordinator.ticket.table.count || !lowerHash(d.acknowledgment.claim.frame.after.commit) ||
		kind != "response" && kind != "tls" {
		return ""
	}
	return providerReplayJoin([]string{"history-original-", d.acknowledgment.claim.frame.after.commit[2:], "-",
		strconv.FormatUint(d.coordinator.ticket.table.stage, 10), "-", strconv.Itoa(ordinal), "-", kind}, d.meter)
}

func (d *providerHistoryGroupDispatch) originalPrefixShapeLocked() bool {
	return d != nil && (d.activeMemberShapeLocked() || d.completedMemberShapeLocked() || d.failedMemberShapeLocked())
}

func (d *providerHistoryGroupDispatch) memberCatalogLocked() bool {
	if d == nil || !d.originalPrefixShapeLocked() || !d.group.activeLocked(d.meter) {
		return false
	}
	o := d.group
	if o.phase == "writing-settle" || o.phase == "writing-failed-settle" {
		c := o.store.frontierFiles.historyPending
		return c != nil && c.dispatch == d && c.frame != nil && c.frame.phase == "settle" && o.exactAdmissionCatalogLocked(c, true)
	}
	names, ok := d.originalCatalogNamesLocked()
	return ok && o.exactHistoryCatalogLocked(names, "")
}

// Re-derive initial/T/C and the exact original prefix, without observing a
// directory or adopting cached allowed-name maps from a write claim.
func (d *providerHistoryGroupDispatch) originalCatalogNamesLocked() ([2]map[string]bool, bool) {
	if d == nil || !d.originalPrefixShapeLocked() || !d.group.activeLocked(d.meter) {
		return [2]map[string]bool{}, false
	}
	o := d.group
	names, ok := o.prefixHistoryCatalogLocked(o.stageBefore)
	if !ok || !o.extendHistoryCatalogLocked(names, o.admission.claim) || !o.extendHistoryCatalogLocked(names, o.last.claim) {
		return [2]map[string]bool{}, false
	}
	count := d.retained
	if d.attempt != nil {
		count++
	}
	for ordinal := range count {
		files := 2
		if ordinal == d.retained {
			files = d.attempt.files
		}
		for index := range files {
			kind := "response"
			if index == 1 {
				kind = "tls"
			}
			name := d.originalNameLocked(ordinal, kind)
			if name == "" || len(name) > 255 || names[1][name] || uint64(len(names[1])) >= o.parent.bounds.catalogEntries {
				return [2]map[string]bool{}, false
			}
			names[1][name] = true
		}
	}
	return names, true
}

// Worst-case completed pack BEFORE any member effect. TLS's closed JSON has
// less than2048 fixed bytes (all fixed field names, bounded hashes, numbers,
// endpoint and punctuation), at most70 bytes per66-character digest and at
// most4 punctuation bytes per selected chain. The descriptor has the existing
// 64KiB ceiling. Actual construction is separately observed and checked.
func (d *providerHistoryGroupDispatch) wholeResultPackFitsLocked() bool {
	if d == nil || d.group == nil || d.group.parent == nil || d.group.store == nil || d.coordinator == nil || d.coordinator.ticket == nil ||
		d.coordinator.ticket.table.count <= 0 || d.coordinator.ticket.table.count > providerHistoryGroupMembers ||
		d.order == nil || d.coordinator.ticket.table.count != d.order.count || !d.group.parent.tls.valid() {
		return false
	}
	b := d.group.parent.tls
	certificates, a := providerUpperProduct(uint64(b.chains), uint64(b.certificates))
	digests, c := providerUpperProduct(certificates, 70)
	chains, e := providerUpperProduct(uint64(b.chains), 4)
	tlsBytes, f := providerUpperSum(2048, digests, chains)
	maximum := min(d.group.store.config.bounds.maximumObjectBytes, uint64(providerHistoryOriginalPackBytes))
	total := uint64(len(providerHistoryOriginalPackMagic) + 4 + providerHistoryOriginalPackHeaderBytes)
	if !a || !c || !e || !f || tlsBytes > 64<<10 || total > maximum {
		return false
	}
	for i := range d.coordinator.ticket.table.count {
		m := d.coordinator.ticket.table.members[i]
		if !m.capture.valid() || len(m.request.wire) >= m.capture.WireBytes {
			return false
		}
		response := uint64(m.capture.WireBytes - len(m.request.wire))
		next, ok := providerUpperSum(total, response, tlsBytes)
		if !ok || next > maximum {
			return false
		}
		total = next
	}
	return true
}

func (d *providerHistoryGroupDispatch) rootsObservedLocked() string {
	if d == nil || d.group == nil || !d.group.activeLocked(d.meter) {
		return ""
	}
	return providerHistoryTLSRootsObserved(d.group.parent.policy.raw, d.meter)
}

// Parse bounded policy DATA with its original copy and codec observations.
// A returned digest is not a selected policy, verified TLS peer or capability;
// the original live owner and native TLS checks remain required by the caller.
func providerHistoryTLSRootsObserved(raw string, meter *providerWorkMeter) string {
	v, ok := providerReplayJSON(providerReplayBytes(raw, meter), meter).(map[string]any)
	if !ok {
		return ""
	}
	p, ok := v["provider"].(map[string]any)
	if !ok {
		return ""
	}
	roots := contentString(p["tls_roots_artifact"])
	if !lowerHash(roots) {
		return ""
	}
	return roots
}

func (a *providerHistoryDispatchAttempt) retainOriginalCandidateLocked(data *providerHistoryCaptureData) (*providerHistoryMemberOriginal, error) {
	if a == nil || a.owner != a || a.dispatch == nil || !a.used || a.connection == nil || a.original != nil || a.files != 0 ||
		data == nil || data.sample == nil || data.knownFailure || data.uncertain || !data.sample.Complete {
		return nil, errProviderIntegration
	}
	d, sample := a.dispatch, data.sample
	o, m := d.group, d.meter
	if o == nil || o.phase != "retaining-member" || !d.activeMemberShapeLocked() || !o.activeLocked(m) ||
		data.requestWritten != len(a.member.request.wire) || data.responseRead != len(sample.ResponseWire) ||
		len(sample.ResponseWire) == 0 || data.responseRead > a.member.capture.WireBytes-data.requestWritten ||
		!providerHistoryRequestBytesMatchObserved(sample, &a.member.request, m) {
		return nil, errProviderIntegration
	}
	roots := d.rootsObservedLocked()
	tlsRaw, err := newProviderHistoryTLSOriginalObserved(d.coordinator.ticket, d.coordinator.consumption, a.ordinal, sample, roots, a.tls, m)
	if err != nil || roots == "" {
		return nil, errProviderIntegration
	}
	response := providerReplayString(sample.ResponseWire, m)
	r := &providerHistoryMemberOriginal{dispatch: d, attempt: a, responseBytes: uint64(len(response)), tlsBytes: uint64(len(tlsRaw))}
	r.owner = r
	r.response, r.tls = providerReplayDigest("", sample.ResponseWire, m), providerReplayDigest("", providerReplayBytes(tlsRaw, m), m)
	if response == "" || r.response == "" || r.tls == "" {
		return nil, errProviderIntegration
	}
	raw, sumOK := providerUpperSum(r.responseBytes, r.tlsBytes)
	overhead, productOK := providerUpperProduct(o.store.config.bounds.fileOverheadBytes, 2)
	physical, physicalOK := providerUpperSum(raw, overhead)
	charge := providerHistoryStorageCharge{physical: physical, raw: raw, payload: r.responseBytes, metadata: r.tlsBytes, objects: 2, files: 2}
	retained, fits := o.retention.add(charge, o.parent.bounds.future)
	groupRetained, groupFits := o.retention.add(charge, o.coordinator.charged)
	storedRaw, rawOK := providerUpperSum(o.store.retainedRaw, raw)
	if !sumOK || !productOK || !physicalOK || !fits || !groupFits || retained != groupRetained || !retained.withinStage(o.stageRetentionStart, d.coordinator.ticket.upper) || !rawOK ||
		storedRaw > o.parent.held.retained || uint64(len(o.store.objects))+retained.objects > uint64(o.store.config.bounds.maximumObjects) {
		return nil, errProviderIntegration
	}
	capture := o.parent.recovery.prepareMemberCaptureLocked(a, r)
	if capture == nil {
		return nil, errProviderIntegration
	}
	// Consume the full physical suballocation before opening either file.
	// Any partial/uncertain write quarantines without refund or member ACK.
	a.original, o.retention = r, retained
	a.recoveryCapture, o.parent.recovery.memberCapture, o.parent.recovery.memberOriginal = capture, capture, r
	if o.writeHistoryFileLocked(o.store.evidence, d.originalNameLocked(a.ordinal, "response"), providerReplayBytes(response, m)) != nil {
		return nil, errProviderIntegration
	}
	a.files = 1
	if !d.memberCatalogLocked() || o.writeHistoryFileLocked(o.store.evidence, d.originalNameLocked(a.ordinal, "tls"), providerReplayBytes(tlsRaw, m)) != nil {
		return nil, errProviderIntegration
	}
	a.files, o.store.retainedRaw = 2, storedRaw
	if !d.memberCatalogLocked() || !o.identityLocked() {
		return nil, errProviderIntegration
	}
	left, err := d.readMemberPrefixCandidateLocked(roots)
	if err != nil || left == nil {
		return nil, errProviderIntegration
	}
	r.result = left.results[a.ordinal].digest
	right, err := d.readMemberPrefixCandidateLocked(roots)
	if err != nil || right == nil || left == right || left.pack == right.pack || r.result == "" ||
		!providerReplayTextEqual(left.pack.digest, right.pack.digest, m) ||
		!providerReplayTextEqual(r.result, right.results[a.ordinal].digest, m) || !d.memberCatalogLocked() || !o.identityLocked() {
		return nil, errProviderIntegration
	}
	return r, nil
}

type providerHistoryRetainedPrefix struct {
	pack    *providerHistoryOriginalPack
	results [providerHistoryGroupMembers]providerHistoryMemberResult
}

// Fresh read of EVERY selected member original, including the active member.
// Neither raw strings nor parsed results are recovered from a producer cache.
func (d *providerHistoryGroupDispatch) readMemberPrefixCandidateLocked(roots string) (*providerHistoryRetainedPrefix, error) {
	if d == nil || !d.activeMemberShapeLocked() || d.attempt == nil || d.attempt.files != 2 || d.attempt.original == nil ||
		!d.group.activeLocked(d.meter) || !d.memberCatalogLocked() {
		return nil, errProviderIntegration
	}
	return d.readOriginalPrefixCandidateLocked(d.retained+1, roots)
}

func (d *providerHistoryGroupDispatch) readOriginalPrefixCandidateLocked(count int, roots string) (*providerHistoryRetainedPrefix, error) {
	if d == nil || !d.originalPrefixShapeLocked() || !d.group.activeLocked(d.meter) || !lowerHash(roots) || count <= 0 || count > d.coordinator.ticket.table.count {
		return nil, errProviderIntegration
	}
	maximumPrefix := d.retained
	if d.attempt != nil && d.attempt.files == 2 && d.attempt.original != nil {
		maximumPrefix++
	}
	if count != maximumPrefix || !d.memberCatalogLocked() {
		return nil, errProviderIntegration
	}
	o, m := d.group, d.meter
	inputs := make([]providerHistoryOriginalInput, 0, 2*count)
	for ordinal := range count {
		var r *providerHistoryMemberOriginal
		if ordinal < d.retained {
			r = d.originals[ordinal]
		} else if d.attempt != nil {
			r = d.attempt.original
		}
		if r == nil || r.owner != r || r.dispatch != d || r.attempt == nil || r.attempt.ordinal != ordinal ||
			r.responseBytes == 0 || r.responseBytes > uint64(d.coordinator.ticket.table.members[ordinal].capture.WireBytes) || r.tlsBytes == 0 || r.tlsBytes > 64<<10 ||
			!lowerHash(r.response) || !lowerHash(r.tls) {
			return nil, errProviderIntegration
		}
		for _, input := range []struct {
			kind, digest string
			bytes        uint64
		}{{"response", r.response, r.responseBytes}, {"tls", r.tls, r.tlsBytes}} {
			raw, err := o.store.read(o.store.evidence, d.originalNameLocked(ordinal, input.kind), int64(input.bytes))
			if err != nil || uint64(len(raw)) != input.bytes || !providerReplayTextEqual(providerReplayDigest("", raw, m), input.digest, m) {
				return nil, errProviderIntegration
			}
			text := providerReplayString(raw, m)
			if text == "" {
				return nil, errProviderIntegration
			}
			inputs = append(inputs, providerHistoryOriginalInput{input.kind, text})
		}
	}
	maximum := min(o.store.config.bounds.maximumObjectBytes, uint64(providerHistoryOriginalPackBytes))
	pack, err := newProviderHistoryOriginalPackObserved(d.coordinator.ticket.table.stage, d.coordinator.ticket.table.digest, "results", 0, count, inputs, maximum, m)
	if err != nil {
		return nil, errProviderIntegration
	}
	checked, results, err := parseProviderHistoryResultPackObserved(providerReplayBytes(pack.raw, m), pack.digest, d.coordinator.ticket, d.coordinator.consumption, count, roots, o.parent.tls, maximum, m)
	if err != nil || checked == nil || checked == pack {
		return nil, errProviderIntegration
	}
	for ordinal := range d.retained {
		if !providerReplayTextEqual(results[ordinal].digest, d.originals[ordinal].result, m) {
			return nil, errProviderIntegration
		}
	}
	return &providerHistoryRetainedPrefix{checked, results}, nil
}
