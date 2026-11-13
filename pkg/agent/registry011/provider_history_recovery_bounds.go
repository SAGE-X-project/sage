package registry011

import "strconv"

// This separate read-only row is held once before H0. It cannot be spent by
// group execution and cannot be recreated after that execution has failed.
// Original reconstruction also interprets retained blocks/transactions/frames/
// logs and verifies historical key proofs. These are read work, not network or
// write permission; proof attempts must fit this separately pre-held allowance.
// Positive numbers alone are not a sufficient production cost derivation.
type providerHistoryRecoveryBounds struct {
	work                       candidateWorkVector
	maximumFiles, maximumBytes uint64
}

func (*providerHistoryRecoveryBounds) UnmarshalJSON([]byte) error { return errProviderIntegration }

func (b providerHistoryRecoveryBounds) valid(store providerIntegrationBounds, catalog uint64) bool {
	files, a := providerUpperProduct(catalog, 2)
	bytes, c := providerUpperProduct(store.maximumStoredBytes, 2)
	if !a || !c || b.maximumFiles == 0 || b.maximumFiles > files || b.maximumBytes == 0 || b.maximumBytes > bytes || b.work[2] == 0 || b.work[10] == 0 {
		return false
	}
	return providerHistoryReadWorkDomain(b.work)
}

// The original byte/codec observations and historical semantic counts share
// this read-only vector. Historical proof verification is a counted local read
// operation. Requests, wire capture, segment acquisition and disk writes remain
// excluded. Primitive kind checks remain separate:
// positive logical units cannot authorize TLS, sync, rename or clock events.
func providerHistoryReadWorkDomain(work candidateWorkVector) bool {
	return work[0] == 0 && work[1] == 0 && work[8] == 0 && work[9] == 0
}

// Structural DATA for TWO fresh original reconstruction passes. Each group's
// S transition replays a prefix of ONE monotone collection journey per pass;
// completed/tail payload readers parse transport originals, not another content
// journey. The existing mechanical child includes acquisition + semantic
// decoding, JSON's extra projection, trace normalization/execution, and the
// first aborting transition without a failed-S retry.
//
// Read-only owners currently select the original wide64MiB native input guard,
// not the collection execution owner's narrower capture-derived guard. Keep
// that exact premise when using the child. Only coordinates3..6 are assigned
// here: byte/codec/logical/read/clock/storage/native/RAM costs remain separate.
// This is not a source derivation, new live allowance or production factory.
func providerHistoryCollectionReadSemanticUpper(manifest *providerHistoryCollectionManifest) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	if !manifest.validSnapshot() {
		return empty, false
	}
	in := manifest.input
	evidence, body := uint64(in.journal.Replay.MaxEvidenceBytes), uint64(in.capture.BodyBytes)
	mechanical, ok := providerHistoryContentJourneyUpper(providerHistoryContentJourneyLimits{
		Windows: manifest.counts.contentWindows, Blocks: in.blocks, PendingBlocks: in.contentBlocks,
		EvidenceBytes: evidence, BodyBytes: body, ValueBytes: min(evidence, body), TransactionBytes: 64 << 20,
		Transactions: uint64(in.journal.Replay.MaxTransactions), Frames: uint64(in.journal.Replay.MaxFrames), Logs: uint64(in.journal.Replay.MaxLogs),
		Effects: in.effects, NativeBlocks: in.native,
	}, in.encoding, in.gas)
	if !ok {
		return empty, false
	}
	var out candidateWorkVector
	for coordinate := 3; coordinate <= 6; coordinate++ {
		units, valid := providerUpperProduct(2, mechanical.envelope[coordinate])
		if !valid {
			return empty, false
		}
		out[coordinate] = units
	}
	return out, true
}

func (b providerHistoryRecoveryBounds) object() map[string]any {
	return map[string]any{"work": candidateLedgerWorkObject(b.work), "maximum_file_inputs": strconv.FormatUint(b.maximumFiles, 10), "maximum_input_bytes": strconv.FormatUint(b.maximumBytes, 10)}
}

func providerHistoryRecoveryEventAllowed(kind providerWorkEvent) bool {
	switch kind {
	case providerReadEvent, providerCodecEvent, providerHashEvent, providerCopyEvent, providerEncodeEvent, providerCompareEvent:
		return true
	}
	return false
}
