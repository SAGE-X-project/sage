package registry011

type providerBootstrapCostingKind uint8

const (
	providerBootstrapCostingOrdinary providerBootstrapCostingKind = iota + 1
	providerBootstrapCostingGuardian
	providerBootstrapCostingHistory
)

// Value-only inputs to the first admission/operation arithmetic. In particular,
// this type has no history bounds, derivation, live owner, store, reader or
// clock pointer. The history kind preserves its distinct typed work and static
// layouts without constructing a pretend derivation or validating a temporary
// history=nil execution profile. These numbers grant no execution authority.
type providerBootstrapCostingInput struct {
	kind                     providerBootstrapCostingKind
	continuation             providerLedgerBounds
	methods                  providerMethodBounds
	tls                      providerTLSBounds
	scan                     providerScanBounds
	maximumObjects           int
	maximumEntries           int
	maximumObjectBytes       uint64
	maximumStoredBytes       uint64
	maximumMetadataPerCommit uint64
	fileOverheadBytes        uint64
}

func providerBootstrapCostingInputFromBounds(b providerIntegrationBounds) (providerBootstrapCostingInput, bool) {
	if b.firstAdmission == nil || b.scan == nil || b.continuation == nil || b.methods == nil || b.tls == nil ||
		b.history != nil && b.guardian != nil || !b.validStructure() {
		return providerBootstrapCostingInput{}, false
	}
	kind := providerBootstrapCostingOrdinary
	if b.history != nil {
		kind = providerBootstrapCostingHistory
	} else if b.guardian != nil {
		kind = providerBootstrapCostingGuardian
	}
	in := providerBootstrapCostingInput{
		kind: kind, continuation: *b.continuation, methods: *b.methods, tls: *b.tls, scan: *b.scan,
		maximumObjects: b.maximumObjects, maximumEntries: b.maximumEntries,
		maximumObjectBytes: b.maximumObjectBytes, maximumStoredBytes: b.maximumStoredBytes,
		maximumMetadataPerCommit: b.maximumMetadataPerCommit, fileOverheadBytes: b.fileOverheadBytes,
	}
	if !in.valid() {
		return providerBootstrapCostingInput{}, false
	}
	return in, true
}

// Check every domain consumed by the kernels as well as the complete scalar
// continuation/method/TLS profiles. The constructor additionally checks the
// original store, scan and first-admission structural relationships. A kernel
// result cannot replace either those checks or the execution witness check.
func (b providerBootstrapCostingInput) valid() bool {
	return b.kind >= providerBootstrapCostingOrdinary && b.kind <= providerBootstrapCostingHistory &&
		b.continuation.valid() && b.methods.valid() && b.tls.valid() &&
		b.maximumObjects > 0 && uint64(b.maximumObjects) == b.continuation.maximumObjects &&
		b.maximumEntries > 0 && b.maximumEntries <= 256 &&
		b.maximumObjectBytes > 0 && b.maximumObjectBytes <= 32<<20 &&
		b.maximumStoredBytes > 0 && b.maximumStoredBytes <= 1<<40 && b.maximumObjectBytes <= b.maximumStoredBytes &&
		b.maximumMetadataPerCommit > 0 && b.maximumMetadataPerCommit <= 1<<20 &&
		b.fileOverheadBytes > 0 && b.fileOverheadBytes <= 1<<20 &&
		b.scan.maximumAttempts > 0 && b.scan.maximumAttempts <= 64 &&
		b.scan.maximumResidentBytes > 0 && b.scan.maximumResidentBytes <= b.continuation.maximumResidentBytes &&
		b.scan.filenameEntryOverhead > 0 && b.scan.filenameEntryOverhead <= 1<<20 &&
		b.scan.maximumCatalogEntries > 0 && b.scan.maximumCatalogEntries <= 32<<20 &&
		b.scan.maximumCatalogNameBytes > 0 && b.scan.maximumCatalogNameBytes <= 16<<30
}

func (b providerBootstrapCostingInput) initialObjectCount() int {
	if b.kind == providerBootstrapCostingGuardian {
		return 8
	}
	return 7
}
