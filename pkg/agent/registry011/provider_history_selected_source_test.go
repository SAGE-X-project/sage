package registry011

import (
	"strings"
	"testing"
	"time"
)

// Unminted numeric UNIT input only. The fixture's provisioned originals keep
// their unchanged ordinary profile; this copied config is never opened.
func historySelectedSourceConfig(t *testing.T) providerCustodyConfig {
	t.Helper()
	b, capture := historyBootstrapInputBounds(t)
	// A separate numeric selection covers the largest ACTIVE method's
	// 256KiB wire plus the original 16KiB failure header and fixed framing.
	// The fixture's actual 128KiB originals/profile are never changed/opened.
	b.maximumObjectBytes = 512 << 10
	in := historyCollectionInput(t)
	in.capture = capture
	manifest := historyCollectionManifest(t, in)
	b.history = &providerHistoryProgramBounds{
		collection: manifest, stages: manifest.counts.groups,
		catalogEntries: 4096, catalogNameBytes: 1044480,
	}
	if !b.validStructure() || b.valid() {
		t.Fatal("separate unminted source input lost structural validity")
	}
	policy, err := ParseProviderPolicyCandidate(providerPolicyTestBytes(t, providerPolicyTestObject()), 64<<10)
	if err != nil || policy == nil {
		t.Fatal("closed policy DATA for the selected source calculation", err)
	}
	return providerCustodyConfig{bounds: b, capture: capture, policy: policy, custodyDirectory: "custody", evidenceDirectory: "evidence"}
}

// Fixed offline clock DATA, never a constructed backend or a native sample.
func historySelectedSourceClock(config providerCustodyConfig) providerHistoryClockSelection {
	return providerHistoryClockSelection{
		id: "fixture-clock", epoch: admissionTestDigest("d"), policy: config.policy.policy,
		origin: time.Unix(100, 0), window: time.Millisecond, uncertainty: 2, maximum: 1 << 20,
	}
}

func TestProviderHistorySourceSnapshotDetachesSelectedPointers(t *testing.T) {
	config := historySelectedSourceConfig(t)
	got, ok := providerHistoryFreezeSourceConfig(config)
	if !ok || got.policy == config.policy || got.bounds.continuation == config.bounds.continuation ||
		got.bounds.methods == config.bounds.methods || got.bounds.tls == config.bounds.tls ||
		got.bounds.scan == config.bounds.scan || got.bounds.firstAdmission == config.bounds.firstAdmission ||
		got.bounds.history == config.bounds.history || got.bounds.history.collection == config.bounds.history.collection {
		t.Fatal("source snapshot did not detach every selected pointer", ok)
	}
	policy, ledger := *got.policy, *got.bounds.continuation
	methods, certificates := *got.bounds.methods, *got.bounds.tls
	scan, first, history, manifest := *got.bounds.scan, *got.bounds.firstAdmission, *got.bounds.history, *got.bounds.history.collection
	before := got
	if policy != *config.policy || ledger != *config.bounds.continuation || methods != *config.bounds.methods ||
		certificates != *config.bounds.tls || scan != *config.bounds.scan || first != *config.bounds.firstAdmission ||
		!got.bounds.history.collection.validSnapshot() || manifest.owner != got.bounds.history.collection ||
		manifest.input != config.bounds.history.collection.input || manifest.raw != config.bounds.history.collection.raw ||
		manifest.digest != config.bounds.history.collection.digest || manifest.counts != config.bounds.history.collection.counts {
		t.Fatal("detached source snapshot changed original values or its own manifest seal")
	}
	history.collection = config.bounds.history.collection
	if history != *config.bounds.history {
		t.Fatal("snapshot changed a selected history value")
	}
	history = *got.bounds.history
	config.policy.raw = "changed"
	config.bounds.continuation.maximumRecords--
	config.bounds.methods.scalar.body++
	config.bounds.tls.chains++
	config.bounds.scan.maximumAttempts++
	config.bounds.firstAdmission.work[2]++
	config.bounds.history.program = "changed"
	config.bounds.history.collection.input.maximumSelected++
	config.bounds.history.collection.raw = "changed"
	config.capture.HeaderBytes++
	config.identity.Generation++
	config.custodyDirectory = "changed"
	if got != before || *got.policy != policy || *got.bounds.continuation != ledger || *got.bounds.methods != methods ||
		*got.bounds.tls != certificates || *got.bounds.scan != scan || *got.bounds.firstAdmission != first ||
		*got.bounds.history != history || *got.bounds.history.collection != manifest || !got.bounds.history.collection.validSnapshot() {
		t.Fatal("later caller changes altered the detached source snapshot")
	}
	if got.bounds.history.derivation != nil || got.bounds.valid() {
		t.Fatal("source snapshot created execution authority")
	}
}

