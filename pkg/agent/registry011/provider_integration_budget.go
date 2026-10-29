package registry011

import (
	"math"
	"math/bits"
	"strconv"
)

func providerIntegrationProfileBytes(b providerIntegrationBounds, capture ProviderCaptureLimits) ([]byte, string) {
	format, domain := "sage-provider-owned-profile-v1", "sage-provider-store-profile-v1"
	if b.continuation != nil {
		format, domain = "sage-provider-owned-profile-v2", "sage-provider-store-profile-v2"
	}
	if b.scan != nil {
		format, domain = "sage-provider-owned-profile-v3", "sage-provider-store-profile-v3"
	}
	if b.firstAdmission != nil {
		format, domain = "sage-provider-owned-profile-v4", "sage-provider-store-profile-v4"
	}
	if b.guardian != nil {
		format, domain = "sage-provider-owned-profile-v5", "sage-provider-store-profile-v5"
	}
	if b.history != nil {
		format, domain = "sage-provider-owned-profile-v6", "sage-provider-store-profile-v6"
	}
	raw := providerIntegrationJSON(map[string]any{"format": format, "store": b.profileObject(), "capture": map[string]string{"timeout_ns": strconv.FormatInt(int64(capture.Timeout), 10), "header_bytes": strconv.Itoa(capture.HeaderBytes), "body_bytes": strconv.Itoa(capture.BodyBytes), "wire_bytes": strconv.Itoa(capture.WireBytes), "json_depth": strconv.Itoa(capture.JSONDepth)}})
	return raw, domain
}

func providerIntegrationProfile(b providerIntegrationBounds, capture ProviderCaptureLimits) string {
	raw, domain := providerIntegrationProfileBytes(b, capture)
	return providerCandidateHash(domain, raw)
}

// The observer charges each actual profile rebuild separately. It cannot
// turn a configured profile digest into physical host qualification.
func providerIntegrationProfileObserved(b providerIntegrationBounds, capture ProviderCaptureLimits, meter *providerWorkMeter) string {
	if meter == nil {
		return providerIntegrationProfile(b, capture)
	}
	if b.maximumMetadataPerCommit == 0 || b.maximumMetadataPerCommit > 1<<20 || !meter.preflight(providerEncodeEvent, b.maximumMetadataPerCommit) {
		return ""
	}
	raw, domain := providerIntegrationProfileBytes(b, capture)
	if len(raw) == 0 || uint64(len(raw)) > b.maximumMetadataPerCommit || !meter.event(providerEncodeEvent, uint64(len(raw))) {
		return ""
	}
	return providerReplayDigest(domain, raw, meter)
}

func (b providerIntegrationBounds) profileObject() map[string]any {
	values := map[string]uint64{"maximum_records": uint64(b.ledger.MaximumRecords), "maximum_payload_bytes": b.ledger.MaximumPayloadBytes, "maximum_metadata_bytes": b.ledger.MaximumMetadataBytes, "maximum_commits": uint64(b.maximumCommits), "maximum_objects": uint64(b.maximumObjects), "maximum_entries": uint64(b.maximumEntries), "maximum_object_bytes": b.maximumObjectBytes, "maximum_stored_bytes": b.maximumStoredBytes, "maximum_metadata_per_commit": b.maximumMetadataPerCommit, "file_overhead_bytes": b.fileOverheadBytes, "maximum_setup_recoveries": uint64(b.maximumSetupRecoveries), "setup_live": b.setupLive, "setup_retained": b.setupRetained, "setup_maximum_clock_reads": b.setupMaximumClockReads, "setup_maximum_duration_ms": b.setupMaximumDurationMS}
	out := map[string]any{"format": "sage-provider-store-profile-v1", "setup_work": candidateLedgerWorkObject(b.setupWork)}
	if b.continuation != nil {
		out["format"], out["continuation"] = "sage-provider-store-profile-v2", b.continuation.object()
		if b.methods != nil {
			out["methods"] = b.methods.object()
		}
		if b.tls != nil {
			out["tls"] = b.tls.object()
		}
	}
	if b.scan != nil {
		out["format"], out["scan"] = "sage-provider-store-profile-v3", b.scan.object()
	}
	if b.firstAdmission != nil {
		out["format"], out["first_admission"] = "sage-provider-store-profile-v4", b.firstAdmission.object()
		out["live_accounting"] = providerFirstAdmissionLiveAccounting
		out["scan_receipt_accounting"] = "two-original-copies-v2-readback-codec-and-final-catalog-v1"
		out["bootstrap_accounting"] = "same-open-meter-through-original-enrollment-prepaid-v1"
	}
	if b.guardian != nil {
		out["format"], out["guardian"] = "sage-provider-store-profile-v5", b.guardian.object()
		out["continuation_accounting"] = "pre-h0-guardian-hold-same-live-original-models-v1"
	}
	if b.history != nil {
		out["format"], out["history"] = "sage-provider-store-profile-v6", b.history.object()
		out["continuation_accounting"] = "pre-h0-whole-program-independent-parent-v1"
	}
	for name, value := range values {
		out[name] = strconv.FormatUint(value, 10)
	}
	return out
}

