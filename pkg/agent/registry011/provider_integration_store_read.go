package registry011

import (
	"bytes"
	"strconv"
)

func (s *providerCustodyStore) loadPrefix(now ClockSample) error {
	b := s.config.bounds
	custodyNames, err := providerDirectoryNames(s.custody, 2+2*b.maximumCommits+2*(b.maximumSetupRecoveries+1))
	if err != nil {
		return errProviderIntegration
	}
	localNames, err := providerDirectoryNames(s.evidence, 2+2*b.maximumCommits+b.maximumObjects+b.maximumSetupRecoveries+1)
	if err != nil {
		return errProviderIntegration
	}
	for _, names := range []map[string]bool{custodyNames, localNames} {
		if !names["ANCHOR"] || !names["AUTHORITY.LOCK"] {
			return errProviderIntegration
		}
		delete(names, "ANCHOR")
		delete(names, "AUTHORITY.LOCK")
	}
	// Full setup charges are retained independently of later ticket sums. Their
	// envelope covers opening, complete-prefix validation and bounded failures.
	setupCharge, ok := providerUpperProduct(uint64(s.config.attempt+1), b.setupRetained)
	if !ok || setupCharge > b.maximumStoredBytes {
		return errProviderIntegration
	}
	s.stored = setupCharge
	anchorBytes, err := s.read(s.evidence, "ANCHOR", int64(b.maximumMetadataPerCommit))
	if err != nil {
		return errProviderIntegration
	}
	s.retainedRaw = uint64(2 * len(anchorBytes))
	s.replay = &providerCustodyReplay{context: &ProviderCandidateContext{s.config.identity, true}, policy: s.config.policy}
	s.bindings = make(map[string]string)
	if s.loadSetupPrefix(custodyNames, localNames, now) != nil {
		return errProviderIntegration
	}
	positions := map[string]providerCustodyPosition{s.config.anchor: {clock: providerClockDiagnostic(s.setupOrigin)}}
	for n := uint64(1); n <= uint64(b.maximumCommits); n++ {
		intentName, commitName := providerCustodyName("intent", n), providerCustodyName("commit", n)
		pendingName := providerCustodyName("pending", n)
		if !custodyNames[intentName] && !custodyNames[commitName] && !localNames[pendingName] {
			break
		}
		if !custodyNames[intentName] || !custodyNames[commitName] || !localNames[pendingName] {
			return errProviderIntegration // Incomplete/unknown tails never prove absence.
		}
		intent, e := s.read(s.custody, intentName, int64(b.maximumMetadataPerCommit))
		commit, f := s.read(s.custody, commitName, int64(b.maximumMetadataPerCommit))
		pending, g := s.read(s.evidence, pendingName, int64(b.maximumMetadataPerCommit))
		m := s.checkIntent(intent, n)
		if e != nil || f != nil || g != nil || m == nil || !bytes.Equal(pending, intent) || !bytes.Equal(commit, providerCustodyCommitBytes(m, intent)) || s.loadInventory(contentString(m["objects_root"]), localNames) != nil {
			return errProviderIntegration
		}
		clock, valid := providerStoredClock(m["clock"])
		before := positions[s.tip]
		if !valid || !candidateLedgerClockAdvance(before.clock, clock) {
			return errProviderIntegration
		}
		before.next = &clock
		positions[s.tip] = before
		rootRef, exists := s.objects["enrollment:"+contentString(m["new_enrollment_root"])]
		s.requiredReferences = make(map[string]bool)
		if !exists || rootRef.kind != "enrollment" || s.checkEnrollment(m) != nil || s.replay.apply(s, m) != nil || !s.addStored(uint64(len(intent)+len(commit)+len(pending)), 3) {
			return errProviderIntegration
		}
		for _, ref := range s.newReferences {
			if !s.requiredReferences[providerIntegrationRefKey(ref)] {
				return errProviderIntegration
			}
		}
		s.requiredReferences = nil
		delete(custodyNames, intentName)
		delete(custodyNames, commitName)
		delete(localNames, pendingName)
		previous := s.tip
		s.tip = providerCandidateHash("sage-provider-custody-commit-v1", commit)
		s.enrollment = contentString(m["new_enrollment_root"])
		s.objectsRoot = contentString(m["objects_root"])
		s.sequence = n
		positions[s.tip] = providerCustodyPosition{sequence: n, clock: clock}
		fence, _ := providerCandidateUint(m["fence"], true)
		operation := contentString(m["operation"])
		s.operations[operation] = true
		s.last = &providerCustodyAcknowledgment{s, previous, s.tip, s.enrollment, operation, s.objectsRoot, fence, n, true}
	}
	if len(custodyNames) != 0 || len(localNames) != 0 || !s.checkSetupLineage(positions) || s.custody.checkIdentity() != nil || s.evidence.checkIdentity() != nil {
		return errProviderIntegration
	}
	return nil
}

