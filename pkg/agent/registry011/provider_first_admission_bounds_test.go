package registry011

import (
	"encoding/base64"
	"strings"
	"testing"
)

func providerFirstAdmissionBoundsFixture() providerFirstAdmissionBounds {
	return providerFirstAdmissionBounds{work: candidateWorkVector{0, 0, 1 << 20, 0, 0, 0, 0, 0, 0, 1 << 20, 1 << 50}, retained: 2 << 20, live: 2 << 20, clocks: 64, durationMS: 300000}
}

func TestProviderFirstAdmissionHasDistinctPreprovisionedProfile(t *testing.T) {
	first := providerFirstAdmissionBoundsFixture()
	config, clock := providerScanProvisionedFixtureWithFirst(t, &first)
	old := config.bounds
	old.firstAdmission = nil
	oldRaw, oldDomain := providerIntegrationProfileBytes(old, config.capture)
	raw, domain := providerIntegrationProfileBytes(config.bounds, config.capture)
	m := contentJSON(raw).(map[string]any)
	store := m["store"].(map[string]any)
	if domain != "sage-provider-store-profile-v4" || oldDomain != "sage-provider-store-profile-v3" || m["format"] != "sage-provider-owned-profile-v4" || store["live_accounting"] != providerFirstAdmissionLiveAccounting || store["first_admission"] == nil || string(raw) == string(oldRaw) || providerIntegrationProfile(old, config.capture) == config.profile {
		t.Fatal("old H0 accounting was silently adopted")
	}
	s, err := openProviderFrontierCandidate(config, clock)
	if err != nil {
		t.Fatal("ordinary fresh V4 DATA open", err)
	}
	t.Cleanup(func() { _ = s.close() })
	if s.firstLimits == nil || !providerFirstAdmissionCapacity(s, providerScanOpenUpper(config)) || s.last != nil || s.sequence != 0 {
		t.Fatal("capacity data became an enrollment or ACK")
	}
	full, err := providerScanBodyUpper(config)
	oldBody, oldErr := providerScanBodyUpper(providerCustodyConfig{bounds: old})
	// V4 additionally funds the two closed receipt decoders, independently of
	// the four original readbacks; old V3 keeps its original output bound.
	oldBody.work[2] += 2 * config.bounds.maximumMetadataPerCommit * config.bounds.scan.maximumChunks
	want, ok := providerScanUpperSum(oldBody, providerScanUpper{first.work, first.retained, first.clocks})
	if err != nil || oldErr != nil || !ok || full != want {
		t.Fatal("first-E1 costs were not held in the complete B before I", err, oldErr)
	}
	if _, err := s.enrollBeforeScan(); err != nil {
		t.Fatal("original O-funded enrollment before inspection", err)
	}
	if err := s.scanControl.claim(providerScanAttemptFixture(t, s, s.scanControl.pin)); err != nil || s.scanControl.held.retained < first.retained || s.last != nil {
		t.Fatal("ordinary fresh claim omitted its finite first-E1 hold", err)
	}
	wrong := config
	wrong.profile = providerIntegrationProfile(old, config.capture)
	if _, err := newProviderScanCodec(wrong, nil); err == nil {
		t.Fatal("old V3 profile accepted as a V4 bridge profile")
	}
}

func TestProviderFirstAdmissionFiniteBoundsRejectIncompleteEnvelopes(t *testing.T) {
	base, _ := providerScanProvisionedFixture(t)
	base.bounds.setupLive = base.bounds.setupRetained
	for _, mode := range []string{"ordinary", "no-scan", "open-live", "zero", "live", "retained", "clocks", "duration", "decoded", "written", "work", "request", "coordinate"} {
		t.Run(mode, func(t *testing.T) {
			b, first := base.bounds, providerFirstAdmissionBoundsFixture()
			switch mode {
			case "no-scan":
				b.scan = nil
			case "open-live":
				b.setupLive++
			case "zero":
				first.retained, first.live = 0, 0
			case "live":
				first.live++
			case "retained":
				first.retained, first.live = b.maximumStoredBytes+1, b.maximumStoredBytes+1
			case "clocks":
				first.clocks = 0
			case "duration":
				first.durationMS = 0
			case "decoded":
				first.work[2] = 0
			case "written":
				first.work[9] = first.retained + 1
			case "work":
				first.work[10] = 0
			case "request":
				first.work[0] = 1
			case "coordinate":
				first.work[10] = b.scan.maximumPreIWork[10] + 1
			}
			b.firstAdmission = &first
			if b.valid() != (mode == "ordinary") {
				t.Fatal("incomplete or unfunded candidate envelope accepted", mode)
			}
		})
	}
}

