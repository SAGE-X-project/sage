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
	if !s.event(providerClockEvent, 0) {
		return ClockSample{}, errProviderIntegration
	}
	return s.clock.sample()
}

func (c *providerIntegration) attachMeter(reservation providerOperationReservation) (*providerWorkMeter, error) {
	if !c.valid() {
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
	if permit == nil || permit.owner != c || !permit.used || permit.meter == nil || permit.meter.integration != c || permit.meter.ticket != permit.ticket.digest || c.store.activeMeter.Load() != permit.meter {
		return errProviderIntegration
	}
	m := newProviderWorkMeter(c, providerOperationReservation{work: permit.ticket.work, execution: permit.meter.remaining})
	m.ticket, m.parent, m.authority = permit.ticket.digest, permit.meter.parent, true
	c.store.activeMeter.Store(m)
	return nil
}
