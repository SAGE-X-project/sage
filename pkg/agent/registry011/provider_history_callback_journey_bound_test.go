package registry011

import (
	"context"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
)

func historyCallbackJourneyTypeOracle(t *testing.T) [23]uint64 {
	t.Helper()
	sizes := [23]uint64{
		uint64(reflect.TypeFor[map[string]any]().Size()), uint64(reflect.TypeFor[map[[32]byte]bool]().Size()),
		uint64(reflect.TypeFor[string]().Size()), uint64(reflect.TypeFor[any]().Size()),
		uint64(reflect.TypeFor[reflect.Value]().Size()), uint64(reflect.TypeFor[reflect.StructField]().Size()),
		uint64(reflect.TypeFor[replayABIKey]().Size()), uint64(reflect.TypeFor[replayABIService]().Size()),
		uint64(reflect.TypeFor[replayKey]().Size()), uint64(reflect.TypeFor[replayService]().Size()),
		uint64(reflect.TypeFor[abi.Method]().Size()), uint64(reflect.TypeFor[abi.Argument]().Size()),
		uint64(reflect.TypeFor[replayClaimID]().Size()), uint64(reflect.TypeFor[replayClaimState]().Size()),
		uint64(reflect.TypeFor[bool]().Size()), uint64(reflect.TypeFor[replayStream]().Size()),
		uint64(reflect.TypeFor[replayState]().Size()), uint64(reflect.TypeFor[replayGrant]().Size()),
		uint64(reflect.TypeFor[replayRecord]().Size()), uint64(reflect.TypeFor[[]replayGrant]().Size()),
		uint64(reflect.TypeFor[HistoryContentStep]().Size()), uint64(reflect.TypeFor[providerHistoryExpectedItem]().Size()),
		uint64(reflect.TypeFor[[]providerHistoryExpectedItem]().Size()),
	}
	for _, n := range sizes {
		if n > 4096 {
			t.Fatal("changed source targets require new portable rows")
		}
	}
	return sizes
}

func TestProviderHistoryCallbackJourneyMatchesIndependentFlatRows(t *testing.T) {
	sizes := historyCallbackJourneyTypeOracle(t)
	for _, row := range []struct {
		windows, versions, logs, evidence, slots, attempts, maximum, moves, old, bytes, decoded, proofs, base, incoming uint64
		coefficients                                                                                                    [23]uint64
	}{
		{1, 0, 1, 1, 1, 0, 0, 0, 0, 0, 1524139, 0, 1846042364052, 2474102493185, [23]uint64{2194, 2, 8677, 11000, 3776, 1888, 1280, 160, 512, 64, 16, 4384, 2, 2, 1, 0, 0, 0, 0, 0, 0, 0, 0}},
		{1, 1, 1, 1024, 1, 1, 1, 2, 0, 1900544, 1636566, 128, 11862977451628, 2517724786757, [23]uint64{3801, 2, 15590, 18534, 3776, 1888, 1280, 160, 18177, 160, 16, 4384, 2, 2, 3, 1, 2, 768, 2, 5, 1, 2, 1}},
		{2, 3, 2, 4096, 4, 3, 2, 9, 1, 1941504, 2212067, 384, 31951969793598, 2517724786757, [23]uint64{10306, 5, 42433, 50102, 9440, 4720, 3200, 400, 54275, 448, 40, 10960, 8, 8, 10, 3, 6, 2304, 6, 15, 3, 9, 3}},
		{3, 4096, 256, 1048576, 768, 768, 256, 197376, 97920, 17629184, 191173823, 98304, 7717675404791647, 2517724786757, [23]uint64{2077769, 769, 8645874, 10015612, 1451872, 725936, 492160, 61520, 13763584, 98336, 6152, 1685648, 1536, 1536, 2304, 768, 1536, 589824, 1536, 3840, 768, 197376, 768}},
		{2, 32, 32, 1048576, 64, 32, 32, 1056, 496, 12386304, 37924479, 4096, 339808539890767, 2517724786757, [23]uint64{122729, 65, 503250, 598588, 122720, 61360, 41600, 5200, 581920, 5152, 520, 142480, 128, 128, 128, 32, 64, 24576, 64, 160, 32, 1056, 32}},
		{2, 33, 33, 1048576, 66, 33, 33, 1122, 528, 12386304, 38212981, 4224, 349825343215539, 2517724786757, [23]uint64{126530, 67, 518841, 617122, 126496, 63248, 42880, 5360, 600097, 5312, 536, 146864, 132, 132, 132, 33, 66, 25344, 66, 165, 33, 1122, 33}},
		{1, 4097, 65536, 67108864, 65536, 4097, 4096, 16785409, 8388607, 337444864, 7485379319, 524416, 41838448107138996, 2517724786757, [23]uint64{78477968, 65537, 312687603, 391320298, 123733856, 61866928, 41943680, 5242960, 89150977, 2490496, 524296, 143657104, 131072, 131072, 73730, 4097, 8194, 3146496, 8194, 20485, 4097, 16785409, 4097}},
		{1048576, 0, 1, 1, 1048576, 0, 0, 0, 0, 0, 99301576639, 0, 4399930961717727, 2474102493185, [23]uint64{1150288969, 1048577, 4549775602, 5767173500, 1979713376, 989856688, 671089280, 83886160, 268435712, 33554464, 8388616, 2298480784, 2097152, 2097152, 1048576, 0, 0, 0, 0, 0, 0, 0, 0}},
	} {
		units := row.base
		for c, copies := range row.coefficients {
			units += copies * sizes[c]
		}
		got, ok := providerHistoryCallbackJourneyUpper(row.windows, row.versions, row.logs, row.evidence)
		normal := candidateWorkVector{2: row.decoded, 7: row.proofs, 10: units}
		incoming := candidateWorkVector{10: row.incoming}
		envelope := normal
		envelope[10] += row.incoming
		if !ok || got.normal != normal || got.incoming != incoming || got.envelope != envelope || got.windows != row.windows || got.eventSlots != row.slots || got.selectedAttempts != row.attempts || got.maximumQueue != row.maximum || got.queueMoves != row.moves || got.oldSlots != row.old || got.queueBytes != row.bytes {
			t.Fatal("independent flat23type/full11vector callback row", row, got, ok)
		}
		if row.attempts == 0 && (got.state != (candidateWorkVector{}) || got.queue != (candidateWorkVector{})) {
			t.Fatal("other-subject callback capacity gained selected state work", got)
		}
	}
}

