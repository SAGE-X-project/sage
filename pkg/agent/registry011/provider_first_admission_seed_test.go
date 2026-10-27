package registry011

import (
	"strconv"
	"strings"
	"testing"
)

// Expected closed DATA is placed only in this isolated fixture's owned root.
// This is not the future prepaid E1 writer, an E1 commit, CAS, ACK or permit.
// The production replay reads it independently under the unchanged F meter.
func providerFirstAdmissionSeedFixture(t *testing.T) (*providerCustodyStore, *providerFirstAdmissionOwner, *providerScanReplayReader, providerScanAdmissionBinding, *CandidateAdmissionIntent) {
	t.Helper()
	s, f, x, binding := providerScanBindingFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := f.startLocked(); err != nil {
		t.Fatal(err)
	}
	r, err := f.replayEnrollmentLocked()
	if err != nil {
		t.Fatal(err)
	}
	c := &providerIntegration{store: s, policy: s.config.policy, context: r.replay.body.context, anchor: r.replay.body.anchor}
	c.intent, err = providerFirstAdmissionIntentDataObserved(c.policy, c.context, c.anchor, f.mainPin.commit, providerClockDiagnostic(f.origin), f.meter)
	if err != nil {
		t.Fatal("expected immutable first-I DATA", err)
	}
	binding.admission = c.intent.digest
	raw, err := x.admissionBinding(binding)
	if err != nil {
		t.Fatal(err)
	}
	providerFirstAdmissionSeedInput(t, s, r.view, "scan-admission-binding", raw)
	r.replay.body.intent = c.intent // The apply test below clears and reparses it.
	r.view.requiredReferences = make(map[string]bool)
	return s, f, r, binding, c.intent
}

func TestProviderFirstAdmissionIntentConstructionPreservesBytesAndObservedCopies(t *testing.T) {
	s, f := providerFirstAdmissionOwnerFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := f.startLocked(); err != nil {
		t.Fatal(err)
	}
	r, err := f.replayEnrollmentLocked()
	if err != nil {
		t.Fatal(err)
	}
	before, stored, held := f.meter.actual, s.stored, s.scanControl.held
	intent, err := providerFirstAdmissionIntentDataObserved(s.config.policy, r.replay.body.context, r.replay.body.anchor, f.mainPin.commit, providerClockDiagnostic(f.origin), f.meter)
	if err != nil || intent == nil {
		t.Fatal("observed immutable I DATA", err)
	}
	expected := &providerIntegration{store: s, policy: s.config.policy, context: r.replay.body.context, anchor: r.replay.body.anchor}
	if err := expected.makeIntent(f.mainPin.commit); err != nil || expected.intent.raw != intent.raw || expected.intent.digest != intent.digest || intent.deadline != f.attempt.deadline || intent.clock != providerClockDiagnostic(s.setupOrigin) {
		t.Fatal("I bytes, original epoch or deadline changed", err)
	}
	plain, err := providerFirstAdmissionIntentDataObserved(s.config.policy, r.replay.body.context, r.replay.body.anchor, f.mainPin.commit, providerClockDiagnostic(f.origin), nil)
	if err != nil || plain == nil || plain.raw != intent.raw || plain.digest != intent.digest {
		t.Fatal("pure expected DATA changed the immutable byte contract", err)
	}
	if f.meter.actual[2]-before[2] != uint64(2*len(s.config.policy.raw)+len(intent.raw)) || f.meter.actual[10] <= before[10] || s.stored != stored || s.scanControl.held != held || s.last != nil || s.sequence != 1 || f.body.active || r.replay.body.intent != nil {
		t.Fatal("I construction omitted current decoding or created retained state")
	}
	for k, upper := range providerFirstAdmissionIntentUpper() {
		if f.meter.actual[k]-before[k] > upper {
			t.Fatal("I primitives exceeded their separate logical component", k)
		}
	}
}

