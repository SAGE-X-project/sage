package registry011

import (
	"bytes"
	"strconv"
)

// commit is called only by the concrete integration coordinator after deriving
// the complete transition and its reservation. It never invokes caller code.
// These local/external writes are a protocol, not an atomic filesystem action.
// Any uncertainty permanently disables this handle and never refunds work.
func (s *providerCustodyStore) commit(kind, operation string, root *CandidateEnrollmentRoot, objects []providerIntegrationObject) (*providerCustodyAcknowledgment, error) {
	if s == nil {
		return nil, errProviderIntegration
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.valid() || s.config.bounds.continuation != nil || root == nil || root.raw == "" || root.identity != s.config.identity || !providerCustodyKind(kind) || !lowerHash(operation) || s.operations[operation] || s.sequence >= uint64(s.config.bounds.maximumCommits) {
		return nil, errProviderIntegration
	}
	b := s.config.bounds
	fresh := make([]providerIntegrationObject, 0, min(len(objects), b.maximumEntries))
	if len(objects) == 0 || len(objects) > b.maximumEntries {
		return nil, errProviderIntegration
	}
	containsRoot := false
	for _, object := range objects {
		if uint64(len(object.raw)) > b.maximumObjectBytes {
			return nil, errProviderIntegration
		}
		if object.kind == "enrollment" && object.digest == root.digest && object.raw == root.raw {
			containsRoot = true
		}
		key := object.kind + ":" + object.digest
		if ref, exists := s.objects[key]; exists {
			if uint64(len(object.raw)) != ref.bytes {
				return nil, errProviderIntegration
			}
			retained, err := s.read(s.evidence, providerObjectName(ref.kind, ref.digest), int64(ref.bytes))
			if err != nil || !bytes.Equal(retained, []byte(object.raw)) {
				return nil, errProviderIntegration
			}
			continue
		}
		fresh = append(fresh, object)
	}
	if !containsRoot || len(s.objects)+len(fresh) > b.maximumObjects {
		return nil, errProviderIntegration
	}
	inventory, err := newProviderIntegrationInventory(s.objectsRoot, fresh, b.maximumEntries, min(b.maximumStoredBytes, 64<<20))
	if err != nil {
		return nil, errProviderIntegration
	}
	if !s.event(providerEncodeEvent, uint64(len(inventory.raw))) || !s.event(providerHashEvent, uint64(len(inventory.raw)+len("sage-provider-custody-objects-v1"))) {
		return nil, errProviderIntegration
	}
	clock, err := s.sample()
	if err != nil {
		return nil, errProviderIntegration
	}
	var previous any
	if s.enrollment != "" {
		previous = s.enrollment
	}
	intent := providerIntegrationJSON(map[string]any{"format": "sage-provider-custody-intent-v1", "namespace": root.identity.Namespace, "instance_key": root.identity.Instance, "generation": strconv.FormatUint(root.identity.Generation, 10), "fence": strconv.FormatUint(root.fence, 10), "sequence": strconv.FormatUint(s.sequence+1, 10), "previous_commit": s.tip, "expected_enrollment_root": previous, "new_enrollment_root": root.digest, "objects_root": inventory.digest, "operation": operation, "kind": kind, "policy": root.identity.Policy, "clock": providerClockObject(clock), "metadata_reservation": providerCustodyMetadata(b)})
	m := s.checkIntent(intent, s.sequence+1)
	if m == nil || providerCheckNewEnrollment(s, root, kind) != nil {
		return nil, errProviderIntegration
	}
	commit := providerCustodyCommitBytes(m, intent)
	if !s.event(providerEncodeEvent, uint64(len(intent))) || !s.event(providerEncodeEvent, uint64(len(commit))) {
		return nil, errProviderIntegration
	}
	if uint64(len(intent)) > b.maximumMetadataPerCommit || uint64(len(commit)) > b.maximumMetadataPerCommit {
		return nil, errProviderIntegration
	}
	// Account for complete payloads, inventory, both local/external metadata and
	// the selected per-file allocation ceiling before the first immutable write.
	bytesUpper, ok := providerUpperSum(inventory.payloadBytes, uint64(len(inventory.raw)), uint64(2*len(intent)+len(commit)))
	allocation, mulOK := providerUpperProduct(uint64(len(fresh)+4), b.fileOverheadBytes)
	total, sumOK := providerUpperSum(bytesUpper, allocation)
	if !ok || !mulOK || !sumOK || !candidateCapacity(0, s.stored, total, b.maximumStoredBytes) {
		return nil, errProviderIntegration
	}
	s.failed = true
	sequence := s.sequence + 1
	if s.write(s.custody, providerCustodyName("intent", sequence), intent) != nil {
		return nil, errProviderIntegration
	}
	for _, object := range fresh {
		if s.write(s.evidence, providerObjectName(object.kind, object.digest), []byte(object.raw)) != nil {
			return nil, errProviderIntegration
		}
	}
	if s.write(s.evidence, "inventory-"+inventory.digest[2:], []byte(inventory.raw)) != nil ||
		s.write(s.evidence, providerCustodyName("pending", sequence), intent) != nil ||
		s.write(s.custody, providerCustodyName("commit", sequence), commit) != nil {
		return nil, errProviderIntegration
	}
	// Reload every selected retained byte and the whole independent prefix.
	// Sync alone, an in-memory root and a successful hash are never this ACK.
	readback := &providerCustodyStore{custody: s.custody, evidence: s.evidence, config: s.config, clock: s.clock, tip: s.config.anchor, objects: make(map[string]providerIntegrationObjectRef), operations: make(map[string]bool)}
	readback.activeMeter.Store(s.activeMeter.Load())
	empty, _ := newProviderIntegrationInventory("", nil, b.maximumEntries, min(b.maximumStoredBytes, 64<<20))
	readback.objectsRoot = empty.digest
	now, err := s.sample()
	if err != nil || readback.loadPrefix(now) != nil || readback.sequence != sequence || readback.tip != providerCandidateHash("sage-provider-custody-commit-v1", commit) || readback.enrollment != root.digest || readback.objectsRoot != inventory.digest {
		return nil, errProviderIntegration
	}
	ack := &providerCustodyAcknowledgment{s, s.tip, readback.tip, root.digest, operation, inventory.digest, root.fence, sequence, false}
	s.tip, s.enrollment, s.objectsRoot, s.sequence, s.stored = readback.tip, readback.enrollment, readback.objectsRoot, readback.sequence, readback.stored
	s.objects, s.operations, s.last, s.failed = readback.objects, readback.operations, ack, false
	s.bindings, s.replay = readback.bindings, readback.replay
	s.retainedRaw = readback.retainedRaw
	return ack, nil
}

func providerCheckNewEnrollment(s *providerCustodyStore, after *CandidateEnrollmentRoot, kind string) error {
	return providerCheckNewEnrollmentObserved(s, after, kind, nil)
}

func providerCheckNewEnrollmentObserved(s *providerCustodyStore, after *CandidateEnrollmentRoot, kind string, meter *providerWorkMeter) error {
	if s.enrollment == "" {
		if kind != "enroll" || after.sequence != 0 || after.fence != 1 || after.status != "enrolling" {
			return errProviderIntegration
		}
		return nil
	}
	raw, err := s.read(s.evidence, providerObjectName("enrollment", s.enrollment), int64(s.config.bounds.maximumObjectBytes))
	before, parseErr := parseCandidateEnrollmentRootObserved(&ProviderCandidateContext{s.config.identity, true}, raw, 64<<10, meter)
	if err != nil || parseErr != nil {
		return errProviderIntegration
	}
	switch kind {
	case "admit":
		kind = "install-budget"
		if before.budget != "" {
			kind = "update-enrolling"
			if before.status == "active" {
				kind = "advance-active"
			}
		}
	case "consume", "settle", "segment", "seal", "verify":
		kind = "update-enrolling"
		if before.status == "active" {
			kind = "advance-active"
		}
	}
	if CheckCandidateEnrollmentChange(before, after, kind) != nil {
		return errProviderIntegration
	}
	return nil
}

func (s *providerCustodyStore) confirms(ack *providerCustodyAcknowledgment, operation string, root *CandidateEnrollmentRoot) bool {
	if s == nil || ack == nil || root == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.valid() && ack.owner == s && !ack.recovered && s.last == ack && ack.commit == s.tip && ack.operation == operation && ack.enrollment == root.digest && ack.fence == root.fence && ack.sequence == s.sequence && ack.objectsRoot == s.objectsRoot
}