func TestProviderHistoryCallbackJourneyInvalidDomainsReturnWholeZero(t *testing.T) {
	for _, row := range [][4]uint64{
		{0, 0, 1, 1}, {(1 << 20) + 1, 0, 1, 1}, {1, (1 << 20) + 1, 1, 1},
		{1, 0, 0, 1}, {1, 0, 65537, 1}, {1, 0, 1, 0}, {1, 0, 1, (64 << 20) + 1},
		{^uint64(0), 0, 1, 1}, {1, ^uint64(0), 1, 1}, {1, 0, ^uint64(0), 1}, {1, 0, 1, ^uint64(0)},
	} {
		if got, ok := providerHistoryCallbackJourneyUpper(row[0], row[1], row[2], row[3]); ok || got != (providerHistoryCallbackJourneyCost{}) {
			t.Fatal("invalid domains leaked partial callback work", row, got, ok)
		}
	}
}

func TestProviderHistoryCallbackJourneyRefusesParentSumAfterFiniteChildren(t *testing.T) {
	historyCallbackJourneyTypeOracle(t)
	const windows, versions, logs, evidence uint64 = 1 << 20, 1 << 20, 1, 1 << 20
	binary, bOK := providerHistoryABIWindowUpper(logs, evidence)
	command, cOK := providerHistoryCommandWindowUpper(logs, evidence)
	state, sOK := providerHistoryStateApplyUpper()
	if !bOK || !cOK || !sOK {
		t.Fatal("finite child rows")
	}
	for _, child := range []struct {
		row   candidateWorkVector
		count uint64
	}{{binary.window, windows}, {command.actualWindow, windows}, {state, versions}} {
		for _, n := range child.row {
			if _, ok := providerUpperProduct(n, child.count); !ok {
				t.Fatal("overflow must belong to parent sum, not child")
			}
		}
	}
	// Independent positive-type-free parent lower19636793474992466399
	// exceeds uint64Max. Scalar DATA only; no workload/source is built and
	// this quote overflow is not a measured feasibility lower bound.
	if got, ok := providerHistoryCallbackJourneyUpper(windows, versions, logs, evidence); ok || got != (providerHistoryCallbackJourneyCost{}) {
		t.Fatal("finite components leaked an overflowing parent", got, ok)
	}
}

func historyCallbackJourneyObservedFixture(t *testing.T, other bool, maximumItems uint64) (*providerHistorySemanticFold, ReplayInput, *providerWorkMeter) {
	t.Helper()
	in := replayTestInput(t)
	c := journalTestConfig(t, in)
	if other {
		c.Agent = "other.selected.subject"
	}
	m := providerHistorySemanticUnitMeter()
	f, err := newProviderHistorySemanticFoldObserved("0x"+strings.Repeat("a", 64), c, in.Anchor, uint64(len(in.Blocks)), maximumItems, m)
	if err != nil || f.beginWindowCandidate(context.Background()) != nil {
		t.Fatal("original observed callback window", err)
	}
	return f, in, m
}

