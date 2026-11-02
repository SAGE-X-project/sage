package registry011

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// Deliberately synthetic DATA: these canonical blocks start at zero and cannot
// pass the production manifest's real Sepolia deployment constructor. The test
// constructs only the private cursor to exercise the same original-S grammar.
// It does not create a Source, dispatch permit, operator key or external RPC.
func historyCollectionContentFixture(t *testing.T, width uint64) (*providerHistoryCollectionCursor, ReplayInput, *providerMethodBounds) {
	t.Helper()
	in := replayTestInput(t)
	return historyCollectionContentInputFixture(t, width, in, SepoliaAmsterdamRootExecutionGasProfile)
}

func historyCollectionContentInputFixture(t *testing.T, width uint64, in ReplayInput, gas string) (*providerHistoryCollectionCursor, ReplayInput, *providerMethodBounds) {
	t.Helper()
	return historyCollectionContentInputLimitsFixture(t, width, in, gas, replayTestLimits())
}

// Select the ordinary replay limits BEFORE constructing any manifest, sealed
// selection, semantic fold or cursor. No live quota or selected state is patched.
func historyCollectionContentInputLimitsFixture(t *testing.T, width uint64, in ReplayInput, gas string, limits ReplayLimits) (*providerHistoryCollectionCursor, ReplayInput, *providerMethodBounds) {
	t.Helper()
	input := historyCollectionInput(t)
	input.journal = journalTestConfig(t, in)
	input.journal.Replay = limits
	input.encoding, input.gas = "debug-raw-rlp-v1", gas
	input.maximumSelected, input.blocks = in.Anchor.Number, uint64(len(in.Blocks))
	input.windowBlocks, input.contentBlocks = width, min(width, 2)
	input.native, input.effects, input.versions, input.presentStops = 0, 0, uint64(len(in.History)), uint64(len(in.Blocks))
	for _, block := range in.Blocks {
		if journalTestHeader(t, block).Timestamp >= sepoliaAmsterdamTime {
			input.native++
		}
		input.effects += uint64(len(block.Executions))
	}
	journal := providerReplayMarshal(struct {
		Format string
		Config JournalConfig
	}{"sage-replay-journal-v1", input.journal}, 64<<10, nil)
	raw := providerReplayCanonical(input.object(string(journal)), candidateLedgerMetadataLimit, nil)
	selection := providerHistoryCollectionSelection{admissionTestDigest("a"), providerReplayDigest("sage-provider-history-collection-manifest-v1", raw, nil), input, in.Anchor}
	semantic, err := newProviderHistorySemanticFoldObserved(selection.program, input.journal, in.Anchor, input.blocks, input.versions, providerHistorySemanticUnitMeter())
	if err != nil {
		t.Fatal("explicit synthetic semantic DATA", err)
	}
	c := &providerHistoryCollectionCursor{selection: selection, sealed: selection, mode: "content", phase: "open", low: 0, high: min(width-1, in.Anchor.Number), semantic: semantic}
	c.owner = c
	for i := len(in.Blocks) - 1; i >= 0; i-- {
		c.spine = append(c.spine, journalTestHeader(t, in.Blocks[i]))
	}
	b, _ := providerFrontierBoundsFixture()
	if !c.valid() || c.commitmentObserved(providerReplayUnitMeter()) == "" {
		t.Fatal("invalid synthetic full-spine DATA fixture")
	}
	return c, in, b.methods
}

