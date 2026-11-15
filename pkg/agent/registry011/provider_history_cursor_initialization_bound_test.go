package registry011

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// These are inert constructor/configuration DATA tests, with the existing
// private UNIT meter. No store/clock/RPC/transaction or Source is opened.
func TestProviderHistoryJournalConfigurationKeepsOriginalJSONAndActualCharge(t *testing.T) {
	base := journalTestConfig(t, replayTestInput(t))
	for _, mode := range []string{"ordinary", "escaped-maximum"} {
		t.Run(mode, func(t *testing.T) {
			c := base
			if mode == "escaped-maximum" {
				c.Agent = strings.Repeat("a", 64)
				c.SourceID = strings.Repeat("\x00", 256)
				c.PolicyGeneration = strings.Repeat("<", 256)
				c.Observation.PolicyGeneration = c.PolicyGeneration
				c.Observation.ClockID = strings.Repeat("\x01", 256)
				c.Observation.MaxClockUncertaintyMS = 1<<63 - 1
				c.Observation.MaxAcquisitionMS = 1<<63 - 1
				c.Observation.MaxObservationAgeMS = 1<<63 - 1
				c.Observation.MaxFinalizedLagMS = 1<<63 - 1
				c.Observation.MaxAncestryHeaders = 1 << 20
				c.Deployment.Number, c.Deployment.Timestamp = ^uint64(0), 1<<53-1
				c.MaxBatches, c.MaxClaims, c.MaxIdentities = 1<<20, 1<<20, 1<<20
				c.MaxStoredBytes = 1 << 40
			}
			original := c
			// Independently retain the pre-observer json.Marshal schema. This
			// compares actual original bytes, not two calls through one wrapper.
			want, err := json.Marshal(struct {
				Format string
				Config JournalConfig
			}{"sage-replay-journal-v1", c})
			if err != nil || len(want) > 9929 {
				t.Fatal("closed original JSON envelope", err, len(want))
			}
			legacy, legacyChecks, err := journalConfiguration(c)
			if err != nil || !bytes.Equal(legacy, want) || legacyChecks == nil || legacyChecks.limits != c.Observation {
				t.Fatal("legacy journal bytes or copied limits changed", err)
			}
			m := providerReplayUnitMeter()
			got, checks, err := journalConfigurationObserved(c, m)
			copyBytes := uint64(reflect.TypeFor[ObservationChecks]().Size())
			expected := candidateWorkVector{10: copyBytes + 1 + uint64(len(want))*uint64(len(want)) + 1}
			if err != nil || !bytes.Equal(got, want) || checks == nil || checks.limits != c.Observation || checks == legacyChecks ||
				m.failed || m.actual != expected || c != original || m.actual[10] > providerHistoryCursorInitializationUpper().configuration[10] {
				t.Fatal("original encoder was omitted, repeated, mutated or charged at preflight size", err, m.actual, expected)
			}
		})
	}
}

func TestProviderHistoryJournalConfigurationSchemaRetainsFiniteEnvelope(t *testing.T) {
	// Count the ACTUAL encoder schema recursively, rejecting any unbounded
	// container/pointer/custom tag before accepting the independent literal.
	root := reflect.TypeFor[struct {
		Format string
		Config JournalConfig
	}]()
	objects, keys, keyBytes, stringsCount, numbers := 0, 0, 0, 0, 0
	var walk func(reflect.Type)
	walk = func(typ reflect.Type) {
		switch typ.Kind() {
		case reflect.Struct:
			objects++
			for i := range typ.NumField() {
				field := typ.Field(i)
				if !field.IsExported() || field.Anonymous || field.Tag != "" {
					t.Fatal("closed default JSON field contract changed", field.Name)
				}
				keys++
				keyBytes += len(field.Name)
				walk(field.Type)
			}
		case reflect.String:
			stringsCount++
		case reflect.Int, reflect.Int64, reflect.Uint64:
			numbers++
		default:
			t.Fatal("unbounded or new configuration kind", typ)
		}
	}
	walk(root)
	if objects != 6 || keys != 36 || keyBytes != 387 || stringsCount != 12 || numbers != 19 ||
		174+387+6*1498+19*20 != 9929 || providerHistoryJournalEncodeLimit != 65536 {
		t.Fatal("configuration source envelope needs fresh derivation", objects, keys, keyBytes, stringsCount, numbers)
	}
}

