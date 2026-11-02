package registry011

import (
	"context"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestProviderHistoryCatalogNamespaceUpperMatchesIndependentLiterals(t *testing.T) {
	for _, tc := range []struct {
		input                                      providerHistoryCatalogNamespaceInput
		catalogs, stages, members, pairs           uint64
		after, names, pending, initialOne, copyOne uint64
		initial, prior, extensions, preparation    uint64
		copies, current, pendingWork, total        uint64
	}{
		{providerHistoryCatalogNamespaceInput{1, 1, 0, 1, 64}, 30, 0, 0, 27, 10, 37, 5, 958, 32768, 28740, 0, 20447, 4070, 327680, 4255, 170, 385362},
		{providerHistoryCatalogNamespaceInput{1, 32, 0, 1, 64}, 526, 0, 0, 523, 10, 17056, 5, 958, 32768, 503908, 0, 388479, 4070, 327680, 1961440, 170, 3185747},
		{providerHistoryCatalogNamespaceInput{2, 3, 2, 4, 1024}, 78, 48, 64, 70, 20, 207, 10, 1732, 524288, 135096, 99904, 52766, 8140, 10485760, 23805, 340, 10805811},
		{providerHistoryCatalogNamespaceInput{2, 4, 64, 64, 1024}, 94, 64, 112, 86, 20, 340, 10, 20116, 524288, 1890904, 151552, 64638, 8140, 10485760, 39100, 340, 12640434},
		{providerHistoryCatalogNamespaceInput{3, 96, 64, 64, 1 << 20}, 1582, 1584, 50688, 1569, 30, 51168, 15, 20116, 536870912, 31823512, 36717120, 1165437, 12210, 16106127360, 5884320, 510, 16181730469},
	} {
		got, ok := providerHistoryCatalogNamespaceUpper(tc.input)
		want := providerHistoryCatalogNamespaceCost{
			catalogs: tc.catalogs, prefixStages: tc.stages, prefixMembers: tc.members, currentPairs: tc.pairs,
			afterCopies: tc.after, currentNames: tc.names, pendingNames: tc.pending, initialPerCatalog: tc.initialOne, copyPerCatalog: tc.copyOne,
			initial: candidateWorkVector{10: tc.initial}, prior: candidateWorkVector{10: tc.prior}, extensions: candidateWorkVector{10: tc.extensions},
			preparation: candidateWorkVector{10: tc.preparation}, copies: candidateWorkVector{10: tc.copies}, current: candidateWorkVector{10: tc.current},
			pending: candidateWorkVector{10: tc.pendingWork}, total: candidateWorkVector{10: tc.total},
		}
		if !ok || got != want {
			t.Fatal("independent catalog namespace literals", tc.input, got, want, ok)
		}
	}
}

// Independent output spelling, without calling providerCustodyName or any
// observed name builder. These test strings create no file or write claim.
func historyNamespaceExpectedIndexed(kind string, index uint64) string {
	n := strconv.FormatUint(index, 10)
	return kind + "-" + strings.Repeat("0", 20-len(n)) + n
}

func historyNamespaceExpectedExtension(before [2]map[string]bool, c *providerHistoryWriteClaim) [2]map[string]bool {
	var want [2]map[string]bool
	for root := range want {
		want[root] = make(map[string]bool)
		for name := range before[root] {
			want[root][name] = true
		}
	}
	for i := range c.inventory.count {
		ref := c.inventory.entries[i]
		want[1]["history-object-"+ref.kind+"-"+ref.digest[2:]] = true
	}
	want[1]["history-inventory-"+c.inventory.digest[2:]] = true
	want[1][historyNamespaceExpectedIndexed("history-pending", c.frame.after.sequence)] = true
	for _, kind := range []string{"intent", "commit", "frontier"} {
		want[0][historyNamespaceExpectedIndexed(kind, c.frame.after.sequence)] = true
	}
	want[0][historyNamespaceExpectedIndexed("highwater", c.frame.after.ordinal)] = true
	return want
}

func TestProviderHistoryCatalogNamespaceCopiesAndExtendsGenuineOriginalClaims(t *testing.T) {
	o, ack := historyDispatchReplayFixture(t, 1)
	d, err := o.prepareDispatchCandidate(context.Background(), ack)
	if err != nil || d == nil {
		t.Fatal("genuine T/C and both original dispatch replays", err)
	}
	historyMemberReplayEnter(t, o, d)
	s, m := o.store, o.meter
	before, ok := o.initialHistoryCatalogLocked()
	if !ok {
		t.Fatal("independently reconstructed original initial allowed set")
	}
	pin, clock, held, upper := o.pin, s.clock.read, s.stored, m.upper
	for i, claim := range [2]*providerHistoryWriteClaim{o.admission.claim, o.last.claim} {
		nameWork := [2]uint64{413, 329}[i]
		want := historyNamespaceExpectedExtension(before, claim)
		var copies uint64
		for _, names := range before {
			for name := range names {
				copies += uint64(len(name)) + 1
			}
		}
		if copies == 0 || copies > 512*o.parent.bounds.catalogEntries {
			t.Fatal("actual bounded original maps required", copies)
		}
		start := m.actual
		after, ok := o.afterHistoryCatalogLocked(before, claim)
		expected := start
		expected[10] += copies + nameWork
		if !ok || m.actual != expected || !reflect.DeepEqual(after, want) {
			t.Fatal("real map copying or original filename debit", i, ok, m.actual, expected)
		}
		// An extension mutates its own DATA map but never aliases either the
		// cloned result above or a cached claim catalog as its input.
		var extension [2]map[string]bool
		for root := range extension {
			extension[root] = make(map[string]bool)
			for name := range before[root] {
				extension[root][name] = true
			}
		}
		start = m.actual
		expected = start
		expected[10] += nameWork
		if !o.extendHistoryCatalogLocked(extension, claim) || m.actual != expected || !reflect.DeepEqual(extension, want) ||
			o.pin != pin || s.clock.read != clock || s.stored != held || m.upper != upper || m.failed || m.closed ||
			s.frontierFiles.historyPending != nil || d.retained != 0 || d.order.sent != 0 {
			t.Fatal("original T/C extension replaced authority or escaped its exact names", i)
		}
		before = after
	}
}

func TestProviderHistoryCatalogNamespaceArchiveNamesChargeCopyAndComparison(t *testing.T) {
	o := historyMultistageOwner(t, 3)
	for stage := uint64(1); stage <= 2; stage++ {
		out := historyMultistageSettle(t, o, stage)
		if err := o.advanceStageCandidate(context.Background(), out.ack); err != nil {
			t.Fatal("genuine completed archive", stage, err)
		}
	}
	d := historyMultistageDispatch(t, o, 3)
	historyMemberReplayEnter(t, o, d)
	m, clock, archives := o.meter, o.store.clock.read, 0
	for _, row := range d.memberReplaySeal.ancestors {
		c := row.value.checkpoint.owner
		if c.memberArchive == nil {
			continue
		}
		archives++
		for _, kind := range []string{"response", "tls"} {
			want := "history-original-" + c.previous.pin.commit[2:] + "-" + strconv.FormatUint(c.frame.stage, 10) + "-0-" + kind
			start := m.actual
			got := o.archivedMemberNameLocked(c, 0, kind)
			expected := start
			expected[10] += 3*uint64(len(want)) + 2
			if got != want || len(got) > 114 || m.actual != expected || m.actual[10]-start[10] > 344 || m.failed ||
				o.store.clock.read != clock || d.retained != 0 || d.order.sent != 0 {
				t.Fatal("original archive name copy and equality were not distinct", got, want, m.actual, expected)
			}
		}
	}
	if archives != 2 {
		t.Fatal("both actual earlier archives required", archives)
	}
}

// Chronological successful call classes and retained-file counts, independent
// of the quote's triangular/weighted/square closed forms.
func historyNamespaceSequence(members []uint64) (stages, priorMembers, pairs, copies, names, pending uint64) {
	var prior uint64
	for group, count := range members {
		catalogs := historyCollectionCatalogReadSequence([]uint64{count}, 1, 1).calls
		stages += catalogs * uint64(group)
		priorMembers += catalogs * prior
		pairs += 2   // Both dispatch catalog endpoints extend T AND C.
		copies += 2  // T and C preparation each clone both maps.
		pending += 2 // T and C each have one before-rename exact check.
		for ordinal := uint64(0); ordinal < count; ordinal++ {
			for range 6 { // Entry, four replay-role boundaries, after models.
				pairs++
				names += 2 * ordinal
			}
			if ordinal != 0 {
				for range 2 { // The two earlier-member prefix reads.
					pairs++
					names += 2 * ordinal
				}
			}
			pairs += 2
			names += (2*ordinal + 1) + (2*ordinal + 2) // After each spool write.
			for range 4 {                              // Two post-write readers: wrapper plus selected prefix.
				pairs++
				names += 2*ordinal + 2
			}
			for range 2 { // Retained-original and outer completion catalogs.
				pairs++
				names += 2*ordinal + 2
			}
		}
		// S has twelve prefix catalogs, one being a discarded initial prefix.
		// The other eleven extend current T/C and include both member files.
		for range 11 {
			pairs++
			names += 2 * count
		}
		copies++ // S preparation.
		for range 2 {
			copies += 2 // Reader's spool catalog and final exact catalog.
		}
		copies += 2 // Generic pre/post rename.
		copies++    // Outer S completion.
		pending += 3
		prior += count
		if group+1 < len(members) {
			stages += 2 * uint64(group+1)
			priorMembers += 2 * prior
		}
	}
	return
}

func TestProviderHistoryCatalogNamespaceUpperCoversSkewedAndOptionalGroups(t *testing.T) {
	maximum, ok := providerHistoryCatalogNamespaceUpper(providerHistoryCatalogNamespaceInput{3, 96, 64, 64, 1024})
	if !ok {
		t.Fatal("prospective original maximum")
	}
	var sizes [3]uint64
	var visit func(int, int)
	visit = func(index, groups int) {
		if index < groups {
			for _, n := range []uint64{1, 2, 31, 32} {
				sizes[index] = n
				visit(index+1, groups)
			}
			return
		}
		members := sizes[:groups]
		var requests uint64
		for _, n := range members {
			requests += n
		}
		got, ok := providerHistoryCatalogNamespaceUpper(providerHistoryCatalogNamespaceInput{uint64(groups), requests, 64, 64, 1024})
		stages, prior, pairs, copies, names, pending := historyNamespaceSequence(members)
		if !ok || got.prefixStages < stages || got.prefixMembers < prior || got.currentPairs != pairs ||
			got.afterCopies != copies || got.currentNames < names || got.pendingNames != pending ||
			maximum.prefixStages < stages || maximum.prefixMembers < prior || maximum.currentPairs < pairs ||
			maximum.afterCopies < copies || maximum.currentNames < names || maximum.pendingNames < pending {
			t.Fatal("actual chronological namespaces exceed prospective row", members, got, stages, prior, pairs, copies, names, pending)
		}
	}
	for groups := 1; groups <= len(sizes); groups++ {
		visit(0, groups)
	}
}

func TestProviderHistoryCatalogNamespaceNameShortageKeepsEnteredDataPrefix(t *testing.T) {
	digest := admissionTestDigest("a")
	positive := providerReplayUnitMeter()
	for _, kind := range []string{"requests", "ticket", "consumption", "results", "settlement"} {
		got := providerHistoryStoredName(providerHistoryStoredReference{kind, digest, 1}, positive)
		if got != "history-object-"+kind+"-"+digest[2:] {
			t.Fatal("independent closed stored-name spelling", kind, got)
		}
	}
	if positive.actual != (candidateWorkVector{10: 447}) || positive.failed {
		t.Fatal("all five original stored names need their own copy event", positive.actual)
	}
	// A separate DATA inspector can copy the two original T names89+87,
	// but lacks the last byte of the following inventory-name event83.
	m := providerHistoryContentUnitMeter(candidateWorkVector{10: 258})
	for _, kind := range []string{"requests", "ticket"} {
		if providerHistoryStoredName(providerHistoryStoredReference{kind, digest, 1}, m) == "" {
			t.Fatal("genuine fixed name prefix must fit")
		}
	}
	if m.actual != (candidateWorkVector{10: 176}) || providerHistoryInventoryName(digest, m) != "" || !m.failed ||
		m.actual != (candidateWorkVector{10: 176}) || providerHistoryInventoryName(digest, m) != "" || m.actual != (candidateWorkVector{10: 176}) {
		t.Fatal("one-short name event rewound or retried entered DATA work", m.actual)
	}
}

func TestProviderHistoryCatalogNamespaceUpperRejectsInvalidSourceLimits(t *testing.T) {
	for _, in := range []providerHistoryCatalogNamespaceInput{
		{0, 1, 0, 1, 1}, {^uint64(0), 1, 0, 1, 1}, {(1<<20)/3 + 1, (1<<20)/3 + 1, 0, 1, 1},
		{1, 0, 0, 1, 1}, {2, 1, 0, 1, 1}, {1, 33, 0, 1, 1}, {1, ^uint64(0), 0, 1, 1},
		{1, 1, 65, 1, 1}, {1, 1, ^uint64(0), 1, 1}, {1, 1, 0, 0, 1}, {1, 1, 0, 65, 1},
		{1, 1, 0, 1, 0}, {1, 1, 0, 1, (1 << 20) + 1},
	} {
		if got, ok := providerHistoryCatalogNamespaceUpper(in); ok || got != (providerHistoryCatalogNamespaceCost{}) {
			t.Fatal("invalid original limits returned a partial quote", in, got, ok)
		}
	}
}

func TestProviderHistoryCatalogNamespaceUpperKeepsLargestSourceDomainFinite(t *testing.T) {
	// No input-sized slice or live source is created for this numeric DATA.
	// Unlike other components, all valid combinations in this bounded row
	// fit uint64; a fabricated valid late-overflow test would be misleading.
	got, ok := providerHistoryCatalogNamespaceUpper(providerHistoryCatalogNamespaceInput{349525, 11184800, 64, 64, 1 << 20})
	if !ok || got.catalogs != 184549198 || got.prefixStages != 32252187290400 || got.prefixMembers != 1032069993292800 ||
		got.currentPairs != 182801575 || got.afterCopies != 3495250 || got.currentNames != 5961498400 || got.pendingNames != 1747625 ||
		got.initialPerCatalog != 20116 || got.copyPerCatalog != 536870912 ||
		got.initial != (candidateWorkVector{10: 3712391666968}) || got.prior != (candidateWorkVector{10: 747605701391472000}) ||
		got.extensions != (candidateWorkVector{10: 135783122475}) || got.preparation != (candidateWorkVector{10: 1422566750}) ||
		got.copies != (candidateWorkVector{10: 1876498055168000}) || got.current != (candidateWorkVector{10: 685572316000}) ||
		got.pending != (candidateWorkVector{10: 59419250}) || got.total != (candidateWorkVector{10: 749486734675731443}) {
		t.Fatal("source-valid maximum namespace quote", got, ok)
	}
}
