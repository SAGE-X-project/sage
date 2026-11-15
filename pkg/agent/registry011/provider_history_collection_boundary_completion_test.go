package registry011

import (
	"context"
	"encoding/json"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// Fixed, local decoder inputs only. No malformed block is sent to a node or
// admitted as history; a selected header still requires its complete body.
func TestProviderHistoryCollectionDecoderRejectsInconsistentBodyRoots(t *testing.T) {
	input := historyCollectionInput(t)
	input.encoding = "debug-raw-rlp-v1"
	decode := func(header *types.Header, withdrawals types.Withdrawals) (ReplayBlock, replayDecodedBlock, error) {
		t.Helper()
		block := types.NewBlockWithHeader(header).WithBody(types.Body{Withdrawals: withdrawals})
		raw, err := rlp.EncodeToBytes(block)
		if err != nil {
			t.Fatal(err)
		}
		result, err := json.Marshal(nodeHex(raw))
		if err != nil {
			t.Fatal(err)
		}
		return providerHistoryCollectionDecodeBlockObserved(context.Background(), string(result), "[]", replayHeader(header), input, providerHistorySemanticUnitMeter())
	}
	ordinary := &types.Header{Number: new(big.Int), Difficulty: new(big.Int), Time: 1, GasLimit: 1000000,
		TxHash: types.EmptyTxsHash, ReceiptHash: types.EmptyReceiptsHash, UncleHash: types.EmptyUncleHash}
	if _, decoded, err := decode(ordinary, nil); err != nil || decoded.header == nil {
		t.Fatal("normal empty canonical body precursor", err)
	}
	for _, mode := range []string{"uncle-root", "withdrawals-without-root", "withdrawal-root"} {
		t.Run(mode, func(t *testing.T) {
			header := types.CopyHeader(ordinary)
			var withdrawals types.Withdrawals
			switch mode {
			case "uncle-root":
				header.UncleHash = common.Hash{1}
			case "withdrawals-without-root":
				withdrawals = types.Withdrawals{{Index: 1, Validator: 2, Address: common.Address{3}, Amount: 4}}
			case "withdrawal-root":
				header.WithdrawalsHash = new(common.Hash)
				withdrawals = types.Withdrawals{{Index: 1, Validator: 2, Address: common.Address{3}, Amount: 4}}
			}
			out, decoded, err := decode(header, withdrawals)
			if err != ErrReplay || !reflect.DeepEqual(out, ReplayBlock{}) || !reflect.DeepEqual(decoded, replayDecodedBlock{}) ||
				ordinary.UncleHash != types.EmptyUncleHash || ordinary.WithdrawalsHash != nil {
				t.Fatal("body mismatch exposed a partial value or changed the normal header", err)
			}
		})
	}
}

func TestProviderHistoryCollectionTransitionsRejectExhaustedStopsAndMissingBlocks(t *testing.T) {
	c, input, methods := historyCollectionContentFixture(t, 12)
	var state *providerHistoryCollectionCursor
	for steps := 0; steps < 64; steps++ {
		if c.phase == "state" && state == nil {
			state = c
		}
		if c.phase == "close" && c.high == c.selection.selected.Number {
			break
		}
		c = historyCollectionContentStep(t, c, input, methods)
	}
	if state == nil || state.semantic.window.run.state.version == 0 || c.phase != "close" || c.high != c.selection.selected.Number || c.semantic.blocks < 2 {
		t.Fatal("ordinary present-state and last-window precursors")
	}
	t.Run("present-stop-capacity", func(t *testing.T) {
		root := state.commitmentObserved(providerReplayUnitMeter())
		working := state.copyObserved(state.semantic.observer)
		if working == nil {
			t.Fatal("independent state working copy")
		}
		working.presentStops = working.selection.input.presentStops
		results := historyCollectionContentResults(t, state, input, historyCollectionBoundaryPlan(t, state, methods))
		if !working.valid() || working.contentBoundaryObserved(context.Background(), results, working.semantic.observer) ||
			state.commitmentObserved(providerReplayUnitMeter()) != root {
			t.Fatal("selected stop capacity was exceeded or the accepted predecessor changed")
		}
	})
	t.Run("incomplete-last-window", func(t *testing.T) {
		root := c.commitmentObserved(providerReplayUnitMeter())
		normal := c.copyObserved(c.semantic.observer)
		if normal == nil || !normal.closeContentObserved(context.Background(), normal.semantic.observer) || normal.phase != "done" || !normal.semantic.atEnd() {
			t.Fatal("normal final DATA closure precursor")
		}
		working := c.copyObserved(c.semantic.observer)
		if working == nil {
			t.Fatal("independent final-window working copy")
		}
		working.semantic.blocks--
		if !working.valid() || working.closeContentObserved(context.Background(), working.semantic.observer) || working.phase == "done" ||
			c.commitmentObserved(providerReplayUnitMeter()) != root {
			t.Fatal("a missing block was accepted as complete DATA or changed the predecessor")
		}
	})
}

func TestProviderHistoryCollectionPendingGasBytesRemainFreshOpaqueData(t *testing.T) {
	base := historyCollectionPendingGuardFixture(t)
	root := base.commitmentObserved(providerReplayUnitMeter())
	pending := base.copyObserved(base.semantic.observer)
	if pending == nil {
		t.Fatal("independent retained-DATA copy")
	}
	// Only the byte-container hasher is exercised. These detached timestamps
	// and opaque strings are not decoded consensus/gas evidence or a replay.
	// The original selection, observer and predecessor are never changed.
	for i := range pending.pending {
		header := &pending.pending[i].header
		header.Timestamp = sepoliaAmsterdamTime
		for j := range pending.spine {
			if pending.spine[j].Number == header.Number {
				pending.spine[j].Timestamp = header.Timestamp
			}
		}
		pending.pending[i].gasProfile = pending.selection.input.gas
		pending.pending[i].gasTrace = "[]"
	}
	before := pending.pendingRootObserved(providerReplayUnitMeter())
	if !pending.valid() || before == "" {
		t.Fatal("nonempty opaque native byte-container precursor")
	}
	for _, mode := range []string{"changed-bytes", "different-profile", "missing-bytes"} {
		t.Run(mode, func(t *testing.T) {
			// No full cursor commitment/copy follows the detached timestamp
			// change. Copy only the byte containers used by this pure hasher;
			// semantic state, inner byte slices and spine are read-only here.
			workingValue := *pending
			workingValue.owner = &workingValue
			workingValue.pending = append([]providerHistoryCollectionBlock(nil), pending.pending...)
			working := &workingValue
			switch mode {
			case "changed-bytes":
				working.pending[0].gasTrace = "[ ]"
			case "different-profile":
				working.pending[0].gasProfile = "different-unit-profile"
			case "missing-bytes":
				working.pending[0].gasTrace = ""
			}
			got := working.pendingRootObserved(providerReplayUnitMeter())
			if mode == "changed-bytes" {
				if got == "" || got == before {
					t.Fatal("a retained gas byte change was omitted from the DATA root")
				}
			} else if got != "" {
				t.Fatal("incomplete retained gas DATA received a root")
			}
			if pending.pendingRootObserved(providerReplayUnitMeter()) != before || base.commitmentObserved(providerReplayUnitMeter()) != root {
				t.Fatal("a working byte-container changed its predecessor")
			}
		})
	}
}

func TestProviderHistoryCollectionPendingCopyRefusalsKeepOriginalArrays(t *testing.T) {
	base := historyCollectionPendingGuardFixture(t)
	root := base.commitmentObserved(providerReplayUnitMeter())
	first := uint64(len(base.pending))*uint64(reflect.TypeFor[providerHistoryCollectionBlock]().Size()) + 1
	second := uint64(len(base.traceResults))*uint64(reflect.TypeFor[string]().Size()) + 1
	for _, fuel := range []uint64{0, first, first + second} {
		meter := providerReplayUnitMeter()
		meter.upper[10] = fuel // Selected before the first UNIT observation.
		working := new(providerHistoryCollectionCursor)
		if base.copyPendingObserved(working, meter) || !meter.failed || meter.actual[10] > fuel || base.commitmentObserved(providerReplayUnitMeter()) != root {
			t.Fatal("unfunded array copy succeeded or changed original retained DATA", fuel, meter.actual)
		}
		charged := meter.actual
		if base.copyPendingObserved(new(providerHistoryCollectionCursor), meter) || meter.actual != charged {
			t.Fatal("a refused copy renewed its original UNIT observer")
		}
		// Discard the private partial copy; it is never published or adopted.
	}
}
