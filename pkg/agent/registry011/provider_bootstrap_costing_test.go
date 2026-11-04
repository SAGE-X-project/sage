package registry011

import "testing"

// Existing isolated UNIT profiles only. A small positive arithmetic quote or
// the synthetic history witness does not qualify a production profile.
func providerBootstrapCostingTestBounds(t *testing.T) (providerIntegrationBounds, ProviderCaptureLimits) {
	t.Helper()
	first := providerFirstWriterBounds()
	config, _ := providerScanProvisionedFixtureWithFirstPolicy(t, &first, true)
	return config.bounds, config.capture
}

func TestProviderBootstrapCostingPreservesValidatedRuntimeQuotes(t *testing.T) {
	ordinary, capture := providerBootstrapCostingTestBounds(t)
	guardian, _ := providerGuardianConfigFixture(t)
	history := providerHistoryParentScalarFixture(t)
	for _, test := range []struct {
		name    string
		bounds  providerIntegrationBounds
		capture ProviderCaptureLimits
		kind    providerBootstrapCostingKind
		objects int
	}{
		{"ordinary", ordinary, capture, providerBootstrapCostingOrdinary, 7},
		{"guardian", guardian.bounds, guardian.capture, providerBootstrapCostingGuardian, 8},
		{"history", history.config.bounds, history.config.capture, providerBootstrapCostingHistory, 7},
	} {
		t.Run(test.name, func(t *testing.T) {
			in, ok := providerBootstrapCostingInputFromBounds(test.bounds)
			if !ok || !in.valid() || in.kind != test.kind || in.initialObjectCount() != test.objects {
				t.Fatal("complete immutable costing shape", ok, in.kind)
			}
			first, ok := in.firstAdmissionPlanUpper(1, 2, 0, 1)
			liveFirst, liveOK := providerFirstAdmissionPlanUpper(test.bounds, 1, 2, 0, 1)
			if !ok || !liveOK || first != liveFirst || first.owner != nil || first.used || first.stored != 0 || first.clocks != 18 {
				t.Fatal("first admission changed or acquired live authority", ok, liveOK)
			}
			operation, ok := in.firstOperationPlanUpper(test.capture, 1, 2, 0, 1)
			liveOperation, liveOK := providerFirstOperationPlanUpper(test.bounds, test.capture, 1, 2, 0, 1)
			if !ok || !liveOK || operation != liveOperation || operation.memory == 0 || operation.reservation.execution[0] != 1 || operation.reservation.remaining[0] != 0 {
				t.Fatal("nested operation quote changed", ok, liveOK)
			}
			raw, physical, ok := in.firstAdmissionStorageUpper()
			liveRaw, livePhysical, liveOK := providerFirstAdmissionStorageUpper(test.bounds)
			if !ok || !liveOK || raw != liveRaw || physical != livePhysical || physical != raw+21*in.fileOverheadBytes {
				t.Fatal("first admission storage changed")
			}
			storage, ok := in.firstOperationStorageUpper(test.capture, uint64(len(providerFirstOperationRequestWire)))
			liveStorage, liveOK := providerFirstOperationStorageUpper(test.bounds, test.capture, uint64(len(providerFirstOperationRequestWire)))
			selectedStorage, selectedOK := providerOperationStorageUpper(test.bounds, operation.capture, uint64(len(providerFirstOperationRequestWire)))
			if !ok || !liveOK || !selectedOK || storage != liveStorage || storage != selectedStorage || storage != operation.storage || storage.physical != storage.raw+32*in.fileOverheadBytes {
				t.Fatal("selected storage lost a distinct raw copy or allocation")
			}
		})
	}
}

