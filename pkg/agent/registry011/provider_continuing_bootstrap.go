package registry011

import (
	"context"
	"strconv"
)

// Immutable result DATA is not a qualified Source, operating selection or
// permission to dispatch another message. No authority is restored from it.
type providerContinuingBootstrapData struct {
	admission, ticket, bootstrap, enrollment string
	selected                                 FinalizedHeader
	highwater                                providerFrontierHighwater
	execution, remaining                     candidateWorkVector
	physical, metadataPhysical               uint64
}

func (*providerContinuingBootstrapData) UnmarshalJSON([]byte) error { return errProviderIntegration }

func (o *providerContinuingOwner) executeBootstrapCandidate(ctx context.Context) (*providerContinuingBootstrapData, error) {
	if o == nil || o.store == nil {
		return nil, errProviderIntegration
	}
	s := o.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if o.state == providerContinuingDone || o.state == providerContinuingFailed {
		return nil, errProviderIntegration
	}
	success := false
	defer func() {
		if !success {
			o.failLocked()
		}
	}()
	if ctx == nil || ctx.Err() != nil || !o.selectedLocked() || o.state != providerContinuingPrepared ||
		o.ticket == nil || o.ticket.kind != "local-work" || o.ticket.input != o.integration.firstExchange || o.integration.bootstrap != "" {
		return nil, errProviderIntegration
	}
	permit, err := o.consumeLocked()
	if err != nil || !o.burnLocked(permit) {
		return nil, errProviderIntegration
	}
	c, m := o.integration, o.execution
	// Re-read the actual previously retained completed X/S. Diagnostic
	// firstResult and the earlier operation's closed E/R are never used.
	raw, err := s.frontierReplay.body.completedExchangeResult(s, c.firstExchange, c.first)
	var header struct{ Hash, ParentHash, Number, Timestamp string }
	parseErr := providerReplayUnmarshal(raw, &header, m)
	number, numberOK := rpcQuantity(header.Number)
	timestamp, timeOK := rpcQuantity(header.Timestamp)
	selected := FinalizedHeader{header.Hash, header.ParentHash, number, timestamp}
	if err != nil || parseErr != nil || !numberOK || !timeOK || !validHeader(selected) || number < sepoliaDeploymentBlock || ctx.Err() != nil {
		return o.failedBootstrapLocked(permit)
	}
	start, err := providerContinuingBootstrapObject(c, selected, m)
	if err != nil {
		return o.failedBootstrapLocked(permit)
	}
	var work candidateWorkVector
	work[3] = 1
	if !m.add(work) {
		return o.failedBootstrapLocked(permit)
	}
	end, err := c.currentClock()
	if err != nil {
		return nil, errProviderIntegration
	}
	measurement, err := m.finish(c, o.ticket, end)
	if err != nil || !o.beginRemainingLocked(permit) {
		return nil, errProviderIntegration
	}
	// Independent result replay derives its own providerStart before S.
	// This private provisional coordinator value cannot escape on failure.
	c.ledger.providerStart = start.digest
	settlement, err := providerFirstOperationSettlementData(c, o.ticket, "complete", start.digest, end, measurement, o.remaining)
	if err != nil {
		return nil, errProviderIntegration
	}
	change, err := c.ledger.AppendCandidateSettlement(settlement)
	if err != nil || o.commitLocked("settle", settlement.digest, change, []providerIntegrationObjectInput{{"bootstrap", providerReplayPhaseBytes(start.raw, o.remaining)}, {"settlement", providerReplayPhaseBytes(settlement.raw, o.remaining)}}, providerClockDiagnostic(end)) != nil ||
		!o.settlement || s.sequence != o.before.sequence+3 || s.frontierReplay.body.start != start.digest || ctx.Err() != nil {
		return nil, errProviderIntegration
	}
	if _, err := c.currentClock(); err != nil || o.execution.failed || !o.execution.closed || !o.execution.claimed || o.remaining.failed ||
		!o.guardian.continuingPrefixMatchesLocked() {
		return nil, errProviderIntegration
	}
	result := &providerContinuingBootstrapData{admission: c.intent.digest, ticket: o.ticket.digest, bootstrap: start.digest, enrollment: s.enrollment, selected: selected, highwater: s.frontierFiles.pin, execution: m.actual, remaining: o.remaining.actual, physical: o.physical, metadataPhysical: o.metadataPhysical}
	if !o.releaseLocked() {
		return nil, errProviderIntegration
	}
	c.bootstrap, c.selected = start.digest, selected
	success = true
	return result, nil
}

