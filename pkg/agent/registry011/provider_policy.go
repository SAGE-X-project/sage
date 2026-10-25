package registry011

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
)

var ErrProviderPolicyCandidate = errors.New("registry011: invalid provider policy candidate")

// ProviderPolicyCandidate retains a parsed, copied local manifest candidate.
// Its digests establish byte identity only: referenced artifacts, operating
// values, retaining authority and clock are NOT qualified by this parser.
// It cannot be used as a strict replay token, checked provider batch, selected
// policy, enrolled Source or permission to dispatch a protected Agent message.
// The zero value is invalid. There is no selection or activation method.
type ProviderPolicyCandidate struct {
	raw                        string
	policy, instance, provider string
}

// ProviderCandidateIdentity is forgeable diagnostic metadata. Neither this
// value nor a matching digest authenticates local configuration or authority.
type ProviderCandidateIdentity struct {
	Policy, Instance, Provider, Namespace, SourceID string
	Generation                                      uint64
}

// ProviderCandidateContext is still candidate data. Binding a Source ID or
// generation never enrolls an instance or proves authoritative absence. A
// future qualified retaining boundary must independently admit these values.
// Only candidate first-enrollment persistence consumes it as proposed data.
// No qualified Source, replay, clock or protected-use API accepts it.
type ProviderCandidateContext struct {
	identity ProviderCandidateIdentity
	bound    bool
}

// UnmarshalJSON refuses reconstruction from external JSON. Only the bounded
// schema parser may create a candidate, and it still creates no authority.
func (*ProviderPolicyCandidate) UnmarshalJSON([]byte) error {
	return ErrProviderPolicyCandidate
}

// UnmarshalJSON refuses restoration of private binding state from JSON.
// Identity metadata may be serialized separately as untrusted diagnostics.
func (*ProviderCandidateContext) UnmarshalJSON([]byte) error {
	return ErrProviderPolicyCandidate
}

// ParseProviderPolicyCandidate checks the exact closed local candidate schema,
// canonical bytes, supported instance/origin and explicit positive budgets.
// maximumBytes is a parsing ceiling (at most 64 KiB), not an operating default.
// Inputs must not be concurrently mutated; no caller-owned bytes are retained.
// Raw JSON must already be canonical; this API never repairs a digest input.
func ParseProviderPolicyCandidate(raw []byte, maximumBytes int) (*ProviderPolicyCandidate, error) {
	if maximumBytes <= 0 || maximumBytes > 64<<10 {
		return nil, ErrConfiguration
	}
	if len(raw) == 0 || len(raw) > maximumBytes {
		return nil, ErrProviderPolicyCandidate
	}
	m := contentObject(contentJSON(raw), "format", "spec_revision", "implementation_artifact", "instance", "provider", "finality", "trace_format", "carrier_formats", "fork_artifact", "limits", "clock", "retention", "assumptions", "claims")
	if m == nil || m["format"] != "sage-provider-registry-observation-v1" ||
		m["spec_revision"] != "a22a275b4d88f1d415b53cdffc5bed41231b77a0" ||
		!providerCandidateDigest(m["implementation_artifact"]) || !providerCandidateDigest(m["fork_artifact"]) ||
		!providerCandidateInstance(m["instance"]) || !providerCandidateOrigin(m["provider"]) ||
		!providerCandidateFormats(m) || !providerCandidateLimits(m["limits"]) || !providerCandidateClock(m["clock"]) {
		return nil, ErrProviderPolicyCandidate
	}
	// The input is JCS. Marshal only already-checked ASCII string objects; their
	// lexical member order and JSON escaping are identical to these JCS subtrees.
	instance, _ := json.Marshal(m["instance"])
	provider, _ := json.Marshal(m["provider"])
	return &ProviderPolicyCandidate{
		raw: string(raw), policy: providerCandidateHash("sage-provider-policy-v1", raw),
		instance: providerCandidateHash("sage-registry-instance-v1", instance),
		provider: providerCandidateHash("sage-provider-origin-v1", provider),
	}, nil
}

// CanonicalBytes returns a fresh diagnostic copy. A zero candidate returns nil.
func (p *ProviderPolicyCandidate) CanonicalBytes() []byte {
	if p == nil || p.raw == "" {
		return nil
	}
	return []byte(p.raw)
}

// BindCandidate records proposed local identities without selecting them.
// In particular generation zero is not authenticated first enrollment; a new
// namespace, Source ID or generation cannot reset any retained instance root.
func (p *ProviderPolicyCandidate) BindCandidate(namespace, sourceID, generation string) (*ProviderCandidateContext, error) {
	if p == nil || p.raw == "" || !providerCandidateDigest(namespace) || !providerCandidateDigest(sourceID) {
		return nil, ErrProviderPolicyCandidate
	}
	n, ok := providerCandidateUint(generation, false)
	if !ok {
		return nil, ErrProviderPolicyCandidate
	}
	return &ProviderCandidateContext{ProviderCandidateIdentity{p.policy, p.instance, p.provider, namespace, sourceID, n}, true}, nil
}

// Identity copies the candidate metadata, not an authenticated selection token.
func (c *ProviderCandidateContext) Identity() (ProviderCandidateIdentity, error) {
	if c == nil || !c.bound {
		return ProviderCandidateIdentity{}, ErrProviderPolicyCandidate
	}
	return c.identity, nil
}

func providerCandidateHash(domain string, raw []byte) string {
	h := sha256.New()
	_, _ = h.Write([]byte(domain))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write(raw)
	return "0x" + hex.EncodeToString(h.Sum(nil))
}

func providerCandidateDigest(value any) bool {
	s, ok := value.(string)
	return ok && lowerHash(s)
}

func providerCandidateUint(value any, positive bool) (uint64, bool) {
	s, ok := value.(string)
	if !ok || s == "" || len(s) > 20 {
		return 0, false
	}
	n, err := strconv.ParseUint(s, 10, 64)
	return n, err == nil && strconv.FormatUint(n, 10) == s && (!positive || n > 0)
}