// These are proposed inputs to the connected implementation, not selected
// operating defaults. Filesystem overhead and profile artifacts must be
// independently qualified before an actual Agent selection can use them.
type providerIntegrationBounds struct {
	ledger CandidateLedgerBounds
	// Distinct private lifetime, never inferred or enabled by exceeding V1.
	continuation             *providerLedgerBounds
	methods                  *providerMethodBounds
	tls                      *providerTLSBounds
	scan                     *providerScanBounds
	firstAdmission           *providerFirstAdmissionBounds
	guardian                 *providerGuardianBounds
	history                  *providerHistoryProgramBounds
	maximumCommits           int
	maximumObjects           int
	maximumEntries           int
	maximumObjectBytes       uint64
	maximumStoredBytes       uint64
	maximumMetadataPerCommit uint64
	fileOverheadBytes        uint64
	maximumSetupRecoveries   int
	setupWork                candidateWorkVector
	setupLive, setupRetained uint64
	setupMaximumClockReads   uint64
	setupMaximumDurationMS   uint64
}

func (b providerIntegrationBounds) valid() bool {
	commits, objects, entries := uint64(65536), uint64(65536), 64
	if b.continuation != nil {
		c := b.continuation
		if !c.valid() || b.methods == nil || !b.methods.valid() || b.tls == nil || !b.tls.valid() || b.maximumCommits != int(c.maximumCommits) || b.maximumObjects != int(c.maximumObjects) || b.ledger.MaximumRecords != int(c.pageRecords) || b.ledger.MaximumPayloadBytes != min(c.pagePayloadBytes, 16<<20) || b.ledger.MaximumMetadataBytes != min(c.pageMetadataBytes, 64<<20) {
			return false
		}
		commits, objects, entries = c.maximumCommits, c.maximumObjects, 256
	} else if b.methods != nil || b.tls != nil {
		return false
	}
	if b.scan != nil && (b.continuation == nil || !b.scan.valid(b)) {
		return false
	}
	if b.firstAdmission != nil && (b.scan == nil || !b.firstAdmission.valid(b)) {
		return false
	}
	if b.guardian != nil && (b.firstAdmission == nil || !b.guardian.valid(b)) {
		return false
	}
	if b.history != nil && !b.history.valid(b) {
		return false
	}
	return candidateLedgerBoundsValid(b.ledger) && b.maximumCommits > 0 && uint64(b.maximumCommits) <= commits && b.maximumObjects > 0 && uint64(b.maximumObjects) <= objects && b.maximumEntries > 0 && b.maximumEntries <= entries &&
		b.maximumObjectBytes > 0 && b.maximumObjectBytes <= 32<<20 && b.maximumStoredBytes > 0 && b.maximumStoredBytes <= 1<<40 && b.maximumObjectBytes <= b.maximumStoredBytes &&
		b.maximumMetadataPerCommit > 0 && b.maximumMetadataPerCommit <= 1<<20 && b.fileOverheadBytes > 0 && b.fileOverheadBytes <= 1<<20 &&
		b.maximumSetupRecoveries >= 0 && b.maximumSetupRecoveries <= 64 && b.setupWork[0] == 0 && b.setupWork[1] == 0 && b.setupWork[9] > 0 && b.setupWork[10] > 0 && b.setupLive > 0 && b.setupRetained > 0 && b.setupRetained <= b.setupLive && b.setupLive <= b.maximumStoredBytes &&
		b.setupMaximumClockReads > 0 && b.setupMaximumClockReads <= 1<<20 && b.setupMaximumDurationMS > 0 && b.setupMaximumDurationMS <= 300000
}

type providerPrimitiveBounds struct {
	capture                                    ProviderCaptureLimits
	blocks, transactions, frames, logs, proofs uint64
	maximumCreatedObjects                      uint64
}

type providerOperationReservation struct {
	work                 candidateWorkVector
	execution, remaining candidateWorkVector
	live, retained       uint64
}