func TestProviderFirstAdmissionIntentConstructionRefusesIncompleteCandidateData(t *testing.T) {
	s, f, reader, _, _ := providerFirstAdmissionSeedFixture(t)
	for _, mode := range []string{"policy", "context", "unbound", "anchor", "authority", "clock", "clock-size", "identity", "status", "anchor-digest", "policy-size", "policy-bytes", "policy-limits", "policy-clock", "duration", "underflow", "budget"} {
		t.Run(mode, func(t *testing.T) {
			policy, context, anchor, authority, clock := s.config.policy, reader.replay.body.context, reader.replay.body.anchor, f.mainPin.commit, providerClockDiagnostic(f.origin)
			var meter *providerWorkMeter // Pure expected parser DATA except final budget case.
			switch mode {
			case "policy":
				policy = nil
			case "context":
				context = nil
			case "unbound":
				copy := *context
				copy.bound = false
				context = &copy
			case "anchor":
				anchor = nil
			case "authority":
				authority = "unselected finite DATA"
			case "clock":
				clock.Epoch = ""
			case "clock-size":
				clock.ClockID = strings.Repeat("a", 257)
			case "identity":
				copy := *policy
				copy.policy = admissionTestDigest("b")
				policy = &copy
			case "status":
				copy := *anchor
				copy.status = "not-enrolling"
				anchor = &copy
			case "anchor-digest":
				copy := *anchor
				copy.digest = "unselected finite DATA"
				anchor = &copy
			case "policy-size":
				copy := *policy
				copy.raw = strings.Repeat(" ", 64<<10+1)
				policy = &copy
			case "policy-bytes", "policy-limits", "policy-clock", "duration":
				copy := *policy
				m := contentJSON([]byte(policy.raw)).(map[string]any)
				switch mode {
				case "policy-bytes":
					copy.raw = `[]`
				case "policy-limits":
					delete(m, "limits")
				case "policy-clock":
					delete(m, "clock")
				case "duration":
					m["limits"].(map[string]any)["bootstrap"].(map[string]any)["deadline_ms"] = "18446744073709551615"
				}
				if mode != "policy-bytes" {
					copy.raw = string(providerIntegrationJSON(m))
				}
				policy = &copy
			case "underflow":
				clock.UncertaintyMS = clock.WallMS + 1
			case "budget":
				meter = f.meter
				meter.upper[10] = meter.actual[10]
			}
			stored, held := s.stored, s.scanControl.held
			if got, err := providerFirstAdmissionIntentDataObserved(policy, context, anchor, authority, clock, meter); err == nil || got != nil || s.stored != stored || s.scanControl.held != held || s.last != nil {
				t.Fatal("incomplete I candidate became a result or retaining authority", mode)
			}
		})
	}
}

// This is inert reservation DATA, not the prospective operation estimator or
// an actual hold. Only first-ticket formatting is measured in these units.
func providerFirstTicketReservationData() providerOperationReservation {
	w := candidateWorkVector{1, 1024, 1, 0, 0, 0, 0, 0, 1, 1, 1}
	return providerOperationReservation{work: w, execution: w, live: 1000, retained: 1000}
}

func TestProviderFirstFinalizedCheckAccountsEachComparisonAndConstruction(t *testing.T) {
	s, f, _, _, _ := providerFirstAdmissionSeedFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := newProviderIntegrationRequest("eth_getBlockByNumber", []any{"finalized", false})
	if err != nil {
		t.Fatal(err)
	}
	before, clocks := f.meter.actual, f.meter.clocks
	if !r.firstFinalizedObserved(f.meter) {
		t.Fatal("closed finalized DATA refused")
	}
	// Native text comparisons pay 2*len+1 without converting to raw bytes.
	// Head encoding/string retention and wire joining are separate events.
	var want candidateWorkVector
	for _, raw := range []string{r.method, r.params, r.body, r.head, r.wire} {
		want[10] += 2*uint64(len(raw)) + 1
	}
	head, wire := uint64(len(r.head)), uint64(len(r.wire))
	want[10] += head*head + 1 + head + 1 + wire + 1
	for k, n := range want {
		if f.meter.actual[k]-before[k] != n {
			t.Fatal("fixed request check omitted or invented a primitive invocation", k)
		}
	}
	const n uint64 = 64 << 10
	if 7*(n*n+1)+17*(2*n+65) > providerFirstAdmissionTicketUpper()[10] || f.meter.clocks != clocks || s.last != nil {
		t.Fatal("unchanged ticket upper omitted actual quadratic/linear costs or sampled a clock")
	}
}

