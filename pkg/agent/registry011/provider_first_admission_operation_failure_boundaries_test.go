package registry011

import (
	"context"
	"crypto/tls"
	"os"
	"path/filepath"
	"testing"
)

// Model cancellation immediately after the first entry check. All subsequent
// checks use the already canceled standard context, before any RPC request.
type providerFirstDialCancelContext struct {
	context.Context
	cancel context.CancelFunc
	checks int
}

func (c *providerFirstDialCancelContext) Err() error {
	c.checks++
	if c.checks == 1 {
		before := c.Context.Err()
		c.cancel()
		return before
	}
	return c.Context.Err()
}

func TestProviderFirstDialCancellationAfterEntryRetainsItsFailureCategory(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	input := &providerFirstDialCancelContext{Context: ctx, cancel: cancel}
	var diagnostic providerFirstOperationDiagnostic
	connection, err := providerIntegrationDialDiagnosed(input, providerTestLimits(), &diagnostic)
	if connection != nil {
		_ = connection.Close()
	}
	// Resolver completion can race with cancellation. Check only the bounded
	// native category here; the pure classifier test checks Canceled exactly.
	if err == nil || connection != nil || input.checks < 2 || ctx.Err() != context.Canceled ||
		diagnostic.Tag != providerFirstFailureDial || diagnostic.Reason == providerFirstReasonRefused || diagnostic.Reason > providerFirstReasonNetwork || diagnostic.Frozen ||
		diagnostic.RequestWritten != 0 || diagnostic.ResponseRead != 0 {
		t.Fatal("post-entry cancellation lost its original dial failure category", err, diagnostic)
	}
}

// The normal test-owned TLS connection closes when the byte primitive
// registers its cancellation cleanup, after SetDeadline and request copying.
// No peer bytes, observer allowance or original clock are modified.
type providerFirstCloseBeforeWriteContext struct {
	context.Context
	connection *tls.Conn
	closed     bool
}

func (c *providerFirstCloseBeforeWriteContext) Done() <-chan struct{} {
	if !c.closed {
		c.closed = true
		_ = c.connection.Close()
	}
	return c.Context.Done()
}

func TestProviderFirstCaptureWriteFailurePreservesThePrecleanupDiagnostic(t *testing.T) {
	request, err := newProviderIntegrationRequest("eth_getBlockByNumber", []any{"finalized", false})
	if err != nil {
		t.Fatal(err)
	}
	limits, certificates := providerTestLimits(), providerTLSBounds{1, 1, 8 << 10}
	meter, upper := providerFirstCaptureMeter(t, request, limits, certificates)
	connection := providerIntegrationOwnedTLS(t, request.body, providerFirstOperationResponse())
	ctx, cancel := context.WithTimeout(context.Background(), limits.Timeout)
	defer cancel()
	input := &providerFirstCloseBeforeWriteContext{Context: ctx, connection: connection}
	var diagnostic providerFirstOperationDiagnostic
	sample, err := providerFirstOperationCaptureDiagnosed(input, connection, limits, request, meter, certificates, &diagnostic)
	if err == nil || sample != nil || !input.closed || !meter.failed || meter.actual[0] != 0 || meter.actual[1] != 0 ||
		meter.actual[10] == 0 || meter.clocks != 0 || meter.authority ||
		diagnostic.Tag != providerFirstFailureWrite || !diagnostic.Frozen || diagnostic.Failed || diagnostic.Closed ||
		diagnostic.RequestWritten != 0 || diagnostic.ResponseRead != 0 {
		t.Fatal("owned connection write refusal lost its pre-cleanup diagnostic or gained an execution role", err, diagnostic)
	}
	for coordinate, actual := range meter.actual {
		if actual > upper[coordinate] {
			t.Fatal("write refusal exceeded its unchanged primitive upper", coordinate)
		}
	}
}

