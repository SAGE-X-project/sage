package registry011

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
)

// ReadReplayBlock adds strictly checked fork gas observations to a complete
// block. The approved provider supplies uncommitted execution/state dimensions;
// successful consistency checking is NOT independent EVM/consensus validation,
// a complete deployment-to-anchor history, or a Source-ready credential.
// No partial block is returned on any refusal, recheck or exhausted budget.
func (r *PinnedAgentReads) ReadReplayBlock(ctx context.Context, number uint64, limits NodeLimits) (*NodeBlockEvidence, error) {
	a, cancel, err := r.nodeStart(ctx, limits)
	if err != nil {
		return nil, err
	}
	defer cancel()
	if err := a.network(); err != nil {
		return nil, err
	}
	out, decoded, err := a.block(number)
	if err != nil {
		return nil, err
	}
	if decoded.header.Time >= sepoliaAmsterdamTime {
		options := map[string]any{"tracer": "stateGasTracer", "timeout": a.limits.TraceTimeout.String(), "reexec": 0}
		trace, err := a.call("debug_traceBlockByHash", []any{out.Header.Hash, options})
		if err != nil {
			return nil, err
		}
		profile, err := replayGasProfile(trace, a.work.limits.MaxTransactions)
		if err != nil {
			return nil, err
		}
		out.Block.Gas = &ReplayGasEvidence{profile, out.Header.Hash, trace}
	}
	if err := replayCheckGas(a.ctx, out.Block, decoded, &a.work); err != nil {
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
			out.Block.Executions = append(out.Block.Executions, trace)
		}
	}
	if err := a.recheck(out.Header); err != nil {
		return nil, err
	}
	out.Exchanges = a.exchanges
	out.ClientVersion = a.clientVersion
	return out, nil
}
