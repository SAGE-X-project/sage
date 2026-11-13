package registry011

import (
	"context"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
)

func TestProviderHistorySharedIndexClonePreservesLifetimeAndOwnership(t *testing.T) {
	create := replayTestCreate(t)
	state := replayState{}
	if _, ok := state.apply(replayTestRegistry, create, 1); !ok {
		t.Fatal("normal required-suite precursor")
	}
	expires := uint64(20)
	state.record.Keys[0].Expires = &expires
	id := replayClaimID{actor: replayTestWallet, digest: [32]byte{1}}
	source := replayStream{state: state, claims: map[replayClaimID]replayClaimState{id: {block: 7, consumed: true}}, created: map[string]bool{replayTestAgent: true}}
	before := providerHistoryCloneSemanticState(source)
	m := providerHistoryStateUnitMeter()
	got, ok := replaySharedIndexCloneObserved(source, 1, 1, m)
	if !ok || !reflect.DeepEqual(got, before) || got.state.record.Keys[0].Expires == source.state.record.Keys[0].Expires || m.actual[10] == 0 {
		t.Fatal("owned complete state and lifetime indexes", m.actual)
	}
	got.claims[id] = replayClaimState{block: 8}
	got.created[replayTestAgent] = false
	got.state.record.Keys[0].State = "revoked"
	*got.state.record.Keys[0].Expires = 0
	if !reflect.DeepEqual(source, before) {
		t.Fatal("next window changed the confirmed prefix")
	}
	upper := m.actual
	upper[10]--
	late := providerHistoryContentUnitMeter(upper)
	if got, ok := replaySharedIndexCloneObserved(source, 1, 1, late); ok || !reflect.DeepEqual(got, replayStream{}) || !late.failed || !reflect.DeepEqual(source, before) {
		t.Fatal("late clone quota failure published a partial prefix")
	}
	empty, ok := replaySharedIndexCloneObserved(replayStream{}, 1, 1, providerHistoryStateUnitMeter())
	if !ok || !reflect.DeepEqual(empty, providerHistoryCloneSemanticState(replayStream{})) {
		t.Fatal("legacy empty-window normalization")
	}
	short := providerHistoryContentUnitMeter(candidateWorkVector{})
	if got, ok := replaySharedIndexCloneObserved(source, 1, 1, short); ok || !reflect.DeepEqual(got, replayStream{}) || !short.failed || !reflect.DeepEqual(source, before) {
		t.Fatal("unfunded clone published state")
	}
	for _, limits := range [][2]int{{0, 1}, {1, 0}, {1<<20 + 1, 1}, {1, 1<<20 + 1}} {
		if _, ok := replaySharedIndexCloneObserved(source, limits[0], limits[1], providerHistoryStateUnitMeter()); ok {
			t.Fatal("invalid lifetime bound", limits)
		}
	}
	source.claims[replayClaimID{actor: replayTestOperator}] = replayClaimState{}
	if _, ok := replaySharedIndexCloneObserved(source, 1, 1, providerHistoryStateUnitMeter()); ok {
		t.Fatal("claim lifetime bound renewed")
	}
	delete(source.claims, replayClaimID{actor: replayTestOperator})
	source.created["other.unit.subject"] = true
	if _, ok := replaySharedIndexCloneObserved(source, 1, 1, providerHistoryStateUnitMeter()); ok {
		t.Fatal("identity lifetime bound renewed")
	}
}

