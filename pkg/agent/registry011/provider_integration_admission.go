package registry011

import (
	"strconv"
)

func (c *providerIntegration) admitRPC(request *providerIntegrationRequest, first bool) (*CandidateAdmissionTicket, error) {
	if c.prepare() != nil || request == nil || first != (c.ledger == nil) || first && !request.firstFinalized() {
		return nil, errProviderIntegration
	}
	primitive, err := c.operationPrimitive(request)
	if err != nil {
		return nil, err
	}
	var reservation providerOperationReservation
	if c.store.config.bounds.continuation == nil {
		reservation, err = providerEstimateOperation(c.store.config.bounds, primitive, uint64(len(request.wire)), c.store.retainedRaw, true, first)
	} else {
		reservation, err = providerEstimateFrontierOperation(c.store.config.bounds, primitive, uint64(len(request.wire)), true, first)
	}
	if err != nil {
		return nil, err
	}
	return c.admit("rpc-capture", request.inputDigest, request.method, request.paramsDigest, reservation, []providerIntegrationObjectInput{{"request", []byte(request.wire)}, {"params", []byte(request.params)}})
}

func (c *providerIntegration) admit(kind, input, method, params string, reservation providerOperationReservation, extra []providerIntegrationObjectInput) (*CandidateAdmissionTicket, error) {
	if !c.valid() || !lowerHash(input) {
		return nil, errProviderIntegration
	}
	meter, err := c.attachMeter(reservation)
	if err != nil {
		return nil, err
	}
	accepted := false
	defer func() {
		if !accepted {
			meter.refuse()
			c.restoreMeter(meter)
		}
	}()
	clock, err := c.currentClock()
	if err != nil {
		return nil, err
	}
	p := contentJSON([]byte(c.policy.raw)).(map[string]any)
	opDuration, _ := providerCandidateUint(p["limits"].(map[string]any)["operation"].(map[string]any)["deadline_ms"], true)
	deadline, ok := providerUpperSum(uint64(clock.WallMS-clock.UncertaintyMS), opDuration)
	if !ok {
		return nil, errProviderIntegration
	}
	deadline = min(deadline, c.intent.deadline)
	if meter.parent != nil {
		parent, exists := c.ledger.tickets[meter.parent.ticket]
		if !exists || parent.ticket == nil || parent.consumption == nil || parent.terminal != "" || candidateLedgerExpired(providerClockDiagnostic(clock), parent.ticket.deadline) {
			return nil, errProviderIntegration
		}
		deadline = min(deadline, parent.ticket.deadline)
	}
	id := c.intent.identity
	var previous, methodValue, paramsValue any
	var sequence uint64
	if c.ledger != nil {
		previous, sequence = c.ledger.lastTicket.digest, c.ledger.lastTicket.sequence+1
	}
	if kind == "rpc-capture" {
		methodValue, paramsValue = method, params
	}
	raw := providerIntegrationJSON(map[string]any{"format": "sage-provider-budget-ticket-v2", "admission_digest": c.intent.digest, "namespace": id.Namespace, "instance_key": id.Instance, "source_id": id.SourceID, "generation": strconv.FormatUint(id.Generation, 10), "fence": strconv.FormatUint(c.intent.fence, 10), "sequence": strconv.FormatUint(sequence, 10), "previous_ticket": previous, "segment": candidateAdmissionSegment(c.intent.digest, 0), "kind": kind, "input_digest": input, "method": methodValue, "params_digest": paramsValue, "reservation": candidateLedgerWorkObject(reservation.work), "live_disk_reserved": strconv.FormatUint(reservation.live, 10), "retained_bytes_reserved": strconv.FormatUint(reservation.retained, 10), "deadline_wall_ms": strconv.FormatUint(deadline, 10)})
	if !c.store.event(providerEncodeEvent, uint64(len(raw))) || !c.store.event(providerCodecEvent, uint64(len(raw))) {
		return nil, errProviderIntegration
	}
	ticket, err := ParseCandidateAdmissionTicket(c.policy, c.intent, 0, raw, 64<<10)
	if err != nil {
		return nil, errProviderIntegration
	}
	meter.ticket = ticket.digest
	c.meters[ticket.digest] = meter
	var change *CandidateAdmissionLedgerChange
	if c.ledger == nil {
		c.ledger, err = NewCandidateAdmissionLedger(c.policy, c.intent, ticket, c.store.config.bounds.ledger)
		if err == nil {
			change, err = c.ledger.Current()
		}
		extra = append(extra, providerIntegrationObjectInput{"admission", []byte(c.intent.raw)})
	} else {
		change, err = c.ledger.AppendCandidateTicket(ticket)
	}
	if err != nil {
		return nil, errProviderIntegration
	}
	extra = append(extra, providerIntegrationObjectInput{"ticket", raw})
	if _, err := c.commitChange("admit", ticket.digest, change, extra); err != nil {
		c.failed = true // The provisional model cannot roll back uncertainty.
		return nil, err
	}
	accepted = true
	return ticket, nil
}

