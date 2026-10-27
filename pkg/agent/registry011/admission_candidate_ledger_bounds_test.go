package registry011

import (
	"math"
	"reflect"
	"strconv"
	"testing"
)

func ledgerTestRefusesWithoutMutation(t *testing.T, m *CandidateAdmissionLedger, operation func() error) {
	t.Helper()
	before := ledgerTestCurrent(t, m)
	tickets := make(map[string]candidateLedgerTicket, len(m.tickets))
	for k, v := range m.tickets {
		tickets[k] = v
	}
	segments := make(map[uint64]candidateBudgetTotals, len(m.segments))
	for k, v := range m.segments {
		segments[k] = v
	}
	lastTicket, lastConsumption := m.lastTicket, m.lastConsumption
	if operation() == nil {
		t.Fatal("expected refusal")
	}
	ledgerTestUnchanged(t, m, before)
	if !reflect.DeepEqual(tickets, m.tickets) || !reflect.DeepEqual(segments, m.segments) || lastTicket != m.lastTicket || lastConsumption != m.lastConsumption {
		t.Fatal("refusal changed retained states or segment totals")
	}
}

func TestAdmissionCandidateLedgerConstructorBounds(t *testing.T) {
	p, i, first := ledgerTestFirst(t, nil)
	for _, mode := range []string{"records-zero", "records-negative", "records-large", "payload-zero", "payload-large", "metadata-zero", "metadata-large"} {
		t.Run(mode, func(t *testing.T) {
			b := ledgerTestBounds()
			switch mode {
			case "records-zero":
				b.MaximumRecords = 0
			case "records-negative":
				b.MaximumRecords = -1
			case "records-large":
				b.MaximumRecords = 4097
			case "payload-zero":
				b.MaximumPayloadBytes = 0
			case "payload-large":
				b.MaximumPayloadBytes = (16 << 20) + 1
			case "metadata-zero":
				b.MaximumMetadataBytes = 0
			case "metadata-large":
				b.MaximumMetadataBytes = (64 << 20) + 1
			}
			if got, err := NewCandidateAdmissionLedger(p, i, first, b); got != nil || err != ErrConfiguration {
				t.Fatal("invalid explicit bounds accepted")
			}
		})
	}
	for _, bad := range []*ProviderPolicyCandidate{nil, {}} {
		if got, err := NewCandidateAdmissionLedger(bad, i, first, ledgerTestBounds()); got != nil || err == nil {
			t.Fatal("invalid policy")
		}
	}
	for _, bad := range []*CandidateAdmissionIntent{nil, {}} {
		if got, err := NewCandidateAdmissionLedger(p, bad, first, ledgerTestBounds()); got != nil || err == nil {
			t.Fatal("invalid intent")
		}
	}
	for _, bad := range []*CandidateAdmissionTicket{nil, {}} {
		if got, err := NewCandidateAdmissionLedger(p, i, bad, ledgerTestBounds()); got != nil || err == nil {
			t.Fatal("invalid first ticket")
		}
	}
	if got, err := NewCandidateAdmissionLedger(p, i, ledgerTestNext(t, p, first, 0), ledgerTestBounds()); got != nil || err == nil {
		t.Fatal("later ticket replaced initial anchor")
	}
	object := providerPolicyTestObject()
	object["clock"].(map[string]any)["id"] = "other-candidate-clock"
	p2, i2, t2 := ledgerTestFirst(t, object)
	if _, err := NewCandidateAdmissionLedger(p2, i, first, ledgerTestBounds()); err == nil {
		t.Fatal("foreign policy context")
	}
	if _, err := NewCandidateAdmissionLedger(p, i2, first, ledgerTestBounds()); err == nil {
		t.Fatal("foreign intent context")
	}
	if _, err := NewCandidateAdmissionLedger(p, i, t2, ledgerTestBounds()); err == nil {
		t.Fatal("foreign initial ticket")
	}
	wide := ledgerTestModel(t, p, i, first, ledgerTestBounds())
	d, _ := ledgerTestCurrent(t, wide).Diagnostic()
	b := ledgerTestBounds()
	b.MaximumPayloadBytes, b.MaximumMetadataBytes = d.PayloadBytes, d.MetadataBytes
	ledgerTestModel(t, p, i, first, b)
	for _, mode := range []string{"payload", "metadata", "anchor"} {
		short := b
		switch mode {
		case "payload":
			short.MaximumPayloadBytes--
		case "metadata":
			short.MaximumMetadataBytes--
		case "anchor":
			short.MaximumMetadataBytes = 1
		}
		if got, err := NewCandidateAdmissionLedger(p, i, first, short); got != nil || err == nil {
			t.Fatal("initial byte quota exceeded")
		}
	}
}

