package registry011

import "strconv"

const providerHistoryGroupRootDomain = "sage-provider-history-group-root-v1"

// One replay model keeps only the current group and cumulative scalar charges.
// It does not cache the transcript, create a hold, authenticate originals or
// issue an ACK/permit. Coordinator and custodian must reconstruct SEPARATE
// instances from their own original reads under the same admitted parent.
// Each enclosing concrete owner must serialize this original under its own
// lock. These bookkeeping methods are not independently concurrent roles.
type providerHistoryGroupLedger struct {
	owner                            *providerHistoryGroupLedger
	program, profile, costs, initial string
	head, previous, entry            string
	maximum, charged                 providerHistoryGroupTotals
	maximumStages, stage, sequence   uint64
	ticket                           *providerHistoryGroupTicket
	consumption                      *providerHistoryGroupConsumption
	phase                            string
	failed, terminal                 bool
}

type providerHistoryGroupTotals struct {
	work                                     candidateWorkVector
	physical, retainedRaw, payload, metadata uint64
	objects, files, clocks                   uint64
}

func (*providerHistoryGroupLedger) UnmarshalJSON([]byte) error { return ErrConfiguration }

func newProviderHistoryGroupLedger(program, profile, costs, initial string, maximumStages uint64, maximum providerHistoryGroupTotals) (*providerHistoryGroupLedger, error) {
	return newProviderHistoryGroupLedgerObserved(program, profile, costs, initial, maximumStages, maximum, nil)
}

func newProviderHistoryGroupLedgerObserved(program, profile, costs, initial string, maximumStages uint64, maximum providerHistoryGroupTotals, meter *providerWorkMeter) (*providerHistoryGroupLedger, error) {
	if !lowerHash(program) || !lowerHash(profile) || !lowerHash(costs) || !lowerHash(initial) ||
		maximumStages == 0 || maximumStages > (1<<20)/3 || !maximum.valid() {
		return nil, ErrConfiguration
	}
	l := &providerHistoryGroupLedger{program: program, profile: profile, costs: costs, initial: initial, maximumStages: maximumStages, maximum: maximum, phase: "prepared"}
	l.owner = l
	raw := l.rootObserved(meter)
	if len(raw) == 0 {
		return nil, ErrConfiguration
	}
	l.head = providerReplayDigest(providerHistoryGroupRootDomain, raw, meter)
	if l.head == "" {
		return nil, ErrConfiguration
	}
	return l, nil
}

func (t providerHistoryGroupTotals) valid() bool {
	return t.work[0] > 0 && t.work[1] > 0 && t.work[10] > 0 && t.physical > 0 && t.physical <= 1<<40 &&
		t.retainedRaw > 0 && t.retainedRaw <= t.physical && t.payload > 0 && t.payload <= 16<<30 && t.payload <= t.retainedRaw &&
		t.metadata > 0 && t.metadata <= 64<<30 && t.metadata <= t.retainedRaw &&
		t.objects > 0 && t.objects <= 1<<20 && t.files >= t.objects && t.files <= t.physical && t.clocks > 0 && t.clocks <= 1<<20
}

func (t providerHistoryGroupTotals) within(maximum providerHistoryGroupTotals) bool {
	for i, n := range t.work {
		if n > maximum.work[i] {
			return false
		}
	}
	return t.physical <= maximum.physical && t.retainedRaw <= maximum.retainedRaw && t.payload <= maximum.payload && t.metadata <= maximum.metadata &&
		t.objects <= maximum.objects && t.files <= maximum.files && t.clocks <= maximum.clocks
}

func (t providerHistoryGroupTotals) add(upper providerHistoryGroupUpper, maximum providerHistoryGroupTotals) (providerHistoryGroupTotals, bool) {
	work, ok := upper.work()
	if !ok {
		return providerHistoryGroupTotals{}, false
	}
	next := t
	for i, n := range work {
		if !candidateCapacity(0, t.work[i], n, maximum.work[i]) {
			return providerHistoryGroupTotals{}, false
		}
		next.work[i] += n
	}
	// Values refer to different resource domains, not interchangeable capacity.
	// All checks precede the single scalar-state replacement, so a refused
	// reservation cannot partially consume a different row or refund a prefix.
	if !candidateCapacity(0, t.physical, upper.physical, maximum.physical) ||
		!candidateCapacity(0, t.retainedRaw, upper.retainedRaw, maximum.retainedRaw) ||
		!candidateCapacity(0, t.payload, upper.payload, maximum.payload) ||
		!candidateCapacity(0, t.metadata, upper.metadata, maximum.metadata) ||
		!candidateCapacity(0, t.objects, upper.objects, maximum.objects) ||
		!candidateCapacity(0, t.files, upper.files, maximum.files) ||
		!candidateCapacity(0, t.clocks, upper.clocks, maximum.clocks) {
		return providerHistoryGroupTotals{}, false
	}
	next.physical += upper.physical
	next.retainedRaw += upper.retainedRaw
	next.payload += upper.payload
	next.metadata += upper.metadata
	next.objects += upper.objects
	next.files += upper.files
	next.clocks += upper.clocks
	return next, true
}

