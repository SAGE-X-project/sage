package registry011

import "context"

// One selected fresh scan chain per independent inventory pass. Only actual
// anchored inventory DATA supplies its binding; no failed producer receipt
// array, caller digest, native sample or revived scan/finishing role is used.
type providerHistoryRecoveryScanWalk struct {
	owner                  *providerHistoryRecoveryScanWalk
	recovery               *providerHistoryRecoveryOwner
	source                 *providerHistoryRecoveryInventoryWalk
	bindingOriginal        *providerIntegrationObject
	codec                  *providerScanCodec
	binding                providerScanAdmissionBinding
	upper                  candidateWorkVector
	pass                   int
	next, command          string
	remaining              uint64
	input                  int // chunk custody/evidence, then command custody/evidence
	cursor                 providerScanCursor
	wantObjects, wantBytes uint64
	wantWork               candidateWorkVector
	nextStart              CandidateClockDiagnostic
	result                 providerScanReceiptClosure
	controlBytes           uint64
	records                []providerHistoryRecoveryScanRecord
	have, complete         bool
}

// Retained diagnostics and originals only, not actual work or a reconstructed
// initial model. Future complete target replay must verify the state/index roots.
type providerHistoryRecoveryScanRecord struct {
	index, objects, bytes                uint64
	start, end                           providerScanCursor
	state, tickets, operations, segments string
	chunk, command                       string
}

func (*providerHistoryRecoveryScanWalk) UnmarshalJSON([]byte) error { return errProviderIntegration }