func TestProviderHistorySourceSnapshotRefusesIncompleteSelection(t *testing.T) {
	cases := []struct {
		name   string
		change func(*providerCustodyConfig)
	}{
		{"policy", func(c *providerCustodyConfig) { c.policy = nil }},
		{"empty-policy", func(c *providerCustodyConfig) { c.policy.raw = "" }},
		{"large-policy", func(c *providerCustodyConfig) { c.policy.raw = strings.Repeat("x", (64<<10)+1) }},
		{"ledger", func(c *providerCustodyConfig) { c.bounds.continuation = nil }},
		{"methods", func(c *providerCustodyConfig) { c.bounds.methods = nil }},
		{"certificates", func(c *providerCustodyConfig) { c.bounds.tls = nil }},
		{"scan", func(c *providerCustodyConfig) { c.bounds.scan = nil }},
		{"first", func(c *providerCustodyConfig) { c.bounds.firstAdmission = nil }},
		{"history", func(c *providerCustodyConfig) { c.bounds.history = nil }},
		{"manifest", func(c *providerCustodyConfig) { c.bounds.history.collection = nil }},
		{"derivation", func(c *providerCustodyConfig) { c.bounds.history.derivation = &providerHistoryProgramDerivation{} }},
		{"guardian", func(c *providerCustodyConfig) { c.bounds.guardian = &providerGuardianBounds{} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := historySelectedSourceConfig(t)
			tc.change(&config)
			if got, ok := providerHistoryFreezeSourceConfig(config); ok || got != (providerCustodyConfig{}) {
				t.Fatal("incomplete source selection returned a partial snapshot", ok)
			}
		})
	}
}

func TestProviderHistorySourceSnapshotReconstructsManifestInsteadOfTrustingSeal(t *testing.T) {
	for _, kind := range []string{"raw", "digest", "counts", "input"} {
		t.Run(kind, func(t *testing.T) {
			config := historySelectedSourceConfig(t)
			m := config.bounds.history.collection
			switch kind {
			case "raw":
				m.raw, m.sealedRaw = "changed", "changed"
			case "digest":
				m.digest, m.sealedDigest = admissionTestDigest("e"), admissionTestDigest("e")
			case "counts":
				m.counts.groups++
				m.sealedCounts = m.counts
				config.bounds.history.stages = m.counts.groups
			case "input":
				m.input.maximumSelected++
				m.sealed = m.input
			}
			if !m.validSnapshot() {
				t.Fatal("unit must reach genuine reconstruction beyond its equal-field seal")
			}
			if got, ok := providerHistoryFreezeSourceConfig(config); ok || got != (providerCustodyConfig{}) {
				t.Fatal("equal-field seal replaced genuine manifest reconstruction", ok)
			}
		})
	}
}

