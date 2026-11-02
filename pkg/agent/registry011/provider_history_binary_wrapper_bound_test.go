package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"reflect"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

type historyBinaryWrapperOriginal struct {
	block          *types.Block
	binary, header []byte
	raw            ReplayBlock
	withdrawals    [][]byte
}

func historyBinaryWrapperRecorded(t *testing.T) historyBinaryWrapperOriginal {
	t.Helper()
	block, _, _, receipts := nodeJSONFixtureBytes(t)
	binary, err := rlp.EncodeToBytes(block)
	if err != nil {
		t.Fatal(err)
	}
	header, err := rlp.EncodeToBytes(block.Header())
	if err != nil {
		t.Fatal(err)
	}
	out := historyBinaryWrapperOriginal{block: block, binary: binary, header: header,
		raw: ReplayBlock{HeaderRLP: header, Receipts: receipts}}
	for _, tx := range block.Transactions() {
		value, err := tx.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		out.raw.Transactions = append(out.raw.Transactions, value)
	}
	for _, withdrawal := range block.Withdrawals() {
		value, err := rlp.EncodeToBytes(withdrawal)
		if err != nil {
			t.Fatal(err)
		}
		out.withdrawals = append(out.withdrawals, value)
	}
	return out
}

// The existing successful empty native-gas schema, not a simulated peer or
// a forged transaction. Strict replay is checked before the copy tests.
func historyBinaryWrapperEmptyGas(t *testing.T) ReplayBlock {
	t.Helper()
	h := &types.Header{Number: new(big.Int), Difficulty: new(big.Int), Time: sepoliaAmsterdamTime, GasLimit: 1000000,
		TxHash: types.EmptyTxsHash, ReceiptHash: types.EmptyReceiptsHash, UncleHash: types.EmptyUncleHash}
	raw, err := rlp.EncodeToBytes(h)
	if err != nil {
		t.Fatal(err)
	}
	return ReplayBlock{HeaderRLP: raw, Gas: &ReplayGasEvidence{SepoliaAmsterdamGasProfile, replayHeader(h).Hash, json.RawMessage(`[]`)}}
}

