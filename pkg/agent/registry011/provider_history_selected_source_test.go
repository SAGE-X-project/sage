package registry011

import (
	"strings"
	"testing"
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
	return providerCustodyConfig{bounds: b, capture: capture, custodyDirectory: "custody", evidenceDirectory: "evidence"}
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
	got, ok := providerHistoryCompileSelectedSource(config)
	if !ok || got.collection.groups != 40 || got.bootstrap.inputs != (providerHistoryBootstrapSourceInputs{1, 2, 1261568, 85256}) ||
		got.collection.future.work[0] != 1280 || got.collection.storage.groups != 40 || got.collection.storage.future.physical == 0 {
		t.Fatal("fresh prefix and full admitted future did not use one original selection", ok)
	}
	// These are composition checks against accepted children, not an independent
	// proof of their kernels or of whole native/physical profile feasibility.
	limits, limitsOK := providerHistorySourceLimitsFromConfig(config)
	prefix, prefixOK := providerHistoryCompileBootstrapSource(b, config.capture)
	future, futureOK := providerHistoryCollectionUniformSourceUpper(b.history.collection, *b.methods, *b.tls, b.maximumObjectBytes, limits)
	if !limitsOK || !prefixOK || !futureOK || got.limits != limits || got.bootstrap != prefix || got.collection != future {
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
			if got, ok := providerHistoryCompileSelectedSource(config); ok || got != (providerHistorySelectedSourceCost{}) {
				t.Fatal("mismatched source selection leaked an earlier compiled component", ok)
			}
		})
	}
}

func TestProviderHistorySelectedSourceRefusesLateChildWithoutPartialPrefix(t *testing.T) {
	base := historySelectedSourceConfig(t)
	if got, ok := providerHistoryCompileSelectedSource(base); !ok || got == (providerHistorySelectedSourceCost{}) {
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
			if got, ok := providerHistoryCompileSelectedSource(config); ok || got != (providerHistorySelectedSourceCost{}) {
				t.Fatal("a late child refusal published limits or an earlier positive prefix", ok)
			}
		})
	}
}
