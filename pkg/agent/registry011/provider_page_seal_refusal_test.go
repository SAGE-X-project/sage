package registry011

import (
	"context"
	"encoding/json"
	"testing"
)

func providerPageSealReadyWithProfileFixture(t *testing.T, decoded, records uint64) (*providerCustodyStore, *providerGuardianOwner) {
	t.Helper()
	s, g := providerContinuingSettledWithProfileFixture(t, true, decoded, records)
	o, err := newProviderContinuingBootstrapCandidate(s)
	if err != nil {
		t.Fatal("normal local admission before page", err)
	}
	sequence, ordinal := s.sequence, g.ordinal
	if value, err := sealProviderContinuingPageCandidate(s); err == nil || value != nil || s.sequence != sequence || g.ordinal != ordinal {
		t.Fatal("page closure bypassed an unfinished bootstrap")
	}
	if _, err := o.executeBootstrapCandidate(context.Background()); err != nil || len(o.integration.ledger.continuation.records) != 6 || !g.ready || s.sequence != 7 {
		t.Fatal("normal local bootstrap before page", err)
	}
	return s, g
}

func TestProviderPageSealKeepsOriginalHoldsOnRefusal(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"small-profile", "partial-page", "active-claim", "prefix-data", "guardian-data", "claim-unavailable"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			decoded, records := uint64(64<<20), uint64(6)
			if mode == "small-profile" {
				decoded = 50 << 20
			}
			if mode == "partial-page" {
				records = 9
			}
			// Every profile is fixed before this fixture's anchor/H0/clock.
			// DATA refusals below neither forge a role nor select a Source.
			s, g := providerPageSealReadyWithProfileFixture(t, decoded, records)
			stored, main, origin, ordinal := s.stored, s.frontierFiles.pin, s.setupOrigin, g.ordinal
			if mode == "active-claim" {
				s.mu.Lock()
				if g.startClaimLocked("rpc-admission") != nil {
					s.mu.Unlock()
					t.Fatal("normal independent metadata claim")
				}
				claim := g.claim
				s.mu.Unlock()
				value, err := sealProviderContinuingPageCandidate(s)
				if err == nil || value != nil || g.ordinal != ordinal+1 || g.claim != claim || g.failed || s.stored != stored || s.frontierFiles.pin != main {
					t.Fatal("page operation consumed or replaced another active claim")
				}
				s.mu.Lock()
				g.failLocked()
				s.mu.Unlock()
				return
			}
			switch mode {
			case "prefix-data":
				g.prefix.budget.digest = admissionTestDigest("f") // Inert mismatching cached DATA.
			case "guardian-data":
				s.frontierReplay.body.guardian.terminal = false // Incomplete old replay DATA.
			case "claim-unavailable":
				if err := s.custody.root.Mkdir(s.name("guardian-intent", ordinal+1), 0700); err != nil {
					t.Fatal("normal owned unavailable metadata path", err)
				}
			}
			value, err := sealProviderContinuingPageCandidate(s)
			if err == nil || value != nil || !g.failed || !s.failed || g.ready || g.ordinal != ordinal+1 ||
				s.stored != stored || s.frontierFiles.pin != main || s.setupOrigin != origin || s.activeMeter.Load() != nil {
				t.Fatal("refused page recovered an original hold or returned usable DATA", mode)
			}
			if value, err := sealProviderContinuingPageCandidate(s); err == nil || value != nil || g.ordinal != ordinal+1 || s.stored != stored {
				t.Fatal("refused metadata role was reused", mode)
			}
		})
	}
}

func TestProviderPageSealCannotReleaseWithoutOriginalDataAndClock(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"model-data", "clock-unavailable"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			s, g := providerPageSealReadyFixture(t)
			s.mu.Lock()
			defer s.mu.Unlock()
			o, err := newProviderPageSealCandidateLocked(s)
			if err != nil {
				t.Fatal("normal closed page preparation", err)
			}
			defer o.failLocked()
			if _, err := o.retainLocked(); err != nil {
				t.Fatal("normal retained independent page", err)
			}
			stored, main := s.stored, s.frontierFiles.pin
			if mode == "model-data" {
				// Finite in-memory disagreement, never a rewritten original.
				s.frontierReplay.body.ledger.current.privateMetadata++
			} else {
				g.clock.mu.Lock()
				g.clock.failed = true
				g.clock.mu.Unlock()
			}
			if value, err := o.releaseLocked(); err == nil || value != nil || !g.failed || !s.failed || !o.closed ||
				g.ready || s.stored != stored || s.frontierFiles.pin != main || s.activeMeter.Load() != nil {
				t.Fatal("uncertain final observation released a page or refunded a hold", mode)
			}
		})
	}
}

func TestProviderPageSealDataCannotRestoreClosedRoles(t *testing.T) {
	t.Parallel()
	for _, value := range []any{&providerPageSealOwner{}, &providerPageSealData{}, &providerPageSealPlan{}, &providerGuardianSealBinding{}} {
		if json.Unmarshal([]byte(`{}`), value) == nil {
			t.Fatal("JSON restored a private page role or receipt")
		}
	}
	if _, ok := providerPageSealMemoryUpper(nil, 0, 0); ok {
		t.Fatal("missing original owner produced a resident quote")
	}
	if value, err := newProviderPageSealCandidateLocked(nil); err == nil || value != nil {
		t.Fatal("missing store produced a page role")
	}
	if _, err := (*providerPageSealOwner)(nil).retainLocked(); err == nil {
		t.Fatal("missing concrete role retained a page")
	}
	if value, err := (*providerPageSealOwner)(nil).releaseLocked(); err == nil || value != nil {
		t.Fatal("missing concrete role released a page")
	}
}
