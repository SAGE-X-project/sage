package registry011

import (
	"context"
	"strconv"
	"testing"
)

func TestProviderFirstOperationRejectsUnfundedPolicyBeforeFirstIO(t *testing.T) {
	for _, level := range []string{"operation", "segment", "bootstrap"} {
		for _, coordinate := range []int{0, 1, 2, 8, 9, 10} {
			t.Run(level+"/"+strconv.Itoa(coordinate), func(t *testing.T) {
				s, f := providerFirstOperationPreparedFixture(t)
				p, ok := providerFirstOperationPlanUpper(s.config.bounds, s.config.capture, f.scanPin.ordinal, f.records, s.retainedRaw, f.readerOutput)
				if !ok {
					t.Fatal("fresh scalar plan")
				}
				limits := *s.firstLimits
				s.firstLimits = &limits
				switch level {
				case "operation":
					limits.operation[coordinate] = p.reservation.work[coordinate] - 1
				case "segment":
					limits.segment[coordinate] = p.reservation.work[coordinate] - 1
				case "bootstrap":
					held, heldOK := providerScanUpperSum(providerScanOpenUpper(s.config), s.scanControl.held)
					if !heldOK {
						t.Fatal("original bootstrap hold")
					}
					limits.work[coordinate] = held.work[coordinate] + p.reservation.work[coordinate] - 1
				}
				before, clocks, stored, pin := f.meter.actual, f.meter.clocks, s.stored, s.frontierFiles.pin
				if owner, err := newProviderFirstOperationCandidate(s); err == nil || owner != nil || !s.failed || f.state != providerFirstAdmissionFailed || f.replayStarted || f.custodianStarted || f.plan != nil || f.meter.actual != before || f.meter.clocks != clocks || s.stored != stored || s.frontierFiles.pin != pin || s.firstOperation != nil || s.activeMeter.Load() != nil {
					t.Fatal("unfunded first-operation policy performed IO or created an owner", level, coordinate)
				}
			})
		}
	}
}

func TestProviderFirstOperationCannotConvertRetainedDataOrRepeatItsConstructor(t *testing.T) {
	var missing *providerCustodyStore
	if owner, err := newProviderFirstOperationCandidate(missing); err == nil || owner != nil {
		t.Fatal("missing store created an owner")
	}
	s, o := providerFirstOperationFixture(t)
	actual, stored := o.execution.actual, s.stored
	if second, err := newProviderFirstOperationCandidate(s); err == nil || second != nil || s.failed || s.stored != stored || o.execution.actual != actual {
		t.Fatal("the live constructor repeated a future hold")
	}
	other, _ := providerFirstWriterFixture(t, providerFirstWriterBounds())
	if data, err := other.retainFirstAdmissionCandidate(providerFirstTicketReservationData()); err != nil || data == nil {
		t.Fatal(err)
	}
	stored = other.stored
	if owner, err := newProviderFirstOperationCandidate(other); err == nil || owner != nil || other.failed || other.stored != stored {
		t.Fatal("closed F DATA was promoted into a future owner")
	}
}

func TestProviderFirstOperationMissingOriginalsOrDestinationsNeverDispatch(t *testing.T) {
	for _, mode := range []string{"ticket", "consumption-destination", "settlement-destination", "cancelled", "missing-context"} {
		t.Run(mode, func(t *testing.T) {
			s, o := providerFirstOperationFixture(t)
			stored, held := s.stored, s.scanControl.held
			switch mode {
			case "ticket":
				if s.evidence.root.Remove(providerObjectName("ticket", o.integration.firstTicket.digest)) != nil {
					t.Fatal("owned missing original")
				}
			case "consumption-destination":
				if s.custody.root.Mkdir(s.name("intent", 3), 0700) != nil {
					t.Fatal("owned unavailable destination")
				}
			case "settlement-destination":
				if s.custody.root.Mkdir(s.name("intent", 4), 0700) != nil {
					t.Fatal("owned unavailable destination")
				}
			}
			ctx := context.Background()
			switch mode {
			case "cancelled":
				cancelled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = cancelled
			case "missing-context":
				ctx = nil
			}
			var err error
			var data *providerFirstOperationResultData
			if mode == "settlement-destination" {
				conn := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, providerFirstOperationResponse())
				data, err = o.executeCandidateWithConnection(ctx, conn)
			} else {
				// These finite early failures must end before the only real dial.
				data, err = o.executeCandidate(ctx)
			}
			if err == nil || data != nil || !s.failed || o.state != providerFirstOperationFailed || s.stored != stored || s.scanControl.held != held || !o.execution.closed || !o.execution.failed || s.activeMeter.Load() != nil || s.last != nil {
				t.Fatal("uncertainty was refunded, retried or exposed a candidate result", mode)
			}
			if mode != "settlement-destination" && (o.execution.actual[0] != 0 || o.permit != nil) {
				t.Fatal("early finite refusal reached request dispatch", mode)
			}
			if again, err := o.executeCandidate(context.Background()); err == nil || again != nil || s.stored != stored {
				t.Fatal("failed candidate obtained another hold or dispatch")
			}
		})
	}
}

