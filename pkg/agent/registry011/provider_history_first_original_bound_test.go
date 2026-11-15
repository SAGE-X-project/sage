package registry011

import (
	"context"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func TestProviderHistoryFirstOriginalUpperMatchesIndependentRows(t *testing.T) {
	// Literal sums were derived separately from the retained read/codec
	// sources. Only the existing architecture-dependent logical copy argument
	// is added here; reflect.Size is not an allocator/RAM measurement.
	backing := uint64(reflect.TypeFor[ObservationChecks]().Size() + reflect.TypeFor[providerHistorySemanticFold]().Size() +
		reflect.TypeFor[providerHistoryCollectionCursor]().Size() + reflect.TypeFor[FinalizedHeader]().Size())
	for _, tc := range []struct {
		limits                         providerHistoryLegacyControlLimits
		headers, body, wire            int
		decoded, logicalWithoutBacking uint64
	}{
		{providerHistoryLegacyControlLimits{4096, 4096, 4, 65536, 1, 1, 1}, 1024, 4096, 8192, 1081036, 5581003738056},
		{historyFirstOriginalCompositionLimits(), 8192, 131072, 262144, 4751393, 6101990678681},
		{providerHistoryLegacyControlLimits{65536, 1 << 20, 64, 64 << 20, 256, 4, 32}, 8192, 131072, 262144, 72892449, 78518378782873},
		{providerHistoryLegacyControlLimits{1 << 20, 32 << 20, 1 << 20, 64 << 20, 256, 64, 1024}, 65536, 8 << 20, 16 << 20, 268509217, 5834982382764025},
	} {
		capture := ProviderCaptureLimits{Timeout: time.Second, HeaderBytes: tc.headers, BodyBytes: tc.body, WireBytes: tc.wire, JSONDepth: 64}
		got, ok := providerHistoryFirstPReadUpper(tc.limits, capture)
		want := providerHistoryFirstPReadBounds{objects: tc.limits.firstObjects, raw: tc.limits.firstRaw,
			work: candidateWorkVector{2: tc.decoded, 10: tc.logicalWithoutBacking + backing}}
		if !ok || got != want || !got.valid() {
			t.Fatal("independent complete first-reader source row", tc, got, want)
		}
	}
}

func historyFirstOriginalCompositionLimits() providerHistoryLegacyControlLimits {
	// Selected UNIT profile inputs, before the manifest/profile/anchor/H0.
	return providerHistoryLegacyControlLimits{16384, 128 << 10, 64, 1 << 20, 256, 4, 42}
}

func TestProviderHistoryFirstOriginalUpperFundsOriginalOwnerAndRecordedReplay(t *testing.T) {
	limits := historyFirstOriginalCompositionLimits()
	composition, ok := providerHistoryCollectionFirstReadCompositionUpper(4, 28, limits, historyCollectionInput(t).capture)
	if !ok {
		t.Fatal("finite selected first-reader source composition")
	}
	read := composition.first
	s := historyCollectionAdmissionStore(t, true, &read)
	b, p := s.config.bounds, s.history
	if b.maximumMetadataPerCommit != limits.metadata || b.maximumObjectBytes != limits.objectBytes ||
		uint64(b.maximumEntries) != limits.entries || b.scan.maximumAttempts != limits.attempts || b.scan.maximumChunks != limits.chunks ||
		p.collectionInput.firstRead != read || p.firstOwner == nil || p.firstOwner != s.firstOperation || p.firstOwner.state != providerFirstOperationDone {
		t.Fatal("derived inputs differ from the original immutable selected profile")
	}
	legacy, legacyOK := providerHistoryGroupInitializationUpper(b)
	initial, initialOK := providerHistoryGroupInitializationForProfileUpper(b)
	if !legacyOK || !initialOK {
		t.Fatal("original initialization accounting precursor")
	}
	for coordinate := range initial {
		if initial[coordinate] != legacy[coordinate]+composition.initialization[coordinate] {
			t.Fatal("first-reader initialization was omitted or counted twice", coordinate)
		}
	}
	held, clock, origin := s.stored, s.clock.read, p.original
	before := providerHistoryProvisionedOriginalBytes(t, s.config)
	o, err := newProviderHistoryGroupOwnerCandidate(s)
	if err != nil || o == nil || o.phase != "prepared" || o.coordinator == o.custodian || o.coordinator.head != o.custodian.head ||
		o.firstPReading || s.clock.read != clock+2 || s.stored != held || p.original != origin || s.activeMeter.Load() != nil ||
		!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
		t.Fatal("derived first-reader row failed the two original initialization passes", err)
	}
	m, upper := o.meter, o.meter.upper
	ack, err := o.retainCollectionAdmissionCandidate(context.Background())
	if err != nil || ack == nil {
		t.Fatal("original collection T must replay the same derived first reader", err)
	}
	consumed, err := o.retainConsumptionCandidate(context.Background(), ack)
	if err != nil || consumed == nil {
		t.Fatal("original C must preserve the same first-reader limit", err)
	}
	d, err := o.prepareDispatchCandidate(context.Background(), consumed)
	if err != nil || d == nil || d.meter != m || o.phase != "dispatch-prepared" || m.replayReads != nil ||
		m.failed || m.closed || o.firstPReading || m.upper != upper || p.original != origin || s.stored != held || s.activeMeter.Load() != nil {
		t.Fatal("recorded dispatch replay escaped or replaced the original first-reader observer", err)
	}
	left, right := d.replayReads[0], d.replayReads[1]
	if left == nil || right == nil || left == right || !left.complete || !right.complete || left.failed || right.failed ||
		left.meter != m || right.meter != m || len(left.rows) == 0 || len(left.rows) != len(right.rows) || &left.rows[0] == &right.rows[0] {
		t.Fatal("first-reader recording was skipped or shared across independent roles")
	}
}