func TestProviderFirstAdmissionTicketConstructionPreservesOriginalBytesAndClock(t *testing.T) {
	s, f, _, _, intent := providerFirstAdmissionSeedFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	request, err := newProviderIntegrationRequest("eth_getBlockByNumber", []any{"finalized", false})
	if err != nil {
		t.Fatal(err)
	}
	clock, err := f.sampleLocked()
	if err != nil {
		t.Fatal(err)
	}
	reservation := providerFirstTicketReservationData()
	before, stored, held, clocks := f.meter.actual, s.stored, s.scanControl.held, f.meter.clocks
	ticket, err := providerFirstAdmissionTicketDataObserved(s.config.policy, intent, request, reservation, providerClockDiagnostic(clock), f.meter)
	if err != nil || ticket == nil {
		t.Fatal("observed first-ticket DATA", err)
	}
	p := contentJSON([]byte(s.config.policy.raw)).(map[string]any)
	duration, ok := providerCandidateUint(p["limits"].(map[string]any)["operation"].(map[string]any)["deadline_ms"], true)
	if !ok {
		t.Fatal("expected operation duration")
	}
	deadline := min(uint64(clock.WallMS-clock.UncertaintyMS)+duration, intent.deadline)
	id := intent.identity
	expectedRaw := providerPolicyTestBytes(t, map[string]any{"format": "sage-provider-budget-ticket-v2", "admission_digest": intent.digest, "namespace": id.Namespace, "instance_key": id.Instance, "source_id": id.SourceID, "generation": "0", "fence": strconv.FormatUint(intent.fence, 10), "sequence": "0", "previous_ticket": nil, "segment": candidateAdmissionSegment(intent.digest, 0), "kind": "rpc-capture", "input_digest": request.inputDigest, "method": request.method, "params_digest": request.paramsDigest, "reservation": candidateLedgerWorkObject(reservation.work), "live_disk_reserved": "1000", "retained_bytes_reserved": "1000", "deadline_wall_ms": strconv.FormatUint(deadline, 10)})
	if ticket.raw != string(expectedRaw) || ticket.digest != enrollmentTestHash("sage-provider-budget-ticket-v2", expectedRaw) || ticket.deadline != deadline || ticket.intent != intent || ticket.sequence != 0 || ticket.segment != 0 {
		t.Fatal("ticket changed its original clock, closed bytes or first segment")
	}
	plain, err := providerFirstAdmissionTicketDataObserved(s.config.policy, intent, request, reservation, providerClockDiagnostic(clock), nil)
	if err != nil || plain == nil || plain.raw != ticket.raw || plain.digest != ticket.digest {
		t.Fatal("pure expected ticket DATA changed its bytes", err)
	}
	if f.meter.actual[2]-before[2] != uint64(2*len(s.config.policy.raw)+len(ticket.raw)) || f.meter.actual[10] <= before[10] || f.meter.clocks != clocks || s.stored != stored || s.scanControl.held != held || s.last != nil || s.sequence != 1 || f.body.active {
		t.Fatal("formatting skipped observation, sampled a new clock or created retained authority")
	}
	for k, upper := range providerFirstAdmissionTicketUpper() {
		if f.meter.actual[k]-before[k] > upper {
			t.Fatal("ticket formatter exceeded its separate logical component", k)
		}
	}
}

