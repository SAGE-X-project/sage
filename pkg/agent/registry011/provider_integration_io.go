package registry011

// Only the selected owned store invokes these fixed primitives. Successful
// exact writes count actual serialized bytes; unknown/partial writes poison
// measurement and cannot produce complete. Physical allocation is separately
// bounded by the selected profile, never inferred from this byte count.
func (s *providerCustodyStore) read(disk *retentionDisk, name string, maximum int64) ([]byte, error) {
	if maximum <= 0 {
		return nil, errProviderIntegration
	}
	if s.scanBody != nil && s.scanBody.active && !s.scanBody.beforeRead(disk, name, uint64(maximum)) {
		return nil, errProviderIntegration
	}
	if s.initialInspection != nil && s.initialInspection.active && !s.initialInspection.beforeRead(disk, name, uint64(maximum)) {
		return nil, errProviderIntegration
	}
	if m := s.activeMeter.Load(); m != nil && !m.preflight(providerReadEvent, uint64(maximum)) {
		return nil, errProviderIntegration
	}
	raw, err := disk.read(name, maximum)
	if err != nil {
		if m := s.activeMeter.Load(); m != nil {
			m.refuse()
		}
		return nil, err
	}
	if !s.event(providerReadEvent, uint64(len(raw))) {
		return nil, errProviderIntegration
	}
	if s.initialInspection != nil && s.initialInspection.active && !s.initialInspection.afterRead(disk, name, uint64(len(raw))) {
		return nil, errProviderIntegration
	}
	if s.scanBody != nil && s.scanBody.active && !s.scanBody.afterRead(disk, name, uint64(len(raw))) {
		return nil, errProviderIntegration
	}
	return raw, nil
}

func (s *providerCustodyStore) write(disk *retentionDisk, name string, raw []byte) error {
	if !s.event(providerWriteEvent, uint64(len(raw))) || !s.event(providerSyncEvent, 0) || !s.event(providerSyncEvent, 0) {
		return errProviderIntegration
	}
	err := disk.write(name, raw)
	if err != nil {
		if m := s.activeMeter.Load(); m != nil {
			m.refuse()
		}
		return err
	}
	return nil
}

func (s *providerCustodyStore) event(kind providerWorkEvent, bytes uint64) bool {
	m := s.activeMeter.Load()
	return m == nil || m.event(kind, bytes)
}

func (s *providerCustodyStore) sample() (ClockSample, error) {
	if m := s.activeMeter.Load(); m != nil && m.continuing != nil {
		if m.continuing.store != s {
			return ClockSample{}, errProviderIntegration
		}
		return m.continuing.sampleLocked()
	}
	if m := s.activeMeter.Load(); m != nil && m.guardian != nil {
		if m.guardian.store != s {
			return ClockSample{}, errProviderIntegration
		}
		return m.guardian.sampleLocked()
	}
	if m := s.activeMeter.Load(); m != nil && m.firstAdmission != nil {
		if m.firstAdmission.store != s {
			return ClockSample{}, errProviderIntegration
		}
		return m.firstAdmission.sampleLocked()
	}
	return s.samplePrimitive()
}

// Only the generic non-F path and the exact first-finishing owner call this
// primitive. F validates the returned time before any caller can use it.
func (s *providerCustodyStore) samplePrimitive() (ClockSample, error) {
	if s == nil || s.config.bounds.history != nil && (s.history == nil || !s.history.matchesLocked()) || s.config.bounds.history == nil && s.history != nil {
		return ClockSample{}, errProviderIntegration
	}
	m := s.activeMeter.Load()
	if s.history != nil && !s.history.clockRoleLocked(m) {
		return ClockSample{}, errProviderIntegration
	}
	if !s.event(providerClockEvent, 0) {
		return ClockSample{}, errProviderIntegration
	}
	var claim *providerHistoryClockClaim
	if s.history != nil {
		claim = s.history.issueClockClaimLocked(m)
		if claim == nil {
			return ClockSample{}, errProviderIntegration
		}
	}
	return s.clock.sampleForHistory(s.history, claim)
}

func (c *providerIntegration) attachMeter(reservation providerOperationReservation) (*providerWorkMeter, error) {
	if !c.legacyAllowed() {
		return nil, errProviderIntegration
	}
	parent := c.store.activeMeter.Load()
	if parent != nil && (parent.integration != c || parent.authority || parent.closed || parent.failed || parent.parent != nil) {
		return nil, errProviderIntegration
	}
	m := newProviderWorkMeter(c, reservation)
	m.parent = parent
	c.store.activeMeter.Store(m)
	return m, nil
}

func (c *providerIntegration) restoreMeter(m *providerWorkMeter) {
	if c == nil || c.store == nil || c.store.config.bounds.history != nil || c.store.history != nil {
		return
	}
	if m == nil || m.integration != c {
		c.failed = true
		return
	}
	active := c.store.activeMeter.Load()
	if active != nil && active != m && (!active.authority || active.integration != c || active.ticket != m.ticket) {
		c.failed = true
		return
	}
	c.store.activeMeter.Store(m.parent)
}

func (c *providerIntegration) beginRemaining(permit *providerIntegrationPermit) error {
	if !c.legacyAllowed() || permit == nil || permit.owner != c || !permit.used || permit.meter == nil || permit.meter.integration != c || permit.meter.ticket != permit.ticket.digest || c.store.activeMeter.Load() != permit.meter {
		return errProviderIntegration
	}
	m := newProviderWorkMeter(c, providerOperationReservation{work: permit.ticket.work, execution: permit.meter.remaining})
	m.ticket, m.parent, m.authority = permit.ticket.digest, permit.meter.parent, true
	c.store.activeMeter.Store(m)
	return nil
}