func TestProviderHistoryBinaryWrapperUpperMatchesIndependentPortablePolynomials(t *testing.T) {
	word := uint64(strconv.IntSize / 8)
	body := uint64(reflect.TypeFor[providerHistoryNativeBlockBody]().Size())
	gas := uint64(reflect.TypeFor[ReplayGasEvidence]().Size())
	if word > 16 || body > 4096 || gas > 4096 {
		t.Fatal("changed source shapes require rederiving the portable literals")
	}
	for _, tc := range []struct {
		input     [9]uint64 // G,R,B,W,E,T,C,V,native.
		encoding  string
		shape     [8]uint64 // F,FW,FB,FN,b,h,bodyPool,headerPool.
		constants [7]uint64 // common,tx,retain,builder,headroom,actual,total.
		coeff     [5]uint64 // word(common/tx/retain),body-size,gas-size.
		decoded   uint64
	}{
		{[9]uint64{1, 1, 1, 1, 65536, 1, 2048, 1024, 0}, EthereumJSONBlocks, [8]uint64{15, 15, 15, 0, 2048, 2048, 2048, 2048},
			[7]uint64{630104760, 1006632975, 1966230, 62914590, 64424509455, 1701618555, 66126128010}, [5]uint64{720, 45, 150, 15, 0}, 153600},
		{[9]uint64{2, 3, 5, 2, 65536, 2, 8192, 1024, 3}, EthereumJSONBlocks, [8]uint64{29, 58, 145, 87, 8192, 8192, 40960, 40960},
			[7]uint64{97324251720, 3892314228, 7603771, 9730785570, 249108103226, 110954955289, 360063058515}, [5]uint64{6960, 348, 1160, 145, 87}, 5939200},
		{[9]uint64{1, 32, 3, 3, 64 << 20, 16384, 8 << 20, 8 << 20, 3}, EthereumJSONBlocks, [8]uint64{15, 45, 45, 45, 8 << 20, 65536, 25165824, 196608},
			[7]uint64{19000335920506920, 25332747904696320, 6042747285, 3166593487994970, 3166593487994925, 47499683355945495, 50666276843940420}, [5]uint64{2160, 2211840, 7372800, 45, 45}, 1512898560},
		{[9]uint64{1, 1, 100, 1, 65536, 2, 8192, 1024, 10}, EthereumJSONBlocks, [8]uint64{15, 15, 1500, 150, 8192, 8192, 145546, 65536},
			[7]uint64{139625824290, 1006632990, 1975650, 17884695480, 64424509455, 158519128410, 222943637865}, [5]uint64{72000, 90, 300, 1500, 150}, 9715800},
		{[9]uint64{1, 1, 1, 1, 65536, 1, 2048, 1024, 1}, "debug-raw-rlp-v1", [8]uint64{15, 15, 15, 15, 2048, 2048, 2048, 2048},
			[7]uint64{1219843800, 1006632975, 1966275, 0, 64424509455, 2228443050, 66652952505}, [5]uint64{737280, 45, 150, 0, 15}, 153600},
		{[9]uint64{2, 3, 5, 2, 65536, 2, 8192, 1024, 3}, "debug-raw-rlp-v1", [8]uint64{29, 58, 145, 87, 8192, 8192, 40960, 40960},
			[7]uint64{103025062440, 3892314228, 7603771, 0, 249108103226, 106924980439, 356033083665}, [5]uint64{7127040, 348, 1160, 0, 87}, 5939200},
	} {
		i := tc.input
		got, ok := providerHistoryBinaryWrapperUpper(providerHistoryCollectionSemanticLimits{Groups: i[0], Requests: i[1], Blocks: i[2], Windows: i[3]}, i[4], i[5], i[6], i[7], i[8], tc.encoding)
		shape := [8]uint64{got.journeys, got.windows, got.blocks, got.native, got.binaryBytes, got.headerBytes, got.bodyPool, got.headerPool}
		common := tc.constants[0] + tc.coeff[0]*word
		transaction := tc.constants[1] + tc.coeff[1]*word
		retention := tc.constants[2] + tc.coeff[2]*word + tc.coeff[4]*gas
		builder := tc.constants[3] + tc.coeff[3]*body
		typeUnits := (tc.coeff[0]+tc.coeff[1]+tc.coeff[2])*word + tc.coeff[3]*body + tc.coeff[4]*gas
		if !ok || shape != tc.shape || got.common != (candidateWorkVector{2: tc.decoded, 10: common}) ||
			got.transaction != (candidateWorkVector{10: transaction}) || got.retention != (candidateWorkVector{10: retention}) ||
			got.builder != (candidateWorkVector{10: builder}) || got.preflight != (candidateWorkVector{10: tc.constants[4]}) ||
			got.actual != (candidateWorkVector{2: tc.decoded, 10: tc.constants[5] + typeUnits}) ||
			got.total != (candidateWorkVector{2: tc.decoded, 10: tc.constants[6] + typeUnits}) {
			t.Fatal("independent full wrapper/copy polynomials", got, ok)
		}
	}
}

func TestProviderHistoryBinaryWrapperNativeHeaderHasExactCodecAndLateEqualityRows(t *testing.T) {
	f := historyBinaryWrapperRecorded(t)
	payload, rest, err := rlp.SplitList(f.binary)
	if err != nil || len(rest) != 0 {
		t.Fatal("ordinary canonical body", err)
	}
	b, d, h := uint64(len(f.binary)), uint64(len(payload)), uint64(len(f.header))
	units, decoded := b*b+d*d+2*h*h+2*h+5, b+d+h
	for _, short := range []bool{false, true} {
		upper := candidateWorkVector{2: decoded, 10: units}
		if short {
			upper[10]--
		}
		m := providerHistoryContentUnitMeter(upper)
		budget := replayBudget{limits: replayTestLimits(), observer: m}
		out, err := providerHistoryNativeHeaderObserved(context.Background(), f.binary, &budget)
		if !short {
			if err != nil || out == nil || out.Hash() != f.block.Hash() || m.actual != upper || budget.bytes != 0 {
				t.Fatal("actual independent native-header row", err, m.actual, upper)
			}
			continue
		}
		want := candidateWorkVector{2: decoded, 10: units - (2*h + 1)}
		if err == nil || out != nil || !m.failed || m.actual != want || budget.bytes != 0 {
			t.Fatal("last native-header comparison lost entered original work", err, m.actual, want)
		}
		if _, retry := providerHistoryNativeHeaderObserved(context.Background(), f.binary, &budget); retry == nil || m.actual != want {
			t.Fatal("failed original native-header observer resumed")
		}
	}
}

