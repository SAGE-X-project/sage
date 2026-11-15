package registry011

import (
	"context"
	"strconv"
	"strings"
	"testing"
)

func TestProviderHistoryDirectFilenameUpperMatchesIndependentLiterals(t *testing.T) {
	for _, tc := range []struct {
		groups, requests, models, stages, members, names, equal uint64
		initial, admission, consumption, dispatch, settlement   uint64
		prior, member, originals, comparisons, total            uint64
	}{
		{1, 1, 14, 0, 0, 18, 4, 1246, 1400, 2542, 1793, 3759, 0, 370, 2070, 916, 14096},
		{1, 32, 14, 0, 0, 4544, 128, 1246, 1400, 2542, 1793, 3759, 0, 11840, 522560, 29312, 574452},
		{2, 3, 28, 14, 28, 78, 12, 2492, 2800, 5084, 3586, 7518, 35476, 1110, 8970, 2748, 69784},
		{2, 4, 28, 14, 42, 120, 16, 2492, 2800, 5084, 3586, 7518, 45108, 1480, 13800, 3664, 85532},
		{2, 33, 28, 14, 448, 4562, 132, 2492, 2800, 5084, 3586, 7518, 324436, 12210, 524630, 30228, 912984},
		{3, 96, 42, 42, 1344, 13632, 384, 3738, 4200, 7626, 5379, 11277, 973308, 35520, 1567680, 87936, 2696664},
	} {
		got, ok := providerHistoryDirectFilenameUpper(tc.groups, tc.requests)
		want := providerHistoryDirectFilenameCost{
			initialModels: tc.models, priorStages: tc.stages, priorMembers: tc.members, currentNames: tc.names, nameComparisons: tc.equal,
			initial: candidateWorkVector{10: tc.initial}, admission: candidateWorkVector{10: tc.admission}, consumption: candidateWorkVector{10: tc.consumption},
			dispatch: candidateWorkVector{10: tc.dispatch}, settlement: candidateWorkVector{10: tc.settlement}, prior: candidateWorkVector{10: tc.prior},
			member: candidateWorkVector{10: tc.member}, originals: candidateWorkVector{10: tc.originals}, comparisons: candidateWorkVector{10: tc.comparisons},
			total: candidateWorkVector{10: tc.total},
		}
		if !ok || got != want {
			t.Fatal("independent complete logical filename row", tc.groups, tc.requests, got, want, ok)
		}
	}
}

func TestProviderHistoryDirectNamesUseGenuineOriginalMeterAndIndependentSpelling(t *testing.T) {
	o, ack := historyDispatchReplayFixture(t, 1)
	d, err := o.prepareDispatchCandidate(context.Background(), ack)
	if err != nil || d == nil {
		t.Fatal("genuine original T/C and both dispatch readers", err)
	}
	historyMemberReplayEnter(t, o, d) // Read-only DATA phase; no send or new sample.
	s, m := o.store, o.meter
	pin, clock, held, upper, window := o.pin, s.clock.read, s.stored, m.upper, o.windowStart
	if s.config.bounds.firstAdmission == nil || s.activeMeter.Load() != m {
		t.Fatal("original V4 filename observer required")
	}
	for _, index := range []uint64{0, 1, 349525, ^uint64(0)} {
		for _, kind := range []string{"highwater", "frontier", "commit", "intent", "history-pending", "HEAD.pending"} {
			decimal := strconv.FormatUint(index, 10)
			want := kind + "-" + strings.Repeat("0", 20-len(decimal)) + decimal
			before := m.actual
			got := s.name(kind, index)
			expected := before
			expected[10] += uint64(len(kind) + 22)
			if got != want || m.actual != expected {
				t.Fatal("direct indexed name bytes or individual +1 event", kind, index, got, want, m.actual, expected)
			}
		}
	}
	for _, kind := range []string{"response", "tls"} {
		want := "history-original-" + d.acknowledgment.claim.frame.after.commit[2:] + "-" +
			strconv.FormatUint(d.coordinator.ticket.table.stage, 10) + "-0-" + kind
		before := m.actual
		got := d.originalNameLocked(0, kind)
		expected := before
		expected[10] += uint64(len(want)) + 1
		if got != want || len(got) > 114 || m.actual != expected {
			t.Fatal("genuine current name was not independently copied", kind, got, want, m.actual, expected)
		}
	}
	if o.pin != pin || s.clock.read != clock || s.stored != held || m.upper != upper || o.windowStart != window ||
		m.failed || m.closed || s.activeMeter.Load() != m || o.order.sent != 0 || d.retained != 0 {
		t.Fatal("name observation moved an original window, allowance, pin or dispatch")
	}
}