func TestProviderHistorySourceLimitsBindOneOriginalConfig(t *testing.T) {
	config := historySelectedSourceConfig(t)
	b := config.bounds
	before, scan, first, history, manifest := config, *b.scan, *b.firstAdmission, *b.history, *b.history.collection
	got, ok := providerHistorySourceLimitsFromConfig(config)
	if !ok || got.legacy.metadata != 16384 || got.legacy.objectBytes != 512<<10 || got.legacy.objectBytes != b.maximumObjectBytes ||
		got.legacy.firstObjects != 64 || got.legacy.firstRaw != 1<<20 || got.legacy.entries != uint64(b.maximumEntries) ||
		got.legacy.attempts != b.scan.maximumAttempts || got.legacy.chunks != 2 ||
		got.setupRecoveries != uint64(b.maximumSetupRecoveries) || got.catalogEntries != 4096 || got.catalogNameBytes != 1044480 ||
		got.directoryBytes != 15 || got.fileOverhead != b.fileOverheadBytes {
		t.Fatal("source inputs did not retain one original store and both path strings", ok, got)
	}
	if config != before || *b.scan != scan || *b.firstAdmission != first || *b.history != history || *b.history.collection != manifest || b.valid() {
		t.Fatal("numeric source binding changed originals or created a history derivation")
	}
	// Paths are byte strings in the original reflection/copy graph. Counting
	// code points or taking the filename catalog would understate this input.
	config.custodyDirectory, config.evidenceDirectory = "가", "é"
	if got, ok := providerHistorySourceLimitsFromConfig(config); !ok || got.directoryBytes != 5 {
		t.Fatal("directory byte lengths were replaced by character or catalog counts", ok, got.directoryBytes)
	}
}

func TestProviderHistorySelectedSourceComposesOneUnchangedSelection(t *testing.T) {
	config := historySelectedSourceConfig(t)
	b := config.bounds
	before, scan, first, history, manifest := config, *b.scan, *b.firstAdmission, *b.history, *b.history.collection
	clock := historySelectedSourceClock(config)
	got, ok := providerHistoryCompileSelectedSource(config, clock)
	if !ok || got.collection.groups != 40 || got.bootstrap.inputs != (providerHistoryBootstrapSourceInputs{1, 2, 1261568, 85256}) ||
		got.collection.future.work[0] != 1280 || got.collection.storage.groups != 40 || got.collection.storage.future.physical == 0 {
		t.Fatal("fresh prefix and full admitted future did not use one original selection", ok)
	}
	// These are composition checks against accepted children, not an independent
	// proof of their kernels or of whole native/physical profile feasibility.
	limits, limitsOK := providerHistorySourceLimitsFromConfig(config)
	prefix, prefixOK := providerHistoryCompileBootstrapSource(b, config.capture)
	future, futureOK := providerHistoryCollectionUniformSourceUpper(b.history.collection, *b.methods, *b.tls, b.maximumObjectBytes, limits)
	if !limitsOK || !prefixOK || !futureOK || got.limits != limits || got.clock != clock || got.bootstrap != prefix || got.collection != future {
		t.Fatal("selected composition replaced an original child or a complete admitted stage")
	}
	if config != before || *b.scan != scan || *b.firstAdmission != first || *b.history != history || *b.history.collection != manifest || b.valid() || b.history.derivation != nil {
		t.Fatal("numeric composition mutated the selected inputs or granted execution")
	}
}

