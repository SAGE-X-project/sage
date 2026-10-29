package registry011

import (
	"encoding/json"

	"github.com/sage-x-project/sage/pkg/agent/internal/rfc8785"
)

// All values here are closed local candidate data, not peer message schemas or
// proof of authority. Referenced checkpoints and policies are not qualified.
func candidateEnrollmentRoot(value any, namespace, instance string) ([]byte, error) {
	m := contentObject(value, "format", "namespace", "instance_key", "policy", "source_id", "generation", "sequence", "previous_root", "status", "checkpoint", "budget_root", "transition_root", "fence")
	if m == nil || !lowerHash(namespace) || !lowerHash(instance) || m["format"] != "sage-provider-enrollment-v1" ||
		m["namespace"] != namespace || m["instance_key"] != instance ||
		!providerCandidateDigest(m["policy"]) || !providerCandidateDigest(m["source_id"]) {
		return nil, ErrProviderRetention
	}
	generation, a := providerCandidateUint(m["generation"], false)
	sequence, b := providerCandidateUint(m["sequence"], false)
	_, c := providerCandidateUint(m["fence"], true)
	status := contentString(m["status"])
	if !a || !b || !c || (status != "enrolling" && status != "active" && status != "quarantined" && status != "fenced" && status != "retired") {
		return nil, ErrProviderRetention
	}
	for _, name := range []string{"previous_root", "checkpoint", "budget_root", "transition_root"} {
		if sequence == 0 {
			if generation != 0 || status != "enrolling" || m[name] != nil {
				return nil, ErrProviderRetention
			}
		} else if !providerCandidateDigest(m[name]) {
			// More complicated intermediate null states are unsupported. They
			// must never be repaired into initial state or a successful read.
			return nil, ErrProviderRetention
		}
	}
	return canonicalCandidateEnrollment(m)
}

func canonicalCandidateEnrollment(value any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, ErrProviderRetention
	}
	raw, err = rfc8785.Canonicalize(raw)
	if err != nil {
		return nil, ErrProviderRetention
	}
	return raw, nil
}

func candidateNamespaceAnchor(raw []byte, namespace, instance, pin string) ([]byte, error) {
	m := contentObject(contentJSON(raw), "format", "namespace", "instance_key", "enrollment")
	if m == nil || m["format"] != "sage-provider-namespace-anchor-v1" ||
		m["namespace"] != namespace || m["instance_key"] != instance ||
		providerCandidateHash("sage-provider-namespace-anchor-v1", raw) != pin {
		return nil, ErrProviderRetention
	}
	if m["enrollment"] == nil {
		return nil, nil
	}
	return candidateEnrollmentRoot(m["enrollment"], namespace, instance)
}

func candidateEnrollmentEntry(raw []byte, namespace, instance, anchor string) ([]byte, error) {
	m := contentObject(contentJSON(raw), "format", "namespace", "instance_key", "index", "previous_tip", "enrollment")
	if m == nil || m["format"] != "sage-provider-namespace-entry-v1" ||
		m["namespace"] != namespace || m["instance_key"] != instance || m["index"] != "1" || m["previous_tip"] != anchor {
		return nil, ErrProviderRetention
	}
	root, err := candidateEnrollmentRoot(m["enrollment"], namespace, instance)
	if err != nil {
		return nil, err
	}
	fields := contentObject(contentJSON(root), "format", "namespace", "instance_key", "policy", "source_id", "generation", "sequence", "previous_root", "status", "checkpoint", "budget_root", "transition_root", "fence")
	if fields["generation"] != "0" || fields["sequence"] != "0" || fields["fence"] != "1" {
		return nil, ErrProviderRetention
	}
	return root, nil
}

func makeCandidateEnrollment(identity ProviderCandidateIdentity, anchor string) ([]byte, []byte, error) {
	if identity.Generation != 0 || !lowerHash(anchor) {
		return nil, nil, ErrProviderRetention
	}
	root := map[string]any{
		"format": "sage-provider-enrollment-v1", "namespace": identity.Namespace, "instance_key": identity.Instance,
		"policy": identity.Policy, "source_id": identity.SourceID, "generation": "0", "sequence": "0",
		"previous_root": nil, "status": "enrolling", "checkpoint": nil, "budget_root": nil, "transition_root": nil, "fence": "1",
	}
	rootBytes, err := candidateEnrollmentRoot(root, identity.Namespace, identity.Instance)
	if err != nil {
		return nil, nil, err
	}
	entry, err := canonicalCandidateEnrollment(map[string]any{
		"format": "sage-provider-namespace-entry-v1", "namespace": identity.Namespace, "instance_key": identity.Instance,
		"index": "1", "previous_tip": anchor, "enrollment": root,
	})
	return rootBytes, entry, err
}
