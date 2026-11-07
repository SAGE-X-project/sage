package registry011

import (
	"context"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
)

func TestProviderHistoryObservedCommitMatchesActualUint256BlockEncoding(t *testing.T) {
	in := replayTestInput(t)
	for _, height := range []uint64{0, 1, ^uint64(0)} {
		var previous map[replayClaimID]replayClaimState
		for _, observer := range []*providerWorkMeter{nil, providerHistoryExecutionUnitMeter()} {
			r := providerHistoryEventUnitRun(t, in, observer)
			frame := replayTestClone(t, in.Blocks[0].Executions[0].Root.Calls[0])
			var err error
			frame.Logs[0].Data, err = r.outcomes.Events["ClaimCommitted"].Inputs.NonIndexed().Pack(new(big.Int).SetUint64(height))
			if err != nil {
				t.Fatal("legacy canonical uint256 precursor", err)
			}
			if err := r.event(context.Background(), providerHistoryEventUnitLog(&frame), &types.Header{Number: new(big.Int).SetUint64(height)}); err != nil || len(r.claims) != 1 {
				t.Fatal("actual caller commit encoding", height, err)
			}
			for id, claim := range r.claims {
				if id.actor != replayTestWallet || claim.block != height || claim.consumed {
					t.Fatal("commit scope/height changed")
				}
			}
			if observer == nil {
				previous = r.claims
			} else if !reflect.DeepEqual(r.claims, previous) || observer.failed {
				t.Fatal("observed commit disagrees with existing ABI result")
			}
		}
	}
}

func TestProviderHistoryObservedCommitLastStoreShortageDoesNotPublishClaim(t *testing.T) {
	in := replayTestInput(t)
	frame := &in.Blocks[0].Executions[0].Root.Calls[0]
	header := &types.Header{Number: big.NewInt(0)}
	m := providerHistoryExecutionUnitMeter()
	r := providerHistoryEventUnitRun(t, in, m)
	if err := r.event(context.Background(), providerHistoryEventUnitLog(frame), header); err != nil || len(r.claims) != 1 {
		t.Fatal("otherwise-valid original commit precursor", err)
	}
	upper := m.actual
	upper[10]--
	short := providerHistoryContentUnitMeter(upper)
	r = providerHistoryEventUnitRun(t, in, short)
	if err := r.event(context.Background(), providerHistoryEventUnitLog(frame), header); err == nil || !short.failed || len(r.claims) != 0 {
		t.Fatal("one-short final claim store was published", err, short.actual)
	}
	last := uint64(reflect.TypeFor[replayClaimID]().Size()+reflect.TypeFor[replayClaimState]().Size()) + 1
	wantPrefix := m.actual
	wantPrefix[10] -= last
	if short.actual != wantPrefix {
		t.Fatal("refusal did not retain actual pre-store prefix", short.actual, wantPrefix)
	}
	before := short.actual
	if err := r.event(context.Background(), providerHistoryEventUnitLog(frame), header); err == nil || short.actual != before || len(r.claims) != 0 {
		t.Fatal("failed original commit observer resumed")
	}
}

func TestProviderHistoryObservedCommitNeverRefreshesTombstonesOrLifetimeLimit(t *testing.T) {
	in := replayTestInput(t)
	frame := &in.Blocks[0].Executions[0].Root.Calls[0]
	header := &types.Header{Number: big.NewInt(0)}
	r := providerHistoryEventUnitRun(t, in, providerHistoryExecutionUnitMeter())
	if err := r.event(context.Background(), providerHistoryEventUnitLog(frame), header); err != nil || len(r.claims) != 1 {
		t.Fatal(err)
	}
	for id := range r.claims {
		r.claims[id] = replayClaimState{block: 0, consumed: true}
		if err := r.event(context.Background(), providerHistoryEventUnitLog(frame), header); err == nil || r.claims[id] != (replayClaimState{block: 0, consumed: true}) {
			t.Fatal("consumed claim was refreshed")
		}
	}
	r = providerHistoryEventUnitRun(t, in, providerHistoryExecutionUnitMeter())
	r.claimLimit = 1
	other := replayClaimID{actor: replayTestOperator}
	r.claims[other] = replayClaimState{block: 9, consumed: true}
	if err := r.event(context.Background(), providerHistoryEventUnitLog(frame), header); err == nil || len(r.claims) != 1 || r.claims[other] != (replayClaimState{block: 9, consumed: true}) {
		t.Fatal("lifetime limit replaced another caller's tombstone")
	}
}

func TestProviderHistoryObservedCommitRejectsFixedInputAndEventBodyChanges(t *testing.T) {
	in := replayTestInput(t)
	for _, change := range []string{"selector", "input-size", "actor-topic", "digest-topic", "height", "high-word", "event-size", "event-type", "negative-header"} {
		t.Run(change, func(t *testing.T) {
			r := providerHistoryEventUnitRun(t, in, providerHistoryExecutionUnitMeter())
			frame := replayTestClone(t, in.Blocks[0].Executions[0].Root.Calls[0])
			header := &types.Header{Number: big.NewInt(0)}
			switch change {
			case "selector":
				frame.Input[0] ^= 1
			case "input-size":
				frame.Input = frame.Input[:35]
			case "actor-topic":
				frame.Logs[0].Topics[1][0] ^= 1
			case "digest-topic":
				frame.Logs[0].Topics[2][0] ^= 1
			case "height":
				frame.Logs[0].Data[31] = 1
			case "high-word":
				frame.Logs[0].Data[0] = 1
			case "event-size":
				frame.Logs[0].Data = append(frame.Logs[0].Data, 0)
			case "event-type":
				event := r.outcomes.Events["ClaimCommitted"]
				event.Inputs[2].Type.Size = 64
				r.outcomes.Events["ClaimCommitted"] = event
			case "negative-header":
				header.Number = big.NewInt(-1)
			}
			if err := r.event(context.Background(), providerHistoryEventUnitLog(&frame), header); err == nil || len(r.claims) != 0 {
				t.Fatal("changed commit produced a claim", change, err)
			}
		})
	}
}
