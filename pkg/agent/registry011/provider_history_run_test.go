package registry011

import (
	"context"
	"crypto/tls"
	"errors"
	"testing"
)

func TestProviderHistoryCallerRefusesMissingContextOrUnmintedSelection(t *testing.T) {
	var missing context.Context
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, ctx := range []context.Context{missing, cancelled, context.Background()} {
		clock := &providerOwnedClock{}
		config := providerCustodyConfig{bounds: providerIntegrationBounds{history: &providerHistoryProgramBounds{}}}
		got, err := runProviderHistoryCandidate(ctx, config, clock)
		if err == nil || got != (providerHistoryFinalData{}) || clock.read != 0 || clock.history != nil {
			t.Fatal("unminted entry observed a clock or produced verified DATA", err)
		}
	}
}

func TestProviderHistoryOwnedCallerRejectsMissingSlotsBeforeOpen(t *testing.T) {
	config, clock := historyCallerFreshConfiguration(t)
	connection := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, providerFirstOperationResponse())
	for _, mode := range []string{"nil-bundle", "nil-first", "wrong-count", "nil-member"} {
		t.Run(mode, func(t *testing.T) {
			bundle := &providerHistoryOwnedConnections{first: connection}
			switch mode {
			case "nil-bundle":
				bundle = nil
			case "nil-first":
				bundle.first = nil
			case "nil-member":
				bundle.members = make([]*tls.Conn, int(config.bounds.history.collection.counts.requests))
				for i := range bundle.members {
					bundle.members[i] = connection // Never used by this refusal subject.
				}
				bundle.members[len(bundle.members)-1] = nil
			}
			got, err := runProviderHistoryWithOwnedConnectionsCandidate(context.Background(), config, clock, bundle)
			if err == nil || got != (providerHistoryFinalData{}) || clock.read != 0 || clock.history != nil {
				t.Fatal("missing fixture slot opened originals or fell through to a real dial", err)
			}
		})
	}
}

func TestProviderHistoryCallerCleanupDoesNotCloseAnOriginalTwice(t *testing.T) {
	s := historyCallerEnrollmentPrefix(t)
	if err := s.close(); err != nil {
		t.Fatal("original successful close precursor", err)
	}
	reads, held := s.clock.read, s.stored
	if err := closeProviderHistoryCallCandidate(s, context.Background(), nil); err != nil {
		t.Fatal("terminal defer converted a completed close into failure", err)
	}
	original := errors.New("original child refusal")
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := closeProviderHistoryCallCandidate(s, cancelled, original); !errors.Is(err, original) || !errors.Is(err, context.Canceled) {
		t.Fatal("terminal cleanup replaced the child or cancellation error", err)
	}
	if s.clock.read != reads || s.stored != held || !s.closed || !s.history.failed || !s.clock.historyRevoked {
		t.Fatal("completed cleanup renewed original observations or allowance")
	}
	historyFinalDataRootsReleased(t, s)
}

func TestProviderHistoryCallerCleanupRetainsOwnedCloseFailure(t *testing.T) {
	s := historyCallerEnrollmentPrefix(t)
	if err := s.evidence.lock.Close(); err != nil {
		t.Fatal("close only this fixture's owned descriptor", err)
	}
	original := errors.New("original child refusal")
	if err := closeProviderHistoryCallCandidate(s, context.Background(), original); err == nil || !errors.Is(err, original) || err.Error() == original.Error() || !s.closed {
		t.Fatal("owned close failure was discarded or the store stayed live", err)
	}
	historyFinalDataRootsReleased(t, s)
}

func TestProviderHistoryCallerFirstConnectionFailureClosesBothRoots(t *testing.T) {
	config, clock := historyCallerFreshConfiguration(t)
	connection := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, providerFirstOperationResponse())
	if err := connection.Close(); err != nil {
		t.Fatal("ordinary owned closed-connection precursor", err)
	}
	bundle := &providerHistoryOwnedConnections{first: connection, members: make([]*tls.Conn, int(config.bounds.history.collection.counts.requests))}
	for i := range bundle.members {
		bundle.members[i] = connection // The first exchange fails before these slots.
	}
	got, err := runProviderHistoryWithOwnedConnectionsCandidate(context.Background(), config, clock, bundle)
	p := clock.history
	if err == nil || got != (providerHistoryFinalData{}) || p == nil || p.firstOwner == nil ||
		p.firstOwner.state != providerFirstOperationFailed || !p.firstOwner.consumption ||
		!p.store.closed || !p.failed || !clock.historyRevoked || p.store.stored != p.held.retained || p.store.activeMeter.Load() != nil {
		t.Fatal("entered first-connection failure leaked DATA, roots or a renewed hold", err)
	}
	historyFinalDataRootsReleased(t, p.store)
}
