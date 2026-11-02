package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestProviderHistoryNativeBlockEncodingPreservesCompleteRecordedBytes(t *testing.T) {
	block, _, _, _ := nodeJSONFixtureBytes(t)
	for _, mode := range []string{"recorded-withdrawals", "legacy-no-withdrawals"} {
		t.Run(mode, func(t *testing.T) {
			header, withdrawals := block.Header(), block.Withdrawals()
			if mode == "legacy-no-withdrawals" {
				header.WithdrawalsHash, withdrawals = nil, nil
			}
			original := types.NewBlockWithHeader(header).WithBody(types.Body{Transactions: block.Transactions(), Withdrawals: withdrawals})
			want, err := rlp.EncodeToBytes(original)
			if err != nil {
				t.Fatal(err)
			}
			before := replayTestClone(t, header)
			actual, err := providerHistoryNativeBlockBytesObserved(header, block.Transactions(), withdrawals, len(want), providerHistorySemanticUnitMeter())
			if err != nil || !bytes.Equal(actual, want) || !reflect.DeepEqual(before, header) {
				t.Fatal("transient observed body changed pinned complete RLP or input header", err)
			}
		})
	}
}

func TestProviderHistoryNativeHeaderDecodesIndependentFieldsFromOriginalBytes(t *testing.T) {
	block, _, _, _ := nodeJSONFixtureBytes(t)
	raw, err := rlp.EncodeToBytes(block)
	if err != nil {
		t.Fatal(err)
	}
	want := block.Header()
	budget := replayBudget{limits: replayTestLimits(), observer: providerHistorySemanticUnitMeter()}
	header, err := providerHistoryNativeHeaderObserved(context.Background(), raw, &budget)
	if err != nil || header == nil || !reflect.DeepEqual(header, want) || header.Number == want.Number || budget.observer.actual[2] <= uint64(len(raw)) {
		t.Fatal("header was cached, aliased or decoded without observing its original", err)
	}
	oldNumber := block.NumberU64()
	header.Number.SetUint64(oldNumber + 1)
	if len(header.Extra) > 0 {
		header.Extra[0] ^= 1
	}
	if block.NumberU64() != oldNumber || !reflect.DeepEqual(block.Header(), want) {
		t.Fatal("decoded header mutation changed the original native block")
	}
}

func TestProviderHistoryNativeHeaderRefusesWithoutReturningPartialObjects(t *testing.T) {
	block, _, _, _ := nodeJSONFixtureBytes(t)
	raw, err := rlp.EncodeToBytes(block)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"nil-context", "cancelled", "nil-budget", "nil-observer", "input-ceiling", "non-list", "late-work"} {
		t.Run(mode, func(t *testing.T) {
			meter := providerHistorySemanticUnitMeter()
			budget := &replayBudget{limits: replayTestLimits(), observer: meter}
			ctx, input := context.Background(), raw
			switch mode {
			case "nil-context":
				ctx = nil
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "nil-budget":
				budget = nil
			case "nil-observer":
				budget.observer = nil
			case "input-ceiling":
				budget.limits.MaxEvidenceBytes = len(raw) - 1
			case "non-list":
				input = []byte{0x01}
			case "late-work":
				measured := providerHistorySemanticUnitMeter()
				full := replayBudget{limits: replayTestLimits(), observer: measured}
				if header, normalErr := providerHistoryNativeHeaderObserved(ctx, raw, &full); normalErr != nil || header == nil || measured.actual[10] == 0 {
					t.Fatal("normal measured precursor", normalErr)
				}
				meter.upper[10] = measured.actual[10] - 1
			}
			if header, decodeErr := providerHistoryNativeHeaderObserved(ctx, input, budget); decodeErr == nil || header != nil {
				t.Fatal("unavailable native header leaked a partial object", mode, decodeErr)
			}
			if mode == "late-work" && (!meter.failed || meter.actual[2] == 0 || meter.actual[10] == 0) {
				t.Fatal("late failure restored the original allowance or failed before original interpretation")
			}
		})
	}
}

func TestProviderHistoryNativeABIsPreserveAllFixedSelectorsAndIndependentMaps(t *testing.T) {
	wantContract, wantOutcomes, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	meter := providerHistorySemanticUnitMeter()
	contract, outcomes, err := replayABIsObserved(meter)
	if err != nil || len(contract.Methods) != len(wantContract.Methods) || len(outcomes.Events) != len(wantOutcomes.Events) ||
		meter.actual[2] != uint64(len(replayContractABI)+len(replayOutcomesABI)) {
		t.Fatal("fixed native initialization changed definitions or omitted actual input observation", err)
	}
	for name, want := range wantContract.Methods {
		got, ok := contract.Methods[name]
		if !ok || got.Sig != want.Sig || got.StateMutability != want.StateMutability || !bytes.Equal(got.ID, want.ID) || len(got.Inputs) != len(want.Inputs) || len(got.Outputs) != len(want.Outputs) {
			t.Fatal("method selector/shape changed", name)
		}
	}
	for name, want := range wantOutcomes.Events {
		got, ok := outcomes.Events[name]
		if !ok || got.ID != want.ID || got.Sig != want.Sig || len(got.Inputs) != len(want.Inputs) {
			t.Fatal("event signature/shape changed", name)
		}
	}
	second, _, secondErr := replayABIsObserved(providerHistorySemanticUnitMeter())
	delete(contract.Methods, "commitClaim")
	if secondErr != nil || second.Methods["commitClaim"].Sig != wantContract.Methods["commitClaim"].Sig {
		t.Fatal("independent initializations share a mutable method map", secondErr)
	}
}

