package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math/big"
	"unicode/utf8"

	"github.com/ethereum/go-ethereum/core/types"
)

// SepoliaAmsterdamGasProfile identifies a pinned provider-observation format,
// not a client-version assertion or an attestation of consensus validity.
// Activation and implementation revisions are documented in registry-source-checks.md.
const SepoliaAmsterdamGasProfile = "sepolia-amsterdam-reth-state-gas-v1"

// Nethermind reports RAW capped refunds, including at a clipped payment floor.
// This is a distinct data contract; it must not inherit Reth's effective-refund
// inference. A known schema is not evidence of the actual node binary identity.
const SepoliaAmsterdamNethermindGasProfile = "sepolia-amsterdam-nethermind-state-gas-v1"

// Root-call witnesses are DERIVED from retained full native callTracer roots.
// Each profile preserves its exact dimension name. Both use the conservative
// refund-floor subset; neither guesses raw refunds from a clipped payment.
const SepoliaAmsterdamRootRegularGasProfile = "sepolia-amsterdam-root-regular-gas-v1"
const SepoliaAmsterdamRootExecutionGasProfile = "sepolia-amsterdam-root-execution-gas-v1"
const sepoliaAmsterdamTime uint64 = 1791294816

// ReplayGasEvidence carries the COMPLETE ordered gas result for its exact
// profile. Root-call profiles are derived witnesses whose original responses
// must also be retained. Gas dimensions are uncommitted RPC assertions:
// roots authenticate transactions/receipts, not execution/state gas individually.
// Source enrollment must authenticate acquisition and retain the original exchange.
type ReplayGasEvidence struct {
	Profile, BlockHash string
	Trace              json.RawMessage
}

func replayCheckGas(ctx context.Context, raw ReplayBlock, block replayDecodedBlock, budget *replayBudget) error {
	h := block.header
	if h.GasUsed > h.GasLimit {
		return ErrReplay
	}
	if h.Time < sepoliaAmsterdamTime {
		var paid uint64
		if len(block.receipts) > 0 {
			paid = block.receipts[len(block.receipts)-1].CumulativeGasUsed
		}
		if raw.Gas != nil || paid != h.GasUsed {
			return ErrReplay
		}
		return nil
	}
	// Never infer compatibility from equal header/receipt totals after activation.
	if raw.Gas == nil {
		return ErrReplay
	}
	g := raw.Gas
	field := "executionGasUsed"
	if g.Profile == SepoliaAmsterdamNethermindGasProfile || g.Profile == SepoliaAmsterdamRootRegularGasProfile {
		field = "regularGasUsed"
	} else if g.Profile != SepoliaAmsterdamGasProfile && g.Profile != SepoliaAmsterdamRootExecutionGasProfile {
		return ErrReplayUnavailable
	}
	if g.BlockHash != replayHeader(h).Hash {
		return ErrReplay
	}
	if !budget.charge(len(g.Profile)+len(g.BlockHash)+len(g.Trace)) || len(g.Trace) == 0 || !utf8.Valid(g.Trace) || !validRPCUnicode(g.Trace) {
		return ErrReplayUnavailable
	}
	d := json.NewDecoder(bytes.NewReader(g.Trace))
	d.UseNumber()
	if !readRPCValue(d, 0, 4) {
		return ErrReplayUnavailable
	}
	if _, err := d.Token(); err != io.EOF {
		return ErrReplayUnavailable
	}
	entries, ok := nodeRawArray(g.Trace, budget.limits.MaxTransactions)
	if !ok || len(entries) != len(block.txs) {
		return ErrReplay
	}
	var execution, state, paid uint64
	for i, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		m, ok := nodeObject(entry, "txHash result")
		if !ok || len(m) != 2 {
			return ErrReplayUnavailable
		}
		hash, ok := nodeString(m["txHash"])
		if !ok || hash != block.txs[i].Hash().Hex() {
			return ErrReplay
		}
		v, ok := nodeObject(m["result"], "gasUsed "+field+" stateGasUsed gasRefund")
		if !ok || len(v) != 4 {
			return ErrReplayUnavailable
		}
		var amounts [4]uint64
		for j, name := range []string{"gasUsed", field, "stateGasUsed", "gasRefund"} {
			s, valid := nodeString(v[name])
			amounts[j], ok = rpcQuantity(s)
			if !valid || !ok {
				return ErrReplayUnavailable
			}
		}
		tx := block.txs[i]
		floor, err := replayGasFloor(tx)
		if err != nil {
			return err
		}
		p, e, s, refund := amounts[0], amounts[1], amounts[2], amounts[3]
		// At the payment floor raw refund cannot generally reconstruct T. Keep
		// only the unambiguous supported zero-state/zero-refund floor shape.
		if g.Profile != SepoliaAmsterdamGasProfile && p == floor && (e != floor || s != 0 || refund != 0) {
			return ErrReplayUnavailable
		}
		// Inclusion reservations are checked BEFORE charging each dimension.
		// These bounds also make every following sum and subtraction safe.
		limit := tx.Gas()
		if limit > 1<<32-1 || limit > h.GasLimit-state || min(limit, 1<<24) > h.GasLimit-execution ||
			e > min(limit, 1<<24) || s > limit || p > limit || floor > min(limit, 1<<24) ||
			p != block.receipts[i].CumulativeGasUsed-paid || !replayGasSettlement(p, e, s, refund, floor, limit) {
			return ErrReplay
		}
		paid = block.receipts[i].CumulativeGasUsed
		execution += e
		state += s
	}
	if max(execution, state) != h.GasUsed {
		return ErrReplay
	}
	return nil
}