func historyCollectionContentResults(t *testing.T, c *providerHistoryCollectionCursor, in ReplayInput, table *providerHistoryGroupTable) []providerHistoryMemberResult {
	t.Helper()
	encode := func(value any) providerHistoryMemberResult {
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return historyCollectionBoundaryResult(t, raw)
	}
	if c.phase == "open" {
		return historyCollectionBoundaryResults(t, c, nil, table)
	}
	results := make([]providerHistoryMemberResult, table.count)
	switch c.phase {
	case "content":
		position := 0
		for i := range int(min(c.selection.input.contentBlocks, c.high-c.next+1)) {
			b := in.Blocks[int(c.next)+i]
			var header types.Header
			if rlp.DecodeBytes(b.HeaderRLP, &header) != nil {
				t.Fatal("canonical fixture header")
			}
			txs := make([]*types.Transaction, len(b.Transactions))
			for j, raw := range b.Transactions {
				txs[j] = new(types.Transaction)
				if txs[j].UnmarshalBinary(raw) != nil {
					t.Fatal("canonical fixture transaction")
				}
			}
			block := types.NewBlockWithHeader(&header).WithBody(types.Body{Transactions: txs})
			raw, err := rlp.EncodeToBytes(block)
			if err != nil {
				t.Fatal(err)
			}
			results[position] = encode(nodeHex(raw))
			receipts := make([]string, len(b.Receipts))
			for j, receipt := range b.Receipts {
				receipts[j] = nodeHex(receipt)
			}
			results[position+1] = encode(receipts)
			position += 2
			if b.Gas != nil {
				results[position] = historyCollectionBoundaryResult(t, b.Gas.Trace)
				position++
			}
		}
		if position != len(results) {
			t.Fatal("synthetic content results lost an expected native result")
		}
	case "effects":
		for i := range results {
			effect := c.effects[c.effectNext+i]
			results[i] = encode(nodeTestFrame(in.Blocks[effect.number].Executions[effect.index].Root))
		}
	case "state":
		record := ""
		present := c.high > 0 && len(in.History) != 0
		if present {
			record = string(replayTestSteps(t)[c.high-1].Record)
		}
		results[0] = encode(returnABI(t, "record", present, record))
	case "grants":
		results[0] = encode(returnABI(t, "activeGrants", string(replayTestSteps(t)[c.high-1].Grants)))
	case "history":
		for i := range results {
			h := in.History[c.historyNext+uint64(i)-1]
			results[i] = encode(returnABI(t, "historyItem", string(h.Entry), string(h.Command)))
		}
	case "close":
		results[0] = historyCollectionBoundaryHeader(t, journalTestHeader(t, in.Blocks[c.high]))
		results[1], results[2] = historyCollectionBoundaryHeader(t, in.Anchor), historyCollectionBoundaryHeader(t, in.Anchor)
		// Reuse the separately recorded public identity DATA, never a network.
		open := *c
		open.phase = "open"
		identity := historyCollectionBoundaryResults(t, &open, nil, &providerHistoryGroupTable{count: 18})
		results[3], results[4], results[5], results[6] = identity[0], identity[1], identity[2], identity[3]
	default:
		t.Fatal("unexpected synthetic phase", c.phase)
	}
	return results
}

func historyCollectionContentStep(t *testing.T, c *providerHistoryCollectionCursor, in ReplayInput, methods *providerMethodBounds) *providerHistoryCollectionCursor {
	t.Helper()
	table := historyCollectionBoundaryPlan(t, c, methods)
	next, err := c.advanceBoundaryObserved(context.Background(), table, historyCollectionContentResults(t, c, in, table), methods, c.semantic.observer)
	if err != nil || next == nil || next == c || next.completed != c.completed+1 {
		t.Fatal("actual original-shaped content transition", c.phase, c.low, c.high, err)
	}
	return next
}

