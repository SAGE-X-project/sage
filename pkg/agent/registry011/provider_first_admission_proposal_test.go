package registry011

import (
	"strconv"
	"testing"
)

// The caller holds s.mu. This returns the genuine single-use F and fresh E0
// reader, without selecting a host/profile or replacing its clock or meter.
func providerFirstProposalInputs(t *testing.T, s *providerCustodyStore, f *providerFirstAdmissionOwner) (*providerScanReplayReader, CandidateClockDiagnostic) {
	t.Helper()
	if f.store != s {
		t.Fatal("fixture must retain its original store")
	}
	if err := f.startLocked(); err != nil {
		t.Fatal(err)
	}
	r, err := f.replayEnrollmentLocked()
	if err != nil {
		t.Fatal(err)
	}
	clock, err := f.sampleLocked()
	if err != nil {
		t.Fatal(err)
	}
	return r, providerClockDiagnostic(clock)
}

func TestProviderFirstAdmissionProposalUsesNewCoordinatorMapsWithoutWriting(t *testing.T) {
	s, f := providerFirstAdmissionOwnerFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	input, clock := providerFirstProposalInputs(t, s, f)
	before, stored, held, clocks := f.meter.actual, s.stored, s.scanControl.held, f.meter.clocks
	p, err := providerFirstAdmissionProposalData(f, input, providerFirstTicketReservationData(), clock)
	if err != nil || p == nil {
		t.Fatal("complete proposed first-admit DATA", err)
	}
	if p.indexes == input.replay.indexes || p.ledger == input.replay.body.ledger || input.replay.body.intent != nil || input.replay.body.ledger != nil || p.ledger.current.totals.bootstrap.charged != p.binding.total.work || p.ledger.current.totals.liveUsed != p.binding.total.retained || p.ledger.current.totals.retainedUsed != p.binding.total.retained || p.ledger.current.totals.bootstrap.outstanding != p.ticket.work || p.ticket.deadline > p.intent.deadline || p.intent.clock != providerClockDiagnostic(f.origin) {
		t.Fatal("proposal imported producer maps or omitted the original O+Q seed")
	}
	counts := map[string]int{}
	for _, object := range p.fresh {
		counts[object.kind]++
		digest, ok := providerIntegrationObjectDigest(object.kind, []byte(object.raw))
		if !ok || digest != object.digest {
			t.Fatal("proposed object did not retain its own immutable bytes")
		}
	}
	if len(p.fresh) != 14 || counts["ledger-node"] != 4 || counts["index-mutation"] != 3 || counts["scan-admission-binding"] != 1 || len(p.inventory.entries) != 14 || p.after.sequence != 2 || p.after.ordinal != 2 || p.after.previous != providerCandidateHash("sage-provider-custody-highwater-v1", []byte(f.mainRaw)) {
		t.Fatal("proposal omitted an exact first-E1 object or original predecessor")
	}
	bound, err := mustProviderBindingCodec(t, s.config).parseAdmissionBinding(mustProviderBindingBytes(t, s.config, p.binding))
	if err != nil || bound != p.binding || p.UnmarshalJSON(nil) == nil || f.meter.actual[2] <= before[2] || f.meter.actual[10] <= before[10] || f.meter.clocks <= clocks {
		t.Fatal("proposal lost original closure observations or became serializable authority", err)
	}
	if s.stored != stored || s.scanControl.held != held || s.sequence != 1 || s.last != nil || f.body.active || s.frontierFiles.pin != f.mainPin {
		t.Fatal("candidate proposal became a physical E1 write, new hold or ACK")
	}
}

func mustProviderBindingCodec(t *testing.T, config providerCustodyConfig) *providerScanCodec {
	t.Helper()
	x, err := newProviderScanCodec(config, nil)
	if err != nil {
		t.Fatal(err)
	}
	return x
}