func (r *providerHistoryRecoveryOwner) newScanWalkCandidate(ctx context.Context, pass int) (*providerHistoryRecoveryScanWalk, error) {
	if r == nil || r.store == nil {
		return nil, errProviderIntegration
	}
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if !r.activeLocked(r.meter) || pass < 0 || pass >= len(r.scanWalks) || r.scanWalks[pass] != nil {
		return nil, errProviderIntegration
	}
	source := r.inventoryWalks[pass]
	if source == nil || !source.selectedLocked(r) || !source.complete || source.scanBinding == nil || source.scanBinding.kind != providerAdmissionBindingKind(r.config.bounds) ||
		pass == 1 && (r.scanWalks[0] == nil || !r.scanWalks[0].selectedLocked(r) || !r.scanWalks[0].complete) {
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
	x, err := newProviderScanCodec(r.config, r.meter)
	if err != nil {
		return nil, errProviderIntegration
	}
	binding, err := x.parseAdmissionBinding(providerReplayPhaseBytes(source.scanBinding.raw, r.meter))
	n := r.initialDescriptor
	if err != nil || binding.ordinal != n.scan.ordinal || binding.clock != n.attempt.clock || binding.deadline != n.attempt.deadline || binding.held != n.scan.held {
		return nil, errProviderIntegration
	}
	for _, pair := range [][2]string{{binding.e0, n.scanMain.commit}, {binding.frontier, n.scanMain.frontier}, {binding.setup, n.scanMain.setup},
		{binding.target, n.attempt.target}, {binding.lastChunk, n.lastChunk}, {binding.admission, n.admission}, {binding.attempt, n.scan.intent}} {
		if !providerReplayTextEqual(pair[0], pair[1], r.meter) {
			return nil, errProviderIntegration
		}
	}
	inspection, err := providerScanInspectionUpper(r.config)
	if err != nil {
		return nil, errProviderIntegration
	}
	if !r.meter.event(providerCopyEvent, n.records) {
		return nil, errProviderIntegration
	}
	w := &providerHistoryRecoveryScanWalk{recovery: r, source: source, bindingOriginal: source.scanBinding, codec: x, binding: binding, upper: inspection.work, pass: pass,
		next: n.lastChunk, remaining: n.records, cursor: providerScanCursor{"done", 0}, records: make([]providerHistoryRecoveryScanRecord, 0, int(n.records))}
	w.owner = w
	if !w.verifyControlsLocked(ctx) || !r.inputReadyLocked(ctx) {
		return nil, errProviderIntegration
	}
	r.scanWalks[pass] = w
	accepted = true
	return w, nil
}

func (w *providerHistoryRecoveryScanWalk) selectedLocked(r *providerHistoryRecoveryOwner) bool {
	return w != nil && w.owner == w && r != nil && w.recovery == r && w.pass >= 0 && w.pass < len(r.scanWalks) && r.scanWalks[w.pass] == w &&
		w.source != nil && w.source == r.inventoryWalks[w.pass] && w.source.selectedLocked(r) && w.source.complete && w.bindingOriginal != nil && w.source.scanBinding == w.bindingOriginal &&
		w.codec != nil && w.codec.config == r.config && w.codec.meter == r.meter && w.remaining <= w.source.descriptor.records && w.input >= 0 && w.input <= 4 &&
		(w.complete && w.remaining == 0 && w.input == 4 && w.next == "" || !w.complete && w.remaining > 0 && lowerHash(w.next))
}

// These wrappers select only existing closed initial control roles. Their
// expected bytes are independently reproduced by the explicit DATA codec.
func (w *providerHistoryRecoveryScanWalk) controlExactLocked(ctx context.Context, kind providerHistoryRecoveryFileKind, ordinal uint64, expected []byte) bool {
	r := w.recovery
	out, err := r.readSelectedInputLocked(ctx, providerHistoryRecoveryInputSelection{checkpoint: r.initial, kind: kind, index: int(ordinal)})
	if err != nil || out == nil || out.state != providerHistoryRecoveryWholeInput || !w.codec.equal(out.raw, expected) {
		return false
	}
	bytes, ok := providerUpperSum(w.controlBytes, uint64(len(out.raw)))
	if !ok || bytes > r.config.bounds.continuation.maximumMetadataBytes {
		return false
	}
	w.controlBytes = bytes
	return true
}

func (w *providerHistoryRecoveryScanWalk) verifyControlsLocked(ctx context.Context) bool {
	r, x, binding := w.recovery, w.codec, w.binding
	n := r.initialDescriptor
	h := providerScanHighwater{}
	raw, err := x.highwater(h)
	head, headErr := x.head(h)
	if err != nil || headErr != nil || !providerReplayTextEqual(x.hash(providerScanHeadDomain, head), r.config.expectedScanHead, r.meter) ||
		!w.controlExactLocked(ctx, providerHistoryRecoveryScanHighwaterCustody, 0, raw) || !w.controlExactLocked(ctx, providerHistoryRecoveryScanHighwaterEvidence, 0, raw) {
		return false
	}
	var last providerScanAttempt
	for ordinal := uint64(1); ordinal <= binding.ordinal; ordinal++ {
		out, readErr := r.readSelectedInputLocked(ctx, providerHistoryRecoveryInputSelection{checkpoint: r.initial, kind: providerHistoryRecoveryScanIntent, index: int(ordinal)})
		if readErr != nil || out == nil || out.state != providerHistoryRecoveryWholeInput {
			return false
		}
		a, parseErr := x.parseAttempt(out.raw)
		if parseErr != nil || a.ordinal != ordinal || a.purpose != "initial-inspection" || a.clock != binding.clock || a.deadline != binding.deadline ||
			!w.controlExactLocked(ctx, providerHistoryRecoveryScanClaim, ordinal, out.raw) {
			return false
		}
		var ok bool
		w.controlBytes, ok = providerUpperSum(w.controlBytes, uint64(len(out.raw)))
		if !ok || w.controlBytes > r.config.bounds.continuation.maximumMetadataBytes {
			return false
		}
		after, commit, successorErr := x.successor(h, a)
		next, nextErr := x.highwater(after)
		if successorErr != nil || nextErr != nil || !w.controlExactLocked(ctx, providerHistoryRecoveryScanCommit, ordinal, commit) ||
			!w.controlExactLocked(ctx, providerHistoryRecoveryScanHighwaterCustody, ordinal, next) || !w.controlExactLocked(ctx, providerHistoryRecoveryScanHighwaterEvidence, ordinal, next) {
			return false
		}
		h, last = after, a
	}
	raw, err = x.highwater(h)
	head, headErr = x.head(h)
	expected, expectedErr := x.attempt(n.attempt)
	actual, actualErr := x.attempt(last)
	if err != nil || headErr != nil || expectedErr != nil || actualErr != nil || !x.equal(expected, actual) || h != n.scan ||
		!providerReplayTextEqual(x.hash(providerScanHighwaterDomain, raw), binding.highwater, r.meter) || !providerReplayTextEqual(x.hash(providerScanHeadDomain, head), binding.head, r.meter) ||
		!w.controlExactLocked(ctx, providerHistoryRecoveryScanHeadCustody, 0, head) || !w.controlExactLocked(ctx, providerHistoryRecoveryScanHeadEvidence, 0, head) {
		return false
	}
	return true
}

func (w *providerHistoryRecoveryScanWalk) inputPlanLocked(r *providerHistoryRecoveryOwner) (*retentionDisk, uint64, bool) {
	if !w.selectedLocked(r) || w.complete || w.input >= 4 || w.input >= 2 && !lowerHash(w.command) {
		return nil, 0, false
	}
	disk := r.custody
	if w.input%2 == 1 {
		disk = r.evidence
	}
	return disk, r.config.bounds.maximumMetadataPerCommit, true
}

func (w *providerHistoryRecoveryScanWalk) inputNameLocked(r *providerHistoryRecoveryOwner) string {
	if w.input < 2 {
		return providerScanReceiptName("chunk", w.next, r.meter)
	}
	return providerScanReceiptName("command", w.command, r.meter)
}

// A current exact pair is selected only by internal stage, not caller flags.
// The anchored hash is checked before interpreting any predecessor/command.
func (w *providerHistoryRecoveryScanWalk) receiptPairLocked(ctx context.Context, command bool) ([]byte, map[string]any, error) {
	r, x := w.recovery, w.codec
	start, kind, domain, digest := 0, "chunk", providerScanChunkV2, w.next
	if command {
		start, kind, domain, digest = 2, "command", providerScanCommandV2, w.command
	}
	if w.input != start || !lowerHash(digest) {
		return nil, nil, errProviderIntegration
	}
	var original []byte
	var value map[string]any
	data := &providerScanReceipts{}
	for range 2 {
		out, err := r.readSelectedInputLocked(ctx, providerHistoryRecoveryInputSelection{scan: w})
		if err != nil || out == nil || out.state != providerHistoryRecoveryWholeInput || !providerReplayTextEqual(x.hash(domain, out.raw), digest, r.meter) {
			return nil, nil, errProviderIntegration
		}
		m := data.decode(x, kind, out.raw)
		canonical, encodeErr := x.encode(m)
		if m == nil || encodeErr != nil || !x.equal(out.raw, canonical) || original != nil && !x.equal(original, out.raw) {
			return nil, nil, errProviderIntegration
		}
		for _, pair := range [][2]string{{contentString(m["format"]), domain}, {contentString(m["target"]), w.binding.target},
			{contentString(m["scan_head"]), w.binding.head}, {contentString(m["scan_highwater"]), w.binding.highwater}, {contentString(m["scan_attempt"]), w.binding.attempt}} {
			if !providerReplayTextEqual(pair[0], pair[1], r.meter) {
				return nil, nil, errProviderIntegration
			}
		}
		original, value = out.raw, m
		w.input++
	}
	return original, value, nil
}

func (w *providerHistoryRecoveryScanWalk) nextRecordCandidate(ctx context.Context) (*providerHistoryRecoveryScanRecord, error) {
	if w == nil || w.recovery == nil || w.recovery.store == nil {
		return nil, errProviderIntegration
	}
	r := w.recovery
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if !r.activeLocked(r.meter) || !w.selectedLocked(r) || w.complete || w.input != 0 {
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
	x, b := w.codec, r.config.bounds.scan
	raw, chunk, err := w.receiptPairLocked(ctx, false)
	index, a := providerCandidateUint(chunk["index"], false)
	records, c := providerCandidateUint(chunk["records"], true)
	objects, d := providerCandidateUint(chunk["objects"], false)
	bytes, e := providerCandidateUint(chunk["bytes"], false)
	totalRecords, f := providerCandidateUint(chunk["cumulative_records"], true)
	totalObjects, g := providerCandidateUint(chunk["cumulative_objects"], false)
	totalBytes, h := providerCandidateUint(chunk["cumulative_bytes"], false)
	work, i := candidateAdmissionWork(chunk["work"])
	totalWork, j := candidateAdmissionWork(chunk["cumulative_work"])
	start, k := providerScanReceiptCursor(chunk["cursor_start"])
	end, l := providerScanReceiptCursor(chunk["cursor_end"])
	clockStart, n := providerStoredClock(chunk["clock_start"])
	clockEnd, p := providerStoredClock(chunk["clock_end"])
	previous, commandDigest := contentString(chunk["previous_chunk"]), contentString(chunk["command_digest"])
	if err != nil || !a || !c || !d || !e || !f || !g || !h || !i || !j || !k || !l || !n || !p || index != w.remaining-1 || index >= b.maximumChunks || index >= b.maximumRecords ||
		records != 1 || records > b.chunkRecords || totalRecords != index+1 || objects > b.chunkObjects || bytes > b.chunkRawBytes || totalObjects > b.maximumObjects || totalBytes > b.maximumRawBytes || totalObjects < objects || totalBytes < bytes ||
		end != w.cursor || !providerScanReceiptStep(start, end) || !providerIndexNullableDigest(chunk["previous_chunk"]) || (index == 0) != (previous == "") || index == 0 && start != (providerScanCursor{"setup", 0}) ||
		!lowerHash(commandDigest) || !providerScanReceiptOriginalTime(w.binding.clock, clockStart, clockEnd, w.binding.deadline, b.chunkDurationMS, b.totalDurationMS) ||
		w.have && (!candidateLedgerClockAdvance(clockEnd, w.nextStart) || totalObjects != w.wantObjects || totalBytes != w.wantBytes || totalWork != w.wantWork) {
		return nil, errProviderIntegration
	}
	var remainingWork candidateWorkVector
	for coordinate, value := range work {
		if value > w.upper[coordinate]/b.maximumChunks || totalWork[coordinate] > w.upper[coordinate] || totalWork[coordinate] < value {
			return nil, errProviderIntegration
		}
		remainingWork[coordinate] = totalWork[coordinate] - value
	}
	state, tickets := contentString(chunk["replay_state_digest"]), contentString(chunk["ticket_index_root"])
	operations, segments := contentString(chunk["operation_index_root"]), contentString(chunk["segment_totals_root"])
	// The original pre-I setup/enrollment prefix may have no replay root yet.
	// Preserve its existing nullable wire field; complete replay must later
	// compare that exact value with the independently reconstructed state.
	if !providerIndexNullableDigest(chunk["replay_state_digest"]) || !lowerHash(tickets) || !lowerHash(operations) || !lowerHash(segments) {
		return nil, errProviderIntegration
	}
	// The command filename is adopted only from both anchored, canonical,
	// bound chunk originals after their counter/cursor/time checks succeed.
	w.command = commandDigest
	command, _, commandErr := w.receiptPairLocked(ctx, true)
	data := &providerScanReceipts{head: w.binding.head, highwater: w.binding.highwater, attempt: w.binding.attempt}
	expected, encodeErr := data.command(x, w.binding.target, index, previous, start)
	metadata, metadataOK := providerUpperSum(w.result.metadata, uint64(2*len(raw)), uint64(2*len(command)))
	if commandErr != nil || encodeErr != nil || !x.equal(command, expected) || !metadataOK || metadata > r.config.bounds.continuation.maximumMetadataBytes ||
		index == 0 && (totalRecords-records != 0 || totalObjects-objects != 0 || totalBytes-bytes != 0 || remainingWork != (candidateWorkVector{})) {
		return nil, errProviderIntegration
	}
	chunkOriginal, commandOriginal := providerReplayString(raw, r.meter), providerReplayString(command, r.meter)
	if chunkOriginal == "" || commandOriginal == "" || uint64(len(w.records))+w.remaining != r.initialDescriptor.records || len(w.records) >= cap(w.records) ||
		!r.meter.event(providerCopyEvent, 1) || !r.inputReadyLocked(ctx) {
		return nil, errProviderIntegration
	}
	record := providerHistoryRecoveryScanRecord{index: index, objects: objects, bytes: bytes, start: start, end: end, state: state, tickets: tickets, operations: operations, segments: segments,
		chunk: chunkOriginal, command: commandOriginal}
	w.records = append(w.records, record)
	if !w.have {
		w.result.records, w.result.objects, w.result.bytes, w.result.work, w.result.end = totalRecords, totalObjects, totalBytes, totalWork, clockEnd
	}
	w.result.metadata, w.have = metadata, true
	w.wantObjects, w.wantBytes, w.wantWork, w.nextStart = totalObjects-objects, totalBytes-bytes, remainingWork, clockStart
	w.next, w.cursor, w.remaining = previous, start, w.remaining-1
	if index == 0 {
		w.complete = true
	} else {
		w.input, w.command = 0, ""
	}
	accepted = true
	return &record, nil
}