func (s *providerCustodyStore) checkSetup(raw []byte, attempt int, now ClockSample) bool {
	m := contentObject(contentJSON(raw), "format", "namespace", "instance_key", "policy", "source_id", "generation", "anchor", "expected_tip", "previous_setup_commit", "operation", "profile", "attempt", "clock", "work", "live_upper", "retained_upper", "maximum_clock_reads", "maximum_duration_ms")
	if m == nil || !providerCandidateDigest(m["expected_tip"]) || !providerCandidateDigest(m["previous_setup_commit"]) {
		return false
	}
	c := contentObject(m["clock"], "id", "epoch", "wall_ms", "monotonic_ms", "uncertainty_ms")
	wall, a := providerCandidateUint(c["wall_ms"], false)
	mono, b := providerCandidateUint(c["monotonic_ms"], false)
	unc, d := providerCandidateUint(c["uncertainty_ms"], false)
	if c == nil || !a || !b || !d || wall > uint64(now.WallMS) || mono > uint64(now.MonotonicMS) || unc != uint64(now.UncertaintyMS) || c["id"] != now.ClockID || c["epoch"] != now.Epoch {
		return false
	}
	copy := s.config
	copy.attempt, copy.expectedTip = attempt, contentString(m["expected_tip"])
	copy.expectedSetupTip = contentString(m["previous_setup_commit"])
	clock := ClockSample{ClockID: now.ClockID, Epoch: now.Epoch, WallMS: int64(wall), MonotonicMS: int64(mono), UncertaintyMS: int64(unc)}
	if !bytes.Equal(raw, providerCustodySetup(copy, clock)) {
		return false
	}
	if attempt == 0 {
		s.setupOrigin = clock
	}
	return candidateLedgerClockAdvance(providerClockDiagnostic(s.setupOrigin), providerClockDiagnostic(clock))
}

func (s *providerCustodyStore) checkIntent(raw []byte, sequence uint64) map[string]any {
	m := contentObject(contentJSON(raw), "format", "namespace", "instance_key", "generation", "fence", "sequence", "previous_commit", "expected_enrollment_root", "new_enrollment_root", "objects_root", "operation", "kind", "policy", "clock", "metadata_reservation")
	id := s.config.identity
	n, a := providerCandidateUint(m["sequence"], true)
	fence, b := providerCandidateUint(m["fence"], true)
	if m == nil || m["format"] != "sage-provider-custody-intent-v1" || m["namespace"] != id.Namespace || m["instance_key"] != id.Instance || m["generation"] != strconv.FormatUint(id.Generation, 10) || m["policy"] != id.Policy || !a || !b || n != sequence || m["previous_commit"] != s.tip || !providerCandidateDigest(m["new_enrollment_root"]) || !providerCandidateDigest(m["objects_root"]) || !providerCandidateDigest(m["operation"]) || s.operations[contentString(m["operation"])] || !providerCustodyKind(contentString(m["kind"])) {
		return nil
	}
	if s.enrollment == "" && (m["expected_enrollment_root"] != nil || m["kind"] != "enroll" || fence != 1) || s.enrollment != "" && m["expected_enrollment_root"] != s.enrollment {
		return nil
	}
	// Every transaction carries the same selected conservative metadata envelope.
	metadata := providerIntegrationJSON(m["metadata_reservation"])
	want := providerIntegrationJSON(providerCustodyMetadata(s.config.bounds))
	clock := contentObject(m["clock"], "id", "epoch", "wall_ms", "monotonic_ms", "uncertainty_ms")
	if !bytes.Equal(metadata, want) || clock == nil || clock["id"] != s.clock.id || clock["epoch"] != s.clock.epoch {
		return nil
	}
	for _, field := range []string{"wall_ms", "monotonic_ms", "uncertainty_ms"} {
		if _, ok := providerCandidateUint(clock[field], false); !ok {
			return nil
		}
	}
	return m
}

func providerCustodyKind(kind string) bool {
	switch kind {
	case "enroll", "admit", "consume", "settle", "segment", "activate", "fence", "quarantine", "retire":
		return true
	}
	return false
}

func providerCustodyMetadata(b providerIntegrationBounds) map[string]string {
	return map[string]string{"bytes_upper": strconv.FormatUint(b.maximumMetadataPerCommit, 10), "file_overhead_upper": strconv.FormatUint(b.fileOverheadBytes, 10)}
}

