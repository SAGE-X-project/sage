package registry011

func providerCandidateInstance(v any) bool {
	m := contentObject(v, "chain_id", "genesis_hash", "registry_id", "deployment_hash")
	// This parser supports only the existing approved Sepolia instance. The
	// deployment block hash is pinned data, not a new historical-code proof.
	return m != nil && m["chain_id"] == "11155111" && m["genesis_hash"] == sepoliaGenesis &&
		m["registry_id"] == "eip155:11155111:"+SepoliaRegistryAddress &&
		m["deployment_hash"] == "0x54138f0dbccac32a1cd6fb0ed049fa013e78ca33910dd0471b09efea663d365e"
}

func providerCandidateOrigin(v any) bool {
	m := contentObject(v, "endpoint", "transport_artifact", "tls_roots_artifact", "redirect_policy", "proxy_policy")
	return m != nil && m["endpoint"] == SepoliaRPCURL && m["redirect_policy"] == "deny" &&
		m["proxy_policy"] == "none" && providerCandidateDigest(m["transport_artifact"]) && providerCandidateDigest(m["tls_roots_artifact"])
}

func providerCandidateFormats(m map[string]any) bool {
	f := contentObject(m["finality"], "selector", "basis", "validation_artifact")
	t := contentObject(m["trace_format"], "normalizer_artifact", "acquisition_artifact")
	r := contentObject(m["retention"], "format", "authority_artifact", "store_artifact", "acknowledgement_artifact")
	a := contentObject(m["assumptions"], "provider_execution", "provider_finality", "historical_code", "agent_boundary", "clock", "retaining_authority")
	c := contentObject(m["claims"], "gas_decomposition", "independent_consensus", "compromised_trusted_boundary")
	carriers, ok := m["carrier_formats"].([]any)
	return f != nil && f["selector"] == "finalized" && f["basis"] == "trusted-endpoint-finalized" && providerCandidateDigest(f["validation_artifact"]) &&
		t != nil && providerCandidateDigest(t["normalizer_artifact"]) && providerCandidateDigest(t["acquisition_artifact"]) &&
		r != nil && r["format"] == "sage-provider-retention-v1" && providerCandidateDigest(r["authority_artifact"]) && providerCandidateDigest(r["store_artifact"]) && providerCandidateDigest(r["acknowledgement_artifact"]) &&
		a != nil && a["provider_execution"] == "trusted-provider-execution" && a["provider_finality"] == "trusted-provider-finality" && a["historical_code"] == "qualified-fixed-code-closure" &&
		a["agent_boundary"] == "trusted-agent-integration" && a["clock"] == "qualified-local-clock" && a["retaining_authority"] == "qualified-retaining-authority" &&
		c != nil && c["gas_decomposition"] == "not-verified" && c["independent_consensus"] == "not-verified" && c["compromised_trusted_boundary"] == "not-verified" &&
		ok && len(carriers) == 3 && carriers[0] == "protected-legacy" && carriers[1] == "access-list" && carriers[2] == "dynamic-fee"
}

// These are parsing relations and supported operation ceilings only. No
// artifact, work estimator, clock, store or workload feasibility is qualified.
func providerCandidateLimits(v any) bool {
	m := contentObject(v, "operation", "segment", "bootstrap", "live_disk_bytes", "retained_bytes", "lifetime_claims", "lifetime_identities")
	if m == nil {
		return false
	}
	for _, key := range []string{"live_disk_bytes", "retained_bytes", "lifetime_claims", "lifetime_identities"} {
		if _, ok := providerCandidateUint(m[key], true); !ok {
			return false
		}
	}
	names := []string{"requests", "wire_bytes", "decoded_bytes", "blocks", "transactions", "frames", "logs", "proofs", "segments", "disk_bytes_written", "work_units", "deadline_ms"}
	var previous map[string]uint64
	for _, level := range []string{"operation", "segment", "bootstrap"} {
		values := contentObject(m[level], names...)
		if values == nil {
			return false
		}
		current := make(map[string]uint64, len(names))
		for _, name := range names {
			n, ok := providerCandidateUint(values[name], true)
			if !ok || n < previous[name] {
				return false
			}
			current[name] = n
		}
		if level == "operation" && (current["requests"] > 65536 || current["wire_bytes"] > 256<<20 || current["decoded_bytes"] > 256<<20 || current["deadline_ms"] > 300000) {
			return false
		}
		previous = current
	}
	return true
}

func providerCandidateClock(v any) bool {
	m := contentObject(v, "id", "artifact", "max_uncertainty_ms", "max_read_window_ms", "max_observation_age_ms", "max_finalized_lag_ms")
	if m == nil || !providerCandidateDigest(m["artifact"]) {
		return false
	}
	id, ok := m["id"].(string)
	if !ok || len(id) == 0 || len(id) > 256 {
		return false
	}
	for _, b := range []byte(id) {
		if b < 33 || b > 126 {
			return false
		}
	}
	for _, key := range []string{"max_uncertainty_ms", "max_read_window_ms", "max_observation_age_ms", "max_finalized_lag_ms"} {
		if _, ok := providerCandidateUint(m[key], true); !ok {
			return false
		}
	}
	return true
}