func TestProviderBootstrapCostingBreaksDerivationCycleWithoutGrantingAuthority(t *testing.T) {
	b, capture := providerBootstrapCostingTestBounds(t)
	ordinary, ok := providerBootstrapCostingInputFromBounds(b)
	if !ok || !b.valid() {
		t.Fatal("ordinary structural control")
	}
	// An unminted history selects only its typed arithmetic graph. Every
	// existing execution wrapper must still require the finished derivation.
	b.history = &providerHistoryProgramBounds{}
	if b.valid() || !b.validStructure() {
		t.Fatal("structural DATA became a completed source witness")
	}
	history, ok := providerBootstrapCostingInputFromBounds(b)
	if !ok || history.kind != providerBootstrapCostingHistory || history.initialObjectCount() != 7 {
		t.Fatal("unminted typed history arithmetic remained circular")
	}
	plain, plainOK := ordinary.firstOperationPlanUpper(capture, 1, 2, 0, 1)
	typed, typedOK := history.firstOperationPlanUpper(capture, 1, 2, 0, 1)
	if !plainOK || !typedOK || plain.storage != typed.storage || plain.capture != typed.capture || plain.headroom != typed.headroom || typed.memory <= plain.memory {
		t.Fatal("history erased storage, serializer headroom or static layouts")
	}
	// C/X and S have independent 552Q and 571Q rows. This profile's global
	// object ceiling is wider than the closed first header schemas.
	typedN := max(history.maximumMetadataPerCommit, uint64(64<<10), typed.storage.transport, typed.storage.exchange, uint64(typed.capture.WireBytes))
	plainN := max(ordinary.maximumMetadataPerCommit, uint64(64<<10), ordinary.maximumObjectBytes, uint64(plain.capture.WireBytes))
	if plainN <= typedN || plain.reservation.execution[2]-typed.reservation.execution[2] != 552*(plainN-typedN) || plain.reservation.remaining[2]-typed.reservation.remaining[2] != 571*(plainN-typedN) {
		t.Fatal("history kind was replaced by temporary ordinary-profile arithmetic")
	}
	if p, ok := providerFirstAdmissionPlanUpper(b, 1, 2, 0, 1); ok || p != (providerFirstAdmissionPlan{}) {
		t.Fatal("unminted first admission acquired execution validity")
	}
	if p, ok := providerFirstOperationPlanUpper(b, capture, 1, 2, 0, 1); ok || p != (providerFirstOperationPlan{}) {
		t.Fatal("unminted operation acquired execution validity")
	}
	if raw, physical, ok := providerFirstAdmissionStorageUpper(b); ok || raw != 0 || physical != 0 {
		t.Fatal("unminted first storage bypassed the execution wrapper")
	}
	if p, ok := providerFirstOperationStorageUpper(b, capture, 262); ok || p != (providerFirstOperationStorage{}) {
		t.Fatal("unminted selected storage bypassed the execution wrapper")
	}
	if p, ok := providerOperationStorageUpper(b, typed.capture, 262); ok || p != (providerFirstOperationStorage{}) {
		t.Fatal("unminted operation storage bypassed the execution wrapper")
	}
	if b.history.derivation != nil {
		t.Fatal("costing manufactured a derivation")
	}
}

func TestProviderBootstrapCostingCopiesInputsBeforeOriginalMutation(t *testing.T) {
	b, capture := providerBootstrapCostingTestBounds(t)
	in, ok := providerBootstrapCostingInputFromBounds(b)
	before, quoteOK := in.firstOperationPlanUpper(capture, 1, 2, 0, 1)
	if !ok || !quoteOK {
		t.Fatal("complete original quote")
	}
	snapshot := in
	b.continuation.maximumResidentBytes = 0
	b.methods.header.wire = 0
	b.tls.certificates = 0
	b.scan.maximumAttempts = 0
	b.firstAdmission.work[10] = 0
	b.maximumMetadataPerCommit = 0
	after, ok := in.firstOperationPlanUpper(capture, 1, 2, 0, 1)
	if !ok || in != snapshot || after != before || b.valid() {
		t.Fatal("original pointers changed the value-only compilation input")
	}
}

