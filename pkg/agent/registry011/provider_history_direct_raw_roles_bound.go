package registry011

// Conditional successful direct raw work, split at original execution roles.
// These rows exclude parser/rebuilder, I/O/name, ledger/cursor/context and
// recorder events owned by other components. Native CPU/backing/RAM and
// complete failure/recovery/final/feasibility/factory work remain separate.
type providerHistoryDirectRawRoleCost struct {
	priorMembers, advanceMembers uint64
	policyCalls, wrapperHashes   [6]uint64
	spoolVisits, resultBindings  [3]uint64
	writers                      [2]uint64
	rows                         [providerHistoryRawRows]providerHistoryRoleControlParts
	joined                       providerHistoryRoleControlParts
}

// Original scalar limits must later bind the selected source before H0.
// This does not consume or refund a hold, move a window or create authority.
// The unchanged original firstReading guard is checked before any quote.
// Full M contains P; the remaining-work composite still needs its extra P.
func providerHistoryCollectionDirectRawRoleUpper(manifest *providerHistoryCollectionManifest, limits providerHistoryLegacyControlLimits, prior uint64, advancing bool) (providerHistoryDirectRawRoleCost, bool) {
	var empty providerHistoryDirectRawRoleCost
	if !manifest.validSnapshot() || prior >= manifest.counts.groups || advancing && prior+1 >= manifest.counts.groups ||
		limits.objectBytes == 0 || limits.objectBytes > providerHistoryOriginalPackBytes ||
		limits.firstObjects != manifest.input.firstRead.objects || limits.firstRaw != manifest.input.firstRead.raw {
		return empty, false
	}
	first, firstOK := providerHistoryFirstPReadUpper(limits, manifest.input.capture)
	if !firstOK {
		return empty, false
	}
	for c, units := range first.work {
		if units > manifest.input.firstRead.work[c] {
			return empty, false
		}
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryDirectRawRoleCost
	var rows [providerHistoryRawRows][6]candidateWorkVector
	const h, policyBytes uint64 = candidateLedgerMetadataLimit, 64 << 10
	i, object := min(limits.metadata, h), limits.objectBytes
	l := a.sum(uint64(manifest.input.capture.WireBytes), 64<<10)
	body := uint64(manifest.input.capture.BodyBytes)
	policyWork := policyBytes*policyBytes + policyBytes + 2
	out.priorMembers = min(a.product(32, prior), manifest.counts.requests-1)
	put := func(kind, role, coordinate int, units uint64) {
		rows[kind][role][coordinate] = a.sum(rows[kind][role][coordinate], units)
	}
	policy := func(role int, calls uint64) {
		out.policyCalls[role] = a.sum(out.policyCalls[role], calls)
		put(providerHistoryRawPolicy, role, 2, a.product(policyBytes, calls))
		put(providerHistoryRawPolicy, role, 10, a.product(policyWork, calls))
	}
	prefix := func(role int, count, stages, members uint64) {
		put(providerHistoryRawHealthy, role, 10, a.product(count, a.sum(5*h+271, a.product(15, stages))))
		put(providerHistoryRawPrior, role, 10, a.product(count, a.product(stages, a.sum(a.product(3, a.sum(14*h, a.product(3, i), 277)), 133))))
		put(providerHistoryRawArchivedSpool, role, 10, a.product(count, a.product(members, a.sum(a.product(4, l), 272))))
		put(providerHistoryRawArchivedResult, role, 10, a.product(count, a.product(members, 133)))
		policy(role, a.product(count, stages))
	}
	// Initial-model raw controls already belong to the legacy component.
	// B9/S3 keep the current prior prefix. Own A includes its completed S.
	prefix(1, 9, prior, out.priorMembers)
	prefix(4, 3, prior, out.priorMembers)
	if advancing {
		out.advanceMembers = min(a.product(32, prior+1), manifest.counts.requests)
		prefix(5, 2, prior+1, out.advanceMembers)
		put(providerHistoryRawAdvance, 5, 10, 3*h+2)
		put(providerHistoryRawArchive, 5, 10, 4097)
	}
	put(providerHistoryRawAdmission, 1, 10, 7*(11*h+273))
	put(providerHistoryRawAdmission, 4, 10, 3*(11*h+273))
	put(providerHistoryRawConsumption, 1, 10, 4*(10*h+139))
	put(providerHistoryRawConsumption, 4, 10, 3*(10*h+139))
	put(providerHistoryRawSettlement, 4, 10, a.product(2, a.sum(5*h, a.product(2, object), 270)))
	put(providerHistoryRawOuterConsumption, 1, 10, 2*(3*h+2))
	put(providerHistoryRawAnchor, 1, 10, 7*h+4)
	put(providerHistoryRawAnchor, 2, 10, 2*(7*h+4))
	put(providerHistoryRawAnchor, 3, 10, 2*(7*h+4))
	// T/C/S reuse each before-pin byte copy once. Each writer has its own
	// seven metadata-input copies. Split actual T+C and S, not a whole
	// collection polynomial or a full three-writer row at each local cut.
	out.writers[0] = a.sum(object, 30*h, a.product(2, i), 27)
	out.writers[1] = a.sum(object, 15*h, i, 14)
	put(providerHistoryRawWriter, 1, 10, out.writers[0])
	put(providerHistoryRawWriter, 4, 10, out.writers[1])
	put(providerHistoryRawOutput, 4, 10, a.sum(a.product(2, object), 2, a.product(32, a.sum(a.product(4, body), 138))))
	put(providerHistoryRawCapture, 3, 10, a.sum(a.product(3, l), 6))
	// At index k<=31, P visits2k pairs and full M visits4k+2. S has3m.
	// Each prefix result loop still compares only retained k results. The
	// one new-result equality after capture is in Wrapper, not Result again.
	out.spoolVisits = [3]uint64{62, 126, 96}
	out.resultBindings = [3]uint64{62, 124, 96}
	for index, role := range [3]int{2, 3, 4} {
		put(providerHistoryRawCurrentSpool, role, 10, a.product(out.spoolVisits[index], a.sum(a.product(2, l), 270)))
		put(providerHistoryRawCurrentResult, role, 10, a.product(out.resultBindings[index], 133))
	}
	policy(2, 1) // Nonfirst preparation; conservative for its first member.
	policy(3, 2) // Entered P plus new original retention.
	policy(4, 3) // Coordinator plus two S readers.
	checkpoint := a.sum(12*h, a.product(3, i), 10)
	put(providerHistoryRawCheckpoint, 0, 10, a.sum(h, a.product(2, limits.metadata), 70))
	put(providerHistoryRawCheckpoint, 1, 10, a.sum(a.product(2, checkpoint), 77))
	put(providerHistoryRawCheckpoint, 4, 10, checkpoint)
	put(providerHistoryRawArchive, 3, 10, 67)
	put(providerHistoryRawArchive, 4, 10, 4097)
	// B: planning4, committed-T6, committed-C4, dispatch pair1, C pub1.
	// S: committed-C6, S pub1. A: final ledger pair1. P: member pair1
	// plus optional prior-pack pair1; full M adds pack+new-result pair2.
	// Existing typed-context model comparisons are different actual calls.
	out.wrapperHashes = [6]uint64{0, 16, 2, 4, 7, 0}
	if advancing {
		out.wrapperHashes[5] = 1
	}
	for role, calls := range out.wrapperHashes {
		put(providerHistoryRawWrapper, role, 10, a.product(133, calls))
	}
	put(providerHistoryRawMemberHead, 2, 10, 2*(3*h+2))
	put(providerHistoryRawMemberHead, 3, 10, 2*(3*h+2))
	parts := func(values [6]candidateWorkVector) providerHistoryRoleControlParts {
		var p [6]providerHistoryStageWorkCost
		for role, work := range values {
			p[role].work = work
		}
		return providerHistoryRoleControlParts{p[0], p[1], p[2], p[3], p[4], p[5]}
	}
	var sums [6]candidateWorkVector
	for kind, values := range rows {
		out.rows[kind] = parts(values)
		for role, work := range values {
			for coordinate, units := range work {
				sums[role][coordinate] = a.sum(sums[role][coordinate], units)
			}
		}
	}
	out.joined = parts(sums)
	if a.failed {
		return empty, false
	}
	return out, true
}
