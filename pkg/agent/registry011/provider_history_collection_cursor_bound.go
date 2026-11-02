package registry011

import (
	"reflect"
	"strconv"
)

// Prospective maxima from the original sealed input, not reported actuals or
// an authority to construct arbitrary cursors. Effects are local to one pending
// content batch; they are distinct from the whole-program effect maximum.
type providerHistoryCollectionCursorLimits struct {
	Spine, Effects, PendingBlocks, Transactions, EvidenceBytes, BodyBytes uint64
	Semantic                                                              providerHistorySemanticCommitmentLimits
}

type providerHistoryCollectionCursorCost struct{ commitment, copy candidateWorkVector }

// One original cursor commitment and one complete copy (which first commits
// that root). Explicit copy metrics do not certify backing arrays/native RAM.
// Factory input binding, repeated replay/root multiplicities and qualification
// remain separate. Every failed arithmetic/component returns no partial cost.
func providerHistoryCollectionCursorUpper(l providerHistoryCollectionCursorLimits) (providerHistoryCollectionCursorCost, bool) {
	var empty providerHistoryCollectionCursorCost
	if l.Spine == 0 || l.Spine > 1<<20 || l.Effects > 10*16384 || l.PendingBlocks > 10 ||
		l.Transactions == 0 || l.Transactions > 16384 || l.EvidenceBytes == 0 || l.EvidenceBytes > 64<<20 || l.BodyBytes == 0 || l.BodyBytes > 8<<20 {
		return empty, false
	}
	semantic, semanticOK := providerHistorySemanticCommitmentUpper(l.Semantic)
	snapshotLimits := providerHistorySemanticSnapshotLimits{
		Confirmed: providerHistorySemanticCloneLimits{Claims: l.Semantic.ConfirmedClaims, Created: l.Semantic.ConfirmedCreated, Keys: 128, Services: 16, Grants: 128},
		Window:    l.Semantic.Window, Queue: l.Semantic.Queue, Pending: l.Semantic.Pending,
	}
	if l.Semantic.Window {
		snapshotLimits.Working = providerHistorySemanticCloneLimits{Claims: l.Semantic.WorkingClaims, Created: l.Semantic.WorkingCreated, Keys: 128, Services: 16, Grants: 128}
	}
	snapshot, snapshotOK := providerHistorySemanticSnapshotUpper(snapshotLimits)
	if !semanticOK || !snapshotOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	metadata := uint64(candidateLedgerMetadataLimit)
	value := a.sum(33, a.product(metadata, metadata), 1, metadata, uint64(len(providerHistoryCollectionStateDomain)), 2)
	spineMaximum := providerHistoryCollectionSpineBytes
	spineValue := a.sum(33, a.product(spineMaximum, spineMaximum), 1, spineMaximum, uint64(len(providerHistoryCollectionStateDomain)), 2)
	const journal uint64 = 65536
	manifest := a.sum(a.product(journal, journal), 1, journal, 1,
		a.product(metadata, metadata), 1, metadata, uint64(len("sage-provider-history-collection-manifest-v1")), 2, 133)
	bytesRoot := func(n uint64) uint64 {
		return a.sum(a.product(2, n), uint64(len("collection-pending-bytes")), 3)
	}
	headerBytes, evidenceBytes, bodyBytes := bytesRoot(65536), bytesRoot(l.EvidenceBytes), bytesRoot(l.BodyBytes)
	profile := uint64(max(len(SepoliaAmsterdamGasProfile), len(SepoliaAmsterdamNethermindGasProfile), len(SepoliaAmsterdamRootRegularGasProfile), len(SepoliaAmsterdamRootExecutionGasProfile)))
	spine := a.sum(a.product(l.Spine, spineValue), a.product(a.difference(l.Spine, 1), 133))
	effects := a.product(l.Effects, a.sum(266, value))
	// Reserve every pending block as unapplied/post-fork and every effect
	// trace as received. Applied/unreceived/empty branches consume less.
	block := a.sum(133, headerBytes, a.product(3, value),
		a.product(a.product(2, l.Transactions), a.sum(evidenceBytes, value)),
		a.product(2, profile), 1, evidenceBytes)
	pending := a.sum(value, a.product(l.PendingBlocks, block), a.product(l.Effects, a.sum(bodyBytes, value)))
	commitment := semantic
	commitment[10] = a.sum(commitment[10], manifest, spine, effects, pending, headerBytes, value)
	// The original first copy event counts numeric1+S+E bytes, not header
	// or effect descriptor sizes. Pending arrays use their actual copied
	// type sizes and the original node slice's two machine words per item.
	copyUnits := a.sum(l.Spine, l.Effects, 4,
		a.product(l.PendingBlocks, a.sum(uint64(reflect.TypeFor[providerHistoryCollectionBlock]().Size()),
			a.product(a.product(4, l.Transactions), uint64(strconv.IntSize/8)), 2)),
		a.product(l.Effects, uint64(reflect.TypeFor[string]().Size())))
	copied := commitment
	for coordinate, units := range snapshot {
		copied[coordinate] = a.sum(copied[coordinate], units)
	}
	copied[10] = a.sum(copied[10], copyUnits)
	if a.failed {
		return empty, false
	}
	return providerHistoryCollectionCursorCost{commitment, copied}, true
}

