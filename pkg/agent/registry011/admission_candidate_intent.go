package registry011

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
)

// CandidateAdmissionIntent is immutable proposed pre-request data. It names an
// existing candidate initial anchor and original clock/deadline, with no RPC
// exchange dependency. It is not selected policy, trusted clock, an enrolled
// Source, authenticated custody or admission to execute a request.
type CandidateAdmissionIntent struct {
	raw, digest string
	identity    ProviderCandidateIdentity
	anchor      string
	fence       uint64
	clock       CandidateClockDiagnostic
	deadline    uint64
}

// CandidateAdmissionDiagnostic is copied, forgeable information only. Its clock
// came from candidate JSON and MUST NOT be passed as a qualified clock sample.
type CandidateAdmissionDiagnostic struct {
	Digest, EnrollmentAnchor string
	Identity                 ProviderCandidateIdentity
	Fence, DeadlineWallMS    uint64
	CandidateClock           CandidateClockDiagnostic
}

// CandidateClockDiagnostic deliberately differs from the trusted-input sample
// type. No operating clock or observation API consumes this JSON-derived data.
type CandidateClockDiagnostic struct {
	ClockID, Epoch                     string
	WallMS, MonotonicMS, UncertaintyMS int64
}

func (*CandidateAdmissionIntent) UnmarshalJSON([]byte) error { return ErrAdmissionCandidate }

// ParseCandidateAdmissionIntent validates the proposed closed pre-start schema
// and acyclic references. It cannot establish actual absence or authorize first
// enrollment. V1 source/strict APIs do not accept its result. The local limits
// artifact digest is SHA-256 of the exact canonical policy.limits subtree, not
// a new wire-message digest or an authentication claim.
func ParseCandidateAdmissionIntent(policy *ProviderPolicyCandidate, context *ProviderCandidateContext, anchor *CandidateEnrollmentRoot, raw []byte, maximumBytes int) (*CandidateAdmissionIntent, error) {
	id, err := context.Identity()
	if err != nil || policy == nil || policy.raw == "" || maximumBytes <= 0 || maximumBytes > 64<<10 {
		return nil, ErrConfiguration
	}
	if anchor == nil || anchor.raw == "" || anchor.identity != id || anchor.sequence != 0 || anchor.status != "enrolling" || id.Generation != 0 || policy.policy != id.Policy {
		return nil, ErrAdmissionCandidate
	}
	if len(raw) == 0 || len(raw) > maximumBytes {
		return nil, ErrAdmissionCandidate
	}
	m := contentObject(contentJSON(raw), "format", "policy", "namespace", "instance_key", "source_id", "generation", "fence", "admission_id", "enrollment_anchor_root", "prior_published_checkpoint", "initial_clock", "deadline_wall_ms", "limits_digest", "authority_metadata_root")
	if m == nil || m["format"] != "sage-provider-admission-intent-v1" || m["policy"] != id.Policy || m["namespace"] != id.Namespace || m["instance_key"] != id.Instance || m["source_id"] != id.SourceID || m["enrollment_anchor_root"] != anchor.digest || m["prior_published_checkpoint"] != nil || !providerCandidateDigest(m["admission_id"]) || !providerCandidateDigest(m["authority_metadata_root"]) {
		return nil, ErrAdmissionCandidate
	}
	generation, a := providerCandidateUint(m["generation"], false)
	fence, b := providerCandidateUint(m["fence"], true)
	deadline, c := providerCandidateUint(m["deadline_wall_ms"], true)
	if !a || !b || !c || generation != id.Generation || fence != anchor.fence || deadline > math.MaxInt64 {
		return nil, ErrAdmissionCandidate
	}
	p := contentJSON([]byte(policy.raw)).(map[string]any)
	// This subtree contains only validated ASCII names and decimal strings, so
	// standard sorted object encoding has the same bytes as the policy's JCS.
	limitsRaw, _ := json.Marshal(p["limits"])
	limitSum := sha256.Sum256(limitsRaw)
	if m["limits_digest"] != "0x"+hex.EncodeToString(limitSum[:]) {
		return nil, ErrAdmissionCandidate
	}
	clock, ok := candidateAdmissionClock(m["initial_clock"], p["clock"].(map[string]any))
	if !ok {
		return nil, ErrAdmissionCandidate
	}
	limits := p["limits"].(map[string]any)
	duration, _ := providerCandidateUint(limits["bootstrap"].(map[string]any)["deadline_ms"], true)
	// Keep the original earliest wall bound across every future restart. Adding
	// u64 values or silently substituting a later clock must not extend it.
	if clock.WallMS < clock.UncertaintyMS {
		return nil, ErrAdmissionCandidate
	}
	initialLower := uint64(clock.WallMS - clock.UncertaintyMS)
	if duration > math.MaxInt64-initialLower || deadline != initialLower+duration {
		return nil, ErrAdmissionCandidate
	}
	return &CandidateAdmissionIntent{string(raw), providerCandidateHash("sage-provider-admission-intent-v1", raw), id, anchor.digest, fence, clock, deadline}, nil
}

func candidateAdmissionClock(value any, policy map[string]any) (CandidateClockDiagnostic, bool) {
	m := contentObject(value, "id", "epoch", "wall_ms", "monotonic_ms", "uncertainty_ms")
	if m == nil || m["id"] != policy["id"] || !providerCandidateDigest(m["epoch"]) {
		return CandidateClockDiagnostic{}, false
	}
	wall, a := providerCandidateUint(m["wall_ms"], false)
	monotonic, b := providerCandidateUint(m["monotonic_ms"], false)
	uncertainty, c := providerCandidateUint(m["uncertainty_ms"], false)
	maximum, _ := providerCandidateUint(policy["max_uncertainty_ms"], true)
	if !a || !b || !c || wall > math.MaxInt64 || monotonic > math.MaxInt64 || uncertainty > math.MaxInt64 || uncertainty > maximum {
		return CandidateClockDiagnostic{}, false
	}
	return CandidateClockDiagnostic{ClockID: contentString(m["id"]), Epoch: contentString(m["epoch"]), WallMS: int64(wall), MonotonicMS: int64(monotonic), UncertaintyMS: int64(uncertainty)}, true
}

func (i *CandidateAdmissionIntent) CanonicalBytes() []byte {
	if i == nil {
		return nil
	}
	return []byte(i.raw)
}
func (i *CandidateAdmissionIntent) Diagnostic() (CandidateAdmissionDiagnostic, error) {
	if i == nil || i.raw == "" {
		return CandidateAdmissionDiagnostic{}, ErrAdmissionCandidate
	}
	clock := CandidateClockDiagnostic{i.clock.ClockID, i.clock.Epoch, i.clock.WallMS, i.clock.MonotonicMS, i.clock.UncertaintyMS}
	return CandidateAdmissionDiagnostic{i.digest, i.anchor, i.identity, i.fence, i.deadline, clock}, nil
}