func TestProviderHistoryDirectNameComparisonUsesOriginalPreparedMember(t *testing.T) {
	o, d := historyMemberDispatchFixture(t, 1)
	a, bounded, cancel, err := o.prepareMemberCandidateLocked(context.Background(), d, 0)
	if err != nil || a == nil || bounded == nil || cancel == nil {
		t.Fatal("normal member preparation without any dial", err)
	}
	defer cancel()
	r, m := o.parent.recovery, o.meter
	n := r.memberAttempt
	if n == nil || n.owner != n || n != a.recoveryAttempt || !a.used || a.connection != nil || a.files != 0 {
		t.Fatal("genuine consumed original intent required")
	}
	// This extra DATA comparison calibrates actual event weights; it does not
	// stand for another production invocation or a completed message exchange.
	before, reads, upper, window, held := m.actual, o.store.clock.read, m.upper, o.windowStart, o.store.stored
	var requestWork, nameCopy, nameEqual uint64
	for _, value := range []string{a.member.request.method, a.member.request.params, a.member.request.head, a.member.request.body, a.member.request.wire} {
		requestWork += 2*uint64(len(value)) + 1
	}
	for _, pair := range [][2]string{{"response", n.responseName}, {"tls", n.tlsName}} {
		want := "history-original-" + d.acknowledgment.claim.frame.after.commit[2:] + "-" +
			strconv.FormatUint(d.coordinator.ticket.table.stage, 10) + "-0-" + pair[0]
		if pair[1] != want || len(want) > 114 {
			t.Fatal("sealed expected name differs from original C/stage/ordinal", pair, want)
		}
		nameCopy += uint64(len(want)) + 1
		nameEqual += 2*uint64(len(want)) + 1
	}
	want := before
	want[10] += requestWork + nameCopy + nameEqual
	if !r.memberAttemptDataMatchesLocked(n, a) || m.actual != want || nameCopy > 230 || nameEqual > 458 ||
		o.store.clock.read != reads || m.upper != upper || o.windowStart != window || o.store.stored != held ||
		a.connection != nil || a.files != 0 || d.retained != 0 || o.order.next != 0 || m.failed || m.closed {
		t.Fatal("expected-name debit was hidden in request comparisons or changed original authority", m.actual, want)
	}
}

// Explicit chronological direct-reader sites, independent of triangular and
// weighted closed forms. Catalog-map construction is deliberately absent.
func historyDirectNameSequence(members []uint64) (models, stages, priorMembers, currentNames, equal uint64) {
	models = 2 // Both actual initial-model readers.
	var prior uint64
	for group, count := range members {
		for range 12 { // Planning2, T2, C3, dispatch2, S3.
			models++
			stages += uint64(group)
			priorMembers += prior
		}
		for ordinal := uint64(0); ordinal < count; ordinal++ {
			currentNames += 2 // Original member intent preparation.
			for range 2 {     // Before send and final retained-member check.
				currentNames += 2
				equal += 2
			}
			currentNames += 2 // Actual response/TLS physical writes.
			for range 2 {     // Preparation reads earlier members twice.
				for previous := uint64(0); previous < ordinal; previous++ {
					currentNames += 2
				}
			}
			for range 2 { // Independent post-write prefix readers include this member.
				for previous := uint64(0); previous <= ordinal; previous++ {
					currentNames += 2
				}
			}
		}
		for range 3 { // S coordinator and two retained-original readers.
			for range count {
				currentNames += 2
			}
		}
		prior += count
		if group+1 < len(members) {
			for range 2 { // Advance includes the just-completed S.
				models++
				stages += uint64(group + 1)
				priorMembers += prior
			}
		}
	}
	return
}

