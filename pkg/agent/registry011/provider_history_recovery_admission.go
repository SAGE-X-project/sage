package registry011

// Immutable original T inputs, sealed by the healthy writer before its first
// irreversible write. They select the independent parser, not an execution
// authority or the failed coordinator's mutable request table.
type providerHistoryRecoveryAdmissionDescriptor struct {
	owner      *providerHistoryRecoveryAdmissionDescriptor
	recovery   *providerHistoryRecoveryOwner
	checkpoint *providerHistoryRecoveryCheckpoint
	purpose    string
	table      string
	members    int
	upper      providerHistoryGroupUpper
	clock      CandidateClockDiagnostic
	requests   providerHistoryStoredReference
	ticket     providerHistoryStoredReference
}

func (*providerHistoryRecoveryAdmissionDescriptor) UnmarshalJSON([]byte) error {
	return errProviderIntegration
}

func (d *providerHistoryRecoveryAdmissionDescriptor) shapeLocked(r *providerHistoryRecoveryOwner, c *providerHistoryRecoveryCheckpoint) bool {
	if d == nil || d.owner != d || d.recovery != r || c == nil || c.admission != d || d.checkpoint == nil || d.checkpoint.owner != d.checkpoint ||
		d.checkpoint.recovery != r || d.checkpoint.admission != d || d.checkpoint.frame.phase != "admit" ||
		d.checkpoint.frame.stage != c.frame.stage || d.checkpoint.inventory.count != 2 ||
		d.requests != d.checkpoint.inventory.entries[0] || d.ticket != d.checkpoint.inventory.entries[1] ||
		d.requests.kind != "requests" || d.ticket.kind != "ticket" || !lowerHash(d.table) ||
		!lowerHash(d.requests.digest) || !lowerHash(d.ticket.digest) || d.requests.bytes == 0 || d.requests.bytes > providerHistoryOriginalPackBytes ||
		d.ticket.bytes == 0 || d.ticket.bytes > candidateLedgerMetadataLimit || !d.upper.valid(d.members) || d.clock != d.checkpoint.frame.clock {
		return false
	}
	switch d.purpose {
	case "selection", "identity", "ancestry", "content", "effects", "state", "history", "final":
	default:
		return false
	}
	if c.frame.phase == "admit" {
		return d.checkpoint == c
	}
	return c.previous != nil && c.previous.admission == d && (c.frame.phase == "consume" || c.frame.phase == "settle")
}

func (r *providerHistoryRecoveryOwner) prepareAdmissionDescriptorLocked(c *providerHistoryWriteClaim, n *providerHistoryRecoveryCheckpoint) bool {
	if c.frame.phase != "admit" {
		n.admission = n.previous.admission
		return n.admission.shapeLocked(r, n)
	}
	t := c.coordinator.ticket
	if t == nil || c.count != 2 || c.members != t.table.count || c.purpose != t.table.purpose || c.upper != t.upper ||
		c.request != c.inventory.entries[0] || c.ticket != c.inventory.entries[1] || c.ticket.digest != t.digest ||
		c.frame.stage != t.table.stage || !c.upper.valid(c.members) {
		return false
	}
	d := &providerHistoryRecoveryAdmissionDescriptor{recovery: r, checkpoint: n, members: c.members, upper: c.upper,
		clock: c.frame.clock, requests: c.request, ticket: c.ticket}
	d.owner = d
	d.purpose = providerHistoryRecoveryCloneObserved(c.purpose, c.meter)
	d.table = providerHistoryRecoveryCloneObserved(t.table.digest, c.meter)
	if d.purpose == "" || d.table == "" {
		return false
	}
	n.admission = d
	return d.shapeLocked(r, n)
}
