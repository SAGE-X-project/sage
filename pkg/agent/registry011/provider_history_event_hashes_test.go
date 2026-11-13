package registry011

import (
	"context"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func providerHistoryEventUnitRun(t *testing.T, in ReplayInput, meter *providerWorkMeter) *replayRun {
	t.Helper()
	contract, outcomes, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	b := &replayBudget{limits: replayTestLimits(), observer: meter}
	return &replayRun{registry: in.Registry, agent: in.Agent, contract: contract, outcomes: outcomes,
		claims: map[replayClaimID]replayClaimState{}, created: map[string]bool{}, budget: b, history: in.History}
}

func providerHistoryEventUnitLog(f *ReplayFrame) replayLog {
	l := f.Logs[0]
	return replayLog{&types.Log{Address: common.HexToAddress(l.Address), Topics: l.Topics, Data: l.Data}, f, true}
}

func TestProviderHistoryEventHashesPreserveAllHandwrittenNormalStates(t *testing.T) {
	in := replayTestInput(t)
	want := replayTestSteps(t)
	for _, observer := range []*providerWorkMeter{nil, providerHistoryStateUnitMeter()} {
		r := providerHistoryEventUnitRun(t, in, observer)
		for i := range in.Blocks {
			frame := &in.Blocks[i].Executions[0].Root.Calls[0]
			header := &types.Header{Number: big.NewInt(int64(i)), Time: uint64(i)}
			if err := r.event(context.Background(), providerHistoryEventUnitLog(frame), header); err != nil {
				t.Fatal("actual event caller with original observer", i, err)
			}
		}
		if !reflect.DeepEqual(r.steps, want) || len(r.claims) != 1 || len(r.created) != 1 {
			t.Fatal("original event states or caller-scoped claim changed")
		}
		for _, claim := range r.claims {
			if !claim.consumed {
				t.Fatal("actual create did not consume its earlier claim")
			}
		}
		if observer != nil && observer.failed {
			t.Fatal("original observer failed on normal source fixture")
		}
	}
}

func TestProviderHistoryEventHashesRejectChangedTopicsDigestsAndClaimScope(t *testing.T) {
	in := replayTestInput(t)
	for _, change := range []string{"did-topic", "actor-topic", "command-digest", "record-digest", "grants-digest", "claim-actor", "claim-digest"} {
		t.Run(change, func(t *testing.T) {
			r := providerHistoryEventUnitRun(t, in, providerHistoryStateUnitMeter())
			commit := &in.Blocks[0].Executions[0].Root.Calls[0]
			if err := r.event(context.Background(), providerHistoryEventUnitLog(commit), &types.Header{Number: big.NewInt(0)}); err != nil || len(r.claims) != 1 {
				t.Fatal("otherwise valid prior claim precursor", err)
			}
			reveal := replayTestClone(t, in.Blocks[1].Executions[0].Root.Calls[0])
			switch change {
			case "did-topic":
				reveal.Logs[0].Topics[1][0] ^= 1
			case "actor-topic":
				reveal.Logs[0].Topics[2][0] ^= 1
			case "command-digest", "record-digest", "grants-digest":
				fields := r.outcomes.Events["RecordChanged"].Inputs.NonIndexed()
				values, err := fields.Unpack(reveal.Logs[0].Data)
				if err != nil {
					t.Fatal(err)
				}
				index := map[string]int{"command-digest": 4, "record-digest": 5, "grants-digest": 6}[change]
				digest := values[index].([32]byte)
				digest[0] ^= 1
				values[index] = digest
				reveal.Logs[0].Data, err = fields.Pack(values...)
				if err != nil {
					t.Fatal(err)
				}
			case "claim-actor", "claim-digest":
				for id, claim := range r.claims {
					delete(r.claims, id)
					if change == "claim-actor" {
						id.actor = replayTestOperator
					} else {
						id.digest[0] ^= 1
					}
					r.claims[id] = claim
					break
				}
			}
			if err := r.event(context.Background(), providerHistoryEventUnitLog(&reveal), &types.Header{Number: big.NewInt(1), Time: 1}); err == nil || len(r.steps) != 0 {
				t.Fatal("changed event published a successful history step", change, err)
			}
		})
	}
}
