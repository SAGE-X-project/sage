package registry011

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestProviderHistorySemanticCloneBoundsMatchHandwrittenEvents(t *testing.T) {
	for _, tc := range []struct {
		limits  providerHistorySemanticCloneLimits
		units64 uint64
	}{
		{providerHistorySemanticCloneLimits{}, 984},
		{providerHistorySemanticCloneLimits{1, 1, 1, 1, 1}, 7297},
		{providerHistorySemanticCloneLimits{3, 5, 2, 1, 2}, 10199},
		{providerHistorySemanticCloneLimits{1 << 20, 1 << 20, 128, 16, 128}, 545510216},
	} {
		l := tc.limits
		// Independent literals use ordinary64-bit DATA layout only. Adjust
		// each copied type's bytes, preserving handwritten event constants.
		copies := [...]struct{ count, bytes64, target uint64 }{
			{1, 216, uint64(reflect.TypeFor[replayStream]().Size())},
			{1, 152, uint64(reflect.TypeFor[replayState]().Size())},
			{3 * l.Keys, 104, uint64(reflect.TypeFor[replayKey]().Size())},
			{2 * l.Services, 48, uint64(reflect.TypeFor[replayService]().Size())},
			{2 * l.Grants, 32, uint64(reflect.TypeFor[replayGrant]().Size())},
			{2 * l.Claims, 48, uint64(reflect.TypeFor[replayClaimID]().Size())},
			{2 * l.Claims, 16, uint64(reflect.TypeFor[replayClaimState]().Size())},
			{4 + 6*l.Keys + 2*l.Created, 16, uint64(reflect.TypeFor[string]().Size())},
			{2 * l.Created, 1, uint64(reflect.TypeFor[bool]().Size())},
		}
		want := tc.units64
		for _, copy := range copies {
			want = want - copy.count*copy.bytes64 + copy.count*copy.target
		}
		got, ok := providerHistorySemanticCloneUpper(l)
		if !ok || got != (candidateWorkVector{10: want}) {
			t.Fatal("clone upper differs from handwritten original-event totals", l, got, want, ok)
		}
	}
}

// Full-width ordinary DATA for the clone primitive, not a validated record,
// credential, registry state, authenticated Source or live fold. Counts here
// are finite small fixtures; the maximum-cardinality oracle allocates no maps.
func historySemanticCloneCostData(l providerHistorySemanticCloneLimits) replayStream {
	s := replayStream{claims: map[replayClaimID]replayClaimState{}, created: map[string]bool{}}
	s.state.record = replayRecord{ID: strings.Repeat("a", 189), Controller: strings.Repeat("a", 42), State: strings.Repeat("a", 16), Version: strings.Repeat("9", 20)}
	s.state.record.Keys = make([]replayKey, int(l.Keys))
	for i := range s.state.record.Keys {
		expires := uint64(1<<53 - 1)
		s.state.record.Keys[i] = replayKey{Name: strings.Repeat("a", 32), Alg: strings.Repeat("a", 24), Key: strings.Repeat("a", 43),
			Proof: replayProof{Signer: strings.Repeat("a", 189), Value: strings.Repeat("a", 86)}, State: strings.Repeat("a", 16), Expires: &expires}
	}
	s.state.record.Services = make([]replayService, int(l.Services))
	for i := range s.state.record.Services {
		s.state.record.Services[i] = replayService{Name: strings.Repeat("a", 32), Type: strings.Repeat("a", 64), URI: strings.Repeat("a", 2048)}
	}
	s.state.grants = make([]replayGrant, int(l.Grants))
	for i := range s.state.grants {
		s.state.grants[i] = replayGrant{Operator: strings.Repeat("a", 42), Scope: strings.Repeat("a", 16)}
	}
	for i := uint64(0); i < l.Claims; i++ {
		s.claims[replayClaimID{actor: strings.Repeat("a", 42), digest: [32]byte{byte(i), byte(i >> 8)}}] = replayClaimState{block: i, consumed: i%2 == 0}
	}
	for i := uint64(0); i < l.Created; i++ {
		s.created[fmt.Sprintf("%064x", i)] = i%2 == 0
	}
	return s
}