func TestAdmissionCandidateLedgerEveryCumulativeWorkDimension(t *testing.T) {
	for _, dimension := range []string{"requests", "wire_bytes", "decoded_bytes", "blocks", "transactions", "frames", "logs", "proofs", "disk_bytes_written", "work_units"} {
		t.Run(dimension, func(t *testing.T) {
			object := providerPolicyTestObject()
			limits := object["limits"].(map[string]any)
			limits["operation"].(map[string]any)[dimension] = "2"
			limits["segment"].(map[string]any)[dimension] = "2"
			limits["bootstrap"].(map[string]any)[dimension] = "4"
			p, i, first := ledgerTestFirst(t, object)
			m := ledgerTestModel(t, p, i, first, ledgerTestBounds())
			if _, err := m.AppendCandidateSettlement(ledgerTestSettlement(t, first, "failed", 10)); err != nil {
				t.Fatal(err)
			}
			second := ledgerTestNext(t, p, first, 0)
			if _, err := m.AppendCandidateTicket(second); err != nil {
				t.Fatal(err)
			}
			thirdSameSegment := ledgerTestNext(t, p, second, 0)
			ledgerTestRefusesWithoutMutation(t, m, func() error { _, err := m.AppendCandidateTicket(thirdSameSegment); return err })
			third := ledgerTestNext(t, p, second, 1)
			if _, err := m.AppendCandidateTicket(third); err != nil {
				t.Fatal("fresh segment should fit remaining bootstrap", err)
			}
			fourth := ledgerTestNext(t, p, third, 1)
			if _, err := m.AppendCandidateTicket(fourth); err != nil {
				t.Fatal(err)
			}
			fifth := ledgerTestNext(t, p, fourth, 2)
			ledgerTestRefusesWithoutMutation(t, m, func() error { _, err := m.AppendCandidateTicket(fifth); return err })
		})
	}
	// Segment count is also bound to the ordinal in the parsed ticket. Charging
	// earlier segments cannot make ordinal2 valid under a two-segment budget.
	object := providerPolicyTestObject()
	for _, level := range []string{"operation", "segment", "bootstrap"} {
		object["limits"].(map[string]any)[level].(map[string]any)["segments"] = "2"
	}
	p, i, first := ledgerTestFirst(t, object)
	m := ledgerTestModel(t, p, i, first, ledgerTestBounds())
	if _, err := m.AppendCandidateSettlement(ledgerTestSettlement(t, first, "failed", 10)); err != nil {
		t.Fatal(err)
	}
	second := ledgerTestNext(t, p, first, 1)
	if _, err := m.AppendCandidateTicket(second); err != nil {
		t.Fatal(err)
	}
	if _, err := m.AppendCandidateSettlement(ledgerTestSettlement(t, second, "cancelled", 20)); err != nil {
		t.Fatal(err)
	}
	d, _ := ledgerTestCurrent(t, m).Diagnostic()
	if d.Charged.Segments != 2 || d.Outstanding.Segments != 0 || len(m.segments) != 2 {
		t.Fatal("charged segments were discarded", d)
	}
	raw := contentJSON(second.CanonicalBytes()).(map[string]any)
	raw["sequence"], raw["previous_ticket"], raw["segment"] = "2", second.digest, candidateAdmissionSegment(i.digest, 2)
	if got, err := ParseCandidateAdmissionTicket(p, i, 2, providerPolicyTestBytes(t, raw), 64<<10); got != nil || err == nil {
		t.Fatal("new segment bypassed retained ordinal limit")
	}
}