func TestProviderFirstAdmissionProposalReplaysInSeparateOriginalCustodianState(t *testing.T) {
	s, f := providerFirstAdmissionOwnerFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	input, clock := providerFirstProposalInputs(t, s, f)
	p, err := providerFirstAdmissionProposalData(f, input, providerFirstTicketReservationData(), clock)
	if err != nil {
		t.Fatal(err)
	}
	// Ordinary candidate read-only replay, using fresh maps and the original
	// E0 files. The proposal's ledger/index maps are never installed in it.
	reader, err := newProviderScanReplayReaderWithOutput(f.control, f.meter, providerReplayBytes(f.targetRaw, f.meter), f.readerOutput)
	if err != nil {
		t.Fatal(err)
	}
	for reader.nextSetup <= reader.selected.ordinal-reader.selected.sequence {
		if reader.setup() != nil {
			t.Fatal("independent setup DATA")
		}
	}
	for reader.view.sequence < reader.selected.sequence {
		if reader.main() != nil {
			t.Fatal("independent E0 DATA")
		}
	}
	if !reader.lineage() {
		t.Fatal("independent E0 lineage")
	}
	// Expected objects are fixture DATA in owned roots, exactly as other typed
	// replay units. This does not call the absent prepaid E1 writer or its CAS.
	for _, object := range p.fresh {
		providerFirstAdmissionSeedInput(t, s, reader.view, object.kind, []byte(object.raw))
	}
	cost, err := reader.replay.applyWithHighwater(reader.view, p.inventory, p.fresh, p.claim, p.commit, p.frontier, p.after)
	if err != nil || cost.objects != 14 || reader.replay.body.ledger == nil || reader.replay.body.ledger == p.ledger || reader.replay.indexes == p.indexes || reader.replay.body.ledger.current.digest != p.ledger.current.digest || reader.replay.body.ledger.current.totals != p.ledger.current.totals || s.sequence != 1 || s.last != nil || s.frontierFiles.pin != f.mainPin {
		t.Fatal("coordinator delta replaced independent typed originals or became E1 authority", err)
	}
}

func TestProviderFirstAdmissionProposalRefusesIncompleteOriginalData(t *testing.T) {
	for _, mode := range []string{"owner", "reader", "replay", "anchor", "sequence", "already-admitted", "clock", "entries", "objects", "reservation", "context", "metadata-overflow", "payload", "old-index", "seed-plus-future-live", "control-original", "receipt-original", "budget"} {
		t.Run(mode, func(t *testing.T) {
			s, f := providerFirstAdmissionOwnerFixture(t)
			s.mu.Lock()
			defer s.mu.Unlock()
			reader, clock := providerFirstProposalInputs(t, s, f)
			owner, input, reservation := f, reader, providerFirstTicketReservationData()
			switch mode {
			case "owner":
				owner = nil
			case "reader":
				input = nil
			case "replay":
				reader.replay = nil
			case "anchor":
				reader.replay.body.anchor = nil
			case "sequence":
				reader.view.sequence = 2
			case "already-admitted":
				reader.replay.current.admission = admissionTestDigest("b")
			case "clock":
				clock.Epoch = admissionTestDigest("b")
			case "entries":
				// Only provisional view DATA; the bound is not changed after H0.
				reader.view.objects["unreferenced finite DATA"] = providerIntegrationObjectRef{}
			case "objects":
				for k := range reader.view.objects {
					delete(reader.view.objects, k)
					break
				}
			case "reservation":
				reservation.work[10]++
			case "context":
				reader.replay.body.context = nil
			case "metadata-overflow":
				reader.replay.preI.metadata = ^uint64(0)
			case "payload":
				reader.replay.preI.payload = ^uint64(0)
			case "old-index":
				reader.replay.current.indexes[0] = admissionTestDigest("b")
			case "seed-plus-future-live":
				p := contentJSON([]byte(s.config.policy.raw)).(map[string]any)
				live, ok := providerCandidateUint(p["limits"].(map[string]any)["live_disk_bytes"], true)
				if !ok {
					t.Fatal("expected immutable live limit")
				}
				reservation.live, reservation.retained = live, live
			case "control-original":
				if s.custody.root.Remove(providerCustodyName("scan-highwater", 0)) != nil {
					t.Fatal("owned missing-original fixture")
				}
			case "receipt-original":
				if s.evidence.root.Remove(providerScanReceiptName("chunk", f.lastChunk, nil)) != nil {
					t.Fatal("owned missing-original fixture")
				}
			case "budget":
				f.meter.upper[10] = f.meter.actual[10]
			}
			stored, held := s.stored, s.scanControl.held
			if got, err := providerFirstAdmissionProposalData(owner, input, reservation, clock); err == nil || got != nil || s.stored != stored || s.scanControl.held != held || s.last != nil {
				t.Fatal("incomplete original DATA became a proposal or retaining authority", mode)
			}
		})
	}
}

