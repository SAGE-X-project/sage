package registry011

import "strconv"

// This record names already existing immutable objects. It is retained data,
// never an authenticated high-water, an ACK or a way to resume an old permit.
// The selected custodian must reconstruct and compare every field itself.
type providerFrontierRecord struct {
	identity                                     ProviderCandidateIdentity
	fence, sequence                              uint64
	previous, commit, setup, objects, enrollment string
	admission, budget, seal, replay, checkpoint  string
	indexes                                      [3]string
	initial, clock                               CandidateClockDiagnostic
	deadline                                     uint64
}

func (f providerFrontierRecord) valid() bool {
	if !lowerHash(f.identity.Namespace) || !lowerHash(f.identity.Instance) || f.fence == 0 || f.sequence == 0 || (f.sequence == 1) != (f.previous == "") {
		return false
	}
	for _, digest := range []string{f.commit, f.setup, f.objects, f.enrollment, f.indexes[0], f.indexes[1], f.indexes[2]} {
		if !lowerHash(digest) {
			return false
		}
	}
	if f.previous != "" && !lowerHash(f.previous) {
		return false
	}
	for _, digest := range []string{f.admission, f.budget, f.seal, f.replay, f.checkpoint} {
		if digest != "" && !lowerHash(digest) {
			return false
		}
	}
	if (f.admission == "") != (f.budget == "") || f.admission == "" && (f.seal != "" || f.replay != "" || f.checkpoint != "") || (f.replay == "") != (f.checkpoint == "") {
		return false
	}
	if f.sequence == 1 && f.admission != "" {
		return false
	}
	if !candidateLedgerClockAdvance(f.initial, f.clock) || f.initial.ClockID == "" || !lowerHash(f.initial.Epoch) {
		return false
	}
	end, ok := providerUpperSum(uint64(f.clock.WallMS), uint64(f.clock.UncertaintyMS))
	return ok && f.deadline != 0 && end <= f.deadline
}

func providerNullableDigest(digest string) any {
	if digest == "" {
		return nil
	}
	return digest
}

func providerDiagnosticClockObject(clock CandidateClockDiagnostic) map[string]string {
	// Serialization of retained diagnostic DATA needs no trusted sample or
	// owned-clock reconstruction, in either direction.
	return map[string]string{"id": clock.ClockID, "epoch": clock.Epoch, "wall_ms": strconv.FormatInt(clock.WallMS, 10), "monotonic_ms": strconv.FormatInt(clock.MonotonicMS, 10), "uncertainty_ms": strconv.FormatInt(clock.UncertaintyMS, 10)}
}

func (f providerFrontierRecord) bytes() ([]byte, error) {
	if !f.valid() {
		return nil, errProviderIntegration
	}
	raw := providerIntegrationJSON(map[string]any{
		"format": "sage-provider-custody-frontier-v1", "namespace": f.identity.Namespace, "instance_key": f.identity.Instance,
		"generation": strconv.FormatUint(f.identity.Generation, 10), "fence": strconv.FormatUint(f.fence, 10), "sequence": strconv.FormatUint(f.sequence, 10),
		"previous_frontier": providerNullableDigest(f.previous), "custody_commit": f.commit, "setup_head": f.setup, "objects_root": f.objects, "enrollment_root": f.enrollment,
		"admission_digest": providerNullableDigest(f.admission), "budget_root": providerNullableDigest(f.budget), "ledger_seal": providerNullableDigest(f.seal),
		"ticket_index_root": f.indexes[0], "operation_index_root": f.indexes[1], "segment_totals_root": f.indexes[2],
		"replay_state_digest": providerNullableDigest(f.replay), "checkpoint": providerNullableDigest(f.checkpoint),
		"initial_clock": providerDiagnosticClockObject(f.initial), "deadline_wall_ms": strconv.FormatUint(f.deadline, 10), "clock": providerDiagnosticClockObject(f.clock),
	})
	if len(raw) == 0 || len(raw) > 64<<10 {
		return nil, errProviderIntegration
	}
	return raw, nil
}

func parseProviderFrontierRecord(raw []byte, id ProviderCandidateIdentity) (providerFrontierRecord, error) {
	if len(raw) == 0 || len(raw) > 64<<10 {
		return providerFrontierRecord{}, errProviderIntegration
	}
	m := contentObject(contentJSON(raw), "format", "namespace", "instance_key", "generation", "fence", "sequence", "previous_frontier", "custody_commit", "setup_head", "objects_root", "enrollment_root", "admission_digest", "budget_root", "ledger_seal", "ticket_index_root", "operation_index_root", "segment_totals_root", "replay_state_digest", "checkpoint", "initial_clock", "deadline_wall_ms", "clock")
	fence, a := providerCandidateUint(m["fence"], true)
	sequence, b := providerCandidateUint(m["sequence"], true)
	deadline, c := providerCandidateUint(m["deadline_wall_ms"], true)
	initial, d := providerStoredClock(m["initial_clock"])
	clock, e := providerStoredClock(m["clock"])
	if m == nil || m["format"] != "sage-provider-custody-frontier-v1" || m["namespace"] != id.Namespace || m["instance_key"] != id.Instance || m["generation"] != strconv.FormatUint(id.Generation, 10) || !a || !b || !c || !d || !e {
		return providerFrontierRecord{}, errProviderIntegration
	}
	for _, name := range []string{"previous_frontier", "admission_digest", "budget_root", "ledger_seal", "replay_state_digest", "checkpoint"} {
		if !providerIndexNullableDigest(m[name]) {
			return providerFrontierRecord{}, errProviderIntegration
		}
	}
	f := providerFrontierRecord{identity: id, fence: fence, sequence: sequence, previous: contentString(m["previous_frontier"]), commit: contentString(m["custody_commit"]), setup: contentString(m["setup_head"]), objects: contentString(m["objects_root"]), enrollment: contentString(m["enrollment_root"]), admission: contentString(m["admission_digest"]), budget: contentString(m["budget_root"]), seal: contentString(m["ledger_seal"]), replay: contentString(m["replay_state_digest"]), checkpoint: contentString(m["checkpoint"]), indexes: [3]string{contentString(m["ticket_index_root"]), contentString(m["operation_index_root"]), contentString(m["segment_totals_root"])}, initial: initial, clock: clock, deadline: deadline}
	if !f.valid() {
		return providerFrontierRecord{}, errProviderIntegration
	}
	return f, nil
}

// This relation is checked in addition to typed payload/index replay. It fixes
// the original clock/deadline and forbids any installed nullable root clearing.
func checkProviderFrontierAdvance(before, after providerFrontierRecord) bool {
	if !before.valid() || !after.valid() || before.identity != after.identity || before.fence != after.fence || before.sequence == ^uint64(0) || after.sequence != before.sequence+1 || before.initial != after.initial || before.deadline != after.deadline || !candidateLedgerClockAdvance(before.clock, after.clock) {
		return false
	}
	raw, err := before.bytes()
	if err != nil || after.previous != providerCandidateHash("sage-provider-custody-frontier-v1", raw) {
		return false
	}
	for _, pair := range [][2]string{{before.admission, after.admission}, {before.budget, after.budget}, {before.seal, after.seal}, {before.replay, after.replay}, {before.checkpoint, after.checkpoint}} {
		if pair[0] != "" && pair[1] == "" {
			return false
		}
	}
	// Admission identity remains fixed, while budget/seal/checkpoint may advance
	// only through the separately validated typed delta.
	return before.admission == "" || before.admission == after.admission
}
