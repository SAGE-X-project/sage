package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// Dividing a prospective repeated row removes F's conservative slack from
// original one-journey observations. It never resets a live original meter.
func historyEffectBranchOneJourney(t *testing.T, row candidateWorkVector, journeys uint64) candidateWorkVector {
	t.Helper()
	if journeys == 0 {
		t.Fatal("positive original journey count")
	}
	for c, n := range row {
		if n%journeys != 0 {
			t.Fatal("component lost its original journey factor", c, n, journeys)
		}
		row[c] = n / journeys
	}
	return row
}

func TestProviderHistoryEffectBranchMatchesIndependentPortableRows(t *testing.T) {
	w := uint64(strconv.IntSize / 8)
	integer := uint64(reflect.TypeFor[big.Int]().Size())
	logSize := uint64(reflect.TypeFor[types.Log]().Size())
	rowSize := uint64(reflect.TypeFor[replayLog]().Size())
	blockSize := uint64(reflect.TypeFor[providerHistoryCollectionBlock]().Size())
	effectSize := uint64(reflect.TypeFor[providerHistoryCollectionEffect]().Size())
	executionSize := uint64(reflect.TypeFor[ReplayExecution]().Size())
	stringSize := uint64(reflect.TypeFor[string]().Size())
	sidecar := uint64(reflect.TypeFor[types.Transaction]().Size()) + uint64(reflect.TypeFor[types.BlobTx]().Size()) + 2
	for _, row := range []struct {
		input         [12]uint64 // G,R,B,W,E,T,A,L,C,V,Q,K.
		shape         [7]uint64  // F,F*n,F*Q,F*nA,F*nL,F*n(A-1),F*raw.
		decoded       [2]uint64
		normalization [2]uint64 // Constant,w.
		execution     [5]uint64 // Constant,w,I,replayLog-size,types.Log-size.
		collection    [6]uint64 // Constant,block,effect,string,execution,sidecar.
		room, batches uint64
	}{
		{[12]uint64{1, 1, 1, 1, 65536, 1, 2, 1, 2048, 1024, 1, 1}, [7]uint64{15, 15, 15, 30, 15, 15, 30720}, [2]uint64{4249080, 4935}, [2]uint64{8688100425, 6083760}, [5]uint64{2343309645, 15, 15, 345, 15}, [6]uint64{1007715120, 30, 15, 45, 15, 15}, 67554509807616015, 1},
		{[12]uint64{2, 3, 5, 2, 65536, 2, 4, 3, 8192, 1024, 3, 2}, [7]uint64{29, 58, 87, 232, 174, 174, 712704}, [2]uint64{98445488, 35815}, [2]uint64{805836440862, 141129312}, [5]uint64{14667857974, 87, 87, 2610, 174}, [6]uint64{3897236891, 290, 174, 261, 87, 87}, 261210771256115258, 1},
		{[12]uint64{1, 1, 3, 3, 65536, 2, 2, 1, 131072, 1024, 5, 3}, [7]uint64{15, 45, 45, 90, 45, 45, 2949120}, [2]uint64{407007720, 14805}, [2]uint64{53343997172955, 583929360}, [5]uint64{200117570535, 45, 45, 1035, 45}, [6]uint64{3026003220, 90, 135, 135, 45, 45}, 202663529422848045, 1},
		{[12]uint64{1, 1, 3, 1, 65536, 2, 4, 4, 131072, 1024, 2, 3}, [7]uint64{15, 15, 30, 60, 60, 45, 1966080}, [2]uint64{271347360, 11070}, [2]uint64{35562665344950, 389288070}, [5]uint64{131137013160, 30, 30, 900, 60}, [6]uint64{1009722675, 90, 90, 90, 30, 30}, 67554509807616015, 1},
		{[12]uint64{1, 1, 2, 2, 65536, 1, 1, 1, 8192, 1024, 0, 1}, [7]uint64{15, 0, 0, 0, 0, 0, 0}, [2]uint64{0, 0}, [2]uint64{0, 0}, [5]uint64{0, 0, 0, 0, 0}, [6]uint64{5370, 60, 0, 0, 0, 0}, 0, 0},
		{[12]uint64{1, 32, 3, 3, 67108864, 16384, 65536, 65536, 8388608, 8388608, 3, 10}, [7]uint64{15, 45, 45, 2949120, 2949120, 2949075, 377487360}, [2]uint64{53485240320, 306715725}, [2]uint64{436990058923622490, 74978425170}, [5]uint64{50666607698531760, 45, 45, 20644560, 2949120}, [6]uint64{25332751420991595, 90, 450, 135, 45, 45}, 202663529422848045, 1},
	} {
		i := row.input
		got, ok := providerHistoryEffectBranchUpper(
			providerHistoryCollectionSemanticLimits{Groups: i[0], Requests: i[1], Blocks: i[2], Windows: i[3]},
			providerHistoryEffectBranchLimits{i[4], i[5], i[6], i[7], i[8], i[9], i[10], i[11]})
		normal := candidateWorkVector{2: row.decoded[0], 5: row.shape[3], 6: row.shape[4], 10: row.normalization[0] + row.normalization[1]*w}
		execution := candidateWorkVector{2: row.decoded[1], 5: row.shape[3], 6: row.shape[4],
			10: row.execution[0] + row.execution[1]*w + row.execution[2]*integer + row.execution[3]*rowSize + row.execution[4]*logSize}
		collection := candidateWorkVector{10: row.collection[0] + row.collection[1]*blockSize + row.collection[2]*effectSize +
			row.collection[3]*stringSize + row.collection[4]*executionSize + row.collection[5]*sidecar}
		var actual, total candidateWorkVector
		for c := range actual {
			actual[c] = normal[c] + execution[c] + collection[c]
			total[c] = actual[c]
		}
		total[10] += row.room
		if !ok || got.normalization != normal || got.execution != execution || got.collection != collection ||
			got.actual != actual || got.total != total || got.preflight != (candidateWorkVector{10: row.room}) || got.blockingBatches != row.batches ||
			([7]uint64{got.journeys, got.windows, got.selected, got.frames, got.logs, got.children, got.traceBytes}) != row.shape {
			t.Fatal("independent complete selected effect rows", got, ok, normal, execution, collection)
		}
	}
}

