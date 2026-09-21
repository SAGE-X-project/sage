package guard010

import (
	"context"
	"time"
)

// mcpProtectedReply is retained only by the authenticated owner. It is never a
// caller-supplied receipt, routing hint or transferable authorization token.
type mcpProtectedReply struct {
	owner            *mcpSetupSession
	gate             *mcpAdmissionGate
	config           *mcpAdmissionConfig
	receipt          *DispatchReceipt
	generation       uint64
	start, deadline  time.Duration
	innerID, outerID string
	intent           []byte
	used             bool
}

// replyProtected sends at most one protected response for the current authenticated
// invocation. Full local handoff does not assert peer acceptance or tool success.
// A lost response never rolls back execution or authorizes another execution.
func (s *mcpSetupSession) replyProtected(ctx context.Context, io mcpSetupIO) (err error) {
	if ctx == nil || io == nil || s.admission == nil || s.session.Initiator() {
		return ErrInvalid
	}
	g := s.admission
	s.mu.Lock()
	if s.running || s.closed {
		s.mu.Unlock()
		s.close()
		return ErrInvalid
	}
	work, cancel := context.WithCancel(ctx)
	s.running, s.cancel = true, cancel
	s.mu.Unlock()
	var p *mcpOutput
	defer func() {
		if recover() != nil {
			err = ErrInvalid
		}
		cancel()
		s.mu.Lock()
		s.running = false
		cleanup := s.closed || err != nil
		if cleanup {
			s.closed = true
		}
		s.mu.Unlock()
		if cleanup {
			s.owner.close()
			s.session.Close()
		}
		g.mu.Lock()
		for n, pending := range g.outputs {
			if p != nil && pending == p {
				g.outputs[n] = nil
			}
		}
		g.mu.Unlock()
	}()
	g.mu.Lock()
	r := s.owner.response
	valid := r != nil && r.owner == s && r.gate == g && !r.used && r.generation == g.generation && s.owner.phase == mcpReady && s.owner.pending == nil && s.owner.deferred == nil && s.owner.validLocked(true)
	stamp, clockErr := g.sampleLocked()
	now := time.Duration(stamp.MonoMS) * time.Millisecond
	slot := -1
	for n, output := range g.outputs {
		if output == nil {
			slot = n
			break
		}
	}
	if !valid || clockErr != nil || g.retired || now < s.owner.last || now >= r.deadline || slot < 0 || work.Err() != nil {
		g.mu.Unlock()
		return ErrInvalid
	}
	r.used = true
	bounded, deadlineCancel := context.WithTimeout(work, r.deadline-now)
	p = &mcpOutput{owner: s.owner, next: mcpReady, deadline: r.deadline, response: r, cancel: deadlineCancel}
	s.owner.pending = p
	g.outputs[slot] = p
	g.mu.Unlock()
	defer deadlineCancel()
	raw, err := g.ledger.Reply(bounded, r.receipt, r.config.signer)
	g.ledger.mu.Lock()
	retired := g.ledger.retired
	g.ledger.mu.Unlock()
	if retired {
		g.poison()
	}
	if err != nil || retired {
		return ErrInvalid
	}
	v, err := VerifyResult(bounded, raw, r.config.resultAuthority, acceptedInvocation(r.intent))
	if err != nil {
		return ErrInvalid
	}
	body, err := v.MCPResult(MCPVersion)
	if err != nil {
		return ErrInvalid
	}
	rpc := mcpRPCResponse(r.innerID, body)
	local, peer, err := s.session.Participants()
	if err != nil {
		return ErrInvalid
	}
	success, code, err := sessionResult(MCPVersion, r.innerID, rpc, r.intent, local, peer)
	if err != nil {
		return ErrInvalid
	}
	env, _, err := object(raw)
	if err != nil {
		return ErrInvalid
	}
	result, ok := env["result"].(map[string]any)
	if !ok {
		return ErrInvalid
	}
	expires, ok := number(result, "expires")
	if !ok {
		return ErrInvalid
	}
	// Only the adapter writes this output before publishing it to the sender.
	g.mu.Lock()
	p.expires = expires
	g.mu.Unlock()
	wire, err := s.session.SealResponse(bounded, r.outerID, rpc, success, code, 30)
	if err != nil || len(wire) > 32768 || wireField(wire, "id") == r.innerID {
		return ErrInvalid
	}
	if err = io.Send(bounded, append([]byte(nil), wire...)); err != nil {
		return ErrInvalid
	}
	// Storage/signing and transport may take time: revalidate the exact signed
	// result and the complete session after handoff, then publish under coordination.
	g.mu.Lock()
	observationStart, observationErr := g.sampleLocked()
	p.resultObserved = time.Duration(observationStart.MonoMS) * time.Millisecond
	g.mu.Unlock()
	if observationErr != nil {
		return ErrInvalid
	}
	if _, err = VerifyResult(bounded, raw, r.config.resultAuthority, acceptedInvocation(r.intent)); err != nil {
		return ErrInvalid
	}
	resultObservation, err := r.config.resultAuthority.observe(bounded)
	if err != nil {
		return ErrInvalid
	}
	g.mu.Lock()
	p.resultSample = time.Duration(resultObservation.stamp.MonoMS) * time.Millisecond
	p.resultUnix = resultObservation.stamp.Unix
	if resultObservation.expires != nil {
		expiry := *resultObservation.expires
		p.keyExpires = &expiry
	}
	g.mu.Unlock()
	observed, err := s.session.Observe(bounded)
	if err != nil {
		return ErrInvalid
	}
	return s.owner.completeObserved(p, true, true, &observed, bounded)
}