func TestProviderFirstAdmissionTicketConstructionRefusesIncompleteData(t *testing.T) {
	s, f, _, _, original := providerFirstAdmissionSeedFixture(t)
	for _, mode := range []string{"policy", "policy-size", "intent", "intent-size", "intent-hash", "identity", "request", "method-size", "params-size", "body-size", "head-size", "wire-size", "input-hash", "params-hash", "wire-digest", "params-digest", "not-finalized", "body", "head", "wire", "clock", "expired", "sum", "overflow", "live", "retained", "requests", "segments", "wire-upper", "remaining-request", "remaining-wire", "remaining-segment", "evm", "policy-bytes", "policy-limits", "policy-clock", "duration", "budget"} {
		t.Run(mode, func(t *testing.T) {
			p, i := *s.config.policy, *original
			policy, intent := &p, &i
			request, err := newProviderIntegrationRequest("eth_getBlockByNumber", []any{"finalized", false})
			if err != nil {
				t.Fatal(err)
			}
			clock, reservation := original.clock, providerFirstTicketReservationData()
			var meter *providerWorkMeter // Pure refusal DATA except the last quota case.
			switch mode {
			case "policy":
				policy = nil
			case "policy-size":
				p.raw = strings.Repeat(" ", 64<<10+1)
			case "intent":
				intent = nil
			case "intent-size":
				i.raw = strings.Repeat(" ", 64<<10+1)
			case "intent-hash":
				i.digest = "incomplete finite DATA"
			case "identity":
				i.identity.Namespace = "incomplete finite DATA"
			case "request":
				request = nil
			case "method-size":
				request.method = strings.Repeat("a", 65)
			case "params-size":
				request.params = strings.Repeat(" ", 8<<10+1)
			case "body-size":
				request.body = strings.Repeat(" ", 16<<10+1)
			case "head-size":
				request.head = strings.Repeat(" ", 513)
			case "wire-size":
				request.wire = strings.Repeat(" ", 16<<10+1)
			case "input-hash", "wire-digest":
				request.inputDigest = "incomplete finite DATA"
				if mode == "wire-digest" {
					request.inputDigest = admissionTestDigest("b")
				}
			case "params-hash", "params-digest":
				request.paramsDigest = "incomplete finite DATA"
				if mode == "params-digest" {
					request.paramsDigest = admissionTestDigest("b")
				}
			case "not-finalized":
				request.params = `["latest",false]`
			case "body":
				request.body = "[]"
			case "head":
				request.head = "finite DATA"
			case "wire":
				request.wire = "finite DATA"
			case "clock":
				clock.Epoch = admissionTestDigest("b")
			case "expired":
				i.deadline = uint64(clock.WallMS - clock.UncertaintyMS)
			case "sum":
				reservation.work[10]++
			case "overflow":
				reservation.execution[10], reservation.remaining[10] = ^uint64(0), 1
			case "live":
				reservation.live = 0
			case "retained":
				reservation.retained--
			case "requests":
				reservation.execution[0], reservation.work[0] = 0, 0
			case "segments":
				reservation.execution[8], reservation.work[8] = 0, 0
			case "wire-upper":
				reservation.execution[1], reservation.work[1] = uint64(len(request.wire)), uint64(len(request.wire))
			case "remaining-request":
				reservation.execution[0], reservation.remaining[0] = 0, 1
			case "remaining-wire":
				reservation.execution[1]--
				reservation.remaining[1] = 1
			case "remaining-segment":
				reservation.execution[8], reservation.remaining[8] = 0, 1
			case "evm":
				reservation.execution[3], reservation.work[3] = 1, 1
			case "policy-bytes", "policy-limits", "policy-clock", "duration":
				m := contentJSON([]byte(p.raw)).(map[string]any)
				switch mode {
				case "policy-bytes":
					p.raw = `[]`
				case "policy-limits":
					delete(m, "limits")
				case "policy-clock":
					delete(m, "clock")
				case "duration":
					m["limits"].(map[string]any)["operation"].(map[string]any)["deadline_ms"] = "18446744073709551615"
				}
				if mode != "policy-bytes" {
					p.raw = string(providerIntegrationJSON(m))
				}
			case "budget":
				meter = f.meter
				meter.upper[10] = meter.actual[10]
			}
			stored, held := s.stored, s.scanControl.held
			if got, err := providerFirstAdmissionTicketDataObserved(policy, intent, request, reservation, clock, meter); err == nil || got != nil || s.stored != stored || s.scanControl.held != held || s.last != nil {
				t.Fatal("incomplete candidate ticket created a result or retained authority", mode)
			}
		})
	}
}

