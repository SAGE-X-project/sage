package registry011

// providerPolicyValue supplies semantic reads, not a selected policy or a grant.
// Both DATA producers share these predicates. Implementations are package-owned
// and this interface is never accepted by a public constructor.
type providerPolicyValue interface {
	object(names ...string) bool
	member(name string) providerPolicyValue
	arrayLength() (int, bool)
	element(index int) providerPolicyValue
	equals(text string) bool
	text(destination []byte) (int, bool)
	decimal(positive bool) (uint64, bool)
}

// This is explicitly the ordinary Go DATA adapter. Native input must not be
// projected into a Go map/string tree merely to invoke it.
type legacyProviderPolicyValue struct{ value any }

func (v legacyProviderPolicyValue) object(names ...string) bool {
	return contentObject(v.value, names...) != nil
}

func (v legacyProviderPolicyValue) member(name string) providerPolicyValue {
	m, _ := v.value.(map[string]any)
	return legacyProviderPolicyValue{m[name]}
}

func (v legacyProviderPolicyValue) arrayLength() (int, bool) {
	a, ok := v.value.([]any)
	return len(a), ok
}

func (v legacyProviderPolicyValue) element(index int) providerPolicyValue {
	a, ok := v.value.([]any)
	if !ok || index < 0 || index >= len(a) {
		return legacyProviderPolicyValue{}
	}
	return legacyProviderPolicyValue{a[index]}
}

func (v legacyProviderPolicyValue) equals(text string) bool {
	s, ok := v.value.(string)
	return ok && s == text
}

func (v legacyProviderPolicyValue) text(destination []byte) (int, bool) {
	s, ok := v.value.(string)
	if !ok || len(s) > len(destination) {
		return 0, false
	}
	copy(destination, s)
	return len(s), true
}

func (v legacyProviderPolicyValue) decimal(positive bool) (uint64, bool) {
	return providerCandidateUint(v.value, positive)
}

func providerPolicyFieldsValid(v providerPolicyValue) bool {
	return v.object("format", "spec_revision", "implementation_artifact", "instance", "provider", "finality", "trace_format", "carrier_formats", "fork_artifact", "limits", "clock", "retention", "assumptions", "claims") &&
		v.member("format").equals("sage-provider-registry-observation-v1") && v.member("spec_revision").equals("a22a275b4d88f1d415b53cdffc5bed41231b77a0") &&
		providerPolicyDigestValue(v.member("implementation_artifact")) && providerPolicyDigestValue(v.member("fork_artifact")) &&
		providerPolicyInstanceValue(v.member("instance")) && providerPolicyOriginValue(v.member("provider")) &&
		providerPolicyFormatsValue(v) && providerPolicyLimitsValue(v.member("limits")) && providerPolicyClockValue(v.member("clock"))
}

func providerPolicyDigestValue(v providerPolicyValue) bool {
	var raw [66]byte
	n, ok := v.text(raw[:])
	if !ok || n != len(raw) || raw[0] != '0' || raw[1] != 'x' {
		return false
	}
	for _, b := range raw[2:] {
		if (b < '0' || b > '9') && (b < 'a' || b > 'f') {
			return false
		}
	}
	return true
}

func providerPolicyInstanceValue(v providerPolicyValue) bool {
	return v.object("chain_id", "genesis_hash", "registry_id", "deployment_hash") && v.member("chain_id").equals("11155111") &&
		v.member("genesis_hash").equals(sepoliaGenesis) && v.member("registry_id").equals("eip155:11155111:"+SepoliaRegistryAddress) &&
		v.member("deployment_hash").equals("0x54138f0dbccac32a1cd6fb0ed049fa013e78ca33910dd0471b09efea663d365e")
}

func providerPolicyOriginValue(v providerPolicyValue) bool {
	return v.object("endpoint", "transport_artifact", "tls_roots_artifact", "redirect_policy", "proxy_policy") &&
		v.member("endpoint").equals(SepoliaRPCURL) && v.member("redirect_policy").equals("deny") && v.member("proxy_policy").equals("none") &&
		providerPolicyDigestValue(v.member("transport_artifact")) && providerPolicyDigestValue(v.member("tls_roots_artifact"))
}

