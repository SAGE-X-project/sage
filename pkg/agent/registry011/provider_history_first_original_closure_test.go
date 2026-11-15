package registry011

import (
	"reflect"
	"testing"
)

// This UNIT owns an unused initialization bridge after the ordinary owned first
// exchange. It enters only the original first-reader phase, with the unchanged
// pre-H0 synthetic UNIT reservation. No checkpoint or completed group is
// published. Cleanup consumes that bridge permanently instead of resetting it.
func historyFirstOriginalClosureUnit(t *testing.T) (*providerHistoryFirstPOriginals, providerFrontierRecord) {
	t.Helper()
	manifest := historyCollectionManifest(t, historyCollectionInput(t))
	s := historyFirstFinalizedStore(t, historyFirstFinalizedResponse(t), manifest)
	s.mu.Lock()
	p := s.history
	if p == nil || p.group != nil || !p.startReadyLocked() {
		s.mu.Unlock()
		t.Fatal("unused genuine first-operation bridge required")
	}
	o := &providerHistoryGroupOwner{parent: p, store: s, start: p.start, pin: p.start.main, phase: "initializing"}
	o.owner = o
	m := newProviderWorkMeter(nil, providerOperationReservation{work: p.bounds.future.work, execution: p.bounds.future.work})
	m.historyGroup, m.historyInputGroup, m.maximumClocks, m.authority = o, o, p.bounds.future.clocks, true
	o.meter, p.group = m, o
	p.recovery.originalGroup, p.recovery.originalMeter = o, m
	s.activeMeter.Store(m)
	t.Cleanup(func() {
		o.firstPReading = false
		o.failLocked()
		s.mu.Unlock()
	})
	upper, ok := providerHistoryGroupInitializationForProfileUpper(s.config.bounds)
	if !ok || !m.preflightWork(upper) || !o.identityLocked() {
		t.Fatal("unchanged original initialization reservation required")
	}
	begin, err := s.samplePrimitive()
	if err != nil || !o.clockFits(begin) {
		t.Fatal("original initialization bracket required", err)
	}
	o.firstPReading, o.firstPStart = true, m.actual
	input := &providerHistoryFirstPOriginals{group: o, bounds: p.collectionInput.firstRead}
	if !m.preflightWork(input.bounds.work) || !o.identityLocked() || input.load() != nil || len(input.objects) == 0 {
		t.Fatal("complete original first-reader object closure required")
	}
	raw, err := input.read(s.custody, s.name("frontier", 4), s.config.bounds.maximumMetadataPerCommit)
	current, parseErr := parseProviderFrontierRecordObserved(raw, s.config.identity, m)
	if err != nil || parseErr != nil || current.sequence != 4 || !input.active() {
		t.Fatal("actual fourth frontier required", err, parseErr)
	}
	return input, current
}

func TestProviderHistoryFirstOriginalClosureRejectsMissingAndChangedPrivateReferences(t *testing.T) {
	for _, mode := range []string{"intact", "missing-enrollment", "missing-ticket", "missing-request", "missing-params", "missing-consumption", "duplicate-consumption", "unknown-object", "changed-object", "repeated-closure"} {
		t.Run(mode, func(t *testing.T) {
			input, current := historyFirstOriginalClosureUnit(t)
			o, s, m := input.group, input.group.store, input.group.meter
			config, held, pin, reads, upper := s.config, s.stored, o.pin, s.clock.read, m.upper
			before := providerHistoryProvisionedOriginalBytes(t, config)
			work, rawBytes := m.actual, input.raw
			kind := ""
			switch mode {
			case "missing-enrollment":
				kind = "enrollment"
			case "missing-ticket":
				kind = "ticket"
			case "missing-request":
				kind = "request"
			case "missing-params":
				kind = "params"
			case "missing-consumption", "duplicate-consumption":
				kind = "consumption"
			}
			if kind != "" {
				found := false
				objects := make([]providerIntegrationObject, 0, len(input.objects)+1)
				for _, object := range input.objects {
					if object.kind == kind {
						found = true
						if mode == "duplicate-consumption" {
							objects = append(objects, object, object)
							continue
						}
						continue
					}
					objects = append(objects, object)
				}
				if !found {
					t.Fatal("ordinary original reference required before the private DATA fault", kind)
				}
				input.objects = objects
			}
			switch mode {
			case "unknown-object":
				digest := admissionTestDigest("f")
				for _, object := range input.objects {
					if object.kind == "request" && object.digest == digest {
						t.Fatal("absent object identity required")
					}
				}
				if input.object("request", digest) != nil {
					t.Fatal("unknown private reference borrowed a producer object")
				}
			case "changed-object":
				object := &input.objects[0]
				if object.raw == "" {
					t.Fatal("nonempty original object DATA required")
				}
				object.raw += " "
				if input.object(object.kind, object.digest) != nil {
					t.Fatal("changed private object retained the original digest")
				}
			case "repeated-closure":
				if input.load() == nil {
					t.Fatal("repeated object closure accepted duplicate original references")
				}
			default:
				selected, err := input.selected(current)
				if mode == "intact" {
					if err != nil || selected != historyFirstFinalizedHeader() {
						t.Fatal("intact independent original selection required", err)
					}
				} else if err == nil || selected != (FinalizedHeader{}) {
					t.Fatal("incomplete private reference closure supplied a selected header", mode, err)
				}
			}
			if m.failed || m.closed || m.actual[10] <= work[10] || m.upper != upper || input.raw < rawBytes ||
				s.config != config || s.stored != held || o.pin != pin || s.frontierFiles.pin != pin || s.clock.read != reads ||
				m.actual[0] != work[0] || m.actual[1] != work[1] || s.activeMeter.Load() != m || !input.active() ||
				o.coordinator != nil || o.custodian != nil || o.last != nil || o.parent.recovery.initial != nil ||
				o.parent.recovery.confirmed != nil || o.parent.recovery.pending != nil ||
				!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, config)) {
				t.Fatal("private first-reader DATA changed originals, renewed execution or published a model", mode)
			}
		})
	}
}