func TestProviderHistorySemanticCloneBoundsFundOriginalFullWidthEvents(t *testing.T) {
	for _, l := range []providerHistorySemanticCloneLimits{{}, {1, 1, 1, 1, 1}, {3, 5, 2, 1, 2}, {257, 257, 128, 16, 128}} {
		s := historySemanticCloneCostData(l)
		want := providerHistoryCloneSemanticState(s)
		upper, ok := providerHistorySemanticCloneUpper(l)
		if !ok {
			t.Fatal("prospective clone row")
		}
		meter := providerHistoryContentUnitMeter(upper)
		got, ok := replaySharedIndexCloneObserved(s, max(1, int(l.Claims)), max(1, int(l.Created)), meter)
		if !ok || meter.failed || meter.actual != upper || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(s, want) {
			t.Fatal("full-width original clone differs from event upper or preserved state", l, meter.actual, upper, ok)
		}
	}
}

func TestProviderHistorySemanticCloneBoundsRefuseLateShortageWithoutPartialState(t *testing.T) {
	l := providerHistorySemanticCloneLimits{3, 5, 2, 1, 2}
	s := historySemanticCloneCostData(l)
	before := providerHistoryCloneSemanticState(s)
	upper, ok := providerHistorySemanticCloneUpper(l)
	baseline := providerHistoryContentUnitMeter(upper)
	if got, normal := replaySharedIndexCloneObserved(s, 3, 5, baseline); !ok || !normal || baseline.failed || baseline.actual != upper || !reflect.DeepEqual(got, before) {
		t.Fatal("exact original-event successful precursor")
	}
	short := upper
	short[10]--
	meter := providerHistoryContentUnitMeter(short)
	got, ok := replaySharedIndexCloneObserved(s, 3, 5, meter)
	if ok || !meter.failed || !reflect.DeepEqual(got, replayStream{}) || !reflect.DeepEqual(s, before) {
		t.Fatal("late copy shortage published a partial clone or changed its source")
	}
}

func TestProviderHistorySemanticCloneBoundsCoverOptionalStorageAndExpiries(t *testing.T) {
	upper, ok := providerHistorySemanticCloneUpper(providerHistorySemanticCloneLimits{3, 5, 2, 1, 2})
	if !ok {
		t.Fatal("prospective optional bound")
	}
	for _, mode := range []string{"nil", "empty", "no-expiry", "short-text"} {
		s := replayStream{}
		if mode != "nil" {
			s = historySemanticCloneCostData(providerHistorySemanticCloneLimits{})
		}
		if mode == "no-expiry" || mode == "short-text" {
			s = historySemanticCloneCostData(providerHistorySemanticCloneLimits{1, 1, 1, 1, 1})
			s.state.record.Keys[0].Expires = nil
		}
		if mode == "short-text" {
			s.state.record.ID = ""
			s.state.record.Keys[0].Proof.Value = ""
			s.state.record.Services[0].URI = ""
			s.state.grants[0].Scope = ""
		}
		before := providerHistoryCloneSemanticState(s)
		meter := providerHistoryContentUnitMeter(upper)
		got, success := replaySharedIndexCloneObserved(s, 3, 5, meter)
		if !success || meter.failed || !reflect.DeepEqual(got, before) || meter.actual[10] > upper[10] || meter.actual != (candidateWorkVector{10: meter.actual[10]}) {
			t.Fatal("optional clone exceeded prospective allowance or lost empty normalization", mode, meter.actual)
		}
	}
}

func TestProviderHistorySemanticCloneBoundsRefuseInvalidCardinalities(t *testing.T) {
	for _, l := range []providerHistorySemanticCloneLimits{{Claims: 1<<20 + 1}, {Created: 1<<20 + 1}, {Keys: 129}, {Services: 17}, {Grants: 129},
		{Claims: ^uint64(0)}, {Created: ^uint64(0)}, {Keys: ^uint64(0)}, {Services: ^uint64(0)}, {Grants: ^uint64(0)}} {
		if got, ok := providerHistorySemanticCloneUpper(l); ok || got != (candidateWorkVector{}) {
			t.Fatal("invalid prospective cardinality returned a partial upper", l, got, ok)
		}
	}
}
