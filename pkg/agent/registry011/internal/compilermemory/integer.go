package compilermemory

import (
	"errors"
	"math"
)

var ErrJSONInteger = errors.New("compiler memory: invalid JSON integer")

// JSONUint64 reads one exact JSON integer token directly from owned storage.
// The span must exclude surrounding whitespace and delimiters. No native slice
// escapes, no intermediate string is made, and Close cannot unmap the token
// during the read. This does not qualify a complete parser or compiler.
func (r *Region) JSONUint64(s Span) (uint64, error) {
	if r == nil || r.self != r {
		return 0, ErrConfiguration
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := r.accessError(); err != nil {
		return 0, err
	}
	if !r.contains(s) {
		return 0, ErrRange
	}
	value, valid := jsonIntegerMagnitude(r.data[s.offset:s.offset+s.length], math.MaxUint64)
	if err := r.actionError(); err != nil {
		return 0, err
	}
	if !valid {
		return 0, ErrJSONInteger
	}
	return value, nil
}

// JSONInt64 accepts the canonical integer spelling used by the preparation
// schema, including MinInt64. Negative zero, leading zeros, fractions and
// exponents are refused; larger numeric DATA never becomes an execution grant.
func (r *Region) JSONInt64(s Span) (int64, error) {
	if r == nil || r.self != r {
		return 0, ErrConfiguration
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := r.accessError(); err != nil {
		return 0, err
	}
	if !r.contains(s) {
		return 0, ErrRange
	}
	raw := r.data[s.offset : s.offset+s.length]
	negative := len(raw) > 0 && raw[0] == '-'
	limit := uint64(math.MaxInt64)
	if negative {
		raw = raw[1:]
		limit++
	}
	magnitude, valid := jsonIntegerMagnitude(raw, limit)
	if err := r.actionError(); err != nil {
		return 0, err
	}
	if !valid || negative && magnitude == 0 {
		return 0, ErrJSONInteger
	}
	if negative && magnitude == uint64(math.MaxInt64)+1 {
		return math.MinInt64, nil
	}
	// Keep the signed conversion's bound explicit for this selected build.
	if magnitude > math.MaxInt64 {
		return 0, ErrJSONInteger
	}
	if negative {
		return -int64(magnitude), nil
	}
	return int64(magnitude), nil
}

func jsonIntegerMagnitude(raw []byte, limit uint64) (uint64, bool) {
	if len(raw) == 0 || len(raw) > 1 && raw[0] == '0' {
		return 0, false
	}
	var value uint64
	for _, b := range raw {
		if b < '0' || b > '9' {
			return 0, false
		}
		digit := uint64(b - '0')
		if value > limit/10 || value == limit/10 && digit > limit%10 {
			return 0, false
		}
		value = value*10 + digit
	}
	return value, true
}