func TestProviderHistoryNativeABIsKeepConsumedWorkOnInitializationFailure(t *testing.T) {
	for _, mode := range []string{"first-codec", "second-codec"} {
		t.Run(mode, func(t *testing.T) {
			meter := providerHistorySemanticUnitMeter()
			ceiling := uint64(0)
			if mode == "second-codec" {
				ceiling = uint64(len(replayContractABI))
			}
			meter.upper[2] = ceiling
			contract, outcomes, err := replayABIsObserved(meter)
			if err == nil || contract.Methods != nil || outcomes.Events != nil || !meter.failed || meter.actual[2] != ceiling || meter.actual[10] == 0 {
				t.Fatal("initialization failure returned a partial ABI or refunded original work", err, meter.actual)
			}
		})
	}
}

func TestProviderHistoryNativeBlockEncodingKeepsFailedOriginalAllowance(t *testing.T) {
	block, _, _, _ := nodeJSONFixtureBytes(t)
	header := block.Header()
	want, err := rlp.EncodeToBytes(block)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"output-ceiling", "spent-original"} {
		t.Run(mode, func(t *testing.T) {
			meter := providerHistorySemanticUnitMeter()
			maximum := len(want) - 1
			if mode == "spent-original" {
				maximum = len(want)
				meter.upper = candidateWorkVector{}
			}
			before := replayTestClone(t, header)
			result, encodeErr := providerHistoryNativeBlockBytesObserved(header, block.Transactions(), block.Withdrawals(), maximum, meter)
			if encodeErr == nil || result != nil || !meter.failed || !reflect.DeepEqual(header, before) {
				t.Fatal("failed native encoding published bytes, restored allowance or changed its input", mode, encodeErr)
			}
		})
	}
}

func TestProviderHistoryNativeABIInitializationUsesTheSameObserverInEveryWindow(t *testing.T) {
	fold, input, meter := historySemanticObservedFixture(t, false)
	for _, interval := range [][2]int{{0, 3}, {4, 7}, {8, 11}} {
		before := meter.actual[2]
		historySemanticApply(t, fold, input, interval[0], interval[1])
		if fold.observer != meter || fold.window.budget.observer != meter || meter.actual[2]-before < uint64(len(replayContractABI)+len(replayOutcomesABI)) {
			t.Fatal("new window skipped native initialization or replaced its original observer")
		}
		historySemanticCompare(t, fold, input, interval[0], interval[1])
	}
	if !fold.atEnd() || fold.confirmed.state.version != 11 || meter.failed {
		t.Fatal("observed native setup prevented the complete original semantic range")
	}
}

func TestProviderHistoryNativeHeaderRefusalDoesNotChargeUnenteredBodyDecode(t *testing.T) {
	block, _, _, receipts := nodeJSONFixtureBytes(t)
	binary, err := rlp.EncodeToBytes(block)
	if err != nil {
		t.Fatal(err)
	}
	blockResult, err := json.Marshal(nodeHex(binary))
	if err != nil {
		t.Fatal(err)
	}
	rows := make([]string, len(receipts))
	for i, receipt := range receipts {
		rows[i] = nodeHex(receipt)
	}
	receiptResult, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	input := historyCollectionInput(t)
	input.encoding = "debug-raw-rlp-v1"
	input.capture.HeaderBytes, input.capture.BodyBytes, input.capture.WireBytes, input.capture.JSONDepth = 64<<10, 8<<20, 16<<20, 128
	want := replayHeader(block.Header())
	if _, _, normalErr := providerHistoryCollectionDecodeBlockObserved(context.Background(), string(blockResult), string(receiptResult), want, input, providerHistorySemanticUnitMeter()); normalErr != nil {
		t.Fatal("complete original normal precursor", normalErr)
	}
	body, rest, splitErr := rlp.SplitList(binary)
	_, _, tail, firstErr := rlp.Split(body)
	if splitErr != nil || firstErr != nil || len(rest) != 0 {
		t.Fatal("independent original RLP dimensions")
	}
	headerBytes := len(body) - len(tail)
	// Entered codecs only: JSON result, hex input, tx-count scan, outer
	// list split, first-element split, header decode. The whole body RLP
	// decoder has not been entered and must not consume another len(binary).
	expectedDecoded := uint64(len(blockResult) + 4*len(binary) + len(body) + headerBytes)
	want.Timestamp++ // Same genuine content, different independently required stop.
	meter := providerHistorySemanticUnitMeter()
	out, decoded, decodeErr := providerHistoryCollectionDecodeBlockObserved(context.Background(), string(blockResult), string(receiptResult), want, input, meter)
	if decodeErr == nil || out.HeaderRLP != nil || decoded.header != nil || meter.actual[2] != expectedDecoded || meter.actual[3] != 0 || meter.actual[4] != 0 {
		t.Fatal("early selected-header refusal charged an unentered body decode or returned partial content", decodeErr, meter.actual[2], expectedDecoded)
	}
}
