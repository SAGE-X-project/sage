package registry011

import (
	"bytes"
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// ReplayExecution is an UNQUALIFIED normalized-adapter boundary. It is not raw
// Geth JSON and has no completeness/trust boolean or default/live adapter.
// Future Source enrollment must authenticate the node, format, full call/log
// coverage and exact raw evidence. Synthetic tests do not qualify that Source.
type ReplayExecution struct {
	BlockHash, TransactionHash string
	TransactionIndex           uint64
	Root                       ReplayFrame
}
type ReplayFrame struct {
	// From means the context address that initiated this call (Geth OnEnter's
	// from), NOT the child's EVM msg.sender. A DELEGATECALL inherits msg.sender
	// but its initiating From is still the parent's storage/context address.
	// A qualified adapter must supply exactly this meaning for every call type.
	Type, From, To, Value string
	Input                 []byte
	Error                 string
	Calls                 []ReplayFrame
	Logs                  []ReplayFrameLog
}
type ReplayFrameLog struct {
	Address string
	Topics  []common.Hash
	Data    []byte
	// Number of immediate child calls completed before this own-frame log.
	// Together with the full children this encodes the original log order.
	AfterChildren int
}
type replayLog struct {
	log      *types.Log
	frame    *ReplayFrame
	eligible bool
}

func replayValue(s string) (*big.Int, bool) {
	if len(s) < 3 || len(s) > 66 || !strings.HasPrefix(s, "0x") || (len(s) > 3 && s[2] == '0') {
		return nil, false
	}
	for _, c := range []byte(s[2:]) {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return nil, false
		}
	}
	v, ok := new(big.Int).SetString(s[2:], 16)
	return v, ok
}
func replayAddress(a common.Address) string { return strings.ToLower(a.Hex()) }

func replayExecution(ctx context.Context, e *ReplayExecution, block replayDecodedBlock, index int, b *replayBudget) ([]replayLog, error) {
	tx := block.txs[index]
	receipt := block.receipts[index]
	if e.TransactionIndex != uint64(index) || e.BlockHash != strings.ToLower(block.header.Hash().Hex()) || e.TransactionHash != strings.ToLower(tx.Hash().Hex()) || receipt.Status != 1 {
		return nil, ErrReplay
	}
	if tx.Type() > types.DynamicFeeTxType || !tx.Protected() || tx.ChainId().Cmp(big.NewInt(11155111)) != 0 {
		return nil, ErrReplayUnavailable
	}
	sender, err := types.Sender(types.LatestSignerForChainID(big.NewInt(11155111)), tx)
	if err != nil {
		return nil, ErrReplay
	}
	root := &e.Root
	value, ok := replayValue(root.Value)
	if !ok || root.From != replayAddress(sender) || value.Cmp(tx.Value()) != 0 || !bytes.Equal(root.Input, tx.Data()) || root.Error != "" {
		return nil, ErrReplay
	}
	to := crypto.CreateAddress(sender, tx.Nonce())
	kind := "CREATE"
	if tx.To() != nil {
		to = *tx.To()
		kind = "CALL"
	}
	if root.Type != kind || root.To != replayAddress(to) {
		return nil, ErrReplay
	}
	logs, err := replayFrame(ctx, root, "", false, true, 0, b)
	if err != nil {
		return nil, err
	}
	if len(logs) != len(receipt.Logs) {
		return nil, ErrReplay
	}
	for i, l := range logs {
		r := receipt.Logs[i]
		if l.log.Address != r.Address || !bytes.Equal(l.log.Data, r.Data) || len(l.log.Topics) != len(r.Topics) {
			return nil, ErrReplay
		}
		for j, t := range l.log.Topics {
			if t != r.Topics[j] {
				return nil, ErrReplay
			}
		}
	}
	return logs, nil
}

func replayFrame(ctx context.Context, f *ReplayFrame, parent string, static, committed bool, depth int, b *replayBudget) ([]replayLog, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if depth > 32 || b.frames >= b.limits.MaxFrames || !b.charge(len(f.Type)+len(f.From)+len(f.To)+len(f.Value)+len(f.Input)+len(f.Error)) {
		return nil, ErrReplayUnavailable
	}
	b.frames++
	if len(f.Calls) > b.limits.MaxFrames-b.frames || len(f.Logs) > b.limits.MaxLogs-b.traceLogs {
		return nil, ErrReplayUnavailable
	}
	b.traceLogs += len(f.Logs)
	if !contentAddress(f.From) || !contentAddress(f.To) {
		return nil, ErrReplay
	}
	value, ok := replayValue(f.Value)
	if !ok || (parent != "" && f.From != parent) {
		return nil, ErrReplay
	}
	contextAddress := f.To
	switch f.Type {
	case "CALL", "CREATE", "CREATE2":
	case "STATICCALL":
		if value.Sign() != 0 {
			return nil, ErrReplay
		}
		static = true
	case "DELEGATECALL", "CALLCODE":
		if parent == "" {
			return nil, ErrReplay
		}
		contextAddress = parent
	default:
		return nil, ErrReplayUnavailable
	}
	committed = committed && f.Error == ""
	eligible := committed && !static && f.Type == "CALL" && contextAddress == SepoliaRegistryAddress && value.Sign() == 0
	out := []replayLog{}
	previous := 0
	child := 0
	appendChildren := func(until int) error {
		for child < until {
			ls, e := replayFrame(ctx, &f.Calls[child], contextAddress, static, committed, depth+1, b)
			if e != nil {
				return e
			}
			out = append(out, ls...)
			child++
		}
		return nil
	}
	for _, l := range f.Logs {
		if l.AfterChildren < previous || l.AfterChildren > len(f.Calls) || l.Address != contextAddress || len(l.Topics) > 4 {
			return nil, ErrReplay
		}
		if !b.charge(len(l.Address) + 32*len(l.Topics) + len(l.Data)) {
			return nil, ErrReplayUnavailable
		}
		if e := appendChildren(l.AfterChildren); e != nil {
			return nil, e
		}
		previous = l.AfterChildren
		if committed {
			out = append(out, replayLog{&types.Log{Address: common.HexToAddress(l.Address), Topics: append([]common.Hash{}, l.Topics...), Data: bytes.Clone(l.Data)}, f, eligible})
		}
	}
	if e := appendChildren(len(f.Calls)); e != nil {
		return nil, e
	}
	return out, nil
}
