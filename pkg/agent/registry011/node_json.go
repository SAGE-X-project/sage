package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math/big"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
)

const nodeJSONHeaderFields = "parentHash sha3Uncles miner stateRoot transactionsRoot receiptsRoot logsBloom difficulty number gasLimit gasUsed timestamp extraData mixHash nonce baseFeePerGas withdrawalsRoot blobGasUsed excessBlobGas parentBeaconBlockRoot requestsHash blockAccessListHash slotNumber"

// These methods choose a single content encoding BEFORE the request. They do
// not retry, select a new provider or make a missing gas witness acceptable.
// Reconstructed consensus bytes are derived; Exchanges retain original JSON.
func (a *nodeAcquisition) blockBytes(h FinalizedHeader) ([]byte, error) {
	method, params := "debug_getRawBlock", []any{h.Hash}
	if a.limits.BlockEncoding == EthereumJSONBlocks {
		method, params = "eth_getBlockByHash", []any{h.Hash, true}
	}
	result, err := a.call(method, params)
	if err != nil {
		return nil, err
	}
	if a.limits.BlockEncoding == EthereumJSONBlocks {
		return nodeJSONBlock(a.ctx, result, h, &a.work)
	}
	if len(result) > 2*a.limits.Replay.MaxEvidenceBytes+4 {
		return nil, ErrReplayUnavailable
	}
	b, ok := nodeData(result)
	if !ok {
		return nil, ErrReplayUnavailable
	}
	return b, nil
}

func (a *nodeAcquisition) receiptBytes(block *types.Block) ([][]byte, error) {
	method := "debug_getRawReceipts"
	if a.limits.BlockEncoding == EthereumJSONBlocks {
		method = "eth_getBlockReceipts"
	}
	result, err := a.call(method, []any{block.Hash().Hex()})
	if err != nil {
		return nil, err
	}
	if a.limits.BlockEncoding == EthereumJSONBlocks {
		return nodeJSONReceipts(a.ctx, result, block, &a.work)
	}
	rows, ok := nodeRawArray(result, a.limits.Replay.MaxTransactions)
	if !ok || len(rows) != len(block.Transactions()) {
		return nil, ErrReplay
	}
	out := make([][]byte, 0, len(rows))
	for _, row := range rows {
		b, ok := nodeData(row)
		if !ok {
			return nil, ErrReplayUnavailable
		}
		out = append(out, b)
	}
	return out, nil
}

func nodeJSONPreflight(ctx context.Context, raw json.RawMessage, budget *replayBudget) bool {
	if budget != nil && budget.observer != nil && len(raw) > 64<<20 {
		return false
	}
	if ctx == nil || ctx.Err() != nil || budget == nil || !budget.charge(len(raw)) || !nodeJSONScanObserved(raw, budget.observer) || !utf8.Valid(raw) || !nodeJSONScanObserved(raw, budget.observer) || !validRPCUnicode(raw) || !budget.decodeObserved(raw) {
		return false
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if !readRPCValue(d, 0, 32) {
		return false
	}
	_, err := d.Token()
	return err == io.EOF
}

func nodeJSONFields(raw json.RawMessage, allowed, required string) (map[string]json.RawMessage, bool) {
	m, ok := nodeObject(raw, allowed)
	if !ok {
		return nil, false
	}
	for _, name := range strings.Fields(required) {
		value, present := m[name]
		if !present || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, false
		}
	}
	return m, true
}