func TestProviderHistoryCollectionContentReconstructsBothIndependentFullRanges(t *testing.T) {
	for _, width := range []uint64{1, 4, 12} {
		t.Run(map[uint64]string{1: "one-block-windows", 4: "three-windows", 12: "one-window"}[width], func(t *testing.T) {
			left, in, methods := historyCollectionContentFixture(t, width)
			right, _, _ := historyCollectionContentFixture(t, width)
			before := replayTestClone(t, in)
			finals := []*providerHistoryCollectionCursor{}
			for _, initial := range []*providerHistoryCollectionCursor{left, right} {
				c, groups, historyItems := initial, 0, 0
				originalObserver := c.semantic.observer
				for c.phase != "done" && groups < 200 {
					root := c.commitmentObserved(providerReplayUnitMeter())
					confirmed := c.semantic.confirmed.state.version
					phase := c.phase
					if phase == "history" {
						historyItems += historyCollectionBoundaryPlan(t, c, methods).count
					}
					next := historyCollectionContentStep(t, c, in, methods)
					if root == "" || c.commitmentObserved(providerReplayUnitMeter()) != root || c.semantic.confirmed.state.version != confirmed ||
						phase != "close" && next.semantic.confirmed.state.version != confirmed || next.semantic.observer != originalObserver {
						t.Fatal("a failed/unclosed candidate changed its prior confirmed prefix")
					}
					c, groups = next, groups+1
				}
				if c.phase != "done" || !c.semantic.atEnd() || c.semantic.blocks != 12 || c.semantic.items != 11 || historyItems != 11 ||
					c.semantic.confirmed.state.version != 11 || c.semantic.confirmed.state.record.State != "deactivated" || c.traces != 12 || c.native != 0 ||
					len(c.pending) != 0 || len(c.effects) != 0 || c.stopRecord != "" || len(c.semantic.confirmed.claims) != 1 || len(c.semantic.confirmed.created) != 1 {
					t.Fatal("whole synthetic range, version delta, indexes or terminal state missing", c.phase, groups, historyItems)
				}
				wantWindows := map[uint64]int{1: 12, 4: 3, 12: 1}[width]
				wantPresent := map[uint64]uint64{1: 11, 4: 3, 12: 1}[width]
				if c.semantic.windows != wantWindows || c.presentStops != wantPresent {
					t.Fatal("window counts or cumulative present stops changed", c.semantic.windows, c.presentStops)
				}
				for _, coordinate := range []int{3, 4, 5, 6, 7} {
					if originalObserver.actual[coordinate] == 0 {
						t.Fatal("normal original observer omitted a block/transaction/frame/log/proof coordinate", coordinate)
					}
				}
				if table, err := c.nextTableObserved(c.completed+1, *methods, c.selection.input.capture, c.semantic.observer); err == nil || table != nil {
					t.Fatal("terminal DATA cursor granted an extra request")
				}
				finals = append(finals, c)
			}
			if !reflect.DeepEqual(before, in) || left.semantic == right.semantic || left.semantic.observer == right.semantic.observer {
				t.Fatal("input or independent reader ownership changed")
			}
			finals[0].semantic.confirmed.state.record.Keys[0].State = "changed-reader-copy"
			if finals[1].semantic.confirmed.state.record.Keys[0].State != "revoked" {
				t.Fatal("the independently reconstructed final records share mutable keys")
			}
		})
	}
}

func TestProviderHistoryCollectionContentKeepsRecordedConsensusBytesInBothEncodings(t *testing.T) {
	block, blockJSON, receiptJSON, receipts := nodeJSONFixtureBytes(t)
	binary, err := rlp.EncodeToBytes(block)
	if err != nil {
		t.Fatal(err)
	}
	wantHeader, err := rlp.EncodeToBytes(block.Header())
	if err != nil {
		t.Fatal(err)
	}
	for _, encoding := range []string{"debug-raw-rlp-v1", EthereumJSONBlocks} {
		t.Run(encoding, func(t *testing.T) {
			input := historyCollectionInput(t)
			input.encoding, input.journal.Replay = encoding, nodeJSONLimits().Replay
			// Explicit UNIT capture selected before decoding recorded originals.
			// Production/default limits and all existing helpers stay unchanged.
			input.capture.HeaderBytes, input.capture.BodyBytes, input.capture.WireBytes, input.capture.JSONDepth = 64<<10, 8<<20, 16<<20, 128
			blockResult, receiptResult := blockJSON, receiptJSON
			if encoding == "debug-raw-rlp-v1" {
				blockResult, err = json.Marshal(nodeHex(binary))
				if err != nil {
					t.Fatal(err)
				}
				rows := make([]string, len(receipts))
				for i, receipt := range receipts {
					rows[i] = nodeHex(receipt)
				}
				receiptResult, err = json.Marshal(rows)
				if err != nil {
					t.Fatal(err)
				}
			}
			if !input.capture.valid() || len(blockResult) > input.capture.BodyBytes || len(receiptResult) > input.capture.BodyBytes {
				t.Fatal("explicit recorded UNIT capture cannot hold its original inputs")
			}
			meter := providerHistorySemanticUnitMeter()
			out, decoded, err := providerHistoryCollectionDecodeBlockObserved(context.Background(), string(blockResult), string(receiptResult), replayHeader(block.Header()), input, meter)
			if err != nil || !reflect.DeepEqual(out.HeaderRLP, wantHeader) || !reflect.DeepEqual(out.Receipts, receipts) || len(out.Transactions) != len(block.Transactions()) ||
				decoded.header == nil || decoded.header.Hash() != block.Hash() || out.Gas != nil || len(out.Executions) != 0 {
				t.Fatal("closed content decoder changed original consensus bytes or invented gas/trace authority", err)
			}
			for i, tx := range block.Transactions() {
				original, marshalErr := tx.MarshalBinary()
				if marshalErr != nil || !reflect.DeepEqual(original, out.Transactions[i]) {
					t.Fatal("canonical transaction changed", i, marshalErr)
				}
			}
		})
	}
}

