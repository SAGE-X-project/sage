package registry011

// Preflight the complete common quota BEFORE any immutable write or map growth.
// Every typed raw ticket/node/budget/index/page is counted here as an upper;
// retained-original replay independently derives and debits the actual amount
// afterward. An upper never appears as actual measurement or a permission.
func (r *providerFrontierReplay) preflightRetention(s *providerCustodyStore, inventory *providerIntegrationInventory, fresh []providerIntegrationObject, intent, commit, frontier []byte, after providerFrontierHighwater) error {
	base, err := providerFrontierRetentionCostObserved(s.config.identity, inventory, fresh, intent, commit, frontier, s.config.bounds, r.meter)
	m := providerFrontierIntentObjectObserved(intent, r.meter)
	if err != nil || m == nil {
		return errProviderIntegration
	}
	clock, ok := providerStoredClock(m["clock"])
	if !ok {
		return errProviderIntegration
	}
	setup, err := r.setupAtBoundary(s, clock)
	if err != nil {
		return errProviderIntegration
	}
	head, err := r.highwaterRetention(s, after, setup, intent, commit, frontier)
	setupBytes, setupOK := r.setupBytes(s, setup.ordinal)
	metadata, metaOK := providerUpperSum(base.metadata, head.metadata, setupBytes)
	payload := base.payload
	var closingBytes uint64
	if err != nil || !setupOK || !metaOK {
		return errProviderIntegration
	}
	for _, object := range fresh {
		if providerFrontierRetentionClass(object.kind) != 0 {
			continue
		}
		var ok bool
		switch object.kind {
		case "ticket", "consumption", "settlement":
			payload, ok = providerUpperSum(payload, uint64(len(object.raw)))
		default:
			metadata, ok = providerUpperSum(metadata, uint64(len(object.raw)))
		}
		if !ok {
			return errProviderIntegration
		}
		if object.kind == "ledger-page" || object.kind == "ledger-seal" {
			closingBytes, ok = providerUpperSum(closingBytes, uint64(len(object.raw)))
			if !ok {
				return errProviderIntegration
			}
		}
	}
	kind := contentString(m["kind"])
	ledger := r.body.ledger
	bounds := *s.config.bounds.continuation
	if ledger == nil {
		if kind != "enroll" && kind != "admit" {
			return errProviderIntegration
		}
		priorMetadata := uint64(0)
		if kind == "admit" {
			priorMetadata, ok = providerUpperSum(r.preI.metadata, r.indexes.schemaBytes)
			if !ok {
				return errProviderIntegration
			}
		}
		page, pageOK := providerLedgerPageUpper(1)
		metadata, metaOK = providerUpperSum(priorMetadata, metadata, page, providerLedgerSealLimit)
		payload, ok = providerUpperSum(r.preI.payload, payload)
		if !pageOK || !metaOK || !ok || metadata > bounds.maximumMetadataBytes || metadata > bounds.pageMetadataBytes || payload > bounds.maximumPayloadBytes || payload > bounds.pagePayloadBytes {
			return errProviderIntegration
		}
		return nil
	}
	c := ledger.continuation
	if !ledger.valid() || c == nil || !c.valid(ledger) || c.final || c.pending != nil || !c.payloadRoom(ledger, payload) {
		return errProviderIntegration
	}
	switch kind {
	case "admit", "consume", "settle":
		if uint64(len(c.records)) >= c.bounds.pageRecords || !c.metadataRoom(ledger, uint64(len(c.records))+1, metadata) {
			return errProviderIntegration
		}
	case "seal":
		if payload != 0 || !providerFrontierSealQuotaFits(ledger, metadata, closingBytes) {
			return errProviderIntegration
		}
	default:
		return errProviderIntegration
	}
	return nil
}

func providerFrontierSealQuotaFits(ledger *CandidateAdmissionLedger, metadata, closingBytes uint64) bool {
	if ledger == nil || !ledger.valid() || ledger.continuation == nil || closingBytes == 0 || closingBytes > metadata {
		return false
	}
	c := ledger.continuation
	// The actual single custody/index debit still requires its reserved future
	// page+seal upper until applyProviderBoundary completes. Match that SAME
	// precondition before writes; exact closing bytes do not release it early.
	if !c.metadataRoom(ledger, uint64(len(c.records)), metadata-closingBytes) {
		return false
	}
	global, a := providerUpperSum(ledger.current.metadata, ledger.current.privateMetadata, metadata)
	active, b := providerUpperSum(ledger.current.metadata-c.sealedMetadata, c.extraMetadata-c.sealedExtra, c.auxiliaryMetadata-c.sealedAuxiliary, metadata)
	return a && b && global <= c.bounds.maximumMetadataBytes && active <= c.bounds.pageMetadataBytes
}