func providerCustodyCommitBytes(intent map[string]any, raw []byte) []byte {
	commit := make(map[string]any)
	for _, name := range []string{"namespace", "instance_key", "generation", "fence", "sequence", "previous_commit", "new_enrollment_root", "objects_root", "operation"} {
		commit[name] = intent[name]
	}
	commit["format"], commit["intent_digest"] = "sage-provider-custody-commit-v1", providerCandidateHash("sage-provider-custody-intent-v1", raw)
	encoded := providerIntegrationJSON(commit)
	return encoded
}

func (s *providerCustodyStore) loadInventory(digest string, names map[string]bool) error {
	name := "inventory-" + digest[2:]
	b := s.config.bounds
	raw, err := s.read(s.evidence, name, 64<<10)
	if err != nil || !names[name] {
		return errProviderIntegration
	}
	inventory, err := parseProviderIntegrationInventory(raw, s.objectsRoot, b.maximumEntries, b.maximumObjectBytes, min(b.maximumStoredBytes, 64<<20))
	if err != nil || inventory.digest != digest || len(s.objects)+len(inventory.entries) > b.maximumObjects {
		return errProviderIntegration
	}
	delete(names, name)
	for _, ref := range inventory.entries {
		key, filename := providerIntegrationRefKey(ref), providerObjectName(ref.kind, ref.digest)
		if _, duplicate := s.objects[key]; duplicate || !names[filename] {
			return errProviderIntegration
		}
		object, err := s.read(s.evidence, filename, int64(ref.bytes))
		domain, _ := providerIntegrationDomain(ref.kind)
		if !s.event(providerHashEvent, uint64(len(object)+len(domain))) {
			return errProviderIntegration
		}
		digest, valid := providerIntegrationObjectDigest(ref.kind, object)
		if err != nil || !valid || digest != ref.digest || uint64(len(object)) != ref.bytes || !s.addStored(ref.bytes, 1) {
			return errProviderIntegration
		}
		s.objects[key] = ref
		if ref.kind == "wire-binding" {
			binding := contentObject(contentJSON(object), "format", "exchange_digest", "request_wire_digest", "response_wire_digest", "tls_digest")
			if binding == nil || binding["format"] != "sage-provider-wire-binding-v1" || !providerCandidateDigest(binding["exchange_digest"]) || !providerCandidateDigest(binding["request_wire_digest"]) || !providerCandidateDigest(binding["response_wire_digest"]) || !providerCandidateDigest(binding["tls_digest"]) || s.bindings[contentString(binding["exchange_digest"])] != "" {
				return errProviderIntegration
			}
			s.bindings[contentString(binding["exchange_digest"])] = ref.digest
		}
		delete(names, filename)
	}
	if !s.addStored(uint64(len(raw)), 1) {
		return errProviderIntegration
	}
	s.newReferences = inventory.entries
	return nil
}

func (s *providerCustodyStore) addStored(bytes, files uint64) bool {
	overhead, ok := providerUpperProduct(files, s.config.bounds.fileOverheadBytes)
	n, sumOK := providerUpperSum(bytes, overhead)
	if !ok || !sumOK || !candidateCapacity(0, s.stored, n, s.config.bounds.maximumStoredBytes) {
		return false
	}
	s.stored += n
	s.retainedRaw += bytes
	return true
}

func (s *providerCustodyStore) checkEnrollment(intent map[string]any) error {
	context := &ProviderCandidateContext{identity: s.config.identity, bound: true}
	raw, err := s.read(s.evidence, providerObjectName("enrollment", contentString(intent["new_enrollment_root"])), int64(s.config.bounds.maximumObjectBytes))
	after, parseErr := ParseCandidateEnrollmentRoot(context, raw, 64<<10)
	fence, _ := providerCandidateUint(intent["fence"], true)
	if err != nil || parseErr != nil || after.fence != fence {
		return errProviderIntegration
	}
	if s.enrollment == "" {
		if after.sequence != 0 || intent["kind"] != "enroll" {
			return errProviderIntegration
		}
		return nil
	}
	beforeBytes, err := s.read(s.evidence, providerObjectName("enrollment", s.enrollment), int64(s.config.bounds.maximumObjectBytes))
	before, parseErr := ParseCandidateEnrollmentRoot(context, beforeBytes, 64<<10)
	kind := contentString(intent["kind"])
	switch kind {
	case "admit":
		kind = "install-budget"
		if before != nil && before.budget != "" {
			kind = "update-enrolling"
			if before.status == "active" {
				kind = "advance-active"
			}
		}
	case "consume", "settle", "segment", "seal", "verify":
		kind = "update-enrolling"
		if before != nil && before.status == "active" {
			kind = "advance-active"
		}
	}
	if err != nil || parseErr != nil || CheckCandidateEnrollmentChange(before, after, kind) != nil {
		return errProviderIntegration
	}
	return nil
}