func TestProviderHistorySourceBindingRefusesCrossSelectionAndPartialResults(t *testing.T) {
	base := historySelectedSourceConfig(t)
	for _, name := range []string{"history", "scan", "first", "guardian", "derivation", "attempt", "custody", "evidence", "manifest", "capture", "stage", "entries-zero", "entries-high", "names", "directories", "store"} {
		t.Run(name, func(t *testing.T) {
			config, history := base, *base.bounds.history
			config.bounds.history = &history
			switch name {
			case "history":
				config.bounds.history = nil
			case "scan":
				config.bounds.scan = nil
			case "first":
				config.bounds.firstAdmission = nil
			case "guardian":
				config.bounds.guardian = &providerGuardianBounds{}
			case "derivation":
				history.derivation = &providerHistoryProgramDerivation{}
			case "attempt":
				config.attempt = 1
			case "custody":
				config.custodyDirectory = ""
			case "evidence":
				config.evidenceDirectory = ""
			case "manifest":
				history.collection = nil
			case "capture":
				config.capture.Timeout--
			case "stage":
				history.stages--
			case "entries-zero":
				history.catalogEntries = 0
			case "entries-high":
				history.catalogEntries = (1 << 20) + 1
			case "names":
				history.catalogNameBytes--
			case "directories":
				config.custodyDirectory = strings.Repeat("x", 2<<20)
			case "store":
				config.bounds.maximumEntries = 0
			}
			if got, ok := providerHistorySourceLimitsFromConfig(config); ok || got != (providerHistoryRoleSourceLimits{}) {
				t.Fatal("mismatched source selection published role inputs", ok)
			}
			if got, ok := providerHistoryCompileSelectedSource(config, historySelectedSourceClock(base)); ok || got != (providerHistorySelectedSourceCost{}) {
				t.Fatal("mismatched source selection leaked an earlier compiled component", ok)
			}
		})
	}
}

func TestProviderHistorySelectedSourceRefusesLateChildWithoutPartialPrefix(t *testing.T) {
	base := historySelectedSourceConfig(t)
	if got, ok := providerHistoryCompileSelectedSource(base, historySelectedSourceClock(base)); !ok || got == (providerHistorySelectedSourceCost{}) {
		t.Fatal("late refusals require a positive complete selected-component baseline")
	}
	for _, name := range []string{"bootstrap", "collection", "failure-record"} {
		t.Run(name, func(t *testing.T) {
			config, first, history := base, *base.bounds.firstAdmission, *base.bounds.history
			config.bounds.firstAdmission, config.bounds.history = &first, &history
			switch name {
			case "bootstrap":
				first.work[2] = 1
			case "collection":
				in := history.collection.input
				in.firstRead.work = candidateWorkVector{2: 1, 10: 1}
				history.collection = historyCollectionManifest(t, in)
			case "failure-record":
				// Restore only the original arithmetic fixture's smaller object
				// ceiling. Its active failure record cannot fit; keep that guard.
				config.bounds.maximumObjectBytes = 128 << 10
			}
			if name != "bootstrap" {
				if prefix, ok := providerHistoryCompileBootstrapSource(config.bounds, config.capture); !ok || prefix == (providerHistoryCompiledBootstrapCost{}) {
					t.Fatal("late collection refusal must follow a positive original prefix")
				}
			}
			if _, ok := providerHistorySourceLimitsFromConfig(config); !ok {
				t.Fatal("late child scenario did not reach a positive selected-input precursor")
			}
			if got, ok := providerHistoryCompileSelectedSource(config, historySelectedSourceClock(base)); ok || got != (providerHistorySelectedSourceCost{}) {
				t.Fatal("a late child refusal published limits or an earlier positive prefix", ok)
			}
		})
	}
}

