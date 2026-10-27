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
			if mode == "cancelled" {
				cancelled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = cancelled
			} else if mode == "missing-context" {
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
