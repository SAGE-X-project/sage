package registry011

import (
	"bytes"
	"strconv"
)

// A concrete codec, not an injected callback. A nil meter is pure candidate
// DATA use only. The live control path installs its prepaid owner meter first.
type providerScanCodec struct {
	config providerCustodyConfig
	meter  *providerWorkMeter
}

func newProviderScanCodec(config providerCustodyConfig, meter *providerWorkMeter) (*providerScanCodec, error) {
	id := config.identity
	if config.bounds.scan == nil || !config.bounds.valid() || !config.capture.valid() || config.policy == nil || config.policy.raw == "" || id.Policy != config.policy.policy || id.Instance != config.policy.instance || id.Provider != config.policy.provider || id.Generation != 0 || !lowerHash(id.Namespace) || !lowerHash(id.SourceID) || !lowerHash(id.Policy) || !lowerHash(id.Instance) || !lowerHash(id.Provider) || !lowerHash(config.profile) {
		return nil, errProviderIntegration
	}
	x := &providerScanCodec{config, meter}
	// The selected finite schema is bounded before its maps/encoding scratch.
	if !x.preflight(providerEncodeEvent, config.bounds.maximumMetadataPerCommit) {
		return nil, errProviderIntegration
	}
	raw, domain := providerIntegrationProfileBytes(config.bounds, config.capture)
	if !x.event(providerEncodeEvent, uint64(len(raw))) || len(raw) == 0 || uint64(len(raw)) > config.bounds.maximumMetadataPerCommit || x.hash(domain, raw) != config.profile {
		return nil, errProviderIntegration
	}
	return x, nil
}

func (x *providerScanCodec) event(kind providerWorkEvent, n uint64) bool {
	return x.meter == nil || x.meter.event(kind, n)
}
func (x *providerScanCodec) preflight(kind providerWorkEvent, n uint64) bool {
	return x.meter == nil || x.meter.preflight(kind, n)
}
func (x *providerScanCodec) identity(domain string) map[string]any {
	id := x.config.identity
	return map[string]any{"format": domain, "namespace": id.Namespace, "instance_key": id.Instance, "generation": strconv.FormatUint(id.Generation, 10), "fence": "1", "profile": x.config.profile}
}
func (x *providerScanCodec) encode(value any) ([]byte, error) {
	if value == nil || !x.preflight(providerEncodeEvent, x.config.bounds.maximumMetadataPerCommit) {
		return nil, errProviderIntegration
	}
	raw := providerIntegrationJSON(value)
	if len(raw) == 0 || len(raw) > candidateLedgerMetadataLimit || uint64(len(raw)) > x.config.bounds.maximumMetadataPerCommit || !x.event(providerEncodeEvent, uint64(len(raw))) {
		return nil, errProviderIntegration
	}
	return raw, nil
}
func (x *providerScanCodec) decode(raw []byte) any {
	if len(raw) == 0 || len(raw) > candidateLedgerMetadataLimit || uint64(len(raw)) > x.config.bounds.maximumMetadataPerCommit || !x.event(providerCodecEvent, uint64(len(raw))) {
		return nil
	}
	return contentJSON(raw)
}
func (x *providerScanCodec) hash(domain string, raw []byte) string {
	if !x.event(providerHashEvent, uint64(len(domain)+1+len(raw))) {
		return ""
	}
	return providerCandidateHash(domain, raw)
}
func (x *providerScanCodec) equal(a, b []byte) bool {
	return x.event(providerCompareEvent, uint64(len(a)+len(b))) && bytes.Equal(a, b)
}

func (x *providerScanCodec) deadline(clock CandidateClockDiagnostic) (uint64, uint64, bool) {
	if clock.WallMS < clock.UncertaintyMS {
		return 0, 0, false
	}
	root, ok := x.decode(providerReplayPhaseBytes(x.config.policy.raw, x.meter)).(map[string]any)
	if !ok {
		return 0, 0, false
	}
	limits := contentObject(root["limits"], "operation", "segment", "bootstrap", "live_disk_bytes", "retained_bytes", "lifetime_claims", "lifetime_identities")
	bootstrap, ok := limits["bootstrap"].(map[string]any)
	duration, valid := providerCandidateUint(bootstrap["deadline_ms"], true)
	deadline, sumOK := providerUpperSum(uint64(clock.WallMS-clock.UncertaintyMS), duration)
	return deadline, duration, ok && valid && sumOK
}
