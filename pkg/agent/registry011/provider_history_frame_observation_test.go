package registry011

import (
	"context"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func providerHistoryExecutionUnitMeter() *providerWorkMeter {
	return providerHistoryContentUnitMeter(candidateWorkVector{2: 1 << 30, 5: 65536, 6: 65536, 10: 1 << 60})
}

// Normalized DATA for ordering/context rules, not a qualified EVM execution.
func providerHistoryExecutionUnitFrame() ReplayFrame {
	other := replayTestWallet
	log := func(address string, data byte, after int) ReplayFrameLog {
		return ReplayFrameLog{Address: address, Data: []byte{data}, AfterChildren: after}
	}
	return ReplayFrame{Type: "CALL", From: replayTestOperator, To: SepoliaRegistryAddress, Value: "0x0",
		Calls: []ReplayFrame{
			{Type: "CALL", From: SepoliaRegistryAddress, To: other, Value: "0x0", Logs: []ReplayFrameLog{log(other, 2, 0)}},
			{Type: "DELEGATECALL", From: SepoliaRegistryAddress, To: other, Value: "0x0",
				Calls: []ReplayFrame{{Type: "CALL", From: SepoliaRegistryAddress, To: SepoliaRegistryAddress, Value: "0x0", Logs: []ReplayFrameLog{log(SepoliaRegistryAddress, 4, 0)}}},
				Logs:  []ReplayFrameLog{log(SepoliaRegistryAddress, 5, 1)}},
		}, Logs: []ReplayFrameLog{log(SepoliaRegistryAddress, 1, 0), log(SepoliaRegistryAddress, 3, 1), log(SepoliaRegistryAddress, 6, 2)},
	}
}

func TestProviderHistoryObservedFramesPreserveOrderContextAndEligibility(t *testing.T) {
	for _, tc := range []struct {
		name              string
		static, committed bool
		reverted          bool
	}{
		{"normal", false, true, false}, {"inherited-static", true, true, false}, {"uncommitted", false, false, false}, {"reverted-child", false, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := providerHistoryExecutionUnitFrame()
			if tc.reverted {
				f.Calls[1].Error = "reverted"
			}
			legacy := &replayBudget{limits: replayTestLimits()}
			want, err := replayFrame(context.Background(), &f, "", tc.static, tc.committed, 0, legacy)
			if err != nil {
				t.Fatal("existing normalized DATA precursor", err)
			}
			m := providerHistoryExecutionUnitMeter()
			b := &replayBudget{limits: replayTestLimits(), observer: m}
			got, err := replayFrame(context.Background(), &f, "", tc.static, tc.committed, 0, b)
			if err != nil || !reflect.DeepEqual(got, want) || b.bytes != legacy.bytes || b.frames != legacy.frames || b.traceLogs != legacy.traceLogs || m.actual[5] != 4 || m.actual[6] != 6 || m.failed {
				t.Fatal("ordered logs/context or actual entry counts", err, m.actual)
			}
			if tc.name == "normal" {
				for i, l := range got {
					if l.log.Data[0] != byte(i+1) || l.eligible != (i == 0 || i == 2 || i == 3 || i == 5) {
						t.Fatal("own/child interleaving or delegated eligibility", i, l)
					}
				}
			}
		})
	}
	for _, kind := range []string{"CREATE", "CREATE2", "CALLCODE", "STATICCALL"} {
		f := ReplayFrame{Type: kind, From: replayTestWallet, To: SepoliaRegistryAddress, Value: "0x0"}
		parent := ""
		if kind == "CALLCODE" {
			parent = replayTestWallet
		}
		old, err := replayFrame(context.Background(), &f, parent, false, true, 0, &replayBudget{limits: replayTestLimits()})
		if err != nil {
			t.Fatal("ordinary opcode DATA precursor", kind, err)
		}
		got, err := replayFrameObserved(context.Background(), &f, parent, false, true, 0, &replayBudget{limits: replayTestLimits(), observer: providerHistoryExecutionUnitMeter()})
		if err != nil || !reflect.DeepEqual(got, old) || got == nil {
			t.Fatal("empty shape or opcode rule", kind, err)
		}
	}
}

