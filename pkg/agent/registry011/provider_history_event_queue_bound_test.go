package registry011

import (
	"context"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
)

func historyEventQueueDelta(t *testing.T, m *providerWorkMeter, before candidateWorkVector) candidateWorkVector {
	t.Helper()
	var out candidateWorkVector
	for i, n := range m.actual {
		if n < before[i] {
			t.Fatal("existing event/queue prefix was refunded")
		}
		out[i] = n - before[i]
	}
	return out
}

func historyEventQueueWithin(t *testing.T, actual, upper candidateWorkVector) {
	t.Helper()
	for i, n := range actual {
		if n > upper[i] {
			t.Fatal("event/queue stage exceeds its source row", i, actual, upper)
		}
	}
}

func TestProviderHistoryEventQueueUpperMatchesIndependentWholePolynomials(t *testing.T) {
	sizes := [...]uint64{
		uint64(reflect.TypeFor[replayClaimID]().Size()), uint64(reflect.TypeFor[replayClaimState]().Size()),
		uint64(reflect.TypeFor[string]().Size()), uint64(reflect.TypeFor[bool]().Size()),
		uint64(reflect.TypeFor[providerHistoryExpectedItem]().Size()), uint64(reflect.TypeFor[[]providerHistoryExpectedItem]().Size()),
		uint64(reflect.TypeFor[int]().Size()), uint64(reflect.TypeFor[replayStream]().Size()),
		uint64(reflect.TypeFor[replayState]().Size()), uint64(reflect.TypeFor[providerHistoryExpectedWindow]().Size()),
		uint64(reflect.TypeFor[replayRun]().Size()), uint64(reflect.TypeFor[FinalizedHeader]().Size()),
		uint64(reflect.TypeFor[*providerHistoryExpectedWindow]().Size()), uint64(reflect.TypeFor[replayKey]().Size()),
		uint64(reflect.TypeFor[replayService]().Size()), uint64(reflect.TypeFor[replayGrant]().Size()),
	}
	for _, n := range sizes {
		if n > 4096 {
			t.Fatal("changed target shapes require rederiving the literal oracle")
		}
	}
	for _, tc := range []struct {
		g, r, blocks, windows, versions, present, logs, evidence, claims, created uint64
		opens, slots, subjects, maximum, moves, old, queueBytes, stopBytes        uint64
		eventConstant, queueConstant, stopConstant, totalDecoded, totalConstant   uint64
		coefficients                                                              [16]uint64
	}{
		{1, 1, 1, 1, 0, 0, 1, 1024, 1, 1, 15, 15, 0, 0, 0, 0, 0, 0,
			185760, 0, 315, 600, 3016305,
			[16]uint64{60, 60, 11625, 45, 0, 0, 30, 30, 15, 15, 15, 15, 15, 5760, 480, 3840}},
		{2, 3, 5, 2, 3, 1, 2, 4096, 2, 3, 58, 116, 87, 2, 348, 43, 1900544, 237568,
			2149248, 1917726, 238786, 4640, 15282231,
			[16]uint64{464, 464, 45414, 638, 348, 87, 290, 116, 58, 58, 58, 145, 58, 22272, 1856, 14848}},
		{1, 32, 3, 3, 4096, 3, 65536, 67108864, 7, 11, 45, 2949120, 61440, 4096, 251781120, 125798400, 24159191040, 29491200,
			36521902080, 24297093120, 29492145, 117964800, 60857109975,
			[16]uint64{5898870, 5898870, 3107730, 3072990, 251781120, 61440, 61530, 90, 45, 45, 45, 45, 45, 17280, 1440, 11520}},
	} {
		q, ok := providerHistoryEventQueueUpper(providerHistoryCollectionSemanticLimits{
			Groups: tc.g, Requests: tc.r, Blocks: tc.blocks, Windows: tc.windows, Versions: tc.versions, PresentStops: tc.present,
		}, tc.logs, tc.evidence, tc.claims, tc.created)
		units := tc.totalConstant
		for i, n := range tc.coefficients {
			units += n * sizes[i]
		}
		eventTypes := tc.slots * (2*sizes[0] + 2*sizes[1] + sizes[2] + sizes[3])
		queueTypes := tc.moves*sizes[4] + tc.subjects*(2*sizes[2]+2*sizes[3]+sizes[5]+sizes[6])
		if !ok || q.windowOpens != tc.opens || q.eventSlots != tc.slots || q.subjects != tc.subjects ||
			q.maximumQueue != tc.maximum || q.queueMoves != tc.moves || q.oldSlots != tc.old ||
			q.queueBytes != tc.queueBytes || q.stopBytes != tc.stopBytes ||
			q.events != (candidateWorkVector{2: tc.totalDecoded, 10: eventTypes + tc.eventConstant}) ||
			q.queue != (candidateWorkVector{10: queueTypes + tc.queueConstant}) ||
			q.stop != (candidateWorkVector{10: tc.stopConstant}) ||
			q.total != (candidateWorkVector{2: tc.totalDecoded, 10: units}) {
			t.Fatal("independent whole event/queue/window polynomials", q, ok)
		}
	}
}

