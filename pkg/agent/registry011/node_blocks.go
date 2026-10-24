package registry011

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// NodeLimits are explicit acquisition budgets, never operating defaults. The
// ceilings are 256 requests, 64 MiB of total response envelopes, five minutes
// per acquisition, and five seconds per native callTracer request. The existing
// transport response/depth/time limits also apply. No trace Javascript is used.
type NodeLimits struct {
	Replay                           ReplayLimits
	MaxRequests, MaxWireBytes        int
	AcquisitionTimeout, TraceTimeout time.Duration
}

// NodeExchange retains exact successful response bytes and their fixed-method
// request parameters. These are public RPC evidence, not trust credentials.
type NodeExchange struct {
	Method           string
	Params, Response json.RawMessage
}

type NodeBlockEvidence struct {
	ClientVersion string
	Header        FinalizedHeader
	Block         ReplayBlock
	Exchanges     []NodeExchange
}

type nodeAcquisition struct {
	reads               *PinnedAgentReads
	limits              NodeLimits
	ctx                 context.Context
	remaining, requests int
	exchanges           []NodeExchange
	work                replayBudget
	clientVersion       string
}

func (r *PinnedAgentReads) nodeStart(ctx context.Context, limits NodeLimits) (*nodeAcquisition, context.CancelFunc, error) {
	if r == nil || r.rpc == nil || r.rpc.client == nil || ctx == nil || r.registry != SepoliaRegistryAddress || !validHeader(r.header) || !replayLimits(limits.Replay) || limits.MaxRequests < 1 || limits.MaxRequests > 256 || limits.MaxWireBytes < 1 || limits.MaxWireBytes > 64<<20 || limits.AcquisitionTimeout <= 0 || limits.AcquisitionTimeout > 5*time.Minute || limits.TraceTimeout <= 0 || limits.TraceTimeout > 5*time.Second || limits.TraceTimeout > limits.AcquisitionTimeout || limits.TraceTimeout > r.rpc.limits.RequestTimeout {
		return nil, nil, ErrConfiguration
	}
	child, cancel := context.WithTimeout(ctx, limits.AcquisitionTimeout)
	return &nodeAcquisition{reads: r, limits: limits, ctx: child, remaining: limits.MaxWireBytes, work: replayBudget{limits: limits.Replay}}, cancel, nil
}

func (a *nodeAcquisition) call(method string, params any) (json.RawMessage, error) {
	if err := a.ctx.Err(); err != nil {
		return nil, err
	}
	if a.requests >= a.limits.MaxRequests || a.remaining <= 0 {
		return nil, ErrReplayUnavailable
	}
	a.requests++
	raw, result, err := a.reads.rpc.callEnvelope(a.ctx, method, params, a.remaining)
	if err != nil {
		return nil, err
	}
	a.remaining -= len(raw)
	request, err := json.Marshal(params)
	if err != nil {
		return nil, ErrConfiguration
	}
	a.exchanges = append(a.exchanges, NodeExchange{method, request, raw})
	return result, nil
}

func (a *nodeAcquisition) network() error {
	raw, err := a.call("eth_chainId", []any{})
	if err != nil {
		return err
	}
	if chain, ok := nodeString(raw); !ok || chain != "0xaa36a7" {
		return ErrRPCState
	}
	genesis, err := a.header("0x0")
	if err != nil {
		return err
	}
	if genesis.Number != 0 || genesis.Hash != sepoliaGenesis {
		return ErrRPCState
	}
	raw, err = a.call("eth_call", []any{map[string]string{"to": SepoliaRegistryAddress, "data": "0x04ef898d"}, a.reads.block()})
	if err != nil {
		return err
	}
	values, _, err := rpcABI(raw, false, 128)
	if err != nil || string(values[0]) != "eip155:11155111:"+SepoliaRegistryAddress {
		return ErrRPCState
	}
	raw, err = a.call("web3_clientVersion", []any{})
	if err != nil {
		return err
	}
	var ok bool
	a.clientVersion, ok = nodeString(raw)
	if !ok || len(a.clientVersion) == 0 || len(a.clientVersion) > 256 {
		return ErrRPCState
	}
	return nil
}

func (a *nodeAcquisition) header(tag string) (FinalizedHeader, error) {
	raw, err := a.call("eth_getBlockByNumber", []any{tag, false})
	if err != nil {
		return FinalizedHeader{}, err
	}
	var v struct{ Hash, ParentHash, Number, Timestamp string }
	if json.Unmarshal(raw, &v) != nil {
		return FinalizedHeader{}, ErrRPCState
	}
	n, ok := rpcQuantity(v.Number)
	stamp, valid := rpcQuantity(v.Timestamp)
	h := FinalizedHeader{v.Hash, v.ParentHash, n, stamp}
	if !ok || !valid || !validHeader(h) {
		return FinalizedHeader{}, ErrRPCState
	}
	return h, nil
}

// ReadNodeBlock acquires one complete canonical block and consensus receipts,
// plus full call trees for every transaction emitting a Registry log. It does
// not attest the interval from deployment, finality, absence or Source readiness.
// A trusted official provider and separate durable continuity gates are required.
func (r *PinnedAgentReads) ReadNodeBlock(ctx context.Context, number uint64, limits NodeLimits) (*NodeBlockEvidence, error) {
	a, cancel, err := r.nodeStart(ctx, limits)
	if err != nil {
		return nil, err
	}
	defer cancel()
	if err := a.network(); err != nil {
		return nil, err
	}
	e, decoded, err := a.block(number)
	if err != nil {
		return nil, err
	}
	for i, receipt := range decoded.receipts {
		needed := false
		for _, l := range receipt.Logs {
			needed = needed || l.Address == common.HexToAddress(SepoliaRegistryAddress)
		}
		if needed {
			trace, err := a.execution(decoded, i)
			if err != nil {
				return nil, err
			}
			e.Block.Executions = append(e.Block.Executions, trace)
		}
	}
	if err := a.recheck(e.Header); err != nil {
		return nil, err
	}
	e.Exchanges = a.exchanges
	e.ClientVersion = a.clientVersion
	return e, nil
}