func TestProviderHistoryEffectBranchClipsOnlySuccessfulOriginalSelection(t *testing.T) {
	l := providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 2, Windows: 2}
	p := providerHistoryEffectBranchLimits{65536, 4, 2, 1, 8192, 1024, 8, 2}
	got, ok := providerHistoryEffectBranchUpper(l, p)
	if !ok || got.selected != 30 || got.windows != 30 || got.frames != 60 || got.logs != 30 || got.children != 30 {
		t.Fatal("original log/frame/transaction capacities did not refine the admitted maximum", got, ok)
	}
	// Four prospective roots do not prove four roots actually enter. A single
	// four-frame trace can still own three child values, not zero children.
	l.Blocks, l.Windows = 1, 1
	p.Transactions, p.Frames, p.Logs, p.Effects = 4, 4, 6, 4
	got, ok = providerHistoryEffectBranchUpper(l, p)
	if !ok || got.selected != 60 || got.frames != 60 || got.children != 45 {
		t.Fatal("prospective root maximum was subtracted from frame capacity", got, ok)
	}
}

func TestProviderHistoryEffectBranchNormalizesCompleteTreesOnSameObserver(t *testing.T) {
	frame := providerHistoryExecutionUnitFrame()
	raw, err := json.Marshal(nodeTestFrame(frame))
	limits := replayTestLimits()
	want, oldErr := DecodeCallTracer(context.Background(), raw, limits)
	if err != nil || oldErr != nil {
		t.Fatal("ordinary finite complete-tree precursor", err, oldErr)
	}
	before := bytes.Clone(raw)
	m := providerHistoryExecutionUnitMeter()
	var first candidateWorkVector
	for pass := uint64(1); pass <= 2; pass++ {
		got, decodeErr := decodeCallTracerObserved(context.Background(), raw, limits, m)
		if decodeErr != nil || !reflect.DeepEqual(got, want) || !bytes.Equal(raw, before) || m.failed {
			t.Fatal("original complete frame decoder changed bytes/tree", decodeErr, m.actual)
		}
		if pass == 1 {
			first = m.actual
			if first[5] != 4 || first[6] != 6 {
				t.Fatal("original frame/log entry counts", first)
			}
		}
		for c, n := range first {
			if m.actual[c] != pass*n {
				t.Fatal("original normalizer reused work or reset its observer", c, m.actual, first)
			}
		}
	}
	upper, ok := providerHistoryEffectBranchUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1},
		providerHistoryEffectBranchLimits{65536, 4, 8, 12, 8192, 1024, 4, 1})
	if !ok || len(raw) > 8192 {
		t.Fatal("independent whole-window fixture limits")
	}
	historyEventQueueWithin(t, m.actual, historyEffectBranchOneJourney(t, upper.normalization, upper.journeys))
}

