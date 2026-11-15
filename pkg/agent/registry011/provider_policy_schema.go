package registry011

// Existing admission callers retain the same Go DATA helper contracts. The
// actual predicates are shared with the owned native policy reader.
func providerCandidateLimits(v any) bool {
	return providerPolicyLimitsValue(legacyProviderPolicyValue{v})
}

func providerCandidateClock(v any) bool {
	return providerPolicyClockValue(legacyProviderPolicyValue{v})
}