// This is deliberately an unselected partial-method DATA subject. It has no
// store facade, clock, permit, observer or operating owner to borrow.
func TestProviderFirstDetachedFinishKeepsTheFirstClockFailure(t *testing.T) {
	o := &providerFirstOperationOwner{integration: &providerIntegration{}, store: &providerCustodyStore{}}
	success := false
	out, err := o.finishLocked(nil, &ProviderTransportSample{Complete: true}, nil, &success)
	if err == nil || out != nil || success || o.owner != nil || o.permit != nil || o.execution != nil || o.remaining != nil ||
		o.store.firstOperation != nil || o.store.clock != nil || o.store.activeMeter.Load() != nil ||
		o.diagnostic.Tag != providerFirstFailureExecutionClock || !o.diagnostic.Frozen {
		t.Fatal("detached clock failure changed its first diagnostic or created an execution owner", err)
	}
}

func TestProviderFirstBurnRejectsChangedPrivateRequestAfterOriginalConsumption(t *testing.T) {
	s, o := providerFirstOperationFixture(t)
	originalRequest := o.request
	if o.integration.first != originalRequest || originalRequest.wire != providerFirstOperationRequestWire {
		t.Fatal("genuine unchanged original request required")
	}
	reached := false
	syncFile := s.custody.syncFile
	s.custody.syncFile = func(file *os.File) error {
		if err := syncFile(file); err != nil {
			return err
		}
		change := o.change
		if !reached && o.state == providerFirstOperationConsumption && change != nil && change.after.sequence == 3 &&
			filepath.Base(file.Name()) == providerCustodyName("HEAD.pending", change.after.ordinal) {
			// Corrupt only the coordinator's private request pointer. The
			// immutable originals, first request, limits and clock stay fixed.
			copy := *originalRequest
			copy.wire = "X" + copy.wire[1:]
			o.request, reached = &copy, true
		}
		return nil
	}
	connection := providerIntegrationOwnedTLS(t, originalRequest.body, providerFirstOperationResponse())
	before, held := o.execution.actual, s.stored
	out, err := o.executeCandidateWithConnection(context.Background(), connection)
	s.custody.syncFile = syncFile
	s.clock.mu.Lock()
	last, failed, refusal := s.clock.last, s.clock.failed, s.clock.nativeRefusal
	s.clock.mu.Unlock()
	if err == nil || out != nil || !reached || o.permit == nil || !o.permit.used || s.sequence != 3 || !o.consumption || o.settlement ||
		o.state != providerFirstOperationFailed || !o.execution.failed || !o.execution.closed || s.activeMeter.Load() != nil ||
		s.stored != held || o.integration.first != originalRequest || originalRequest.wire != providerFirstOperationRequestWire ||
		failed || refusal != 0 || last == (ClockSample{}) ||
		!candidateLedgerClockAdvance(providerClockDiagnostic(o.permit.clock), providerClockDiagnostic(last)) ||
		candidateLedgerExpired(providerClockDiagnostic(last), o.permit.ticket.deadline) ||
		o.execution.actual[0] != before[0] || o.execution.actual[1] != before[1] ||
		o.diagnostic.Tag != providerFirstFailureBurn || !o.diagnostic.Frozen || o.diagnostic.Failed {
		t.Fatal("changed private request escaped burn or the fixture failed before its healthy original clock", err, o.diagnostic)
	}
}