func TestProviderHistoryJournalEncodeShortageRetainsEarlierCopy(t *testing.T) {
	c := journalTestConfig(t, replayTestInput(t))
	m := providerReplayUnitMeter()
	if !m.event(providerCopyEvent, 17) {
		t.Fatal("genuine earlier unit work")
	}
	before := m.actual
	copyWork := uint64(reflect.TypeFor[ObservationChecks]().Size()) + 1
	m.upper[10] = before[10] + copyWork + 4294967297 - 1
	raw, checks, err := journalConfigurationObserved(c, m)
	want := before
	want[10] += copyWork
	if err == nil || raw != nil || checks != nil || !m.failed || m.actual != want {
		t.Fatal("encoder preflight should fail after real earlier copies, not erase or charge it", err, m.actual, want)
	}
	if raw, checks, err = journalConfigurationObserved(c, m); err == nil || raw != nil || checks != nil || m.actual != want {
		t.Fatal("failed original observer renewed configuration work")
	}
}

func TestProviderHistoryObservedJournalKeepsInvalidAndStoredByteRefusal(t *testing.T) {
	base := journalTestConfig(t, replayTestInput(t))
	for _, mode := range []string{"clock", "registry", "source-utf8", "long-source", "generation", "long-clock", "replay", "storage"} {
		t.Run(mode, func(t *testing.T) {
			c := base
			switch mode {
			case "clock":
				c.Observation.MaxAcquisitionMS = 0
			case "registry":
				c.Registry = "eip155:1:" + SepoliaRegistryAddress
			case "source-utf8":
				c.SourceID = string([]byte{0xff})
			case "long-source":
				c.SourceID = strings.Repeat("s", 257)
			case "generation":
				c.Observation.PolicyGeneration += "-different"
			case "long-clock":
				c.Observation.ClockID = strings.Repeat("c", 257)
			case "replay":
				c.Replay.MaxBlocks = 0
			case "storage":
				c.MaxStoredBytes = 1
			}
			original := c
			if raw, checks, err := journalConfiguration(c); err == nil || raw != nil || checks != nil {
				t.Fatal("original invalid journal configuration accepted", mode)
			}
			m := providerReplayUnitMeter()
			raw, checks, err := journalConfigurationObserved(c, m)
			want := candidateWorkVector{10: uint64(reflect.TypeFor[ObservationChecks]().Size()) + 1}
			if mode == "storage" {
				encoded, e := json.Marshal(struct {
					Format string
					Config JournalConfig
				}{"sage-replay-journal-v1", c})
				if e != nil {
					t.Fatal(e)
				}
				want[10] += uint64(len(encoded))*uint64(len(encoded)) + 1
			}
			if err == nil || raw != nil || checks != nil || m.failed || m.actual != want || c != original {
				t.Fatal("invalid configuration lost its exact attempted-work prefix", mode, err, m.actual, want)
			}
			if mode == "storage" {
				// A real storage/config error does not poison the budget. Keep
				// the same already charged meter for a separate valid DATA call.
				raw, checks, err = journalConfigurationObserved(base, m)
				want[10] += uint64(reflect.TypeFor[ObservationChecks]().Size()) + 1 + uint64(len(raw))*uint64(len(raw)) + 1
				if err != nil || len(raw) == 0 || checks == nil || checks.limits != base.Observation || m.failed || m.actual != want {
					t.Fatal("storage refusal discarded actuals or poisoned a healthy original observer", err, m.actual, want)
				}
			}
		})
	}
}

