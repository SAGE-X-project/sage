package registry011

import (
	"context"
	"crypto/tls"
)

// runCollectionCandidate connects the original T/C/member/S owners without
// accepting a caller's request list, upper, replacement clock or retry policy.
// Entry still requires the genuine completed first-operation bridge and the
// complete pre-H0 production derivation. This function does not manufacture
// that derivation or make an unfinished numeric profile executable.
//
// The final S remains private collection DATA. A successful return does not
// perform final independent inspection, publish a page, enroll a Source or
// authorize an Agent message. Their separate original roles and complete
// costs must be connected before a public constructor may use this path.
func (p *providerHistoryProgramOwner) runCollectionCandidate(ctx context.Context) (*providerHistorySettledResults, error) {
	return p.runCollectionWithConnectionsCandidate(ctx, nil)
}

// Only package-owned normal TLS fixtures supply connections, just as for the
// original first-operation/member fixtures. Production entry always passes
// nil and uses the original fixed dial. No exported constructor accepts this
// input, and a missing fixture connection never falls back to a real dial.
func (p *providerHistoryProgramOwner) runCollectionWithConnectionsCandidate(ctx context.Context, owned []*tls.Conn) (*providerHistorySettledResults, error) {
	if ctx == nil || ctx.Err() != nil || p == nil || p.store == nil {
		return nil, errProviderIntegration
	}
	s := p.store
	s.mu.Lock()
	ready := p.owner == p && s.history == p && p.group == nil &&
		p.bounds.collection != nil && p.startReadyLocked()
	maximum := p.bounds.stages
	requests := p.collectionCounts.requests
	s.mu.Unlock()
	if !ready || maximum == 0 || maximum > (1<<20)/3 || owned != nil && uint64(len(owned)) != requests {
		return nil, errProviderIntegration
	}

	group, err := newProviderHistoryGroupOwnerCandidate(s)
	if err != nil {
		return nil, errProviderIntegration
	}
	completed := false
	defer func() {
		if !completed {
			// Preserve the original prefix, debit, clock and recovery owner.
			// No cleanup I/O or replacement context creates a later failed S.
			s.mu.Lock()
			group.failLocked()
			s.mu.Unlock()
		}
	}()

	nextConnection := 0
	for stage := uint64(1); stage <= maximum; stage++ {
		admission, err := group.retainCollectionAdmissionCandidate(ctx)
		if err != nil {
			return nil, errProviderIntegration
		}
		consumption, err := group.retainConsumptionCandidate(ctx, admission)
		if err != nil {
			return nil, errProviderIntegration
		}
		dispatch, err := group.prepareDispatchCandidate(ctx, consumption)
		if err != nil {
			return nil, errProviderIntegration
		}
		s.mu.Lock()
		count := 0
		if dispatch != nil && dispatch.owner == dispatch && dispatch.group == group &&
			group.dispatch == dispatch && group.order != nil && dispatch.order == group.order &&
			dispatch.retained == 0 && dispatch.order.valid() {
			count = dispatch.order.count
		}
		s.mu.Unlock()
		if count < 1 || count > providerHistoryGroupMembers {
			return nil, errProviderIntegration
		}
		for ordinal := range count {
			// The member owner alone dials and handles known failed captures.
			// An error is terminal here, even if that owner retained failed S.
			var memberErr error
			if owned == nil {
				_, memberErr = group.retainMemberCandidate(ctx, dispatch, ordinal)
			} else {
				if nextConnection >= len(owned) || owned[nextConnection] == nil {
					return nil, errProviderIntegration
				}
				_, memberErr = group.retainMemberCandidateWithConnection(ctx, dispatch, ordinal, owned[nextConnection])
				nextConnection++
			}
			if memberErr != nil {
				return nil, errProviderIntegration
			}
		}
		settled, err := group.retainSettlementCandidate(ctx, dispatch)
		if err != nil {
			return nil, errProviderIntegration
		}
		s.mu.Lock()
		done, valid := group.collectionSettlementStateLocked(settled, stage)
		s.mu.Unlock()
		if !valid || ctx.Err() != nil {
			return nil, errProviderIntegration
		}
		if done {
			if owned != nil && nextConnection != len(owned) {
				return nil, errProviderIntegration
			}
			completed = true
			return settled, nil
		}
		if stage == maximum || group.advanceStageCandidate(ctx, settled.ack) != nil {
			return nil, errProviderIntegration
		}
	}
	return nil, errProviderIntegration
}

// Scalar navigation after the same S's two original reconstructions. No
// encoding, sampling, filesystem observation or new allowance occurs here.
// A matching private terminal cursor selects only the next internal role;
// it never substitutes for final original verification or Source readiness.
func (o *providerHistoryGroupOwner) collectionSettlementStateLocked(settled *providerHistorySettledResults, stage uint64) (bool, bool) {
	if o == nil || o.store == nil || o.parent == nil || o.parent.bounds.collection == nil ||
		settled == nil || settled.owner != settled || settled.group != o || settled.ack != o.last ||
		o.phase != "settled" || o.failed || stage == 0 || stage > o.parent.bounds.stages ||
		!o.parent.matchesLocked() || !o.successfulSettlementShapeLocked() ||
		o.store.activeMeter.Load() != nil || o.coordinator.stage != stage || o.custodian.stage != stage {
		return false, false
	}
	left, right := o.coordinator.collection, o.custodian.collection
	if left == nil || right == nil || left == right || !left.valid() || !right.valid() ||
		left.semantic == right.semantic || left.selection != right.selection ||
		left.selection.program != o.parent.bounds.program ||
		left.selection.manifest != o.parent.collectionDigest ||
		left.completed != stage || right.completed != stage || left.phase != right.phase {
		return false, false
	}
	done := left.phase == "done"
	if done && (!left.semantic.atEnd() || !right.semantic.atEnd()) {
		return false, false
	}
	return done, true
}
