package registry011

// Only closed internal commands call this after constructing their exact
// measured result. No caller may provide a measurement or work callback.
func (c *providerIntegration) finishLocalResult(permit *providerIntegrationPermit, kind string, raw []byte) (string, error) {
	if !c.valid() || permit == nil || permit.owner != c || !permit.used || permit.request != nil || c.store.activeMeter.Load() != permit.meter {
		return "", errProviderIntegration
	}
	switch kind {
	case "pin":
	default:
		return "", errProviderIntegration
	}
	if !permit.meter.event(providerEncodeEvent, uint64(len(raw))) || !permit.meter.event(providerCodecEvent, uint64(len(raw))) || !permit.meter.event(providerHashEvent, uint64(len("sage-provider-owned-pin-v1")+1+len(raw))) || !permit.meter.event(providerCopyEvent, uint64(len(raw))) {
		c.failed = true
		return "", errProviderIntegration
	}
	object, err := newProviderIntegrationObject(kind, raw, c.store.config.bounds.maximumObjectBytes)
	if err != nil {
		c.failed = true
		return "", err
	}
	clock, err := c.store.sample()
	if err != nil {
		c.failed = true
		return "", err
	}
	measurement, err := permit.meter.finish(c, permit.ticket, clock)
	if err != nil || c.beginRemaining(permit) != nil {
		c.failed = true
		return "", errProviderIntegration
	}
	if err := c.settle(permit.ticket, "complete", object.digest, clock, []providerIntegrationObjectInput{{kind, raw}}, measurement); err != nil {
		c.failed = true
		return "", err
	}
	if clock, err := c.currentClock(); err != nil || candidateLedgerExpired(providerClockDiagnostic(clock), permit.ticket.deadline) {
		c.failed = true
		return "", errProviderIntegration
	}
	return object.digest, nil
}

func (c *providerIntegration) failLocal(permit *providerIntegrationPermit) {
	if c.valid() && permit != nil && permit.owner == c && permit.used && c.store.activeMeter.Load() == permit.meter {
		clock, err := c.store.sample()
		if err == nil && c.beginRemaining(permit) == nil {
			_ = c.settle(permit.ticket, "failed", "", clock, nil, nil)
		}
	}
	c.failed = true // Unknown/failed work is never retried or refunded.
}
