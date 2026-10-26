package registry011

import (
	"errors"
	"math"
)

var ErrAdmissionCandidate = errors.New("registry011: invalid admission candidate")

// CandidateEnrollmentRoot is proposed V2 local data, never an authenticated
// root, absence witness, custody ACK or Source/work authority. Existing V1
// enrollment parsing remains unchanged. No operating API accepts this type.
type CandidateEnrollmentRoot struct {
	raw, digest                                      string
	identity                                         ProviderCandidateIdentity
	sequence, fence                                  uint64
	previous, status, checkpoint, budget, transition string
}

// CandidateAdmissionRootDiagnostic is copied, forgeable data only.
type CandidateAdmissionRootDiagnostic struct {
	Digest, Status, Previous, Checkpoint, Budget, Transition string
	Identity                                                 ProviderCandidateIdentity
	Sequence, Fence                                          uint64
}

func (*CandidateEnrollmentRoot) UnmarshalJSON([]byte) error { return ErrAdmissionCandidate }

// ParseCandidateEnrollmentRoot validates only the separately proposed closed
// V2 schema. maximumBytes is explicit and at most64KiB, not an operating limit.
// All digests name proposed data; parsing does not qualify their referenced
// evidence, select a format/policy or authenticate any clock/store/authority.
func ParseCandidateEnrollmentRoot(context *ProviderCandidateContext, raw []byte, maximumBytes int) (*CandidateEnrollmentRoot, error) {
	id, err := context.Identity()
	if err != nil || maximumBytes <= 0 || maximumBytes > 64<<10 {
		return nil, ErrConfiguration
	}
	if len(raw) == 0 || len(raw) > maximumBytes {
		return nil, ErrAdmissionCandidate
	}
	m := contentObject(contentJSON(raw), "format", "namespace", "instance_key", "policy", "source_id", "generation", "sequence", "previous_root", "status", "checkpoint", "budget_root", "transition_root", "fence")
	if m == nil || m["format"] != "sage-provider-enrollment-v2" || m["namespace"] != id.Namespace || m["instance_key"] != id.Instance || m["policy"] != id.Policy || m["source_id"] != id.SourceID {
		return nil, ErrAdmissionCandidate
	}
	generation, a := providerCandidateUint(m["generation"], false)
	sequence, b := providerCandidateUint(m["sequence"], false)
	fence, c := providerCandidateUint(m["fence"], true)
	status := contentString(m["status"])
	if !a || !b || !c || generation != id.Generation || !candidateEnrollmentStatus(status) {
		return nil, ErrAdmissionCandidate
	}
	for _, name := range []string{"previous_root", "checkpoint", "budget_root", "transition_root"} {
		if m[name] != nil && !providerCandidateDigest(m[name]) {
			return nil, ErrAdmissionCandidate
		}
	}
	if sequence == 0 {
		if generation != 0 || fence != 1 || status != "enrolling" || m["previous_root"] != nil || m["checkpoint"] != nil || m["budget_root"] != nil || m["transition_root"] != nil {
			return nil, ErrAdmissionCandidate
		}
	} else {
		if m["previous_root"] == nil {
			return nil, ErrAdmissionCandidate
		}
		if status == "enrolling" && m["budget_root"] == nil {
			return nil, ErrAdmissionCandidate
		}
		if status == "active" && (m["checkpoint"] == nil || m["budget_root"] == nil || m["transition_root"] == nil) {
			return nil, ErrAdmissionCandidate
		}
	}
	return &CandidateEnrollmentRoot{raw: string(raw), digest: providerCandidateHash("sage-provider-enrollment-v2", raw), identity: id, sequence: sequence, fence: fence, previous: contentString(m["previous_root"]), status: status, checkpoint: contentString(m["checkpoint"]), budget: contentString(m["budget_root"]), transition: contentString(m["transition_root"])}, nil
}

func candidateEnrollmentStatus(status string) bool {
	switch status {
	case "enrolling", "active", "quarantined", "fenced", "retired":
		return true
	}
	return false
}

// CheckCandidateEnrollmentChange checks a same-identity candidate transition.
// It does not authenticate its kind/roots or qualify the evidence. Migration,
// generation changes, owner transfer and recovery activation are unsupported.
// Actual custody must independently authorize and durably bind a transition.
func CheckCandidateEnrollmentChange(before, after *CandidateEnrollmentRoot, kind string) error {
	if before == nil || after == nil || before.raw == "" || after.raw == "" || before.identity != after.identity || before.sequence == math.MaxUint64 || after.sequence != before.sequence+1 || after.previous != before.digest {
		return ErrAdmissionCandidate
	}
	if before.status == "retired" {
		return ErrAdmissionCandidate
	}
	if (before.status == "quarantined" || before.status == "fenced") && kind != "retire" {
		return ErrAdmissionCandidate
	}
	switch kind {
	case "install-budget":
		if before.status != "enrolling" || after.status != "enrolling" || before.budget != "" || after.budget == "" || before.checkpoint != "" || after.checkpoint != "" || before.transition != "" || after.transition != "" {
			return ErrAdmissionCandidate
		}
	case "update-enrolling":
		if before.status != "enrolling" || after.status != "enrolling" || before.budget == "" || after.budget == "" || before.transition != after.transition || before.checkpoint != "" && after.checkpoint == "" {
			return ErrAdmissionCandidate
		}
	case "activate":
		if before.status != "enrolling" || after.status != "active" || before.budget == "" || after.budget == "" || after.checkpoint == "" || after.transition == "" {
			return ErrAdmissionCandidate
		}
	case "advance-active":
		if before.status != "active" || after.status != "active" || before.transition != after.transition {
			return ErrAdmissionCandidate
		}
	case "quarantine", "fence", "retire":
		want := map[string]string{"quarantine": "quarantined", "fence": "fenced", "retire": "retired"}[kind]
		if after.status != want || after.checkpoint != before.checkpoint || after.budget != before.budget || after.transition != before.transition || before.fence == math.MaxUint64 || after.fence != before.fence+1 {
			return ErrAdmissionCandidate
		}
		return nil
	default:
		return ErrAdmissionCandidate
	}
	if after.fence != before.fence {
		return ErrAdmissionCandidate
	}
	return nil
}

func (r *CandidateEnrollmentRoot) CanonicalBytes() []byte {
	if r == nil {
		return nil
	}
	return []byte(r.raw)
}
func (r *CandidateEnrollmentRoot) Diagnostic() (CandidateAdmissionRootDiagnostic, error) {
	if r == nil || r.raw == "" {
		return CandidateAdmissionRootDiagnostic{}, ErrAdmissionCandidate
	}
	return CandidateAdmissionRootDiagnostic{r.digest, r.status, r.previous, r.checkpoint, r.budget, r.transition, r.identity, r.sequence, r.fence}, nil
}