func TestProviderHistoryEffectBranchCoversOptionalDiagnosticsAndLogForms(t *testing.T) {
	limits := replayTestLimits()
	base := ReplayFrame{Type: "CALL", From: replayTestWallet, To: SepoliaRegistryAddress, Value: "0x0"}
	values := []map[string]any{}
	for _, kind := range []string{"CALL", "STATICCALL", "CALLCODE", "DELEGATECALL", "CREATE", "CREATE2"} {
		f := base
		f.Type = kind
		v := nodeTestFrame(f)
		v["output"], v["error"], v["revertReason"] = "0x0001", strings.Repeat("e", 4096), strings.Repeat("r", 4096)
		v["gas"], v["gasUsed"], v["value"] = "0xffffffffffffffff", "0xffffffffffffffff", "0x"+strings.Repeat("f", 64)
		v["executionGasUsed"], v["stateGasUsed"], v["gasRefund"] = "0x1", "0x0", "0x0"
		v["calls"] = []any{}
		v["logs"] = []any{map[string]any{"address": SepoliaRegistryAddress, "data": "0x0001",
			"topics":   []string{common.Hash{1}.Hex(), common.Hash{2}.Hex(), common.Hash{3}.Hex(), common.Hash{4}.Hex()},
			"position": 0, "index": ^uint64(0)}}
		if kind == "STATICCALL" {
			delete(v, "value")
		}
		values = append(values, v)
	}
	root, child := nodeTestFrame(base), nodeTestFrame(ReplayFrame{Type: "CALL", From: SepoliaRegistryAddress, To: replayTestWallet, Value: "0x0"})
	root["regularGasUsed"], root["stateGasUsed"], root["gasRefund"], root["stateGasReservoir"] = "0x1", "0x0", "0x0", "0x1"
	child["stateGasUsed"], child["stateGasReservoir"] = "-0x1", "0x1"
	root["calls"] = []any{child}
	values = append(values, root)
	upper, ok := providerHistoryEffectBranchUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1},
		providerHistoryEffectBranchLimits{65536, 1, 2, 1, 16384, 1024, 1, 1})
	if !ok {
		t.Fatal("ordinary full diagnostic numeric domain")
	}
	for _, v := range values {
		raw, err := json.Marshal(v)
		want, oldErr := DecodeCallTracer(context.Background(), raw, limits)
		if err != nil || oldErr != nil || len(raw) > 16384 {
			t.Fatal("ordinary supported diagnostic/log precursor", err, oldErr)
		}
		m := providerHistoryExecutionUnitMeter()
		got, err := decodeCallTracerObserved(context.Background(), raw, limits, m)
		if err != nil || !reflect.DeepEqual(got, want) || m.failed {
			t.Fatal("optional diagnostic/log normalization changed", err, m.actual)
		}
		historyEventQueueWithin(t, m.actual, historyEffectBranchOneJourney(t, upper.normalization, upper.journeys))
	}
}