// providerEstimateOperation charges a conservative upper bound before the
// owned primitive. It takes no caller-reported actual-work measurement. The
// current prefix size is the store's owned retained byte counter, not JSON.
// Every result still needs the ledger's operation/segment/bootstrap admission.
// No reservation or matching arithmetic is a custody ACK or work permit.
func providerEstimateOperation(store providerIntegrationBounds, primitive providerPrimitiveBounds, requestBytes, prefixBytes uint64, rpc, newSegment bool) (providerOperationReservation, error) {
	if !store.valid() || !primitive.capture.valid() || primitive.maximumCreatedObjects == 0 || primitive.maximumCreatedObjects > uint64(store.maximumEntries) || prefixBytes > store.maximumStoredBytes || rpc && (requestBytes == 0 || requestBytes >= uint64(primitive.capture.WireBytes)) || !rpc && requestBytes != 0 {
		return providerOperationReservation{}, errProviderIntegration
	}
	// The wire definition is the existing complete plaintext HTTP exchange.
	// TLS/kernel traffic is excluded from that counter; bounded TLS/clock/store
	// primitive overhead remains covered by the fixed logical work allowance.
	response := uint64(primitive.capture.WireBytes)
	if !rpc {
		response = uint64(primitive.capture.BodyBytes)
	}
	// Retain request/response plus bounded copies, all commit metadata, object
	// files, local pending/readback and independently retained custody records.
	// An actual filesystem profile must qualify the declared per-file upper
	// bound; serial lengths alone do not establish physical disk allocation.
	newBytes, ok := providerUpperSum(requestBytes, response)
	if !ok {
		return providerOperationReservation{}, errProviderIntegration
	}
	newBytes, ok = providerUpperProduct(newBytes, 4)
	if !ok {
		return providerOperationReservation{}, errProviderIntegration
	}
	metadata, ok := providerUpperProduct(store.maximumMetadataPerCommit, 12)
	if !ok {
		return providerOperationReservation{}, errProviderIntegration
	}
	files, ok := providerUpperSum(primitive.maximumCreatedObjects, 20)
	if !ok {
		return providerOperationReservation{}, errProviderIntegration
	}
	allocation, ok := providerUpperProduct(files, store.fileOverheadBytes)
	if !ok {
		return providerOperationReservation{}, errProviderIntegration
	}
	disk, ok := providerUpperSum(newBytes, metadata, allocation)
	if !ok || disk > store.maximumStoredBytes {
		return providerOperationReservation{}, errProviderIntegration
	}
	// Quadratic byte allowance conservatively bounds the selected finite JSON
	// parse/canonicalization/copy path rather than treating response length as
	// an exact CPU measurement. Revalidation hashes the bounded retained prefix;
	// every new attempt obtains another reservation and never refunds it here.
	processing, ok := providerUpperSum(uint64(primitive.capture.BodyBytes), metadata)
	if !ok {
		return providerOperationReservation{}, errProviderIntegration
	}
	processing, ok = providerUpperProduct(processing, processing)
	if !ok {
		return providerOperationReservation{}, errProviderIntegration
	}
	prefixWork, ok := providerUpperProduct(prefixBytes, store.maximumObjectBytes)
	if !ok {
		return providerOperationReservation{}, errProviderIntegration
	}
	prefixWork, ok = providerUpperProduct(prefixWork, 64)
	if !ok {
		return providerOperationReservation{}, errProviderIntegration
	}
	units, ok := providerUpperSum(processing, prefixWork, disk, 65536)
	if !ok {
		return providerOperationReservation{}, errProviderIntegration
	}
	decoded, ok := providerUpperProduct(prefixBytes, 4)
	if !ok {
		return providerOperationReservation{}, errProviderIntegration
	}
	decoded, ok = providerUpperSum(decoded, uint64(primitive.capture.BodyBytes), metadata)
	if !ok {
		return providerOperationReservation{}, errProviderIntegration
	}
	execution := candidateWorkVector{0, 0, decoded, primitive.blocks, primitive.transactions, primitive.frames, primitive.logs, primitive.proofs, 0, disk, units}
	if rpc {
		execution[0], execution[1] = 1, uint64(primitive.capture.WireBytes)
		for i := 3; i <= 7; i++ {
			execution[i] = 0 // Raw RPC capture does not replay a block or transaction.
		}
	}
	if newSegment {
		execution[8] = 1
	}
	remaining := candidateWorkVector{0, 0, decoded, 0, 0, 0, 0, 0, 0, disk, units}
	var total candidateWorkVector
	for i := range total {
		total[i], ok = providerUpperSum(execution[i], remaining[i])
		if !ok {
			return providerOperationReservation{}, errProviderIntegration
		}
	}
	capacity, ok := providerUpperProduct(disk, 2)
	if !ok || capacity > store.maximumStoredBytes {
		return providerOperationReservation{}, errProviderIntegration
	}
	return providerOperationReservation{work: total, execution: execution, remaining: remaining, live: capacity, retained: capacity}, nil
}

func providerUpperProduct(a, b uint64) (uint64, bool) {
	hi, lo := bits.Mul64(a, b)
	return lo, hi == 0
}

func providerUpperSum(values ...uint64) (uint64, bool) {
	var sum uint64
	for _, n := range values {
		if !candidateCapacity(0, sum, n, math.MaxUint64) {
			return 0, false
		}
		sum += n
	}
	return sum, true
}