func TestProviderHistorySelectedSourceKeepsBothInspectionRolesAndTheirProofs(t *testing.T) {
	config := historySelectedSourceConfig(t)
	clock := historySelectedSourceClock(config)
	before, history, manifest, policy := config, *config.bounds.history, *config.bounds.history.collection, *config.policy
	got, ok := providerHistoryCompileSelectedSource(config, clock)
	if !ok || got.clock != clock || got.finalization.clockIDBytes != 13 || got.finalization.clocks != 2 {
		t.Fatal("closing work did not bind the same explicit clock identity", ok)
	}
	// Each failed pass may additionally read eight metadata inputs, three
	// packs and one unfinished member pair. A successful final pass cannot
	// borrow those rows, while both independent read selections share the
	// same original source limits and immutable manifest.
	if got.recovery.io.limits != got.limits || got.finalRead.io.limits != got.limits ||
		got.recovery.io.files.metadataPerPass != got.finalRead.io.files.metadataPerPass+8 ||
		got.recovery.io.files.packsPerPass != got.finalRead.io.files.packsPerPass+3 ||
		got.recovery.io.files.spoolPairsPerPass != got.finalRead.io.files.spoolPairsPerPass+1 {
		t.Fatal("inspection roles lost their separate selected original graphs")
	}
	// The fixture has four prospective mutations. Each may verify128 keys in
	// EACH of two fresh models, in EACH disjoint R/F reservation:2048 proofs.
	// This is not extra permission to dispatch, acquire segments or write.
	const perReadProofs, bothReadProofs = 1024, 2048
	priorProofs := got.bootstrap.source.selectedPreI.work[7] + got.bootstrap.source.startup.work[7] + got.collection.future.work[7]
	if got.recovery.envelope[7] != perReadProofs || got.finalWork[7] != perReadProofs || got.assigned.work[7] != priorProofs+bothReadProofs ||
		!providerHistoryReadWorkDomain(got.recovery.envelope) || !providerHistoryReadWorkDomain(got.finalWork) {
		t.Fatal("historical proof work was dropped, MAXed across held roles or excluded from its read domain")
	}
	// Forty whole32-member T windows plus the separate initial E1. The actual
	// grammar's242 requests must not shrink that prospective admitted SUM.
	if got.collection.groups != 40 || got.assigned.work[0] != 1281 || manifest.counts.requests != 242 {
		t.Fatal("whole future admission was replaced by successful actual requests")
	}
	for _, coordinate := range []int{0, 1, 8, 9} {
		if got.recovery.envelope[coordinate] != 0 || got.finalWork[coordinate] != 0 {
			t.Fatal("read-only source composition acquired a producer permission", coordinate)
		}
	}
	if got.finalWork[10] != got.finalRead.normal[10]+got.finalization.normal[10]+max(got.finalRead.incoming[10], got.finalization.incoming[10])+2 ||
		got.assigned.clocks != got.bootstrap.source.selectedPreI.clocks+got.bootstrap.source.startup.clocks+got.collection.future.clocks+2 {
		t.Fatal("sequential final phases duplicated an incoming guard or lost the two original samples")
	}
	if config != before || *config.bounds.history != history || *config.bounds.history.collection != manifest || *config.policy != policy ||
		config.bounds.valid() || config.bounds.history.derivation != nil {
		t.Fatal("joined source DATA mutated originals or created execution authority")
	}
}

func TestProviderHistorySelectedSourceRejectsUnmatchedClockAndPolicyData(t *testing.T) {
	base := historySelectedSourceConfig(t)
	clock := historySelectedSourceClock(base)
	if _, ok := providerHistoryCompileSelectedSource(base, clock); !ok {
		t.Fatal("clock refusals need a positive joined-source precursor")
	}
	for _, name := range []string{
		"nil-policy", "empty-policy", "large-policy", "malformed-policy", "array-policy", "missing-clock", "wrong-clock-type", "different-policy-id",
		"empty-clock-id", "large-clock-id", "different-clock-id", "empty-clock-policy", "different-clock-policy", "epoch", "origin",
		"zero-window", "negative-window", "uncertainty", "zero-capacity", "large-capacity",
	} {
		t.Run(name, func(t *testing.T) {
			config, policy, selected := base, *base.policy, clock
			config.policy = &policy
			switch name {
			case "nil-policy":
				config.policy = nil
			case "empty-policy":
				policy.raw = ""
			case "large-policy":
				policy.raw = strings.Repeat("x", (64<<10)+1)
			case "malformed-policy":
				policy.raw = "{"
			case "array-policy":
				policy.raw = "[]"
			case "missing-clock":
				policy.raw = "{}"
			case "wrong-clock-type":
				policy.raw = `{"clock":[]}`
			case "different-policy-id":
				policy.policy = admissionTestDigest("a")
			case "empty-clock-id":
				selected.id = ""
			case "large-clock-id":
				selected.id = strings.Repeat("x", 257)
			case "different-clock-id":
				selected.id = "other-clock"
			case "empty-clock-policy":
				selected.policy = ""
			case "different-clock-policy":
				selected.policy = admissionTestDigest("b")
			case "epoch":
				selected.epoch = "changed"
			case "origin":
				selected.origin = time.Time{}
			case "zero-window":
				selected.window = 0
			case "negative-window":
				selected.window = -time.Millisecond
			case "uncertainty":
				selected.uncertainty = 0
			case "zero-capacity":
				selected.maximum = 0
			case "large-capacity":
				selected.maximum = (1 << 20) + 1
			}
			if _, ok := providerHistorySourceLimitsFromConfig(config); !ok {
				t.Fatal("clock/policy refusal lost the independent structural source precursor")
			}
			if got, ok := providerHistoryCompileSelectedSource(config, selected); ok || got != (providerHistorySelectedSourceCost{}) {
				t.Fatal("unmatched clock/policy published a partial source result", ok)
			}
		})
	}
}

