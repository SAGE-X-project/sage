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
	"github.com/ethereum/go-ethereum/rlp"
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
	if ctx == nil || ctx.Err() != nil || budget == nil || !budget.charge(len(raw)) || !utf8.Valid(raw) || !validRPCUnicode(raw) {
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
	m, ok := nodeJSONFields(raw, nodeJSONHeaderFields+" hash size totalDifficulty transactions uncles withdrawals", nodeJSONHeaderFields+" hash transactions uncles withdrawals")
	if !ok {
		return nil, ErrReplayUnavailable
	}
	for _, key := range strings.Fields(nodeJSONHeaderFields) {
		if len(m[key]) > 1030 {
			return nil, ErrReplayUnavailable
		}
	}
	for _, key := range strings.Fields("number gasLimit gasUsed timestamp blobGasUsed excessBlobGas slotNumber") {
		value, valid := nodeString(m[key])
		if _, quantity := rpcQuantity(value); !valid || !quantity {
			return nil, ErrReplayUnavailable
		}
	}
	for _, key := range []string{"difficulty", "baseFeePerGas"} {
		value, valid := nodeString(m[key])
		if _, quantity := replayValue(value); !valid || !quantity {
			return nil, ErrReplayUnavailable
		}
	}
	if value, present := m["totalDifficulty"]; present && !bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		text, valid := nodeString(value)
		if _, quantity := replayValue(text); !valid || !quantity {
			return nil, ErrReplayUnavailable
		}
	}
	var header types.Header
	if json.Unmarshal(raw, &header) != nil || !nodeJSONCanonical(m, header, "size totalDifficulty transactions uncles withdrawals") || replayHeader(&header) != selected || header.UncleHash != types.EmptyUncleHash {
		return nil, ErrReplay
	}
	uncles, ok := nodeRawArray(m["uncles"], 0)
	if !ok || len(uncles) != 0 {
		return nil, ErrReplayUnavailable
	}
	rows, ok := nodeRawArray(m["transactions"], budget.limits.MaxTransactions)
	if !ok {
		return nil, ErrReplayUnavailable
	}
	txs := make([]*types.Transaction, 0, len(rows))
	for i, row := range rows {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		tx, err := nodeJSONTransaction(row, &header, uint64(i))
		if err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	withdrawals, ok := nodeRawArray(m["withdrawals"], 16)
	if !ok {
		return nil, ErrReplayUnavailable
	}
	ws := make(types.Withdrawals, 0, len(withdrawals))
	for _, row := range withdrawals {
		v, valid := nodeJSONFields(row, "index validatorIndex address amount", "index validatorIndex address amount")
		var withdrawal types.Withdrawal
		if !valid || json.Unmarshal(row, &withdrawal) != nil || !nodeJSONCanonical(v, withdrawal, "") {
			return nil, ErrReplayUnavailable
		}
		ws = append(ws, &withdrawal)
	}
	if header.WithdrawalsHash == nil || types.DeriveSha(ws, trie.NewStackTrie(nil)) != *header.WithdrawalsHash || types.DeriveSha(types.Transactions(txs), trie.NewStackTrie(nil)) != header.TxHash {
		return nil, ErrReplay
	}
	block := types.NewBlockWithHeader(&header).WithBody(types.Body{Transactions: txs, Withdrawals: ws})
	encoded, err := rlp.EncodeToBytes(block)
	if err != nil || !budget.charge(len(encoded)) {
		return nil, ErrReplayUnavailable
	}
	if value, present := m["size"]; present {
		text, valid := nodeString(value)
		size, quantity := rpcQuantity(text)
		if !valid || !quantity || size != uint64(len(encoded)) {
			return nil, ErrReplay
		}
	}
	return encoded, nil
}

func nodeJSONTransaction(raw json.RawMessage, header *types.Header, index uint64) (*types.Transaction, error) {
	const fields = "type chainId nonce to gas gasPrice maxPriorityFeePerGas maxFeePerGas maxFeePerBlobGas value input accessList blobVersionedHashes authorizationList v r s yParity hash from blockHash blockNumber blockTimestamp transactionIndex"
	m, ok := nodeJSONFields(raw, fields, "type nonce gas value input r s hash from blockHash blockNumber transactionIndex")
	var tx types.Transaction
	if !ok || len(m["to"]) == 0 || !nodeJSONTransactionLists(m) || json.Unmarshal(raw, &tx) != nil || tx.Type() > types.SetCodeTxType {
		return nil, ErrReplayUnavailable
	}
	sender, err := types.Sender(types.LatestSignerForChainID(big.NewInt(11155111)), &tx)
	from, _ := nodeString(m["from"])
	if err != nil || from != replayAddress(sender) || !nodeJSONLocation(m, header.Hash(), header.Number.Uint64(), tx.Hash(), index) {
		return nil, ErrReplay
	}
	if value, present := m["blockTimestamp"]; present {
		text, valid := nodeString(value)
		timestamp, quantity := rpcQuantity(text)
		if !valid || !quantity || timestamp != header.Time {
			return nil, ErrReplay
		}
	}
	// Standard RPC adds the effective price to dynamic-fee transactions; this
	// metadata is not part of their signed payload or consensus RLP.
	if tx.Type() >= types.DynamicFeeTxType && len(m["gasPrice"]) != 0 && !bytes.Equal(bytes.TrimSpace(m["gasPrice"]), []byte("null")) {
		text, valid := nodeString(m["gasPrice"])
		price, quantity := replayValue(text)
		want := new(big.Int).Add(header.BaseFee, tx.GasTipCap())
		if want.Cmp(tx.GasFeeCap()) > 0 {
			want.Set(tx.GasFeeCap())
		}
		if !valid || !quantity || text != "0x"+price.Text(16) || price.Cmp(want) != 0 {
			return nil, ErrReplay
		}
		m["gasPrice"] = json.RawMessage("null")
	}
	if !nodeJSONCanonical(m, &tx, "from blockHash blockNumber blockTimestamp transactionIndex") {
		return nil, ErrReplayUnavailable
	}
	return &tx, nil
}