func TestProviderHistoryObservedFramesRefuseInvalidNormalizedData(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*ReplayFrame)
	}{
		{"from", func(f *ReplayFrame) { f.From = "invalid" }},
		{"to", func(f *ReplayFrame) { f.To = "invalid" }},
		{"value", func(f *ReplayFrame) { f.Value = "0x00" }},
		{"parent-context", func(f *ReplayFrame) { f.Calls[0].From = replayTestOperator }},
		{"opcode", func(f *ReplayFrame) { f.Type = "UNKNOWN" }},
		{"static-value", func(f *ReplayFrame) { f.Type = "STATICCALL"; f.Value = "0x1" }},
		{"root-delegate", func(f *ReplayFrame) { f.Type = "DELEGATECALL" }},
		{"log-order", func(f *ReplayFrame) { f.Logs[1].AfterChildren = -1 }},
		{"log-position", func(f *ReplayFrame) { f.Logs[0].AfterChildren = 3 }},
		{"log-context", func(f *ReplayFrame) { f.Logs[0].Address = replayTestWallet }},
		{"topics", func(f *ReplayFrame) { f.Logs[0].Topics = make([]common.Hash, 5) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := providerHistoryExecutionUnitFrame()
			tc.edit(&f)
			if _, err := replayFrame(context.Background(), &f, "", false, true, 0, &replayBudget{limits: replayTestLimits()}); err == nil {
				t.Fatal("legacy DATA must first reject this single change")
			}
			if got, err := replayFrameObserved(context.Background(), &f, "", false, true, 0, &replayBudget{limits: replayTestLimits(), observer: providerHistoryExecutionUnitMeter()}); err == nil || got != nil {
				t.Fatal("invalid DATA returned partial logs", err)
			}
		})
	}
}

