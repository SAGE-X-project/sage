package registry011

import "strconv"

// This separate read-only row is held once before H0. It cannot be spent by
// group execution and cannot be recreated after that execution has failed.
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
	for i, n := range b.work {
		if i != 2 && i != 10 && n != 0 {
			return false
		}
	}
	return true
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
