package registry011

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

func TestProviderHistoryReconstructionInitialMatchesWholeSourceLedger(t *testing.T) {
	for _, row := range []struct {
		name                                     string
		metadata, object, policy, objects, setup uint64
		parts                                    [9][2]uint64
		body, normal, incoming                   [2]uint64
		pairs                                    uint64
	}{
		{"small", 4096, 4096, 1024, 4, 0, [9][2]uint64{
			{0, 234938830}, {26624, 42977626488}, {155648, 805544918}, {0, 1208353530}, {0, 2138164},
			{131072, 8926294934930}, {8192, 100810828}, {0, 34362957812}, {0, 5426},
		}, [2]uint64{321536, 9005987310926}, [2]uint64{321536, 9014711611918}, [2]uint64{65536, 1099511627777}, 12},
		{"medium", 65536, 1 << 20, 8192, 64, 3, [9][2]uint64{
			{0, 60130460110}, {212992, 44695437688}, {16252928, 15496417141718}, {0, 1208353530}, {0, 314740356},
			{2097152, 9097427944850}, {16384, 17314234444}, {0, 34632885752}, {0, 12458},
		}, [2]uint64{18579456, 24752141210906}, [2]uint64{18579456, 24760865511898}, [2]uint64{1048576, 1099511627777}, 4032},
		{"maximum", 1 << 20, 32 << 20, 65536, 1024, 64, [9][2]uint64{
			{0, 15393177469390}, {1703936, 154621706616}, {471334912, 15762702080173014}, {0, 1208353530}, {0, 10026840964},
			{25690112, 44144423994770}, {16384, 4398180876364}, {0, 34730102072}, {0, 138802},
		}, [2]uint64{498745344, 15826838449655522}, [2]uint64{498745344, 15826847173956514}, [2]uint64{33554432, 1125899906842625}, 261120},
	} {
		t.Run(row.name, func(t *testing.T) {
			limits := providerHistoryLegacyControlLimits{row.metadata, row.object, row.objects, 64 << 20, 256, 4, 32}
			before := limits
			got, ok := providerHistoryReconstructionInitialUpper(limits, row.setup, row.policy)
			if !ok || limits != before || got.body != (candidateWorkVector{2: row.body[0], 10: row.body[1]}) ||
				got.normal != (candidateWorkVector{2: row.normal[0], 10: row.normal[1]}) ||
				got.incoming != (candidateWorkVector{2: row.incoming[0], 10: row.incoming[1]}) ||
				got.envelope != (candidateWorkVector{2: row.normal[0] + row.incoming[0], 10: row.normal[1] + row.incoming[1]}) ||
				got.boundary != (candidateWorkVector{10: 8724300992}) || got.inventoryPairs != row.pairs ||
				[5]uint64{got.profileCalls, got.policyDecodes, got.semanticLookups, got.frameLookups, got.frontierEncodes} != [5]uint64{14, 26, 130, 12, 30} {
				t.Fatal("whole initial source ledger lost a repeated call or independent boundary", got, ok)
			}
			for i, part := range [...]candidateWorkVector{got.profiles, got.policies, got.records, got.ledger, got.materials, got.frontiers, got.seed, got.retention, got.control} {
				if part != (candidateWorkVector{2: row.parts[i][0], 10: row.parts[i][1]}) {
					t.Fatal("independent initial-source component literal", i, part, row.parts[i])
				}
			}
		})
	}
}

func TestProviderHistoryReconstructionInitialRefusesUnselectedBounds(t *testing.T) {
	for _, mode := range []string{"metadata-zero", "metadata-large", "object-zero", "object-large", "first-objects-zero", "first-objects-large", "first-raw-zero", "first-raw-large", "entries-zero", "entries-large", "attempts-zero", "attempts-large", "chunks-zero", "chunks-overflow", "setups-large", "setups-overflow", "policy-zero", "policy-large"} {
		t.Run(mode, func(t *testing.T) {
			limits, setups, policy := historyLegacyControlLimits(), uint64(0), uint64(8192)
			switch mode {
			case "metadata-zero":
				limits.metadata = 0
			case "metadata-large":
				limits.metadata = 1<<20 + 1
			case "object-zero":
				limits.objectBytes = 0
			case "object-large":
				limits.objectBytes = 32<<20 + 1
			case "first-objects-zero":
				limits.firstObjects = 0
			case "first-objects-large":
				limits.firstObjects = 1<<20 + 1
			case "first-raw-zero":
				limits.firstRaw = 0
			case "first-raw-large":
				limits.firstRaw = 64<<20 + 1
			case "entries-zero":
				limits.entries = 0
			case "entries-large":
				limits.entries = 257
			case "attempts-zero":
				limits.attempts = 0
			case "attempts-large":
				limits.attempts = 65
			case "chunks-zero":
				limits.chunks = 0
			case "chunks-overflow":
				limits.chunks = ^uint64(0)
			case "setups-large":
				setups = 65
			case "setups-overflow":
				setups = ^uint64(0)
			case "policy-zero":
				policy = 0
			case "policy-large":
				policy = 64<<10 + 1
			}
			if got, ok := providerHistoryReconstructionInitialUpper(limits, setups, policy); ok || got != (providerHistoryReconstructionInitialCost{}) {
				t.Fatal("unselected initial source returned a partial quote", mode, got, ok)
			}
		})
	}
}