func TestProviderHistoryCollectionContentRefusalsKeepTheOriginalPrefix(t *testing.T) {
	for _, phase := range []string{"content", "effects", "state", "grants", "history", "close"} {
		t.Run(phase, func(t *testing.T) {
			c, in, methods := historyCollectionContentFixture(t, 4)
			for c.phase != phase {
				c = historyCollectionContentStep(t, c, in, methods)
			}
			table := historyCollectionBoundaryPlan(t, c, methods)
			results := historyCollectionContentResults(t, c, in, table)
			root := c.commitmentObserved(providerReplayUnitMeter())
			confirmed, blocks, items, windows := c.semantic.confirmed.state.version, c.semantic.blocks, c.semantic.items, c.semantic.windows
			// A local literal parse/comparison refusal, not an attack producer.
			results[0] = historyCollectionBoundaryResult(t, []byte(`"ordinary unavailable fixture"`))
			if next, err := c.advanceBoundaryObserved(context.Background(), table, results, methods, c.semantic.observer); err == nil || next != nil ||
				c.semantic.failed || c.commitmentObserved(providerReplayUnitMeter()) != root || c.semantic.confirmed.state.version != confirmed ||
				c.semantic.blocks != blocks || c.semantic.items != items || c.semantic.windows != windows {
				t.Fatal("refusal published or changed a previously accepted prefix", phase, err)
			}
		})
	}
}

func TestProviderHistoryCollectionPendingCopyOwnsEveryMutableArray(t *testing.T) {
	c, in, methods := historyCollectionContentFixture(t, 4)
	for c.phase != "effects" {
		c = historyCollectionContentStep(t, c, in, methods)
	}
	root := c.commitmentObserved(providerReplayUnitMeter())
	next := c.copyObserved(c.semantic.observer)
	if next == nil || root == "" || next.commitmentObserved(providerReplayUnitMeter()) != root ||
		&next.pending[0] == &c.pending[0] || &next.effects[0] == &c.effects[0] || &next.traceResults[0] == &c.traceResults[0] ||
		&next.pending[0].transactions[0] == &c.pending[0].transactions[0] || &next.pending[0].receipts[0] == &c.pending[0].receipts[0] {
		t.Fatal("pending consensus/trace DATA aliases its predecessor")
	}
	next.pending[0].transactions[0], next.pending[0].receipts[0], next.traceResults[0] = "changed-copy", "changed-copy", "changed-copy"
	if c.commitmentObserved(providerReplayUnitMeter()) != root || c.traceResults[0] != "" || next.commitmentObserved(providerReplayUnitMeter()) == root {
		t.Fatal("pending mutation was shared or omitted from the fresh root")
	}
}

