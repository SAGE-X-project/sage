package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestProviderHistoryJSONBranchUpperMatchesIndependentWholePolynomials(t *testing.T) {
	w := uint64(strconv.IntSize / 8)
	for _, row := range []struct {
		input                         [9]uint64 // G,R,B,W,E,T,C,V,L.
		shape                         [6]uint64 // F,FW,FB,H,extremal block,extremal receipt.
		decoded, constant, word, room uint64
	}{
		{[9]uint64{1, 1, 1, 1, 65536, 1, 2048, 1024, 0}, [6]uint64{15, 15, 15, 2048, 2048, 2048}, 1667535, 9720752430, 923730, 67554509807616015},
		{[9]uint64{2, 3, 5, 2, 65536, 2, 8192, 1024, 3}, [6]uint64{29, 58, 145, 8192, 40960, 40960}, 48132692, 485972362401, 35659560, 261210771256115258},
		{[9]uint64{1, 1, 2, 1, 65536, 1, 131072, 1024, 1}, [6]uint64{15, 15, 30, 65536, 131072, 0}, 48275295, 6729100103340, 35393850, 67554509807616015},
		{[9]uint64{1, 32, 3, 3, 67108864, 16384, 8388608, 8388608, 65536}, [6]uint64{15, 45, 45, 65536, 25165824, 25165824}, 15432114240, 248586137586552375, 11405727360, 202663529422848045},
	} {
		i := row.input
		got, ok := providerHistoryJSONBranchUpper(providerHistoryCollectionSemanticLimits{Groups: i[0], Requests: i[1], Blocks: i[2], Windows: i[3]}, i[4], i[5], i[8], i[6], i[7], EthereumJSONBlocks)
		actual := candidateWorkVector{2: row.decoded, 3: row.shape[2], 4: row.shape[1] * i[5], 6: row.shape[1] * i[8], 10: row.constant + row.word*w}
		total := actual
		total[10] += row.room
		if !ok || got.actual != actual || got.preflight != (candidateWorkVector{10: row.room}) || got.total != total || got.arraySlots != 0 ||
			([6]uint64{got.journeys, got.windows, got.blocks, got.headerBytes, got.extremalBlock, got.extremalReceipt}) != row.shape {
			t.Fatal("independent original JSON branch rows", got, ok, actual)
		}
	}
}

func TestProviderHistoryJSONBranchDebugUpperMatchesIndependentPortableRows(t *testing.T) {
	w := uint64(strconv.IntSize / 8)
	for _, row := range []struct {
		input                          [8]uint64 // G,R,B,W,E,T,C,V.
		slots, decoded, constant, word uint64
	}{
		{[8]uint64{1, 1, 1, 1, 65536, 1, 2048, 1024}, 1, 153540, 314388810, 135},
		{[8]uint64{2, 3, 5, 2, 65536, 2, 8192, 1024}, 2, 5938620, 48646802666, 2088},
		{[8]uint64{1, 1, 2, 1, 65536, 1, 131072, 1024}, 1, 19660680, 2576956785285, 225},
	} {
		i := row.input
		got, ok := providerHistoryJSONBranchUpper(providerHistoryCollectionSemanticLimits{Groups: i[0], Requests: i[1], Blocks: i[2], Windows: i[3]}, i[4], i[5], 0, i[6], i[7], "debug-raw-rlp-v1")
		want := candidateWorkVector{2: row.decoded, 10: row.constant + row.word*w}
		if !ok || got.actual != want || got.total != want || got.preflight != (candidateWorkVector{}) || got.arraySlots != row.slots || got.headerBytes != 0 || got.extremalBlock != 0 || got.extremalReceipt != 0 {
			t.Fatal("independent debug alternative", got, ok, want)
		}
	}
	// This finite arithmetic tuple checks the original raw-length slot cap;
	// it does not claim that a four-byte capture can contain a native block.
	got, ok := providerHistoryJSONBranchUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1}, 65536, 16, 0, 4, 1, "debug-raw-rlp-v1")
	if !ok || got.arraySlots != 1 || got.actual != (candidateWorkVector{2: 240, 10: 1395 + 810*w}) {
		t.Fatal("raw-length slot clipping", got, ok)
	}
}

