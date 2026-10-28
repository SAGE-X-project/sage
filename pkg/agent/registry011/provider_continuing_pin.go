package registry011

import "context"

// Private immutable DATA cannot restore the command, select custody or enable
// protected Agent work. The caller still needs full history and qualification.
type providerContinuingPinData struct {
	admission, ticket, bootstrap, input, pin, selected string
	exchanges                                          [4]string
	highwater                                          providerFrontierHighwater
	execution, remaining                               candidateWorkVector
	physical, metadataPhysical                         uint64
}

type providerContinuingPinCommand struct {
	parent    *providerContinuingOwner
	input     providerIntegrationObject
	selected  string
	exchanges [4]string
	requests  [4]*providerIntegrationRequest
	captures  [4]ProviderCaptureLimits
}

func (*providerContinuingPinData) UnmarshalJSON([]byte) error    { return errProviderIntegration }
func (*providerContinuingPinCommand) UnmarshalJSON([]byte) error { return errProviderIntegration }

func (o *providerContinuingOwner) executePinCandidate(ctx context.Context) (*providerContinuingPinData, error) {
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
	if ctx == nil || ctx.Err() != nil || !o.selectedLocked() || o.state != providerContinuingPrepared || o.pin == nil || o.pin.parent != o || o.ticket == nil || o.ticket.kind != "local-work" || o.ticket.input != o.pin.input.digest || o.request != nil || o.integration.pin != "" {
		return nil, errProviderIntegration
	}
	permit, err := o.consumeLocked()
	if err != nil || !o.burnLocked(permit) {
		return nil, errProviderIntegration
	}
	c, m, r := o.integration, o.execution, s.frontierReplay.body
	_, selected, err := r.pinInput(s, o.ticket.input)
	if err != nil || selected != o.pin.selected || ctx.Err() != nil {
		return o.failedPinLocked(permit)
	}
	results, err := r.continuingPinResults(s, o.pin.exchanges, selected)
	if err != nil {
		return o.failedPinLocked(permit)
	}
	for part, result := range results {
		if !providerCheckPinPartObserved(part, result, m) || ctx.Err() != nil {
			return o.failedPinLocked(permit)
		}
	}
	var work candidateWorkVector
	work[3] = 1 // This local command validates exactly one genesis header.
	if !m.add(work) {
		return o.failedPinLocked(permit)
	}
	object, err := providerContinuingPinObject(o.ticket.input, selected, o.pin.exchanges, m)
	if err != nil || ctx.Err() != nil {
		return o.failedPinLocked(permit)
	}
	// Exactly one E end sample. The same frozen sample goes to finish and S;
	// this local path does not copy the RPC path's second end-clock read.
	end, err := c.currentClock()
	if err != nil {
		return nil, errProviderIntegration
	}
	measurement, err := m.finish(c, o.ticket, end)
	if err != nil {
		return o.failedPinLocked(permit)
	}
	if !o.beginRemainingLocked(permit) {
		return nil, errProviderIntegration
	}
	settlement, err := providerFirstOperationSettlementData(c, o.ticket, "complete", object.digest, end, measurement, o.remaining)
	if err != nil {
		return nil, errProviderIntegration
	}
	change, err := c.ledger.AppendCandidateSettlement(settlement)
	if err != nil || o.commitLocked("settle", settlement.digest, change, []providerIntegrationObjectInput{{"pin", providerReplayPhaseBytes(object.raw, o.remaining)}, {"settlement", providerReplayPhaseBytes(settlement.raw, o.remaining)}}, providerClockDiagnostic(end)) != nil || !o.settlement || s.sequence != o.before.sequence+3 || ctx.Err() != nil {
		return nil, errProviderIntegration
	}
	// The independent S path has now re-read pin input and all four children.
	if _, err := c.currentClock(); err != nil || !o.guardian.continuingPrefixMatchesLocked() || o.execution.failed || !o.execution.closed || !o.execution.claimed || o.remaining.failed {
		return nil, errProviderIntegration
	}
	data := &providerContinuingPinData{admission: c.intent.digest, ticket: o.ticket.digest, bootstrap: c.bootstrap, input: o.ticket.input, pin: object.digest, selected: selected, exchanges: o.pin.exchanges, highwater: s.frontierFiles.pin, execution: m.actual, remaining: o.remaining.actual, physical: o.physical, metadataPhysical: o.metadataPhysical}
	if !o.releaseLocked() {
		return nil, errProviderIntegration
	}
	c.pin = object.digest
	success = true
	return data, nil
}

// One failed S attempt only. It cannot fall through to the success final
// clock/release path or restart after a remaining-phase failure.
func (o *providerContinuingOwner) failedPinLocked(permit *providerIntegrationPermit) (*providerContinuingPinData, error) {
	_, err := o.failedBootstrapLocked(permit)
	return nil, err
}

func providerContinuingPinObject(input, selected string, exchanges [4]string, meter *providerWorkMeter) (providerIntegrationObject, error) {
	if !lowerHash(input) || !lowerHash(selected) || meter == nil || meter.closed || meter.failed {
		return providerIntegrationObject{}, errProviderIntegration
	}
	for i, digest := range exchanges {
		if !lowerHash(digest) {
			return providerIntegrationObject{}, errProviderIntegration
		}
		for j := 0; j < i; j++ {
			if digest == exchanges[j] {
				return providerIntegrationObject{}, errProviderIntegration
			}
		}
	}
	raw := providerReplayCanonical(map[string]any{"format": "sage-provider-owned-pin-v1", "input": input, "selected_block_hash": selected, "exchanges": exchanges[:]}, 4096, meter)
	return newProviderIntegrationObjectObserved("pin", raw, 4096, meter)
}
