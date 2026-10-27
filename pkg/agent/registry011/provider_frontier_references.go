package registry011

import "bytes"

// Cached private maps are valid only under the selected append-only custodian
// assumptions. Every transitive old payload/head used in a new transition is
// still read from its retained original object and checked against that state.
// A matching map value never substitutes for available exact original bytes.
func (r *providerFrontierReplay) checkOldMaterials(s *providerCustodyStore, kind, operation string) error {
	for index, head := range r.indexes.heads {
		objectKind := "index-mutation"
		if r.indexes.counts[index] == 0 {
			objectKind = "index-empty"
		}
		if _, err := s.object(objectKind, head); err != nil {
			return errProviderIntegration
		}
	}
	if r.body.anchor != nil {
		original, err := s.object("enrollment", r.body.anchor.digest)
		if err != nil || !bytes.Equal(original, []byte(r.body.anchor.raw)) {
			return errProviderIntegration
		}
	} else if kind != "enroll" {
		return errProviderIntegration
	}
	m := r.body.ledger
	if m == nil {
		return nil
	}
	if !m.valid() || r.body.intent == nil || r.body.anchor == nil {
		return errProviderIntegration
	}
	for _, ref := range []struct{ kind, digest, raw string }{
		{"admission", r.body.intent.digest, r.body.intent.raw},
		{"budget", m.current.digest, m.current.raw},
	} {
		original, err := s.object(ref.kind, ref.digest)
		if err != nil || !bytes.Equal(original, []byte(ref.raw)) {
			return errProviderIntegration
		}
	}
	for _, head := range m.current.heads {
		if _, err := s.object("ledger-node", head); err != nil {
			return errProviderIntegration
		}
	}
	var state candidateLedgerTicket
	switch kind {
	case "admit":
		if m.lastTicket == nil {
			return errProviderIntegration
		}
		state = m.tickets[m.lastTicket.digest]
	case "consume", "settle":
		objectKind := "consumption"
		if kind == "settle" {
			objectKind = "settlement"
		}
		payload, err := s.object(objectKind, operation)
		value, ok := contentJSON(payload).(map[string]any)
		if err != nil || !ok {
			return errProviderIntegration
		}
		var exists bool
		state, exists = m.tickets[contentString(value["ticket_digest"])]
		if !exists {
			return errProviderIntegration
		}
	default:
		return nil
	}
	if state.ticket == nil {
		return errProviderIntegration
	}
	original, err := s.object("ticket", state.ticket.digest)
	if err != nil || !bytes.Equal(original, []byte(state.ticket.raw)) || r.body.checkTicketBytes(s, state.ticket) != nil {
		return errProviderIntegration
	}
	if state.consumption != nil {
		original, err := s.object("consumption", state.consumption.digest)
		if err != nil || !bytes.Equal(original, []byte(state.consumption.raw)) {
			return errProviderIntegration
		}
	}
	if kind == "consume" && m.lastConsumption != nil {
		original, err := s.object("consumption", m.lastConsumption.digest)
		if err != nil || !bytes.Equal(original, []byte(m.lastConsumption.raw)) {
			return errProviderIntegration
		}
	}
	return nil
}
