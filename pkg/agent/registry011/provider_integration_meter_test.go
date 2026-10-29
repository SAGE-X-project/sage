package registry011

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"
)

// These are finite inert unit subjects. An owner pointer here is not an Agent
// selection, clock credential, actual filesystem profile or dispatch permit.
func providerMeterFixture(t *testing.T) (*providerIntegration, *CandidateAdmissionTicket, *providerWorkMeter, ClockSample) {
	t.Helper()
	object := providerPolicyTestObject()
	for _, level := range []string{"operation", "segment", "bootstrap"} {
		limits := object["limits"].(map[string]any)[level].(map[string]any)
		for _, field := range []string{"wire_bytes", "decoded_bytes", "disk_bytes_written"} {
			limits[field] = "1000"
		}
		limits["work_units"] = "1000000"
	}
	p, i, raw := admissionTestTicketPolicyFixture(t, object)
	work := raw["reservation"].(map[string]any)
	for _, field := range []string{"wire_bytes", "decoded_bytes", "disk_bytes_written"} {
		work[field] = "1000"
	}
	work["work_units"] = "1000000"
	ticket := admissionTestTicket(t, p, i, 0, raw)
	execution := ticket.work
	execution[2], execution[9], execution[10] = 800, 800, 900000
	remaining := candidateWorkVector{}
	remaining[2], remaining[9], remaining[10] = 200, 200, 100000
	owner := &providerIntegration{}
	meter := newProviderWorkMeter(owner, providerOperationReservation{work: ticket.work, execution: execution, remaining: remaining})
	meter.ticket = ticket.digest
	clock := ClockSample{i.clock.ClockID, i.clock.Epoch, i.clock.WallMS + 1, i.clock.MonotonicMS + 1, i.clock.UncertaintyMS}
	return owner, ticket, meter, clock
}

func TestProviderIntegrationMeasuredWorkIsNotReservation(t *testing.T) {
	owner, ticket, meter, clock := providerMeterFixture(t)
	actual := candidateWorkVector{1, 60, 0, 0, 0, 0, 0, 0, 1, 0, 0}
	if !meter.add(actual) || !meter.event(providerWriteEvent, 8) || !meter.event(providerCodecEvent, 20) {
		t.Fatal("finite measured events refused")
	}
	before := meter.actual
	measurement, err := meter.finish(owner, ticket, clock)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := measurement.claim(owner, ticket, clock)
	if !ok || got != before || got == ticket.work || got[9] != 8 || got[2] != 20 || got[10] != 410 {
		t.Fatal("reservation substituted for measured completed events", got)
	}
	for n := range got {
		if got[n]+meter.remaining[n] > ticket.work[n] {
			t.Fatal("remaining envelope not funded", n)
		}
	}
	if meter.add(candidateWorkVector{}) || meter.event(providerReadEvent, 1) {
		t.Fatal("closed measurement accepted more work")
	}
	if _, ok := measurement.claim(owner, ticket, clock); ok {
		t.Fatal("measurement consumed twice")
	}
}

func TestProviderIntegrationMeasurementOwnershipAndEnd(t *testing.T) {
	for _, mode := range []string{"foreign-owner", "foreign-ticket", "wrong-clock", "expired", "authority", "missing-write", "missing-segment", "failed", "copied-meter", "closed"} {
		t.Run(mode, func(t *testing.T) {
			owner, ticket, meter, clock := providerMeterFixture(t)
			if !meter.add(candidateWorkVector{1, 1, 0, 0, 0, 0, 0, 0, 1, 1, 1}) {
				t.Fatal("setup")
			}
			switch mode {
			case "foreign-owner":
				owner = &providerIntegration{}
			case "foreign-ticket":
				copy := *ticket
				copy.digest = admissionTestDigest("c")
				ticket = &copy
			case "wrong-clock":
				clock.Epoch = admissionTestDigest("d")
			case "expired":
				clock.WallMS = int64(ticket.deadline) + 1
				clock.MonotonicMS = ticket.intent.clock.MonotonicMS + clock.WallMS - ticket.intent.clock.WallMS
			case "authority":
				meter.authority = true
			case "missing-write":
				meter.actual[9] = 0
			case "missing-segment":
				meter.actual[8] = 0
			case "failed":
				meter.refuse()
			case "copied-meter":
				meter = &providerWorkMeter{owner: meter, integration: owner, ticket: ticket.digest, upper: meter.upper, remaining: meter.remaining, reservation: meter.reservation, actual: meter.actual}
			case "closed":
				meter.closed = true
			}
			if value, err := meter.finish(owner, ticket, clock); err == nil || value != nil {
				t.Fatal("unsupported ownership/end produced complete")
			}
		})
	}
	var empty *providerWorkMeter
	if empty.add(candidateWorkVector{}) || empty.event(providerReadEvent, 1) || empty.preflight(providerReadEvent, 1) {
		t.Fatal("nil meter accepted")
	}
	if value, err := empty.finish(nil, nil, ClockSample{}); value != nil || err == nil {
		t.Fatal("nil complete")
	}
}

