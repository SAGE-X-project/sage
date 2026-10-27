package registry011

import (
	"encoding/json"
	"strconv"
)

// Local work is a closed internal command, not a caller callback. Its immutable
// input object must already be retained. Parsing and replay begin only after
// the separate consumption ACK and irreversible private one-use transition.
func (c *providerIntegration) beginLocal(input string) (*providerIntegrationPermit, error) {
	return c.beginLocalObjects(input, nil)
}

func (c *providerIntegration) beginLocalObjects(input string, objects []providerIntegrationObjectInput) (*providerIntegrationPermit, error) {
	if c.prepare() != nil || !lowerHash(input) {
		return nil, errProviderIntegration
	}
	primitive, err := c.localPrimitive(input, objects)
	if err != nil {
		return nil, err
	}
	reservation, err := providerEstimateOperation(c.store.config.bounds, primitive, 0, c.store.retainedRaw, false, false)
	if err != nil {
		return nil, err
	}
	ticket, err := c.admit("local-work", input, "", "", reservation, objects)
	if err != nil {
		return nil, err
	}
	permit, err := c.consume(ticket, nil)
	if err != nil || !c.use(permit) {
		c.failed = true
		return nil, errProviderIntegration
	}
	return permit, nil
}

// These two closed commands decode one header each. They do not execute EVM
// transactions/frames/logs/proofs, so reserving the unrelated full replay
// maxima would exhaust the shared lifetime quota before the second command.
// Unsupported local input cannot borrow this narrower reservation.
func (c *providerIntegration) localPrimitive(input string, objects []providerIntegrationObjectInput) (providerPrimitiveBounds, error) {
	if !c.valid() || !lowerHash(input) {
		return providerPrimitiveBounds{}, errProviderIntegration
	}
	bootstrap := c.bootstrap == "" && lowerHash(c.firstExchange) && input == c.firstExchange && c.firstResult != "" && len(objects) == 0
	pin := false
	if lowerHash(c.bootstrap) && c.pin == "" && len(objects) == 1 && objects[0].kind == "pin-input" {
		m := contentObject(contentJSON(objects[0].raw), "format", "bootstrap_start", "registry", "agent")
		if m != nil {
			expected := providerPinInput(c.bootstrap, contentString(m["agent"]))
			pin = len(expected) != 0 && string(expected) == string(objects[0].raw) && input == providerCandidateHash("sage-provider-owned-pin-input-v1", expected)
		}
	}
	if !bootstrap && !pin {
		return providerPrimitiveBounds{}, errProviderIntegration
	}
	primitive := c.primitive
	primitive.blocks, primitive.transactions, primitive.frames, primitive.logs, primitive.proofs = 1, 0, 0, 0, 0
	return primitive, nil
}

// Called once after the first real RPC has been settled and durably retained.
// The start names E0 and the completed X, never a future enrollment/budget.
func (c *providerIntegration) installBootstrap() error {
	if c == nil {
		return errProviderIntegration
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid() || c.bootstrap != "" || !lowerHash(c.firstExchange) || c.firstResult == "" || c.ledger.providerStart != "" {
		return errProviderIntegration
	}
	permit, err := c.beginLocal(c.firstExchange)
	if err != nil {
		return err
	}
	defer c.restoreMeter(permit.meter)
	var header struct {
		Hash, ParentHash, Number, Timestamp string
	}
	if !permit.meter.event(providerCodecEvent, uint64(len(c.firstResult))) {
		c.failed = true
		return errProviderIntegration
	}
	parseErr := json.Unmarshal([]byte(c.firstResult), &header)
	number, validNumber := rpcQuantity(header.Number)
	timestamp, validTime := rpcQuantity(header.Timestamp)
	selected := FinalizedHeader{header.Hash, header.ParentHash, number, timestamp}
	clock, clockErr := c.store.sample()
	if parseErr != nil || !validNumber || !validTime || !validHeader(selected) || selected.Number < sepoliaDeploymentBlock || clockErr != nil || candidateLedgerExpired(providerClockDiagnostic(clock), permit.ticket.deadline) {
		if clockErr == nil {
			if c.beginRemaining(permit) == nil {
				_ = c.settle(permit.ticket, "failed", "", clock, nil, nil)
			}
		}
		c.failed = true
		return errProviderIntegration
	}
	// These are additional decoder invocations after the consumption permit,
	// not work already observed by the earlier admission or result decoder.
	if !permit.meter.event(providerCodecEvent, uint64(len(c.policy.raw))) || !permit.meter.event(providerCodecEvent, uint64(len(c.intent.raw))) {
		c.failed = true
		return errProviderIntegration
	}
	p := contentJSON([]byte(c.policy.raw)).(map[string]any)
	i := contentJSON([]byte(c.intent.raw)).(map[string]any)
	id := c.intent.identity
	const bootstrapDomain = "sage-provider-bootstrap-id-v2"
	if !permit.meter.event(providerHashEvent, uint64(len(bootstrapDomain)+1+len(c.intent.digest))) {
		c.failed = true
		return errProviderIntegration
	}
	bootstrapID := providerCandidateHash(bootstrapDomain, []byte(c.intent.digest))
	raw := providerIntegrationJSON(map[string]any{"format": "sage-provider-start-v2", "admission_digest": c.intent.digest, "policy": id.Policy, "namespace": id.Namespace, "instance_key": id.Instance, "source_id": id.SourceID, "generation": strconv.FormatUint(id.Generation, 10), "bootstrap_id": bootstrapID, "enrollment_anchor_root": c.anchor.digest, "selected_finality_exchange": c.firstExchange, "selected_block_hash": selected.Hash, "deployment_block_hash": p["instance"].(map[string]any)["deployment_hash"], "prior_published_checkpoint": nil, "initial_clock": providerClockObject(c.store.setupOrigin), "deadline_wall_ms": strconv.FormatUint(c.intent.deadline, 10), "limits_digest": i["limits_digest"]})
	// Observe the encoder and the new object's decoder/hash/copy separately.
	if !permit.meter.event(providerEncodeEvent, uint64(len(raw))) || !permit.meter.event(providerCodecEvent, uint64(len(raw))) || !permit.meter.event(providerHashEvent, uint64(len(raw)+len("sage-provider-start-v2")+1)) || !permit.meter.event(providerCopyEvent, uint64(len(raw))) {
		c.failed = true
		return errProviderIntegration
	}
	start, err := newProviderIntegrationObject("bootstrap", raw, c.store.config.bounds.maximumObjectBytes)
	if err != nil {
		c.failed = true
		return err
	}
	var blocks candidateWorkVector
	blocks[3] = 1
	if !permit.meter.add(blocks) {
		c.failed = true
		return errProviderIntegration
	}
	clock, err = c.store.sample()
	if err != nil {
		c.failed = true
		return errProviderIntegration
	}
	measurement, err := permit.meter.finish(c, permit.ticket, clock)
	if err != nil {
		c.failed = true
		return errProviderIntegration
	}
	if c.beginRemaining(permit) != nil {
		c.failed = true
		return errProviderIntegration
	}
	// This private instance has never escaped to a public candidate-data caller.
	// Installation occurs only after real capture/consumption/settlement checks.
	c.ledger.providerStart = start.digest
	if err := c.settle(permit.ticket, "complete", start.digest, clock, []providerIntegrationObjectInput{{"bootstrap", raw}}, measurement); err != nil {
		c.failed = true
		return err
	}
	if _, err := c.currentClock(); err != nil {
		c.failed = true
		return err
	}
	c.bootstrap, c.selected = start.digest, selected
	return nil
}
