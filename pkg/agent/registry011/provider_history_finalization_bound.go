package registry011

// The closing subgraph AFTER both fresh original models reach the terminal
// collection state. These are source-cost DATA, not a final owner, completed
// reconstruction, physical resource proof or permission to sample a clock.
// Reconstruction/model work and the enclosing caller's root cleanup remain
// separate. Clock-event units are added by finalBounds.upper, exactly once.
type providerHistoryFinalizationCost struct {
	limits                     providerHistoryCollectionCursorLimits
	cursor                     providerHistoryAppliedCursorCost
	preparation                candidateWorkVector
	normal, incoming, envelope candidateWorkVector
	clockIDBytes, clocks       uint64
}

// Terminal cardinalities come from closeContentObserved and semantic.atEnd:
// no pending block/effect/trace, stop record, working window or history queue
// survives. The confirmed lifetime indexes and complete original spine DO
// survive and keep their selected maxima. This does not assert that a caller
// has reached that state. The complete compiler must bind this manifest and
// clockIDBytes to its SAME selected backend before admitting any work.
func providerHistoryFinalizationUpper(manifest *providerHistoryCollectionManifest, clockIDBytes uint64) (providerHistoryFinalizationCost, bool) {
	var empty providerHistoryFinalizationCost
	if !manifest.validSnapshot() || clockIDBytes == 0 {
		return empty, false
	}
	in := manifest.input
	limits := providerHistoryCollectionCursorLimits{
		Spine: in.blocks, Transactions: uint64(in.journal.Replay.MaxTransactions),
		EvidenceBytes: uint64(in.journal.Replay.MaxEvidenceBytes), BodyBytes: uint64(in.capture.BodyBytes),
		Semantic: providerHistorySemanticCommitmentLimits{
			ConfirmedClaims: uint64(in.journal.MaxClaims), ConfirmedCreated: uint64(in.journal.MaxIdentities),
		},
	}
	// Neither stream is assumed empty. The applied serializer retains every
	// original metadata preflight and deferred serializer/index sibling.
	cursor, cursorOK := providerHistoryAppliedCursorUpper(limits, 0, providerHistoryAppliedStateKinds{})
	preparation, preparationOK := providerHistoryFinalDataPreparationUpper(clockIDBytes)
	if !cursorOK || !preparationOK {
		return empty, false
	}
	out := providerHistoryFinalizationCost{limits: limits, cursor: cursor, preparation: preparation, clockIDBytes: clockIDBytes, clocks: 2}
	var a providerHistoryReadArithmetic
	for coordinate := range out.normal {
		// Two independent commitments precede their equality test. copyObserved
		// then commits a THIRD time before cloning the confirmed stream/spine.
		out.normal[coordinate] = a.sum(a.product(2, cursor.normalCommitment[coordinate]), cursor.normalCopy[coordinate], preparation[coordinate])
		rootExtra := a.difference(cursor.envelopeCommitment[coordinate], cursor.normalCommitment[coordinate])
		copyExtra := a.difference(cursor.envelopeCopy[coordinate], cursor.normalCopy[coordinate])
		// BOTH direct commitments run before either empty result is tested.
		// A late nonpoisoning serializer refusal in the left model does NOT
		// remove the right model's closure. Their paired extra work must SUM.
		// Copy is entered only after both commitments succeed; its alternative
		// extra is MAX, not another independently outstanding allowance.
		out.incoming[coordinate] = max(a.product(2, rootExtra), copyExtra)
	}
	// Two copy(1) calls (begin and final result), the 66+66 hash comparison,
	// and final identity4/Stat1/metadata-compare1 for EACH original root.
	// C0/C1's two logical units and samples remain in finalBounds.upper.
	out.normal[10] = a.sum(out.normal[10], 4, 133, 12)
	for coordinate := range out.envelope {
		out.envelope[coordinate] = a.sum(out.normal[coordinate], out.incoming[coordinate])
	}
	if a.failed || !providerHistoryReadWorkDomain(out.envelope) {
		return empty, false
	}
	return out, true
}
