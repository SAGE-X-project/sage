package registry011

// One already-metered native clock invocation, never a request/ACK/Source or
// reusable budget. Only the selected store sampler constructs this original.
type providerHistoryClockClaim struct {
	owner        *providerHistoryClockClaim
	parent       *providerHistoryProgramOwner
	meter        *providerWorkMeter
	expectedRead uint64
	used         bool
}

func (*providerHistoryClockClaim) UnmarshalJSON([]byte) error { return errProviderOwnedClock }

// Called under store.mu, before the clock event. Each accepted observer is
// either the exact pre-H0 O, an explicitly matched inspection phase, the same
// F observer, or the already reserved fixed first-P observer. Idle/DONE is not
// a role; future group execution must add its own closed admitted role.
func (p *providerHistoryProgramOwner) clockRoleLocked(m *providerWorkMeter) bool {
	if p == nil || p.owner != p || m == nil || m.owner != m || p.store == nil || p.store.activeMeter.Load() != m || m.guardian != nil || m.continuing != nil {
		return false
	}
	s := p.store
	if m == p.openMeter {
		if m.scan != p.openControl || m.integration != nil || m.scanBody != nil || m.firstAdmission != nil || m.firstOperation != nil || m.upper != p.config.bounds.setupWork || m.maximumClocks != p.config.bounds.setupMaximumClockReads || p.startupClaimed {
			return false
		}
		o := s.bootstrapOwner
		return o == nil && s.sequence == 0 || o != nil && o.store == s && o.meter == m && o.started && !o.done && !o.failed && s.sequence <= 1
	}
	if m.firstOperation != nil {
		o := s.firstOperation
		return p.startupClaimed && o != nil && o.owner == o && m.firstOperation == o && o.integration == m.integration && o.store == s && o.state >= providerFirstOperationPrepared && o.state < providerFirstOperationDone && (m == o.execution || m == o.remaining)
	}
	c := s.scanControl
	if c == nil || c != p.inspectionControl || c.owner != c || c.store != s || c.held != p.inspection || m.scan != c || m.integration != nil || m.firstOperation != nil {
		return false
	}
	if m == p.preparationMeter {
		return p.inspectionPhase == 1 && c.failed && m.scanBody == nil && m.firstAdmission == nil && m.upper == p.preparation.work && m.maximumClocks == p.preparation.clocks
	}
	if m != p.bodyMeter || p.inspectionPhase != 2 || m.upper != p.body.work || m.maximumClocks != p.body.clocks {
		return false
	}
	if m.firstAdmission != nil {
		f := s.firstAdmissionOwner
		return f != nil && m.firstAdmission == f && f.store == s && f.meter == m && f.state == providerFirstAdmissionStarted
	}
	b := s.scanBody
	return b != nil && b.owner == b && b.active && !b.failed && b.meter == m && b.control == c && m.scanBody == b
}

func (p *providerHistoryProgramOwner) issueClockClaimLocked(m *providerWorkMeter) *providerHistoryClockClaim {
	if !p.matchesLocked() || !p.clockRoleLocked(m) {
		return nil
	}
	m.mu.Lock()
	valid := !m.failed && !m.closed && m.clocks > m.historyClockClaims && m.clocks <= m.maximumClocks
	if valid {
		m.historyClockClaims = m.clocks // No event may mint two native calls.
	}
	m.mu.Unlock()
	if !valid {
		return nil
	}
	c := p.clock
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.history != p || c.historyRevoked || c.failed || c.read >= p.clockCeiling || p.clockClaim != nil && !p.clockClaim.used {
		return nil
	}
	claim := &providerHistoryClockClaim{parent: p, meter: m, expectedRead: c.read}
	claim.owner = claim
	p.clockClaim = claim
	return claim
}

// These bindings are made only at the two closed inspection constructors,
// after their irreversible combined P+B claim. They cannot renew either cap.
func (p *providerHistoryProgramOwner) bindInspectionMeterLocked(m *providerWorkMeter, body bool) bool {
	if !p.matchesLocked() || m == nil || m.owner != m || p.inspectionControl == nil || m.scan != p.inspectionControl || m.integration != nil || m.firstOperation != nil || m.firstAdmission != nil {
		return false
	}
	if !body {
		if p.inspectionPhase != 0 || p.preparationMeter != nil || m.scanBody != nil || m.upper != p.preparation.work || m.maximumClocks != p.preparation.clocks {
			return false
		}
		p.preparationMeter, p.inspectionPhase = m, 1
		return true
	}
	b := p.store.scanBody
	if p.inspectionPhase != 1 || p.bodyMeter != nil || b == nil || b.owner != b || b.control != p.inspectionControl || b.meter != m || m.scanBody != b || m.upper != p.body.work || m.maximumClocks != p.body.clocks {
		return false
	}
	p.bodyMeter, p.inspectionPhase = m, 2
	return true
}
