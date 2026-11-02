package registry011

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

func TestProviderHistoryHealthyFirstOriginalIgnoresProducerMapsAndReplaysBothModels(t *testing.T) {
	m := historyCollectionManifest(t, historyCollectionInput(t))
	s := historyFirstFinalizedStore(t, historyFirstFinalizedResponse(t), m)
	clocks, held, pin := s.clock.read, s.stored, s.frontierFiles.pin
	before := providerHistoryProvisionedOriginalBytes(t, s.config)
	// Inert cache faults only. Every actual retained file and reference stays
	// intact; no hostile peer, filesystem bypass or network attack is built.
	s.history.firstIntegration.firstResult = `"obsolete returned DATA"`
	originalRefs := make(map[string]providerIntegrationObjectRef, len(s.objects))
	for key, ref := range s.objects {
		originalRefs[key] = ref
		ref.digest = admissionTestDigest("e")
		s.objects[key] = ref
	}
	for exchange := range s.bindings {
		s.bindings[exchange] = admissionTestDigest("f")
	}
	o, err := newProviderHistoryGroupOwnerCandidate(s)
	if err != nil || o == nil || o.coordinator == o.custodian || o.coordinator.head != o.custodian.head ||
		o.firstPReading || o.meter.actual[2] == 0 || s.clock.read != clocks+2 || s.stored != held || o.pin != pin ||
		o.last != nil || o.admission != nil || s.activeMeter.Load() != nil || !reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
		t.Fatal("healthy first P borrowed producer DATA or changed its original hold", err)
	}
	healthyInitial := o.coordinator.initial
	// This helper selects its expected refs from this inert fixture map;
	// restore those fixture expectations after the healthy no-cache proof.
	s.objects = originalRefs
	// Independent terminal readers must derive the very same original model
	// through their own full legacy/HTTP/TLS replay and selected header.
	r, refs := historyRecoveryGroupStart(t, o)
	for pass := range 2 {
		historyRecoveryModelPrefix(t, r, pass, refs)
		initial, err := r.replayInitialModelCandidate(context.Background(), pass)
		if err != nil {
			t.Fatal("complete independent original model", pass, err)
		}
		if selected, err := initial.selectFirstFinalizedCandidate(context.Background()); err != nil || selected != historyFirstFinalizedHeader() {
			t.Fatal("independent header selection", pass, err)
		}
		model, err := r.replayGroupModelCandidate(context.Background(), pass)
		if err != nil || model == nil || model.ledger.head != r.initial.model || model.ledger.initial != healthyInitial {
			t.Fatal("healthy and terminal readers disagree about the original selected P", pass, err)
		}
	}
	if s.clock.read != clocks+2 || s.stored != held || s.frontierFiles.pin != pin || !reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
		t.Fatal("independent model comparison renewed time, wrote originals or refunded the hold")
	}
}

func TestProviderHistoryHealthyFirstOriginalRejectsUnusableHeaderAfterRealOwnedTransport(t *testing.T) {
	for _, mode := range []string{"legacy-result", "above-maximum", "old-time"} {
		t.Run(mode, func(t *testing.T) {
			in, response := historyCollectionInput(t), historyFirstFinalizedResponse(t)
			switch mode {
			case "legacy-result":
				response = providerFirstOperationResponse()
			case "above-maximum":
				in.maximumSelected--
				in.blocks--
				in.presentStops = 3
			case "old-time":
				in.journal.Deployment.Timestamp = historyFirstFinalizedHeader().Timestamp + 1
			}
			s := historyFirstFinalizedStore(t, response, historyCollectionManifest(t, in))
			held, pin, clocks := s.stored, s.frontierFiles.pin, s.clock.read
			before := providerHistoryProvisionedOriginalBytes(t, s.config)
			if o, err := newProviderHistoryGroupOwnerCandidate(s); err == nil || o != nil || !s.failed || !s.history.failed ||
				s.history.group == nil || !s.history.group.failed || !s.history.group.meter.closed || s.history.group.firstPReading ||
				s.history.group.last != nil || s.history.recovery.initial != nil || s.activeMeter.Load() != nil ||
				s.clock.read != clocks+1 || s.stored != held || s.frontierFiles.pin != pin || !reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
				t.Fatal("an unusable first P became collection permission", mode, err)
			}
		})
	}
}