func TestProviderHistoryFirstCursorUsesSameObserverAndOriginalSelection(t *testing.T) {
	in := historyCollectionInput(t)
	manifest := historyCollectionManifest(t, in)
	for _, mode := range []string{"deployment", "ancestry"} {
		t.Run(mode, func(t *testing.T) {
			selected := in.journal.Deployment
			if mode == "ancestry" {
				selected = historyFirstFinalizedHeader()
				selected.Number = in.maximumSelected
			}
			original, sealed, digest := manifest.input, manifest.sealed, manifest.digest
			m := providerReplayUnitMeter()
			c, err := newProviderHistoryCollectionCursorObserved(admissionTestDigest("a"), manifest, selected, m)
			if err != nil || c == nil || !c.valid() || c.semantic.observer != m || c.semantic.sealedObserver != m ||
				c.selection.selected != selected || c.semantic.selection.selected != selected || c.semantic.selection.config != in.journal ||
				len(c.spine) != 1 || cap(c.spine) != 1 || c.spine[0] != selected || c.owner != c || c.semantic.owner != c.semantic {
				t.Fatal("constructor replaced the original selection or observer", err)
			}
			legacy, err := newProviderHistorySemanticFold(admissionTestDigest("a"), in.journal, selected, in.blocks, in.versions)
			if err != nil || legacy.observer != nil || legacy.sealedObserver != nil || legacy.selection != c.semantic.selection || legacy.last != c.semantic.last {
				t.Fatal("observed semantic constructor changed the original DATA", err)
			}
			encoded, _, err := journalConfiguration(in.journal)
			if err != nil {
				t.Fatal(err)
			}
			fixed := uint64(reflect.TypeFor[providerHistoryCollectionCursor]().Size()+reflect.TypeFor[FinalizedHeader]().Size()) + 1 +
				uint64(reflect.TypeFor[providerHistorySemanticFold]().Size()) + 1 + uint64(reflect.TypeFor[ObservationChecks]().Size()) + 1
			want := candidateWorkVector{10: 265 + 265 + fixed + uint64(len(encoded))*uint64(len(encoded)) + 1}
			if m.failed || m.actual != want || m.actual[10] > providerHistoryCursorInitializationUpper().cursor[10] ||
				manifest.input != original || manifest.sealed != sealed || manifest.digest != digest || in != original {
				t.Fatal("distinct initialization invocation omitted/repeated or input mutated", m.actual, want)
			}
			if mode == "deployment" && (c.mode != "content" || c.low != selected.Number || c.high != selected.Number || c.next != selected.Number) ||
				mode == "ancestry" && (c.mode != "ancestry" || c.next != selected.Number-1 || c.high != c.next || c.low != selected.Number-4) {
				t.Fatal("original first-cursor phase/window changed")
			}
		})
	}
}

func TestProviderHistoryCursorInitializationShortagesKeepGenuinePrefixes(t *testing.T) {
	in := historyCollectionInput(t)
	manifest := historyCollectionManifest(t, in)
	selected := in.journal.Deployment
	cursorWork := uint64(reflect.TypeFor[providerHistoryCollectionCursor]().Size()+reflect.TypeFor[FinalizedHeader]().Size()) + 1
	foldWork := uint64(reflect.TypeFor[providerHistorySemanticFold]().Size()) + 1
	checksWork := uint64(reflect.TypeFor[ObservationChecks]().Size()) + 1
	for _, tc := range []struct {
		name         string
		prefix, next uint64
	}{
		{"cursor-backing", 530, cursorWork},
		{"semantic-copy", 530 + cursorWork, foldWork},
		{"checks-copy", 530 + cursorWork + foldWork, checksWork},
		{"journal-preflight", 530 + cursorWork + foldWork + checksWork, 4294967297},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := providerReplayUnitMeter()
			if !m.event(providerCopyEvent, 11) {
				t.Fatal("original nonzero prefix")
			}
			before := m.actual
			m.upper[10] = before[10] + tc.prefix + tc.next - 1
			c, err := newProviderHistoryCollectionCursorObserved(admissionTestDigest("a"), manifest, selected, m)
			want := before
			want[10] += tc.prefix
			if err == nil || c != nil || !m.failed || m.actual != want || manifest.input != in {
				t.Fatal("initialization failure changed the prior observed prefix", err, m.actual, want)
			}
			if c, err = newProviderHistoryCollectionCursorObserved(admissionTestDigest("a"), manifest, selected, m); err == nil || c != nil || m.actual != want {
				t.Fatal("failed constructor installed a replacement observer")
			}
		})
	}
}

