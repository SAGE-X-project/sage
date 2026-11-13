package registry011

import (
	"math/big"
	"reflect"
	"strconv"

	"github.com/ethereum/go-ethereum/core/types"
)

type providerHistoryGasJourneyLimits struct {
	Windows, Blocks, PendingBlocks uint64
	EvidenceBytes, BodyBytes       uint64
	ValueBytes, TransactionBytes   uint64
	Transactions, NativeBlocks     uint64
}

type providerHistoryGasJourneyCost struct {
	normalization, strict, assembly    candidateWorkVector
	normal, envelope                   candidateWorkVector
	originalPool, strictPool           uint64
	canonicalOriginal, canonicalStrict uint64
	originalSlots, strictSlots         uint64
	rootRows, strictRows               uint64
}

// Conditional numeric DATA for the gas portion of ONE original serialized
// content journey, including its first aborting transition, without failed-S
// retry. W/B/K/E/T/C/H/V/N must be bound BEFORE use to that original range,
// caps, raw canonical transactions and native owner. A positive quote creates
// neither these bindings nor a source, observer, reservation or authority.
//
// A cleared batch has matching original semantic byte/transaction charges.
// Only one outstanding acquired batch of at most K blocks can remain. BOTH
// strict passes therefore share (2W+K)E and (2W+K)T; acquisition native rows
// share (W+K)E/T. Original root trace fields discarded during normalization
// cannot borrow the later normalized trace pool: keep their complete NC.
// Direct classification can precede byte refusal, so its raw pool is bounded
// by min(NC, WE+KC), retaining C rather than E for the pending batch.
//
// Raw-array allocated/copied slots are NOT native transaction rows. Even an
// empty whitespace-padded array allocates min(T,(n-1)/2) headers. Fund those
// slots from original raw pools separately; only rows entered AFTER the exact
// array/native-count check may use the matched canonical transaction pools.
//
// This replaces overlapping gas normalization, classification, assembly and
// both strict-check contributions only. Copies in retained/rebuilt blocks,
// canonical decoding, execution/effects, other readers/roles, full admitted
// stage sums, native CPU/backing/RAM, resources/factory and feasibility remain
// separate. E adds one current native encoding preflight for this component;
// it does not refund irrevocable held uppers or authorize one global pad.
func providerHistoryGasJourneyUpper(p providerHistoryGasJourneyLimits, profile string) (providerHistoryGasJourneyCost, bool) {
	var empty providerHistoryGasJourneyCost
	if p.Blocks == 0 || p.Blocks > 1<<20 || p.Windows == 0 || p.Windows > p.Blocks ||
		p.PendingBlocks == 0 || p.PendingBlocks > 10 || p.EvidenceBytes == 0 || p.EvidenceBytes > 64<<20 ||
		p.BodyBytes < 4 || p.BodyBytes > 8<<20 || p.TransactionBytes == 0 || p.TransactionBytes > 64<<20 ||
		p.ValueBytes == 0 || p.ValueBytes > min(p.EvidenceBytes, p.BodyBytes, p.TransactionBytes) ||
		p.Transactions == 0 || p.Transactions > 16384 || p.NativeBlocks > p.Blocks {
		return empty, false
	}
	root, profileBytes := false, uint64(max(len(SepoliaAmsterdamGasProfile), len(SepoliaAmsterdamNethermindGasProfile)))
	switch profile {
	case SepoliaAmsterdamGasProfile, SepoliaAmsterdamNethermindGasProfile:
	case SepoliaAmsterdamRootRegularGasProfile, SepoliaAmsterdamRootExecutionGasProfile:
		root = true
		profileBytes = uint64(max(len(SepoliaAmsterdamRootRegularGasProfile), len(SepoliaAmsterdamRootExecutionGasProfile)))
	default:
		return empty, false
	}
	var a providerHistoryReadArithmetic
	windows, k, evidence, c := p.Windows, p.PendingBlocks, p.EvidenceBytes, p.BodyBytes
	n, txs, v, h := p.NativeBlocks, p.Transactions, p.ValueBytes, p.TransactionBytes
	w := uint64(strconv.IntSize / 8)
	sidecar := a.sum(uint64(reflect.TypeFor[types.Transaction]().Size()), uint64(reflect.TypeFor[types.BlobTx]().Size()), 2)
	value := a.sum(a.product(a.sum(h, w-1)/w, w), uint64(reflect.TypeFor[big.Int]().Size()))
	one := a.sum(windows, k)
	two := a.sum(a.product(2, windows), k)
	nativeRows := a.product(n, txs)
	strictCalls := a.product(2, n)
	rawSlots := min(txs, (c-1)/2)
	var out providerHistoryGasJourneyCost
	out.originalPool = a.product(n, c)
	if !root {
		out.originalPool = min(out.originalPool, a.sum(a.product(windows, evidence), a.product(k, c)))
	}
	out.strictPool = min(a.product(strictCalls, c), a.product(two, evidence))
	// Complete canonical transactions are disjoint within the already
	// guarded original whole-block encoding, at most min(C,E) per block.
	canonicalBody := a.product(n, min(c, evidence))
	out.canonicalOriginal = min(a.product(one, evidence), canonicalBody, a.product(nativeRows, v))
	out.canonicalStrict = min(a.product(two, evidence), a.product(2, canonicalBody), a.product(2, a.product(nativeRows, v)))
	out.originalSlots = min(a.product(n, rawSlots), out.originalPool/2)
	out.strictSlots = min(a.product(strictCalls, rawSlots), out.strictPool/2)
	out.strictRows = min(a.product(2, nativeRows), a.product(two, txs), out.strictSlots, out.canonicalStrict)
	strictFixed := a.sum(sidecar, value, 21078, a.product(256, a.sum(v, 258)))
	const header = uint64(65536)
	out.strict[2] = a.sum(a.product(6, out.strictPool), a.product(97, out.strictRows))
	out.strict[10] = a.sum(a.product(strictCalls, a.sum(a.product(header, header), header, 4497)),
		a.product(a.sum(a.product(6, c), 16), out.strictPool),
		a.product(a.sum(a.product(6, w), 1), out.strictSlots),
		a.product(strictFixed, out.strictRows), a.product(a.sum(a.product(2, v), 260), out.canonicalStrict))
	if root {
		out.rootRows = min(nativeRows, a.product(one, txs), out.originalSlots, out.canonicalOriginal)
		rootFixed := a.sum(sidecar, 37691, a.product(6, w), a.product(256, a.sum(v, 258)))
		out.normalization[2] = a.sum(a.product(8, out.originalPool), a.product(161, out.rootRows))
		out.normalization[10] = a.sum(a.product(a.sum(a.product(9, c), 29), out.originalPool), a.product(5, n),
			a.product(a.sum(a.product(6, w), 1), out.originalSlots), a.product(rootFixed, out.rootRows),
			a.product(a.sum(a.product(2, v), 260), out.canonicalOriginal))
	} else {
		// Complete first-row classifier: raw array plus two closed objects.
		// A/S=2/12 and5/58, including their first unknown-key refusal, give
		// 3n²+9n+740+(6w+1)s. No successful-only663 constant is reused.
		out.normalization[2] = a.product(3, out.originalPool)
		out.normalization[10] = a.sum(a.product(a.sum(a.product(3, c), 9), out.originalPool), a.product(740, n),
			a.product(a.sum(a.product(6, w), 1), out.originalSlots))
	}
	// Initial raw copy may precede normalization/strict byte refusal.
	// Keep the longest possible returned profile in the selected family for
	// a mismatching profile comparison as well as a successful comparison.
	gas := uint64(reflect.TypeFor[ReplayGasEvidence]().Size())
	out.assembly[10] = a.sum(out.originalPool, a.product(a.sum(a.product(2, profileBytes), gas, 3), n))
	for coordinate := range out.normal {
		out.normal[coordinate] = a.sum(out.normalization[coordinate], out.strict[coordinate], out.assembly[coordinate])
	}
	out.envelope = out.normal
	if n != 0 {
		m := a.sum(h, 256)
		out.envelope[10] = a.sum(out.normal[10], a.product(m, m), 1)
	}
	if a.failed {
		return empty, false
	}
	return out, true
}
