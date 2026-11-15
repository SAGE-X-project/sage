package registry011

import (
	"context"
	"crypto/tls"
	"errors"
	"testing"
)

func TestProviderHistoryCallerRefusesAnUnusableClockBeforeOpeningRoots(t *testing.T) {
	config, clock := historyCallerFreshConfiguration(t)
	// The fresh backend has not observed an original. Its existing failed
	// state must be refused by open, without reaching the production dial.
	clock.mu.Lock()
	clock.failed = true
	clock.mu.Unlock()
	got, err := runProviderHistoryCandidate(context.Background(), config, clock)
	if err == nil || got != (providerHistoryFinalData{}) || clock.read != 0 || clock.history != nil {
		t.Fatal("unusable fresh backend opened originals or produced DATA", err)
	}
}

func TestProviderHistoryCallerClosesOriginalsAfterInsufficientInitialPreparation(t *testing.T) {
	for _, mode := range []string{"enrollment-entries", "inspection-capacity", "inspection-chunks"} {
		t.Run(mode, func(t *testing.T) {
			config, clock := historyCallerFreshConfigurationWithRefusal(t, mode)
			connection := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, providerFirstOperationResponse())
			bundle := &providerHistoryOwnedConnections{first: connection, members: make([]*tls.Conn, int(config.bounds.history.collection.counts.requests))}
			for i := range bundle.members {
				bundle.members[i] = connection // No exchange is reached by this subject.
			}
			got, err := runProviderHistoryWithOwnedConnectionsCandidate(context.Background(), config, clock, bundle)
			p := clock.history
			if err == nil || got != (providerHistoryFinalData{}) || p == nil || p.store == nil || p.group != nil ||
				p.firstOwner != nil || p.startupClaimed || !p.failed || !clock.historyRevoked || !p.store.closed ||
				p.store.stored != p.held.retained || p.store.activeMeter.Load() != nil {
				t.Fatal("initial refusal leaked DATA, reached transport or renewed the original hold", mode, err)
			}
			s := p.store
			switch mode {
			case "enrollment-entries":
				if s.bootstrapOwner == nil || !s.bootstrapOwner.failed || s.bootstrapOwner.done || s.sequence != 0 || s.scanControl.prepared != nil {
					t.Fatal("enrollment refusal did not stop at its original seven-object requirement")
				}
			case "inspection-capacity":
				if !s.bootstrapOwner.done || s.sequence != 1 || s.scanControl.prepared != nil || p.inspection != (providerScanUpper{}) {
					t.Fatal("unfunded initial claim consumed inspection or passed E0")
				}
			case "inspection-chunks":
				if !s.bootstrapOwner.done || s.sequence != 1 || s.scanControl.prepared == nil || p.preparationMeter == nil ||
					p.bodyMeter != nil || s.scanBody != nil || p.inspection == (providerScanUpper{}) {
					t.Fatal("insufficient two-record inspection did not preserve the completed original preparation")
				}
			}
			reads, stored := clock.read, s.stored
			historyFinalDataRootsReleased(t, s)
			if clock.read != reads || s.stored != stored {
				t.Fatal("checking terminal cleanup repeated an original observation")
			}
		})
	}
}

func TestProviderHistoryCallerCleanupRefusesMissingInputsWithoutLosingTheOriginalError(t *testing.T) {
	original := errors.New("original child refusal")
	var missing context.Context
	for _, tc := range []struct {
		store *providerCustodyStore
		ctx   context.Context
	}{{nil, context.Background()}, {&providerCustodyStore{}, missing}} {
		err := closeProviderHistoryCallCandidate(tc.store, tc.ctx, original)
		if !errors.Is(err, original) || !errors.Is(err, errProviderIntegration) || tc.store != nil && tc.store.closed {
			t.Fatal("invalid terminal inputs hid the child error or entered a close", err)
		}
	}
}
