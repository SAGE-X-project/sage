package registry011

import (
	"reflect"
	"testing"
)

func historyProfileProjectionConfig(t *testing.T) providerCustodyConfig {
	t.Helper()
	config := historySelectedSourceConfig(t)
	config.bounds.history.program = admissionTestDigest("a")
	config.bounds.history.costs = admissionTestDigest("b")
	return config
}

func historyProfileProjectionUpper(capture ProviderCaptureLimits) providerHistoryGroupUpper {
	upper := historyMultistageTestUpper(capture)
	upper.execution[0] = 32
	upper.execution[1] = 32 * uint64(capture.WireBytes+1)
	return upper
}

func TestProviderHistoryProfileProjectionPricesFinalSchemaWithoutDerivation(t *testing.T) {
	config := historyProfileProjectionConfig(t)
	b := config.bounds
	b.history.final = providerHistoryFinalBounds{read: providerHistoryRecoveryBounds{work: candidateWorkVector{2: 1, 10: 2}, maximumFiles: 1, maximumBytes: 1}, elapsedMS: 300000}
	before, history, scan, first, manifest := config, *b.history, *b.scan, *b.firstAdmission, *b.history.collection
	upper := historyProfileProjectionUpper(config.capture)
	oldRaw, oldDomain := providerIntegrationProfileBytes(b, config.capture)
	projection, ok := providerHistoryProjectFinalProfile(config, upper)
	if !ok || projection.domain != "sage-provider-store-profile-v6" || projection.domain != oldDomain ||
		projection.raw == "" || projection.raw == string(oldRaw) {
		t.Fatal("projection omitted the final admission schema or used another profile version", ok)
	}
	// Independently add the final field to the old parsed schema. Every field
	// is a decimal string; its size is not replaced by a digest or UNIT grant.
	expected := contentJSON(oldRaw).(map[string]any)
	store := expected["store"].(map[string]any)
	h := store["history"].(map[string]any)
	if _, present := h["collection_admission_maximum"]; present {
		t.Fatal("unminted ordinary profile unexpectedly has admission authority")
	}
	if h["successful_final_inspection"] == nil {
		t.Fatal("projected schema omitted the independent successful-final row")
	}
	h["collection_admission_maximum"] = map[string]any{
		"execution": candidateLedgerWorkObject(upper.execution), "custody": candidateLedgerWorkObject(upper.custody),
		"physical_bytes": "8388608", "retained_raw_bytes": "4194304", "payload_bytes": "2097152", "metadata_bytes": "2097152",
		"objects": "128", "files": "256", "clock_reads": "256",
	}
	if projection.raw != string(providerIntegrationJSON(expected)) {
		t.Fatal("projected final schema differs from independently extended ordinary bytes")
	}
	// Serialization parity only. This deliberately incomplete UNIT marker is
	// not valid and is NEVER opened, installed in the input or used to execute.
	copyBounds, copyHistory := b, history
	copyHistory.derivation = &providerHistoryProgramDerivation{admission: providerHistoryCollectionAdmissionBounds{maximum: upper}}
	copyBounds.history = &copyHistory
	finalRaw, finalDomain := providerIntegrationProfileBytes(copyBounds, config.capture)
	if projection.raw != string(finalRaw) || projection.domain != finalDomain || copyBounds.valid() ||
		providerCandidateHash(projection.domain, []byte(projection.raw)) != providerIntegrationProfile(copyBounds, config.capture) {
		t.Fatal("ordinary final serializer and unminted projection disagree, or a marker became valid")
	}
	if config != before || *b.history != history || *b.scan != scan || *b.firstAdmission != first || *b.history.collection != manifest || b.valid() || b.history.derivation != nil {
		t.Fatal("pure projection changed originals or installed an execution derivation")
	}
}

func TestProviderHistoryProfileProjectionRejectsMismatchedOrUnfundedInputs(t *testing.T) {
	base := historyProfileProjectionConfig(t)
	for _, name := range []string{"history", "program", "costs", "derivation", "manifest", "capture", "attempt", "stages", "upper", "count", "wire", "work-overflow", "metadata-limit"} {
		t.Run(name, func(t *testing.T) {
			config, history := base, *base.bounds.history
			config.bounds.history = &history
			upper := historyProfileProjectionUpper(config.capture)
			switch name {
			case "history":
				config.bounds.history = nil
			case "program":
				history.program = "unbounded or invalid program DATA"
			case "costs":
				history.costs = ""
			case "derivation":
				history.derivation = &providerHistoryProgramDerivation{}
			case "manifest":
				history.collection = nil
			case "capture":
				config.capture.Timeout--
			case "attempt":
				config.attempt = 1
			case "stages":
				history.stages--
			case "upper":
				upper = providerHistoryGroupUpper{}
			case "count":
				upper.execution[0] = 31
			case "wire":
				upper.execution[1]--
			case "work-overflow":
				upper.execution[10], upper.custody[10] = ^uint64(0), 1
			case "metadata-limit":
				config.bounds.maximumMetadataPerCommit = 1
			}
			if name == "metadata-limit" {
				if _, ok := providerHistorySourceLimitsFromConfig(config); !ok {
					t.Fatal("serialized-size refusal lost its positive source-selection precursor")
				}
			}
			if got, ok := providerHistoryProjectFinalProfile(config, upper); ok || got != (providerHistoryProfileProjection{}) {
				t.Fatal("invalid final profile projection leaked partial bytes", ok)
			}
		})
	}
}

func TestProviderHistoryProfileProjectionKeepsLegacyAndAdmissionOmissionRules(t *testing.T) {
	config := historyProfileProjectionConfig(t)
	upper := historyProfileProjectionUpper(config.capture)
	for _, mode := range []string{"unminted", "zero-admission", "owner-only-admission", "admission", "no-collection"} {
		t.Run(mode, func(t *testing.T) {
			history := *config.bounds.history
			switch mode {
			case "zero-admission":
				history.derivation = &providerHistoryProgramDerivation{}
			case "owner-only-admission":
				history.derivation = &providerHistoryProgramDerivation{}
				history.derivation.admission.owner = history.derivation
			case "admission":
				history.derivation = &providerHistoryProgramDerivation{admission: providerHistoryCollectionAdmissionBounds{maximum: upper}}
			case "no-collection":
				history.collection = nil
				history.derivation = &providerHistoryProgramDerivation{admission: providerHistoryCollectionAdmissionBounds{maximum: upper}}
			}
			object := history.object()
			_, present := object["collection_admission_maximum"]
			if present != (mode == "admission" || mode == "owner-only-admission") {
				t.Fatal("ordinary admission omission changed", mode)
			}
			if present && !reflect.DeepEqual(object["collection_admission_maximum"], history.derivation.admission.maximum.object()) {
				t.Fatal("ordinary admission fields changed")
			}
			// An explicit projected field still requires a collection manifest.
			projected := history.objectWithAdmissionMaximum(&upper)
			_, projectedPresent := projected["collection_admission_maximum"]
			if projectedPresent != (history.collection != nil) {
				t.Fatal("projection changed the collection-manifest boundary")
			}
		})
	}
	// The shared envelope must preserve the existing non-history version.
	b := config.bounds
	b.history = nil
	raw, domain := providerIntegrationProfileBytes(b, config.capture)
	object := contentJSON(raw).(map[string]any)
	store := object["store"].(map[string]any)
	if domain != "sage-provider-store-profile-v4" || object["format"] != "sage-provider-owned-profile-v4" || store["format"] != "sage-provider-store-profile-v4" || store["history"] != nil {
		t.Fatal("history projection changed the ordinary legacy profile")
	}
}
