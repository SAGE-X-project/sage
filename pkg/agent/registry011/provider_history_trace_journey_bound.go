package registry011

import (
	"math/big"
	"reflect"
	"strconv"

	"github.com/ethereum/go-ethereum/core/types"
)

type providerHistoryTraceJourneyLimits struct {
	Windows, Blocks, PendingBlocks uint64
	EvidenceBytes, BodyBytes       uint64
	ValueBytes, TransactionBytes   uint64
	Transactions, Frames, Logs     uint64
	Effects                        uint64
}

type providerHistoryTraceJourneyCost struct {
	normalization, execution, assembly                    candidateWorkVector
	normal, envelope                                      candidateWorkVector
	normalizationAttempts, executionRoots, executionSlots uint64
	normalizationRaw, canonicalPool, receiptPool          uint64
}

// Conditional DATA for ONE original monotone content journey, including its
// first aborting local transition, without failed-S retries. Original selected
// caps, complete raw Unicode-checked results, native canonical V/H and the
// remaining semantic-window frame/log counters must be bound BEFORE use.
// Numerical success grants no observer, source, reservation or authority.
//
// Normalization is seeded from original semantic frames/logs and carries them
// over all ready-block roots. It can read one final raw prefix without a root
// frame; its internal raw length check uses E, while the preceding raw copy
// still needs C. Declared ReplayExecution slots are initialized BEFORE any
// normalization and cannot be replaced by the count of visited roots/frames.
//
// Only semantic execution uses the persistent canonical WE/WT pools. It enters
// each needed canonical receipt index once, after complete upfront decoding
// and after ALL that block's original roots normalized. The two traversals
// retain separate observed frame/log counts. Native CPU/accessors/backing/RAM,
// other callers/roles, full admitted stage sums, resources/factory, feasibility
// and operating authority remain separate. This is not a global pad refund.
func providerHistoryTraceJourneyUpper(p providerHistoryTraceJourneyLimits) (providerHistoryTraceJourneyCost, bool) {
	var empty providerHistoryTraceJourneyCost
	if p.Blocks == 0 || p.Blocks > 1<<20 || p.Windows == 0 || p.Windows > p.Blocks ||
		p.PendingBlocks == 0 || p.PendingBlocks > 10 || p.EvidenceBytes == 0 || p.EvidenceBytes > 64<<20 ||
		p.BodyBytes < 4 || p.BodyBytes > 8<<20 || p.TransactionBytes == 0 || p.TransactionBytes > 64<<20 ||
		p.ValueBytes == 0 || p.ValueBytes > min(p.EvidenceBytes, p.BodyBytes, p.TransactionBytes) ||
		p.Transactions == 0 || p.Transactions > 16384 || p.Frames == 0 || p.Frames > 65536 ||
		p.Logs == 0 || p.Logs > 65536 || p.Effects > 1<<20 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	windows, e, c, v, h := p.Windows, p.EvidenceBytes, p.BodyBytes, p.ValueBytes, p.TransactionBytes
	w := uint64(strconv.IntSize / 8)
	frames, logs := a.product(windows, p.Frames), a.product(windows, p.Logs)
	var out providerHistoryTraceJourneyCost
	out.normalizationAttempts = min(p.Effects, a.sum(frames, 1))
	out.executionRoots = min(p.Effects, a.product(windows, p.Transactions), frames)
	out.executionSlots = min(p.Effects, a.product(a.sum(windows, p.PendingBlocks), p.Transactions), a.product(p.Blocks, p.Transactions))
	out.normalizationRaw = a.product(min(c, e), out.normalizationAttempts)
	out.canonicalPool = min(a.product(windows, e), a.product(out.executionRoots, v), a.product(p.Blocks, min(c, e)))
	out.receiptPool = a.product(min(windows, out.executionRoots), e)
	nf, nl := min(frames, out.normalizationRaw), min(logs, out.normalizationRaw)
	ef, el := min(frames, a.product(out.executionRoots, c)), min(logs, a.product(out.executionRoots, c))
	out.normalization[2] = a.sum(a.product(138, out.normalizationRaw), a.product(64, nf), a.product(256, nl))
	out.normalization[5], out.normalization[6] = nf, nl
	out.normalization[10] = a.sum(a.product(a.sum(a.product(138, c), a.product(413, w), 803), out.normalizationRaw),
		a.product(a.sum(13062, a.product(19, w)), nf), a.product(a.sum(18167, a.product(42, w)), nl),
		a.product(4, out.normalizationAttempts))
	sidecar := a.sum(uint64(reflect.TypeFor[types.Transaction]().Size()), uint64(reflect.TypeFor[types.BlobTx]().Size()), 2)
	value := a.sum(a.product(a.sum(h, w-1)/w, w), uint64(reflect.TypeFor[big.Int]().Size()))
	row, log := uint64(reflect.TypeFor[replayLog]().Size()), uint64(reflect.TypeFor[types.Log]().Size())
	const header = uint64(65536)
	root := a.sum(a.product(header, header), header, sidecar, 100947, a.product(2, value), a.product(3, c), a.product(16, row))
	out.execution[2] = a.sum(a.product(161, out.executionRoots), a.product(64, ef), a.product(40, el))
	out.execution[5], out.execution[6] = ef, el
	out.execution[10] = a.sum(a.product(out.executionRoots, root),
		a.product(a.sum(a.product(2, v), 516), out.canonicalPool), out.receiptPool,
		a.product(4433, ef), a.product(a.sum(log, 2315, a.product(7, row)), el))
	out.assembly[10] = a.sum(a.product(out.normalizationAttempts, a.sum(c, 1)),
		a.product(out.executionSlots, uint64(reflect.TypeFor[ReplayExecution]().Size())), min(p.Blocks, out.executionSlots))
	for coordinate := range out.normal {
		out.normal[coordinate] = a.sum(out.normalization[coordinate], out.execution[coordinate], out.assembly[coordinate])
	}
	out.envelope = out.normal
	if out.executionRoots != 0 {
		m := a.sum(h, 256)
		out.envelope[10] = a.sum(out.normal[10], a.product(m, m), 1)
	}
	if a.failed {
		return empty, false
	}
	return out, true
}