func TestProviderHistoryJSONBranchOriginalBlockAndReceiptsKeepCumulativeObservations(t *testing.T) {
	block, blockJSON, receiptJSON, receipts := nodeJSONFixtureBytes(t)
	header := block.Header()
	canonical, err := rlp.EncodeToBytes(block)
	if err != nil {
		t.Fatal("ordinary canonical block", err)
	}
	var txs, withdrawals [][]byte
	for _, tx := range block.Transactions() {
		raw, err := tx.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		txs = append(txs, raw)
	}
	for _, v := range block.Withdrawals() {
		raw, err := rlp.EncodeToBytes(v)
		if err != nil {
			t.Fatal(err)
		}
		withdrawals = append(withdrawals, raw)
	}
	var logs uint64
	for _, raw := range receipts {
		var v types.Receipt
		if v.UnmarshalBinary(raw) != nil {
			t.Fatal("ordinary receipt")
		}
		logs += uint64(len(v.Logs))
	}
	m := providerHistorySemanticUnitMeter()
	// Observe the disjoint body builder and two indexed tries once, on the
	// SAME original meter. The target repeats that prefix once per pass.
	// Subtraction below compares totals; it never edits a quota or actual.
	if _, err := providerHistoryNativeBlockBytesObserved(header, block.Transactions(), block.Withdrawals(), len(blockJSON), m); err != nil {
		t.Fatal(err)
	}
	for _, rows := range [][][]byte{txs, withdrawals} {
		if _, err := providerHistoryContentRootObserved(context.Background(), rows, m); err != nil {
			t.Fatal(err)
		}
	}
	prefix := m.actual
	if prefix[10] > 1<<50 {
		t.Fatal("ordinary exclusion prefix no longer bounded")
	}
	for i, n := range prefix {
		if i != 10 && n != 0 {
			t.Fatal("excluded component has another coordinate", prefix)
		}
	}
	l := nodeJSONLimits().Replay
	upper, ok := providerHistoryJSONBranchUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1}, uint64(l.MaxEvidenceBytes), uint64(l.MaxTransactions), uint64(l.MaxLogs), uint64(max(len(blockJSON), len(receiptJSON))), uint64(max(len(blockJSON), len(receiptJSON))), EthereumJSONBlocks)
	if !ok {
		t.Fatal("ordinary prospective branch ceiling")
	}
	for pass := uint64(1); pass <= 2; pass++ {
		budget := replayBudget{limits: l, observer: m}
		got, err := nodeJSONBlock(context.Background(), blockJSON, replayHeader(header), &budget)
		if err != nil || !bytes.Equal(got, canonical) {
			t.Fatal("original JSON block consensus bytes", err)
		}
		out, err := nodeJSONReceiptsHeaderObserved(context.Background(), receiptJSON, block, header, &budget)
		if err != nil || len(out) != len(receipts) {
			t.Fatal("original JSON receipt consensus bytes", err)
		}
		for i, raw := range out {
			if !bytes.Equal(raw, receipts[i]) {
				t.Fatal("original receipt bytes changed")
			}
		}
		branch := m.actual
		excluded := (pass + 1) * prefix[10]
		if branch[10] < excluded {
			t.Fatal("actual exclusion prefix missing")
		}
		branch[10] -= excluded
		if m.failed || branch[3] != pass || branch[4] != pass*uint64(len(txs)) || branch[6] != pass*logs {
			t.Fatal("original branch counters were reset", branch)
		}
		historyEventQueueWithin(t, branch, upper.actual)
	}
}