func TestAdmissionCandidateLedgerCapacitySurvivesSettlement(t *testing.T) {
	for _, mode := range []string{"live", "retained"} {
		t.Run(mode, func(t *testing.T) {
			object := providerPolicyTestObject()
			limits := object["limits"].(map[string]any)
			limits["live_disk_bytes"], limits["retained_bytes"] = "1000", "150"
			if mode == "live" {
				limits["live_disk_bytes"], limits["retained_bytes"] = "150", "100"
			}
			p, i, raw := admissionTestTicketPolicyFixture(t, object)
			if mode == "live" {
				raw["retained_bytes_reserved"] = "40"
			}
			first := admissionTestTicket(t, p, i, 0, raw)
			m := ledgerTestModel(t, p, i, first, ledgerTestBounds())
			if _, err := m.AppendCandidateSettlement(ledgerTestSettlement(t, first, "uncertain", 10)); err != nil {
				t.Fatal(err)
			}
			second := ledgerTestNext(t, p, first, 0)
			ledgerTestRefusesWithoutMutation(t, m, func() error { _, err := m.AppendCandidateTicket(second); return err })
		})
	}
}

func TestAdmissionCandidateLedgerPayloadRecordMetadataQuotas(t *testing.T) {
	p, i, first := ledgerTestFirst(t, nil)
	c := ledgerTestConsumption(t, first, nil, 10)
	s := ledgerTestSettlement(t, first, "failed", 20)
	wide := ledgerTestModel(t, p, i, first, ledgerTestBounds())
	if _, err := wide.AppendCandidateConsumption(c); err != nil {
		t.Fatal(err)
	}
	afterC, _ := ledgerTestCurrent(t, wide).Diagnostic()
	for _, mode := range []string{"records", "payload", "metadata"} {
		t.Run(mode, func(t *testing.T) {
			bounds := ledgerTestBounds()
			switch mode {
			case "records":
				bounds.MaximumRecords = 1
			case "payload":
				bounds.MaximumPayloadBytes = uint64(len(first.raw) + len(c.raw) - 1)
			case "metadata":
				bounds.MaximumMetadataBytes = afterC.MetadataBytes - 1
			}
			m := ledgerTestModel(t, p, i, first, bounds)
			ledgerTestRefusesWithoutMutation(t, m, func() error { _, err := m.AppendCandidateConsumption(c); return err })
		})
	}
	bounds := ledgerTestBounds()
	bounds.MaximumRecords, bounds.MaximumPayloadBytes, bounds.MaximumMetadataBytes = 2, afterC.PayloadBytes, afterC.MetadataBytes
	m := ledgerTestModel(t, p, i, first, bounds)
	if _, err := m.AppendCandidateConsumption(c); err != nil {
		t.Fatal("exact model quotas refused", err)
	}
	ledgerTestRefusesWithoutMutation(t, m, func() error { _, err := m.AppendCandidateSettlement(s); return err })
	ledgerTestRefusesWithoutMutation(t, m, func() error { _, err := m.AppendCandidateTicket(ledgerTestNext(t, p, first, 0)); return err })
}

func TestAdmissionCandidateLedgerOverflowAndForeignRecords(t *testing.T) {
	object := providerPolicyTestObject()
	for _, level := range []string{"operation", "segment", "bootstrap"} {
		object["limits"].(map[string]any)[level].(map[string]any)["wire_bytes"] = strconv.FormatUint(math.MaxUint64, 10)
	}
	p, i, raw := admissionTestTicketPolicyFixture(t, object)
	raw["reservation"].(map[string]any)["wire_bytes"] = strconv.FormatUint(math.MaxUint64, 10)
	first := admissionTestTicket(t, p, i, 0, raw)
	m := ledgerTestModel(t, p, i, first, ledgerTestBounds())
	second := ledgerTestNext(t, p, first, 0)
	ledgerTestRefusesWithoutMutation(t, m, func() error { _, err := m.AppendCandidateTicket(second); return err })
	_, _, foreign := ledgerTestFirst(t, nil)
	ledgerTestRefusesWithoutMutation(t, m, func() error { _, err := m.AppendCandidateTicket(foreign); return err })
	ledgerTestRefusesWithoutMutation(t, m, func() error {
		_, err := m.AppendCandidateConsumption(ledgerTestConsumption(t, foreign, nil, 10))
		return err
	})
	ledgerTestRefusesWithoutMutation(t, m, func() error {
		_, err := m.AppendCandidateSettlement(ledgerTestSettlement(t, foreign, "failed", 10))
		return err
	})
}