func TestProviderFirstAdmissionCachesObservedLimitsBeforeAnyClaim(t *testing.T) {
	first := providerFirstAdmissionBoundsFixture()
	config, clock := providerScanProvisionedFixtureWithFirst(t, &first)
	s, err := openProviderFrontierCandidate(config, clock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.close() })
	if _, err := s.enrollBeforeScan(); err != nil {
		t.Fatal("funded candidate enrollment before claim capacity checks", err)
	}
	want := *s.firstLimits
	for _, mode := range []string{"missing", "live", "retained", "work"} {
		t.Run(mode, func(t *testing.T) {
			limits := want
			s.firstLimits = &limits
			switch mode {
			case "missing":
				s.firstLimits = nil
			case "live":
				limits.live = 0
			case "retained":
				limits.retained = 0
			case "work":
				limits.work[10] = 0
			}
			held, stored := s.scanControl.held, s.stored
			if err := s.scanControl.claim(providerScanAttemptFixture(t, s, s.scanControl.pin)); err == nil || s.stored != stored || s.scanControl.held != held || s.scanControl.prepared != nil || s.failed {
				t.Fatal("capacity refusal spent or replenished a claim", mode)
			}
		})
	}
}

func TestProviderFirstAdmissionLimitDecoderRefusesMalformedOrUnfundedData(t *testing.T) {
	first := providerFirstAdmissionBoundsFixture()
	config, _ := providerScanProvisionedFixtureWithFirst(t, &first)
	for _, raw := range []string{`[]`, `{"limits":[]}`, `{"limits":{"bootstrap":{},"live_disk_bytes":"1","retained_bytes":"1"}}`} {
		// Inert parser DATA only: this object never opens roots or creates an I.
		c := config
		c.policy = &ProviderPolicyCandidate{raw: raw}
		s := &providerCustodyStore{config: c}
		x := &providerScanCodec{config: c}
		if s.firstAdmissionLimits(x) || s.firstLimits != nil {
			t.Fatal("invalid limit bytes populated a cached capacity")
		}
	}
	s := &providerCustodyStore{config: config}
	control := &providerScanControl{store: s}
	control.owner = control
	m := &providerWorkMeter{scan: control}
	m.owner = m
	if s.firstAdmissionLimits(&providerScanCodec{config: config, meter: m}) || !m.failed || s.firstLimits != nil {
		t.Fatal("unfunded decoder populated a capacity")
	}
}

func TestProviderFirstAdmissionStorageQuoteIncludesBindingAndEveryTemporaryCopy(t *testing.T) {
	first := providerFirstAdmissionBoundsFixture()
	config, _ := providerScanProvisionedFixtureWithFirst(t, &first)
	b := config.bounds // Candidate arithmetic only; no store/owner is opened.
	raw, physical, ok := providerFirstAdmissionStorageUpper(b)
	// Conservative closed file quote: binding and intent/pending/commit;
	// I/ticket/E1/inventory/frontier; B0/four nodes/three mutations/two heads;
	// request and params. All21 allocations include HEAD.pending.
	want := 4*b.maximumMetadataPerCommit + 5*65536 + 10*8192 + 16384 + 8192
	if !ok || raw != want || physical != want+21*b.fileOverheadBytes {
		t.Fatal("first storage quote omitted an original or temporary copy")
	}
	// This quote intentionally grows all four metadata allowances. The actual
	// binding still has min(m,8192) as its codec cap. No encoded binding, store,
	// previously provisioned profile, hold or finishing owner is enlarged.
	b.maximumMetadataPerCommit *= 2
	largeRaw, largePhysical, ok := providerFirstAdmissionStorageUpper(b)
	if !ok || largeRaw != raw+4*config.bounds.maximumMetadataPerCommit || largePhysical != physical+4*config.bounds.maximumMetadataPerCommit {
		t.Fatal("conservative metadata quote did not scale arithmetically")
	}
	for _, mode := range []string{"legacy", "entries", "objects", "metadata", "capacity"} {
		x := config.bounds
		switch mode {
		case "legacy":
			x.firstAdmission = nil
		case "entries":
			x.maximumEntries = 13
		case "objects":
			x.maximumObjects = 20
		case "metadata":
			x.maximumMetadataPerCommit = 0
		case "capacity":
			x.maximumStoredBytes = physical - 1
		}
		if _, _, ok := providerFirstAdmissionStorageUpper(x); ok {
			t.Fatal("unfunded candidate first storage shape accepted", mode)
		}
	}
}