// Compare known typed values with the pinned codec's canonical JSON. Null-only
// optional fields may be omitted; absent mandatory fields cannot become zero.
// The only missing alias permitted is yParity when v is explicitly present.
func nodeJSONCanonical(m map[string]json.RawMessage, value any, extras string) bool {
	b, err := json.Marshal(value)
	if err != nil {
		return false
	}
	var expected map[string]json.RawMessage
	if json.Unmarshal(b, &expected) != nil || expected == nil {
		return false
	}
	for key := range m {
		if _, known := expected[key]; !known && !strings.Contains(" "+extras+" ", " "+key+" ") {
			return false
		}
	}
	for key, want := range expected {
		got, present := m[key]
		if bytes.Equal(want, []byte("null")) && !present || key == "yParity" && !present && len(m["v"]) != 0 {
			continue
		}
		var actualValue, expectedValue any
		if !present || json.Unmarshal(got, &actualValue) != nil || json.Unmarshal(want, &expectedValue) != nil || !reflect.DeepEqual(actualValue, expectedValue) {
			return false
		}
	}
	return true
}

// This closed encoding supports only the selected post-Amsterdam PoS interval:
// all 23 header fields must exist, including BAL hash and slot number. It never
// fills a missing header field from zero/defaults or guesses a future fork.
func nodeJSONBlock(ctx context.Context, raw json.RawMessage, selected FinalizedHeader, budget *replayBudget) ([]byte, error) {
	if !nodeJSONPreflight(ctx, raw, budget) || selected.Timestamp < sepoliaAmsterdamTime {
		return nil, ErrReplayUnavailable
	}
	m, ok := nodeJSONFieldsObserved(raw, nodeJSONHeaderFields+" hash size totalDifficulty transactions uncles withdrawals", nodeJSONHeaderFields+" hash transactions uncles withdrawals", budget.observer)
	if !ok {
		return nil, ErrReplayUnavailable
	}
	for _, key := range strings.Fields(nodeJSONHeaderFields) {
		if len(m[key]) > 1030 {
			return nil, ErrReplayUnavailable
		}
	}
	for _, key := range strings.Fields("number gasLimit gasUsed timestamp blobGasUsed excessBlobGas slotNumber") {
		value, valid := nodeStringObserved(m[key], budget.observer)
		if _, quantity := rpcQuantityObserved(value, budget.observer); !valid || !quantity {
			return nil, ErrReplayUnavailable
		}
	}
	for _, key := range []string{"difficulty", "baseFeePerGas"} {
		value, valid := nodeStringObserved(m[key], budget.observer)
		if _, quantity := nodeValueObserved(value, budget.observer); !valid || !quantity {
			return nil, ErrReplayUnavailable
		}
	}
	if value, present := m["totalDifficulty"]; present && !nodeLiteralObserved(value, "null", budget.observer) {
		text, valid := nodeStringObserved(value, budget.observer)
		if _, quantity := nodeValueObserved(text, budget.observer); !valid || !quantity {
			return nil, ErrReplayUnavailable
		}
	}
	var header types.Header
	if !budget.blockObserved() || providerReplayUnmarshal(raw, &header, budget.observer) != nil || !nodeJSONCanonicalObserved(m, header, "size totalDifficulty transactions uncles withdrawals", len(raw), budget.observer) || !nodeJSONSelectedObserved(&header, selected, budget.observer) || header.UncleHash != types.EmptyUncleHash {
		return nil, ErrReplay
	}
	uncles, ok := nodeRawArrayObserved(m["uncles"], 0, budget.observer)
	if !ok || len(uncles) != 0 {
		return nil, ErrReplayUnavailable
	}
	rows, ok := nodeRawArrayObserved(m["transactions"], budget.limits.MaxTransactions, budget.observer)
	if !ok {
		return nil, ErrReplayUnavailable
	}
	if !nodeJSONSliceObserved(len(rows), 1, budget.observer) {
		return nil, ErrReplayUnavailable
	}
	txs := make([]*types.Transaction, 0, len(rows))
	var txBytes [][]byte
	if budget.observer != nil {
		if !nodeJSONSliceObserved(len(rows), 3, budget.observer) {
			return nil, ErrReplayUnavailable
		}
		txBytes = make([][]byte, 0, len(rows))
	}
	for i, row := range rows {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if !budget.transactionObserved() {
			return nil, ErrReplayUnavailable
		}
		tx, err := nodeJSONTransactionObserved(row, &header, uint64(i), budget.observer)
		if err != nil {
			return nil, err
		}
		if !nodeJSONSliceObserved(1, 1, budget.observer) {
			return nil, ErrReplayUnavailable
		}
		txs = append(txs, tx)
		if budget.observer != nil {
			binary, err := nodeJSONRLPObserved(tx, len(row), budget.observer)
			if err != nil || !nodeJSONSliceObserved(1, 3, budget.observer) {
				return nil, ErrReplayUnavailable
			}
			txBytes = append(txBytes, binary)
		}
	}
	withdrawals, ok := nodeRawArrayObserved(m["withdrawals"], 16, budget.observer)
	if !ok {
		return nil, ErrReplayUnavailable
	}
	if !nodeJSONSliceObserved(len(withdrawals), 1, budget.observer) {
		return nil, ErrReplayUnavailable
	}
	ws := make(types.Withdrawals, 0, len(withdrawals))
	var withdrawalBytes [][]byte
	if budget.observer != nil {
		if !nodeJSONSliceObserved(len(withdrawals), 3, budget.observer) {
			return nil, ErrReplayUnavailable
		}
		withdrawalBytes = make([][]byte, 0, len(withdrawals))
	}
	for _, row := range withdrawals {
		v, valid := nodeJSONFieldsObserved(row, "index validatorIndex address amount", "index validatorIndex address amount", budget.observer)
		var withdrawal types.Withdrawal
		if !valid || providerReplayUnmarshal(row, &withdrawal, budget.observer) != nil || !nodeJSONCanonicalObserved(v, withdrawal, "", len(row), budget.observer) {
			return nil, ErrReplayUnavailable
		}
		if !nodeJSONSliceObserved(1, 1, budget.observer) {
			return nil, ErrReplayUnavailable
		}
		ws = append(ws, &withdrawal)
		if budget.observer != nil {
			binary, err := nodeJSONRLPObserved(&withdrawal, len(row), budget.observer)
			if err != nil || !nodeJSONSliceObserved(1, 3, budget.observer) {
				return nil, ErrReplayUnavailable
			}
			withdrawalBytes = append(withdrawalBytes, binary)
		}
	}
	if budget.observer != nil {
		if !nodeJSONRootsObserved(ctx, txBytes, withdrawalBytes, &header, budget.observer) {
			return nil, ErrReplay
		}
	} else if header.WithdrawalsHash == nil || types.DeriveSha(ws, trie.NewStackTrie(nil)) != *header.WithdrawalsHash || types.DeriveSha(types.Transactions(txs), trie.NewStackTrie(nil)) != header.TxHash {
		return nil, ErrReplay
	}
	var encoded []byte
	var err error
	if budget.observer != nil {
		encoded, err = providerHistoryNativeBlockBytesObserved(&header, txs, ws, len(raw), budget.observer)
	} else {
		block := types.NewBlockWithHeader(&header).WithBody(types.Body{Transactions: txs, Withdrawals: ws})
		encoded, err = nodeJSONRLPObserved(block, len(raw), nil)
	}
	if err != nil || !budget.charge(len(encoded)) {
		return nil, ErrReplayUnavailable
	}
	if value, present := m["size"]; present {
		text, valid := nodeStringObserved(value, budget.observer)
		size, quantity := rpcQuantityObserved(text, budget.observer)
		if !valid || !quantity || size != uint64(len(encoded)) {
			return nil, ErrReplay
		}
	}
	if budget.observer != nil && !budget.observer.preflightWork(candidateWorkVector{}) {
		return nil, ErrReplayUnavailable
	}
	return encoded, nil
}