func TestProviderHistoryHealthyFirstOriginalSubLimitsNeverRenewFutureAllowance(t *testing.T) {
	for _, mode := range []string{"objects", "raw", "decoded", "work", "future-cap", "overflow"} {
		t.Run(mode, func(t *testing.T) {
			in := historyCollectionInput(t)
			switch mode {
			case "objects":
				in.firstRead.objects = 1
			case "raw":
				in.firstRead.raw = 1
			case "decoded":
				in.firstRead.work[2] = 1
			case "work":
				in.firstRead.work[10] = 1
			case "future-cap":
				in.firstRead.work[10] = 1 << 60
			case "overflow":
				in.firstRead.work[10] = ^uint64(0)
			}
			s := historyFirstFinalizedStore(t, historyFirstFinalizedResponse(t), historyCollectionManifest(t, in))
			clocks, held, pin := s.clock.read, s.stored, s.frontierFiles.pin
			before := providerHistoryProvisionedOriginalBytes(t, s.config)
			wantSamples := uint64(1)
			if mode == "future-cap" || mode == "overflow" {
				wantSamples = 0
			}
			if o, err := newProviderHistoryGroupOwnerCandidate(s); err == nil || o != nil || !s.failed || !s.history.failed ||
				s.history.group == nil || !s.history.group.meter.closed || s.history.group.last != nil || s.history.group.firstPReading ||
				s.history.recovery.initial != nil || s.activeMeter.Load() != nil || s.clock.read != clocks+wantSamples || s.stored != held ||
				s.frontierFiles.pin != pin || !reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
				t.Fatal("first-original failure renewed the original row or acknowledged work", mode, err)
			}
		})
	}
}

func TestProviderHistoryFirstOriginalReadBoundsAreExplicitAndReadOnly(t *testing.T) {
	in := historyCollectionInput(t)
	for _, mode := range []string{"objects-zero", "objects-large", "raw-zero", "raw-large", "decoded-zero", "work-zero", "request", "wire", "blocks", "transactions", "frames", "logs", "proofs", "segments", "disk"} {
		t.Run(mode, func(t *testing.T) {
			changed := in
			switch mode {
			case "objects-zero":
				changed.firstRead.objects = 0
			case "objects-large":
				changed.firstRead.objects = 1<<20 + 1
			case "raw-zero":
				changed.firstRead.raw = 0
			case "raw-large":
				changed.firstRead.raw = 64<<20 + 1
			case "decoded-zero":
				changed.firstRead.work[2] = 0
			case "work-zero":
				changed.firstRead.work[10] = 0
			default:
				index := map[string]int{"request": 0, "wire": 1, "blocks": 3, "transactions": 4, "frames": 5, "logs": 6, "proofs": 7, "segments": 8, "disk": 9}[mode]
				changed.firstRead.work[index] = 1
			}
			if m, err := newProviderHistoryCollectionManifestCandidate(changed); err == nil || m != nil {
				t.Fatal("missing or non-read-only row supplied a default", mode)
			}
		})
	}
	// Large work counts retain decimal-string identity in the pre-H0 profile;
	// JSON's floating-point number domain cannot round a different row into it.
	in.firstRead.work[10] = 1 << 60
	a := historyCollectionManifest(t, in)
	in.firstRead.work[10]++
	b := historyCollectionManifest(t, in)
	if a.digest == b.digest || a.raw == b.raw {
		t.Fatal("different original work rows have the same encoded identity")
	}
	for _, tc := range []struct {
		manifest *providerHistoryCollectionManifest
		want     string
	}{{a, "1152921504606846976"}, {b, "1152921504606846977"}} {
		var fields map[string]json.RawMessage
		var row struct {
			Work map[string]string `json:"work"`
		}
		if json.Unmarshal([]byte(tc.manifest.raw), &fields) != nil || len(fields["first_original_read"]) == 0 ||
			json.Unmarshal(fields["first_original_read"], &row) != nil || row.Work["work_units"] != tc.want {
			t.Fatal("original work identity is not its exact decimal string", tc.want)
		}
	}
}

func TestProviderHistoryRecoveryCollectionModelRequiresItsOwnOriginalSelection(t *testing.T) {
	m := historyCollectionManifest(t, historyCollectionInput(t))
	_, r, refs := historyFirstFinalizedFixtureWithManifest(t, historyFirstFinalizedResponse(t), m)
	historyRecoveryModelPrefix(t, r, 0, refs)
	if _, err := r.replayInitialModelCandidate(context.Background(), 0); err != nil {
		t.Fatal("complete transport model", err)
	}
	files, clocks, held := r.inputFiles, r.store.clock.read, r.store.stored
	if model, err := r.replayGroupModelCandidate(context.Background(), 0); err == nil || model != nil || !r.failed || !r.closed || r.active ||
		r.inputFiles != files || r.store.clock.read != clocks || r.store.stored != held || r.initialModels[0].finalizedSelected {
		t.Fatal("terminal reader borrowed the healthy header or admitted an unselected model", err)
	}
}
