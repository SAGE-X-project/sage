package registry011

// Scalar quote arithmetic only. These rows never create measured work,
// consume a hold, read an original, or grant a protected execution permit.
type providerRPCSizeRow struct{ bytes, quadratic, linear uint64 }

func providerRPCAddSizeRows(work *candidateWorkVector, rows []providerRPCSizeRow) bool {
	if work == nil || len(rows) > 16 {
		return false
	}
	for _, row := range rows {
		q, a := providerUpperProduct(row.bytes, row.bytes)
		q, b := providerUpperSum(q, 1)
		q, c := providerUpperProduct(q, row.quadratic)
		l, e := providerUpperSum(row.bytes, row.bytes, 101)
		l, f := providerUpperProduct(l, row.linear)
		decoded, h := providerUpperProduct(row.bytes, row.quadratic)
		total, j := providerUpperSum(q, l)
		value, k := providerUpperSum(work[2], decoded)
		units, m := providerUpperSum(work[10], total)
		if !a || !b || !c || !e || !f || !h || !j || !k || !m {
			return false
		}
		work[2], work[10] = value, units
	}
	return true
}

// One retained method-class reconstruction: body unmarshal1, request's
// params/body/head encodes3, and at most one method-params decode1.
// At most16KiB normalization nodes cost32768 units. The request builder
// visits at most513 nodes (1026) and <=8192 string bytes+513 events(8705).
const providerRPCClassRecursionUpper uint64 = 32768 + 1026 + 8705

func providerRPCClassExtraUpper(n uint64) (candidateWorkVector, bool) {
	var work candidateWorkVector
	if !providerRPCAddSizeRows(&work, []providerRPCSizeRow{{n, 5, 11}}) {
		return candidateWorkVector{}, false
	}
	units, ok := providerUpperSum(work[10], providerRPCClassRecursionUpper)
	work[10] = units
	return work, ok
}