func providerPolicyFormatsValue(v providerPolicyValue) bool {
	f, trace, retention := v.member("finality"), v.member("trace_format"), v.member("retention")
	a, claims, carriers := v.member("assumptions"), v.member("claims"), v.member("carrier_formats")
	length, array := carriers.arrayLength()
	return f.object("selector", "basis", "validation_artifact") && f.member("selector").equals("finalized") && f.member("basis").equals("trusted-endpoint-finalized") && providerPolicyDigestValue(f.member("validation_artifact")) &&
		trace.object("normalizer_artifact", "acquisition_artifact") && providerPolicyDigestValue(trace.member("normalizer_artifact")) && providerPolicyDigestValue(trace.member("acquisition_artifact")) &&
		retention.object("format", "authority_artifact", "store_artifact", "acknowledgement_artifact") && retention.member("format").equals("sage-provider-retention-v1") &&
		providerPolicyDigestValue(retention.member("authority_artifact")) && providerPolicyDigestValue(retention.member("store_artifact")) && providerPolicyDigestValue(retention.member("acknowledgement_artifact")) &&
		a.object("provider_execution", "provider_finality", "historical_code", "agent_boundary", "clock", "retaining_authority") &&
		a.member("provider_execution").equals("trusted-provider-execution") && a.member("provider_finality").equals("trusted-provider-finality") && a.member("historical_code").equals("qualified-fixed-code-closure") &&
		a.member("agent_boundary").equals("trusted-agent-integration") && a.member("clock").equals("qualified-local-clock") && a.member("retaining_authority").equals("qualified-retaining-authority") &&
		claims.object("gas_decomposition", "independent_consensus", "compromised_trusted_boundary") && claims.member("gas_decomposition").equals("not-verified") &&
		claims.member("independent_consensus").equals("not-verified") && claims.member("compromised_trusted_boundary").equals("not-verified") &&
		array && length == 3 && carriers.element(0).equals("protected-legacy") && carriers.element(1).equals("access-list") && carriers.element(2).equals("dynamic-fee")
}

// These are the original parsing relations and supported operation ceilings.
// A successful reader does not fund a budget or select an operating policy.
func providerPolicyLimitsValue(v providerPolicyValue) bool {
	if !v.object("operation", "segment", "bootstrap", "live_disk_bytes", "retained_bytes", "lifetime_claims", "lifetime_identities") {
		return false
	}
	for _, name := range [...]string{"live_disk_bytes", "retained_bytes", "lifetime_claims", "lifetime_identities"} {
		if _, ok := v.member(name).decimal(true); !ok {
			return false
		}
	}
	names := [...]string{"requests", "wire_bytes", "decoded_bytes", "blocks", "transactions", "frames", "logs", "proofs", "segments", "disk_bytes_written", "work_units", "deadline_ms"}
	var previous [len(names)]uint64
	for levelIndex, level := range [...]string{"operation", "segment", "bootstrap"} {
		values := v.member(level)
		if !values.object(names[:]...) {
			return false
		}
		var current [len(names)]uint64
		for index, name := range names {
			n, ok := values.member(name).decimal(true)
			if !ok || n < previous[index] {
				return false
			}
			current[index] = n
		}
		if levelIndex == 0 && (current[0] > 65536 || current[1] > 256<<20 || current[2] > 256<<20 || current[11] > 300000) {
			return false
		}
		previous = current
	}
	return true
}

func providerPolicyClockValue(v providerPolicyValue) bool {
	if !v.object("id", "artifact", "max_uncertainty_ms", "max_read_window_ms", "max_observation_age_ms", "max_finalized_lag_ms") || !providerPolicyDigestValue(v.member("artifact")) {
		return false
	}
	var id [256]byte
	n, ok := v.member("id").text(id[:])
	if !ok || n == 0 {
		return false
	}
	for _, b := range id[:n] {
		if b < 33 || b > 126 {
			return false
		}
	}
	for _, name := range [...]string{"max_uncertainty_ms", "max_read_window_ms", "max_observation_age_ms", "max_finalized_lag_ms"} {
		if _, ok := v.member(name).decimal(true); !ok {
			return false
		}
	}
	return true
}