func TestProviderHistoryBinaryWrapperJSONBodyKeepsCanonicalBytesAndOriginalPreflight(t *testing.T) {
	f := historyBinaryWrapperRecorded(t)
	if len(f.block.Uncles()) != 0 || len(f.block.Withdrawals()) > 16 {
		t.Fatal("ordinary existing JSON body precursor")
	}
	var canonical uint64 = uint64(len(f.header))
	for _, raw := range f.raw.Transactions {
		canonical += uint64(len(raw))
	}
	if uint64(len(f.binary)) > canonical+5*uint64(len(f.raw.Transactions))+800 {
		t.Fatal("original canonical JSON body exceeds the independent enclosing-byte relation")
	}
	body := uint64(reflect.TypeFor[providerHistoryNativeBlockBody]().Size())
	b := uint64(len(f.binary))
	units := body + b*b + 2
	for _, short := range []bool{false, true} {
		upper := candidateWorkVector{10: units}
		if short {
			upper[10]--
		}
		m := providerHistoryContentUnitMeter(upper)
		header := f.block.Header()
		out, err := providerHistoryNativeBlockBytesObserved(header, f.block.Transactions(), f.block.Withdrawals(), len(f.binary), m)
		if !short {
			if err != nil || !bytes.Equal(out, f.binary) || m.actual != upper {
				t.Fatal("original JSON body changed canonical encoding or explicit copies", err, m.actual)
			}
		} else {
			want := candidateWorkVector{10: body + 1}
			if err == nil || out != nil || !m.failed || m.actual != want {
				t.Fatal("body encode preflight refusal lost its original struct copy", err, m.actual, want)
			}
			if _, retry := providerHistoryNativeBlockBytesObserved(header, f.block.Transactions(), f.block.Withdrawals(), len(f.binary), m); retry == nil || m.actual != want {
				t.Fatal("failed body observer resumed or refunded its prefix")
			}
		}
		after, err := rlp.EncodeToBytes(header)
		if err != nil || !bytes.Equal(after, f.header) {
			t.Fatal("borrowed original header was changed", err)
		}
	}
}

func TestProviderHistoryBinaryWrapperMeasuresActualCommonDebugRouteSeparately(t *testing.T) {
	f := historyBinaryWrapperRecorded(t)
	blockResult, err := json.Marshal(nodeHex(f.binary))
	if err != nil {
		t.Fatal(err)
	}
	texts := make([]string, len(f.raw.Receipts))
	for i, raw := range f.raw.Receipts {
		texts[i] = nodeHex(raw)
	}
	receiptResult, err := json.Marshal(texts)
	if err != nil {
		t.Fatal(err)
	}
	input := historyCollectionInput(t)
	input.encoding, input.journal.Replay = "debug-raw-rlp-v1", nodeJSONLimits().Replay
	input.capture.HeaderBytes, input.capture.BodyBytes, input.capture.WireBytes, input.capture.JSONDepth = 64<<10, 8<<20, 16<<20, 128
	m := providerHistorySemanticUnitMeter()
	before := m.actual
	if _, err := rpcDataObserved(blockResult, m); err != nil {
		t.Fatal("ordinary original block hex branch", err)
	}
	rows, ok := nodeRawArrayObserved(receiptResult, input.journal.Replay.MaxTransactions, m)
	if !ok || !nodeJSONSliceObserved(len(rows), 3, m) {
		t.Fatal("ordinary original receipt branch")
	}
	for _, row := range rows {
		if _, err := rpcDataObserved(row, m); err != nil {
			t.Fatal("ordinary original receipt hex", err)
		}
	}
	budget := replayBudget{limits: input.journal.Replay, observer: m}
	if _, err := replayDecodeContent(context.Background(), f.raw, &budget); err != nil {
		t.Fatal("separate original core and its two trie computations", err)
	}
	header := f.block.Header()
	if header.WithdrawalsHash != nil {
		if _, err := providerHistoryContentRootObserved(context.Background(), f.withdrawals, m); err != nil {
			t.Fatal("separate original withdrawal trie computation", err)
		}
	}
	excluded := historyEventQueueDelta(t, m, before)
	payload, _, err := rlp.SplitList(f.binary)
	if err != nil {
		t.Fatal(err)
	}
	uncles, err := rlp.EncodeToBytes(f.block.Uncles())
	if err != nil {
		t.Fatal(err)
	}
	q := func(n uint64) uint64 { return n*n + 1 }
	b, d, h, u := uint64(len(f.binary)), uint64(len(payload)), uint64(len(f.header)), uint64(len(uncles))
	word := uint64(strconv.IntSize / 8)
	units := uint64(len(blockResult)+len(receiptResult)) + 2 + 4*q(b) + q(d) + 4*q(h) + 3*h + 2 + 8980 + 2*b + 1 + q(u) + u + 66
	units += 3*uint64(len(f.raw.Transactions))*word + 1
	for _, raw := range f.raw.Transactions {
		units += q(uint64(len(raw)))
	}
	if header.WithdrawalsHash != nil {
		units += 3*uint64(len(f.withdrawals))*word + 66
		for _, raw := range f.withdrawals {
			units += q(uint64(len(raw)))
		}
	}
	common := candidateWorkVector{2: 3*b + d + h, 10: units}
	want := common
	for i, n := range excluded {
		want[i] += n
	}
	before = m.actual
	out, decoded, err := providerHistoryCollectionDecodeBlockObserved(context.Background(), string(blockResult), string(receiptResult), replayHeader(header), input, m)
	if err != nil || decoded.header == nil || !bytes.Equal(out.HeaderRLP, f.header) || len(out.Transactions) != len(f.raw.Transactions) ||
		len(out.Receipts) != len(f.raw.Receipts) || historyEventQueueDelta(t, m, before) != want {
		t.Fatal("actual common decoder lost a native observation or borrowed a separate row", err, historyEventQueueDelta(t, m, before), want)
	}
	for i := range out.Transactions {
		if !bytes.Equal(out.Transactions[i], f.raw.Transactions[i]) || !bytes.Equal(out.Receipts[i], f.raw.Receipts[i]) {
			t.Fatal("original canonical array bytes changed")
		}
	}
	l := input.journal.Replay
	upper, quoted := providerHistoryBinaryWrapperUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1},
		uint64(l.MaxEvidenceBytes), uint64(l.MaxTransactions), uint64(input.capture.BodyBytes), uint64(input.capture.BodyBytes), 0, input.encoding)
	if !quoted {
		t.Fatal("ordinary prospective common-wrapper ceiling")
	}
	historyEventQueueWithin(t, common, upper.common)
}

