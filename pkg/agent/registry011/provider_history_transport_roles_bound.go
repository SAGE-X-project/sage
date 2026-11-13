package registry011

// Prospective BYTE components from the same sealed collection and explicitly
// selected method/TLS/object limits. A zero grammar class has no member and
// cannot supply an unrelated failure allowance. These are numeric DATA, not
// a source witness, admitted invocation or replacement for native qualification.
type providerHistoryTransportRoleCost struct {
	grammar       providerHistoryCollectionCaptureCost
	requestUpper  [8]uint64
	capture       [8]candidateWorkVector
	failure       [8]providerHistoryFailedSettlementExtraCost
	member        candidateWorkVector
	failed        providerHistoryStageWorkCost
	failedStorage providerHistoryStorageCharge
	joined        providerHistoryRoleControlParts
}

// ONE attempted member and ONE terminal failed-S extra, not whole-program
// capture/G or a whole-program quote repeated at each local role. The complete
// full-M and closure rows still need incoming/refusal/recording/typed-context,
// native CPU/backing/peak/lifetime RAM, recovery/final and feasible windows.
func providerHistoryCollectionTransportRoleUpper(manifest *providerHistoryCollectionManifest, methods providerMethodBounds, certificates providerTLSBounds, maximum, metadata, overhead uint64) (providerHistoryTransportRoleCost, bool) {
	var empty providerHistoryTransportRoleCost
	if !manifest.validSnapshot() {
		return empty, false
	}
	grammar, ok := providerHistoryCollectionCaptureUpper(manifest.input, methods, certificates)
	if !ok {
		return empty, false
	}
	out := providerHistoryTransportRoleCost{grammar: grammar}
	limits := [8]providerMethodLimit{methods.scalar, methods.header, methods.code, methods.state, methods.block, methods.receipts, methods.gas, methods.execution}
	for class, count := range grammar.classes {
		if count == 0 {
			continue
		}
		capture := manifest.input.capture
		capture.BodyBytes, capture.WireBytes = limits[class].body, limits[class].wire
		// q(r)-q(1)=(r-1)*(r+4-2W)<=0 for 1<=r<W. The actual
		// request is not changed to one byte. The failed-body R terms instead
		// increase with r, so use its separate original request-length maximum.
		one, captureOK := providerHistoryCaptureUpper(1, capture, certificates)
		r := min(uint64(16<<10), uint64(capture.WireBytes)-1)
		failed, failureOK := providerHistoryFailedSettlementExtraUpper(r, uint64(capture.WireBytes), maximum, metadata, overhead, certificates)
		if !captureOK || !failureOK {
			return empty, false
		}
		out.requestUpper[class], out.capture[class], out.failure[class] = r, one, failed
		for c := range one {
			out.member[c] = max(out.member[c], one[c])
			out.failed.work[c] = max(out.failed.work[c], failed.extra.work[c])
		}
		out.failed.clocks = max(out.failed.clocks, failed.extra.clocks)
		// The distinct storage dimensions may attain their maxima in
		// different classes; none is summed over all hypothetical failures.
		p, f := &out.failedStorage, failed.storage
		p.physical, p.raw = max(p.physical, f.physical), max(p.raw, f.raw)
		p.payload, p.metadata = max(p.payload, f.payload), max(p.metadata, f.metadata)
		p.objects, p.files = max(p.objects, f.objects), max(p.files, f.files)
	}
	// The owner consumes this one irrevocable request before dial. The
	// BYTE primitive does not consume it again. Original owned clock events
	// remain exclusively assigned to the existing full-M/closure components.
	out.member[0] = 1
	out.joined.member.work, out.joined.closure = out.member, out.failed
	return out, true
}

// The six previously assigned components and this local transport/failure
// addition retain their provenance. The successful S allocation remains in
// full even if failure omitted an ordinary spool/result; there is no refund.
// firstInitialization is still the separate existing constructor bridge.
type providerHistoryTransportSourceCost struct {
	source    providerHistoryRoleSourceCost
	transport providerHistoryTransportRoleCost
	joined    providerHistoryRoleControlParts
	storage   [6]providerHistoryStorageCharge
}

func providerHistoryCollectionTransportSourceUpper(manifest *providerHistoryCollectionManifest, methods providerMethodBounds, certificates providerTLSBounds, maximum uint64, limits providerHistoryRoleSourceLimits, prior, members uint64, advancing bool) (providerHistoryTransportSourceCost, bool) {
	var empty providerHistoryTransportSourceCost
	source, sourceOK := providerHistoryCollectionRoleSourceUpper(manifest, methods, certificates, maximum, limits, prior, members, advancing)
	transport, transportOK := providerHistoryCollectionTransportRoleUpper(manifest, methods, certificates, maximum, limits.legacy.metadata, limits.fileOverhead)
	if !sourceOK || !transportOK {
		return empty, false
	}
	joined, joinedOK := providerHistoryJoinRoleComponentParts(source.joined, transport.joined)
	if !joinedOK {
		return empty, false
	}
	out := providerHistoryTransportSourceCost{source: source, transport: transport, joined: joined, storage: source.files.storage}
	var a providerHistoryReadArithmetic
	p, f := &out.storage[4], transport.failedStorage
	p.physical, p.raw = a.sum(p.physical, f.physical), a.sum(p.raw, f.raw)
	p.payload, p.metadata = a.sum(p.payload, f.payload), a.sum(p.metadata, f.metadata)
	p.objects, p.files = a.sum(p.objects, f.objects), a.sum(p.files, f.files)
	if a.failed {
		return empty, false
	}
	return out, true
}