func TestProviderHistoryCollectionContentRequiresTheOriginalObserverAndLiveCaller(t *testing.T) {
	for _, mode := range []string{"nil-context", "cancelled", "different-observer", "spent-original"} {
		t.Run(mode, func(t *testing.T) {
			c, in, methods := historyCollectionContentFixture(t, 4)
			c = historyCollectionContentStep(t, c, in, methods)
			table := historyCollectionBoundaryPlan(t, c, methods)
			results := historyCollectionContentResults(t, c, in, table)
			root := c.commitmentObserved(providerReplayUnitMeter())
			ctx, meter := context.Background(), c.semantic.observer
			switch mode {
			case "nil-context":
				ctx = nil
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "different-observer":
				meter = providerReplayUnitMeter()
			case "spent-original":
				meter.upper = meter.actual
			}
			if next, err := c.advanceBoundaryObserved(ctx, table, results, methods, meter); err == nil || next != nil || c.commitmentObserved(providerReplayUnitMeter()) != root {
				t.Fatal("candidate renewed its original allowance or ignored caller/observer binding", mode, err)
			}
		})
	}
}

func TestProviderHistoryCollectionEmptyNativeKeepsEachPreselectedSchema(t *testing.T) {
	for _, profile := range []string{SepoliaAmsterdamGasProfile, SepoliaAmsterdamNethermindGasProfile, SepoliaAmsterdamRootRegularGasProfile, SepoliaAmsterdamRootExecutionGasProfile} {
		for _, mode := range []string{"empty", "null", "object", "extra-row", "nonzero-header-gas", "nonempty-block"} {
			t.Run(profile+"/"+mode, func(t *testing.T) {
				header := &types.Header{Number: new(big.Int), Difficulty: new(big.Int), Time: sepoliaAmsterdamTime, GasLimit: 1000000,
					TxHash: types.EmptyTxsHash, ReceiptHash: types.EmptyReceiptsHash, UncleHash: types.EmptyUncleHash}
				if mode == "nonzero-header-gas" {
					header.GasUsed = 1
				}
				binary, err := rlp.EncodeToBytes(header)
				if err != nil {
					t.Fatal(err)
				}
				trace := map[string]string{"empty": "[]", "null": "null", "object": "{}", "extra-row": "[{}]", "nonzero-header-gas": "[]", "nonempty-block": "[]"}[mode]
				input := ReplayInput{Registry: replayTestRegistry, Agent: replayTestAgent, Anchor: replayHeader(header),
					Blocks: []ReplayBlock{{HeaderRLP: binary, Gas: &ReplayGasEvidence{profile, replayHeader(header).Hash, json.RawMessage(trace)}}}}
				if mode == "nonempty-block" {
					ordinary := replayTestInput(t).Blocks[0]
					if rlp.DecodeBytes(ordinary.HeaderRLP, header) != nil {
						t.Fatal("normal nonempty canonical block")
					}
					header.Time = sepoliaAmsterdamTime
					ordinary.HeaderRLP, err = rlp.EncodeToBytes(header)
					if err != nil {
						t.Fatal(err)
					}
					ordinary.Executions[0].BlockHash = replayHeader(header).Hash
					ordinary.Gas = &ReplayGasEvidence{profile, replayHeader(header).Hash, json.RawMessage(trace)}
					input.Anchor, input.Blocks = replayHeader(header), []ReplayBlock{ordinary}
				}
				c, in, methods := historyCollectionContentInputFixture(t, 1, input, profile)
				c = historyCollectionContentStep(t, c, in, methods)
				table := historyCollectionBoundaryPlan(t, c, methods)
				results := historyCollectionContentResults(t, c, in, table)
				root := c.commitmentObserved(providerReplayUnitMeter())
				observer := c.semantic.observer
				next, advanceErr := c.advanceBoundaryObserved(context.Background(), table, results, methods, observer)
				if mode != "empty" {
					if advanceErr == nil || next != nil || c.commitmentObserved(providerReplayUnitMeter()) != root || c.semantic.blocks != 0 {
						t.Fatal("empty branch accepted missing/extra native rows or invented nonzero gas", advanceErr)
					}
					return
				}
				if advanceErr != nil || next == nil || next.phase != "state" || next.native != 1 || next.traces != 0 || next.semantic.blocks != 1 ||
					next.selection.input.gas != profile || next.semantic.observer != observer || c.commitmentObserved(providerReplayUnitMeter()) != root {
					t.Fatal("complete empty native result changed the preselected schema/observer or failed strict gas", advanceErr)
				}
				for next.phase != "done" {
					next = historyCollectionContentStep(t, next, in, methods)
				}
				if !next.semantic.atEnd() || next.semantic.confirmed.state.version != 0 || next.presentStops != 0 {
					t.Fatal("empty block invented a present record or Source authority")
				}
			})
		}
	}
}