func TestProviderIntegrationMeterOverflowRefusesBeforeMutation(t *testing.T) {
	for dimension := 0; dimension < 11; dimension++ {
		owner, _, meter, _ := providerMeterFixture(t)
		_ = owner
		meter.upper[dimension] = 1
		one := candidateWorkVector{}
		one[dimension] = 1
		if !meter.add(one) {
			t.Fatal("exact upper")
		}
		before := meter.actual
		if meter.add(one) || meter.actual != before || !meter.failed {
			t.Fatal("overflow mutated measured vector", dimension)
		}
	}
	for _, test := range []struct {
		kind providerWorkEvent
		size uint64
	}{
		{providerReadEvent, math.MaxUint64}, {providerCodecEvent, 1 << 32}, {providerClockEvent, 1}, {providerSyncEvent, 1}, {providerRenameEvent, 1}, {providerWorkEvent(255), 0},
	} {
		_, _, meter, _ := providerMeterFixture(t)
		before := meter.actual
		if meter.event(test.kind, test.size) || meter.actual != before {
			t.Fatal("invalid primitive event accepted", test)
		}
	}
}

func TestProviderIntegrationMeterAllClosedPrimitives(t *testing.T) {
	for _, kind := range []providerWorkEvent{providerReadEvent, providerWriteEvent, providerCodecEvent, providerHashEvent, providerCopyEvent, providerClockEvent, providerSyncEvent, providerTLSEvent, providerEncodeEvent, providerRenameEvent, providerCompareEvent} {
		_, _, meter, _ := providerMeterFixture(t)
		n := uint64(7)
		if kind == providerClockEvent || kind == providerSyncEvent || kind == providerRenameEvent {
			n = 0
		}
		before := meter.actual
		if !meter.preflight(kind, n) || meter.actual != before || !meter.event(kind, n) || meter.actual[10] == 0 {
			t.Fatal("primitive guard or event failed", kind)
		}
		if kind == providerWriteEvent && meter.actual[9] != n || kind == providerCodecEvent && meter.actual[2] != n || kind == providerEncodeEvent && meter.actual[2] != 0 {
			t.Fatal("primitive dimensions mixed", kind)
		}
	}
	_, _, meter, _ := providerMeterFixture(t)
	meter.upper[10] = 0
	if meter.preflight(providerReadEvent, 1) || !meter.failed || !reflect.DeepEqual(meter.actual, candidateWorkVector{}) {
		t.Fatal("preflight did work or exceeded its budget")
	}
}

func TestProviderIntegrationMeasurementCannotBeDecoded(t *testing.T) {
	for _, target := range []any{new(providerWorkMeter), new(providerMeasuredWork), new(providerIntegrationPermit), new(providerCustodyAcknowledgment), new(providerIntegration), new(providerCustodyStore)} {
		if json.Unmarshal([]byte(`{}`), target) == nil {
			t.Fatal("serialized data recreated private authority", reflect.TypeOf(target))
		}
	}
	var missing *providerMeasuredWork
	if _, ok := missing.claim(nil, nil, ClockSample{}); ok {
		t.Fatal("nil measurement")
	}
	owner, ticket, meter, clock := providerMeterFixture(t)
	meter.add(candidateWorkVector{1, 1, 0, 0, 0, 0, 0, 0, 1, 1, 1})
	measurement, err := meter.finish(owner, ticket, clock)
	if err != nil {
		t.Fatal(err)
	}
	wrong := clock
	wrong.MonotonicMS++
	if _, ok := measurement.claim(owner, ticket, wrong); ok {
		t.Fatal("different end accepted")
	}
	if _, ok := measurement.claim(&providerIntegration{}, ticket, clock); ok {
		t.Fatal("different owner accepted")
	}
	if _, ok := measurement.claim(owner, ticket, clock); !ok {
		t.Fatal("refusal burned rightful metadata claim")
	}
}