func TestProviderHistoryReconstructionInitialKeepsIndependentEncoderGuards(t *testing.T) {
	limits := historyLegacyControlLimits()
	limits.metadata, limits.objectBytes, limits.firstObjects = 4096, 4096, 4
	got, ok := providerHistoryReconstructionInitialUpper(limits, 0, 1024)
	if !ok || got.incoming != (candidateWorkVector{2: 65536, 10: 1099511627777}) ||
		got.boundary != (candidateWorkVector{10: 8724300992}) || got.ledger != (candidateWorkVector{10: 1208353530}) ||
		got.inventoryPairs != 12 || got.materials[10] != 2138164 {
		t.Fatal("small original inputs replaced independent64KiB/1MiB encoders or generated8192-byte materials", got, ok)
	}
	for _, coordinate := range []int{0, 1, 3, 4, 5, 6, 7, 8, 9} {
		if got.envelope[coordinate] != 0 {
			t.Fatal("initial model numeric DATA acquired execution or another resource", coordinate)
		}
	}
}

// This length oracle reads actual retained frames after the WHOLE original
// initial-model invocation. Its own stdlib codecs never debit the observer.
func historyReconstructionInitialDecoded(t *testing.T, m *providerHistoryRecoveryInitialModel) uint64 {
	t.Helper()
	if m == nil || !m.selectedLocked(m.recovery) || len(m.prefix.records) != 4 {
		t.Fatal("completed original initial model for independent length attribution")
	}
	lookup := func(kind, digest string) []byte {
		var raw []byte
		var count int
		for _, frame := range m.prefix.source.frames {
			for _, object := range frame.objects {
				if object.kind == kind && object.digest == digest {
					raw, count = []byte(object.raw), count+1
				}
			}
		}
		if count != 1 || len(raw) == 0 {
			t.Fatal("one selected initial source dependency", kind, count)
		}
		return raw
	}
	var rootBytes, intentBytes, frontierBytes uint64
	for _, record := range m.prefix.records {
		rootBytes += uint64(len(lookup("enrollment", record.root.digest)))
		intentBytes += uint64(len(record.intent))
		frontierBytes += uint64(len(record.rawFrontier))
	}
	first, admission := m.prefix.records[0], m.prefix.records[1]
	ticket := lookup("ticket", admission.operation)
	consumption := lookup("consumption", m.prefix.records[2].operation)
	settlement := lookup("settlement", m.prefix.records[3].operation)
	var claim []byte
	for _, setup := range m.prefix.setups {
		if setup.commit == first.before.setup {
			claim = lookup("setup", setup.claim)
		}
	}
	if len(claim) == 0 {
		t.Fatal("original first setup claim")
	}
	policy := uint64(len(m.frontier.body.policy.raw))
	decoded := 13*policy + rootBytes + uint64(len(lookup("enrollment", first.root.digest))) + 3*intentBytes + frontierBytes +
		uint64(len(lookup("budget", admission.root.budget))+2*len(lookup("admission", m.recovery.initialDescriptor.admission))+
			2*len(ticket)+4*len(consumption)+4*len(settlement)+len(claim)+
			len(lookup(providerAdmissionBindingKind(m.recovery.config.bounds), m.prefix.scan.bindingOriginal.digest)))
	var anchors, mutations, latest uint64
	var anchorCount, mutationCount, latestCount int
	for _, frame := range m.prefix.source.frames {
		for _, object := range frame.objects {
			switch object.kind {
			case "index-empty":
				anchors += uint64(len(object.raw))
				anchorCount++
			case "index-mutation":
				mutations += uint64(len(object.raw))
				mutationCount++
			case "ledger-node":
				var node struct{ Index string }
				if json.Unmarshal([]byte(object.raw), &node) != nil || node.Index == "" {
					t.Fatal("original ordinary ledger-node index")
				}
				if node.Index != "0" {
					latest += uint64(len(object.raw))
					latestCount++
				}
			}
		}
	}
	if anchorCount != 3 || mutationCount != 9 || latestCount != 3 {
		t.Fatal("original initial index/node multiplicities", anchorCount, mutationCount, latestCount)
	}
	decoded += anchors + 2*rootBytes + 2*latest + 2*uint64(len(settlement)) + 3*mutations
	state := m.frontier.body.ledger.tickets[m.recovery.initialDescriptor.ticket]
	if state.ticket == nil || m.frontier.body.firstExchange == "" {
		t.Fatal("actual completed initial ticket/exchange")
	}
	// Arguments for a read-only TEST LENGTH oracle only; this value is never
	// parsed, installed, passed to a protocol method or treated as a credential.
	oracle := &CandidateAdmissionSettlement{ticket: state.ticket, retained: m.frontier.body.firstExchange}
	request, exchange := historyReconstructionMessageDecoded(t, m, oracle)
	return decoded + 3*request + exchange
}