// Forty ordinary, distinct claim commitments in two synthetic complete blocks.
// The existing public disposable transaction fixture supplies signed envelopes;
// nothing is sent to a node. The target blocks have 2 and 38 Registry effects,
// so the first 32-result S applies only the complete first block.
func historyCollectionStreamingInput(t *testing.T) ReplayInput {
	t.Helper()
	return historyCollectionStreamingCountInput(t, 40)
}

func historyCollectionStreamingCountInput(t *testing.T, count int) ReplayInput {
	t.Helper()
	if count < 3 || count > 40 {
		t.Fatal("ordinary finite streaming fixture count")
	}
	contract, outcomes, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	event := outcomes.Events["ClaimCommitted"]
	material := ReplayInput{Registry: replayTestRegistry, Agent: replayTestAgent}
	for i := range count {
		claim := sha256.Sum256([]byte(fmt.Sprintf("ordinary-unit-claim-%d", i)))
		data, packErr := contract.Pack("commitClaim", claim)
		height := int64(0)
		if i >= 2 {
			height = 1
		}
		entry, eventErr := event.Inputs.NonIndexed().Pack(big.NewInt(height))
		if packErr != nil || eventErr != nil {
			t.Fatal(packErr, eventErr)
		}
		log := &types.Log{Address: common.HexToAddress(SepoliaRegistryAddress),
			Topics: []common.Hash{event.ID, common.BytesToHash(common.HexToAddress(replayTestWallet).Bytes()), common.Hash(claim)}, Data: entry}
		replayTestAppendBlock(t, &material, data, log, replayTestWallet, uint64(i))
	}
	input := ReplayInput{Registry: replayTestRegistry, Agent: replayTestAgent}
	for ordinal, interval := range [][2]int{{0, 2}, {2, count}} {
		txs, receipts, executions := types.Transactions{}, types.Receipts{}, []ReplayExecution{}
		for i := interval[0]; i < interval[1]; i++ {
			tx, receipt := new(types.Transaction), new(types.Receipt)
			if tx.UnmarshalBinary(material.Blocks[i].Transactions[0]) != nil || receipt.UnmarshalBinary(material.Blocks[i].Receipts[0]) != nil {
				t.Fatal("ordinary canonical claim material")
			}
			receipt.CumulativeGasUsed = uint64(i-interval[0]+1) * 21000
			txs, receipts, executions = append(txs, tx), append(receipts, receipt), append(executions, material.Blocks[i].Executions[0])
		}
		header := &types.Header{Number: big.NewInt(int64(ordinal)), Difficulty: new(big.Int), Time: uint64(ordinal), GasLimit: 50000000,
			GasUsed: uint64(len(txs)) * 21000, TxHash: types.DeriveSha(txs, trie.NewStackTrie(nil)),
			ReceiptHash: types.DeriveSha(receipts, trie.NewStackTrie(nil)), UncleHash: types.EmptyUncleHash}
		if ordinal > 0 {
			header.ParentHash = common.HexToHash(input.Anchor.Hash)
		}
		binary, encodeErr := rlp.EncodeToBytes(header)
		if encodeErr != nil {
			t.Fatal(encodeErr)
		}
		block := ReplayBlock{HeaderRLP: binary, Transactions: make([][]byte, len(txs)), Receipts: make([][]byte, len(receipts)), Executions: executions}
		for i := range txs {
			var txErr, receiptErr error
			block.Transactions[i], txErr = txs[i].MarshalBinary()
			block.Receipts[i], receiptErr = receipts[i].MarshalBinary()
			if txErr != nil || receiptErr != nil {
				t.Fatal(txErr, receiptErr)
			}
			block.Executions[i].BlockHash, block.Executions[i].TransactionIndex = replayHeader(header).Hash, uint64(i)
		}
		input.Blocks = append(input.Blocks, block)
		input.Anchor = replayHeader(header)
	}
	return input
}