func TestProviderHistoryFirstOriginalCompositionMatchesIndependentWindowLiterals(t *testing.T) {
	backing := uint64(reflect.TypeFor[ObservationChecks]().Size() + reflect.TypeFor[providerHistorySemanticFold]().Size() +
		reflect.TypeFor[providerHistoryCollectionCursor]().Size() + reflect.TypeFor[FinalizedHeader]().Size())
	for _, tc := range []struct {
		groups, requests, readers, wholeReaders uint64
	}{
		{1, 1, 12, 14}, {2, 33, 14, 30}, {4, 28, 14, 58}, {9226, 162391, 14, 129166},
	} {
		got, ok := providerHistoryCollectionFirstReadCompositionUpper(tc.groups, tc.requests,
			historyFirstOriginalCompositionLimits(), historyFirstInterpretationCapture())
		first := providerHistoryFirstPReadBounds{objects: 64, raw: 1048576,
			work: candidateWorkVector{2: 4751393, 10: 6101990678681 + backing}}
		want := providerHistoryFirstPReadCompositionCost{first: first,
			initialization: candidateWorkVector{2: 9502786, 10: 12203981357362 + 2*backing},
			stage:          candidateWorkVector{2: tc.readers * 4751393, 10: tc.readers * (6101990678681 + backing)},
			whole:          candidateWorkVector{2: tc.wholeReaders * 4751393, 10: tc.wholeReaders * (6101990678681 + backing)}}
		if !ok || got != want {
			t.Fatal("independent enclosing first-reader rows differ", tc, got, want, ok)
		}
	}
}

func TestProviderHistoryFirstOriginalCompositionCoversChronologicalWindowCuts(t *testing.T) {
	limits, capture := historyFirstOriginalCompositionLimits(), historyFirstInterpretationCapture()
	for _, maximum := range []struct{ groups, requests uint64 }{{1, 32}, {2, 33}, {4, 65}} {
		upper, ok := providerHistoryCollectionFirstReadCompositionUpper(maximum.groups, maximum.requests, limits, capture)
		if !ok {
			t.Fatal("finite common original-window row")
		}
		for _, members := range [][]uint64{nil, {1}, {32}, {1, 32}, {32, 1}, {1, 32, 1}, {32, 1, 1, 31}} {
			var requests uint64
			for _, count := range members {
				requests += count
			}
			if uint64(len(members)) > maximum.groups || requests > maximum.requests {
				continue
			}
			// This is the existing chronological caller grammar, with cuts
			// AFTER advancement. No cost function supplies these counts.
			total, stages := historyCollectionEnumeratedRootStages(members)
			for coordinate, units := range upper.first.work {
				charged := upper.initialization[coordinate]
				for _, row := range stages {
					if row.initialModels*units > upper.stage[coordinate] {
						t.Fatal("an original window borrowed the following window", maximum, members, coordinate)
					}
					charged += upper.stage[coordinate] // full irrevocable row, no actual refund
				}
				if total.initialModels*units > charged || charged > upper.whole[coordinate] {
					t.Fatal("initialization/prefix counts escaped the original future", maximum, members, coordinate)
				}
			}
		}
	}
}

