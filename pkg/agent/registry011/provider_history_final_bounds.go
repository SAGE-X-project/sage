package registry011

import "strconv"

// The successful final reader has a disjoint pre-H0 row. It cannot spend the
// execution or failed-inspection row. Read work includes original historical
// semantic observations; the two C0/C1 samples are funded separately here.
// Numeric bounds remain DATA until the complete source derivation exists.
type providerHistoryFinalBounds struct {
	read      providerHistoryRecoveryBounds
	elapsedMS uint64
}

func (*providerHistoryFinalBounds) UnmarshalJSON([]byte) error { return errProviderIntegration }

func (b providerHistoryFinalBounds) valid(store providerIntegrationBounds, catalog, durationMS uint64) bool {
	return b.read.valid(store, catalog) && b.elapsedMS > 0 && b.elapsedMS <= 300000 && b.elapsedMS <= durationMS
}

// Final verification creates no durable files, network allowance or new
// clock. Both samples are part of the same original lifetime reservation.
// Keep the read envelope intact: the samples are additional actual events.
func (b providerHistoryFinalBounds) upper() (providerScanUpper, bool) {
	var empty providerScanUpper
	if b == (providerHistoryFinalBounds{}) {
		return empty, true // Existing incomplete candidate, never final authority.
	}
	if b.elapsedMS == 0 || b.elapsedMS > 300000 || b.read.maximumFiles == 0 || b.read.maximumBytes == 0 || b.read.work[2] == 0 || b.read.work[10] == 0 {
		return empty, false
	}
	if !providerHistoryReadWorkDomain(b.read.work) {
		return empty, false
	}
	units, ok := providerUpperSum(b.read.work[10], 2)
	if !ok {
		return empty, false
	}
	work := b.read.work
	work[10] = units
	return providerScanUpper{work: work, clocks: 2}, true
}

func (b providerHistoryFinalBounds) object() map[string]any {
	return map[string]any{
		"read": b.read.object(), "clock_reads": "2", "maximum_elapsed_ms": strconv.FormatUint(b.elapsedMS, 10),
	}
}
