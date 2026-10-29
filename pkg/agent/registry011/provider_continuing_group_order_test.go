package registry011

import (
	"encoding/json"
	"testing"
)

func historyOrderFixture(t *testing.T, count int) *providerHistoryGroupOrder {
	t.Helper()
	o, err := newProviderHistoryGroupOrder(admissionTestDigest("a"), admissionTestDigest("b"), count)
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func TestProviderHistoryGroupOrderConsumesAllBeforeEveryOrderedMember(t *testing.T) {
	for _, count := range []int{1, providerHistoryGroupMembers} {
		o := historyOrderFixture(t, count)
		all, _ := providerHistoryMemberPrefix(count)
		if !o.valid() || !o.consumeAll() || o.consumed != all || o.sent != 0 || !o.confirmConsumption() {
			t.Fatal("all members were not consumed before confirmation")
		}
		for ordinal := range count {
			before := o.sent
			if !o.begin(ordinal) || o.next != ordinal || o.sent == before || o.consumed != all || !o.complete(ordinal) || o.next != ordinal+1 {
				t.Fatal("normal ordered member progress refused", ordinal)
			}
		}
		if !o.close() || !o.valid() || o.state != providerHistoryOrderClosed {
			t.Fatal("complete bookkeeping refused closure")
		}
		if o.begin(0) || o.complete(0) || o.consumeAll() || o.confirmConsumption() || o.close() || o.state != providerHistoryOrderClosed || o.consumed != all || o.sent != all {
			t.Fatal("closed bookkeeping reopened or lost its history")
		}
	}
}

func TestProviderHistoryGroupOrderRejectsSkippedAndRepeatedTransitions(t *testing.T) {
	for _, state := range []string{"begin-before-consumption", "confirm-before-consumption", "close-before-consumption", "complete-before-consumption", "consume-twice", "begin-before-confirmation", "confirm-twice", "skip-first", "negative", "past-count", "complete-before-begin", "close-before-end", "begin-twice", "complete-wrong", "repeat-completed", "extra-after-end"} {
		t.Run(state, func(t *testing.T) {
			o := historyOrderFixture(t, 2)
			var result bool
			switch state {
			case "begin-before-consumption":
				result = o.begin(0)
			case "confirm-before-consumption":
				result = o.confirmConsumption()
			case "close-before-consumption":
				result = o.close()
			case "complete-before-consumption":
				result = o.complete(0)
			default:
				if !o.consumeAll() {
					t.Fatal("normal consume failed")
				}
				switch state {
				case "consume-twice":
					result = o.consumeAll()
				case "begin-before-confirmation":
					result = o.begin(0)
				default:
					if !o.confirmConsumption() {
						t.Fatal("normal confirmation failed")
					}
					switch state {
					case "confirm-twice":
						result = o.confirmConsumption()
					case "skip-first":
						result = o.begin(1)
					case "negative":
						result = o.begin(-1)
					case "past-count":
						result = o.begin(2)
					case "complete-before-begin":
						result = o.complete(0)
					case "close-before-end":
						result = o.close()
					default:
						if !o.begin(0) {
							t.Fatal("normal first member failed")
						}
						switch state {
						case "begin-twice":
							result = o.begin(0)
						case "complete-wrong":
							result = o.complete(1)
						case "repeat-completed":
							if !o.complete(0) {
								t.Fatal("normal first completion failed")
							}
							result = o.begin(0)
						case "extra-after-end":
							if !o.complete(0) || !o.begin(1) || !o.complete(1) {
								t.Fatal("normal second member failed")
							}
							result = o.begin(2)
						}
					}
				}
			}
			consumed, sent, next := o.consumed, o.sent, o.next
			if result || o.state != providerHistoryOrderFailed || o.consumeAll() || o.confirmConsumption() || o.begin(next) || o.complete(next) || o.close() || o.consumed != consumed || o.sent != sent || o.next != next {
				t.Fatal("invalid or failed bookkeeping advanced or reset", state)
			}
		})
	}
}

func TestProviderHistoryGroupOrderKeepsAllConsumedAfterFailedMember(t *testing.T) {
	o := historyOrderFixture(t, 32)
	if !o.consumeAll() || !o.confirmConsumption() || !o.begin(0) || !o.complete(0) || !o.begin(1) {
		t.Fatal("normal prefix refused")
	}
	o.refuse()
	if o.consumed != ^uint32(0) || o.sent != 3 || o.next != 1 || o.begin(2) || o.complete(1) || o.close() {
		t.Fatal("failure restored unused members or accepted partial completion")
	}
}

func TestProviderHistoryGroupOrderRejectsUnboundAndCopiedState(t *testing.T) {
	for _, row := range []struct {
		program, table string
		count          int
	}{{"", admissionTestDigest("b"), 1}, {admissionTestDigest("a"), "", 1}, {admissionTestDigest("a"), admissionTestDigest("b"), 0}, {admissionTestDigest("a"), admissionTestDigest("b"), 33}} {
		if o, err := newProviderHistoryGroupOrder(row.program, row.table, row.count); err == nil || o != nil {
			t.Fatal("unbound counter prepared")
		}
	}
	var missing *providerHistoryGroupOrder
	if missing.valid() || missing.consumeAll() || missing.confirmConsumption() || missing.begin(0) || missing.complete(0) || missing.close() || missing.refuse() {
		t.Fatal("missing state became progress")
	}
	original := historyOrderFixture(t, 2)
	copy := *original
	if copy.valid() || copy.consumeAll() || copy.state != providerHistoryOrderPrepared || !original.consumeAll() || json.Unmarshal([]byte(`{}`), original) == nil {
		t.Fatal("copied or JSON state recreated original progress")
	}
	for _, state := range []string{"program", "table", "count", "next-negative", "next-large", "prepared-sent", "consumed-next", "confirmed-mask", "inflight-past-end", "inflight-mask", "closed-incomplete", "unknown"} {
		o := historyOrderFixture(t, 2)
		switch state {
		case "program":
			o.program = ""
		case "table":
			o.table = ""
		case "count":
			o.count = 33
		case "next-negative":
			o.next = -1
		case "next-large":
			o.next = 3
		case "prepared-sent":
			o.sent = 1
		case "consumed-next":
			o.state, o.consumed, o.next = providerHistoryOrderConsumed, 3, 1
		case "confirmed-mask":
			o.state, o.consumed, o.sent = providerHistoryOrderConfirmed, 3, 1
		case "inflight-past-end":
			o.state, o.consumed, o.sent, o.next = providerHistoryOrderInFlight, 3, 3, 2
		case "inflight-mask":
			o.state, o.consumed, o.sent = providerHistoryOrderInFlight, 3, 0
		case "closed-incomplete":
			o.state, o.consumed = providerHistoryOrderClosed, 3
		case "unknown":
			o.state = -1
		}
		if o.valid() || o.consumeAll() || o.begin(0) || o.close() {
			t.Fatal("inconsistent finite state advanced", state)
		}
	}
	for _, count := range []int{-1, 33} {
		if bits, ok := providerHistoryMemberPrefix(count); ok || bits != 0 {
			t.Fatal("invalid prefix produced consumed bits")
		}
	}
}
