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
	reservation, err := providerEstimateOperation(c.store.config.bounds, c.primitive, 0, c.store.retainedRaw, false, false)
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
	p := contentJSON([]byte(c.policy.raw)).(map[string]any)
	i := contentJSON([]byte(c.intent.raw)).(map[string]any)
	id := c.intent.identity
	raw := providerIntegrationJSON(map[string]any{"format": "sage-provider-start-v2", "admission_digest": c.intent.digest, "policy": id.Policy, "namespace": id.Namespace, "instance_key": id.Instance, "source_id": id.SourceID, "generation": strconv.FormatUint(id.Generation, 10), "bootstrap_id": providerCandidateHash("sage-provider-bootstrap-id-v2", []byte(c.intent.digest)), "enrollment_anchor_root": c.anchor.digest, "selected_finality_exchange": c.firstExchange, "selected_block_hash": selected.Hash, "deployment_block_hash": p["instance"].(map[string]any)["deployment_hash"], "prior_published_checkpoint": nil, "initial_clock": providerClockObject(c.store.setupOrigin), "deadline_wall_ms": strconv.FormatUint(c.intent.deadline, 10), "limits_digest": i["limits_digest"]})
	start, err := newProviderIntegrationObject("bootstrap", raw, c.store.config.bounds.maximumObjectBytes)
	if err != nil {
		c.failed = true
		return err
	}
	var blocks candidateWorkVector
	blocks[3] = 1
	if !permit.meter.add(blocks) || !permit.meter.event(providerCodecEvent, uint64(len(raw))) || !permit.meter.event(providerHashEvent, uint64(len(raw))) || !permit.meter.event(providerCopyEvent, uint64(len(raw))) {
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