func (t providerHistoryGroupTotals) object() map[string]any {
	return map[string]any{
		"work": candidateLedgerWorkObject(t.work), "physical_bytes": strconv.FormatUint(t.physical, 10),
		"retained_raw_bytes": strconv.FormatUint(t.retainedRaw, 10), "payload_bytes": strconv.FormatUint(t.payload, 10),
		"metadata_bytes": strconv.FormatUint(t.metadata, 10),
		"objects":        strconv.FormatUint(t.objects, 10), "files": strconv.FormatUint(t.files, 10), "clock_reads": strconv.FormatUint(t.clocks, 10),
	}
}

func (l *providerHistoryGroupLedger) root() []byte {
	return l.rootObserved(nil)
}

func (l *providerHistoryGroupLedger) rootObserved(meter *providerWorkMeter) []byte {
	var ticket, consumption any
	if l.ticket != nil {
		ticket = l.ticket.digest
	}
	if l.consumption != nil {
		consumption = l.consumption.digest
	}
	return providerReplayCanonical(map[string]any{
		"format": providerHistoryGroupRootDomain, "program": l.program, "profile": l.profile, "cost_revision": l.costs,
		"initial": l.initial, "previous": l.previous, "entry": l.entry, "stage": strconv.FormatUint(l.stage, 10), "sequence": strconv.FormatUint(l.sequence, 10),
		"maximum_stages": strconv.FormatUint(l.maximumStages, 10), "maximum": l.maximum.object(), "charged": l.charged.object(),
		"phase": l.phase, "ticket": ticket, "consumption": consumption, "failed": l.failed, "terminal": l.terminal,
	}, candidateLedgerMetadataLimit, meter)
}

func (l *providerHistoryGroupLedger) valid() bool {
	return l.validObserved(nil)
}

func (l *providerHistoryGroupLedger) validObserved(meter *providerWorkMeter) bool {
	if l == nil || l.owner != l || !lowerHash(l.program) || !lowerHash(l.profile) || !lowerHash(l.costs) || !lowerHash(l.initial) ||
		l.failed || l.terminal || l.maximumStages == 0 || l.maximumStages > (1<<20)/3 || l.stage > l.maximumStages ||
		l.sequence > 1<<20 || !l.maximum.valid() || !l.charged.within(l.maximum) {
		return false
	}
	raw := l.rootObserved(meter)
	if len(raw) == 0 || !providerReplayTextEqual(providerReplayDigest(providerHistoryGroupRootDomain, raw, meter), l.head, meter) {
		return false
	}
	switch l.phase {
	case "prepared":
		return l.stage == 0 && l.sequence == 0 && l.previous == "" && l.entry == "" && l.ticket == nil && l.consumption == nil && l.charged == (providerHistoryGroupTotals{})
	case "settled":
		return l.stage > 0 && l.sequence == 3*l.stage && lowerHash(l.previous) && lowerHash(l.entry) && l.ticket == nil && l.consumption == nil
	case "admitted", "consumed":
		if l.ticket == nil || !providerHistoryGroupTicketValidObserved(l.ticket, meter) || l.ticket.program != l.program ||
			l.ticket.profile != l.profile || l.ticket.costs != l.costs || l.ticket.table.stage != l.stage+1 || !lowerHash(l.previous) {
			return false
		}
		if l.phase == "admitted" {
			return l.sequence == 3*l.stage+1 && providerReplayTextEqual(l.entry, l.ticket.digest, meter) && l.consumption == nil
		}
		return l.sequence == 3*l.stage+2 && providerHistoryGroupConsumptionValidObserved(l.consumption, l.ticket, meter) && providerReplayTextEqual(l.entry, l.consumption.digest, meter)
	}
	return false
}

func (l *providerHistoryGroupLedger) refuse() error {
	if l != nil && l.owner == l && !l.terminal {
		// Preserve the entire charged upper and every consumed bit. Poison is
		// a local quarantine, not a manufactured durable failure receipt/root.
		l.failed = true
	}
	return ErrConfiguration
}