func TestProviderHistoryCursorInitializationComponentsAndRepetitions(t *testing.T) {
	one := providerHistoryCursorInitializationUpper()
	checks, fold := uint64(reflect.TypeFor[ObservationChecks]().Size()), uint64(reflect.TypeFor[providerHistorySemanticFold]().Size())
	cursor := uint64(reflect.TypeFor[providerHistoryCollectionCursor]().Size() + reflect.TypeFor[FinalizedHeader]().Size())
	wantConfiguration := candidateWorkVector{10: checks + 4294967298}
	wantSemantic := candidateWorkVector{10: fold + checks + 4294967299}
	wantCursor := candidateWorkVector{10: cursor + fold + checks + 4294967830}
	if one.configuration != wantConfiguration || one.semantic != wantSemantic || one.cursor != wantCursor {
		t.Fatal("independent fixed preflight/copy literal", one, wantConfiguration, wantSemantic, wantCursor)
	}
	for _, tc := range []struct{ groups, requests, calls uint64 }{{1, 1, 14}, {1, 32, 14}, {2, 3, 28}, {2, 64, 28}, {3, 96, 42}, {349525, 11184800, 4893350}} {
		got, ok := providerHistoryCollectionRepeatedCursorInitializationUpper(tc.groups, tc.requests)
		want := candidateWorkVector{10: tc.calls * wantCursor[10]}
		if !ok || got != want {
			t.Fatal("new-cursor14G counts differ from roots/copies/window counts", tc, got, want, ok)
		}
	}
	for _, tc := range [][2]uint64{{0, 0}, {1, 0}, {1, 33}, {2, 1}, {349526, 349526}, {^uint64(0), ^uint64(0)}} {
		if got, ok := providerHistoryCollectionRepeatedCursorInitializationUpper(tc[0], tc[1]); ok || got != (candidateWorkVector{}) {
			t.Fatal("invalid constructor topology returned partial work", tc, got, ok)
		}
	}
}

func TestProviderHistoryInitializationRefusesMissingObserverBeforeSelection(t *testing.T) {
	in := historyCollectionInput(t)
	manifest := historyCollectionManifest(t, in)
	selected := in.journal.Deployment
	if c, err := newProviderHistoryCollectionCursorObserved(admissionTestDigest("a"), manifest, selected, nil); err == nil || c != nil {
		t.Fatal("implicit missing original observer")
	}
	m := providerReplayUnitMeter()
	bad := selected
	bad.ParentHash = admissionTestDigest("b")
	if c, err := newProviderHistoryCollectionCursorObserved(admissionTestDigest("a"), manifest, bad, m); err == nil || c != nil || m.failed || m.actual != (candidateWorkVector{10: 265}) {
		t.Fatal("different deployment should refuse after its real comparison", err, m.actual)
	}
	badConfig := in.journal
	badConfig.MaxClaims = 0
	before := m.actual
	f, err := newProviderHistorySemanticFoldObserved(admissionTestDigest("a"), badConfig, selected, in.blocks, in.versions, m)
	want := before
	want[10] += uint64(reflect.TypeFor[providerHistorySemanticFold]().Size()) + 1 + uint64(reflect.TypeFor[ObservationChecks]().Size()) + 1
	if err == nil || f != nil || m.failed || m.actual != want || manifest.input != in {
		t.Fatal("invalid semantic configuration replaced or reset the original observer", err, m.actual, want)
	}
}
