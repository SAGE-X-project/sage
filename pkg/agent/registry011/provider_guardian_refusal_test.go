package registry011

import (
	"context"
	"testing"
)

func providerGuardianSettledFixture(t *testing.T) (*providerCustodyStore, *providerGuardianOwner) {
	t.Helper()
	s, o := providerGuardianFirstFixture(t)
	conn := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, providerFirstOperationResponse())
	if _, err := o.executeCandidateWithConnection(context.Background(), conn); err != nil {
		t.Fatal("ordinary owned first settlement", err)
	}
	return s, s.guardian
}

// Finite inert state and quota units. No runtime adversarial peer or host
// bypass is involved; each candidate is an isolated owned test namespace.
func TestProviderGuardianCannotStartFromMissingClosedOrUnselectedState(t *testing.T) {
	for _, mode := range []string{"missing", "copied", "purpose", "not-ready", "active", "attempts", "first", "main", "last", "pending", "budget", "page", "backend", "work", "retained", "clocks", "records", "objects", "raw", "metadata"} {
		t.Run(mode, func(t *testing.T) {
			s, g := providerGuardianSettledFixture(t)
			s.mu.Lock()
			defer s.mu.Unlock()
			candidate, purpose := g, "bootstrap-admission"
			var copied providerGuardianOwner
			switch mode {
			case "missing":
				candidate = nil
			case "copied":
				copied = *g
				candidate = &copied
			case "purpose":
				purpose = "unknown"
			case "not-ready":
				g.ready = false
			case "active":
				g.active = true
			case "attempts":
				g.ordinal = g.bounds.attempts
			case "first":
				g.first = nil
			case "main":
				g.main.commit = admissionTestDigest("f")
			case "last":
				s.last = &providerCustodyAcknowledgment{}
			case "pending":
				s.frontierFiles.pending = &providerFrontierWriteClaim{}
			case "budget":
				g.integration.ledger.current = nil
			case "page":
				g.integration.ledger.continuation.pending = &providerLedgerBoundary{}
			case "backend":
				s.clock.mu.Lock()
				s.clock.read = s.clock.maximum
				s.clock.mu.Unlock()
			case "work":
				g.bounds.lifetime.work[2] = 1
			case "retained":
				g.bounds.lifetime.retained = 1
			case "clocks":
				g.bounds.lifetime.clocks = 1
			case "records":
				g.bounds.lifetime.records = 1
			case "objects":
				g.bounds.lifetime.objects = 1
			case "raw":
				g.bounds.lifetime.raw = 1
			case "metadata":
				g.bounds.lifetime.metadata = 1
			}
			stored, raw, sequence, clockReads := s.stored, s.retainedRaw, s.sequence, s.clock.read
			if candidate.startClaimLocked(purpose) == nil || s.stored != stored || s.retainedRaw != raw || s.sequence != sequence || s.clock.read != clockReads || s.activeMeter.Load() != nil {
				t.Fatal("refusal spent a root/sample or restored a guardian", mode)
			}
		})
	}
}

func TestProviderGuardianClaimQuotaCannotRefundItsOriginalLifetime(t *testing.T) {
	s, g := providerGuardianSettledFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	if !g.claimRoomLocked(1) || g.claimRoomLocked(g.bounds.attempts+1) {
		t.Fatal("finite lifetime claim arithmetic")
	}
	stored := s.stored
	if g.startClaimLocked("bootstrap-admission") != nil {
		t.Fatal("ordinary metadata start")
	}
	if physical, raw, ok := g.claimStorageLocked([]byte(`{}`), []byte(`{}`), []byte(`{}`), []byte(`{}`)); !ok || raw != 10 || physical != raw+5*s.config.bounds.fileOverheadBytes {
		t.Fatal("pending copy missing from physical upper")
	}
	for _, mode := range []string{"empty", "claim-physical", "lifetime-physical", "claim-raw", "lifetime-raw", "claim-metadata", "lifetime-metadata", "stored", "overflow"} {
		t.Run(mode, func(t *testing.T) {
			bounds, physical, raw, metadata, retained := g.bounds, g.physical, g.raw, g.metadata, s.retainedRaw
			defer func() {
				g.bounds, g.physical, g.raw, g.metadata, s.retainedRaw = bounds, physical, raw, metadata, retained
			}()
			input := []byte(`{}`)
			switch mode {
			case "empty":
				input = nil
			case "claim-physical":
				g.bounds.claim.retained = 1
			case "lifetime-physical":
				g.physical = g.bounds.lifetime.retained
			case "claim-raw":
				g.bounds.claim.raw = 1
			case "lifetime-raw":
				g.raw = g.bounds.lifetime.raw
			case "claim-metadata":
				g.bounds.claim.metadata = 1
			case "lifetime-metadata":
				g.metadata = g.bounds.lifetime.metadata
			case "stored":
				s.retainedRaw = s.stored
			case "overflow":
				g.physical = ^uint64(0)
			}
			if _, _, ok := g.claimStorageLocked(input, input, input, input); ok || s.stored != stored {
				t.Fatal("failed scalar quote reset an upper", mode)
			}
		})
	}
	// Exhausted metadata work fails before any immutable claim file is written.
	g.meter.upper[2] = 0
	if g.retainClaimLocked(g.integration.firstExchange) == nil || !g.failed || !s.failed || !g.meter.closed || s.activeMeter.Load() != nil || s.stored != stored || g.ordinal != 1 || s.sequence != 4 {
		t.Fatal("failed consumed claim was refunded or left dispatchable")
	}
	if g.startClaimLocked("bootstrap-admission") == nil {
		t.Fatal("failed metadata claim resumed")
	}
}

func TestProviderGuardianOriginalsRequireExactPriorMainPurposeAndTarget(t *testing.T) {
	s, g := providerGuardianSettledFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	if g.startClaimLocked("bootstrap-admission") != nil || g.retainClaimLocked(g.integration.firstExchange) != nil {
		t.Fatal("ordinary retained G metadata")
	}
	for _, mode := range []string{"prior", "main", "admission", "target", "purpose", "clock", "nil-meter"} {
		t.Run(mode, func(t *testing.T) {
			before, main, admission, target, purpose, meter := g.before, g.main, g.integration.intent.digest, g.integration.firstExchange, "bootstrap-admission", g.meter
			switch mode {
			case "prior":
				before.origin = admissionTestDigest("f")
			case "main":
				main.commit = admissionTestDigest("f")
			case "admission":
				admission = admissionTestDigest("f")
			case "target":
				target = admissionTestDigest("f")
			case "purpose":
				purpose = "ledger-page-seal"
			case "clock":
				before.clock.Epoch = admissionTestDigest("f")
			case "nil-meter":
				meter = nil
			}
			if data, err := providerGuardianClaimOriginals(s, meter, before, main, admission, target, purpose); err == nil || data != nil {
				t.Fatal("crossed original supplied a claim", mode)
			}
		})
	}
}