func TestProviderHistoryObservedFramesUseOriginalExactAndShortQuotas(t *testing.T) {
	f := providerHistoryExecutionUnitFrame()
	probe := providerHistoryExecutionUnitMeter()
	if _, err := replayFrameObserved(context.Background(), &f, "", false, true, 0, &replayBudget{limits: replayTestLimits(), observer: probe}); err != nil {
		t.Fatal("normal original-meter precursor", err)
	}
	for _, tc := range []struct {
		name  string
		upper candidateWorkVector
	}{
		{"zero", candidateWorkVector{}}, {"decoded", candidateWorkVector{5: 65536, 6: 65536, 10: 1 << 60}},
		{"next-frame", candidateWorkVector{2: 1 << 30, 5: 1, 6: 65536, 10: 1 << 60}},
		{"next-log", candidateWorkVector{2: 1 << 30, 5: 65536, 6: 1, 10: 1 << 60}},
		{"late-work", candidateWorkVector{2: 1 << 30, 5: 65536, 6: 65536, 10: probe.actual[10] - 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := providerHistoryContentUnitMeter(tc.upper)
			b := &replayBudget{limits: replayTestLimits(), observer: m}
			if got, err := replayFrameObserved(context.Background(), &f, "", false, true, 0, b); err == nil || got != nil || !m.failed {
				t.Fatal("short original quota returned partial log prefix", err, m.actual)
			}
			before := m.actual
			if got, err := replayFrameObserved(context.Background(), &f, "", false, true, 0, b); err == nil || got != nil || m.actual != before {
				t.Fatal("failed original observer resumed")
			}
		})
	}
	m := providerHistoryContentUnitMeter(probe.actual)
	if got, err := replayFrameObserved(context.Background(), &f, "", false, true, 0, &replayBudget{limits: replayTestLimits(), observer: m}); err != nil || len(got) != 6 || m.actual != probe.actual {
		t.Fatal("exact concrete comparison/copy/codec allowance", err, m.actual, probe.actual)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	fresh := func() *replayBudget {
		return &replayBudget{limits: replayTestLimits(), observer: providerHistoryExecutionUnitMeter()}
	}
	for _, input := range []struct {
		name  string
		ctx   context.Context
		frame *ReplayFrame
		depth int
		b     *replayBudget
	}{
		{"context", nil, &f, 0, fresh()}, {"cancelled", ctx, &f, 0, fresh()},
		{"frame", context.Background(), nil, 0, fresh()}, {"budget", context.Background(), &f, 0, nil},
		{"negative-depth", context.Background(), &f, -1, fresh()}, {"past-depth", context.Background(), &f, 33, fresh()},
		{"observer", context.Background(), &f, 0, &replayBudget{limits: replayTestLimits()}},
	} {
		t.Run(input.name, func(t *testing.T) {
			if got, err := replayFrameObserved(input.ctx, input.frame, "", false, true, input.depth, input.b); err == nil || got != nil {
				t.Fatal("single missing/cancelled/deep original walk")
			}
			if input.b != nil && input.b.observer != nil && input.b.observer.actual != (candidateWorkVector{}) {
				t.Fatal("guard entered a frame", input.b.observer.actual)
			}
		})
	}
	limits := replayTestLimits()
	limits.MaxFrames = 1
	b := &replayBudget{limits: limits, observer: providerHistoryExecutionUnitMeter()}
	if got, err := replayFrameObserved(context.Background(), &f, "", false, true, 0, b); err == nil || got != nil || b.observer.actual[5] != 1 || b.observer.actual[6] != 0 {
		t.Fatal("unentered child/log must not be counted", err, b.observer.actual)
	}
}

func TestProviderHistoryObservedLogAccumulatorOwnsBoundedGrowthAndCopies(t *testing.T) {
	m := providerHistoryExecutionUnitMeter()
	out := replayObservedLogs{maximum: 40, meter: m}
	for i := 0; i < 40; i++ {
		if !out.append(replayLog{log: &types.Log{Index: uint(i)}}) {
			t.Fatal("bounded positive append", i)
		}
	}
	size := uint64(reflect.TypeFor[replayLog]().Size())
	// Capacity16->32->40 initializes88 slots, copies48 old slots and writes40.
	if len(out.rows) != 40 || cap(out.rows) != 40 || m.actual[10] != 176*size+45 {
		t.Fatal("explicit storage/copy arithmetic", len(out.rows), cap(out.rows), m.actual, size)
	}
	for i, l := range out.rows {
		if l.log.Index != uint(i) {
			t.Fatal("log prefix changed", i)
		}
	}
	before := m.actual
	if out.append(replayLog{}) || m.actual != before {
		t.Fatal("full accumulator grew")
	}
	for _, maximum := range []int{-1, 0, 65537} {
		x := replayObservedLogs{maximum: maximum, meter: providerHistoryExecutionUnitMeter()}
		if x.append(replayLog{}) {
			t.Fatal("invalid output ceiling")
		}
	}
	if (*replayObservedLogs)(nil).append(replayLog{}) || (&replayObservedLogs{maximum: 1}).append(replayLog{}) {
		t.Fatal("missing output observer")
	}
	// First16 appends, then growth to32 with one unit short of the next write.
	upper := candidateWorkVector{10: (16+16+32+16+1)*size + 20 - 1}
	short := replayObservedLogs{maximum: 40, meter: providerHistoryContentUnitMeter(upper)}
	for i := 0; i < 16; i++ {
		if !short.append(replayLog{}) {
			t.Fatal("funded old prefix", i)
		}
	}
	if short.append(replayLog{}) || len(short.rows) != 16 || !short.meter.failed {
		t.Fatal("late append failure exposed a new row", short.meter.actual)
	}
	before = short.meter.actual
	if short.append(replayLog{}) || short.meter.actual != before {
		t.Fatal("failed append original resumed")
	}
}

func TestProviderHistoryObservedLogPayloadClonePreservesNilAndOwnedBytes(t *testing.T) {
	for _, l := range []*ReplayFrameLog{{Address: SepoliaRegistryAddress}, {Address: SepoliaRegistryAddress, Topics: []common.Hash{{1}}, Data: []byte{1, 2}}} {
		m := providerHistoryExecutionUnitMeter()
		got, err := replayFrameLogObserved(l, m)
		if err != nil || got.Address != common.HexToAddress(l.Address) || !reflect.DeepEqual(got.Topics, append([]common.Hash{}, l.Topics...)) || !reflect.DeepEqual(got.Data, l.Data) || m.actual[2] != 40 {
			t.Fatal("owned log clone shape", err, m.actual)
		}
		if len(got.Data) > 0 {
			got.Data[0] = 9
			if l.Data[0] != 1 {
				t.Fatal("data alias")
			}
		}
		if len(got.Topics) > 0 {
			got.Topics[0][0] = 9
			if l.Topics[0][0] != 1 {
				t.Fatal("topic alias")
			}
		}
	}
	for _, l := range []*ReplayFrameLog{nil, {Address: "invalid"}, {Address: SepoliaRegistryAddress, Topics: make([]common.Hash, 5)}} {
		if got, err := replayFrameLogObserved(l, providerHistoryExecutionUnitMeter()); err == nil || got != nil {
			t.Fatal("invalid clone input")
		}
	}
	if got, err := replayFrameLogObserved(&ReplayFrameLog{Address: SepoliaRegistryAddress}, nil); err == nil || got != nil {
		t.Fatal("missing clone observer")
	}
	m := providerHistoryContentUnitMeter(candidateWorkVector{})
	if got, err := replayFrameLogObserved(&ReplayFrameLog{Address: SepoliaRegistryAddress}, m); err == nil || got != nil || !m.failed {
		t.Fatal("unfunded clone")
	}
}