func providerFirstAdmissionSeedInput(t *testing.T, owner, view *providerCustodyStore, kind string, raw []byte) providerIntegrationObject {
	t.Helper()
	object, err := newProviderIntegrationObject(kind, raw, owner.config.bounds.maximumObjectBytes)
	if err != nil || owner.write(owner.evidence, owner.objectName(kind, object.digest), raw) != nil {
		t.Fatal("owned expected DATA input", kind, err)
	}
	view.objects[providerIntegrationRefKey(providerIntegrationObjectRef{kind: kind, digest: object.digest})] = providerIntegrationObjectRef{kind, object.digest, uint64(len(raw))}
	return object
}

func TestProviderFirstAdmissionSeedUsesHistoricalOriginalsWithoutCurrentHeadSubstitution(t *testing.T) {
	s, f, r, binding, _ := providerFirstAdmissionSeedFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	before, stored, held := f.meter.actual, s.stored, s.scanControl.held
	seed, err := r.replay.body.admissionScanSeed(r.view)
	if err != nil || seed == nil || seed.seed.work != binding.total.work || seed.seed.retained != binding.total.retained || seed.metadata == 0 || !r.view.requiredReferences["scan-admission-binding:"+providerCandidateHash(providerScanAdmissionBindingDomain, mustProviderBindingBytes(t, s.config, binding))] || f.meter.actual[2] <= before[2] || f.meter.actual[10] <= before[10] || s.stored != stored || s.scanControl.held != held || s.last != nil || f.body.active {
		t.Fatal("original scan binding omitted a hold, reference or observation", err)
	}
	// Only provisional current state changes. This historical DATA still names
	// immutable E0/Hk; it is deliberately not proof that a later live pin agrees.
	s.frontierFiles.pinRaw = "unselected later provisional head DATA"
	s.scanControl.pin = providerScanHighwater{}
	before = f.meter.actual
	again, err := r.replay.body.admissionScanSeed(r.view)
	if err != nil || again == nil || *again != *seed || f.meter.actual[2] <= before[2] || f.meter.actual[10] <= before[10] || s.last != nil {
		t.Fatal("historical typed seed adopted a later producer pin", err)
	}
}