func (a *nodeAcquisition) block(number uint64) (*NodeBlockEvidence, replayDecodedBlock, error) {
	var zero replayDecodedBlock
	if number < sepoliaDeploymentBlock || number > a.reads.header.Number {
		return nil, zero, ErrRPCState
	}
	h, err := a.header("0x" + strconv.FormatUint(number, 16))
	if err != nil {
		return nil, zero, err
	}
	if h.Number != number || (number == a.reads.header.Number && h != a.reads.header) {
		return nil, zero, ErrRPCState
	}
	result, err := a.call("debug_getRawBlock", []any{h.Hash})
	if err != nil {
		return nil, zero, err
	}
	if len(result) > 2*a.limits.Replay.MaxEvidenceBytes+4 {
		return nil, zero, ErrReplayUnavailable
	}
	raw, ok := nodeData(result)
	if !ok || len(raw) > a.limits.Replay.MaxEvidenceBytes {
		return nil, zero, ErrReplayUnavailable
	}
	if !nodeBlockCount(raw, a.limits.Replay.MaxTransactions) {
		return nil, zero, ErrReplayUnavailable
	}
	var block types.Block
	if rlp.DecodeBytes(raw, &block) != nil {
		return nil, zero, ErrReplayUnavailable
	}
	canonical, err := rlp.EncodeToBytes(&block)
	if err != nil || !bytes.Equal(canonical, raw) || replayHeader(block.Header()) != h || len(block.Transactions()) > a.limits.Replay.MaxTransactions {
		return nil, zero, ErrReplay
	}
	if types.CalcUncleHash(block.Uncles()) != block.UncleHash() {
		return nil, zero, ErrReplay
	}
	if w := block.Header().WithdrawalsHash; w != nil {
		if types.DeriveSha(types.Withdrawals(block.Withdrawals()), trie.NewStackTrie(nil)) != *w {
			return nil, zero, ErrReplay
		}
	} else if block.Withdrawals() != nil {
		return nil, zero, ErrReplay
	}
	header, err := rlp.EncodeToBytes(block.Header())
	if err != nil {
		return nil, zero, ErrReplayUnavailable
	}
	out := &NodeBlockEvidence{Header: h, Block: ReplayBlock{HeaderRLP: header}}
	for _, tx := range block.Transactions() {
		b, err := tx.MarshalBinary()
		if err != nil {
			return nil, zero, ErrReplayUnavailable
		}
		out.Block.Transactions = append(out.Block.Transactions, b)
	}
	result, err = a.call("debug_getRawReceipts", []any{h.Hash})
	if err != nil {
		return nil, zero, err
	}
	receipts, ok := nodeRawArray(result, a.limits.Replay.MaxTransactions)
	if !ok || len(receipts) != len(out.Block.Transactions) {
		return nil, zero, ErrReplay
	}
	for _, receipt := range receipts {
		b, ok := nodeData(receipt)
		if !ok {
			return nil, zero, ErrReplayUnavailable
		}
		out.Block.Receipts = append(out.Block.Receipts, b)
	}
	decoded, err := replayDecodeBlock(a.ctx, out.Block, &a.work)
	if err != nil {
		return nil, zero, err
	}
	return out, decoded, nil
}

func (a *nodeAcquisition) execution(b replayDecodedBlock, index int) (ReplayExecution, error) {
	var zero ReplayExecution
	tx := b.txs[index]
	options := map[string]any{"tracer": "callTracer", "timeout": a.limits.TraceTimeout.String(), "reexec": 0, "tracerConfig": map[string]bool{"onlyTopCall": false, "withLog": true}}
	result, err := a.call("debug_traceTransaction", []any{tx.Hash().Hex(), options})
	if err != nil {
		return zero, err
	}
	f, err := DecodeCallTracer(a.ctx, result, a.limits.Replay)
	if err != nil {
		return zero, err
	}
	e := ReplayExecution{BlockHash: replayHeader(b.header).Hash, TransactionHash: tx.Hash().Hex(), TransactionIndex: uint64(index), Root: f}
	if _, err := replayExecution(a.ctx, &e, b, index, &a.work); err != nil {
		return zero, err
	}
	return e, nil
}

func (a *nodeAcquisition) recheck(block FinalizedHeader) error {
	for _, expected := range []FinalizedHeader{block, a.reads.header} {
		h, err := a.header("0x" + strconv.FormatUint(expected.Number, 16))
		if err != nil {
			return err
		}
		if h != expected {
			return ErrAncestry
		}
	}
	f, err := a.header("finalized")
	if err != nil {
		return err
	}
	if f.Number < a.reads.header.Number || (f.Number == a.reads.header.Number && f != a.reads.header) {
		return ErrAncestry
	}
	return a.ctx.Err()
}

func nodeHex(b []byte) string { return "0x" + hex.EncodeToString(b) }

// Count transaction RLP values before types.Block allocates transaction objects.
func nodeBlockCount(raw []byte, maximum int) bool {
	body, rest, err := rlp.SplitList(raw)
	if err != nil || len(rest) != 0 {
		return false
	}
	_, _, tail, err := rlp.Split(body)
	if err != nil {
		return false
	}
	txs, _, err := rlp.SplitList(tail)
	if err != nil {
		return false
	}
	count, err := rlp.CountValues(txs)
	return err == nil && count <= maximum
}