func TestProviderHistoryEventQueueConsumptionFundsBothIndexesBeforeEitherWrite(t *testing.T) {
	q, ok := providerHistoryEventQueueUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1}, 1, 1024, 1, 1)
	if !ok {
		t.Fatal("finite source row")
	}
	in, c := replayTestInput(t), replayTestCreate(t)
	typeTerms := uint64(2*reflect.TypeFor[replayClaimID]().Size() + 2*reflect.TypeFor[replayClaimState]().Size() + reflect.TypeFor[string]().Size() + reflect.TypeFor[bool]().Size())
	if q.consume != (candidateWorkVector{10: typeTerms + 359}) {
		t.Fatal("independent maximum actor42/agent64 consumption row", q.consume)
	}
	if len(c.actor) != 42 || len(c.agent) != 11 {
		t.Fatal("changed ordinary fixture requires rederiving its exact row")
	}
	want := candidateWorkVector{10: typeTerms + 3*uint64(len(c.actor)+len(c.agent)) + 41}
	historyEventQueueWithin(t, want, q.consume)
	digest := replayClaim(in.Registry, c)
	id := replayClaimID{actor: c.actor, digest: digest}
	last := uint64(reflect.TypeFor[replayClaimID]().Size()+reflect.TypeFor[replayClaimState]().Size()+reflect.TypeFor[string]().Size()+reflect.TypeFor[bool]().Size()) + 1
	for _, short := range []bool{false, true} {
		upper := want
		if short {
			upper[10]--
		}
		m := providerHistoryContentUnitMeter(upper)
		r := providerHistoryEventUnitRun(t, in, m)
		r.claims[id] = replayClaimState{block: 0}
		err := r.consumeCreateObserved(c, digest, &types.Header{Number: big.NewInt(1)}, m)
		if !short {
			if err != nil || m.actual != want || !r.claims[id].consumed || !r.created[c.agent] {
				t.Fatal("actual complete two-index consumption row", err, m.actual, want)
			}
			continue
		}
		if err == nil || !m.failed || m.actual != (candidateWorkVector{10: want[10] - last}) || r.claims[id].consumed || len(r.created) != 0 {
			t.Fatal("late funded-write shortage changed an index or lost its prefix", err, m.actual)
		}
		before := m.actual
		if r.consumeCreateObserved(c, digest, &types.Header{Number: big.NewInt(1)}, m) == nil || m.actual != before || r.claims[id].consumed || len(r.created) != 0 {
			t.Fatal("failed consumption observer resumed")
		}
	}
}