func TestProviderHistoryCallbackJourneyCoversOrdinaryCompleteEventSequence(t *testing.T) {
	f, in, m := historyCallbackJourneyObservedFixture(t, false, 11)
	limits := f.selection.config.Replay
	upper, ok := providerHistoryCallbackJourneyUpper(1, f.selection.maximumItems, uint64(limits.MaxLogs), uint64(limits.MaxEvidenceBytes))
	if !ok {
		t.Fatal("original numeric callback capacity")
	}
	before := m.actual
	for i := range in.Blocks {
		frame := &in.Blocks[i].Executions[0].Root.Calls[0]
		if err := f.window.run.eventDeriving(context.Background(), providerHistoryEventUnitLog(frame), &types.Header{Number: big.NewInt(int64(i)), Time: uint64(i)}, f.window); err != nil {
			t.Fatal("ordinary actual commit/create/allmutation callback", i, err)
		}
	}
	delta := historyEventQueueDelta(t, m, before)
	if m.failed || m.clocks != 0 || delta[7] != 3 || len(f.window.queue) != 11 || f.items != 11 || !f.window.run.created[replayTestAgent] {
		t.Fatal("ordinary callback sequence changed", delta)
	}
	historyEventQueueWithin(t, delta, upper.normal)
	// Only actual private callbacks are measured here; ABI setup/opening,
	// block decoding/traces, history/stop/final and Source counts are separate.
}

func TestProviderHistoryCallbackJourneyZeroSelectedCapacityKeepsOtherAgentIndexes(t *testing.T) {
	// Zero is selected BEFORE the original constructor. No quota or actual
	// count is changed after construction to fabricate a positive precursor.
	f, in, m := historyCallbackJourneyObservedFixture(t, true, 0)
	limits := f.selection.config.Replay
	upper, ok := providerHistoryCallbackJourneyUpper(1, 0, uint64(limits.MaxLogs), uint64(limits.MaxEvidenceBytes))
	if !ok {
		t.Fatal("zero-selected callback capacity")
	}
	before := m.actual
	for i := range in.Blocks {
		frame := &in.Blocks[i].Executions[0].Root.Calls[0]
		if err := f.window.run.eventDeriving(context.Background(), providerHistoryEventUnitLog(frame), &types.Header{Number: big.NewInt(int64(i)), Time: uint64(i)}, f.window); err != nil {
			t.Fatal("other-agent actual callback", i, err)
		}
	}
	delta := historyEventQueueDelta(t, m, before)
	if m.failed || m.clocks != 0 || delta[7] != 0 || f.items != 0 || len(f.window.queue) != 0 || f.window.run.state.version != 0 || !f.window.run.created[replayTestAgent] || len(f.window.run.claims) != 1 {
		t.Fatal("other-agent changed selected state or lost global indexes", delta)
	}
	historyEventQueueWithin(t, delta, upper.normal)
}

func TestProviderHistoryCallbackJourneyFundsStateBeforeFailedQueueRetention(t *testing.T) {
	in := replayTestInput(t)
	changed := replayTestClone(t, in.Blocks[1].Executions[0].Root.Calls[0])
	_, outcomes, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	fields := outcomes.Events["RecordChanged"].Inputs.NonIndexed()
	values, err := fields.Unpack(changed.Logs[0].Data)
	if err != nil {
		t.Fatal("ordinary changed event precursor", err)
	}
	digest := values[5].([32]byte)
	digest[0] ^= 1 // One in-memory wrong record digest; no peer or RPC.
	values[5] = digest
	changed.Logs[0].Data, err = fields.Pack(values...)
	if err != nil {
		t.Fatal(err)
	}
	c := journalTestConfig(t, in)
	m := providerHistorySemanticUnitMeter()
	f, err := newProviderHistorySemanticFoldObserved("0x"+strings.Repeat("a", 64), c, in.Anchor, uint64(len(in.Blocks)), 1, m)
	if err != nil || f.beginWindowCandidate(context.Background()) != nil {
		t.Fatal("original one-attempt callback window", err)
	}
	commit := &in.Blocks[0].Executions[0].Root.Calls[0]
	if err := f.window.run.eventDeriving(context.Background(), providerHistoryEventUnitLog(commit), &types.Header{Number: big.NewInt(0)}, f.window); err != nil {
		t.Fatal("actual prior commit", err)
	}
	upper, ok := providerHistoryCallbackJourneyUpper(1, 1, uint64(c.Replay.MaxLogs), uint64(c.Replay.MaxEvidenceBytes))
	if !ok {
		t.Fatal("one selected-attempt quote")
	}
	before := m.actual
	err = f.window.run.eventDeriving(context.Background(), providerHistoryEventUnitLog(&changed), &types.Header{Number: big.NewInt(1), Time: 1}, f.window)
	delta := historyEventQueueDelta(t, m, before)
	if err == nil || m.failed || m.clocks != 0 || delta[7] != 2 || f.items != 0 || len(f.window.queue) != 0 || !f.window.pending || f.window.pendingCommand == "" || f.window.run.state.version != 1 || f.confirmed.state.version != 0 {
		t.Fatal("entered state vanished behind unsuccessful retention", err, delta)
	}
	historyEventQueueWithin(t, delta, upper.normal)
	// End on the first private callback error: no retry/ack/next event. The
	// outer block/fold caller owns quarantine; this UNIT does not assert an
	// acquired Source, frame/receipt witness or a fold failure bit it never set.
}