func TestProviderHistoryBinaryWrapperRetainReplayHasExactOwnedCopyRows(t *testing.T) {
	for _, raw := range []ReplayBlock{replayTestInput(t).Blocks[0], historyBinaryWrapperEmptyGas(t)} {
		m := providerHistorySemanticUnitMeter()
		budget := replayBudget{limits: replayTestLimits(), observer: m}
		decoded, err := replayDecodeBlock(context.Background(), raw, &budget)
		if err != nil {
			t.Fatal("unchanged strict ordinary block/gas precursor", err)
		}
		header := replayHeader(decoded.header)
		shape := historyBinaryCoreOriginalShape(t, raw)
		word := uint64(strconv.IntSize / 8)
		units := 2*shape.bytes + (10*word+4)*shape.transactions + 6
		if raw.Gas != nil {
			units += 2*uint64(len(raw.Gas.Trace)) + uint64(reflect.TypeFor[ReplayGasEvidence]().Size()) + 3
		}
		before := m.actual
		retained, ok := providerHistoryCollectionRetainBlockObserved(raw, header, m)
		if !ok {
			t.Fatal("ordinary immutable retain")
		}
		out, ok := retained.replayObserved(m)
		if !ok || !bytes.Equal(out.HeaderRLP, raw.HeaderRLP) || len(out.Transactions) != len(raw.Transactions) ||
			len(out.Receipts) != len(raw.Receipts) || historyEventQueueDelta(t, m, before) != (candidateWorkVector{10: units}) {
			t.Fatal("exact original retain/replay copy rows", m.actual, units)
		}
		for i := range out.Transactions {
			if !bytes.Equal(out.Transactions[i], raw.Transactions[i]) || !bytes.Equal(out.Receipts[i], raw.Receipts[i]) {
				t.Fatal("retained canonical array bytes changed")
			}
		}
		if raw.Gas == nil && out.Gas != nil || raw.Gas != nil && (out.Gas == nil || out.Gas.Profile != raw.Gas.Profile || out.Gas.BlockHash != raw.Gas.BlockHash || !bytes.Equal(out.Gas.Trace, raw.Gas.Trace)) {
			t.Fatal("immutable block/gas bytes changed")
		}
		original := string(raw.HeaderRLP)
		out.HeaderRLP[0] ^= 1 // Own returned buffer; no peer or decoded reuse.
		if retained.headerRLP != original || string(raw.HeaderRLP) != original {
			t.Fatal("replayed bytes alias an original or retained immutable string")
		}
	}
}