// A refused local E may retain only a full-charge failed S under its distinct
// pre-held R. It cannot invent actual work or return successful result DATA.
func (o *providerContinuingOwner) failedBootstrapLocked(permit *providerIntegrationPermit) (*providerContinuingBootstrapData, error) {
	if !o.beginRemainingLocked(permit) {
		return nil, errProviderIntegration
	}
	c := o.integration
	end, err := c.currentClock()
	if err != nil {
		return nil, errProviderIntegration
	}
	settlement, err := providerFirstOperationSettlementData(c, o.ticket, "failed", "", end, nil, o.remaining)
	if err != nil {
		return nil, errProviderIntegration
	}
	change, err := c.ledger.AppendCandidateSettlement(settlement)
	if err == nil {
		_ = o.commitLocked("settle", settlement.digest, change, []providerIntegrationObjectInput{{"settlement", providerReplayPhaseBytes(settlement.raw, o.remaining)}}, providerClockDiagnostic(end))
	}
	return nil, errProviderIntegration
}

func (o *providerContinuingOwner) releaseLocked() bool {
	if !o.selectedLocked() || o.state != providerContinuingSettlement || !o.consumption || !o.settlement || o.remaining == nil ||
		o.remaining.failed || o.remaining.closed || !o.execution.closed || !o.execution.claimed || o.execution.failed || !o.guardian.continuingPrefixMatchesLocked() {
		return false
	}
	g, s := o.guardian, o.store
	r := s.frontierReplay.body.guardian
	if r == nil || r.failed || !r.terminal || !r.consumed || r.ticket != o.ticket.digest || r.head != g.pin ||
		r.metadata != g.genesisMetadata+g.auxiliaryMetadata {
		return false
	}
	o.remaining.mu.Lock()
	o.remaining.closed = true
	o.remaining.mu.Unlock()
	g.main, g.ready, g.active = s.frontierFiles.pin, true, false
	g.claim, g.meter, g.continuing = nil, nil, nil
	o.state = providerContinuingDone
	s.activeMeter.Store(nil)
	return true
}

// Canonical construction uses bounded observed helpers before every decode,
// encode, digest and copy. It has no caller callback or operating authority.
func providerContinuingBootstrapObject(c *providerIntegration, selected FinalizedHeader, m *providerWorkMeter) (providerIntegrationObject, error) {
	if c == nil || !c.valid() || c.intent == nil || c.policy == nil || c.anchor == nil || m == nil || m.continuing == nil ||
		!m.continuing.fits(m) || m.authority || !validHeader(selected) || selected.Number < sepoliaDeploymentBlock {
		return providerIntegrationObject{}, errProviderIntegration
	}
	p, pOK := providerReplayJSON(providerReplayPhaseBytes(c.policy.raw, m), m).(map[string]any)
	i, iOK := providerReplayJSON(providerReplayPhaseBytes(c.intent.raw, m), m).(map[string]any)
	instance, instanceOK := p["instance"].(map[string]any)
	if !pOK || !iOK || !instanceOK {
		return providerIntegrationObject{}, errProviderIntegration
	}
	id := c.intent.identity
	bootstrapID := providerReplayDigest("sage-provider-bootstrap-id-v2", providerReplayPhaseBytes(c.intent.digest, m), m)
	raw := providerReplayCanonical(map[string]any{"format": "sage-provider-start-v2", "admission_digest": c.intent.digest, "policy": id.Policy, "namespace": id.Namespace, "instance_key": id.Instance, "source_id": id.SourceID, "generation": strconv.FormatUint(id.Generation, 10), "bootstrap_id": bootstrapID, "enrollment_anchor_root": c.anchor.digest, "selected_finality_exchange": c.firstExchange, "selected_block_hash": selected.Hash, "deployment_block_hash": instance["deployment_hash"], "prior_published_checkpoint": nil, "initial_clock": providerClockObject(c.store.setupOrigin), "deadline_wall_ms": strconv.FormatUint(c.intent.deadline, 10), "limits_digest": i["limits_digest"]}, candidateLedgerMetadataLimit, m)
	return newProviderIntegrationObjectObserved("bootstrap", raw, candidateLedgerMetadataLimit, m)
}