func TestProviderFirstAdmissionProposalKeepsEachFreshOriginalDecodedCeiling(t *testing.T) {
	// Each amount selects a NEW isolated candidate BEFORE H0. No used F/B/O
	// limit, actual measurement, clock or operating profile is replaced.
	refused, completed := 0, 0
	for decoded := uint64(32 << 10); decoded <= 256<<10; decoded += 8 << 10 {
		t.Run(strconv.FormatUint(decoded, 10), func(t *testing.T) {
			first := providerFirstAdmissionBoundsFixture()
			first.work[2] = decoded
			s := providerScanEnrollmentOriginalsWithFirst(t, &first)
			x := mustProviderBindingCodec(t, s.config)
			raw, err := x.target(providerScanTarget{setup: s.setupTip, custody: s.tip, frontier: s.frontierFiles.pin.frontier, objects: s.objectsRoot, enrollment: s.enrollment})
			if err != nil {
				t.Fatal(err)
			}
			a := providerScanAttemptFixture(t, s, s.scanControl.pin)
			a.target = x.hash("sage-provider-scan-target-v1", raw)
			if s.scanControl.claim(a) != nil {
				t.Fatal("fresh bounded claim DATA")
			}
			if _, err := s.scanControl.inspectBeforeAdmission(raw); err != nil {
				t.Fatal("inspection cannot borrow first-admission funds", err)
			}
			f := s.firstAdmissionOwner
			if f == nil {
				t.Fatal("inspection did not retain its finishing owner")
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			defer f.abortLocked()
			stored, held := s.stored, s.scanControl.held
			if err := f.startLocked(); err != nil {
				t.Fatal(err)
			}
			reader, err := f.replayEnrollmentLocked()
			var p *providerFirstAdmissionProposal
			if err == nil {
				clock, sampleErr := f.sampleLocked()
				if sampleErr != nil {
					t.Fatal(sampleErr)
				}
				p, err = providerFirstAdmissionProposalData(f, reader, providerFirstTicketReservationData(), providerClockDiagnostic(clock))
			}
			if err != nil {
				refused++
				if p != nil || !f.meter.failed {
					t.Fatal("exhausted fresh candidate yielded proposed DATA without refusal")
				}
			} else {
				completed++
				if p == nil || len(p.fresh) != 14 {
					t.Fatal("bounded proposal DATA was incomplete")
				}
			}
			if f.meter.actual[2] < f.startWork[2] || f.meter.actual[2]-f.startWork[2] > decoded || s.stored != stored || s.scanControl.held != held || s.sequence != 1 || s.last != nil || s.frontierFiles.pin != f.mainPin {
				t.Fatal("candidate exceeded its original decoded ceiling or retained E1")
			}
		})
	}
	if refused == 0 || completed == 0 {
		t.Fatal("finite fresh profiles must exercise both exhaustion and bounded DATA construction", refused, completed)
	}
}
