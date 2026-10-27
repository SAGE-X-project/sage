package registry011

import "testing"

// These are finite DATA comparisons under one isolated observer. Passing this
// predicate does not install a prefix, retain a transition or supply a permit.
func TestProviderContinuingDeltaRejectsDifferentOwnedModelsAndChangedValues(t *testing.T) {
	s, o := providerContinuingPreparedFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	p, r, m := o.indexes, s.frontierReplay.indexes, o.execution
	base := providerIndexDelta{owner: p, ledger: p.ledger, used: true, mutations: []providerIndexMutation{
		{chain: 0, key: o.ticket.digest}, {chain: 1, key: o.ticket.digest},
		{chain: 2, key: candidateAdmissionSegment(o.ticket.intent.digest, o.ticket.segment)},
	}}
	if !providerContinuingDeltaEqual(s, p, r, &base, o.ticket, m) {
		t.Fatal("independent retained T models disagreed")
	}
	for _, mode := range []string{"owner", "delta", "unused", "too-many", "start", "budget", "consumed-shape", "consumed-bytes", "ticket-index", "operation-index", "segment-index", "index-kind", "ticket-state", "page-record"} {
		t.Run(mode, func(t *testing.T) {
			delta := base
			delta.mutations = append([]providerIndexMutation(nil), base.mutations...)
			selected, change := p, &delta
			switch mode {
			case "owner":
				selected = nil
			case "delta":
				change = nil
			case "unused":
				delta.used = false
			case "too-many":
				delta.mutations = append(delta.mutations, delta.mutations[0])
			case "start":
				before := r.ledger.providerStart
				defer func() { r.ledger.providerStart = before }()
				r.ledger.providerStart = admissionTestDigest("f")
			case "budget":
				before := r.ledger.current.raw
				defer func() { r.ledger.current.raw = before }()
				r.ledger.current.raw = "different finite budget DATA"
			case "consumed-shape":
				before := r.ledger.lastConsumption
				defer func() { r.ledger.lastConsumption = before }()
				r.ledger.lastConsumption = nil
			case "consumed-bytes":
				before := r.ledger.lastConsumption.raw
				defer func() { r.ledger.lastConsumption.raw = before }()
				r.ledger.lastConsumption.raw = "different finite consumption DATA"
			case "ticket-index":
				delta.mutations[0].key = admissionTestDigest("f")
			case "operation-index":
				delta.mutations[1].key = admissionTestDigest("f")
			case "segment-index":
				delta.mutations[2].key = admissionTestDigest("f")
			case "index-kind":
				delta.mutations[0].chain = 3
			case "ticket-state":
				before := r.ledger.tickets[o.ticket.digest]
				defer func() { r.ledger.tickets[o.ticket.digest] = before }()
				changed := before
				changed.terminal = admissionTestDigest("f")
				r.ledger.tickets[o.ticket.digest] = changed
			case "page-record":
				page := r.ledger.continuation
				last := len(page.records) - 1
				before := page.records[last]
				defer func() { page.records[last] = before }()
				page.records[last].budget = admissionTestDigest("f")
			}
			if providerContinuingDeltaEqual(s, selected, r, change, o.ticket, m) {
				t.Fatal("changed owned model or affected value was accepted", mode)
			}
		})
	}
	if !o.guardian.continuingPrefixMatchesLocked() || s.sequence != 5 || o.permit != nil || o.consumption || m.actual[0] != 0 || m.actual[1] != 0 || m.actual[9] != 0 {
		t.Fatal("finite DATA comparison issued work or changed the original prefix")
	}
}