func TestProviderHistoryFirstOriginalCompositionRefusesInvalidAndOverflowingRows(t *testing.T) {
	limits, capture := historyFirstOriginalCompositionLimits(), historyFirstInterpretationCapture()
	for _, domain := range [][2]uint64{{0, 1}, {1, 0}, {2, 1}, {1, 33}, {349526, 349526}, {^uint64(0), 1}, {1, ^uint64(0)}} {
		if got, ok := providerHistoryCollectionFirstReadCompositionUpper(domain[0], domain[1], limits, capture); ok || got != (providerHistoryFirstPReadCompositionCost{}) {
			t.Fatal("invalid original count domain exposed a partial enclosing row", domain, got, ok)
		}
	}
	bad := limits
	bad.chunks = ^uint64(0)
	if got, ok := providerHistoryCollectionFirstReadCompositionUpper(1, 1, bad, capture); ok || got != (providerHistoryFirstPReadCompositionCost{}) {
		t.Fatal("overflowing child exposed a partial enclosing row", got, ok)
	}
	if got, ok := providerHistoryCollectionFirstReadCompositionUpper(1, 1, limits, ProviderCaptureLimits{}); ok || got != (providerHistoryFirstPReadCompositionCost{}) {
		t.Fatal("invalid capture exposed a partial enclosing row", got, ok)
	}
	// All original child rows and stage counts are finite. Only the whole
	// repeated future overflows; never cap it, omit initialization or return
	// the already-computed first/stage fields as a successful partial quote.
	limits = providerHistoryLegacyControlLimits{65536, 1 << 20, 64, 64 << 20, 256, 4, 32}
	if first, ok := providerHistoryFirstPReadUpper(limits, capture); !ok || !first.valid() {
		t.Fatal("finite large original first-reader precursor")
	}
	if got, ok := providerHistoryCollectionFirstReadCompositionUpper(349525, 11184800, limits, capture); ok || got != (providerHistoryFirstPReadCompositionCost{}) {
		t.Fatal("whole-future overflow returned partial first/stage work", got, ok)
	}
}

func TestProviderHistoryFirstOriginalUpperShortagesKeepOriginalFailedPrefix(t *testing.T) {
	for _, coordinate := range []int{2, 10} {
		t.Run(strconv.Itoa(coordinate), func(t *testing.T) {
			read, ok := providerHistoryFirstPReadUpper(historyFirstOriginalCompositionLimits(), historyCollectionInput(t).capture)
			if !ok {
				t.Fatal("normal selected source row")
			}
			// A known-insufficient prospective UNIT input, selected before H0.
			// A conservative upper minus one need not fail actual execution.
			read.work[coordinate] = 1
			s := historyCollectionAdmissionStore(t, true, &read)
			clock, held, pin := s.clock.read, s.stored, s.frontierFiles.pin
			before := providerHistoryProvisionedOriginalBytes(t, s.config)
			o, err := newProviderHistoryGroupOwnerCandidate(s)
			g := s.history.group
			if err == nil || o != nil || g == nil || !g.failed || !g.meter.closed || !s.failed || !s.history.failed ||
				g.firstPReading || g.last != nil || s.history.recovery.initial != nil || s.activeMeter.Load() != nil ||
				s.clock.read != clock+1 || s.stored != held || s.frontierFiles.pin != pin || g.meter.actual[10] == 0 ||
				!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
				t.Fatal("first-reader shortage returned permission, erased its prefix or changed originals", err)
			}
			actual, upper := g.meter.actual, g.meter.upper
			if again, err := newProviderHistoryGroupOwnerCandidate(s); err == nil || again != nil ||
				g.meter.actual != actual || g.meter.upper != upper || s.clock.read != clock+1 || s.stored != held {
				t.Fatal("failed original bridge acquired a replacement observer or allowance", err)
			}
		})
	}
}

func TestProviderHistoryFirstOriginalUpperRejectsInvalidAndOverflowingChildren(t *testing.T) {
	base, capture := historyFirstOriginalCompositionLimits(), historyFirstInterpretationCapture()
	for _, mode := range []string{"metadata-zero", "metadata-large", "object-zero", "object-large", "objects-zero", "objects-large",
		"raw-zero", "raw-large", "entries-zero", "entries-large", "attempts-zero", "attempts-large", "chunks-zero", "catalog-overflow"} {
		t.Run(mode, func(t *testing.T) {
			limits := base
			switch mode {
			case "metadata-zero":
				limits.metadata = 0
			case "metadata-large":
				limits.metadata = 1<<20 + 1
			case "object-zero":
				limits.objectBytes = 0
			case "object-large":
				limits.objectBytes = 32<<20 + 1
			case "objects-zero":
				limits.firstObjects = 0
			case "objects-large":
				limits.firstObjects = 1<<20 + 1
			case "raw-zero":
				limits.firstRaw = 0
			case "raw-large":
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
			case "catalog-overflow":
				limits.chunks = ^uint64(0)
			}
			if got, ok := providerHistoryFirstPReadUpper(limits, capture); ok || got != (providerHistoryFirstPReadBounds{}) {
				t.Fatal("invalid or overflowing child returned a partial first-reader row", mode, got)
			}
		})
	}
	if got, ok := providerHistoryFirstPReadUpper(base, ProviderCaptureLimits{}); ok || got != (providerHistoryFirstPReadBounds{}) {
		t.Fatal("invalid capture returned a partial first-reader row", got)
	}
}