func TestProviderFirstOperationInitialAvailabilityCannotBorrowFutureWork(t *testing.T) {
	for _, mode := range []string{"closed-inspection", "incomplete-records", "physical-capacity", "finishing-allowance", "missing-enrollment", "admission-destination"} {
		t.Run(mode, func(t *testing.T) {
			s, f := providerFirstOperationPreparedFixture(t)
			before, stored := f.meter.actual, s.stored
			switch mode {
			case "closed-inspection":
				f.meter.closed = true
			case "incomplete-records":
				f.records = 3
			case "physical-capacity":
				plan, ok := providerFirstOperationPlanUpper(s.config.bounds, s.config.capture, f.scanPin.ordinal, f.records, s.retainedRaw, f.readerOutput)
				if !ok {
					t.Fatal("scalar quote")
				}
				limits := *s.firstLimits
				limits.live = stored + plan.storage.physical - 1
				s.firstLimits = &limits
			case "finishing-allowance":
				limits := *s.config.bounds.firstAdmission
				limits.work[10] = 1
				s.config.bounds.firstAdmission, f.limits = &limits, limits
			case "missing-enrollment":
				if s.evidence.root.Remove(providerObjectName("enrollment", s.enrollment)) != nil {
					t.Fatal("owned missing original")
				}
			case "admission-destination":
				if s.custody.root.Mkdir(s.name("intent", 2), 0700) != nil {
					t.Fatal("owned unavailable destination")
				}
			}
			if data, err := newProviderFirstOperationCandidate(s); err == nil || data != nil || s.sequence != 1 || s.stored < stored || s.last != nil || s.activeMeter.Load() != nil {
				t.Fatal("initial uncertainty created a request owner or refunded a hold", mode)
			}
			if mode != "missing-enrollment" && mode != "admission-destination" && f.meter.actual != before {
				t.Fatal("unfunded initial work performed a primitive", mode)
			}
		})
	}
}

// The inconsistent private states below remain finite unit DATA; they never
// write a changed original or run a malformed peer. Each selected live owner
// must refuse before producing C, a request or a reusable receipt.
func TestProviderFirstOperationMissingLiveCommandDataCannotDispatch(t *testing.T) {
	for _, mode := range []string{"no-clock", "later-first-ticket", "spent-consumption", "absent-change"} {
		t.Run(mode, func(t *testing.T) {
			s, o := providerFirstOperationFixture(t)
			stored := s.stored
			var err error
			switch mode {
			case "no-clock":
				s.clock.failed = true
			case "later-first-ticket":
				ticket := *o.integration.firstTicket
				ticket.sequence = 1
				o.integration.firstTicket = &ticket
			case "spent-consumption":
				ledger := o.integration.ledger
				state := ledger.tickets[o.integration.firstTicket.digest]
				state.consumption = &CandidateConsumption{}
				ledger.tickets[o.integration.firstTicket.digest] = state
			case "absent-change":
				o.state = providerFirstOperationConsumption
				err = o.commitLocked("consume", admissionTestDigest("a"), nil, nil, o.integration.intent.clock)
			}
			if mode != "absent-change" {
				_, err = o.consumeLocked()
			}
			if err == nil || s.sequence != 2 || s.stored != stored || s.last != nil || o.permit != nil || o.execution.actual[0] != 0 || o.execution.actual[9] != 0 {
				t.Fatal("missing live command state wrote, dispatched or created an ACK", mode)
			}
		})
	}
}

func TestProviderFirstOperationCannotSettleWithoutItsOriginalConsumption(t *testing.T) {
	var missing *providerFirstOperationOwner
	if result, err := missing.executeCandidate(nil); err == nil || result != nil {
		t.Fatal("missing owner executed")
	}
	s, o := providerFirstOperationFixture(t)
	before, stored := o.execution.actual, s.stored
	success := false
	result, err := o.finishLocked(nil, nil, errProviderIntegration, &success)
	if err == nil || result != nil || success || s.sequence != 2 || s.stored != stored || s.last != nil || o.execution.actual != before || o.remaining != nil || o.settlement {
		t.Fatal("settlement without the original consumed ticket performed work")
	}
}

// A proposed coordinator packet can disagree with the retained transaction
// arguments. These finite in-memory cases must fail before the first write;
// no altered original is persisted and no hostile peer runs.
func TestProviderFirstOperationCustodianRefusesDifferentProposedOriginals(t *testing.T) {
	for _, mode := range []string{"intent", "fresh-object"} {
		t.Run(mode, func(t *testing.T) {
			s, o := providerFirstOperationFixture(t)
			s.mu.Lock()
			defer s.mu.Unlock()
			c, meter := o.integration, o.execution
			clock, err := c.currentClock()
			if err != nil {
				t.Fatal(err)
			}
			consumption, err := providerFirstOperationConsumptionData(c.firstTicket, providerClockDiagnostic(clock), meter)
			if err != nil {
				t.Fatal(err)
			}
			change, err := c.ledger.AppendCandidateConsumption(consumption)
			if err != nil {
				t.Fatal(err)
			}
			o.state = providerFirstOperationConsumption
			p, err := providerFirstOperationChangeData(s, c.root, c.ledger, o.indexes, s.frontierReplay.current, "consume", consumption.digest, change, []providerIntegrationObjectInput{{"consumption", providerReplayPhaseBytes(consumption.raw, meter)}}, providerClockDiagnostic(clock), meter)
			if err != nil {
				t.Fatal(err)
			}
			intent, commit, frontier := p.intent, p.commit, p.frontier
			fresh := append([]providerIntegrationObject(nil), p.fresh...)
			o.change = p
			switch mode {
			case "intent":
				p.intent = []byte("different finite proposed intent")
			case "fresh-object":
				p.fresh[0].raw += " "
			}
			stored := s.stored
			cost, err := s.frontierReplay.retainOperationOwnedLocked(s, p.inventory, fresh, intent, commit, frontier, p.after, nil, nil, o)
			if err == nil || cost != (providerFrontierRetention{}) || !s.failed || !s.frontierReplay.failed || !s.frontierFiles.failed || s.sequence != 2 || s.stored != stored || meter.actual[9] != 0 || meter.actual[0] != 0 || s.last != nil || o.consumption {
				t.Fatal("different proposed originals reached a retained C or released its hold", mode)
			}
		})
	}
}