func TestProviderHistoryEffectBranchNormalizationRefusalKeepsFirstScan(t *testing.T) {
	raw, err := json.Marshal(nodeTestFrame(providerHistoryExecutionUnitFrame()))
	if err != nil || len(raw) == 0 {
		t.Fatal("ordinary nonempty complete-tree input", err)
	}
	m := providerHistoryContentUnitMeter(candidateWorkVector{2: uint64(len(raw) - 1), 5: 65536, 6: 65536, 10: 1 << 60})
	want := candidateWorkVector{10: uint64(len(raw)) + 1}
	for attempt := 0; attempt < 2; attempt++ {
		got, err := decodeCallTracerObserved(context.Background(), raw, replayTestLimits(), m)
		if err == nil || !reflect.DeepEqual(got, ReplayFrame{}) || !m.failed || m.actual != want {
			t.Fatal("first codec refusal lost entered scan or resumed", err, m.actual, want)
		}
	}
}

func TestProviderHistoryEffectBranchCoversOriginalTransactionAndReceiptJoin(t *testing.T) {
	e, block := providerHistoryExecutionUnitInput(t)
	raw, err := json.Marshal(nodeTestFrame(e.Root))
	canonical, txErr := block.txs[0].MarshalBinary()
	want, oldErr := replayExecution(context.Background(), &e, block, 0, &replayBudget{limits: replayTestLimits()})
	if err != nil || txErr != nil || oldErr != nil {
		t.Fatal("ordinary original execution/receipt precursor", err, txErr, oldErr)
	}
	m := providerHistoryExecutionUnitMeter()
	got, err := replayExecutionObserved(context.Background(), &e, block, 0, &replayBudget{limits: replayTestLimits(), observer: m})
	after, afterErr := block.txs[0].MarshalBinary()
	rootAfter, rootErr := json.Marshal(nodeTestFrame(e.Root))
	if err != nil || !reflect.DeepEqual(got, want) || m.failed || afterErr != nil || rootErr != nil ||
		!bytes.Equal(canonical, after) || !bytes.Equal(raw, rootAfter) || m.actual[5] != 2 || m.actual[6] != 1 {
		t.Fatal("root/caller/receipt/log order or original bytes changed", err, m.actual)
	}
	upper, ok := providerHistoryEffectBranchUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1},
		providerHistoryEffectBranchLimits{65536, 1, 2, 1, 8192, 1024, 1, 1})
	if !ok {
		t.Fatal("ordinary one-execution bound")
	}
	historyEventQueueWithin(t, m.actual, historyEffectBranchOneJourney(t, upper.execution, upper.journeys))
}

func TestProviderHistoryEffectBranchCoversOriginalCreateExecution(t *testing.T) {
	// Same public finite normalized DATA as the original CREATE unit; no
	// account key, broadcast, deployment or EVM execution is constructed.
	tx := types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(11155111), GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(2),
		Gas: 21000, Value: new(big.Int), V: big.NewInt(0), R: big.NewInt(1), S: big.NewInt(1)})
	from, err := types.Sender(types.LatestSignerForChainID(big.NewInt(11155111)), tx)
	if err != nil {
		t.Fatal("existing finite native recovery precursor", err)
	}
	header := &types.Header{Number: big.NewInt(1), Difficulty: new(big.Int)}
	block := replayDecodedBlock{header: header, txs: types.Transactions{tx}, receipts: types.Receipts{{Status: 1}}}
	e := ReplayExecution{BlockHash: header.Hash().Hex(), TransactionHash: tx.Hash().Hex(),
		Root: ReplayFrame{Type: "CREATE", From: replayAddress(from), To: replayAddress(crypto.CreateAddress(from, tx.Nonce())), Value: "0x0"}}
	want, oldErr := replayExecution(context.Background(), &e, block, 0, &replayBudget{limits: replayTestLimits()})
	if oldErr != nil {
		t.Fatal("ordinary CREATE precursor", oldErr)
	}
	m := providerHistoryExecutionUnitMeter()
	got, err := replayExecutionObserved(context.Background(), &e, block, 0, &replayBudget{limits: replayTestLimits(), observer: m})
	upper, ok := providerHistoryEffectBranchUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1},
		providerHistoryEffectBranchLimits{65536, 1, 1, 1, 8192, 1024, 1, 1})
	if err != nil || !ok || !reflect.DeepEqual(got, want) || got == nil || m.actual[5] != 1 || m.actual[6] != 0 {
		t.Fatal("CREATE/empty returned logs changed", err, got, m.actual)
	}
	historyEventQueueWithin(t, m.actual, historyEffectBranchOneJourney(t, upper.execution, upper.journeys))
}

