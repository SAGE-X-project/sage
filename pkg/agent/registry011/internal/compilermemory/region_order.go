package compilermemory

// Precedes compares two committed ranges of this original region. It exports
// their ordering only, never offsets or native aliases. A semantic reader uses
// it to check canonical member order without materializing an object tree.
func (r *Region) Precedes(left, right Span) (bool, error) {
	if r == nil || r.self != r {
		return false, ErrConfiguration
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := r.accessError(); err != nil {
		return false, err
	}
	if !r.contains(left) || !r.contains(right) {
		return false, ErrRange
	}
	return left.offset+left.length <= right.offset, nil
}

// SpanLength reports only the size of an original committed range. Consumers
// can stream its bytes through Read without exporting offsets or native aliases.
func (r *Region) SpanLength(span Span) (int, error) {
	if r == nil || r.self != r {
		return 0, ErrConfiguration
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := r.accessError(); err != nil {
		return 0, err
	}
	if !r.contains(span) {
		return 0, ErrRange
	}
	return span.length, nil
}
