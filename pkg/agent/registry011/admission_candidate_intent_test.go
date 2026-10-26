package registry011

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"reflect"
	"strconv"
	"testing"
)

// Explicit inert data only: no fixture clock, identity, metadata root or parsed
// candidate is an operating selection or a Source/work admission credential.
func admissionTestIntentFixture(t *testing.T, object map[string]any) (*ProviderPolicyCandidate, *ProviderCandidateContext, *CandidateEnrollmentRoot, map[string]any) {
	t.Helper()
	if object == nil {
		object = providerPolicyTestObject()
	}
	policy, err := ParseProviderPolicyCandidate(providerPolicyTestBytes(t, object), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	context, err := policy.BindCandidate(admissionTestDigest("2"), admissionTestDigest("3"), "0")
	if err != nil {
		t.Fatal(err)
	}
	id, _ := context.Identity()
	rootObject := admissionTestRootObject(t)
	rootObject["policy"] = id.Policy
	anchor, err := ParseCandidateEnrollmentRoot(context, providerPolicyTestBytes(t, rootObject), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	limits := object["limits"].(map[string]any)
	duration, _ := providerCandidateUint(limits["bootstrap"].(map[string]any)["deadline_ms"], true)
	deadline := uint64(1000000)
	if duration > math.MaxInt64-deadline {
		deadline = math.MaxInt64
	} else {
		deadline += duration
	}
	limitsRaw := providerPolicyTestBytes(t, limits)
	sum := sha256.Sum256(limitsRaw)
	clockPolicy := object["clock"].(map[string]any)
	m := map[string]any{"format": "sage-provider-admission-intent-v1", "policy": id.Policy, "namespace": id.Namespace, "instance_key": id.Instance, "source_id": id.SourceID, "generation": "0", "fence": "1", "admission_id": admissionTestDigest("7"), "enrollment_anchor_root": anchor.digest, "prior_published_checkpoint": nil, "initial_clock": map[string]any{"id": clockPolicy["id"], "epoch": admissionTestDigest("9"), "wall_ms": "1000000", "monotonic_ms": "100", "uncertainty_ms": "0"}, "deadline_wall_ms": strconv.FormatUint(deadline, 10), "limits_digest": "0x" + hex.EncodeToString(sum[:]), "authority_metadata_root": admissionTestDigest("8")}
	return policy, context, anchor, m
}

func TestAdmissionCandidateIntentBeforeFirstResponse(t *testing.T) {
	policy, context, anchor, m := admissionTestIntentFixture(t, nil)
	raw := providerPolicyTestBytes(t, m)
	intent, err := ParseCandidateAdmissionIntent(policy, context, anchor, raw, 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	d, err := intent.Diagnostic()
	if err != nil || d.Digest != enrollmentTestHash("sage-provider-admission-intent-v1", raw) || d.EnrollmentAnchor != anchor.digest || d.CandidateClock.Epoch != admissionTestDigest("9") {
		t.Fatal(d, err)
	}
	trustedInput := reflect.TypeOf(ClockSample{})
	if reflect.TypeOf(intent.clock).AssignableTo(trustedInput) || reflect.TypeOf(d.CandidateClock).AssignableTo(trustedInput) {
		t.Fatal("JSON-derived candidate clock is assignable to the trusted-input type")
	}
	raw[0] = '!'
	copy := intent.CanonicalBytes()
	copy[0] = '!'
	if bytes.Equal(copy, intent.CanonicalBytes()) {
		t.Fatal("mutable intent")
	}
	m["selected_finality_exchange"] = admissionTestDigest("a")
	if got, err := ParseCandidateAdmissionIntent(policy, context, anchor, providerPolicyTestBytes(t, m), 64<<10); err == nil || got != nil {
		t.Fatal("future response dependency admitted")
	}
	if _, exists := contentJSON(intent.CanonicalBytes()).(map[string]any)["selected_finality_exchange"]; exists {
		t.Fatal("pre-start depends on response")
	}
}

func TestAdmissionCandidateIntentClosedFields(t *testing.T) {
	_, _, _, good := admissionTestIntentFixture(t, nil)
	for key := range good {
		for _, mode := range []string{"missing", "wrong"} {
			t.Run(key+"/"+mode, func(t *testing.T) {
				p, c, a, m := admissionTestIntentFixture(t, nil)
				if mode == "missing" {
					delete(m, key)
				} else {
					m[key] = "invalid"
				}
				if got, err := ParseCandidateAdmissionIntent(p, c, a, providerPolicyTestBytes(t, m), 64<<10); err == nil || got != nil {
					t.Fatal("invalid field accepted")
				}
			})
		}
	}
	for _, mode := range []string{"generation", "fence", "deadline", "deadline-alias", "deadline-overflow", "prior-checkpoint", "limits-digest", "anchor", "null", "array", "noncanonical", "clock-underflow", "duration-overflow"} {
		t.Run(mode, func(t *testing.T) {
			object := providerPolicyTestObject()
			if mode == "duration-overflow" {
				object["limits"].(map[string]any)["bootstrap"].(map[string]any)["deadline_ms"] = "18446744073709551615"
			}
			p, c, a, m := admissionTestIntentFixture(t, object)
			var value any = m
			switch mode {
			case "generation":
				m["generation"] = "1"
			case "fence":
				m["fence"] = "2"
			case "deadline":
				m["deadline_wall_ms"] = "1"
			case "deadline-alias":
				m["deadline_wall_ms"] = "01"
			case "deadline-overflow":
				m["deadline_wall_ms"] = "18446744073709551615"
			case "prior-checkpoint":
				m["prior_published_checkpoint"] = admissionTestDigest("a")
			case "limits-digest":
				m["limits_digest"] = admissionTestDigest("a")
			case "anchor":
				m["enrollment_anchor_root"] = admissionTestDigest("a")
			case "null":
				value = nil
			case "array":
				value = []any{}
			case "clock-underflow":
				clock := m["initial_clock"].(map[string]any)
				clock["wall_ms"] = "0"
				clock["uncertainty_ms"] = "1"
			}
			raw := providerPolicyTestBytes(t, value)
			if mode == "noncanonical" {
				raw = append(raw, '\n')
			}
			if got, err := ParseCandidateAdmissionIntent(p, c, a, raw, 64<<10); err == nil || got != nil {
				t.Fatal("invalid intent accepted", mode)
			}
		})
	}
}

func TestAdmissionCandidateClockIsOnlyData(t *testing.T) {
	for _, mode := range []string{"id", "epoch", "wall_ms", "monotonic_ms", "uncertainty_ms", "extra", "missing", "wall-overflow", "monotonic-overflow", "uncertainty-overflow", "uncertainty-policy"} {
		t.Run(mode, func(t *testing.T) {
			p, c, a, m := admissionTestIntentFixture(t, nil)
			clock := m["initial_clock"].(map[string]any)
			switch mode {
			case "extra":
				clock["trusted"] = true
			case "missing":
				delete(clock, "epoch")
			case "wall-overflow":
				clock["wall_ms"] = "9223372036854775808"
			case "monotonic-overflow":
				clock["monotonic_ms"] = "9223372036854775808"
			case "uncertainty-overflow":
				clock["uncertainty_ms"] = "9223372036854775808"
			case "uncertainty-policy":
				maximum, _ := providerCandidateUint(providerPolicyTestObject()["clock"].(map[string]any)["max_uncertainty_ms"], true)
				clock["uncertainty_ms"] = strconv.FormatUint(maximum+1, 10)
			default:
				clock[mode] = "invalid"
			}
			if got, err := ParseCandidateAdmissionIntent(p, c, a, providerPolicyTestBytes(t, m), 64<<10); err == nil || got != nil {
				t.Fatal("invalid candidate clock")
			}
		})
	}
}

func TestAdmissionCandidateIntentArgumentsAndOpaqueCopies(t *testing.T) {
	p, c, a, m := admissionTestIntentFixture(t, nil)
	raw := providerPolicyTestBytes(t, m)
	for _, limit := range []int{0, 64<<10 + 1, len(raw) - 1} {
		if got, err := ParseCandidateAdmissionIntent(p, c, a, raw, limit); err == nil || got != nil {
			t.Fatal("invalid bound")
		}
	}
	if _, err := ParseCandidateAdmissionIntent(p, c, a, nil, 64<<10); err == nil {
		t.Fatal("empty raw")
	}
	if _, err := ParseCandidateAdmissionIntent(nil, c, a, raw, 64<<10); err == nil {
		t.Fatal("nil policy")
	}
	if _, err := ParseCandidateAdmissionIntent(&ProviderPolicyCandidate{}, c, a, raw, 64<<10); err == nil {
		t.Fatal("zero policy")
	}
	if _, err := ParseCandidateAdmissionIntent(p, nil, a, raw, 64<<10); err == nil {
		t.Fatal("nil context")
	}
	if _, err := ParseCandidateAdmissionIntent(p, c, nil, raw, 64<<10); err == nil {
		t.Fatal("nil anchor")
	}
	if _, err := ParseCandidateAdmissionIntent(p, c, &CandidateEnrollmentRoot{}, raw, 64<<10); err == nil {
		t.Fatal("zero anchor")
	}
	next := admissionTestNext(t, a, map[string]any{"budget_root": admissionTestDigest("4")})
	if _, err := ParseCandidateAdmissionIntent(p, c, next, raw, 64<<10); err == nil {
		t.Fatal("original anchor replaced")
	}
	other, err := p.BindCandidate(admissionTestDigest("2"), admissionTestDigest("a"), "0")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseCandidateAdmissionIntent(p, other, a, raw, 64<<10); err == nil {
		t.Fatal("foreign context")
	}
	var empty *CandidateAdmissionIntent
	if empty.CanonicalBytes() != nil {
		t.Fatal("nil bytes")
	}
	if _, err := empty.Diagnostic(); err == nil {
		t.Fatal("nil diagnostic")
	}
	if _, err := (&CandidateAdmissionIntent{}).Diagnostic(); err == nil {
		t.Fatal("zero diagnostic")
	}
	if err := json.Unmarshal([]byte(`{}`), &CandidateAdmissionIntent{}); err != ErrAdmissionCandidate {
		t.Fatal("JSON restored private candidate")
	}
}