func TestProviderHistoryEffectBranchRetainsEffectsWithExactPrefixCopies(t *testing.T) {
	c, in, _ := historyCollectionContentInputFixture(t, 2, historyCollectionStreamingInput(t), SepoliaAmsterdamRootExecutionGasProfile)
	m := c.semantic.observer
	before := m.actual
	size := uint64(reflect.TypeFor[providerHistoryCollectionEffect]().Size())
	// Two then38 effects initialize2+40 descriptors, not a global B*Q
	// for every subsequent independent content batch.
	want := uint64(1682) + 42*size
	for i, raw := range in.Blocks {
		block, err := replayDecodeContent(context.Background(), raw, &replayBudget{limits: replayTestLimits()})
		if err != nil || !c.retainEffectsObserved(block, journalTestHeader(t, raw), m) {
			t.Fatal("ordinary canonical selected receipt precursor", i, err)
		}
		for _, tx := range raw.Transactions {
			n := uint64(len(tx))
			want += n*n + n + 4359
		}
	}
	delta := historyEventQueueDelta(t, m, before)
	if len(c.effects) != 40 || delta != (candidateWorkVector{10: want}) || m.failed {
		t.Fatal("exact original selection scan/hash/descriptor copy accounting", delta, want)
	}
	for i, effect := range c.effects {
		blockIndex, transactionIndex := 0, i
		if i >= 2 {
			blockIndex, transactionIndex = 1, i-2
		}
		expected := in.Blocks[blockIndex].Executions[transactionIndex]
		if effect.block != expected.BlockHash || effect.transaction != expected.TransactionHash || effect.index != uint64(transactionIndex) {
			t.Fatal("selected effects lost canonical receipt order", i, effect)
		}
	}
	upper, ok := providerHistoryEffectBranchUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 2, Windows: 1},
		providerHistoryEffectBranchLimits{64 << 20, 40, 80, 40, 8 << 20, 4096, 40, 2})
	if !ok {
		t.Fatal("ordinary two-block effect bound")
	}
	historyEventQueueWithin(t, delta, historyEffectBranchOneJourney(t, upper.collection, upper.journeys))
}

func TestProviderHistoryEffectBranchWaitsForAllOriginalTraceResults(t *testing.T) {
	c, in, methods := historyCollectionContentInputFixture(t, 2, historyCollectionStreamingInput(t), SepoliaAmsterdamRootExecutionGasProfile)
	c = historyCollectionContentStep(t, c, in, methods) // Original open.
	c = historyCollectionContentStep(t, c, in, methods) // Two blocks,40 effects.
	c = historyCollectionContentStep(t, c, in, methods) // First32 complete results.
	m := c.semantic.observer
	before := m.actual
	if c.pendingApplied != 1 || c.effectNext != 32 || c.semantic.blocks != 1 || len(c.effects) != 40 ||
		!c.applyReadyBlocksObserved(context.Background(), m) || c.pendingApplied != 1 || c.semantic.blocks != 1 ||
		historyEventQueueDelta(t, m, before) != (candidateWorkVector{10: 360}) {
		t.Fatal("blocked whole-block scan changed the confirmed semantic prefix", m.actual, before)
	}
	c = historyCollectionContentStep(t, c, in, methods)
	if c.phase != "state" || c.semantic.blocks != 2 || c.semantic.confirmed.state.version != 0 || len(c.pending) != 0 || len(c.effects) != 0 {
		t.Fatal("complete final eight results did not release the original batch", c.phase)
	}
}