func TestProviderHistoryCollectionStreamingAppliesOnlyCompleteBlocksAcrossOriginalGroups(t *testing.T) {
	for _, mode := range []string{"complete", "last-trace-unavailable", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			c, in, methods := historyCollectionContentInputFixture(t, 2, historyCollectionStreamingInput(t), SepoliaAmsterdamRootExecutionGasProfile)
			c = historyCollectionContentStep(t, c, in, methods) // open
			c = historyCollectionContentStep(t, c, in, methods) // complete content ->40 effects
			if len(c.effects) != 40 || len(c.pending) != 2 || c.pendingApplied != 0 || c.semantic.blocks != 0 {
				t.Fatal("normal two-block/40-effect precursor is incomplete")
			}
			c = historyCollectionContentStep(t, c, in, methods) // first32 traces
			if c.phase != "effects" || c.effectNext != 32 || c.traces != 32 || c.pendingApplied != 1 || c.semantic.blocks != 1 ||
				c.semantic.last.Number != 0 || c.semantic.confirmed.state.version != 0 || len(c.semantic.confirmed.claims) != 0 ||
				len(c.semantic.window.run.claims) != 2 || c.pending[0].headerRLP != "" || c.pending[1].headerRLP == "" ||
				c.traceResults[0] != "" || c.traceResults[1] != "" || c.traceResults[2] == "" || c.traceResults[31] == "" || c.traceResults[32] != "" {
				t.Fatal("first32 S applied an incomplete block, published state or cleared unconsumed originals")
			}
			root := c.commitmentObserved(providerReplayUnitMeter())
			copyCursor := c.copyObserved(c.semantic.observer)
			if root == "" || copyCursor == nil || copyCursor.commitmentObserved(providerReplayUnitMeter()) != root ||
				&copyCursor.pending[1].transactions[0] == &c.pending[1].transactions[0] || &copyCursor.traceResults[2] == &c.traceResults[2] {
				t.Fatal("partially applied batch root/copy lost actual pending bytes")
			}
			table := historyCollectionBoundaryPlan(t, c, methods)
			results := historyCollectionContentResults(t, c, in, table)
			if table.count != 8 {
				t.Fatal("the last original S did not select all eight remaining effects")
			}
			ctx := context.Background()
			if mode == "last-trace-unavailable" {
				results[7] = historyCollectionBoundaryResult(t, []byte(`"ordinary unavailable fixture"`))
			} else if mode == "cancelled" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			next, err := c.advanceBoundaryObserved(ctx, table, results, methods, c.semantic.observer)
			if mode != "complete" {
				if err == nil || next != nil || c.commitmentObserved(providerReplayUnitMeter()) != root || c.semantic.blocks != 1 || len(c.semantic.window.run.claims) != 2 {
					t.Fatal("late failure changed the preceding accepted partial batch", err)
				}
				return
			}
			if err != nil || next == nil || next.phase != "state" || next.traces != 40 || next.semantic.blocks != 2 || len(next.pending) != 0 || len(next.semantic.window.run.claims) != 40 {
				t.Fatal("last eight traces did not complete and release the entire batch", err)
			}
			for next.phase != "done" {
				next = historyCollectionContentStep(t, next, in, methods)
			}
			if !next.semantic.atEnd() || next.semantic.items != 0 || len(next.semantic.confirmed.claims) != 40 || len(next.semantic.confirmed.created) != 0 ||
				next.semantic.confirmed.state.version != 0 || c.commitmentObserved(providerReplayUnitMeter()) != root {
				t.Fatal("final close lost shared claims, invented a subject or changed its predecessor")
			}
		})
	}
}