func nodeJSONTransaction(raw json.RawMessage, header *types.Header, index uint64) (*types.Transaction, error) {
	return nodeJSONTransactionObserved(raw, header, index, nil)
}

func nodeJSONTransactionObserved(raw json.RawMessage, header *types.Header, index uint64, meter *providerWorkMeter) (*types.Transaction, error) {
	if header == nil || header.Number == nil {
		return nil, ErrReplayUnavailable
	}
	bound, eligible := providerHistoryNativeInputBoundObserved(meter)
	if !eligible {
		return nil, ErrReplayUnavailable
	}
	// This precedes typed decode and the early sender/hash calls, including
	// JSON rows whose later location or canonical binding will refuse.
	if bound.collection && uint64(len(raw)) > bound.body {
		meter.refuse()
		return nil, ErrReplayUnavailable
	}
	const fields = "type chainId nonce to gas gasPrice maxPriorityFeePerGas maxFeePerGas maxFeePerBlobGas value input accessList blobVersionedHashes authorizationList v r s yParity hash from blockHash blockNumber blockTimestamp transactionIndex"
	m, ok := nodeJSONFieldsObserved(raw, fields, "type nonce gas value input r s hash from blockHash blockNumber transactionIndex", meter)
	var tx types.Transaction
	if !ok || len(m["to"]) == 0 || !nodeJSONTransactionListsObserved(m, meter) || providerReplayUnmarshal(raw, &tx, meter) != nil || tx.Type() > types.SetCodeTxType {
		return nil, ErrReplayUnavailable
	}
	sender, err := nodeSenderObserved(&tx, meter)
	from, _ := nodeStringObserved(m["from"], meter)
	address, addressOK := nodeAddressTextObserved(sender, meter)
	headerHash, headerOK := nodeHeaderHashObserved(header, meter)
	txHash, txOK := nodeTransactionHashObserved(&tx, meter)
	if err != nil || !addressOK || !headerOK || !txOK || !providerReplayTextEqual(from, address, meter) || !nodeJSONLocationObserved(m, headerHash, header.Number.Uint64(), txHash, index, meter) {
		return nil, ErrReplay
	}
	if value, present := m["blockTimestamp"]; present {
		text, valid := nodeStringObserved(value, meter)
		timestamp, quantity := rpcQuantityObserved(text, meter)
		if !valid || !quantity || timestamp != header.Time {
			return nil, ErrReplay
		}
	}
	// This unsigned RPC field is not part of a dynamic-fee transaction's RLP.
	// The selected provider has reported its signed fee cap here instead of the
	// effective price. Retain that distinction in Exchanges: accept only these
	// two bound representations, and never use either as a root-gas witness.
	if tx.Type() >= types.DynamicFeeTxType && len(m["gasPrice"]) != 0 && !nodeLiteralObserved(m["gasPrice"], "null", meter) {
		text, valid := nodeStringObserved(m["gasPrice"], meter)
		price, quantity := nodeValueObserved(text, meter)
		want := new(big.Int).Add(header.BaseFee, tx.GasTipCap())
		if want.Cmp(tx.GasFeeCap()) > 0 {
			want.Set(tx.GasFeeCap())
		}
		if !valid || !quantity || text != "0x"+price.Text(16) || price.Cmp(want) != 0 && price.Cmp(tx.GasFeeCap()) != 0 {
			return nil, ErrReplay
		}
		m["gasPrice"] = json.RawMessage("null")
	}
	if !nodeJSONCanonicalObserved(m, &tx, "from blockHash blockNumber blockTimestamp transactionIndex", len(raw), meter) {
		return nil, ErrReplayUnavailable
	}
	return &tx, nil
}

