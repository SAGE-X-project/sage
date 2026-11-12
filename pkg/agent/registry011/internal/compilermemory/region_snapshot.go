package compilermemory

// SealSnapshot seals the original region and returns a checked DATA span over
// exactly its committed bytes. The capacity and unused mapping are excluded.
// A retained record reader can derive bounded fields with Slice without keeping
// a Go span table or exposing native aliases. Failure returns no snapshot even
// if sealing already occurred; it does not undo sealing or renew the lifetime.
func (r *Region) SealSnapshot() (Span, error) {
	if err := r.Seal(); err != nil {
		return Span{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := r.accessError(); err != nil {
		return Span{}, err
	}
	return Span{owner: r, length: r.used}, nil
}