func TestProviderHistoryEventQueueMeasuresPreparationAndRepeatedQueueGrowth(t *testing.T) {
	f, _, m := historySemanticObservedFixture(t, false)
	if f.beginWindowCandidate(context.Background()) != nil {
		t.Fatal("normal empty owned window")
	}
	contract, _, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	s := uint64(reflect.TypeFor[string]().Size())
	b := uint64(reflect.TypeFor[bool]().Size())
	a := uint64(reflect.TypeFor[[]providerHistoryExpectedItem]().Size())
	i := uint64(reflect.TypeFor[providerHistoryExpectedItem]().Size())
	w := f.window
	for ordinal, original := range replayTestSteps(t)[:2] {
		c, ok := replayParseCommand(replayTestRegistry, replayTestAgent, original.History.Command, contract)
		if !ok {
			t.Fatal("ordinary create/operator-grant command")
		}
		before := m.actual
		if !w.prepareSubject(w.run, c) || historyEventQueueDelta(t, m, before) != (candidateWorkVector{10: uint64(len(c.raw)) + s + b + 10}) {
			t.Fatal("actual pending command copy and slot row")
		}
		step, ok := w.run.state.applyObserved(replayTestRegistry, c, uint64(ordinal+1), m)
		if !ok {
			t.Fatal("normal state work outside queue snapshots")
		}
		before = m.actual
		bytesBefore, historyBefore := w.budget.bytes, w.historyBytes
		q := uint64(len(w.queue))
		want := candidateWorkVector{10: 4*uint64(len(c.raw)) + uint64(len(step.History.Entry)) + (2*q+2)*i + q + s + b + a + 32}
		if !w.retainExpected(w.run, step) || historyEventQueueDelta(t, m, before) != want || len(w.queue) != ordinal+1 || cap(w.queue) != len(w.queue) {
			t.Fatal("actual queue growth must include every old item", ordinal, m.actual)
		}
		charge := len(step.Record) + len(step.Grants) + len(step.History.Entry) + len(step.History.Command)
		if w.budget.bytes-bytesBefore != charge || w.historyBytes-historyBefore != charge {
			t.Fatal("queue bytes not charged to the same original replay/history budget")
		}
	}
}

func TestProviderHistoryEventQueueLastStoreShortageRetainsOriginalChargedPrefix(t *testing.T) {
	f, _, m := historySemanticObservedFixture(t, false)
	if f.beginWindowCandidate(context.Background()) != nil {
		t.Fatal("normal window")
	}
	w, c := f.window, replayTestCreate(t)
	if !w.prepareSubject(w.run, c) {
		t.Fatal("normal pending precursor")
	}
	step, ok := w.run.state.applyObserved(replayTestRegistry, c, 1, m)
	if !ok {
		t.Fatal("normal complete state precursor")
	}
	s := uint64(reflect.TypeFor[string]().Size())
	b := uint64(reflect.TypeFor[bool]().Size())
	a := uint64(reflect.TypeFor[[]providerHistoryExpectedItem]().Size())
	i := uint64(reflect.TypeFor[providerHistoryExpectedItem]().Size())
	want := 4*uint64(len(c.raw)) + uint64(len(step.History.Entry)) + 2*i + s + b + a + 32
	before := m.actual
	bytesBefore := w.budget.bytes
	// Reduce the remaining DATA allowance on the SAME existing observer;
	// prior constructor/pending/state work is neither replaced nor refunded.
	m.upper[10] = before[10] + want - 1
	if w.retainExpected(w.run, step) || !m.failed || !w.pending || len(w.queue) != 0 || f.items != 0 {
		t.Fatal("late queue refusal published its pending item")
	}
	expected := before
	expected[10] += want - (s + b + a + 9)
	charge := len(step.Record) + len(step.Grants) + len(step.History.Entry) + len(step.History.Command)
	if m.actual != expected || w.budget.bytes != bytesBefore+charge {
		t.Fatal("late queue refusal refunded actual or original byte charge", m.actual, expected)
	}
	before = m.actual
	if w.retainExpected(w.run, step) || m.actual != before || len(w.queue) != 0 {
		t.Fatal("failed original queue observer resumed")
	}
}