// Explicit encoding ceilings, separate from the count of block transactions.
// All nested lists are counted before the transaction codec allocates them.
func nodeJSONTransactionLists(m map[string]json.RawMessage) bool {
	return nodeJSONTransactionListsObserved(m, nil)
}

func nodeJSONTransactionListsObserved(m map[string]json.RawMessage, meter *providerWorkMeter) bool {
	for _, name := range []string{"accessList", "blobVersionedHashes", "authorizationList"} {
		value, present := m[name]
		if !present || nodeLiteralObserved(value, "null", meter) {
			continue
		}
		rows, ok := nodeRawArrayObserved(value, 16384, meter)
		if !ok {
			return false
		}
		for _, row := range rows {
			switch name {
			case "accessList":
				v, valid := nodeJSONFieldsObserved(row, "address storageKeys", "address storageKeys", meter)
				if !valid {
					return false
				}
				if _, valid := nodeRawArrayObserved(v["storageKeys"], 16384, meter); !valid {
					return false
				}
			case "blobVersionedHashes":
				hash, valid := nodeStringObserved(row, meter)
				if !valid || !lowerHash(hash) {
					return false
				}
			case "authorizationList":
				if _, valid := nodeJSONFieldsObserved(row, "chainId address nonce yParity r s", "chainId address nonce yParity r s", meter); !valid {
					return false
				}
			}
		}
	}
	return meter == nil || meter.preflightWork(candidateWorkVector{})
}