// Explicit encoding ceilings, separate from the count of block transactions.
// All nested lists are counted before the transaction codec allocates them.
func nodeJSONTransactionLists(m map[string]json.RawMessage) bool {
	for _, name := range []string{"accessList", "blobVersionedHashes", "authorizationList"} {
		value, present := m[name]
		if !present || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			continue
		}
		rows, ok := nodeRawArray(value, 16384)
		if !ok {
			return false
		}
		for _, row := range rows {
			switch name {
			case "accessList":
				v, valid := nodeJSONFields(row, "address storageKeys", "address storageKeys")
				if !valid {
					return false
				}
				if _, valid := nodeRawArray(v["storageKeys"], 16384); !valid {
					return false
				}
			case "blobVersionedHashes":
				hash, valid := nodeString(row)
				if !valid || !lowerHash(hash) {
					return false
				}
			case "authorizationList":
				if _, valid := nodeJSONFields(row, "chainId address nonce yParity r s", "chainId address nonce yParity r s"); !valid {
					return false
				}
			}
		}
	}
	return true
}

func nodeJSONLocation(m map[string]json.RawMessage, block common.Hash, number uint64, tx common.Hash, index uint64) bool {
	b, a := nodeString(m["blockHash"])
	h, c := nodeString(m["transactionHash"])
	if _, present := m["hash"]; present {
		h, c = nodeString(m["hash"])
	}
	n, d := nodeString(m["blockNumber"])
	numberValue, e := rpcQuantity(n)
	i, f := nodeString(m["transactionIndex"])
	indexValue, g := rpcQuantity(i)
	return a && c && d && e && f && g && b == block.Hex() && h == tx.Hex() && numberValue == number && indexValue == index
}

func nodeJSONReceipts(ctx context.Context, raw json.RawMessage, block *types.Block, budget *replayBudget) ([][]byte, error) {
	if !nodeJSONPreflight(ctx, raw, budget) || block == nil {
		return nil, ErrReplayUnavailable
	}
	rows, ok := nodeRawArray(raw, budget.limits.MaxTransactions)
	if !ok || len(rows) != len(block.Transactions()) {
		return nil, ErrReplay
	}
	out := make([][]byte, 0, len(rows))
	var cumulative uint64
	var logIndex uint
	for i, row := range rows {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		const fields = "type root status cumulativeGasUsed logsBloom logs transactionHash contractAddress gasUsed effectiveGasPrice blockHash blockNumber transactionIndex from to blobGasUsed blobGasPrice"
		m, valid := nodeJSONFields(row, fields, "type status cumulativeGasUsed logsBloom logs transactionHash gasUsed blockHash blockNumber transactionIndex")
		if !valid {
			return nil, ErrReplayUnavailable
		}
		logs, valid := nodeRawArray(m["logs"], budget.limits.MaxLogs-int(logIndex))
		if !valid {
			return nil, ErrReplayUnavailable
		}
		for _, encodedLog := range logs {
			const names = "address topics data blockNumber transactionHash transactionIndex blockHash logIndex removed blockTimestamp"
			v, present := nodeJSONFields(encodedLog, names, "address topics data blockNumber transactionHash transactionIndex blockHash logIndex removed")
			if !present {
				return nil, ErrReplayUnavailable
			}
			if _, valid := nodeRawArray(v["topics"], 4); !valid {
				return nil, ErrReplayUnavailable
			}
			if value, present := v["blockTimestamp"]; present {
				text, valid := nodeString(value)
				timestamp, quantity := rpcQuantity(text)
				if !valid || !quantity || timestamp != block.Time() {
					return nil, ErrReplay
				}
			}
		}
		var receipt types.Receipt
		typeText, typePresent := nodeString(m["type"])
		typeValue, typeQuantity := rpcQuantity(typeText)
		if !typePresent || !typeQuantity || typeValue != uint64(block.Transactions()[i].Type()) || json.Unmarshal(row, &receipt) != nil || receipt.Type != block.Transactions()[i].Type() || !nodeJSONLocation(m, block.Hash(), block.NumberU64(), block.Transactions()[i].Hash(), uint64(i)) || receipt.Status > 1 || len(receipt.PostState) != 0 || receipt.CumulativeGasUsed < cumulative || receipt.GasUsed != receipt.CumulativeGasUsed-cumulative {
			return nil, ErrReplay
		}
		if len(logs) != len(receipt.Logs) {
			return nil, ErrReplayUnavailable
		}
		for j, encodedLog := range logs {
			const names = "address topics data blockNumber transactionHash transactionIndex blockHash logIndex removed blockTimestamp"
			v, present := nodeJSONFields(encodedLog, names, "address topics data blockNumber transactionHash transactionIndex blockHash logIndex removed")
			entry := receipt.Logs[j]
			if !present || entry == nil || !nodeJSONLocation(v, block.Hash(), block.NumberU64(), receipt.TxHash, uint64(i)) || entry.Index != logIndex || entry.Removed || !bytes.Equal(bytes.TrimSpace(v["removed"]), []byte("false")) {
				return nil, ErrReplay
			}
			logIndex++
		}
		encoded, err := receipt.MarshalBinary()
		if err != nil || !budget.charge(len(encoded)) {
			return nil, ErrReplayUnavailable
		}
		out = append(out, encoded)
		cumulative = receipt.CumulativeGasUsed
	}
	return out, nil
}