func TestProviderHistoryEventQueueMeasuresActualHistoryComparisonWrapper(t *testing.T) {
	f, in, m := historySemanticObservedFixture(t, false)
	historySemanticApply(t, f, in, 0, 3)
	stop := replayTestSteps(t)[2]
	if f.compareStateCandidate(context.Background(), true, stop.Record, stop.Grants) != nil {
		t.Fatal("actual separate stop before history")
	}
	i := uint64(reflect.TypeFor[providerHistoryExpectedItem]().Size())
	n := uint64(reflect.TypeFor[int]().Size())
	for _, step := range replayTestSteps(t)[:3] {
		before := m.actual
		bytesBefore := f.window.budget.bytes
		h := step.History
		if f.compareHistoryItemCandidate(context.Background(), h.Version, h.Entry, h.Command) != nil ||
			historyEventQueueDelta(t, m, before) != (candidateWorkVector{10: i + 3*uint64(len(h.Entry)+len(h.Command)) + n + 23}) ||
			f.window.budget.bytes-bytesBefore != len(h.Entry)+len(h.Command) {
			t.Fatal("full wrapper omitted original byte charge or compared counter", h.Version)
		}
	}
}

func TestProviderHistoryEventQueueCoversActualBeginStopAndFinishCopies(t *testing.T) {
	for _, present := range []bool{false, true} {
		f, in, m := historySemanticObservedFixture(t, false)
		q, ok := providerHistoryEventQueueUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 2, Windows: 1, Versions: 1, PresentStops: 1}, 65536, 64<<20, 1, 1)
		abi, abiOK := providerHistoryABIUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1}, 1, 1024)
		if !ok || !abiOK {
			t.Fatal("finite independent begin/ABI rows")
		}
		before := m.actual
		if f.beginWindowCandidate(context.Background()) != nil {
			t.Fatal("normal begin")
		}
		upper := q.begin
		for i, n := range abi.fixed {
			upper[i] += n
		}
		historyEventQueueWithin(t, historyEventQueueDelta(t, m, before), upper)
		end := 0
		if present {
			end = 1
		}
		for i := 0; i <= end; i++ {
			if f.applyBlockCandidate(context.Background(), in.Blocks[i], journalTestHeader(t, in.Blocks[i])) != nil {
				t.Fatal("ordinary complete block precursor")
			}
		}
		var record, grants []byte
		if present {
			step := replayTestSteps(t)[0]
			record, grants = step.Record, step.Grants
		}
		// Measure the separate serializer invocation on the SAME observer.
		// The target stop's difference must be only its exact caller events.
		before = m.actual
		_, _, encoded := (replayStream{state: f.window.run.state}).stateBytesObserved(m)
		serialized := historyEventQueueDelta(t, m, before)
		if !encoded {
			t.Fatal("normal serialization precursor")
		}
		before = m.actual
		if f.compareStateCandidate(context.Background(), present, record, grants) != nil {
			t.Fatal("ordinary stop comparison")
		}
		want := serialized
		want[10] += 2*uint64(len(record)+len(grants)) + 21
		if historyEventQueueDelta(t, m, before) != want {
			t.Fatal("stop wrapper comparisons/history charge/counter copy", m.actual, want)
		}
		if present {
			h := in.History[0]
			if f.compareHistoryItemCandidate(context.Background(), h.Version, h.Entry, h.Command) != nil {
				t.Fatal("original separate history before finish")
			}
		}
		before = m.actual
		if f.finishWindowCandidate(context.Background()) != nil || historyEventQueueDelta(t, m, before) != q.finish || f.window != nil {
			t.Fatal("exact finish ownership/counter copy")
		}
	}
}