func TestProviderHistoryJSONBranchDebugOriginalCodecsHaveExactCumulativeRows(t *testing.T) {
	f := historyBinaryWrapperRecorded(t)
	blockJSON, err := json.Marshal(nodeHex(f.binary))
	if err != nil {
		t.Fatal(err)
	}
	texts := make([]string, len(f.raw.Receipts))
	for i, raw := range f.raw.Receipts {
		texts[i] = nodeHex(raw)
	}
	receiptJSON, err := json.Marshal(texts)
	if err != nil {
		t.Fatal(err)
	}
	q := func(n uint64) uint64 { return n*n + 1 }
	w := uint64(strconv.IntSize / 8)
	n := uint64(len(texts))
	b := uint64(len(f.binary))
	r := uint64(len(receiptJSON))
	slots := min(uint64(16384), (r-1)/2)
	want := candidateWorkVector{2: uint64(len(blockJSON)) + 2*b + r, 10: q(uint64(len(blockJSON))) + 2*b + 3 + q(2*b) + q(r) + 3*w*slots + 1 + (6*w+1)*n + 1}
	for _, raw := range f.raw.Receipts {
		v := uint64(len(raw))
		row, err := json.Marshal(nodeHex(raw))
		if err != nil {
			t.Fatal(err)
		}
		want[2] += uint64(len(row)) + 2*v
		want[10] += q(uint64(len(row))) + 2*v + 3 + q(2*v)
	}
	m := providerHistorySemanticUnitMeter()
	for pass := uint64(1); pass <= 2; pass++ {
		got, err := rpcDataObserved(blockJSON, m)
		if err != nil || !bytes.Equal(got, f.binary) {
			t.Fatal("original block hex", err)
		}
		rows, ok := nodeRawArrayObserved(receiptJSON, 16384, m)
		if !ok || !nodeJSONSliceObserved(len(rows), 3, m) {
			t.Fatal("original receipt array")
		}
		for i, row := range rows {
			got, err := rpcDataObserved(row, m)
			if err != nil || !bytes.Equal(got, f.raw.Receipts[i]) {
				t.Fatal("original receipt hex", err)
			}
		}
		var exact candidateWorkVector
		for i, n := range want {
			exact[i] = pass * n
		}
		if m.failed || m.actual != exact {
			t.Fatal("exact original debug rows", m.actual, exact)
		}
	}
	upper, ok := providerHistoryJSONBranchUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1}, 64<<20, 16384, 65536, uint64(max(len(blockJSON), len(receiptJSON))), uint64(max(len(blockJSON), len(receiptJSON))), "debug-raw-rlp-v1")
	if !ok {
		t.Fatal("debug prospective branch ceiling")
	}
	historyEventQueueWithin(t, m.actual, upper.actual)
}

func TestProviderHistoryJSONBranchTransactionLocationPricesTheAbsentFirstAlias(t *testing.T) {
	block, raw, _, _ := nodeJSONFixtureBytes(t)
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil {
		t.Fatal("original block JSON")
	}
	rows, ok := nodeRawArray(object["transactions"], 16384)
	if !ok || len(rows) == 0 {
		t.Fatal("original transaction")
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(rows[0], &fields) != nil {
		t.Fatal("original location")
	}
	if _, present := fields["transactionHash"]; present {
		t.Fatal("ordinary transaction uses hash, not transactionHash")
	}
	want := candidateWorkVector{10: 8981} // two hash texts/comparisons + missing string attempt.
	for _, key := range []string{"blockHash", "hash", "blockNumber", "transactionIndex"} {
		n := uint64(len(fields[key]))
		want[2] += n
		want[10] += n*n + n + 2
	}
	for _, key := range []string{"blockNumber", "transactionIndex"} {
		var text string
		if json.Unmarshal(fields[key], &text) != nil || len(text) < 3 {
			t.Fatal("ordinary quantity")
		}
		d := uint64(len(text) - 2)
		want[2] += d
		want[10] += 2*d*d + 3*d + 6
	}
	m := providerHistorySemanticUnitMeter()
	for pass := uint64(1); pass <= 2; pass++ {
		if !nodeJSONLocationObserved(fields, block.Hash(), block.NumberU64(), block.Transactions()[0].Hash(), 0, m) {
			t.Fatal("ordinary bound location")
		}
		var exact candidateWorkVector
		for i, n := range want {
			exact[i] = pass * n
		}
		if m.actual != exact {
			t.Fatal("missing original alias attempt was lost", m.actual, exact)
		}
	}
}

func TestProviderHistoryJSONBranchNestedListsUseDisjointOriginalRowPools(t *testing.T) {
	w := uint64(strconv.IntSize / 8)
	for _, tx := range providerHistoryNodeJSONUnitTransactions() {
		raw, err := json.Marshal(tx)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]json.RawMessage
		if json.Unmarshal(raw, &fields) != nil {
			t.Fatal("ordinary typed fields")
		}
		n := uint64(len(raw))
		m := providerHistorySemanticUnitMeter()
		if !nodeJSONTransactionListsObserved(fields, m) || m.failed {
			t.Fatal("existing finite typed lists")
		}
		historyEventQueueWithin(t, m.actual, candidateWorkVector{2: 5 * n, 10: (5*n+12*w+515)*n + 24})
	}
}

