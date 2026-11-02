package registry011

import (
	"bytes"
	"context"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestProviderHistoryABIFirstRefusalMatchesIndependentPortableRows(t *testing.T) {
	argument := uint64(reflect.TypeFor[abi.Argument]().Size())
	method := uint64(reflect.TypeFor[abi.Method]().Size())
	claim := uint64(reflect.TypeFor[replayClaimID]().Size())
	state := uint64(reflect.TypeFor[replayClaimState]().Size())
	if max(argument, method, claim, state) > 4096 {
		t.Fatal("changed source types need new portable callback rows")
	}
	got, ok := providerHistoryABIRefusalUpper()
	want := providerHistoryControlPair{
		normal:   candidateWorkVector{2: 161552, 10: 44284068714 + 8*method + 2192*argument},
		envelope: candidateWorkVector{2: 161552, 10: 66425508715 + 8*method + 2192*argument},
	}
	if !ok || got != want || got.envelope[10]-got.normal[10] != 22141440001 {
		t.Fatal("first ABI prefix or one original pending changed", got, ok)
	}
}

func TestProviderHistoryABIFirstRefusalKeepsUnknownSelectorPrefix(t *testing.T) {
	contract, _, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	// Four zero bytes are an ordinary unsupported selector, no peer/network
	// request, oversized value, alternate ABI schema or attack fixture.
	data := []byte{0, 0, 0, 0}
	for _, name := range []string{"revealClaim", "activate", "deactivate", "addKey", "revokeKey", "updateServices", "authorizeOperator", "revokeOperator"} {
		if bytes.Equal(contract.Methods[name].ID, data) {
			t.Fatal("unsupported selector precursor changed")
		}
	}
	upper, ok := providerHistoryABIRefusalUpper()
	if !ok {
		t.Fatal("numeric first ABI prefix")
	}
	m := providerHistorySemanticUnitMeter()
	got, accepted := replayDecodeCommandObserved(replayTestRegistry, replayTestWallet, data, contract, m)
	if accepted || !reflect.DeepEqual(got, replayCommand{}) || m.failed || m.clocks != 0 || m.actual[2] != 0 || m.actual[10] == 0 {
		t.Fatal("unknown selector lost its original entered prefix", m.actual)
	}
	historyEventQueueWithin(t, m.actual, upper.normal)
}

func TestProviderHistoryABIFirstRefusalKeepsShortOrdinaryMethodPrefix(t *testing.T) {
	contract, _, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	var call []byte
	actor := ""
	for _, step := range replayTestSteps(t) {
		c, ok := replayParseCommand(replayTestRegistry, replayTestAgent, step.History.Command, contract)
		if !ok {
			t.Fatal("ordinary method precursor")
		}
		if c.op == "activate" {
			call = bytes.Clone(c.calldata[:len(c.calldata)-1])
			actor = c.actor
			break
		}
	}
	if len(call) < 4 || actor == "" {
		t.Fatal("ordinary activate precursor missing")
	}
	upper, ok := providerHistoryABIRefusalUpper()
	if !ok {
		t.Fatal("numeric first ABI prefix")
	}
	m := providerHistorySemanticUnitMeter()
	got, accepted := replayDecodeCommandObserved(replayTestRegistry, actor, call, contract, m)
	if accepted || !reflect.DeepEqual(got, replayCommand{}) || m.failed || m.clocks != 0 || m.actual[10] == 0 {
		t.Fatal("short ordinary ABI method lost entered prefix", m.actual, got)
	}
	historyEventQueueWithin(t, m.actual, upper.normal)
	// This separate private DATA invocation ends on first refusal. No meter
	// reset/retry, actual Source construction, authority or live work is used.
}

func TestProviderHistoryABIFirstRefusalKeepsDuplicateCommitPrefix(t *testing.T) {
	in := replayTestInput(t)
	frame := &in.Blocks[0].Executions[0].Root.Calls[0]
	log := providerHistoryEventUnitLog(frame)
	header := &types.Header{Number: big.NewInt(0)}
	upper, ok := providerHistoryABIRefusalUpper()
	if !ok {
		t.Fatal("numeric first ABI prefix")
	}
	m := providerHistorySemanticUnitMeter()
	r := providerHistoryEventUnitRun(t, in, m)
	if err := r.event(context.Background(), log, header); err != nil || len(r.claims) != 1 || m.failed {
		t.Fatal("ordinary accepted commit precursor", err)
	}
	claims := map[replayClaimID]replayClaimState{}
	for id, state := range r.claims {
		claims[id] = state
	}
	before := m.actual
	if err := r.event(context.Background(), log, header); err == nil || m.failed || m.clocks != 0 || !reflect.DeepEqual(r.claims, claims) || len(r.created) != 0 {
		t.Fatal("ordinary duplicate claim changed state or lost its refusal", err)
	}
	delta := historyEventQueueDelta(t, m, before)
	if delta[2] == 0 || delta[10] == 0 {
		t.Fatal("duplicate lost original entered prefix")
	}
	historyEventQueueWithin(t, delta, upper.normal)
	// A normal success followed by its first rejected duplicate, isolated
	// entirely in UNIT memory. No failed retry, peer, live key or RPC is used.
}

func TestProviderHistoryABIFirstRefusalKeepsShortChangedFramingPrefix(t *testing.T) {
	in := replayTestInput(t)
	contract, outcomes, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	fields := outcomes.Events["RecordChanged"].Inputs.NonIndexed()
	frame := &in.Blocks[1].Executions[0].Root.Calls[0]
	raw := bytes.Clone(frame.Logs[0].Data)
	if len(raw) == 0 {
		t.Fatal("ordinary changed event precursor missing")
	}
	if _, err := fields.Unpack(raw); err != nil {
		t.Fatal("original changed ABI precursor", err)
	}
	c, ok := replayParseCommand(in.Registry, in.Agent, in.History[0].Command, contract)
	if !ok || c.op != "create" {
		t.Fatal("ordinary first changed event identity")
	}
	raw = raw[:len(raw)-1]
	upper, ok := providerHistoryABIRefusalUpper()
	if !ok {
		t.Fatal("numeric first ABI prefix")
	}
	m := providerHistorySemanticUnitMeter()
	if _, accepted := replayABIFieldsObserved(raw, fields, m); accepted || m.failed || m.clocks != 0 || m.actual[2] == 0 || m.actual[10] == 0 {
		t.Fatal("short changed fields lost their original entered prefix", m.actual)
	}
	historyEventQueueWithin(t, m.actual, upper.normal)
	// Original private framing DATA only, not a complete callback/count or
	// Source witness. No schema rewrite, cap/observer edit, payload or peer.
}