func TestProviderHistoryEventQueueCoversOrdinaryEventRouteWithDisjointPriorRows(t *testing.T) {
	f, in, m := historySemanticObservedFixture(t, false)
	if f.beginWindowCandidate(context.Background()) != nil {
		t.Fatal("actual observed expected-window precursor")
	}
	l := providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 12, Windows: 1, Versions: 11, PresentStops: 1}
	q, ok := providerHistoryEventQueueUpper(l, 65536, 64<<20, 1, 1)
	abi, abiOK := providerHistoryABIUpper(l, 65536, 64<<20)
	command, commandOK := providerHistoryCommandUpper(l, 65536, 64<<20)
	state, stateOK := providerHistoryStateApplyUpper()
	if !ok || !abiOK || !commandOK || !stateOK {
		t.Fatal("finite disjoint component precursors")
	}
	var upper candidateWorkVector
	for i := range upper {
		upper[i] = abi.window[i] + command.window[i] + state[i] + q.events[i] + q.queue[i]
	}
	// Event DATA only: normal frames come from the existing complete fixture.
	// Native block/trace execution and final collection qualification are not
	// claimed by this isolated event-stage envelope.
	for i := range in.Blocks {
		before := m.actual
		frame := &in.Blocks[i].Executions[0].Root.Calls[0]
		if f.window.run.eventDeriving(context.Background(), providerHistoryEventUnitLog(frame), &types.Header{Number: big.NewInt(int64(i)), Time: uint64(i)}, f.window) != nil {
			t.Fatal("ordinary commit/create/mutation expected-window event", i)
		}
		historyEventQueueWithin(t, historyEventQueueDelta(t, m, before), upper)
	}
	if len(f.window.queue) != 11 || f.items != 11 || !f.window.run.created[replayTestAgent] {
		t.Fatal("ordinary complete event sequence lost queue/global creation state")
	}
}

func TestProviderHistoryEventQueueUpperInvalidDomainsReturnWholeZero(t *testing.T) {
	for _, mode := range []string{"group", "request", "blocks", "windows", "versions", "stops", "logs-zero", "logs-wide", "bytes-zero", "bytes-wide", "claims-zero", "claims-wide", "created-zero", "created-wide"} {
		l := providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1}
		logs, evidence, claims, created := uint64(1), uint64(1024), uint64(1), uint64(1)
		switch mode {
		case "group":
			l.Groups = 0
		case "request":
			l.Requests = 33
		case "blocks":
			l.Blocks = 0
		case "windows":
			l.Windows = 2
		case "versions":
			l.Versions = 1<<20 + 1
		case "stops":
			l.PresentStops = 2
		case "logs-zero":
			logs = 0
		case "logs-wide":
			logs = 65537
		case "bytes-zero":
			evidence = 0
		case "bytes-wide":
			evidence = 64<<20 + 1
		case "claims-zero":
			claims = 0
		case "claims-wide":
			claims = 1<<20 + 1
		case "created-zero":
			created = 0
		case "created-wide":
			created = 1<<20 + 1
		}
		if got, ok := providerHistoryEventQueueUpper(l, logs, evidence, claims, created); ok || got != (providerHistoryEventQueueWork{}) {
			t.Fatal("invalid peripheral limits returned partial work", mode, got, ok)
		}
	}
}

func TestProviderHistoryEventQueueUpperLateLogicalOverflowReturnsWholeZero(t *testing.T) {
	const decoded uint64 = 367001862144000000
	logical, ok := new(big.Int).SetString("28479344502374400000", 10)
	if decoded == 0 || !ok || logical.Cmp(new(big.Int).SetUint64(^uint64(0))) <= 0 {
		t.Fatal("independent finite decoded/repeated event overflow prerequisite")
	}
	if got, ok := providerHistoryEventQueueUpper(providerHistoryCollectionSemanticLimits{
		Groups: 100000, Requests: 100000, Blocks: 100000, Windows: 100000, Versions: 1,
	}, 65536, 1, 1, 1); ok || got != (providerHistoryEventQueueWork{}) {
		t.Fatal("late logical row overflow leaked finite decoded/count fields", got, ok)
	}
}