func mustProviderBindingBytes(t *testing.T, config providerCustodyConfig, binding providerScanAdmissionBinding) []byte {
	t.Helper()
	x, err := newProviderScanCodec(config, nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := x.admissionBinding(binding)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestProviderAdmissionHistoricalViewRequiresTheSameOwnedRootsAndActiveMeter(t *testing.T) {
	s, f, reader, _, _ := providerFirstAdmissionSeedFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.activeMeter.Store(f.meter)
	defer reader.view.activeMeter.Store(f.meter)
	// Inert owner-relationship DATA only: zero work capacity, no sample/IO,
	// admission, completion or request is executed with this unit meter.
	c := &providerIntegration{store: s}
	c.owner = c
	m := &providerWorkMeter{integration: c}
	m.owner = m
	s.activeMeter.Store(m)
	reader.view.activeMeter.Store(m)
	if providerAdmissionOriginalOwner(reader.view, m) != s || m.actual != (candidateWorkVector{}) {
		t.Fatal("same-root historical view lost its concrete owner")
	}
	m.scan = s.scanControl
	if providerAdmissionOriginalOwner(reader.view, m) != nil {
		t.Fatal("ambiguous scan/integration owner was selected")
	}
	m.scan, m.integration = nil, nil
	if providerAdmissionOriginalOwner(reader.view, m) != nil {
		t.Fatal("unowned view selected an original store")
	}
}

func TestProviderFirstAdmissionSeedRefusesMissingBindingOriginalsAndOwnership(t *testing.T) {
	for _, mode := range []string{"missing-replay", "missing-view", "no-meter", "detached", "foreign-root", "binding-absent", "binding-duplicate", "binding-key", "binding-admission", "binding-target", "frontier", "highwater", "control", "chunk", "budget", "clock", "metadata"} {
		t.Run(mode, func(t *testing.T) {
			s, f, reader, binding, _ := providerFirstAdmissionSeedFixture(t)
			s.mu.Lock()
			defer s.mu.Unlock()
			r, v := reader.replay.body, reader.view
			switch mode {
			case "missing-replay":
				r = nil
			case "missing-view":
				v = nil
			case "no-meter":
				r.meter = nil
			case "detached":
				s.activeMeter.Store(nil)
			case "foreign-root":
				v.custody = nil
			case "binding-absent", "binding-admission", "binding-target":
				for key, ref := range v.objects {
					if ref.kind == "scan-admission-binding" {
						delete(v.objects, key)
					}
				}
				if mode != "binding-absent" {
					if mode == "binding-admission" {
						binding.admission = admissionTestDigest("b")
					} else {
						binding.target = admissionTestDigest("b")
					}
					providerFirstAdmissionSeedInput(t, s, v, "scan-admission-binding", mustProviderBindingBytes(t, s.config, binding))
				}
			case "binding-duplicate":
				ref := providerIntegrationObjectRef{"scan-admission-binding", admissionTestDigest("b"), 1}
				v.objects[providerIntegrationRefKey(ref)] = ref
			case "binding-key":
				for key, ref := range v.objects {
					if ref.kind == "scan-admission-binding" {
						delete(v.objects, key)
						v.objects["inconsistent finite key DATA"] = ref
					}
				}
			case "frontier":
				reader.replay.current.commit = admissionTestDigest("b")
			case "highwater":
				if err := s.custody.root.Remove(providerCustodyName("highwater", 1)); err != nil {
					t.Fatal(err)
				}
			case "control":
				if err := s.evidence.root.Remove(providerCustodyName("scan-highwater", 0)); err != nil {
					t.Fatal(err)
				}
			case "chunk":
				if err := s.evidence.root.Remove(providerScanReceiptName("chunk", binding.lastChunk, nil)); err != nil {
					t.Fatal(err)
				}
			case "budget":
				f.meter.actual[10] = f.meter.upper[10]
			case "clock":
				s.clock.mu.Lock()
				s.clock.read = s.clock.maximum
				s.clock.mu.Unlock()
			case "metadata":
				reader.replay.preI.metadata = ^uint64(0)
			}
			stored, held := s.stored, s.scanControl.held
			if got, err := r.admissionScanSeed(v); err == nil || got != nil || s.stored != stored || s.scanControl.held != held || s.last != nil {
				t.Fatal("missing or unowned originals seeded a first admission", mode)
			}
		})
	}
	if providerAdmissionOriginalOwner(nil, nil) != nil {
		t.Fatal("missing original owner")
	}
}

func TestProviderFirstTypedAdmissionDerivesSeedBeforeTicketAppend(t *testing.T) {
	for _, mode := range []string{"typed", "zero-seed", "full-frontier"} {
		t.Run(mode, func(t *testing.T) {
			zeroSeed := mode == "zero-seed"
			s, f, reader, binding, intent := providerFirstAdmissionSeedFixture(t)
			s.mu.Lock()
			defer s.mu.Unlock()
			request, err := newProviderIntegrationRequest("eth_getBlockByNumber", []any{"finalized", false})
			if err != nil {
				t.Fatal(err)
			}
			_, _, ticketData := admissionTestTicketFixture(t)
			id := intent.identity
			ticketData["admission_digest"], ticketData["namespace"], ticketData["instance_key"], ticketData["source_id"] = intent.digest, id.Namespace, id.Instance, id.SourceID
			ticketData["segment"], ticketData["input_digest"], ticketData["params_digest"], ticketData["deadline_wall_ms"] = candidateAdmissionSegment(intent.digest, 0), request.inputDigest, request.paramsDigest, strconv.FormatUint(intent.deadline, 10)
			ticket := admissionTestTicket(t, s.config.policy, intent, 0, ticketData)
			seed := &providerLedgerInitialCharge{binding.total.work, binding.total.retained}
			if zeroSeed {
				seed = nil // Fixed incorrect B0 fixture, no operating request.
			}
			expected, err := newProviderContinuingLedgerSeeded(s.config.policy, intent, ticket, *s.config.bounds.continuation, nil, seed)
			if err != nil {
				t.Fatal(err)
			}
			change, err := expected.Current()
			if err != nil {
				t.Fatal(err)
			}
			rootData := contentJSON([]byte(reader.replay.body.anchor.raw)).(map[string]any)
			rootData["sequence"], rootData["previous_root"], rootData["budget_root"] = "1", reader.replay.body.anchor.digest, change.digest
			root := providerFirstAdmissionSeedInput(t, s, reader.view, "enrollment", providerIntegrationJSON(rootData))
			bindingObject, bindingErr := newProviderIntegrationObject("scan-admission-binding", mustProviderBindingBytes(t, s.config, binding), s.config.bounds.maximumObjectBytes)
			if bindingErr != nil {
				t.Fatal(bindingErr)
			}
			fresh := []providerIntegrationObject{root, bindingObject}
			for _, input := range []providerIntegrationObjectInput{{"admission", []byte(intent.raw)}, {"ticket", []byte(ticket.raw)}, {"budget", []byte(change.raw)}, {"request", []byte(request.wire)}, {"params", []byte(request.params)}} {
				fresh = append(fresh, providerFirstAdmissionSeedInput(t, s, reader.view, input.kind, input.raw))
			}
			for _, node := range change.LedgerNodeBytes() {
				fresh = append(fresh, providerFirstAdmissionSeedInput(t, s, reader.view, "ledger-node", node))
			}
			body := reader.replay.body
			body.intent = nil
			before, stored, held := f.meter.actual, s.stored, s.scanControl.held
			if mode == "full-frontier" {
				err = providerFirstAdmissionExpectedFrontier(t, s, f, reader, intent, ticket, root, expected, fresh)
			} else {
				err = body.apply(reader.view, map[string]any{"kind": "admit", "operation": ticket.digest, "new_enrollment_root": root.digest})
			}
			if zeroSeed {
				if err == nil {
					t.Fatal("zero-seed B0 replaced the original O+Q charge")
				}
			} else if err != nil || body.ledger == nil || body.ledger.current.digest != change.digest || body.ledger.current.totals.bootstrap.charged != binding.total.work || body.ledger.current.totals.liveUsed != binding.total.retained || body.ledger.current.totals.retainedUsed != binding.total.retained || body.ledger.current.totals.bootstrap.outstanding != ticket.work || body.scanMetadata == 0 || f.meter.actual[2] <= before[2] || f.meter.actual[10] <= before[10] {
				t.Fatal("first typed B0 did not independently seed before the ticket", err)
			}
			if s.stored != stored || s.scanControl.held != held || s.last != nil || s.sequence != 1 || f.body.active {
				t.Fatal("typed DATA replay became an E1 write, refund or ACK")
			}
		})
	}
}

// Normal, bounded candidate DATA construction only. The independent replay
// gets fresh originals and its own ledger/index maps; no first-E1 writer,
// physical HEAD update, current-tip acknowledgment or RPC is called.
func providerFirstAdmissionExpectedFrontier(t *testing.T, owner *providerCustodyStore, f *providerFirstAdmissionOwner, reader *providerScanReplayReader, intent *CandidateAdmissionIntent, ticket *CandidateAdmissionTicket, rootObject providerIntegrationObject, expected *CandidateAdmissionLedger, fresh []providerIntegrationObject) error {
	t.Helper()
	v, replay := reader.view, reader.replay
	indexes, _, err := newProviderLedgerIndexes(v.config.identity, 1, *v.config.bounds.continuation)
	anchor := replay.body.anchor
	e0, objectErr := newProviderIntegrationObject("enrollment", []byte(anchor.raw), 64<<10)
	id := v.config.identity
	enroll := providerCandidateHash("sage-provider-enroll-operation-v1", []byte(v.config.anchor+id.Instance+id.Policy+id.SourceID))
	if err != nil || objectErr != nil {
		t.Fatal(err, objectErr)
	}
	initial, err := indexes.prepare("enroll", enroll, 1, e0, anchor, nil)
	if err != nil || indexes.apply(initial) != nil || indexes.heads != replay.current.indexes {
		t.Fatal("independent expected E0 index DATA", err)
	}
	replay.body.intent = intent
	controls, err := replay.body.admissionScanSeed(v)
	if err != nil {
		t.Fatal(err)
	}
	prior := replay.preI
	prior.metadata += controls.metadata // Finite owned fixture, product uses checked-sum.
	if err := indexes.bindRetainedLedger(expected, prior); err != nil {
		t.Fatal(err)
	}
	root, err := ParseCandidateEnrollmentRoot(replay.body.context, []byte(rootObject.raw), 64<<10)
	entry, objectErr := newProviderIntegrationObject("ticket", []byte(ticket.raw), 64<<10)
	if err != nil || objectErr != nil {
		t.Fatal(err, objectErr)
	}
	delta, err := indexes.prepare("admit", ticket.digest, 2, entry, root, expected)
	if err != nil || len(delta.mutations) != 3 {
		t.Fatal("expected three independent first-admit index mutations", err)
	}
	heads := delta.previous
	for _, mutation := range delta.mutations {
		fresh = append(fresh, providerFirstAdmissionSeedInput(t, owner, v, "index-mutation", []byte(mutation.object.raw)))
		heads[mutation.chain] = mutation.object.digest
	}
	if len(fresh) != 14 {
		t.Fatal("closed first-E1 expected object count", len(fresh))
	}
	inventory, err := newProviderFrontierInventory(v.objectsRoot, fresh, v.config.bounds.maximumEntries, 64<<20)
	now, clockErr := f.sampleLocked()
	previous, previousErr := replay.current.bytes()
	if err != nil || clockErr != nil || previousErr != nil {
		t.Fatal(err, clockErr, previousErr)
	}
	link := providerFrontierLink{identity: id, fence: 1, sequence: 1, commit: v.tip, setup: replay.current.setup, enrollment: v.enrollment, objects: v.objectsRoot, frontier: providerCandidateHash("sage-provider-custody-frontier-v1", previous)}
	claim, err := providerFrontierIntent(link, root, inventory, "admit", ticket.digest, now, v.config.bounds)
	commit := providerFrontierCommit(claim)
	if err != nil {
		t.Fatal(err)
	}
	frontier := replay.current
	frontier.sequence, frontier.previous, frontier.commit = 2, link.frontier, providerCandidateHash("sage-provider-custody-commit-v2", commit)
	frontier.objects, frontier.enrollment, frontier.indexes = inventory.digest, root.digest, heads
	frontier.admission, frontier.budget, frontier.clock = intent.digest, expected.current.digest, providerClockDiagnostic(now)
	raw, err := frontier.bytes()
	if err != nil {
		t.Fatal(err)
	}
	after := f.mainPin
	after.ordinal, after.sequence, after.previous = 2, 2, providerCandidateHash("sage-provider-custody-highwater-v1", []byte(f.mainRaw))
	after.commit, after.frontier = frontier.commit, providerCandidateHash("sage-provider-custody-frontier-v1", raw)
	preI := replay.preI.metadata
	replay.body.intent = nil
	cost, err := replay.applyWithHighwater(v, inventory, fresh, claim, commit, raw, after)
	if err == nil && (cost.objects != 14 || replay.preI.metadata != preI+controls.metadata || replay.body.scanMetadata != controls.metadata || replay.indexes.ledger != replay.body.ledger) {
		t.Fatal("common scan metadata omitted, repeated or detached from typed indexes")
	}
	return err
}