func TestProviderHistoryJSONBranchEarlyCodecShortagesKeepOriginalScans(t *testing.T) {
	block, raw, receipts, _ := nodeJSONFixtureBytes(t)
	for _, receipt := range []bool{false, true} {
		input := raw
		if receipt {
			input = receipts
		}
		n := uint64(len(input))
		prefix := candidateWorkVector{10: 2*n + 2}
		m := providerHistoryContentUnitMeter(prefix)
		budget := replayBudget{limits: nodeJSONLimits().Replay, observer: m}
		for attempt := 0; attempt < 2; attempt++ {
			if receipt {
				out, err := nodeJSONReceiptsHeaderObserved(context.Background(), input, block, block.Header(), &budget)
				if err == nil || out != nil {
					t.Fatal("receipt codec shortage returned partial DATA")
				}
			} else {
				out, err := nodeJSONBlock(context.Background(), input, replayHeader(block.Header()), &budget)
				if err == nil || out != nil {
					t.Fatal("block codec shortage returned partial DATA")
				}
			}
			if !m.failed || m.actual != prefix {
				t.Fatal("entered scans were refunded or a failed observer resumed", m.actual, prefix)
			}
		}
	}
}

func TestProviderHistoryJSONBranchInvalidDomainsReturnEntirelyZero(t *testing.T) {
	for _, mode := range []string{"groups", "requests", "blocks", "windows", "versions", "stops", "evidence-small", "evidence-wide", "tx-zero", "tx-wide", "logs-wide", "body-small", "body-wide", "value-zero", "value-wide", "encoding"} {
		l := providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1}
		e, tcount, logs, c, v := uint64(65536), uint64(1), uint64(0), uint64(2048), uint64(1024)
		encoding := EthereumJSONBlocks
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
			e = 65535
		case "evidence-wide":
			e = 64<<20 + 1
		case "tx-zero":
			tcount = 0
		case "tx-wide":
			tcount = 16385
		case "logs-wide":
			logs = 65537
		case "body-small":
			c = 3
		case "body-wide":
			c = 8<<20 + 1
		case "value-zero":
			v = 0
		case "value-wide":
			v = c + 1
		case "encoding":
			encoding = "unselected"
		}
		if got, ok := providerHistoryJSONBranchUpper(l, e, tcount, logs, c, v, encoding); ok || got != (providerHistoryJSONBranchCost{}) {
			t.Fatal("invalid branch tuple leaked a partial quote", mode, got, ok)
		}
	}
}

func TestProviderHistoryJSONBranchLateActualAndRoomOverflowsReturnWholeZero(t *testing.T) {
	// The first whole raw-block quadratic contribution alone exceeds uint64.
	minimum, ok := new(big.Int).SetString("263882790666240000000", 10)
	if !ok || minimum.BitLen() <= 64 {
		t.Fatal("independent actual overflow precursor")
	}
	for _, encoding := range []string{EthereumJSONBlocks, "debug-raw-rlp-v1"} {
		got, ok := providerHistoryJSONBranchUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 10000, Windows: 10000}, 64<<20, 16384, 65536, 8<<20, 8<<20, encoding)
		if ok || got != (providerHistoryJSONBranchCost{}) {
			t.Fatal("wide repeated actual leaked partial decoded or shape", got, ok)
		}
	}
	// At this small body/value tuple actual logical work fits uint64; only
	// the required unchanged signing preflight reserve overflows later.
	got, ok := providerHistoryJSONBranchUpper(providerHistoryCollectionSemanticLimits{Groups: 10000, Requests: 10000, Blocks: 1, Windows: 1}, 65536, 1, 0, 4, 1, EthereumJSONBlocks)
	if ok || got != (providerHistoryJSONBranchCost{}) {
		t.Fatal("late unconsumed room overflow leaked an actual row", got, ok)
	}
}
