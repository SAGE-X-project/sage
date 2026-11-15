package registry011

import (
	"context"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func providerHistoryExecutionUnitInput(t *testing.T) (ReplayExecution, replayDecodedBlock) {
	t.Helper()
	raw := replayTestInput(t).Blocks[0]
	block, err := replayDecodeContent(context.Background(), raw, &replayBudget{limits: replayTestLimits()})
	if err != nil || len(raw.Executions) != 1 {
		t.Fatal("existing finite canonical unit precursor", err)
	}
	return raw.Executions[0], block
}

func TestProviderHistoryObservedExecutionPreservesExistingCallerAndReceiptChecks(t *testing.T) {
	e, block := providerHistoryExecutionUnitInput(t)
	legacy := &replayBudget{limits: replayTestLimits()}
	want, err := replayExecution(context.Background(), &e, block, 0, legacy)
	if err != nil || len(want) != 1 || !want[0].eligible {
		t.Fatal("existing canonical unit execution precursor", err)
	}
	m := providerHistoryExecutionUnitMeter()
	b := &replayBudget{limits: replayTestLimits(), observer: m}
	got, err := replayExecution(context.Background(), &e, block, 0, b)
	if err != nil || !reflect.DeepEqual(got, want) || b.bytes != legacy.bytes || m.actual[5] != 2 || m.actual[6] != 1 || m.actual[2] < 97 || m.failed {
		t.Fatal("execution root/caller/receipt or original entries", err, m.actual)
	}
	for i, n := range m.actual {
		if i != 2 && i != 5 && i != 6 && i != 10 && n != 0 {
			t.Fatal("normalized DATA invented network/proof/disk operations", m.actual)
		}
	}
	// Input and log payload storage in the original frame is unchanged.
	if got[0].frame != &e.Root.Calls[0] {
		t.Fatal("legacy borrowed frame identity")
	}
}

func TestProviderHistoryObservedExecutionCreatesSameNormalizedContractAddress(t *testing.T) {
	// A finite normalized root subject, not an on-chain block/Source claim.
	tx := types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(11155111), GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(2), Gas: 21000, Value: new(big.Int), V: big.NewInt(0), R: big.NewInt(1), S: big.NewInt(1)})
	from, err := types.Sender(types.LatestSignerForChainID(big.NewInt(11155111)), tx)
	if err != nil {
		t.Fatal("finite recovery precursor", err)
	}
	header := &types.Header{Number: big.NewInt(1), Difficulty: new(big.Int)}
	block := replayDecodedBlock{header: header, txs: types.Transactions{tx}, receipts: types.Receipts{{Status: 1}}}
	e := ReplayExecution{BlockHash: header.Hash().Hex(), TransactionHash: tx.Hash().Hex(), Root: ReplayFrame{Type: "CREATE", From: replayAddress(from), To: replayAddress(crypto.CreateAddress(from, tx.Nonce())), Value: "0x0"}}
	want, err := replayExecution(context.Background(), &e, block, 0, &replayBudget{limits: replayTestLimits()})
	if err != nil {
		t.Fatal("existing normalized creation rule", err)
	}
	got, err := replayExecutionObserved(context.Background(), &e, block, 0, &replayBudget{limits: replayTestLimits(), observer: providerHistoryExecutionUnitMeter()})
	if err != nil || !reflect.DeepEqual(got, want) || got == nil {
		t.Fatal("observed CREATE address/root empty shape", err)
	}
}

