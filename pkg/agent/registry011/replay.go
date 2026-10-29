package registry011

import (
	"bytes"
	"context"
	"crypto/sha256"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ReplayInput is one complete acquisition, from the separately authenticated
// deployment block through the separately selected finalized Anchor. History
// and final record/grants must be separate block-pinned reads at that anchor.
// No absence, Source-ready credential, trust enrollment or durable state is
// produced. Input buffers must not be concurrently mutated. No buffers retained.
type ReplayInput struct {
	Registry, Agent string
	DeploymentBlock uint64
	Anchor          FinalizedHeader
	Blocks          []ReplayBlock
	History         []HistoryBytes
	Record, Grants  []byte
}

// ReplayCommands reconstructs every intermediate state for the selected DID,
// checking strict required-suite proofs, ACL/lifecycle and actual Registry
// caller/canonical calldata/logs. It also reconstructs all claim admissions
// and consumptions for the shared Registry over the complete block interval.
//
// Crucially, cryptographic content consistency does NOT establish finality,
// consensus, code identity or honesty/completeness of a trace normalizer. Those
// are mandatory separate Source gates. This function cannot mark a Source ready.
func ReplayCommands(ctx context.Context, in ReplayInput, limits ReplayLimits) ([]HistoryContentStep, error) {
	if ctx == nil || !replayLimits(limits) || in.Registry != "eip155:11155111:"+SepoliaRegistryAddress || !rpcAgent(in.Agent) || !validHeader(in.Anchor) {
		return nil, ErrConfiguration
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(in.Blocks) == 0 || len(in.Blocks) > limits.MaxBlocks || in.Anchor.Number < in.DeploymentBlock || in.Anchor.Number-in.DeploymentBlock != uint64(len(in.Blocks)-1) || len(in.History) == 0 || uint64(len(in.History)) > limits.History.MaxItems {
		return nil, ErrReplayUnavailable
	}
	b := replayBudget{limits: limits}
	if len(in.Record) == 0 || len(in.Record) > 65536 || len(in.Grants) == 0 || len(in.Grants) > 262144 || !b.charge(len(in.Record)) || !b.charge(len(in.Grants)) {
		return nil, ErrReplayUnavailable
	}
	for _, h := range in.History {
		if len(h.Entry) == 0 || len(h.Entry) > 262144 || len(h.Command) == 0 || len(h.Command) > 262144 || !b.charge(len(h.Entry)) || !b.charge(len(h.Command)) {
			return nil, ErrReplayUnavailable
		}
	}
	contract, outcomes, e := replayABIs()
	if e != nil {
		return nil, ErrConfiguration
	}
	r := replayRun{registry: in.Registry, agent: in.Agent, contract: contract, outcomes: outcomes, claims: map[replayClaimID]replayClaimState{}, created: map[string]bool{}, state: replayState{}, budget: &b, history: in.History, steps: []HistoryContentStep{}}
	var previous *types.Header
	for i, block := range in.Blocks {
		decoded, err := replayDecodeBlock(ctx, block, &b)
		if err != nil {
			return nil, err
		}
		h := decoded.header
		if h.Number.Uint64() != in.DeploymentBlock+uint64(i) || (previous != nil && (h.ParentHash != previous.Hash() || h.Time < previous.Time)) {
			return nil, ErrReplay
		}
		if err := r.block(ctx, block, decoded); err != nil {
			return nil, err
		}
		previous = h
	}
	if replayHeader(previous) != in.Anchor || len(r.steps) != len(in.History) {
		return nil, ErrReplay
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := CheckHistoryContent(in.Registry, in.Agent, in.Record, in.Grants, r.steps, limits.History); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return r.steps, nil
}
func replayHeader(h *types.Header) FinalizedHeader {
	return FinalizedHeader{Hash: strings.ToLower(h.Hash().Hex()), ParentHash: strings.ToLower(h.ParentHash.Hex()), Number: h.Number.Uint64(), Timestamp: h.Time}
}

type replayClaimID struct {
	actor  string
	digest [32]byte
}
type replayClaimState struct {
	block    uint64
	consumed bool
}
type replayRun struct {
	registry, agent          string
	contract, outcomes       abi.ABI
	claims                   map[replayClaimID]replayClaimState
	created                  map[string]bool
	state                    replayState
	budget                   *replayBudget
	history                  []HistoryBytes
	steps                    []HistoryContentStep
	baseVersion              uint64 // Zero for the existing complete-range API.
	claimLimit, createdLimit int    // Explicit lifetime work bounds for a journal.
}

func (r *replayRun) block(ctx context.Context, raw ReplayBlock, b replayDecodedBlock) error {
	traces := map[uint64]*ReplayExecution{}
	if len(raw.Executions) > len(b.txs) {
		return ErrReplay
	}
	for i := range raw.Executions {
		e := &raw.Executions[i]
		if e.TransactionIndex >= uint64(len(b.txs)) || traces[e.TransactionIndex] != nil {
			return ErrReplay
		}
		traces[e.TransactionIndex] = e
	}
	for index, receipt := range b.receipts {
		needed := false
		for _, l := range receipt.Logs {
			if l.Address == common.HexToAddress(SepoliaRegistryAddress) {
				needed = true
				break
			}
		}
		e := traces[uint64(index)]
		if !needed {
			if e != nil {
				return ErrReplay
			}
			continue
		}
		if e == nil {
			return ErrReplayUnavailable
		}
		logs, err := replayExecution(ctx, e, b, index, r.budget)
		if err != nil {
			return err
		}
		seen := map[*ReplayFrame]bool{}
		for _, l := range logs {
			if l.log.Address != common.HexToAddress(SepoliaRegistryAddress) {
				continue
			}
			if !l.eligible || seen[l.frame] {
				return ErrReplay
			}
			seen[l.frame] = true
			if err := r.event(ctx, l, b.header); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *replayRun) event(ctx context.Context, l replayLog, h *types.Header) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	log := l.log
	if len(log.Topics) != 3 {
		return ErrReplay
	}
	commit := r.outcomes.Events["ClaimCommitted"]
	changed := r.outcomes.Events["RecordChanged"]
	if log.Topics[0] == commit.ID {
		return r.commit(l, h, commit)
	}
	if log.Topics[0] != changed.ID {
		return ErrReplayUnavailable
	}
	c, ok := replayDecodeCommand(r.registry, l.frame.From, l.frame.Input, r.contract)
	if !ok {
		return ErrReplay
	}
	did := "did:sage:" + r.registry + ":" + c.agent
	didHash := sha256.Sum256([]byte(did))
	actor := common.BytesToHash(common.HexToAddress(c.actor).Bytes())
	if log.Topics[1] != common.Hash(didHash) || log.Topics[2] != actor {
		return ErrReplay
	}
	eventFields := changed.Inputs.NonIndexed()
	n, framed := replayABIFields(log.Data, eventFields)
	if !framed || n != len(log.Data) {
		return ErrReplay
	}
	values, e := eventFields.Unpack(log.Data)
	if e != nil {
		return ErrReplay
	}
	canonical, e := changed.Inputs.NonIndexed().Pack(values...)
	if e != nil || !bytes.Equal(canonical, log.Data) || values[0].(string) != did || values[1].(uint64) != c.expected || c.expected == ^uint64(0) || values[2].(uint64) != c.expected+1 || values[3].(string) != c.op || values[4].([32]byte) != sha256.Sum256(c.raw) || values[7].(uint64) != h.Time {
		return ErrReplay
	}
	if c.op == "create" {
		id := replayClaimID{c.actor, replayClaim(r.registry, c)}
		claim, present := r.claims[id]
		if r.created[c.agent] || !present || claim.consumed || h.Number.Uint64() <= claim.block || h.Number.Uint64()-claim.block > 256 {
			return ErrReplay
		}
		if r.createdLimit > 0 && len(r.created) >= r.createdLimit {
			return ErrReplayUnavailable
		}
		claim.consumed = true
		r.claims[id] = claim
		r.created[c.agent] = true
	}
	if c.agent != r.agent {
		return nil
	}
	i := len(r.steps)
	if i >= len(r.history) || uint64(i) >= ^uint64(0)-r.baseVersion || r.history[i].Version != r.baseVersion+uint64(i)+1 || !bytes.Equal(r.history[i].Command, c.raw) {
		return ErrReplay
	}
	if len(c.keys) > r.budget.limits.MaxProofs-r.budget.proofs {
		return ErrReplayUnavailable
	}
	r.budget.proofs += len(c.keys)
	step, ok := r.state.apply(r.registry, c, h.Time)
	if !ok || !bytes.Equal(step.History.Entry, r.history[i].Entry) || values[5].([32]byte) != sha256.Sum256(step.Record) || values[6].([32]byte) != sha256.Sum256(step.Grants) {
		return ErrReplay
	}
	if !r.budget.charge(len(step.Record) + len(step.Grants)) {
		return ErrReplayUnavailable
	}
	r.steps = append(r.steps, step)
	return nil
}

func (r *replayRun) commit(l replayLog, h *types.Header, event abi.Event) error {
	method := r.contract.Methods["commitClaim"]
	if len(l.frame.Input) != 36 || !bytes.Equal(l.frame.Input[:4], method.ID) || l.log.Topics[1] != common.BytesToHash(common.HexToAddress(l.frame.From).Bytes()) {
		return ErrReplay
	}
	digest := [32]byte(l.frame.Input[4:])
	if l.log.Topics[2] != common.Hash(digest) {
		return ErrReplay
	}
	expected, e := event.Inputs.NonIndexed().Pack(new(big.Int).Set(h.Number))
	if e != nil || !bytes.Equal(l.log.Data, expected) {
		return ErrReplay
	}
	id := replayClaimID{l.frame.From, digest}
	if _, present := r.claims[id]; present {
		return ErrReplay
	}
	if r.claimLimit > 0 && len(r.claims) >= r.claimLimit {
		return ErrReplayUnavailable
	}
	r.claims[id] = replayClaimState{block: h.Number.Uint64()}
	return nil
}
