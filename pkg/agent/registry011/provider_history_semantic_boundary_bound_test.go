package registry011

import (
	"bytes"
	"context"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestProviderHistorySemanticBoundaryPartsKeepIndependentStateRows(t *testing.T) {
	l := providerHistorySemanticCloneLimits{Keys: 128, Services: 16, Grants: 128}
	p, ok := providerHistorySemanticBoundaryControlUpper(l)
	if !ok {
		t.Fatal("original typed boundary parts")
	}
	r := providerHistoryStateSerializationLimits{Keys: 128, Services: 16, Record: true}
	g := providerHistoryStateSerializationLimits{Grants: 128}
	// Independent64-bit serializer literals, adjusting only original copy
	// argument sizes for this target. Source compare/copy tails are literal.
	normal := historyAppliedSerializerLogicalUnits(r, 4295314140) + historyAppliedSerializerLogicalUnits(g, 115676421)
	envelope := historyAppliedSerializerLogicalUnits(r, 317600220444) + historyAppliedSerializerLogicalUnits(g, 3415049256)
	if p.absentState != (providerHistoryControlPair{candidateWorkVector{10: 38}, candidateWorkVector{10: 38}}) ||
		p.state.normal != (candidateWorkVector{2: 2048, 10: normal + 196610}) ||
		p.state.envelope != (candidateWorkVector{2: 2048, 10: envelope + 196610}) ||
		p.grants.normal != (candidateWorkVector{2: 2048, 10: normal + 218153}) ||
		p.grants.envelope != (candidateWorkVector{2: 2048, 10: envelope + 469544}) {
		t.Fatal("present/absent and mismatching grant result rows", p)
	}
	item := uint64(reflect.TypeFor[providerHistoryExpectedItem]().Size())
	integer := uint64(reflect.TypeFor[int]().Size())
	want := candidateWorkVector{10: item + integer + 1572887}
	if p.historyItem != (providerHistoryControlPair{want, want}) {
		t.Fatal("history copy/equality/charge events", p.historyItem, want)
	}
	// Small typed graphs retain the larger selected record comparison BEFORE
	// equality can reject it; successful record-copy width stays separate.
	small, ok := providerHistorySemanticBoundaryControlUpper(providerHistorySemanticCloneLimits{})
	smallSerializers := historyAppliedSerializerLogicalUnits(providerHistoryStateSerializationLimits{Record: true}, 3842700) +
		historyAppliedSerializerLogicalUnits(providerHistoryStateSerializationLimits{}, 4904)
	if !ok || small.state.envelope[10]-smallSerializers != 65536+1960+1 {
		t.Fatal("raw selected record width was replaced by successful typed width", small)
	}
}

func TestProviderHistorySemanticBoundaryWindowPartsFundOriginalCalls(t *testing.T) {
	f, in, m := historySemanticObservedFixture(t, false)
	p, ok := providerHistorySemanticBoundaryControlUpper(providerHistorySemanticCloneLimits{Claims: 3, Created: 5, Keys: 128, Services: 16, Grants: 128})
	if !ok {
		t.Fatal("window component")
	}
	before := m.actual
	if f.beginWindowCandidate(context.Background()) != nil {
		t.Fatal("original begin")
	}
	historyEventQueueWithin(t, historyEventQueueDelta(t, m, before), p.opening.normal)
	if f.applyBlockCandidate(context.Background(), in.Blocks[0], journalTestHeader(t, in.Blocks[0])) != nil {
		t.Fatal("ordinary initial block")
	}
	before = m.actual
	bytesBefore, historyBefore := f.window.budget.bytes, f.window.historyBytes
	if f.compareStateCandidate(context.Background(), false, nil, nil) != nil ||
		historyEventQueueDelta(t, m, before) != (candidateWorkVector{10: 21}) ||
		f.window.budget.bytes != bytesBefore || f.window.historyBytes != historyBefore {
		t.Fatal("absent stop charged invented payload bytes or omitted original events")
	}
	before = m.actual
	if f.finishWindowCandidate(context.Background()) != nil || historyEventQueueDelta(t, m, before) != p.closing.normal || m.failed {
		t.Fatal("original finish cost or observer changed")
	}
	// Original fold stop is21; the collection select-history call adds17.
	if p.absentState.normal[10] != 21+17 {
		t.Fatal("collection selection site missing or counted twice")
	}
}

func TestProviderHistorySemanticBoundaryPresentPartsFundOriginalSerializers(t *testing.T) {
	p, ok := providerHistorySemanticBoundaryControlUpper(providerHistorySemanticCloneLimits{Keys: 128, Services: 16, Grants: 128})
	if !ok {
		t.Fatal("present components")
	}
	for _, version := range []int{1, 3, 11} {
		state := providerHistoryStateUnitPrefix(t, version)
		before := providerHistoryStateUnitSnapshot(t, state)
		s := replayStream{state: state}
		record, grants := s.stateBytes()
		for _, grantPhase := range []bool{false, true} {
			part := p.state
			if grantPhase {
				part = p.grants
			}
			m := providerHistoryContentUnitMeter(part.envelope)
			stopped := record
			if grantPhase {
				stopped = providerReplayBytes(string(record), m)
			}
			wantRecord, wantGrants, encoded := s.stateBytesObserved(m)
			if !encoded || !providerReplayEqual(stopped, wantRecord, m) {
				t.Fatal("original record equality", version, grantPhase)
			}
			if grantPhase {
				if !providerReplayEqual(grants, wantGrants, m) || !m.event(providerCopyEvent, 16) ||
					!m.event(providerCopyEvent, 1) || !m.event(providerCopyEvent, 16) {
					t.Fatal("original grant equality and charge/select sites")
				}
			} else if !bytes.Equal([]byte(providerReplayString(record, m)), record) {
				t.Fatal("original stopped record copy")
			}
			historyBoundaryRPCWithin(t, m, part.normal)
		}
		if !reflect.DeepEqual(state, before) {
			t.Fatal("read-only boundary mutated parsed state")
		}
	}
}

func TestProviderHistorySemanticBoundaryEnvelopeRetainsDeferredGrantSerialization(t *testing.T) {
	// Bounded in-memory serializer UNIT DATA only. A late output-size refusal
	// must still include the second serializer; there is no network or owner.
	s := replayStream{state: replayState{version: 1, admitted: 1,
		record: replayRecord{ID: "unit", State: "active", Version: "1", Services: make([]replayService, 16)},
		grants: []replayGrant{{Operator: replayTestWallet, Scope: "update-services"}}}}
	for i := range s.state.record.Services {
		s.state.record.Services[i] = replayService{Name: "unit", Type: "unit", URI: strings.Repeat("\x01", 2048)}
	}
	p, ok := providerHistorySemanticBoundaryControlUpper(providerHistorySemanticCloneLimits{Services: 16, Grants: 1})
	if !ok {
		t.Fatal("typed late-refusal component")
	}
	m := providerHistoryContentUnitMeter(p.state.envelope)
	record, grants, encoded := s.stateBytesObserved(m)
	if encoded || record != nil || len(grants) == 0 || m.failed {
		t.Fatal("late record refusal omitted the original grant sibling")
	}
	left, right := providerHistoryContentUnitMeter(p.state.envelope), providerHistoryContentUnitMeter(p.state.envelope)
	if raw, ok := replayStateRecordBytesObserved(s.state.record, left); ok || raw != nil {
		t.Fatal("independent late record precursor")
	}
	if raw, ok := replayStateGrantsBytesObserved(s.state.grants, right); !ok || !bytes.Equal(raw, grants) {
		t.Fatal("independent grant sibling precursor")
	}
	for c := range m.actual {
		if m.actual[c] != left.actual[c]+right.actual[c] || m.actual[c] > p.state.envelope[c] {
			t.Fatal("complete entered sibling closure", c, m.actual)
		}
	}
}

func TestProviderHistorySemanticBoundaryHistoryUsesOriginalSharedBudget(t *testing.T) {
	f, in, m := historySemanticObservedFixture(t, false)
	historySemanticApply(t, f, in, 0, 1)
	step := replayTestSteps(t)[0]
	if f.compareStateCandidate(context.Background(), true, step.Record, step.Grants) != nil {
		t.Fatal("ordinary present stop")
	}
	p, ok := providerHistorySemanticBoundaryControlUpper(providerHistorySemanticCloneLimits{Keys: 128, Services: 16, Grants: 128})
	if !ok {
		t.Fatal("history component")
	}
	before, bytesBefore, historyBefore := m.actual, f.window.budget.bytes, f.window.historyBytes
	if f.compareHistoryItemCandidate(context.Background(), 1, step.History.Entry, step.History.Command) != nil {
		t.Fatal("original history comparison")
	}
	n := len(step.History.Entry) + len(step.History.Command)
	want := candidateWorkVector{10: uint64(reflect.TypeFor[providerHistoryExpectedItem]().Size()+reflect.TypeFor[int]().Size()) + 23 + 3*uint64(n)}
	got := historyEventQueueDelta(t, m, before)
	if got != want || f.window.budget.bytes-bytesBefore != n || f.window.historyBytes-historyBefore != n || f.window.compared != 1 {
		t.Fatal("shared history charge was reset or reported as decoded work", got, want)
	}
	historyEventQueueWithin(t, got, p.historyItem.normal)
}

func TestProviderHistorySemanticBoundaryModelSeparatesContentAndSharedItems(t *testing.T) {
	for _, row := range []struct {
		b, w, c, e, v, p uint64
		counts           [11]uint64
	}{
		{1, 1, 1, 0, 0, 0, [11]uint64{1, 1, 1, 0, 0, 0, 0, 0, 1}},
		{13, 4, 2, 4, 4, 4, [11]uint64{4, 4, 0, 4, 4, 0, 1, 3, 20}},
		{71, 64, 10, 64, 64, 1, [11]uint64{2, 2, 1, 1, 1, 2, 0, 1, 25}},
		{13, 4, 2, 1, 1, 0, [11]uint64{4, 4, 4, 0, 0, 0, 1, 0, 17}},
	} {
		manifest := historyBoundaryRoleManifest(t, row.b, row.w, row.c, row.e, row.v, row.p)
		beforeInput, beforeRaw, beforeCounts := manifest.input, manifest.raw, manifest.counts
		m, ok := providerHistoryCollectionSemanticBoundaryRoleModelUpper(manifest)
		if !ok || !m.valid() {
			t.Fatal("source semantic role model")
		}
		for i, tier := range m.tiers {
			if tier.count != row.counts[i] {
				t.Fatal("ancestry openings or renewed history items entered semantic tiers", i, m.tiers, row.counts)
			}
		}
		if manifest.input != beforeInput || manifest.raw != beforeRaw || manifest.counts != beforeCounts || !manifest.validSnapshot() {
			t.Fatal("cost arithmetic changed the original manifest")
		}
		parts, partsOK := providerHistorySemanticBoundaryControlUpper(providerHistorySemanticCloneLimits{Keys: 128, Services: 16, Grants: 128})
		if !partsOK {
			t.Fatal("independent state failure parts")
		}
		if row.p < manifest.counts.contentWindows && (m.delta[2] != 2048 || m.delta[10] < parts.state.envelope[10]-38) {
			// Capacity is checked after serialization/equality. A failed extra
			// present state needs its decoded work even with successful P==0.
			t.Fatal("late present-stop cap refusal lost entered serialization", row.p, m.delta)
		}
	}
}

func TestProviderHistorySemanticBoundaryRankAndFutureMatchIndependentPhaseAllocation(t *testing.T) {
	manifest := historyBoundaryRoleManifest(t, 71, 64, 10, 64, 64, 1)
	m, ok := providerHistoryCollectionSemanticBoundaryRoleModelUpper(manifest)
	j := manifest.input.journal
	p, partsOK := providerHistorySemanticBoundaryControlUpper(providerHistorySemanticCloneLimits{uint64(j.MaxClaims), uint64(j.MaxIdentities), 128, 16, 128})
	if !ok || !partsOK || m.groups != 35 {
		t.Fatal("normal source allocation")
	}
	// Independently assigned original phases: only TWO content windows, ONE
	// optional present stop and64 history items shared across THREE positions.
	base := []candidateWorkVector{p.opening.normal, p.opening.normal, p.closing.normal, p.closing.normal,
		p.absentState.normal, p.state.normal, p.grants.normal}
	for range 2 {
		v := p.historyItem.normal
		for c := range v {
			v[c] *= 32
		}
		base = append(base, v)
	}
	for len(base) < 35 {
		base = append(base, candidateWorkVector{})
	}
	var future candidateWorkVector
	for c := range future {
		values := make([]uint64, len(base))
		for i, v := range base {
			values[i] = v[c]
		}
		sort.Slice(values, func(i, j int) bool { return values[i] > values[j] })
		var prefix, prefixes uint64
		for k := uint64(0); k <= 35; k++ {
			got, sums, ok := m.rankedUpper(k, true)
			if !ok || got[c] != prefix || sums[c] != prefixes {
				t.Fatal("ranked phase allocation or weighted prior prefixes", c, k)
			}
			if k < 35 {
				prefixes += prefix
				prefix += values[k]
			}
		}
		for k := range 35 {
			u, next := uint64(0), uint64(0)
			for _, n := range values[:k] {
				u += n
			}
			next = u + values[k]
			if k > 0 {
				future[c] += 9*u + m.delta[c]
			}
			future[c] += 3*next + m.delta[c]
			if k < 34 {
				future[c] += 2*next + m.delta[c]
			}
		}
	}
	if got, ok := m.futureUpper(); !ok || got != future {
		t.Fatal("semantic role future differs from original B/closure/own-A placement", got, future)
	}
}

func TestProviderHistorySemanticBoundaryPartsAndModelRefuseInvalidData(t *testing.T) {
	for _, l := range []providerHistorySemanticCloneLimits{{Claims: 1<<20 + 1}, {Created: 1<<20 + 1}, {Keys: 129}, {Services: 17}, {Grants: 129}} {
		if got, ok := providerHistorySemanticBoundaryControlUpper(l); ok || got != (providerHistorySemanticBoundaryCost{}) {
			t.Fatal("invalid typed state exposed partial components", l)
		}
	}
	if got, ok := providerHistoryCollectionSemanticBoundaryRoleModelUpper(nil); ok || got != (providerHistoryBoundaryRoleModel{}) {
		t.Fatal("nil source supplied role data")
	}
	for _, mode := range []string{"changed-seal", "zero-claims", "negative-identities", "wide-claims"} {
		m := historyBoundaryRoleManifest(t, 13, 4, 2, 4, 4, 4)
		// Private numeric UNIT DATA, never an authority/source credential. Even
		// scalar snapshot equality cannot authorize malformed configuration.
		switch mode {
		case "changed-seal":
			m.input.versions++
		case "zero-claims":
			m.input.journal.MaxClaims = 0
			m.sealed = m.input
		case "negative-identities":
			m.input.journal.MaxIdentities = -1
			m.sealed = m.input
		case "wide-claims":
			m.input.journal.MaxClaims = 1<<20 + 1
			m.sealed = m.input
		}
		if got, ok := providerHistoryCollectionSemanticBoundaryRoleModelUpper(m); ok || got != (providerHistoryBoundaryRoleModel{}) {
			t.Fatal("inconsistent numeric source exposed partial role data", mode)
		}
	}
}