func TestProviderHistorySelectedSourceRejectsOverflowAfterPositiveChildRoles(t *testing.T) {
	base := historySelectedSourceConfig(t)
	clock := historySelectedSourceClock(base)
	if _, ok := providerHistoryCompileSelectedSource(base, clock); !ok {
		t.Fatal("joined overflow tests need a positive original selection")
	}
	for _, row := range []struct {
		name       string
		coordinate int
	}{{"decoded", 2}, {"proofs", 7}, {"writes", 9}, {"logical", 10}} {
		t.Run(row.name, func(t *testing.T) {
			config, scan := base, *base.bounds.scan
			config.bounds.scan = &scan
			scan.maximumPreIWork[row.coordinate] = ^uint64(0)
			// The unchanged structural inputs and separately selected roles
			// still compile. Only their enclosing full-role SUM cannot fit.
			limits, limitsOK := providerHistorySourceLimitsFromConfig(config)
			prefix, prefixOK := providerHistoryCompileBootstrapSource(config.bounds, config.capture)
			future, futureOK := providerHistoryCollectionUniformSourceUpper(config.bounds.history.collection, *config.bounds.methods, *config.bounds.tls,
				config.bounds.maximumObjectBytes, limits)
			if !limitsOK || !prefixOK || !futureOK || prefix.source.selectedPreI.work[row.coordinate] != ^uint64(0) || future.future.work[row.coordinate] == 0 {
				t.Fatal("overflow scenario failed before the enclosing positive-role sum")
			}
			if got, ok := providerHistoryCompileSelectedSource(config, clock); ok || got != (providerHistorySelectedSourceCost{}) {
				t.Fatal("overflow retained an earlier successful prefix or wrapped a role", ok)
			}
		})
	}
}

func TestProviderHistorySelectedSourcePricesThePolicyClockIdentifierBytes(t *testing.T) {
	for _, row := range []struct {
		name string
		size int
	}{{"one", 1}, {"ordinary", 13}, {"maximum", 256}} {
		t.Run(row.name, func(t *testing.T) {
			config := historySelectedSourceConfig(t)
			object := providerPolicyTestObject()
			object["clock"].(map[string]any)["id"] = strings.Repeat("c", row.size)
			policy, err := ParseProviderPolicyCandidate(providerPolicyTestBytes(t, object), 64<<10)
			if err != nil || policy == nil {
				t.Fatal("closed policy clock-ID precursor", err)
			}
			config.policy = policy
			clock := historySelectedSourceClock(config)
			clock.id = strings.Repeat("c", row.size)
			got, ok := providerHistoryCompileSelectedSource(config, clock)
			if !ok || got.clock != clock || got.finalization.clockIDBytes != uint64(row.size) || config.bounds.history.derivation != nil || config.bounds.valid() {
				t.Fatal("closing source used a default clock ID or qualified numeric DATA", ok)
			}
		})
	}
}
