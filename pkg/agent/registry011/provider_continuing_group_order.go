package registry011

const (
	providerHistoryOrderPrepared = iota + 1
	providerHistoryOrderConsumed
	providerHistoryOrderConfirmed
	providerHistoryOrderInFlight
	providerHistoryOrderClosed
	providerHistoryOrderFailed
)

// In-memory order bookkeeping only, never an ACK, retained consumption or an
// execution permit. A future concrete owner must hold its original instance
// under its own lock, consume all members before C, verify the independent C
// ACK before confirmation, and verify S/comparison before closing. No current
// Source, owner or transport accepts this object as authority. Constructing
// another counter does not restore any operational capacity or failed lifetime.
type providerHistoryGroupOrder struct {
	owner          *providerHistoryGroupOrder
	program, table string
	count, next    int
	consumed, sent uint32
	state          int
}

func (*providerHistoryGroupOrder) UnmarshalJSON([]byte) error { return ErrConfiguration }

func newProviderHistoryGroupOrder(program, table string, count int) (*providerHistoryGroupOrder, error) {
	if !lowerHash(program) || !lowerHash(table) || count < 1 || count > providerHistoryGroupMembers {
		return nil, ErrConfiguration
	}
	o := &providerHistoryGroupOrder{program: program, table: table, count: count, state: providerHistoryOrderPrepared}
	o.owner = o
	return o, nil
}

func providerHistoryMemberPrefix(count int) (uint32, bool) {
	if count < 0 || count > providerHistoryGroupMembers {
		return 0, false
	}
	if count == 0 {
		return 0, true
	}
	return ^uint32(0) >> (32 - uint(count)), true
}

func (o *providerHistoryGroupOrder) valid() bool {
	if o == nil || o.owner != o || !lowerHash(o.program) || !lowerHash(o.table) ||
		o.count < 1 || o.count > providerHistoryGroupMembers || o.next < 0 || o.next > o.count {
		return false
	}
	all, _ := providerHistoryMemberPrefix(o.count)
	prefix, _ := providerHistoryMemberPrefix(o.next)
	switch o.state {
	case providerHistoryOrderPrepared:
		return o.next == 0 && o.consumed == 0 && o.sent == 0
	case providerHistoryOrderConsumed:
		return o.next == 0 && o.consumed == all && o.sent == 0
	case providerHistoryOrderConfirmed:
		return o.consumed == all && o.sent == prefix
	case providerHistoryOrderInFlight:
		prefix, ok := providerHistoryMemberPrefix(o.next + 1)
		return ok && o.next < o.count && o.consumed == all && o.sent == prefix
	case providerHistoryOrderClosed:
		return o.next == o.count && o.consumed == all && o.sent == all
	}
	return false
}

func (o *providerHistoryGroupOrder) refuse() bool {
	if o != nil && o.owner == o && o.state != providerHistoryOrderClosed {
		o.state = providerHistoryOrderFailed
	}
	return false
}

// Called before the concrete C write. A later failed or uncertain write keeps
// every bit consumed. This transition alone does not perform or acknowledge C.
func (o *providerHistoryGroupOrder) consumeAll() bool {
	if !o.valid() || o.state != providerHistoryOrderPrepared {
		return o.refuse()
	}
	o.consumed, _ = providerHistoryMemberPrefix(o.count)
	o.state = providerHistoryOrderConsumed
	return true
}

// This records caller-side progress only. The future typed owner must verify
// actual independent retained originals/ACK before calling it; a bool or DATA
// counter state is not that verification and cannot authorize dispatch.
func (o *providerHistoryGroupOrder) confirmConsumption() bool {
	if !o.valid() || o.state != providerHistoryOrderConsumed {
		return o.refuse()
	}
	o.state = providerHistoryOrderConfirmed
	return true
}

func (o *providerHistoryGroupOrder) begin(ordinal int) bool {
	if !o.valid() || o.state != providerHistoryOrderConfirmed || ordinal != o.next || ordinal >= o.count {
		return o.refuse()
	}
	o.sent |= uint32(1) << uint(ordinal)
	o.state = providerHistoryOrderInFlight
	return true
}

func (o *providerHistoryGroupOrder) complete(ordinal int) bool {
	if !o.valid() || o.state != providerHistoryOrderInFlight || ordinal != o.next {
		return o.refuse()
	}
	o.next++
	o.state = providerHistoryOrderConfirmed
	return true
}

// Closing the bookkeeping is not S, comparison or a returned protected result.
// Those effects still need the independently qualified concrete owner.
func (o *providerHistoryGroupOrder) close() bool {
	if !o.valid() || o.state != providerHistoryOrderConfirmed || o.next != o.count {
		return o.refuse()
	}
	o.state = providerHistoryOrderClosed
	return true
}
