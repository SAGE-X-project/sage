package registry011

import "context"

func (g *providerHistoryRecoveryGroupModel) settlement(ctx context.Context, c *providerHistoryRecoveryCheckpoint, record *providerHistoryRecoveryGroupRecord, retention *providerHistoryStorageCharge) (bool, error) {
	if ctx == nil || ctx.Err() != nil {
		return false, errProviderIntegration
	}
	r, ledger := g.recovery, g.ledger
	if retention == nil || ledger.ticket == nil || ledger.consumption == nil || c.previous == nil || c.previous.frame.phase != "consume" ||
		c.completed < 0 || c.completed > ledger.ticket.table.count || !providerHistoryGroupClockFits(c.admission.clock, record.frame.clock) {
		return false, errProviderIntegration
	}
	outcome, failureDigest, resultIndex := "complete", "", 0
	wantCount := 2
	if c.failedS {
		if c.completed >= ledger.ticket.table.count {
			return false, errProviderIntegration
		}
		outcome, resultIndex = "failed", 1
		if c.completed > 0 {
			wantCount = 3
		}
	} else if c.completed != ledger.ticket.table.count {
		return false, errProviderIntegration
	}
	if c.inventory.count != wantCount || record.objects[c.inventory.count-1].kind != "settlement" {
		return false, errProviderIntegration
	}
	policy, policyOK := providerReplayJSON(providerReplayBytes(g.initial.frontier.body.policy.raw, r.meter), r.meter).(map[string]any)
	provider, providerOK := policy["provider"].(map[string]any)
	roots := contentString(provider["tls_roots_artifact"])
	if !policyOK || !providerOK || !lowerHash(roots) {
		return false, errProviderIntegration
	}
	if c.failedS {
		if record.objects[0].kind != "failure" || r.memberAttempt == nil || r.memberAttempt.ordinal != c.completed ||
			r.memberAttempt.consumption != c.previous || r.memberAttempt.stage != record.frame.stage {
			return false, errProviderIntegration
		}
		failureDigest = record.objects[0].digest
		failure, err := parseProviderHistoryFailureOriginalObserved(providerReplayBytes(record.objects[0].raw, r.meter), failureDigest,
			ledger.ticket, ledger.consumption, c.completed, roots, r.parent.tls, r.config.bounds.maximumObjectBytes, r.meter)
		if err != nil || failure.stage != c.failureStage || failure.written != c.failureWritten || failure.received != c.failureReceived ||
			!providerReplayTextEqual(r.memberAttempt.member.request.wire, ledger.ticket.table.members[c.completed].request.wire, r.meter) {
			return false, errProviderIntegration
		}
	}
	var results [providerHistoryGroupMembers]providerHistoryMemberResult
	if c.completed > 0 {
		object := record.objects[resultIndex]
		if object.kind != "results" {
			return false, errProviderIntegration
		}
		pack, parsed, err := parseProviderHistoryResultPackObserved(providerReplayBytes(object.raw, r.meter), object.digest, ledger.ticket,
			ledger.consumption, c.completed, roots, r.parent.tls, r.config.bounds.maximumObjectBytes, r.meter)
		if err != nil {
			return false, errProviderIntegration
		}
		results = parsed
		for i := range c.completed {
			whole, err := g.memberOriginals(ctx, c, pack, results[i], i, retention)
			if err != nil || !whole {
				return false, err
			}
		}
	}
	if !r.meter.event(providerCopyEvent, uint64(c.completed)) {
		return false, errProviderIntegration
	}
	digests := make([]string, c.completed)
	for i := range digests {
		digests[i] = results[i].digest
	}
	last := record.objects[c.inventory.count-1]
	settlement, err := parseProviderHistoryGroupSettlementObserved(providerReplayBytes(last.raw, r.meter), ledger.ticket, ledger.consumption,
		ledger.head, g.inventory, outcome, failureDigest, digests, r.meter)
	if err != nil || !providerReplayTextEqual(settlement.digest, record.frame.operation, r.meter) || ledger.settleOriginalsObserved(ctx, settlement, results[:c.completed], r.config.bounds.methods, r.meter) != nil {
		return false, errProviderIntegration
	}
	if ctx.Err() != nil || c.failedS && !ledger.terminalDataValidObserved(r.meter) || !c.failedS && !ledger.validObserved(r.meter) {
		return false, errProviderIntegration
	}
	return true, nil
}

// Each actual spool must equal both the sealed original completion and the S
// pack's verbatim payload. This cannot turn TLS diagnostic JSON into a fresh
// live connection, resend a member, or acknowledge an uncertain S.
func (g *providerHistoryRecoveryGroupModel) memberOriginals(ctx context.Context, c *providerHistoryRecoveryCheckpoint, pack *providerHistoryOriginalPack, result providerHistoryMemberResult, ordinal int, retention *providerHistoryStorageCharge) (bool, error) {
	r := g.recovery
	sealed := r.settledMemberLocked(c, ordinal)
	if retention == nil || sealed == nil || sealed.capture.intent.consumption != c.previous {
		return false, errProviderIntegration
	}
	response, whole, err := g.read(ctx, c, providerHistoryRecoveryMemberResponse, ordinal)
	if err != nil || !whole {
		return false, err
	}
	tls, whole, err := g.read(ctx, c, providerHistoryRecoveryMemberTLS, ordinal)
	if err != nil || !whole {
		return false, err
	}
	capture := sealed.capture
	_, expectedResponse, a := pack.original(2 * ordinal)
	_, expectedTLS, b := pack.original(2*ordinal + 1)
	if !a || !b || uint64(len(response)) != capture.responseBytes || uint64(len(tls)) != capture.tlsBytes ||
		!providerReplayTextEqual(providerReplayDigest("", response, r.meter), capture.response, r.meter) ||
		!providerReplayTextEqual(providerReplayDigest("", tls, r.meter), capture.tls, r.meter) ||
		!providerReplayEqual(response, providerReplayBytes(expectedResponse, r.meter), r.meter) ||
		!providerReplayEqual(tls, providerReplayBytes(expectedTLS, r.meter), r.meter) || !providerReplayTextEqual(result.digest, sealed.result, r.meter) {
		return false, errProviderIntegration
	}
	raw, a := providerUpperSum(uint64(len(response)), uint64(len(tls)))
	overhead, b := providerUpperProduct(r.config.bounds.fileOverheadBytes, 2)
	physical, ok := providerUpperSum(raw, overhead)
	charge := providerHistoryStorageCharge{physical: physical, raw: raw, payload: uint64(len(response)), metadata: uint64(len(tls)), objects: 2, files: 2}
	retained, x := retention.add(charge, r.parent.bounds.future)
	groupRetained, y := retention.add(charge, g.ledger.charged)
	if !a || !b || !ok || !x || !y || retained != groupRetained {
		return false, errProviderIntegration
	}
	*retention = retained
	return true, nil
}
