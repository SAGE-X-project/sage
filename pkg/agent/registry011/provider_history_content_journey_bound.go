package registry011

import (
	"reflect"
	"strconv"
)

type providerHistoryContentJourneyLimits struct {
	Windows, Blocks, PendingBlocks uint64
	EvidenceBytes, BodyBytes       uint64
	ValueBytes, TransactionBytes   uint64
	Transactions, Frames, Logs     uint64
	Effects, NativeBlocks          uint64
}

type providerHistoryContentJourneyCost struct {
	projection, gas, trace, effects, control candidateWorkVector
	normal, incoming, envelope               candidateWorkVector
}

// Conditional mechanical DATA for ONE original monotone content journey.
// All children belong to the SAME original range, caps and native selector;
// canonical byte maxima must be established BEFORE their use. Successful and
// first-refusing prefixes are included without retrying a failed transition.
// No numeric result creates these bindings or an owner, observer or Source.
//
// Projection includes the original acquisition and semantic content decoder
// once. Gas includes both strict passes. Trace includes normalization and
// execution; effect control includes pending allocation/release and waits.
// Each event has one current incoming preflight, so the disjoint full actual
// rows join with their MAX outstanding room. This is a component composition,
// never a refund of initializer plus SUM FULL admitted stage reservations.
//
// Semantic event/command/claim/state/queue work, window/stop/history work,
// cursor roots/copies, outer roles and original resource/remaining composites
// remain separate. Native CPU/accessors/backing/peak/lifetime RAM, factory,
// feasibility and operating authority are not qualified by this private row.
func providerHistoryContentJourneyUpper(p providerHistoryContentJourneyLimits, encoding, profile string) (providerHistoryContentJourneyCost, bool) {
	var empty providerHistoryContentJourneyCost
	projection, projectionOK := providerHistoryBlockProjectionUpper(providerHistoryBlockProjectionLimits{
		p.Windows, p.Blocks, p.PendingBlocks, p.EvidenceBytes, p.BodyBytes,
		p.TransactionBytes, p.Transactions, p.Logs, p.NativeBlocks,
	}, encoding)
	gas, gasOK := providerHistoryGasJourneyUpper(providerHistoryGasJourneyLimits{
		p.Windows, p.Blocks, p.PendingBlocks, p.EvidenceBytes, p.BodyBytes,
		p.ValueBytes, p.TransactionBytes, p.Transactions, p.NativeBlocks,
	}, profile)
	trace, traceOK := providerHistoryTraceJourneyUpper(providerHistoryTraceJourneyLimits{
		p.Windows, p.Blocks, p.PendingBlocks, p.EvidenceBytes, p.BodyBytes,
		p.ValueBytes, p.TransactionBytes, p.Transactions, p.Frames, p.Logs, p.Effects,
	})
	effects, effectsOK := providerHistoryEffectReadyUpper(providerHistoryEffectReadyLimits{
		p.Windows, p.Blocks, p.PendingBlocks, p.EvidenceBytes, p.BodyBytes,
		p.TransactionBytes, p.ValueBytes, p.Transactions, p.Logs, p.Effects,
	})
	if !projectionOK || !gasOK || !traceOK || !effectsOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	out := providerHistoryContentJourneyCost{
		projection: projection.normal, gas: gas.normal, trace: trace.normal, effects: effects.normal,
	}
	// Original spine lookup is indexed, not a linear traversal. Its checked
	// two fixed hash strings cost133. A successful semantic block copies
	// FinalizedHeader+uint64+int, including its one event constant.
	header := uint64(reflect.TypeFor[FinalizedHeader]().Size())
	word := uint64(strconv.IntSize / 8)
	out.control[10] = a.product(p.Blocks, a.sum(header, word, 142))
	for c := range out.normal {
		out.normal[c] = a.sum(out.projection[c], out.gas[c], out.trace[c], out.effects[c], out.control[c])
		for _, part := range [...]providerHistoryControlPair{
			{projection.normal, projection.envelope}, {gas.normal, gas.envelope},
			{trace.normal, trace.envelope}, {effects.normal, effects.envelope},
		} {
			if part.envelope[c] < part.normal[c] {
				return empty, false
			}
			out.incoming[c] = max(out.incoming[c], part.envelope[c]-part.normal[c])
		}
		out.envelope[c] = a.sum(out.normal[c], out.incoming[c])
	}
	if a.failed {
		return empty, false
	}
	return out, true
}
