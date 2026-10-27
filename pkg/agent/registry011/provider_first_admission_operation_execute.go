package registry011

import (
	"context"
	"crypto/tls"
)

// The production entry has no endpoint, connection or transport injection.
func (o *providerFirstOperationOwner) executeCandidate(ctx context.Context) (*providerFirstOperationResultData, error) {
	return o.executeCandidateWithConnection(ctx, nil)
}

// A nonnil connection is used only by ordinary owned TLS tests of this private
// candidate. It cannot construct/select a Source, principal or Agent permit.
// Production always dials the one fixed official endpoint AFTER C and burn.
func (o *providerFirstOperationOwner) executeCandidateWithConnection(ctx context.Context, connection *tls.Conn) (*providerFirstOperationResultData, error) {
	if o == nil || o.store == nil {
		return nil, errProviderIntegration
	}
	o.store.mu.Lock()
	defer o.store.mu.Unlock()
	if o.state == providerFirstOperationDone || o.state == providerFirstOperationFailed {
		return nil, errProviderIntegration
	}
	success := false
	defer func() {
		if !success {
			o.failLocked()
		}
	}()
	if ctx == nil || ctx.Err() != nil || !o.selectedLocked() || o.state != providerFirstOperationPrepared {
		return nil, errProviderIntegration
	}
	permit, err := o.consumeLocked()
	if err != nil || !o.burnLocked(permit) {
		return nil, errProviderIntegration
	}
	bounded, cancel := context.WithTimeout(ctx, o.plan.capture.Timeout)
	defer cancel()
	if connection == nil {
		connection, err = providerIntegrationDial(bounded, o.plan.capture)
	}
	var sample *ProviderTransportSample
	if err == nil {
		defer connection.Close()
		sample, err = providerFirstOperationCapture(bounded, connection, o.plan.capture, o.request, o.execution, o.tls)
	}
	return o.finishLocked(permit, sample, err, &success)
}

func (o *providerFirstOperationOwner) finishLocked(permit *providerIntegrationPermit, sample *ProviderTransportSample, executionErr error, success *bool) (*providerFirstOperationResultData, error) {
	c, s := o.integration, o.store
	outcome, retained := "failed", ""
	var measurement *providerMeasuredWork
	var extra []providerIntegrationObjectInput
	var end ClockSample
	if executionErr == nil && sample != nil && sample.Complete {
		var err error
		end, err = c.currentClock()
		if err == nil {
			extra, retained, err = providerFirstOperationExchangeData(c, permit, sample, end, o.execution)
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
	if o.commitLocked("settle", settlement.digest, change, extra, providerClockDiagnostic(end)) != nil || !o.settlement || s.sequence != 4 {
		return nil, errProviderIntegration
	}
	if _, err := c.currentClock(); err != nil || outcome != "complete" || sample == nil || s.frontierReplay.body.firstExchange != retained || o.execution.failed || !o.execution.closed || !o.execution.claimed {
		return nil, errProviderIntegration
	}
	// Publication copies are separately observed under R after durable S.
	// They are not part of the previously frozen execution measurement.
	if !o.remaining.event(providerCopyEvent, uint64(len(sample.ResponseBody))) || !o.remaining.event(providerCopyEvent, uint64(len(sample.Result))) {
		return nil, errProviderIntegration
	}
	result := &providerFirstOperationResultData{admission: c.intent.digest, ticket: permit.ticket.digest, exchange: retained, enrollment: s.enrollment, highwater: s.frontierFiles.pin, execution: o.execution.actual, remaining: o.remaining.actual, physical: o.physical, response: append([]byte(nil), sample.ResponseBody...), result: append([]byte(nil), sample.Result...)}
	if !o.selectedLocked() || o.remaining.failed || !providerFirstOperationModelsEqual(s, o.indexes, s.frontierReplay.indexes, o.remaining) {
		return nil, errProviderIntegration
	}
	// Include the final bounded equality check in R, never in E's actual.
	result.remaining = o.remaining.actual
	o.remaining.mu.Lock()
	o.remaining.closed = true
	o.remaining.mu.Unlock()
	c.firstExchange, c.lastExchange = retained, retained
	o.state = providerFirstOperationDone
	s.activeMeter.Store(nil)
	*success = true
	return result, nil
}