func TestProviderHistoryObservedExecutionRefusesSingleRootAndReceiptChanges(t *testing.T) {
	base, block := providerHistoryExecutionUnitInput(t)
	if _, err := replayExecution(context.Background(), &base, block, 0, &replayBudget{limits: replayTestLimits()}); err != nil {
		t.Fatal("normal root precursor", err)
	}
	for _, tc := range []struct {
		name string
		edit func(*ReplayExecution)
	}{
		{"block", func(e *ReplayExecution) { e.BlockHash = common.Hash{}.Hex() }},
		{"transaction", func(e *ReplayExecution) { e.TransactionHash = common.Hash{}.Hex() }},
		{"ordinal", func(e *ReplayExecution) { e.TransactionIndex = 1 }},
		{"sender", func(e *ReplayExecution) { e.Root.From = SepoliaRegistryAddress }},
		{"value", func(e *ReplayExecution) { e.Root.Value = "0x1" }},
		{"input", func(e *ReplayExecution) { e.Root.Input = []byte{2} }},
		{"root-error", func(e *ReplayExecution) { e.Root.Error = "reverted" }},
		{"kind", func(e *ReplayExecution) { e.Root.Type = "CREATE" }},
		{"recipient", func(e *ReplayExecution) { e.Root.To = SepoliaRegistryAddress }},
		{"missing-log", func(e *ReplayExecution) { e.Root.Calls[0].Logs = nil }},
		{"log-data", func(e *ReplayExecution) { e.Root.Calls[0].Logs[0].Data = []byte{1} }},
		{"log-topic", func(e *ReplayExecution) { e.Root.Calls[0].Logs[0].Topics[0] = common.Hash{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := replayTestClone(t, base)
			tc.edit(&e)
			if _, err := replayExecution(context.Background(), &e, block, 0, &replayBudget{limits: replayTestLimits()}); err == nil {
				t.Fatal("legacy must reject this single change")
			}
			if got, err := replayExecution(context.Background(), &e, block, 0, &replayBudget{limits: replayTestLimits(), observer: providerHistoryExecutionUnitMeter()}); err == nil || got != nil {
				t.Fatal("changed root/receipt returned partial result", err)
			}
		})
	}
	bad := block
	receipt := *block.receipts[0]
	receipt.Status = 0
	bad.receipts = types.Receipts{&receipt}
	if got, err := replayExecutionObserved(context.Background(), &base, bad, 0, &replayBudget{limits: replayTestLimits(), observer: providerHistoryExecutionUnitMeter()}); err == nil || got != nil {
		t.Fatal("failed receipt cannot authorize execution")
	}
	receipt = *block.receipts[0]
	receipt.Logs = append([]*types.Log{}, receipt.Logs...)
	entry := *receipt.Logs[0]
	entry.Address = common.Address{}
	receipt.Logs[0] = &entry
	if got, err := replayExecutionObserved(context.Background(), &base, bad, 0, &replayBudget{limits: replayTestLimits(), observer: providerHistoryExecutionUnitMeter()}); err == nil || got != nil {
		t.Fatal("receipt address mismatch")
	}
	entry = *block.receipts[0].Logs[0]
	entry.Topics = nil
	receipt.Logs[0] = &entry
	if got, err := replayExecutionObserved(context.Background(), &base, bad, 0, &replayBudget{limits: replayTestLimits(), observer: providerHistoryExecutionUnitMeter()}); err == nil || got != nil {
		t.Fatal("receipt topic count mismatch")
	}
	receipt.Logs[0] = nil
	if got, err := replayExecutionObserved(context.Background(), &base, bad, 0, &replayBudget{limits: replayTestLimits(), observer: providerHistoryExecutionUnitMeter()}); err == nil || got != nil {
		t.Fatal("missing receipt log")
	}
}

func TestProviderHistoryObservedExecutionRefusesMissingAndShortOriginalInputs(t *testing.T) {
	e, block := providerHistoryExecutionUnitInput(t)
	fresh := func() *replayBudget {
		return &replayBudget{limits: replayTestLimits(), observer: providerHistoryExecutionUnitMeter()}
	}
	if _, err := replayExecutionObserved(context.Background(), &e, block, 0, fresh()); err != nil {
		t.Fatal("otherwise valid observed precursor", err)
	}
	noHeader := block
	noHeader.header = nil
	noTransactions := block
	noTransactions.txs = nil
	noReceipts := block
	noReceipts.receipts = nil
	nilTransaction := block
	nilTransaction.txs = types.Transactions{nil}
	nilReceipt := block
	nilReceipt.receipts = types.Receipts{nil}
	for _, tc := range []struct {
		name  string
		ctx   context.Context
		e     *ReplayExecution
		block replayDecodedBlock
		index int
		b     *replayBudget
	}{
		{"context", nil, &e, block, 0, fresh()},
		{"execution", context.Background(), nil, block, 0, fresh()},
		{"header", context.Background(), &e, noHeader, 0, fresh()},
		{"transaction-list", context.Background(), &e, noTransactions, 0, fresh()},
		{"receipt-list", context.Background(), &e, noReceipts, 0, fresh()},
		{"nil-transaction", context.Background(), &e, nilTransaction, 0, fresh()},
		{"nil-receipt", context.Background(), &e, nilReceipt, 0, fresh()},
		{"negative-index", context.Background(), &e, block, -1, fresh()},
		{"past-index", context.Background(), &e, block, 1, fresh()},
		{"budget", context.Background(), &e, block, 0, nil},
		{"observer", context.Background(), &e, block, 0, &replayBudget{limits: replayTestLimits()}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := replayExecutionObserved(tc.ctx, tc.e, tc.block, tc.index, tc.b); err == nil || got != nil {
				t.Fatal("single missing original input")
			}
			if tc.b != nil && tc.b.observer != nil && tc.b.observer.actual != (candidateWorkVector{}) {
				t.Fatal("guard performed execution work", tc.b.observer.actual)
			}
		})
	}
	for _, upper := range []candidateWorkVector{{}, {2: 96, 5: 65536, 6: 65536, 10: 1 << 60}, {2: 1 << 30, 5: 1, 6: 65536, 10: 1 << 60}} {
		m := providerHistoryContentUnitMeter(upper)
		b := &replayBudget{limits: replayTestLimits(), observer: m}
		if got, err := replayExecution(context.Background(), &e, block, 0, b); err == nil || got != nil || !m.failed {
			t.Fatal("unfunded execution returned partial logs", err, m.actual)
		}
		before := m.actual
		if got, err := replayExecution(context.Background(), &e, block, 0, b); err == nil || got != nil || m.actual != before {
			t.Fatal("failed original execution resumed")
		}
	}
}