func TestProviderHistoryDirectFilenameUpperCoversChronologicalOptionalGroups(t *testing.T) {
	maximum, ok := providerHistoryDirectFilenameUpper(3, 96)
	if !ok {
		t.Fatal("original prospective upper")
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
		var requests uint64
		for _, n := range sizes[:groups] {
			requests += n
		}
		got, ok := providerHistoryDirectFilenameUpper(uint64(groups), requests)
		models, stages, members, names, equal := historyDirectNameSequence(sizes[:groups])
		if !ok || got.initialModels != models || got.priorStages != stages || got.priorMembers < members || got.currentNames < names || got.nameComparisons != equal ||
			maximum.initialModels < models || maximum.priorStages < stages || maximum.priorMembers < members || maximum.currentNames < names || maximum.nameComparisons < equal {
			t.Fatal("actual direct call sequence exceeds the prospectively selected row", sizes[:groups], got, models, stages, members, names, equal)
		}
	}
	for groups := 1; groups <= len(sizes); groups++ {
		visit(0, groups)
	}
}

func TestProviderHistoryDirectNameEqualityShortagePreservesCopiedData(t *testing.T) {
	parts := []string{"history-original-", strings.Repeat("a", 64), "-18446744073709551615-31-response"}
	want := strings.Join(parts, "")
	if len(want) != 114 {
		t.Fatal("independent maximum uint64 stage and bounded ordinal spelling")
	}
	// This inspector is DATA, not a substituted live owner or window. It can
	// copy the actual name115 but is one unit short of comparing it229.
	m := providerHistoryContentUnitMeter(candidateWorkVector{10: 343})
	got := providerReplayJoin(parts, m)
	if got != want || m.actual != (candidateWorkVector{10: 115}) || m.failed {
		t.Fatal("ordinary maximum name prefix must fit", got, m.actual)
	}
	if providerReplayTextEqual(got, want, m) || !m.failed || m.actual != (candidateWorkVector{10: 115}) ||
		providerReplayTextEqual(got, want, m) || m.actual != (candidateWorkVector{10: 115}) {
		t.Fatal("failed name comparison refunded, retried or hid earlier name work", m.actual)
	}
}

func TestProviderHistoryDirectFilenameUpperRejectsInvalidOriginalLimits(t *testing.T) {
	for _, in := range [][2]uint64{{0, 0}, {0, 1}, {^uint64(0), 1}, {(1<<20)/3 + 1, (1<<20)/3 + 1},
		{1, 0}, {2, 1}, {1, 33}, {1, ^uint64(0)}} {
		if got, ok := providerHistoryDirectFilenameUpper(in[0], in[1]); ok || got != (providerHistoryDirectFilenameCost{}) {
			t.Fatal("invalid original maxima yielded a partial filename quote", in, got, ok)
		}
	}
}

func TestProviderHistoryDirectFilenameUpperKeepsLargestOriginalDomainFinite(t *testing.T) {
	got, ok := providerHistoryDirectFilenameUpper(349525, 11184800)
	want := providerHistoryDirectFilenameCost{
		initialModels: 4893350, priorStages: 855171632700, priorMembers: 27365492246400, currentNames: 1588241600, nameComparisons: 44739200,
		initial: candidateWorkVector{10: 435508150}, admission: candidateWorkVector{10: 489335000}, consumption: candidateWorkVector{10: 888492550},
		dispatch: candidateWorkVector{10: 626698325}, settlement: candidateWorkVector{10: 1313864475}, prior: candidateWorkVector{10: 19817747416189800},
		member: candidateWorkVector{10: 4138376000}, originals: candidateWorkVector{10: 182647784000}, comparisons: candidateWorkVector{10: 10245276800},
		total: candidateWorkVector{10: 19817948201525100},
	}
	// All valid intermediates are finite in this component. This pure DATA
	// does not allocate a program-sized fixture or prove native feasibility.
	if !ok || got != want {
		t.Fatal("independent largest original filename row", got, want, ok)
	}
}
