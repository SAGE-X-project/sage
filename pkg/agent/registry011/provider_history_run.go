package registry011

import (
	"context"
	"crypto/tls"
	"errors"
)

// One private owning invocation, never a derivation factory. The selected
// compiler must already have closed all source/native/resource obligations;
// an unminted numeric profile refuses before open. Only detached DATA leaves
// the lifetime, after final verification and original-root cleanup.
func runProviderHistoryCandidate(ctx context.Context, config providerCustodyConfig, clock *providerOwnedClock) (providerHistoryFinalData, error) {
	return runProviderHistorySelectedCandidate(ctx, config, clock, nil)
}

// Package-owned normal TLS fixtures only. Production entry never accepts a
// bundle, connection list, callback or alternate dial. An absent fixture slot
// cannot turn into the production network path.
type providerHistoryOwnedConnections struct {
	first   *tls.Conn
	members []*tls.Conn
}

func runProviderHistoryWithOwnedConnectionsCandidate(ctx context.Context, config providerCustodyConfig, clock *providerOwnedClock, owned *providerHistoryOwnedConnections) (providerHistoryFinalData, error) {
	if owned == nil {
		return providerHistoryFinalData{}, errProviderIntegration
	}
	return runProviderHistorySelectedCandidate(ctx, config, clock, owned)
}

func runProviderHistorySelectedCandidate(ctx context.Context, config providerCustodyConfig, clock *providerOwnedClock, owned *providerHistoryOwnedConnections) (data providerHistoryFinalData, err error) {
	h := config.bounds.history
	if ctx == nil || ctx.Err() != nil || clock == nil || config.attempt != 0 || config.bounds.guardian != nil ||
		h == nil || h.derivation == nil || h.collection == nil || h.final == (providerHistoryFinalBounds{}) ||
		!h.collection.validSnapshot() || config.capture != h.collection.input.capture || !config.bounds.valid() {
		return data, errProviderIntegration
	}
	if owned != nil {
		if owned.first == nil || uint64(len(owned.members)) != h.collection.counts.requests {
			return data, errProviderIntegration
		}
		for _, connection := range owned.members {
			if connection == nil {
				return data, errProviderIntegration
			}
		}
	}
	s, err := openProviderFrontierCandidate(config, clock)
	if err != nil || s == nil {
		return data, errors.Join(errProviderIntegration, err)
	}
	defer func() {
		err = closeProviderHistoryCallCandidate(s, ctx, err)
		if err != nil {
			data = providerHistoryFinalData{}
		}
	}()
	if _, err = s.enrollBeforeScan(); err != nil {
		return data, err
	}
	target, err := s.scanControl.claimInitialTargetCandidate()
	if err != nil {
		return data, err
	}
	if _, err = s.scanControl.inspectBeforeAdmission(target); err != nil {
		return data, err
	}
	first, err := newProviderFirstOperationCandidate(s)
	if err != nil || first == nil {
		return data, errors.Join(errProviderIntegration, err)
	}
	if owned == nil {
		if result, firstErr := first.executeCandidate(ctx); firstErr != nil || result == nil {
			return data, errors.Join(errProviderIntegration, firstErr)
		}
	} else {
		connection := owned.first
		if connection == nil {
			return data, errProviderIntegration
		}
		if result, firstErr := first.executeCandidateWithConnection(ctx, connection); firstErr != nil || result == nil {
			return data, errors.Join(errProviderIntegration, firstErr)
		}
	}
	var result *providerHistoryFinalResult
	if owned == nil {
		result, err = s.history.runVerifiedCollectionCandidate(ctx)
	} else {
		result, err = s.history.runVerifiedCollectionWithConnectionsCandidate(ctx, owned.members)
	}
	if err != nil || result == nil {
		return data, errors.Join(errProviderIntegration, err)
	}
	return result.closeAndTakeDataCandidate(ctx)
}

// This is only the creator's terminal cleanup, never result verification.
// closeAndTake also closes on its post-verification error paths. Inspect the
// actual state after child locks unwind, so a completed close is not retried.
func closeProviderHistoryCallCandidate(s *providerCustodyStore, ctx context.Context, previous error) error {
	if s == nil || ctx == nil {
		return errors.Join(errProviderIntegration, previous)
	}
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if !closed {
		previous = errors.Join(previous, s.close())
	}
	return errors.Join(previous, ctx.Err())
}