func (c *providerIntegration) commitChange(kind, operation string, change *CandidateAdmissionLedgerChange, extra []providerIntegrationObjectInput) (*providerCustodyAcknowledgment, error) {
	if !c.valid() || change == nil || change.raw == "" || change.admission != c.intent.digest {
		return nil, errProviderIntegration
	}
	before := c.root
	m := contentJSON([]byte(before.raw)).(map[string]any)
	m["sequence"], m["previous_root"], m["budget_root"] = strconv.FormatUint(before.sequence+1, 10), before.digest, change.digest
	rootBytes := providerIntegrationJSON(m)
	root, err := ParseCandidateEnrollmentRoot(c.context, rootBytes, 64<<10)
	if err != nil {
		return nil, errProviderIntegration
	}
	for _, node := range change.LedgerNodeBytes() {
		if !c.store.event(providerEncodeEvent, uint64(len(node))) {
			return nil, errProviderIntegration
		}
		extra = append(extra, providerIntegrationObjectInput{"ledger-node", node})
	}
	extra = append(extra, providerIntegrationObjectInput{"budget", []byte(change.raw)}, providerIntegrationObjectInput{"enrollment", rootBytes})
	if !c.store.event(providerEncodeEvent, uint64(len(change.raw))) || !c.store.event(providerEncodeEvent, uint64(len(rootBytes))) {
		return nil, errProviderIntegration
	}
	objects, err := c.objects(extra)
	if err != nil {
		return nil, err
	}
	ack, err := c.store.commit(kind, operation, root, objects)
	if err != nil || !c.store.confirms(ack, operation, root) {
		return nil, errProviderIntegration
	}
	c.root = root
	return ack, nil
}

func (c *providerIntegration) consume(ticket *CandidateAdmissionTicket, request *providerIntegrationRequest) (*providerIntegrationPermit, error) {
	if !c.valid() || ticket == nil || ticket.intent != c.intent || ticket.kind == "rpc-capture" && (request == nil || ticket.input != request.inputDigest || ticket.method != request.method || ticket.params != request.paramsDigest) || ticket.kind == "local-work" && request != nil {
		return nil, errProviderIntegration
	}
	meter := c.meters[ticket.digest]
	if meter == nil || c.store.activeMeter.Load() != meter || meter.ticket != ticket.digest || meter.integration != c {
		return nil, errProviderIntegration
	}
	clock, err := c.currentClock()
	if err != nil {
		return nil, err
	}
	var previous any
	var sequence uint64
	if c.ledger.lastConsumption != nil {
		previous, sequence = c.ledger.lastConsumption.digest, c.ledger.lastConsumption.sequence+1
	}
	id := c.intent.identity
	raw := providerIntegrationJSON(map[string]any{"format": "sage-provider-budget-consumption-v1", "ticket_digest": ticket.digest, "namespace": id.Namespace, "instance_key": id.Instance, "source_id": id.SourceID, "generation": strconv.FormatUint(id.Generation, 10), "fence": strconv.FormatUint(c.intent.fence, 10), "sequence": strconv.FormatUint(sequence, 10), "previous_consumption": previous, "kind": ticket.kind, "input_digest": ticket.input, "clock_start": providerClockObject(clock)})
	consumption, err := ParseCandidateConsumption(ticket, raw, 64<<10)
	if err != nil {
		return nil, errProviderIntegration
	}
	change, err := c.ledger.AppendCandidateConsumption(consumption)
	if err != nil {
		return nil, errProviderIntegration
	}
	ack, err := c.commitChange("consume", consumption.digest, change, []providerIntegrationObjectInput{{"consumption", raw}})
	if err != nil {
		c.failed = true
		return nil, err
	}
	return &providerIntegrationPermit{owner: c, ticket: ticket, ack: ack, root: c.root, request: request, clock: clock, meter: meter}, nil
}

func (c *providerIntegration) use(permit *providerIntegrationPermit) bool {
	if !c.valid() || permit == nil || permit.owner != c || permit.used || permit.ticket == nil || permit.ack == nil || permit.root != c.root || !c.store.confirms(permit.ack, permit.ack.operation, permit.root) {
		return false
	}
	// Even an expired permit is burned; no retry/recovered ACK dispatch exists.
	permit.used = true
	clock, err := c.currentClock()
	if err != nil || !candidateLedgerClockAdvance(providerClockDiagnostic(permit.clock), providerClockDiagnostic(clock)) || candidateLedgerExpired(providerClockDiagnostic(clock), permit.ticket.deadline) {
		return false
	}
	if permit.request != nil {
		input, _ := providerIntegrationObjectDigest("request", []byte(permit.request.wire))
		if input != permit.ticket.input || permit.ticket.params != providerCandidateHash("sage-provider-params-v1", []byte(permit.request.params)) || permit.ticket.method != permit.request.method {
			return false
		}
	}
	var began candidateWorkVector
	began[8] = permit.ticket.work[8]
	if permit.request != nil {
		began[0] = 1
	}
	if permit.meter == nil || !permit.meter.add(began) {
		return false
	}
	permit.clock = clock
	return true
}