func TestProviderHistorySharedCreateConsumptionKeepsHeightAndLifetimeRules(t *testing.T) {
	in := replayTestInput(t)
	command := replayTestCreate(t)
	digest := replayClaim(in.Registry, command)
	id := replayClaimID{actor: command.actor, digest: digest}
	probe := providerHistoryStateUnitMeter()
	probeRun := providerHistoryEventUnitRun(t, in, probe)
	probeRun.claims[id] = replayClaimState{block: 0}
	if err := probeRun.consumeCreateObserved(command, digest, &types.Header{Number: big.NewInt(1)}, probe); err != nil {
		t.Fatal("otherwise valid index-write precursor", err)
	}
	for _, mode := range []string{"normal", "last-valid-height", "missing", "consumed", "same-height", "expired", "already-created", "identity-cap", "unfunded", "late-write-quota"} {
		t.Run(mode, func(t *testing.T) {
			m := providerHistoryStateUnitMeter()
			r := providerHistoryEventUnitRun(t, in, m)
			r.claims[id] = replayClaimState{block: 0}
			h := &types.Header{Number: big.NewInt(1)}
			switch mode {
			case "last-valid-height":
				h.Number.SetUint64(256)
			case "missing":
				delete(r.claims, id)
			case "consumed":
				r.claims[id] = replayClaimState{block: 0, consumed: true}
			case "same-height":
				h.Number.SetUint64(0)
			case "expired":
				h.Number.SetUint64(257)
			case "already-created":
				r.created[command.agent] = true
			case "identity-cap":
				r.createdLimit = 1
				r.created["other.unit.subject"] = true
			case "unfunded":
				m = providerHistoryContentUnitMeter(candidateWorkVector{})
			case "late-write-quota":
				upper := probe.actual
				upper[10]--
				m = providerHistoryContentUnitMeter(upper)
			}
			before := providerHistoryCloneSemanticState(replayStream{claims: r.claims, created: r.created})
			r.budget.observer = m
			err := r.consumeCreateObserved(command, digest, h, m)
			if mode == "normal" || mode == "last-valid-height" {
				if err != nil || !r.claims[id].consumed || !r.created[command.agent] || len(r.claims) != 1 || m.actual[7] != 0 {
					t.Fatal("normal global create consumption", err)
				}
				return
			}
			if err == nil || !reflect.DeepEqual(r.claims, before.claims) || !reflect.DeepEqual(r.created, before.created) {
				t.Fatal("refused create changed either lifetime index", mode, err)
			}
		})
	}
}

func TestProviderHistoryObservedStepStoragePreservesAllNormalEvents(t *testing.T) {
	in := replayTestInput(t)
	m := providerHistoryStateUnitMeter()
	r := providerHistoryEventUnitRun(t, in, m)
	for i := range in.Blocks {
		frame := &in.Blocks[i].Executions[0].Root.Calls[0]
		if err := r.event(context.Background(), providerHistoryEventUnitLog(frame), &types.Header{Number: big.NewInt(int64(i)), Time: uint64(i)}); err != nil {
			t.Fatal("normal original event", i, err)
		}
		if len(r.steps) != cap(r.steps) {
			t.Fatal("observed result storage used implicit append growth")
		}
	}
	if !reflect.DeepEqual(r.steps, replayTestSteps(t)) || m.actual[7] != 3 {
		t.Fatal("result storage changed handwritten states or reverified historical proofs")
	}
	before := replayTestClone(t, r.steps)
	short := providerHistoryContentUnitMeter(candidateWorkVector{})
	r.budget.observer = short
	if r.appendStepObserved(HistoryContentStep{}, short) || !reflect.DeepEqual(r.steps, before) || !short.failed {
		t.Fatal("unfunded result storage replaced prior results")
	}
	probe := providerHistoryStateUnitMeter()
	probeRun := providerHistoryEventUnitRun(t, in, probe)
	probeRun.steps = replayTestClone(t, before)
	nextStep := replayTestClone(t, before[len(before)-1])
	if !probeRun.appendStepObserved(nextStep, probe) || len(probeRun.steps) != len(before)+1 {
		t.Fatal("normal append precursor for the final assignment allowance")
	}
	upper := probe.actual
	upper[10]--
	late := providerHistoryContentUnitMeter(upper)
	r.budget.observer = late
	if r.appendStepObserved(nextStep, late) || !late.failed || !reflect.DeepEqual(r.steps, before) {
		t.Fatal("late append failure changed prior result arrays or payload")
	}
	if _, ok := replayExpectedTextObserved(nil, m); ok {
		t.Fatal("empty expected bytes")
	}
	if _, ok := replayExpectedTextObserved(make([]byte, 262145), m); ok {
		t.Fatal("expected byte ceiling")
	}
}
