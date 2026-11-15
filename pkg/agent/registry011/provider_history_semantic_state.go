package registry011

// A semantic window owns its copies. Claims/created indexes cover ALL Registry
// subjects and are not reset at content/window boundaries. The queue is separate
// from these lifetime tombstones. No collector/RAM/observer sufficiency is
// implied by this DATA copy; its entire backing graph must be source-funded.
func providerHistoryCloneSemanticState(s replayStream) replayStream {
	next := replayStream{state: s.state, selected: s.selected,
		claims: make(map[replayClaimID]replayClaimState, len(s.claims)), created: make(map[string]bool, len(s.created))}
	next.state.record.Keys = append([]replayKey{}, s.state.record.Keys...)
	for i, k := range next.state.record.Keys {
		if k.Expires != nil {
			x := *k.Expires
			next.state.record.Keys[i].Expires = &x
		}
	}
	next.state.record.Services = append([]replayService{}, s.state.record.Services...)
	next.state.grants = append([]replayGrant{}, s.state.grants...)
	for k, v := range s.claims {
		next.claims[k] = v
	}
	for k, v := range s.created {
		next.created[k] = v
	}
	return next
}