func TestProviderHistoryEffectBranchZeroSelectionStillPricesPendingContent(t *testing.T) {
	got, ok := providerHistoryEffectBranchUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 2, Windows: 2},
		providerHistoryEffectBranchLimits{65536, 1, 1, 1, 8192, 1024, 0, 1})
	block := uint64(reflect.TypeFor[providerHistoryCollectionBlock]().Size())
	if !ok || got.normalization != (candidateWorkVector{}) || got.execution != (candidateWorkVector{}) || got.preflight != (candidateWorkVector{}) ||
		got.collection != (candidateWorkVector{10: 5370 + 60*block}) || got.total != got.collection || got.blockingBatches != 0 {
		t.Fatal("zero selection removed receipt scans/pending arrays or invented trace work", got, ok)
	}
	for _, q := range []uint64{1, 31, 32, 33, 40} {
		got, ok := providerHistoryEffectBranchUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 2, Windows: 1},
			providerHistoryEffectBranchLimits{65536, 40, 80, 40, 8192, 1024, q, 2})
		want := uint64(1)
		if q > 32 {
			want = 2
		}
		if !ok || got.blockingBatches != want {
			t.Fatal("nonempty original32-member group ceiling", q, got, ok)
		}
	}
}

func TestProviderHistoryEffectBranchInvalidDomainsReturnWholeZero(t *testing.T) {
	for _, mode := range []string{"groups", "requests", "blocks", "windows", "versions", "stops", "evidence-small", "evidence-wide",
		"transactions-zero", "transactions-wide", "frames-zero", "frames-wide", "logs-zero", "logs-wide",
		"body-small", "body-wide", "value-zero", "value-wide", "effects-wide", "content-zero", "content-wide"} {
		l := providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1}
		p := providerHistoryEffectBranchLimits{65536, 1, 2, 1, 2048, 1024, 1, 1}
		switch mode {
		case "groups":
			l.Groups = 0
		case "requests":
			l.Requests = 33
		case "blocks":
			l.Blocks = 0
		case "windows":
			l.Windows = 2
		case "versions":
			l.Versions = 1<<20 + 1
		case "stops":
			l.PresentStops = 2
		case "evidence-small":
			p.Evidence = 65535
		case "evidence-wide":
			p.Evidence = 64<<20 + 1
		case "transactions-zero":
			p.Transactions = 0
		case "transactions-wide":
			p.Transactions = 16385
		case "frames-zero":
			p.Frames = 0
		case "frames-wide":
			p.Frames = 65537
		case "logs-zero":
			p.Logs = 0
		case "logs-wide":
			p.Logs = 65537
		case "body-small":
			p.BodyBytes = 3
		case "body-wide":
			p.BodyBytes = 8<<20 + 1
		case "value-zero":
			p.ValueBytes = 0
		case "value-wide":
			p.ValueBytes = p.BodyBytes + 1
		case "effects-wide":
			p.Effects = 2
		case "content-zero":
			p.ContentBlocks = 0
		case "content-wide":
			p.ContentBlocks = 11
		}
		if got, ok := providerHistoryEffectBranchUpper(l, p); ok || got != (providerHistoryEffectBranchCost{}) {
			t.Fatal("invalid effect tuple leaked partial DATA", mode, got, ok)
		}
	}
}

func TestProviderHistoryEffectBranchActualAndLateRoomOverflowReturnWholeZero(t *testing.T) {
	leading := new(big.Int).Mul(big.NewInt(15*138*10000), new(big.Int).Lsh(big.NewInt(1), 46))
	if leading.BitLen() <= 64 || leading.String() != "1456633004477644800000" {
		t.Fatal("independent leading trace quadratic overflow", leading)
	}
	got, ok := providerHistoryEffectBranchUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 10000, Windows: 10000},
		providerHistoryEffectBranchLimits{64 << 20, 16384, 65536, 65536, 8 << 20, 8 << 20, 10000, 10})
	if ok || got != (providerHistoryEffectBranchCost{}) {
		t.Fatal("wide actual overflow leaked partial metadata/vector", got, ok)
	}
	// Numeric C4 does not assert that a native transaction/trace fits four
	// bytes. Actual arithmetic fits; only original unconsumed room overflows.
	got, ok = providerHistoryEffectBranchUpper(providerHistoryCollectionSemanticLimits{Groups: 10000, Requests: 10000, Blocks: 1, Windows: 1},
		providerHistoryEffectBranchLimits{65536, 1, 1, 1, 4, 1, 1, 1})
	if ok || got != (providerHistoryEffectBranchCost{}) {
		t.Fatal("late signing reserve leaked the actual vector", got, ok)
	}
}
