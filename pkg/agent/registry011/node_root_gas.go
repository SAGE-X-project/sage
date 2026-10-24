package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// Only the exact top-only result contract is normalized. Never accept a nested
// execution tree as a gas-only witness or discard unknown frame fields. The
// original RPC response remains separately retained by nodeAcquisition.call.
func nodeRootGas(ctx context.Context, trace json.RawMessage, block replayDecodedBlock, budget *replayBudget) (string, json.RawMessage, error) {
	if ctx == nil || ctx.Err() != nil || !budget.charge(len(trace)) {
		return "", nil, ErrReplayUnavailable
	}
	d := json.NewDecoder(bytes.NewReader(trace))
	d.UseNumber()
	if !readRPCValue(d, 0, 4) {
		return "", nil, ErrReplayUnavailable
	}
	if _, e := d.Token(); e != io.EOF {
		return "", nil, ErrReplayUnavailable
	}
	rows, ok := nodeRawArray(trace, budget.limits.MaxTransactions)
	if !ok || len(rows) != len(block.txs) {
		return "", nil, ErrReplayUnavailable
	}
	type rootWitness struct {
		TxHash string                     `json:"txHash"`
		Result map[string]json.RawMessage `json:"result"`
	}
	result := make([]rootWitness, 0, len(rows))
	field, profile := "", SepoliaAmsterdamRootRegularGasProfile
	for i, raw := range rows {
		if e := ctx.Err(); e != nil {
			return "", nil, e
		}
		m, valid := nodeObject(raw, "txHash result")
		if !valid || len(m) != 2 {
			return "", nil, ErrReplayUnavailable
		}
		hash, valid := nodeString(m["txHash"])
		if !valid || hash != block.txs[i].Hash().Hex() {
			return "", nil, ErrReplay
		}
		v, valid := nodeObject(m["result"], "type from to gas gasUsed input output error revertReason value regularGasUsed executionGasUsed stateGasUsed gasRefund stateGasReservoir")
		if !valid || !nodeRootContext(v, block.txs[i], block.receipts[i].Status) {
			return "", nil, ErrReplayUnavailable
		}
		_, regular := v["regularGasUsed"]
		_, execution := v["executionGasUsed"]
		if regular == execution {
			return "", nil, ErrReplayUnavailable
		}
		selected := "regularGasUsed"
		if execution {
			selected = "executionGasUsed"
		}
		if field == "" {
			field = selected
		}
		if field != selected {
			return "", nil, ErrReplayUnavailable
		}
		dimensions := map[string]json.RawMessage{}
		for _, name := range []string{"gasUsed", field, "stateGasUsed", "gasRefund"} {
			s, valid := nodeString(v[name])
			if _, quantity := rpcQuantity(s); !valid || !quantity {
				return "", nil, ErrReplayUnavailable
			}
			dimensions[name] = bytes.Clone(v[name])
		}
		result = append(result, rootWitness{hash, dimensions})
	}
	if field == "executionGasUsed" {
		profile = SepoliaAmsterdamRootExecutionGasProfile
	}
	encoded, e := json.Marshal(result)
	if e != nil {
		return "", nil, e
	}
	return profile, encoded, nil
}

func nodeRootContext(m map[string]json.RawMessage, tx *types.Transaction, status uint64) bool {
	from, a := nodeString(m["from"])
	kind, b := nodeString(m["type"])
	gas, c := nodeString(m["gas"])
	limit, q := rpcQuantity(gas)
	input, d := nodeData(m["input"])
	value, e := nodeString(m["value"])
	amount, v := replayValue(value)
	sender, err := types.Sender(types.LatestSignerForChainID(big.NewInt(11155111)), tx)
	if status > 1 || !a || !b || !c || !q || !d || !e || !v || err != nil || from != replayAddress(sender) || limit != tx.Gas() || !bytes.Equal(input, tx.Data()) || amount.Cmp(tx.Value()) != 0 {
		return false
	}
	to, present := m["to"]
	if tx.To() != nil {
		address, valid := nodeString(to)
		if kind != "CALL" || !present || !valid || address != replayAddress(*tx.To()) {
			return false
		}
	} else {
		if kind != "CREATE" || !present && status != 0 {
			return false
		}
		if present {
			address, valid := nodeString(to)
			if !valid || address != replayAddress(crypto.CreateAddress(sender, tx.Nonce())) {
				return false
			}
		}
	}
	if raw, present := m["output"]; present {
		if _, valid := nodeData(raw); !valid {
			return false
		}
	}
	_, failed := m["error"]
	if (status == 0) != failed {
		return false
	}
	for _, name := range []string{"error", "revertReason"} {
		if raw, present := m[name]; present {
			s, valid := nodeString(raw)
			if !valid || len(s) > 4096 || status != 0 || name == "error" && s == "" {
				return false
			}
		}
	}
	if raw, present := m["stateGasReservoir"]; present {
		s, valid := nodeString(raw)
		if _, quantity := rpcQuantity(s); !valid || !quantity {
			return false
		}
	}
	return true
}