func nodeJSONLocation(m map[string]json.RawMessage, block common.Hash, number uint64, tx common.Hash, index uint64) bool {
	return nodeJSONLocationObserved(m, block, number, tx, index, nil)
}

func nodeJSONLocationObserved(m map[string]json.RawMessage, block common.Hash, number uint64, tx common.Hash, index uint64, meter *providerWorkMeter) bool {
	b, a := nodeStringObserved(m["blockHash"], meter)
	h, c := nodeStringObserved(m["transactionHash"], meter)
	if _, present := m["hash"]; present {
		h, c = nodeStringObserved(m["hash"], meter)
	}
	n, d := nodeStringObserved(m["blockNumber"], meter)
	numberValue, e := rpcQuantityObserved(n, meter)
	i, f := nodeStringObserved(m["transactionIndex"], meter)
	indexValue, g := rpcQuantityObserved(i, meter)
	blockText, blockOK := nodeHashTextObserved(block, meter)
	txText, txOK := nodeHashTextObserved(tx, meter)
	return a && c && d && e && f && g && blockOK && txOK && providerReplayTextEqual(b, blockText, meter) && providerReplayTextEqual(h, txText, meter) && numberValue == number && indexValue == index
}

func nodeJSONReceipts(ctx context.Context, raw json.RawMessage, block *types.Block, budget *replayBudget) ([][]byte, error) {
	if block == nil || budget == nil || ctx == nil || ctx.Err() != nil {
		return nil, ErrReplayUnavailable
	}
	var header *types.Header
	if budget.observer != nil {
		// Other observed DATA callers have only a native Block. Reconstruct
		// its canonical bytes under the same observer; no hidden Header copy.
		binary, err := nodeJSONRLPObserved(block, budget.limits.MaxEvidenceBytes, budget.observer)
		if err != nil {
			return nil, err
		}
		header, err = providerHistoryNativeHeaderObserved(ctx, binary, budget)
		if err != nil {
			return nil, err
		}
	} else {
		header = block.Header()
	}
	return nodeJSONReceiptsHeaderObserved(ctx, raw, block, header, budget)
}