func TestProviderHistoryReconstructionInitialContainsBothWholeOriginalModels(t *testing.T) {
	s, in, _ := historyCollectionRecordedRunStore(t)
	o, err := newProviderHistoryGroupOwnerCandidate(s)
	if err != nil || o == nil || o.coordinator == nil || o.custodian == nil || o.coordinator.initial == "" || o.coordinator.initial != o.custodian.initial {
		t.Fatal("original pre-H0 collection initialization and matching diagnostic root", err)
	}
	// Expected diagnostic digest only. Failed producer models/caches are then
	// removed; neither independent reconstruction may use them as inputs.
	expectedRoot := o.coordinator.initial
	r, refs := historyRecoveryGroupStart(t, o)
	b := s.config.bounds
	limits := providerHistoryLegacyControlLimits{b.maximumMetadataPerCommit, b.maximumObjectBytes, in.firstRead.objects, in.firstRead.raw,
		uint64(b.maximumEntries), b.scan.maximumAttempts, b.scan.maximumChunks}
	if b.history.collection == nil || b.history.collection.input.firstRead != in.firstRead || s.objects != nil {
		t.Fatal("same selected first-read limits and discarded caches")
	}
	quote, ok := providerHistoryReconstructionInitialUpper(limits, uint64(b.maximumSetupRecoveries), uint64(len(s.config.policy.raw)))
	messages, messageOK := providerHistoryReconstructionMessageUpper(limits, s.config.capture, uint64(len(s.config.policy.raw)))
	indexes := providerHistoryReconstructionIndexUpper()
	if !ok || !messageOK {
		t.Fatal("same-source initial model and its disjoint children")
	}
	old, clocks, held, pin, ack := o.meter.actual, s.clock.read, s.stored, o.pin, o.last
	originals := providerHistoryProvisionedOriginalBytes(t, s.config)
	var actual, boundary candidateWorkVector
	for pass := range 2 {
		historyRecoveryModelPrefix(t, r, pass, refs)
		start, files, rawBytes := r.meter.actual, r.inputFiles, r.inputBytes
		m, err := r.replayInitialModelCandidate(context.Background(), pass)
		if err != nil || m == nil || !m.selectedLocked(r) || r.inputFiles != files+2 || r.inputBytes != rawBytes || r.meter.actual[2] < start[2] ||
			r.meter.actual[10] <= start[10] {
			t.Fatal("whole original initial-model invocation and its two empty lock reads", pass, err)
		}
		decoded := historyReconstructionInitialDecoded(t, m)
		if decoded == 0 || r.meter.actual[2]-start[2] != decoded {
			t.Fatal("whole initial-model decoded stream differs from actual frame lengths", pass, r.meter.actual[2]-start[2], decoded)
		}
		actual[2] += decoded
		actual[10] += r.meter.actual[10] - start[10]
		if selected, err := m.selectFirstFinalizedCandidate(context.Background()); err != nil || selected != in.journal.Deployment {
			t.Fatal("separately priced original finalized selection before root boundary", pass, err)
		}
		func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			start, files, rawBytes := r.meter.actual, r.inputFiles, r.inputBytes
			f := m.frontier.current
			raw, err := f.bytesObserved(r.meter)
			if err != nil || !providerReplayTextEqual(providerReplayDigest("sage-provider-custody-frontier-v1", raw, r.meter), r.initial.pin.frontier, r.meter) {
				t.Fatal("original final frontier boundary", pass, err)
			}
			root := providerHistoryInitialRootObserved(r.parent.bounds, r.config.profile, r.initial.pin, m.frontier.body.firstExchange,
				f.budget, f.objects, &m.firstFinalized, r.meter)
			if root != expectedRoot || r.meter.actual[2] != start[2] || r.meter.actual[10] <= start[10] || r.inputFiles != files || r.inputBytes != rawBytes {
				t.Fatal("original initial-root boundary changed bytes, input effects or its diagnostic root", pass)
			}
			boundary[10] += r.meter.actual[10] - start[10]
		}()
	}
	// Only FOUR intentional empty-file inputs from the ALREADY COMPOSED file
	// row are added to this measured window. No unrelated catalog/legacy/I/O
	// headroom or later root boundary can mask a body/codec mismatch.
	fileWork := 4 * (3 + 18 + 767 + b.history.catalogEntries*511 + 1)
	if actual[2] > quote.body[2]+indexes.normal[2]+messages.normal[2] ||
		actual[10] > quote.body[10]+indexes.normal[10]+messages.normal[10]+fileWork || boundary[10] > quote.boundary[10] ||
		o.meter.actual != old || s.clock.read != clocks || s.stored != held || o.pin != pin || o.last != ack ||
		!o.failed || !o.meter.closed || s.activeMeter.Load() != nil || s.objects != nil || !r.active || r.closed || r.failed ||
		!reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, s.config)) {
		t.Fatal("whole initial or separate root interval exceeded its rows or renewed execution", actual, boundary)
	}
}
