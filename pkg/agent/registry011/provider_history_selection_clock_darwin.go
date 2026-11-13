//go:build darwin

package registry011

import "github.com/sage-x-project/sage/pkg/agent/registry011/internal/compilermemory"

// Read the actual validated native policy through its original live owner.
// The common compiler still freezes Go configuration. The clock, its immutable
// text projection and this bounded Go scratch do not gain native RAM backing.
func (v *nativeHistoryOwnedSelection) readOwnedClockPolicy(policy *ProviderPolicyCandidate) (providerOwnedClockPolicyFields, error) {
	var empty providerOwnedClockPolicyFields
	if err := v.check(); err != nil {
		return empty, err
	}
	if policy == nil || v.policy == nil || v.policy.self != v.policy || !v.policy.used.Load() || v.policy.err != nil ||
		v.policy.source != v.preparer.policy.region {
		return empty, ErrConfiguration
	}
	span, err := v.policy.root.Raw()
	if err != nil || span != v.policy.input {
		return empty, nativeHistoryDataError(err)
	}
	// Bind the reader to the same frozen policy passed by the original compiler.
	// This compatibility projection remains Go working memory, not native credit.
	equal, err := v.policy.source.Equal(v.policy.input, []byte(policy.raw))
	if err != nil || !equal {
		return empty, nativeHistoryDataError(err)
	}
	clock, found, err := v.policy.root.Member("clock")
	if err != nil || !found {
		return empty, nativeHistoryDataError(err)
	}
	maximumUncertainty, err := v.nativeClockPolicyDecimal(clock, "max_uncertainty_ms")
	if err != nil {
		return empty, err
	}
	maximumWindow, err := v.nativeClockPolicyDecimal(clock, "max_read_window_ms")
	if err != nil {
		return empty, err
	}
	id, found, err := clock.Member("id")
	if err != nil || !found {
		return empty, nativeHistoryDataError(err)
	}
	length, err := id.DecodedStringLength()
	if err != nil || length == 0 || length > providerHistorySourceFixedTextLimit {
		return empty, nativeHistoryDataError(err)
	}
	var text [providerHistorySourceFixedTextLimit]byte
	if err := id.CopyString(text[:length]); err != nil {
		return empty, nativeHistoryDataError(err)
	}
	if err := v.check(); err != nil {
		return empty, err
	}
	return providerOwnedClockPolicyFields{string(text[:length]), maximumUncertainty, maximumWindow}, nil
}

// The original strict policy validator already required minimal canonical
// decimal strings. Read those actual digit bytes in place, without appending
// a second numeric pass to its one-use temporary number arena.
func (v *nativeHistoryOwnedSelection) nativeClockPolicyDecimal(clock compilermemory.JSONValue, name string) (uint64, error) {
	value, found, err := clock.Member(name)
	if err != nil || !found {
		return 0, nativeHistoryDataError(err)
	}
	length, err := value.DecodedStringLength()
	if err != nil || length == 0 || length > 20 {
		return 0, nativeHistoryDataError(err)
	}
	span, err := value.Raw()
	if err != nil {
		return 0, nativeHistoryDataError(err)
	}
	bytes, err := v.policy.source.SpanLength(span)
	if err != nil || bytes != length+2 {
		return 0, nativeHistoryDataError(err)
	}
	digits, err := v.policy.source.Slice(span, 1, length)
	if err != nil {
		return 0, nativeHistoryDataError(err)
	}
	number, err := v.policy.source.JSONUint64(digits)
	if err != nil || number == 0 {
		return 0, nativeHistoryDataError(err)
	}
	return number, nil
}