func TestProviderHistoryBinaryWrapperFinalGasCopyShortageKeepsTheOriginalPrefix(t *testing.T) {
	raw := historyBinaryWrapperEmptyGas(t)
	validation := replayBudget{limits: replayTestLimits(), observer: providerHistorySemanticUnitMeter()}
	decoded, err := replayDecodeBlock(context.Background(), raw, &validation)
	if err != nil {
		t.Fatal("normal successful strict gas precursor", err)
	}
	g := uint64(len(raw.Gas.Trace))
	units := 2*uint64(len(raw.HeaderRLP)) + 2*g + uint64(reflect.TypeFor[ReplayGasEvidence]().Size()) + 9
	m := providerHistoryContentUnitMeter(candidateWorkVector{10: units - 1})
	retained, ok := providerHistoryCollectionRetainBlockObserved(raw, replayHeader(decoded.header), m)
	if !ok {
		t.Fatal("ordinary retention precedes the final replay copy")
	}
	out, ok := retained.replayObserved(m)
	want := candidateWorkVector{10: units - (g + 1)}
	if ok || out.HeaderRLP != nil || out.Gas != nil || !m.failed || m.actual != want || retained.gasTrace != string(raw.Gas.Trace) || retained.headerRLP != string(raw.HeaderRLP) {
		t.Fatal("last owned gas-copy refusal lost its prefix or changed retained input", m.actual, want)
	}
	if _, retry := retained.replayObserved(m); retry || m.actual != want {
		t.Fatal("failed original replay-copy observer resumed")
	}
}

func TestProviderHistoryBinaryWrapperUpperInvalidDomainsReturnWholeZero(t *testing.T) {
	base := providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1}
	for _, mode := range []string{"groups", "requests", "blocks", "windows", "evidence-zero", "evidence-header-preflight", "evidence-wide", "tx-zero", "tx-wide", "body-zero", "body-wide", "value-zero", "value-wide", "native-wide", "encoding"} {
		l, evidence, txs, body, value, native, encoding := base, uint64(65536), uint64(1), uint64(2048), uint64(1024), uint64(1), EthereumJSONBlocks
		switch mode {
		case "groups":
			l.Groups = 0
		case "requests":
			l.Requests = 33
		case "blocks":
			l.Blocks = 0
		case "windows":
			l.Windows = 2
		case "evidence-zero":
			evidence = 0
		case "evidence-header-preflight":
			evidence = 65535
		case "evidence-wide":
			evidence = 64<<20 + 1
		case "tx-zero":
			txs = 0
		case "tx-wide":
			txs = 16385
		case "body-zero":
			body = 0
		case "body-wide":
			body = 8<<20 + 1
		case "value-zero":
			value = 0
		case "value-wide":
			value = body + 1
		case "native-wide":
			native = 2
		case "encoding":
			encoding = "unselected"
		}
		if got, ok := providerHistoryBinaryWrapperUpper(l, evidence, txs, body, value, native, encoding); ok || got != (providerHistoryBinaryWrapperCost{}) {
			t.Fatal("invalid wrapper domain returned partial costs", mode, got, ok)
		}
	}
}

func TestProviderHistoryBinaryWrapperUpperLateLogicalOverflowReturnsWholeZero(t *testing.T) {
	minimum, ok := new(big.Int).SetString("63331869759897600000", 10)
	if !ok || minimum.BitLen() <= 64 {
		t.Fatal("independent repeated body-encoder overflow")
	}
	// Just6*b*bodyPool*F already overflows; decoded2 remains5042995200000.
	// This numeric tuple does not certify a native profile or operating budget.
	got, admitted := providerHistoryBinaryWrapperUpper(providerHistoryCollectionSemanticLimits{
		Groups: 1, Requests: 1, Blocks: 10000, Windows: 10000,
	}, 64<<20, 16384, 8<<20, 8<<20, 0, "debug-raw-rlp-v1")
	if admitted || got != (providerHistoryBinaryWrapperCost{}) {
		t.Fatal("late logical overflow leaked decoded counts or partial costs", got, admitted)
	}
}