func TestProviderFirstOperationStorageSeparatesFutureConsumptionAndSettlement(t *testing.T) {
	first := providerFirstAdmissionBoundsFixture()
	config, _ := providerScanProvisionedFixtureWithFirst(t, &first)
	request, err := newProviderIntegrationRequest("eth_getBlockByNumber", []any{"finalized", false})
	if err != nil {
		t.Fatal(err)
	}
	b := config.bounds
	quote, ok := providerFirstOperationStorageUpper(b, config.capture, uint64(len(request.wire)))
	if !ok || quote.transport == 0 || quote.exchange <= quote.transport {
		t.Fatal("finite future-only storage quote")
	}
	// Independent C/S file inventory: C has consumption/B/enrollment/node/
	// three mutations; S adds X/response/TLS/wire binding. Each has five
	// main files and two possible head copies, hence14+18 allocations.
	c := 8*uint64(8192) + 3*65536 + 3*b.maximumMetadataPerCommit
	s := 9*uint64(8192) + 3*65536 + 3*b.maximumMetadataPerCommit + quote.exchange + uint64(b.methods.header.wire) + quote.transport
	if quote.consumption != c || quote.settlement != s || quote.raw != c+s || quote.physical != c+s+32*b.fileOverheadBytes {
		t.Fatal("future C/S omitted or doubled files/transport copies")
	}
	// Quoting future storage neither includes the first21-file E1 quote nor
	// obtains funds from it. Both remain independent arithmetic components.
	_, firstPhysical, valid := providerFirstAdmissionStorageUpper(b)
	if !valid || quote.physical == firstPhysical || quote.physical == 2*(c+s+32*b.fileOverheadBytes) {
		t.Fatal("future-only quote reused the old double-capacity estimator")
	}
	b.maximumMetadataPerCommit *= 2
	larger, valid := providerFirstOperationStorageUpper(b, config.capture, uint64(len(request.wire)))
	if !valid || larger.physical != quote.physical+6*config.bounds.maximumMetadataPerCommit || larger.transport != quote.transport || larger.exchange != quote.exchange {
		t.Fatal("C/S metadata quote silently included earlier first-E1 files")
	}
}

func TestProviderFirstOperationStorageRefusesUnboundedOrUnfundedCandidateData(t *testing.T) {
	first := providerFirstAdmissionBoundsFixture()
	config, _ := providerScanProvisionedFixtureWithFirst(t, &first)
	for _, mode := range []string{"legacy", "global", "request-zero", "request-limit", "request-wire", "object", "store"} {
		t.Run(mode, func(t *testing.T) {
			b, capture, request := config.bounds, config.capture, uint64(256)
			switch mode {
			case "legacy":
				b.firstAdmission = nil
			case "global":
				capture.Timeout = 0
			case "request-zero":
				request = 0
			case "request-limit":
				request = 16<<10 + 1
			case "request-wire":
				// Pure scalar DATA; the selected header wire class cannot hold it.
				copy := *b.methods
				copy.header.body, copy.header.wire = 4<<10, 12<<10
				b.methods, request = &copy, 12<<10
			case "object":
				b.maximumObjectBytes = 8192
			case "store":
				b.maximumStoredBytes = 1
			}
			if quote, ok := providerFirstOperationStorageUpper(b, capture, request); ok || quote != (providerFirstOperationStorage{}) {
				t.Fatal("invalid future C/S quote", mode)
			}
		})
	}
}

