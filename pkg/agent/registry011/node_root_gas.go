package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strconv"

	"github.com/ethereum/go-ethereum/core/types"
)

// Only the exact top-only result contract is normalized. Never accept a nested
// execution tree as a gas-only witness or discard unknown frame fields. The
// original RPC response remains separately retained by nodeAcquisition.call.
func nodeRootGas(ctx context.Context, trace json.RawMessage, block replayDecodedBlock, budget *replayBudget) (string, json.RawMessage, error) {
	if ctx == nil || ctx.Err() != nil || budget == nil || !budget.charge(len(trace)) || !budget.decodeObserved(trace) {
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
	observer := budget.observer
	rows, ok := nodeRawArrayObserved(trace, budget.limits.MaxTransactions, observer)
	if !ok || len(rows) != len(block.txs) {
		return "", nil, ErrReplayUnavailable
	}
	type rootWitness struct {
		TxHash string                     `json:"txHash"`
		Result map[string]json.RawMessage `json:"result"`
	}
	const witnessBytes = uint64(3 * (strconv.IntSize / 8))
	if observer != nil && !observer.event(providerCopyEvent, uint64(len(rows))*witnessBytes) {
		return "", nil, ErrReplayUnavailable
	}
	result := make([]rootWitness, 0, len(rows))
	field, profile := "", SepoliaAmsterdamRootRegularGasProfile
	for i, raw := range rows {
		if e := ctx.Err(); e != nil {
			return "", nil, e
		}
		m, valid := nodeJSONFieldsObserved(raw, "txHash result", "txHash result", observer)
		if !valid || len(m) != 2 {
			return "", nil, ErrReplayUnavailable
		}
		hash, valid := nodeStringObserved(m["txHash"], observer)
		txHash, hashOK := nodeTransactionHashObserved(block.txs[i], observer)
		txText, textOK := nodeHashTextObserved(txHash, observer)
		if !valid || !hashOK || !textOK || !providerReplayTextEqual(hash, txText, observer) {
			return "", nil, ErrReplay
		}
		v, valid := nodeObjectObserved(m["result"], "type from to gas gasUsed input output error revertReason value regularGasUsed executionGasUsed stateGasUsed gasRefund stateGasReservoir", observer)
		if !valid || !nodeRootContextObserved(v, block.txs[i], block.receipts[i].Status, observer) {
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
			s, valid := nodeStringObserved(v[name], observer)
			if _, quantity := rpcQuantityObserved(s, observer); !valid || !quantity {
				return "", nil, ErrReplayUnavailable
			}
			if observer != nil && !observer.event(providerCopyEvent, uint64(len(v[name]))) {
				return "", nil, ErrReplayUnavailable
			}
			dimensions[name] = bytes.Clone(v[name])
		}
		if observer != nil && !observer.event(providerCopyEvent, witnessBytes) {
			return "", nil, ErrReplayUnavailable
		}
		result = append(result, rootWitness{hash, dimensions})
	}
	if field == "executionGasUsed" {
		profile = SepoliaAmsterdamRootExecutionGasProfile
	}
	// Every output field is a validated subset of the same raw top-only JSON
	// rows. Hex values/hash and fixed field names cannot grow past that input.
	if observer != nil && !budget.encodePreflight(len(trace)) {
		return "", nil, ErrReplayUnavailable
	}
	encoded, e := json.Marshal(result)
	if e != nil {
		return "", nil, e
	}
	if observer != nil && !budget.encodedObserved(encoded, len(trace)) {
		return "", nil, ErrReplayUnavailable
	}
	return profile, encoded, nil
}

func nodeRootContext(m map[string]json.RawMessage, tx *types.Transaction, status uint64) bool {
	return nodeRootContextObserved(m, tx, status, nil)
}

func nodeRootContextObserved(m map[string]json.RawMessage, tx *types.Transaction, status uint64, observer *providerWorkMeter) bool {
	from, a := nodeStringObserved(m["from"], observer)
	kind, b := nodeStringObserved(m["type"], observer)
	gas, c := nodeStringObserved(m["gas"], observer)
	limit, q := rpcQuantityObserved(gas, observer)
	input, d := nodeDataObserved(m["input"], observer)
	value, e := nodeStringObserved(m["value"], observer)
	amount, v := nodeValueObserved(value, observer)
	if observer != nil && (!a || !b || !c || !q || !d || !e || !v || !observer.preflightWork(candidateWorkVector{})) {
		return false
	}
	// Recover from the original signature; a warm library cache is not the
	// original observer's evidence of an entered recovery.
	sender, err := nodeSenderObserved(tx, observer)
	address, addressOK := nodeAddressTextObserved(sender, observer)
	if status > 1 || !a || !b || !c || !q || !d || !e || !v || err != nil || !addressOK || !providerReplayTextEqual(from, address, observer) || limit != tx.Gas() || !providerReplayEqual(input, tx.Data(), observer) || amount.Cmp(tx.Value()) != 0 {
		return false
	}
	to, present := m["to"]
	if tx.To() != nil {
		address, valid := nodeStringObserved(to, observer)
		want, wantOK := nodeAddressTextObserved(*tx.To(), observer)
		if kind != "CALL" || !present || !valid || !wantOK || !providerReplayTextEqual(address, want, observer) {
			return false
		}
	} else {
		if kind != "CREATE" || !present && status != 0 {
			return false
		}
		if present {
			address, valid := nodeStringObserved(to, observer)
			created, createdOK := nodeCreateAddressObserved(sender, tx.Nonce(), observer)
			want, wantOK := nodeAddressTextObserved(created, observer)
			if !valid || !createdOK || !wantOK || !providerReplayTextEqual(address, want, observer) {
				return false
			}
		}
	}
	if raw, present := m["output"]; present {
		if _, valid := nodeDataObserved(raw, observer); !valid {
			return false
		}
	}
	_, failed := m["error"]
	if (status == 0) != failed {
		return false
	}
	for _, name := range []string{"error", "revertReason"} {
		if raw, present := m[name]; present {
			s, valid := nodeStringObserved(raw, observer)
			if !valid || len(s) > 4096 || status != 0 || name == "error" && s == "" {
				return false
			}
		}
	}
	if raw, present := m["stateGasReservoir"]; present {
		s, valid := nodeStringObserved(raw, observer)
		if _, quantity := rpcQuantityObserved(s, observer); !valid || !quantity {
			return false
		}
	}
	return true
}
