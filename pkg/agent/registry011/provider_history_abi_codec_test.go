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

func TestProviderHistoryABICanonicalCodecUsesOriginalExactAndShortQuotas(t *testing.T) {
	typ, err := abi.NewType("uint64", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	fields := abi.Arguments{{Name: "version", Type: typ}}
	raw := providerHistoryABIUnitWord(11)
	m := providerHistoryContentUnitMeter(candidateWorkVector{2: 32, 10: 2164})
	if _, ok := replayABIFieldsObserved(raw, fields, m); !ok {
		t.Fatal("canonical fixed scalar precursor")
	}
	values, err := replayABICanonicalObserved(fields, raw, m)
	// Framing49 + decode1025 + actual encode1025 + equality65.
	if err != nil || len(values) != 1 || values[0] != uint64(11) || m.actual != (candidateWorkVector{2: 32, 10: 2164}) {
		t.Fatal("exact concrete codec/equality arithmetic", values, err, m.actual)
	}
	for _, upper := range []candidateWorkVector{{}, {2: 31, 10: 2164}, {2: 32, 10: 1100}, {2: 32, 10: 2163}} {
		short := providerHistoryContentUnitMeter(upper)
		framed := false
		if _, ok := replayABIFieldsObserved(raw, fields, short); ok {
			framed = true
			if got, err := replayABICanonicalObserved(fields, raw, short); err == nil || got != nil {
				t.Fatal("short canonical decoder returned partial values", got, err)
			}
		}
		if !short.failed || upper[10] != 0 && !framed {
			t.Fatal("quota did not reach its intended operation", framed, short.actual)
		}
		before := short.actual
		if got, err := replayABICanonicalObserved(fields, raw, short); err == nil || got != nil || short.actual != before {
			t.Fatal("failed original canonical decoder resumed")
		}
	}
	values, err = replayABICanonicalObserved(fields, raw, nil)
	if err != nil || len(values) != 1 || values[0] != uint64(11) {
		t.Fatal("legacy canonical codec")
	}
	if got, err := replayABICanonicalObserved(fields, nil, providerHistoryExecutionUnitMeter()); err == nil || got != nil {
		t.Fatal("incomplete private decoder input")
	}
	if got, err := replayABICanonicalObserved(make(abi.Arguments, 129), nil, providerHistoryExecutionUnitMeter()); err == nil || got != nil {
		t.Fatal("private argument cap")
	}
}

func TestProviderHistoryABISelectorAndCommandRoundTripsUseSameOriginalObserver(t *testing.T) {
	contract, _, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, step := range replayTestSteps(t) {
		c, ok := replayParseCommand(replayTestRegistry, replayTestAgent, step.History.Command, contract)
		if !ok {
			t.Fatal("normal command precursor")
		}
		m := providerHistoryExecutionUnitMeter()
		got, ok := replayDecodeCommandObserved(replayTestRegistry, c.actor, c.calldata, contract, m)
		if !ok || !bytes.Equal(got.raw, c.raw) || !bytes.Equal(got.calldata, c.calldata) || m.failed || m.actual[2] < uint64(len(c.calldata)-4) || m.actual[10] == 0 {
			t.Fatal("original selector/framing/codec round trip", c.op, m.actual)
		}
		seen[c.op] = true
		short := providerHistoryContentUnitMeter(candidateWorkVector{})
		if got, ok := replayDecodeCommandObserved(replayTestRegistry, c.actor, c.calldata, contract, short); ok || !reflect.DeepEqual(got, replayCommand{}) || !short.failed {
			t.Fatal("unfunded command returned a partial value")
		}
		bad := append(bytes.Clone(c.calldata), make([]byte, 32)...)
		if _, ok := replayDecodeCommandObserved(replayTestRegistry, c.actor, bad, contract, providerHistoryExecutionUnitMeter()); ok {
			t.Fatal("noncanonical command returned a value")
		}
	}
	if len(seen) != 8 {
		t.Fatal("all adopted operations must be exercised")
	}
	for _, raw := range [][]byte{nil, {0, 0, 0}, {0, 0, 0, 0}, contract.Methods["commitClaim"].ID, contract.Methods["record"].ID} {
		if method, ok := replayABIMethodObserved(contract, raw, providerHistoryExecutionUnitMeter()); ok || method != nil {
			t.Fatal("non-mutation or unknown selector")
		}
	}
	method := contract.Methods["activate"]
	if got, ok := replayABIMethodObserved(contract, method.ID, nil); !ok || got.Name != "activate" {
		t.Fatal("legacy selector lookup")
	}
	if got, ok := replayABIMethodObserved(contract, []byte{0, 0, 0, 0}, nil); ok || got != nil {
		t.Fatal("legacy unknown selector")
	}
	if got, ok := replayABIMethodObserved(abi.ABI{}, method.ID, providerHistoryExecutionUnitMeter()); ok || got != nil {
		t.Fatal("missing internal ABI method")
	}
}

func TestProviderHistoryABINonIndexedFieldsHaveOwnedExplicitStorage(t *testing.T) {
	_, outcomes, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	fields := outcomes.Events["RecordChanged"].Inputs
	m := providerHistoryExecutionUnitMeter()
	got, ok := replayABINonIndexedObserved(fields, m)
	if !ok || !reflect.DeepEqual(got, fields.NonIndexed()) || len(got) != 8 || m.actual[10] != 16*uint64(reflect.TypeFor[abi.Argument]().Size())+9 {
		t.Fatal("bounded event arguments storage and actual copies", m.actual)
	}
	got[0].Name = "changed"
	if fields[2].Name != "did" {
		t.Fatal("argument slice aliases pinned event ABI")
	}
	if old, ok := replayABINonIndexedObserved(fields, nil); !ok || !reflect.DeepEqual(old, fields.NonIndexed()) {
		t.Fatal("legacy non-indexed projection")
	}
	if got, ok := replayABINonIndexedObserved(make(abi.Arguments, 129), providerHistoryExecutionUnitMeter()); ok || got != nil {
		t.Fatal("unbounded event ABI")
	}
	short := providerHistoryContentUnitMeter(candidateWorkVector{})
	if got, ok := replayABINonIndexedObserved(fields, short); ok || got != nil || !short.failed {
		t.Fatal("unfunded event argument storage")
	}
	short = providerHistoryContentUnitMeter(candidateWorkVector{10: 8*uint64(reflect.TypeFor[abi.Argument]().Size()) + 1})
	if got, ok := replayABINonIndexedObserved(fields, short); ok || got != nil || !short.failed {
		t.Fatal("funded storage but unfunded first field copy")
	}
}

func TestProviderHistoryRegistryEventRoutesABIObservationWithoutChangingStateRules(t *testing.T) {
	in := replayTestInput(t)
	contract, outcomes, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	c, ok := replayParseCommand(in.Registry, in.Agent, in.History[0].Command, contract)
	if !ok {
		t.Fatal("existing create precursor")
	}
	f := in.Blocks[1].Executions[0].Root.Calls[0]
	entry := f.Logs[0]
	newRun := func(m *providerWorkMeter) *replayRun {
		return &replayRun{registry: in.Registry, agent: in.Agent, contract: contract, outcomes: outcomes,
			claims:  map[replayClaimID]replayClaimState{{actor: c.actor, digest: replayClaim(in.Registry, c)}: {block: 0}},
			created: map[string]bool{}, history: in.History, budget: &replayBudget{limits: replayTestLimits(), observer: m}}
	}
	log := replayLog{log: &types.Log{Topics: entry.Topics, Data: entry.Data}, frame: &f, eligible: true}
	header := &types.Header{Number: big.NewInt(1), Time: 1}
	legacy := newRun(nil)
	if err := legacy.event(context.Background(), log, header); err != nil || len(legacy.steps) != 1 {
		t.Fatal("existing valid claim/event/state precursor", err)
	}
	m := providerHistoryStateUnitMeter()
	observed := newRun(m)
	if err := observed.event(context.Background(), log, header); err != nil || !reflect.DeepEqual(observed.steps, legacy.steps) || !reflect.DeepEqual(observed.state, legacy.state) || m.failed || m.actual[7] != 2 || m.actual[2] < uint64(len(f.Input)-4+len(entry.Data)) {
		t.Fatal("event did not retain its actual original ABI observer", err, m.actual)
	}
	noProof := providerHistoryExecutionUnitMeter()
	proofRun := newRun(noProof)
	if err := proofRun.event(context.Background(), log, header); err == nil || len(proofRun.steps) != 0 || !reflect.DeepEqual(proofRun.state, replayState{}) || !noProof.failed || noProof.actual[7] != 0 {
		t.Fatal("execution-only allowance admitted an actual key proof", err, noProof.actual)
	}
	short := providerHistoryContentUnitMeter(candidateWorkVector{})
	r := newRun(short)
	if err := r.event(context.Background(), log, header); err == nil || len(r.steps) != 0 || !reflect.DeepEqual(r.state, replayState{}) || !short.failed {
		t.Fatal("unfunded ABI work altered semantic state", err)
	}
}
