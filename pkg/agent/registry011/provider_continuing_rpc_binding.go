package registry011

// Reconstruct only the new original binding under the current R observer.
// Old binding/index maps are neither imported, copied, nor scanned.
func (o *providerContinuingOwner) bindRetainedExchangeLocked(s *providerCustodyStore, fresh []providerIntegrationObject) error {
	if o == nil || !o.selectedLocked() || o.store != s || o.state != providerContinuingSettlement ||
		!o.consumption || !o.settlement || o.ticket == nil || o.ticket.kind != "rpc-capture" || o.ticket.sequence == 0 ||
		s.sequence != o.before.sequence+2 || s.activeMeter.Load() != o.remaining || len(fresh) != 11 || s.bindings == nil {
		return errProviderIntegration
	}
	binding, exchange, err := providerFirstOperationExchangeReferences(fresh)
	if err != nil {
		return errProviderIntegration
	}
	if _, exists := s.bindings[exchange]; exists {
		return errProviderIntegration
	}
	raw, err := s.object("wire-binding", binding)
	if err != nil || providerFirstOperationBindingExchange(raw, exchange, o.remaining) != nil {
		return errProviderIntegration
	}
	s.bindings[exchange] = binding
	return nil
}
