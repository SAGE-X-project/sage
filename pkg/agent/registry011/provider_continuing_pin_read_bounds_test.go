package registry011

import (
	"testing"
	"time"
)

// Inert index DATA and an unattached unit observer only. No filesystem,
// original X/S, private G/T/C/S, credential, custody ACK or dispatch exists.
// A true size check must never be confused with proof of an original read.
func TestProviderContinuingPinOriginalBoundsPrecedeAnyRead(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"exchange", "settlement", "wire-binding", "tls", "response"} {
		for _, mode := range []string{"normal", "nil-replay", "nil-store", "nil-meter", "closed", "failed", "other-observer", "bad-digest", "zero-cap", "huge-cap", "bad-kind", "missing", "wrong-kind", "wrong-digest", "zero-size", "oversized", "global-cap", "work-budget"} {
			t.Run(kind+"/"+mode, func(t *testing.T) {
				meter := newProviderWorkMeter(&providerIntegration{}, providerOperationReservation{execution: candidateWorkVector{2: 1 << 20, 10: 1 << 48}})
				s := &providerCustodyStore{config: providerCustodyConfig{bounds: providerIntegrationBounds{maximumObjectBytes: 1 << 20}}, objects: make(map[string]providerIntegrationObjectRef)}
				s.activeMeter.Store(meter)
				r := &providerCustodyReplay{meter: meter}
				digest, selectedKind, maximum := admissionTestDigest("a"), kind, uint64(8192)
				ref := providerIntegrationObjectRef{kind: kind, digest: digest, bytes: maximum}
				key := providerIntegrationRefKey(ref)
				switch mode {
				case "nil-replay":
					r = nil
				case "nil-store":
					s = nil
				case "nil-meter":
					r.meter = nil
				case "closed":
					meter.closed = true
				case "failed":
					meter.failed = true
				case "other-observer":
					s.activeMeter.Store(nil)
				case "bad-digest":
					digest = "invalid"
				case "zero-cap":
					maximum = 0
				case "huge-cap":
					maximum = (4 << 20) + 1
				case "bad-kind":
					selectedKind = "request"
				case "wrong-kind":
					ref.kind = "params"
				case "wrong-digest":
					ref.digest = admissionTestDigest("b")
				case "zero-size":
					ref.bytes = 0
				case "oversized":
					ref.bytes++
				case "global-cap":
					maximum, ref.bytes = 2<<20, (1<<20)+1
				case "work-budget":
					meter.upper[10] = 0
				}
				if s != nil && mode != "missing" {
					s.objects[key] = ref
				}
				if ok := r.continuingPinBoundOriginal(s, selectedKind, digest, maximum); ok != (mode == "normal") || meter.actual[2] != 0 {
					t.Fatal("size-only DATA read/decode or invalid reference escaped", mode, ok)
				}
			})
		}
	}
}

// All cases stop before a binding original can be read. These finite empty
// inputs have no filesystem or source role; no success is inferred from them.
func TestProviderContinuingPinAncillaryBoundsRefuseMissingDataBeforeRead(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"nil-replay", "nil-store", "nil-request", "nil-exchange", "invalid-capture", "missing-binding"} {
		meter := newProviderWorkMeter(&providerIntegration{}, providerOperationReservation{execution: candidateWorkVector{10: 1 << 30}})
		s := &providerCustodyStore{bindings: map[string]string{}}
		s.activeMeter.Store(meter)
		r := &providerCustodyReplay{meter: meter}
		request := &providerIntegrationRequest{}
		exchange := map[string]any{}
		capture := ProviderCaptureLimits{Timeout: time.Second, HeaderBytes: 1024, BodyBytes: 1024, WireBytes: 4096, JSONDepth: 8}
		switch mode {
		case "nil-replay":
			r = nil
		case "nil-store":
			s = nil
		case "nil-request":
			request = nil
		case "nil-exchange":
			exchange = nil
		case "invalid-capture":
			capture.BodyBytes = 0
		}
		if r.continuingPinAncillaryBounds(s, admissionTestDigest("a"), request, exchange, capture, providerFirstOperationStorage{}) || meter.actual[2] != 0 || meter.actual[9] != 0 {
			t.Fatal("missing finite DATA attempted to read or write an original", mode)
		}
	}
}

// Parameter selection and storage quotes are DATA-only. These views have no
// filesystem or operation owner and must refuse before the first original.
func TestProviderContinuingPinClassAndStorageBoundsPrecedeOriginalReads(t *testing.T) {
	t.Parallel()
	config, _ := providerGuardianConfigWithSelectionFixture(t, 64<<20, 6, &providerScanFixtureSelection{pin: true, pinResidentGiB: true, pinLargeObject: true})
	for _, mode := range []string{"class", "storage", "missing-exchange"} {
		meter := newProviderWorkMeter(&providerIntegration{}, providerOperationReservation{execution: candidateWorkVector{2: 64 << 20, 10: 1 << 60}})
		s := &providerCustodyStore{config: config, objects: map[string]providerIntegrationObjectRef{}}
		s.activeMeter.Store(meter)
		r := &providerCustodyReplay{meter: meter, ledger: &CandidateAdmissionLedger{}}
		digests := [4]string{admissionTestDigest("a"), admissionTestDigest("b"), admissionTestDigest("c"), admissionTestDigest("d")}
		switch mode {
		case "class":
			s.config.bounds.methods = &providerMethodBounds{}
		case "storage":
			s.config.bounds.maximumObjectBytes = 1
		case "missing-exchange":
			requests, err := providerPinRequests(admissionTestDigest("a"))
			if err != nil {
				t.Fatal(err)
			}
			capture, err := config.bounds.methods.capture(requests[0], config.capture)
			if err != nil {
				t.Fatal(err)
			}
			limits, ok := providerOperationStorageUpper(config.bounds, capture, uint64(len(requests[0].wire)))
			if !ok {
				t.Fatal("normal expected class quote")
			}
			if value, err := r.continuingPinSettlement(s, digests[0], requests[0], capture, limits); err == nil || value != nil || meter.actual[2] != 0 || meter.actual[9] != 0 {
				t.Fatal("absent exchange bound yielded settlement DATA")
			}
			continue
		}
		values, err := r.continuingPinResults(s, digests, admissionTestDigest("a"))
		if err == nil || s.valid() || meter.actual[9] != 0 {
			t.Fatal("invalid class or storage DATA attempted original processing", mode)
		}
		for _, value := range values {
			if value != nil {
				t.Fatal("invalid quote supplied a response", mode)
			}
		}
	}
}
