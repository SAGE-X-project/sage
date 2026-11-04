package registry011

// Pure costs for the original open and scan preparation/inspection. These
// functions accept bounded scalar DATA, not a profile with an execution
// witness. Existing runtime wrappers must still validate the whole profile.
// Compiler validation/copies and native resources are separate obligations.
func providerScanCostingParametersValid(metadata, overhead uint64) bool {
	return metadata > 0 && metadata <= 1<<20 && overhead > 0 && overhead <= 1<<20
}

func providerScanOpenCostUpper(metadata, overhead uint64) (providerScanUpper, bool) {
	if !providerScanCostingParametersValid(metadata, overhead) {
		return providerScanUpper{}, false
	}
	n := max(metadata, uint64(64<<10))
	var a providerHistoryReadArithmetic
	units := a.product(a.sum(a.product(n, n), n, 16), 128)
	decoded, raw := a.product(n, 128), a.product(n, 9)
	physical := a.sum(raw, a.product(overhead, 13))
	if a.failed {
		return providerScanUpper{}, false
	}
	return providerScanUpper{work: candidateWorkVector{2: decoded, 9: raw, 10: units}, retained: physical, clocks: 16}, true
}

func providerScanPreparationCostUpper(metadata, overhead, ordinal, attempts uint64) (providerScanUpper, bool) {
	if !providerScanCostingParametersValid(metadata, overhead) || attempts == 0 || attempts > 64 || ordinal == 0 || ordinal > attempts {
		return providerScanUpper{}, false
	}
	var a providerHistoryReadArithmetic
	count := a.sum(a.product(ordinal, 128), 256)
	units := a.product(a.sum(a.product(metadata, metadata), metadata, 8), count)
	decoded, written := a.product(metadata, count), a.product(metadata, 7)
	physical := a.sum(written, a.product(overhead, 7))
	if a.failed {
		return providerScanUpper{}, false
	}
	return providerScanUpper{work: candidateWorkVector{2: decoded, 9: written, 10: units}, retained: physical, clocks: 4}, true
}

// Only domains consumed by the scan arithmetic are checked here. The original
// constructor's complete structural validation remains required before a
// compiler can bind these DATA to an immutable selected profile.
func providerScanCostingDomainValid(b providerScanBounds) bool {
	if b.maximumRecords == 0 || b.maximumRecords > (2<<20)+65 || b.maximumChunks == 0 || b.maximumChunks > b.maximumRecords ||
		b.maximumClockReads == 0 || b.maximumClockReads > b.maximumPreIClockReads || b.maximumPreIClockReads > 1<<20 ||
		b.maximumPreIRetainedBytes == 0 || b.maximumPreIRetainedBytes > 1<<40 ||
		b.chunkWork[0] != 0 || b.chunkWork[1] != 0 || b.chunkWork[2] == 0 || b.chunkWork[10] == 0 ||
		b.maximumPreIWork[0] != 0 || b.maximumPreIWork[1] != 0 || b.maximumPreIWork[9] == 0 || b.maximumPreIWork[10] == 0 {
		return false
	}
	for coordinate, amount := range b.chunkWork {
		if amount > b.maximumPreIWork[coordinate] {
			return false
		}
	}
	return true
}

func providerScanInspectionCostUpper(b providerScanBounds, metadata, overhead uint64, first bool, opening providerScanUpper) (providerScanUpper, bool) {
	if !providerScanCostingParametersValid(metadata, overhead) || !providerScanCostingDomainValid(b) {
		return providerScanUpper{}, false
	}
	var a providerHistoryReadArithmetic
	units := a.product(a.sum(a.product(metadata, metadata), metadata, 16), 32)
	written := a.product(metadata, 4)
	physical := a.sum(written, a.product(overhead, 4))
	decoded := written
	if first {
		decoded = a.product(metadata, 6) // Two distinct closed receipt decoders.
	}
	var out providerScanUpper
	for coordinate, amount := range b.chunkWork {
		extra := uint64(0)
		switch coordinate {
		case 2:
			extra = decoded
		case 9:
			extra = written
		case 10:
			extra = units
		}
		out.work[coordinate] = a.product(a.sum(amount, extra), b.maximumChunks)
	}
	out.retained, out.clocks = a.product(physical, b.maximumChunks), b.maximumClockReads
	if a.failed || !b.fits(opening, out) {
		return providerScanUpper{}, false
	}
	return out, true
}

// Inspection and the disjoint selected first-E1 hold belong to the same B.
// The old V3 path passes an empty first hold; no old execution profile is
// temporarily changed to select a smaller first-admission decoder graph.
func providerScanBodyCostUpper(b providerScanBounds, opening, inspection, first providerScanUpper) (providerScanUpper, bool) {
	if !providerScanCostingDomainValid(b) {
		return providerScanUpper{}, false
	}
	out, ok := providerScanUpperSum(inspection, first)
	if !ok || !b.fits(opening, out) {
		return providerScanUpper{}, false
	}
	return out, true
}
