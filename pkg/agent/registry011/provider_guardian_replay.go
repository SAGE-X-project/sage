package registry011

// Independently reconstructed DATA state belonging to one typed replay.
// Never imported from the coordinator and never used to restore a live owner.
type providerGuardianReplay struct {
	head               providerGuardianHeadData
	metadata           uint64
	ticket             string
	consumed, terminal bool
	failed             bool
}

func (*providerGuardianReplay) UnmarshalJSON([]byte) error { return errProviderIntegration }

func (r *providerCustodyReplay) guardianRawBytes() uint64 {
	if r == nil || r.guardian == nil {
		return 0
	}
	return r.guardian.metadata
}

// First I is already bound to original G0 by admissionScanSeed. This records
// the initial first T in that independently derived cache; no later G claim
// is fabricated for that special, separately prepaid first operation.
func (r *providerCustodyReplay) guardianFirstTicket(s *providerCustodyStore, ticket *CandidateAdmissionTicket, data *providerGuardianInitialData) error {
	if r == nil || s == nil || s.config.bounds.guardian == nil || r.guardian != nil || r.intent == nil || r.ledger != nil || ticket == nil || ticket.intent != r.intent || ticket.sequence != 0 || data == nil || data.head.ordinal != 0 || data.head.clock != r.intent.clock || data.head.e0 != s.tip || data.head.setup != s.setupTip {
		return errProviderIntegration
	}
	r.guardian = &providerGuardianReplay{head: data.head, metadata: data.metadata, ticket: ticket.digest}
	return nil
}

// A later admit has exactly one NEW binding in this inventory. Reading an old
// global object with the same kind is not sufficient. The G chain advances by
// one from this replay's own old head and immediately prior main boundary.
func (r *providerCustodyReplay) guardianTicket(s *providerCustodyStore, ticket *CandidateAdmissionTicket) error {
	if r == nil || r.guardian == nil || r.guardian.failed || !r.guardian.terminal || r.intent == nil || r.ledger == nil ||
		ticket == nil || ticket.intent != r.intent || ticket.sequence == 0 || s == nil || s.config.bounds.guardian == nil ||
		r.frontier == nil || r.frontier.body != r || r.frontier.current.commit != s.tip || r.frontier.current.sequence != s.sequence ||
		r.meter == nil || r.meter.closed || r.meter.failed || s.activeMeter.Load() != r.meter {
		return errProviderIntegration
	}
	var digest string
	if len(s.newReferences) == 0 || len(s.newReferences) > s.config.bounds.maximumEntries {
		return errProviderIntegration
	}
	for _, ref := range s.newReferences {
		if !r.meter.event(providerCompareEvent, uint64(len(ref.kind))) {
			return errProviderIntegration
		}
		if ref.kind == "guardian-admission-binding" {
			if digest != "" || !lowerHash(ref.digest) {
				return errProviderIntegration
			}
			digest = ref.digest
		}
	}
	raw, readErr := s.object("guardian-admission-binding", digest)
	x, codecErr := newProviderScanCodec(s.config, r.meter)
	if readErr != nil || codecErr != nil {
		return errProviderIntegration
	}
	binding, err := x.parseGuardianAdmissionBinding(raw)
	if err != nil || binding.ordinal != r.guardian.head.ordinal+1 || binding.ticket != ticket.digest || binding.admission != r.intent.digest || binding.clock != r.intent.clock || binding.input != ticket.input || binding.kind != ticket.kind || binding.method != ticket.method || binding.params != ticket.params {
		return errProviderIntegration
	}
	ordinal, ok := providerUpperSum(s.sequence, uint64(max(0, r.frontier.setupOrdinal)))
	if !ok {
		return errProviderIntegration
	}
	main, _, err := s.originalHighwaterObserved(ordinal, r.meter)
	if err != nil || main.commit != s.tip || main.frontier != binding.frontier || main.setup != binding.setup || main.sequence != s.sequence {
		return errProviderIntegration
	}
	data, err := providerGuardianClaimOriginals(s, r.meter, r.guardian.head, main, r.intent.digest, binding.target, binding.purpose)
	if err != nil {
		return errProviderIntegration
	}
	expected, err := providerGuardianBindingForTicket(x, data, ticket)
	metadata, sumOK := providerUpperSum(r.guardian.metadata, data.metadata)
	if err != nil || expected.digest != digest || !providerReplayPhaseRawTextEqual(raw, expected.raw, r.meter) || !sumOK || metadata > s.config.bounds.continuation.maximumMetadataBytes || r.ledger.addProviderRetention(r.ledger.current, 0, data.metadata) != nil {
		return errProviderIntegration
	}
	// A failed subsequent main transition poisons this whole replay. This
	// provisional cache is never returned as a retained ACK by itself.
	r.guardian.head, r.guardian.metadata = data.head, metadata
	r.guardian.ticket, r.guardian.consumed, r.guardian.terminal = ticket.digest, false, false
	return nil
}

func (r *providerCustodyReplay) guardianConsumption(consumption *CandidateConsumption) bool {
	if r == nil {
		return false
	}
	g := r.guardian
	if g == nil || g.failed || g.terminal || g.consumed || consumption == nil || consumption.ticket == nil || consumption.ticket.digest != g.ticket {
		return false
	}
	g.consumed = true
	return true
}

func (r *providerCustodyReplay) guardianSettlement(settlement *CandidateAdmissionSettlement) bool {
	if r == nil {
		return false
	}
	g := r.guardian
	if g == nil || g.failed || g.terminal || !g.consumed || settlement == nil || settlement.ticket == nil || settlement.ticket.digest != g.ticket {
		return false
	}
	g.terminal = true
	g.failed = settlement.outcome != "complete"
	return true
}
