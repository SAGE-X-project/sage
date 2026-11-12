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