// Conditional parts of ONE original cursor root/copy, not a selected cursor,
// original reader, stage/future allowance or factory certificate. The selected
// history/evidence limits and state kinds must bind the actual original owned
// graph. General nonempty states and every old standalone result stay intact.
type providerHistoryAppliedCursorCost struct {
	normalCommitment, envelopeCommitment candidateWorkVector
	normalCopy, envelopeCopy             candidateWorkVector
}

func providerHistoryAppliedCursorUpper(l providerHistoryCollectionCursorLimits, historyBytes uint64, kinds providerHistoryAppliedStateKinds) (providerHistoryAppliedCursorCost, bool) {
	var empty providerHistoryAppliedCursorCost
	old, oldOK := providerHistoryCollectionCursorUpper(l)
	oldSemantic, oldSemanticOK := providerHistorySemanticCommitmentUpper(l.Semantic)
	semantic, semanticOK := providerHistoryAppliedCommitmentKindsUpper(l.Semantic, historyBytes, kinds)
	snapshotLimits := providerHistorySemanticSnapshotLimits{
		Confirmed: providerHistorySemanticCloneLimits{l.Semantic.ConfirmedClaims, l.Semantic.ConfirmedCreated, 128, 16, 128},
		Window:    l.Semantic.Window, Queue: l.Semantic.Queue, Pending: l.Semantic.Pending,
	}
	if l.Semantic.Window {
		snapshotLimits.Working = providerHistorySemanticCloneLimits{l.Semantic.WorkingClaims, l.Semantic.WorkingCreated, 128, 16, 128}
	}
	snapshot, snapshotOK := providerHistoryAppliedSnapshotUpper(snapshotLimits, historyBytes, kinds)
	if !oldOK || !oldSemanticOK || !semanticOK || !snapshotOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	value := func(kind string, raw uint64) uint64 {
		return a.sum(uint64(len(kind)), 1, a.product(raw, raw), 1, raw, uint64(len(providerHistoryCollectionStateDomain)), 2)
	}
	// Success of the LATER manifest8192 guard bounds its embedded journal
	// string to8192 too. Earlier journal65536 and all metadata preflights are
	// still funded by the original complete envelope, including late refusal.
	const metadata uint64 = 8192
	manifest := a.sum(a.product(metadata, metadata), 1, metadata, 1,
		a.product(metadata, metadata), 1, metadata, uint64(len("sage-provider-history-collection-manifest-v1")), 2, 133)
	spine := a.sum(a.product(l.Spine, value("spine", 402)), a.product(a.difference(l.Spine, 1), 133))
	effects := a.product(l.Effects, a.sum(266, value("effects", 393)))
	profile := uint64(max(len(SepoliaAmsterdamGasProfile), len(SepoliaAmsterdamNethermindGasProfile), len(SepoliaAmsterdamRootRegularGasProfile), len(SepoliaAmsterdamRootExecutionGasProfile)))
	bytesRoot := func(n uint64) uint64 { return a.sum(a.product(2, n), uint64(len("collection-pending-bytes")), 3) }
	// A stored unapplied block has header+ALL tx+ALL receipts<=selected E,
	// checked by replayDecodeContent before retention. Gas has a SEPARATE
	// selected-E budget, not another per-transaction allowance. Its original
	// preflights and all deferred list siblings remain in the old envelope.
	byteParts := a.sum(a.product(4, l.EvidenceBytes), a.product(a.sum(a.product(2, l.Transactions), 2), uint64(len("collection-pending-bytes"))+3))
	block := a.sum(133, byteParts, a.product(2, value("pending-list", 90)),
		a.product(a.product(2, l.Transactions), value("pending-list-item", 256)), 2*profile, 1, value("pending-block", 829))
	// pending-batch180 also dominates the actual pending-empty77 branch.
	// Effect trace results are separate from gas/base bytes and keep their
	// own per-field capture.BodyBytes bound; no unproved aggregate is used.
	pending := a.sum(value("pending-batch", 180), a.product(l.PendingBlocks, block), a.product(l.Effects, a.sum(bytesRoot(l.BodyBytes), value("pending-trace", 284))))
	out := providerHistoryAppliedCursorCost{normalCommitment: semantic.normal}
	out.normalCommitment[10] = a.sum(out.normalCommitment[10], manifest, spine, effects, pending, bytesRoot(65536), value("cursor", 3062))
	for coordinate := range out.envelopeCommitment {
		// Preserve ALL remaining original caller work, including pending and
		// optional stop-record roots AFTER semantic failure. Only substitute
		// the independently complete eligible semantic child envelope.
		out.envelopeCommitment[coordinate] = a.sum(a.difference(old.commitment[coordinate], oldSemantic[coordinate]), semantic.envelope[coordinate])
	}
	copyUnits := a.sum(l.Spine, l.Effects, 4,
		a.product(l.PendingBlocks, a.sum(uint64(reflect.TypeFor[providerHistoryCollectionBlock]().Size()),
			a.product(a.product(4, l.Transactions), uint64(strconv.IntSize/8)), 2)),
		a.product(l.Effects, uint64(reflect.TypeFor[string]().Size())))
	for coordinate := range out.normalCopy {
		out.normalCopy[coordinate] = a.sum(out.normalCommitment[coordinate], snapshot[coordinate])
		out.envelopeCopy[coordinate] = a.sum(out.envelopeCommitment[coordinate], snapshot[coordinate])
	}
	out.normalCopy[10] = a.sum(out.normalCopy[10], copyUnits)
	out.envelopeCopy[10] = a.sum(out.envelopeCopy[10], copyUnits)
	if a.failed {
		return empty, false
	}
	return out, true
}