func nodeJSONReceiptsHeaderObserved(ctx context.Context, raw json.RawMessage, block *types.Block, header *types.Header, budget *replayBudget) ([][]byte, error) {
	if !nodeJSONPreflight(ctx, raw, budget) || block == nil {
		return nil, ErrReplayUnavailable
	}
	if header == nil || header.Number == nil || !header.Number.IsUint64() || header.Number.Uint64() != block.NumberU64() || header.Time != block.Time() {
		return nil, ErrReplay
	}
	rows, ok := nodeRawArrayObserved(raw, budget.limits.MaxTransactions, budget.observer)
	if !ok || len(rows) != len(block.Transactions()) {
		return nil, ErrReplay
	}
	if !nodeJSONSliceObserved(len(rows), 3, budget.observer) {
		return nil, ErrReplayUnavailable
	}
	out := make([][]byte, 0, len(rows))
	blockHash, hashOK := nodeHeaderHashObserved(header, budget.observer)
	if !hashOK {
		return nil, ErrReplayUnavailable
	}
	var cumulative uint64
	var logIndex uint
	for i, row := range rows {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		const fields = "type root status cumulativeGasUsed logsBloom logs transactionHash contractAddress gasUsed effectiveGasPrice blockHash blockNumber transactionIndex from to blobGasUsed blobGasPrice"
		m, valid := nodeJSONFieldsObserved(row, fields, "type status cumulativeGasUsed logsBloom logs transactionHash gasUsed blockHash blockNumber transactionIndex", budget.observer)
		if !valid {
			return nil, ErrReplayUnavailable
		}
		logs, valid := nodeRawArrayObserved(m["logs"], budget.limits.MaxLogs-int(logIndex), budget.observer)
		if !valid {
			return nil, ErrReplayUnavailable
		}
		for _, encodedLog := range logs {
			const names = "address topics data blockNumber transactionHash transactionIndex blockHash logIndex removed blockTimestamp"
			v, present := nodeJSONFieldsObserved(encodedLog, names, "address topics data blockNumber transactionHash transactionIndex blockHash logIndex removed", budget.observer)
			if !present {
				return nil, ErrReplayUnavailable
			}
			if _, valid := nodeRawArrayObserved(v["topics"], 4, budget.observer); !valid {
				return nil, ErrReplayUnavailable
			}
			if value, present := v["blockTimestamp"]; present {
				text, valid := nodeStringObserved(value, budget.observer)
				timestamp, quantity := rpcQuantityObserved(text, budget.observer)
				if !valid || !quantity || timestamp != block.Time() {
					return nil, ErrReplay
				}
			}
		}
		var receipt types.Receipt
		typeText, typePresent := nodeStringObserved(m["type"], budget.observer)
		typeValue, typeQuantity := rpcQuantityObserved(typeText, budget.observer)
		txHash, txOK := nodeTransactionHashObserved(block.Transactions()[i], budget.observer)
		if !txOK || !typePresent || !typeQuantity || typeValue != uint64(block.Transactions()[i].Type()) || providerReplayUnmarshal(row, &receipt, budget.observer) != nil || receipt.Type != block.Transactions()[i].Type() || !nodeJSONLocationObserved(m, blockHash, block.NumberU64(), txHash, uint64(i), budget.observer) || receipt.Status > 1 || len(receipt.PostState) != 0 || receipt.CumulativeGasUsed < cumulative || receipt.GasUsed != receipt.CumulativeGasUsed-cumulative {
			return nil, ErrReplay
		}
		if len(logs) != len(receipt.Logs) {
			return nil, ErrReplayUnavailable
		}
		for j, encodedLog := range logs {
			if !budget.logsObserved(1) {
				return nil, ErrReplayUnavailable
			}
			const names = "address topics data blockNumber transactionHash transactionIndex blockHash logIndex removed blockTimestamp"
			v, present := nodeJSONFieldsObserved(encodedLog, names, "address topics data blockNumber transactionHash transactionIndex blockHash logIndex removed", budget.observer)
			entry := receipt.Logs[j]
			if !present || entry == nil || !nodeJSONLocationObserved(v, blockHash, block.NumberU64(), receipt.TxHash, uint64(i), budget.observer) || entry.Index != logIndex || entry.Removed || !nodeLiteralObserved(v["removed"], "false", budget.observer) {
				return nil, ErrReplay
			}
			logIndex++
		}
		encoded, err := nodeJSONRLPObserved(&receipt, len(row), budget.observer)
		if err != nil || !budget.charge(len(encoded)) {
			return nil, ErrReplayUnavailable
		}
		if !nodeJSONSliceObserved(1, 3, budget.observer) {
			return nil, ErrReplayUnavailable
		}
		out = append(out, encoded)
		cumulative = receipt.CumulativeGasUsed
	}
	if budget.observer != nil && !budget.observer.preflightWork(candidateWorkVector{}) {
		return nil, ErrReplayUnavailable
	}
	return out, nil
}