// Construct the actual original C/burn/capture prefix through the production
// methods with a normal owned peer. It never dials the official endpoint,
// manufactures a permit, replaces a selected bound or resets observations.
func providerFirstDiagnosticCapturedPrefix(t *testing.T) (*providerCustodyStore, *providerFirstOperationOwner, *providerIntegrationPermit, *ProviderTransportSample) {
	t.Helper()
	s, o := providerFirstOperationFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	permit, err := o.consumeLocked()
	if err != nil || permit == nil || !o.burnLocked(permit) {
		t.Fatal("genuine original C and irreversible first burn required", err)
	}
	connection := providerIntegrationOwnedTLS(t, o.request.body, providerFirstOperationResponse())
	ctx, cancel := context.WithTimeout(context.Background(), o.plan.capture.Timeout)
	defer cancel()
	sample, err := providerFirstOperationCaptureDiagnosed(ctx, connection, o.plan.capture, o.request, o.execution, o.tls, &o.diagnostic)
	_ = connection.Close()
	if err != nil || sample == nil || !sample.Complete || !permit.used || o.permit != permit || o.state != providerFirstOperationUsed ||
		s.sequence != 3 || !o.consumption || o.settlement || o.execution.failed || o.execution.closed ||
		o.diagnostic.Tag != providerFirstFailureUnclassified || o.diagnostic.Frozen || o.execution.actual[0] != 1 ||
		o.execution.actual[1] != uint64(sample.WireBytes) {
		t.Fatal("complete normal original capture before private finishing DATA", err)
	}
	return s, o, permit, sample
}

func TestProviderFirstFinishRejectsChangedPrivateDataWithoutPublishingResults(t *testing.T) {
	for _, mode := range []string{"same", "exchange", "settlement-policy", "ledger-state"} {
		t.Run(mode, func(t *testing.T) {
			s, o, permit, originalSample := providerFirstDiagnosticCapturedPrefix(t)
			s.mu.Lock()
			defer s.mu.Unlock()
			success := false
			defer func() {
				if !success {
					o.diagnostic.freeze(o.execution)
					o.failLocked()
				}
			}()
			sample := *originalSample
			want := providerFirstFailureUnclassified
			switch mode {
			case "exchange":
				sample.Endpoint, want = "private-invalid-endpoint-DATA", providerFirstFailureExchange
			case "settlement-policy":
				sample.Complete = false
				permit.ticket.policy, want = "!", providerFirstFailureSettlementData
			case "ledger-state":
				sample.Complete = false
				state := o.integration.ledger.tickets[permit.ticket.digest]
				if state.ticket == nil || state.ticket.digest != permit.ticket.digest || state.consumption == nil || state.terminal != "" {
					t.Fatal("genuine nonterminal consumed coordinator ticket required")
				}
				state.terminal = permit.ticket.digest // Private nonempty terminal DATA, not an admitted settlement.
				o.integration.ledger.tickets[permit.ticket.digest] = state
				want = providerFirstFailureSettlementLedger
			}
			before, held := o.execution.actual, s.stored
			out, err := o.finishLocked(permit, &sample, nil, &success)
			if mode == "same" {
				if err != nil || out == nil || !success || o.state != providerFirstOperationDone || s.sequence != 4 || !o.settlement ||
					o.diagnostic.Tag != want || s.activeMeter.Load() != nil || !o.execution.claimed {
					t.Fatal("positive original finish prerequisite", err)
				}
				return
			}
			if err == nil || out != nil || success || o.diagnostic.Tag != want || o.remaining == nil || !o.execution.closed ||
				o.execution.claimed || !permit.used || s.stored != held || o.remaining.actual[10] == 0 ||
				originalSample.Endpoint != SepoliaRPCURL || !originalSample.Complete {
				t.Fatal("changed private finishing DATA published a result or bypassed its intended diagnostic", mode, err, o.diagnostic)
			}
			if mode == "exchange" {
				if s.sequence != 4 || !o.settlement || !o.diagnostic.Frozen {
					t.Fatal("rejected exchange failed to retain its original terminal failed settlement")
				}
			} else if s.sequence != 3 || o.settlement {
				t.Fatal("rejected proposed settlement advanced the original retained prefix", mode)
			}
			for coordinate, actual := range before {
				if o.execution.actual[coordinate] < actual {
					t.Fatal("private finishing failure refunded original execution", mode, coordinate)
				}
			}
		})
	}
}
