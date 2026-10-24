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
	return r.readReplayBlock(ctx, number, limits, false)
}

// ReadReplayBlockWithRootGas explicitly selects the bounded native callTracer
// top-only gas adapter. It never tries stateGasTracer, another provider or a
// less strict checker on failure. Original root results remain in Exchanges;
// Gas.Trace is a derived, transaction-bound witness, not the raw node response.
// Top-only gas results do not replace full call/log evidence for Registry effects.
func (r *PinnedAgentReads) ReadReplayBlockWithRootGas(ctx context.Context, number uint64, limits NodeLimits) (*NodeBlockEvidence, error) {
	return r.readReplayBlock(ctx, number, limits, true)
}

func (r *PinnedAgentReads) readReplayBlock(ctx context.Context, number uint64, limits NodeLimits, roots bool) (*NodeBlockEvidence, error) {
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
		if roots {
			options["tracer"] = "callTracer"
			options["tracerConfig"] = map[string]bool{"onlyTopCall": true, "withLog": false, "includePrecompiles": true}
		}
		trace, err := a.call("debug_traceBlockByHash", []any{out.Header.Hash, options})
		if err != nil {
			return nil, err
		}
		var profile string
		if roots {
			profile, trace, err = nodeRootGas(a.ctx, trace, decoded, &a.work)
		} else {
			profile, err = replayGasProfile(trace, a.work.limits.MaxTransactions)
		}
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