// Classify a bounded native data shape, not clientVersion or trust. The strict
// checker still validates EVERY row against exactly that profile. No alternate
// profile is attempted if validation fails. Empty results retain the old empty
// representation; complete block checks still require zero transactions/gas.
func replayGasProfile(trace json.RawMessage, maximum int) (string, error) {
	rows, ok := nodeRawArray(trace, maximum)
	if !ok {
		return "", ErrReplayUnavailable
	}
	if len(rows) == 0 {
		return SepoliaAmsterdamGasProfile, nil
	}
	m, ok := nodeObject(rows[0], "txHash result")
	if !ok || len(m) != 2 {
		return "", ErrReplayUnavailable
	}
	v, ok := nodeObject(m["result"], "gasUsed executionGasUsed regularGasUsed stateGasUsed gasRefund")
	if !ok || len(v) != 4 {
		return "", ErrReplayUnavailable
	}
	_, execution := v["executionGasUsed"]
	_, regular := v["regularGasUsed"]
	if execution == regular {
		return "", ErrReplayUnavailable
	}
	if regular {
		return SepoliaAmsterdamNethermindGasProfile, nil
	}
	return SepoliaAmsterdamGasProfile, nil
}

// replayGasSettlement checks consistency with EIP-8037 + EIP-7778 and the
// effective refund emitted by the pinned Reth tracer. It does not reconstruct
// an unreported refund counter or re-execute the EVM. Both floors matter.
func replayGasSettlement(paid, execution, state, refund, floor, limit uint64) bool {
	if paid < floor || execution < floor || refund > limit || paid > limit-refund {
		return false
	}
	if paid > floor || refund > 0 {
		// The payment floor is inactive (equality with a nonzero refund is allowed).
		total := paid + refund
		return state <= total && refund <= total/5 && execution == max(total-state, floor)
	}
	// At the payment floor the tracer may hide the raw refund. The narrow
	// zero-state case still commits exactly the needed dimensions. Refuse other
	// ambiguous settlements instead of inventing total-before-refund evidence.
	if state != 0 {
		return false
	}
	if execution == floor {
		return true
	}
	// With zero state and an execution value above the floor, T=execution.
	// A hidden raw refund must STRICTLY cross the floor and respect its 20% cap.
	return execution <= limit && execution-floor < execution/5
}

// The floor follows the pinned Amsterdam EIP-2780/7976/7981 decomposition.
// Signed transaction fields, including the recovered sender for self-transfers,
// supply the input. The parser's evidence bound caps all byte/access-list counts.
func replayGasFloor(tx *types.Transaction) (uint64, error) {
	if tx.Type() > types.SetCodeTxType || tx.Protected() && tx.ChainId().Cmp(big.NewInt(11155111)) != 0 {
		return 0, ErrReplayUnavailable
	}
	from, err := types.Sender(types.LatestSignerForChainID(big.NewInt(11155111)), tx)
	if err != nil {
		return 0, ErrReplay
	}
	base := uint64(12000)
	if tx.To() == nil {
		base += 12000
	} else if *tx.To() != from {
		base += 3000
		if tx.Value().Sign() != 0 {
			base += 6000
		}
	}
	bytes := uint64(len(tx.Data()))
	for _, entry := range tx.AccessList() {
		bytes += 20 + 32*uint64(len(entry.StorageKeys))
	}
	return base + 64*bytes, nil
}