func TestProviderBootstrapCostingRejectsIncompleteOriginalStructure(t *testing.T) {
	base, _ := providerBootstrapCostingTestBounds(t)
	for _, name := range []string{"first", "scan", "continuation", "methods", "tls", "mixed-kind", "metadata", "setup", "entries", "objects", "stored"} {
		t.Run(name, func(t *testing.T) {
			b := base
			switch name {
			case "first":
				b.firstAdmission = nil
			case "scan":
				b.scan = nil
			case "continuation":
				b.continuation = nil
			case "methods":
				b.methods = nil
			case "tls":
				b.tls = nil
			case "mixed-kind":
				b.history, b.guardian = &providerHistoryProgramBounds{}, &providerGuardianBounds{}
			case "metadata":
				b.maximumMetadataPerCommit = 0
			case "setup":
				b.setupWork[10] = 0
			case "entries":
				b.maximumEntries = 257
			case "objects":
				b.maximumObjects--
			case "stored":
				b.maximumStoredBytes = 1<<40 + 1
			}
			if in, ok := providerBootstrapCostingInputFromBounds(b); ok || in != (providerBootstrapCostingInput{}) {
				t.Fatal("invalid original structure produced a partial costing snapshot", name)
			}
		})
	}
}

func TestProviderBootstrapCostingKernelsRefuseInvalidDomainsAndLateOverflow(t *testing.T) {
	b, capture := providerBootstrapCostingTestBounds(t)
	in, ok := providerBootstrapCostingInputFromBounds(b)
	if !ok {
		t.Fatal("original structure")
	}
	for _, name := range []string{"kind", "continuation", "methods", "tls", "objects", "entries", "object", "stored", "metadata", "overhead", "attempts", "resident", "entry-overhead", "catalog", "names"} {
		t.Run(name, func(t *testing.T) {
			local := in
			switch name {
			case "kind":
				local.kind = 0
			case "continuation":
				local.continuation.maximumRecords = 0
			case "methods":
				local.methods.header.body = 0
			case "tls":
				local.tls.certificates = 0
			case "objects":
				local.maximumObjects--
			case "entries":
				local.maximumEntries = 0
			case "object":
				local.maximumObjectBytes = 0
			case "stored":
				local.maximumStoredBytes = 1<<40 + 1
			case "metadata":
				local.maximumMetadataPerCommit = 1<<20 + 1
			case "overhead":
				local.fileOverheadBytes = 0
			case "attempts":
				local.scan.maximumAttempts = 65
			case "resident":
				local.scan.maximumResidentBytes = local.continuation.maximumResidentBytes + 1
			case "entry-overhead":
				local.scan.filenameEntryOverhead = 0
			case "catalog":
				local.scan.maximumCatalogEntries = 32<<20 + 1
			case "names":
				local.scan.maximumCatalogNameBytes = 0
			}
			if local.valid() {
				t.Fatal("invalid kernel domain accepted", name)
			}
			if p, ok := local.firstAdmissionPlanUpper(1, 2, 0, 1); ok || p != (providerFirstAdmissionPlan{}) {
				t.Fatal("invalid input produced a first-admission quote", name)
			}
			if p, ok := local.firstOperationPlanUpper(capture, 1, 2, 0, 1); ok || p != (providerFirstOperationPlan{}) {
				t.Fatal("invalid input produced an operation quote", name)
			}
			if raw, physical, ok := local.firstAdmissionStorageUpper(); ok || raw != 0 || physical != 0 {
				t.Fatal("invalid input produced storage DATA", name)
			}
			if p, ok := local.firstOperationStorageUpper(capture, 262); ok || p != (providerFirstOperationStorage{}) {
				t.Fatal("invalid input produced selected storage DATA", name)
			}
			if p, ok := local.operationStorageUpper(capture, 262); ok || p != (providerFirstOperationStorage{}) {
				t.Fatal("invalid input produced operation storage DATA", name)
			}
		})
	}
	for _, args := range [][4]uint64{{0, 2, 0, 1}, {in.scan.maximumAttempts + 1, 2, 0, 1}, {1, 3, 0, 1}, {1, 2, in.maximumStoredBytes + 1, 1}, {1, 2, 0, 0}, {1, 2, 0, ^uint64(0)}, {1, 2, 0, in.scan.maximumResidentBytes}} {
		if p, ok := in.firstAdmissionPlanUpper(args[0], args[1], args[2], args[3]); ok || p != (providerFirstAdmissionPlan{}) {
			t.Fatal("invalid or overflowing first-admission arguments", args)
		}
		if p, ok := in.firstOperationPlanUpper(capture, args[0], args[1], args[2], args[3]); ok || p != (providerFirstOperationPlan{}) {
			t.Fatal("nested operation lost an invalid or late-overflow input", args)
		}
	}
}