// Serialization-bound DATA only. No sample is completed, retained, returned
// through an exchange factory, or used to qualify a TLS/Registry/Agent peer.
func TestProviderFirstOperationStorageCoversIndependentJSONAndBase64Bytes(t *testing.T) {
	first := providerFirstAdmissionBoundsFixture()
	config, _ := providerScanProvisionedFixtureWithFirst(t, &first)
	request, err := newProviderIntegrationRequest("eth_getBlockByNumber", []any{"finalized", false})
	if err != nil {
		t.Fatal(err)
	}
	quote, ok := providerFirstOperationStorageUpper(config.bounds, config.capture, uint64(len(request.wire)))
	if !ok {
		t.Fatal("finite serialization upper")
	}
	headers := make([]map[string]string, 128)
	for n := range headers {
		headers[n] = map[string]string{"name": "X-Example", "value": `an "ordinary" value`}
	}
	chain := make([]string, config.bounds.tls.certificates)
	for n := range chain {
		chain[n] = admissionTestDigest("a")
	}
	transport := map[string]any{"endpoint": SepoliaRPCURL, "method": "POST", "http_version": "HTTP/1.1", "status": "200", "response_headers": headers, "tls_version": "65535", "cipher_suite": "65535", "server_name": "rpc.sepolia.ethpandaops.io", "verified_chain": chain, "roots_artifact": admissionTestDigest("a"), "redirects": "0", "proxy": "none"}
	clock := map[string]string{"id": strings.Repeat(`"`, 256), "epoch": admissionTestDigest("a"), "wall_ms": "9223372036854775807", "monotonic_ms": "9223372036854775807", "uncertainty_ms": "9223372036854775807"}
	metadata := map[string]any{"format": "sage-provider-exchange-v2", "policy": admissionTestDigest("a"), "namespace": admissionTestDigest("a"), "instance_key": admissionTestDigest("a"), "source_id": admissionTestDigest("a"), "generation": "18446744073709551615", "ticket_digest": admissionTestDigest("a"), "provider_digest": admissionTestDigest("a"), "transport": transport, "request": base64.RawURLEncoding.EncodeToString([]byte(request.body)), "response": base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("ordinary bytes ", config.bounds.methods.header.body/15))), "clock_start": clock, "clock_end": clock}
	rawTransport, rawExchange := providerIntegrationJSON(transport), providerIntegrationJSON(metadata)
	if len(rawTransport) == 0 || len(rawExchange) == 0 || uint64(len(rawTransport)) > quote.transport || uint64(len(rawExchange)) > quote.exchange {
		t.Fatal("JSON structure, escaping, identity or base64 bytes exceeded quote")
	}
}

func TestProviderFirstOperationStorageChecksTheSeparateResponseObjectLimit(t *testing.T) {
	first := providerFirstAdmissionBoundsFixture()
	config, _ := providerScanProvisionedFixtureWithFirst(t, &first)
	b := config.bounds
	methods := *b.methods
	methods.header.body, methods.header.wire = 1024, 256<<10
	b.methods, b.maximumObjectBytes = &methods, 128<<10
	// Finite scalar DATA: small body/X/TLS do not bound a separate raw wire
	// object. No network, completion, retained object or admission is made.
	if quote, ok := providerFirstOperationStorageUpper(b, config.capture, 256); ok || quote != (providerFirstOperationStorage{}) {
		t.Fatal("aggregate capacity hid an oversized single response object")
	}
	b.maximumObjectBytes = uint64(methods.header.wire) - 256
	if _, ok := providerFirstOperationStorageUpper(b, config.capture, 256); !ok {
		t.Fatal("exact response-object upper boundary refused")
	}
	b.maximumObjectBytes--
	if _, ok := providerFirstOperationStorageUpper(b, config.capture, 256); ok {
		t.Fatal("response-object upper exceeding the boundary accepted")
	}
}
