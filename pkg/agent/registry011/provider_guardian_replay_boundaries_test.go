package registry011

import "testing"

func TestProviderContinuingPrefixRejectsDifferentConsumptionAndBudgetData(t *testing.T) {
	t.Parallel()
	s, o := providerContinuingPreparedFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.frontierReplay.body.ledger
	for _, mode := range []string{"consumption-shape", "budget", "consumption-digest"} {
		t.Run(mode, func(t *testing.T) {
			switch mode {
			case "consumption-shape":
				before := r.lastConsumption
				defer func() { r.lastConsumption = before }()
				r.lastConsumption = nil
			case "budget":
				before := r.current.clock
				defer func() { r.current.clock = before }()
				r.current.clock.MonotonicMS++
			case "consumption-digest":
				before := r.lastConsumption.digest
				defer func() { r.lastConsumption.digest = before }()
				r.lastConsumption.digest = admissionTestDigest("f")
			}
			if _, ok := providerContinuingPrefixData(o.guardian); ok {
				t.Fatal("different typed DATA produced a continuing prefix", mode)
			}
		})
	}
	if !o.guardian.continuingPrefixMatchesLocked() || s.sequence != 5 || o.permit != nil {
		t.Fatal("finite comparison changed the selected prefix")
	}
}

func TestProviderGuardianTypedReplayRejectsMissingOrDifferentIncomingBindings(t *testing.T) {
	t.Parallel()
	s, o := providerContinuingPreparedFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.frontierReplay.body
	ticket := *o.ticket
	ticket.intent = r.intent
	for _, mode := range []string{"references", "duplicate", "missing", "ticket-kind", "old-main"} {
		t.Run(mode, func(t *testing.T) {
			before, references := *r.guardian, s.newReferences
			defer func() { *r.guardian, s.newReferences = before, references }()
			// Fixed candidate DATA: simulate an attempted later admit. This
			// does not roll back either live model or selected main/G head.
			r.guardian.terminal, r.guardian.head = true, o.priorGuardian
			s.newReferences = append([]providerIntegrationObjectRef(nil), references...)
			selected := ticket
			switch mode {
			case "references":
				s.newReferences = nil
			case "duplicate":
				for _, ref := range references {
					if ref.kind == "guardian-admission-binding" {
						s.newReferences = append(s.newReferences, ref)
					}
				}
			case "missing":
				for i := range s.newReferences {
					if s.newReferences[i].kind == "guardian-admission-binding" {
						s.newReferences[i].digest = admissionTestDigest("f")
					}
				}
			case "ticket-kind":
				selected.kind = "different-unit-kind"
			}
			if r.guardianTicket(s, &selected) == nil {
				t.Fatal("incomplete binding or old-main DATA advanced the G chain", mode)
			}
		})
	}
	if s.sequence != 5 || o.permit != nil || o.execution.actual[0] != 0 || o.execution.actual[1] != 0 || o.execution.actual[9] != 0 {
		t.Fatal("failed typed DATA replay dispatched or wrote")
	}
}

func TestProviderGuardianTypedMainReplayCannotReuseCompletedConsumption(t *testing.T) {
	t.Parallel()
	s, o := providerContinuingPreparedFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.frontierReplay.body
	for _, input := range []struct{ kind, operation string }{
		{"admit", o.ticket.digest},
		{"consume", r.ledger.lastConsumption.digest},
	} {
		if r.apply(s, map[string]any{"new_enrollment_root": s.enrollment, "kind": input.kind, "operation": input.operation}) == nil {
			t.Fatal("completed or currently admitted DATA advanced the live typed ledger", input.kind)
		}
	}
	if s.sequence != 5 || o.consumption || o.permit != nil || !o.guardian.continuingPrefixMatchesLocked() || o.execution.actual[9] != 0 {
		t.Fatal("refused typed replay changed the actual admission")
	}
}