// Failure after an admitted debit quarantines this model without restoring
// any charged coordinate. A failed encode/hash cannot manufacture a new root.
func (l *providerHistoryGroupLedger) advanceObserved(phase, entry string, meter *providerWorkMeter) error {
	l.previous, l.phase, l.entry = l.head, phase, entry
	l.sequence++
	raw := l.rootObserved(meter)
	if len(raw) == 0 {
		return l.refuse()
	}
	root := providerReplayDigest(providerHistoryGroupRootDomain, raw, meter)
	if root == "" {
		return l.refuse()
	}
	l.head = root
	return nil
}

// The caller must pass the independently reconstructed original T, including
// its own table. This model performs typed replay only. A producer digest or
// this model's successful return cannot acknowledge a filesystem transaction.
func (l *providerHistoryGroupLedger) admit(ticket *providerHistoryGroupTicket) error {
	return l.admitObserved(ticket, nil)
}

func (l *providerHistoryGroupLedger) admitObserved(ticket *providerHistoryGroupTicket, meter *providerWorkMeter) error {
	if !l.validObserved(meter) || l.stage >= l.maximumStages || l.phase != "prepared" && l.phase != "settled" ||
		!providerHistoryGroupTicketValidObserved(ticket, meter) || ticket.program != l.program || ticket.profile != l.profile ||
		ticket.costs != l.costs || !providerReplayTextEqual(ticket.previous, l.head, meter) || ticket.table.stage != l.stage+1 {
		return l.refuse()
	}
	charged, ok := l.charged.add(ticket.upper, l.maximum)
	if !ok {
		return l.refuse()
	}
	copy := *ticket
	l.ticket, l.charged = &copy, charged
	return l.advanceObserved("admitted", ticket.digest, meter)
}

func (l *providerHistoryGroupLedger) consume(consumption *providerHistoryGroupConsumption) error {
	return l.consumeObserved(consumption, nil)
}

func (l *providerHistoryGroupLedger) consumeObserved(consumption *providerHistoryGroupConsumption, meter *providerWorkMeter) error {
	if !l.validObserved(meter) || l.phase != "admitted" || !providerHistoryGroupConsumptionValidObserved(consumption, l.ticket, meter) || !providerReplayTextEqual(consumption.previous, l.head, meter) {
		return l.refuse()
	}
	copy := *consumption
	l.consumption = &copy
	return l.advanceObserved("consumed", consumption.digest, meter)
}

func (l *providerHistoryGroupLedger) settle(settlement *providerHistoryGroupSettlement) error {
	return l.settleObserved(settlement, nil)
}

func (l *providerHistoryGroupLedger) settleObserved(settlement *providerHistoryGroupSettlement, meter *providerWorkMeter) error {
	if !l.validObserved(meter) || l.phase != "consumed" || settlement == nil || !providerReplayTextEqual(settlement.previous, l.head, meter) || settlement.count < 0 || settlement.count > providerHistoryGroupMembers {
		return l.refuse()
	}
	copy, err := newProviderHistoryGroupSettlementObserved(l.ticket, l.consumption, settlement.previous, settlement.inventory, settlement.outcome, settlement.failure, settlement.results[:settlement.count], meter)
	if err != nil || !providerHistoryGroupSettlementEqualObserved(copy, settlement, meter) {
		return l.refuse()
	}
	// Retain the exact S reference in the root's predecessor chain. No actual
	// report is subtracted from the admitted upper after success or failure.
	l.ticket, l.consumption = nil, nil
	l.stage++
	phase := "settled"
	if settlement.outcome == "failed" {
		l.failed, l.terminal, phase = true, true, "failed"
	}
	return l.advanceObserved(phase, settlement.digest, meter)
}

// The other fields are the exact inputs of the reconstruction above. Its
// derived ticket/C identifiers, entire fixed result array and encoded binding
// are compared explicitly, including any replaced unused array tail.
func providerHistoryGroupSettlementEqualObserved(a, b *providerHistoryGroupSettlement, meter *providerWorkMeter) bool {
	for _, pair := range [4][2]string{{a.ticket, b.ticket}, {a.consumption, b.consumption}, {a.raw, b.raw}, {a.digest, b.digest}} {
		if !providerReplayTextEqual(pair[0], pair[1], meter) {
			return false
		}
	}
	for i := range a.results {
		if !providerReplayTextEqual(a.results[i], b.results[i], meter) {
			return false
		}
	}
	return true
}
