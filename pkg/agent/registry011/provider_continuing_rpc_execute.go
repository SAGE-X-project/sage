package registry011

import (
	"context"
	"crypto/tls"
)

// Separate immutable result DATA cannot restore a completed role or select a
// Source. The caller must still validate the method-specific result semantics.
type providerContinuingRPCData struct {
	admission, ticket, exchange, enrollment string
	highwater                               providerFrontierHighwater
	execution, remaining                    candidateWorkVector
	physical, metadataPhysical              uint64
	response, result                        []byte
}

func (*providerContinuingRPCData) UnmarshalJSON([]byte) error { return errProviderIntegration }

// The production entry has no endpoint, connection or transport injection.
func (o *providerContinuingOwner) executeRPCCandidate(ctx context.Context) (*providerContinuingRPCData, error) {
	return o.executeRPCCandidateWithConnection(ctx, nil)
}

// A nonnil connection is used only by ordinary owned TLS tests of this private
// candidate. It cannot construct/select a Source, principal or Agent permit.
// Production always dials the one fixed official endpoint AFTER C and burn.
func (o *providerContinuingOwner) executeRPCCandidateWithConnection(ctx context.Context, connection *tls.Conn) (*providerContinuingRPCData, error) {
	if o == nil || o.store == nil {
		return nil, errProviderIntegration
	}
	o.store.mu.Lock()
	defer o.store.mu.Unlock()
	if o.state == providerContinuingDone || o.state == providerContinuingFailed {
		return nil, errProviderIntegration
	}
	success := false
	defer func() {
		if !success {
			o.failLocked()
		}
	}()
	if ctx == nil || ctx.Err() != nil || !o.selectedLocked() || o.state != providerContinuingPrepared || o.ticket == nil || o.ticket.kind != "rpc-capture" || o.request == nil || !o.capture.valid() || o.config.bounds.tls == nil {
		return nil, errProviderIntegration
	}
	permit, err := o.consumeLocked()
	if err != nil || !o.burnLocked(permit) {
		return nil, errProviderIntegration
	}
	bounded, cancel := context.WithTimeout(ctx, o.capture.Timeout)
	defer cancel()
	if connection == nil {
		connection, err = providerIntegrationDial(bounded, o.capture)
	}
	var sample *ProviderTransportSample
	if err == nil {
		defer func() { _ = connection.Close() }()
		sample, err = providerFirstOperationCapture(bounded, connection, o.capture, o.request, o.execution, *o.config.bounds.tls)
	}
	return o.finishRPCLocked(permit, sample, err, &success)
}

func (o *providerContinuingOwner) finishRPCLocked(permit *providerIntegrationPermit, sample *ProviderTransportSample, executionErr error, success *bool) (*providerContinuingRPCData, error) {
	c, s := o.integration, o.store
	outcome, retained := "failed", ""
	var measurement *providerMeasuredWork
	var extra []providerIntegrationObjectInput
	var end ClockSample
	if executionErr == nil && sample != nil && sample.Complete {
		var err error
		end, err = c.currentClock()
		if err == nil {
			extra, retained, err = providerOperationExchangeData(c, permit, sample, end, o.execution, o.capture)
		}
		if err == nil {
			end, err = c.currentClock()
		}
		if err == nil {
			measurement, err = o.execution.finish(c, permit.ticket, end)
		}
		if err == nil {
			outcome = "complete"
		} else {
			retained, extra, measurement = "", nil, nil
		}
	}
	if !o.beginRemainingLocked(permit) {
		return nil, errProviderIntegration
	}
	// A poisoned transport observer cannot supply an end sample. Use the
	// distinct pre-held R clock budget; never report a fictitious E actual.
	if end == (ClockSample{}) {
		var err error
		end, err = c.currentClock()
		if err != nil {
			return nil, errProviderIntegration
		}
	}
	settlement, err := providerFirstOperationSettlementData(c, permit.ticket, outcome, retained, end, measurement, o.remaining)
	if err != nil {
		return nil, errProviderIntegration
	}
	change, err := c.ledger.AppendCandidateSettlement(settlement)
	if err != nil {
		return nil, errProviderIntegration
	}
	extra = append(extra, providerIntegrationObjectInput{"settlement", providerReplayPhaseBytes(settlement.raw, o.remaining)})
	if o.commitLocked("settle", settlement.digest, change, extra, providerClockDiagnostic(end)) != nil || !o.settlement || s.sequence != o.before.sequence+3 {
		return nil, errProviderIntegration
	}
	if _, err := c.currentClock(); err != nil || outcome != "complete" || sample == nil || s.frontierReplay.body.firstExchange != c.firstExchange || o.execution.failed || !o.execution.closed || !o.execution.claimed {
		return nil, errProviderIntegration
	}
	// Publication copies are separately observed under R after durable S.
	// They are not part of the previously frozen execution measurement.
	if !o.remaining.event(providerCopyEvent, uint64(len(sample.ResponseBody))) || !o.remaining.event(providerCopyEvent, uint64(len(sample.Result))) {
		return nil, errProviderIntegration
	}
	result := &providerContinuingRPCData{admission: c.intent.digest, ticket: permit.ticket.digest, exchange: retained, enrollment: s.enrollment, highwater: s.frontierFiles.pin, execution: o.execution.actual, remaining: o.remaining.actual, physical: o.physical, metadataPhysical: o.metadataPhysical, response: append([]byte(nil), sample.ResponseBody...), result: append([]byte(nil), sample.Result...)}
	if !o.selectedLocked() || o.remaining.failed || !o.guardian.continuingPrefixMatchesLocked() || s.bindings[retained] == "" {
		return nil, errProviderIntegration
	}
	result.remaining = o.remaining.actual
	if !o.releaseLocked() {
		return nil, errProviderIntegration
	}
	c.lastExchange = retained
	*success = true
	return result, nil
}
