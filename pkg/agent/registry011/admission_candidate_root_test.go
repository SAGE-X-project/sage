package registry011

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

func admissionTestRootObject(t *testing.T) map[string]any {
	t.Helper()
	m := enrollmentTestRoot(t, true, "enrolling")
	m["format"] = "sage-provider-enrollment-v2"
	return m
}
func admissionTestRoot(t *testing.T, m map[string]any) *CandidateEnrollmentRoot {
	t.Helper()
	root, err := ParseCandidateEnrollmentRoot(enrollmentTestContext(t), providerPolicyTestBytes(t, m), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	return root
}
func admissionTestNext(t *testing.T, before *CandidateEnrollmentRoot, changes map[string]any) *CandidateEnrollmentRoot {
	t.Helper()
	m := contentJSON(before.CanonicalBytes()).(map[string]any)
	m["sequence"] = strconv.FormatUint(before.sequence+1, 10)
	m["previous_root"] = before.digest
	for k, v := range changes {
		m[k] = v
	}
	return admissionTestRoot(t, m)
}
func admissionTestDigest(c string) string { return "0x" + strings.Repeat(c, 64) }

func TestAdmissionCandidateRootClosedSchema(t *testing.T) {
	m := admissionTestRootObject(t)
	initial := admissionTestRoot(t, m)
	if initial.sequence != 0 || initial.fence != 1 || initial.status != "enrolling" {
		t.Fatal("initial candidate shape")
	}
	for _, status := range []string{"enrolling", "active", "quarantined", "fenced", "retired"} {
		t.Run(status, func(t *testing.T) {
			m := admissionTestRootObject(t)
			m["sequence"] = "1"
			m["previous_root"] = initial.digest
			m["status"] = status
			if status == "enrolling" || status == "active" {
				m["budget_root"] = admissionTestDigest("4")
			}
			if status == "active" {
				m["checkpoint"] = admissionTestDigest("5")
				m["transition_root"] = admissionTestDigest("6")
			}
			_ = admissionTestRoot(t, m)
		})
	}
	active := admissionTestNext(t, initial, map[string]any{"status": "active", "budget_root": admissionTestDigest("4"), "checkpoint": admissionTestDigest("5"), "transition_root": admissionTestDigest("6")})
	for key := range contentJSON(active.CanonicalBytes()).(map[string]any) {
		for _, mode := range []string{"missing", "wrong"} {
			t.Run(key+"/"+mode, func(t *testing.T) {
				m := contentJSON(active.CanonicalBytes()).(map[string]any)
				if mode == "missing" {
					delete(m, key)
				} else {
					m[key] = "invalid"
				}
				if got, err := ParseCandidateEnrollmentRoot(enrollmentTestContext(t), providerPolicyTestBytes(t, m), 64<<10); err == nil || got != nil {
					t.Fatal("invalid closed root")
				}
			})
		}
	}
	for _, mode := range []string{"extra", "initial-generation", "initial-fence", "initial-status", "initial-previous", "initial-checkpoint", "initial-budget", "initial-transition", "noninitial-previous", "enrolling-budget", "active-checkpoint", "active-transition", "null", "array", "noncanonical", "overflow"} {
		t.Run(mode, func(t *testing.T) {
			var value any = admissionTestRootObject(t)
			switch mode {
			case "extra":
				value.(map[string]any)["ready"] = true
			case "initial-generation":
				value.(map[string]any)["generation"] = "1"
			case "initial-fence":
				value.(map[string]any)["fence"] = "2"
			case "initial-status":
				value.(map[string]any)["status"] = "active"
			case "initial-previous":
				value.(map[string]any)["previous_root"] = admissionTestDigest("7")
			case "initial-checkpoint":
				value.(map[string]any)["checkpoint"] = admissionTestDigest("7")
			case "initial-budget":
				value.(map[string]any)["budget_root"] = admissionTestDigest("7")
			case "initial-transition":
				value.(map[string]any)["transition_root"] = admissionTestDigest("7")
			case "noninitial-previous", "enrolling-budget", "active-checkpoint", "active-transition":
				m := contentJSON(active.CanonicalBytes()).(map[string]any)
				value = m
				switch mode {
				case "noninitial-previous":
					m["previous_root"] = nil
				case "enrolling-budget":
					m["status"] = "enrolling"
					m["budget_root"] = nil
				case "active-checkpoint":
					m["checkpoint"] = nil
				case "active-transition":
					m["transition_root"] = nil
				}
			case "null":
				value = nil
			case "array":
				value = []any{}
			case "overflow":
				value.(map[string]any)["sequence"] = "18446744073709551616"
			}
			raw := providerPolicyTestBytes(t, value)
			if mode == "noncanonical" {
				raw = append(raw, '\n')
			}
			if got, err := ParseCandidateEnrollmentRoot(enrollmentTestContext(t), raw, 64<<10); err == nil || got != nil {
				t.Fatal("invalid root", mode)
			}
		})
	}
	for _, limit := range []int{0, 64<<10 + 1, len(initial.raw) - 1} {
		if got, err := ParseCandidateEnrollmentRoot(enrollmentTestContext(t), initial.CanonicalBytes(), limit); err == nil || got != nil {
			t.Fatal("invalid bound")
		}
	}
	if _, err := ParseCandidateEnrollmentRoot(nil, initial.CanonicalBytes(), 64<<10); err == nil {
		t.Fatal("nil context")
	}
	if _, err := ParseCandidateEnrollmentRoot(enrollmentTestContext(t), nil, 64<<10); err == nil {
		t.Fatal("empty root")
	}
}

func TestAdmissionCandidateRootTransitionDataOnly(t *testing.T) {
	initial := admissionTestRoot(t, admissionTestRootObject(t))
	budget := admissionTestNext(t, initial, map[string]any{"budget_root": admissionTestDigest("4")})
	checkpoint := admissionTestNext(t, budget, map[string]any{"checkpoint": admissionTestDigest("5"), "budget_root": admissionTestDigest("6")})
	active := admissionTestNext(t, checkpoint, map[string]any{"status": "active", "transition_root": admissionTestDigest("7")})
	advanced := admissionTestNext(t, active, map[string]any{"checkpoint": admissionTestDigest("8"), "budget_root": admissionTestDigest("9")})
	quarantined := admissionTestNext(t, advanced, map[string]any{"status": "quarantined", "fence": "2"})
	fenced := admissionTestNext(t, advanced, map[string]any{"status": "fenced", "fence": "2"})
	retired := admissionTestNext(t, quarantined, map[string]any{"status": "retired", "fence": "3"})
	for _, step := range []struct {
		kind          string
		before, after *CandidateEnrollmentRoot
	}{{"install-budget", initial, budget}, {"update-enrolling", budget, checkpoint}, {"activate", checkpoint, active}, {"advance-active", active, advanced}, {"quarantine", advanced, quarantined}, {"fence", advanced, fenced}, {"retire", quarantined, retired}} {
		t.Run(step.kind, func(t *testing.T) {
			if err := CheckCandidateEnrollmentChange(step.before, step.after, step.kind); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, bad := range []struct {
		kind          string
		before, after *CandidateEnrollmentRoot
	}{
		{"install-budget", nil, budget}, {"install-budget", initial, nil}, {"install-budget", &CandidateEnrollmentRoot{}, budget},
		{"install-budget", initial, checkpoint}, {"install-budget", budget, checkpoint}, {"update-enrolling", initial, budget},
		{"activate", initial, budget}, {"advance-active", budget, checkpoint}, {"unknown", initial, budget},
		{"advance-active", retired, advanced}, {"activate", quarantined, active}, {"activate", fenced, active},
		{"quarantine", advanced, admissionTestNext(t, advanced, map[string]any{"status": "quarantined"})},
		{"retire", advanced, admissionTestNext(t, advanced, map[string]any{"status": "retired", "fence": "2", "checkpoint": admissionTestDigest("a")})},
		{"update-enrolling", checkpoint, admissionTestNext(t, checkpoint, map[string]any{"checkpoint": nil})},
		{"install-budget", initial, admissionTestNext(t, initial, map[string]any{"budget_root": admissionTestDigest("4"), "fence": "2"})},
		{"advance-active", active, admissionTestNext(t, active, map[string]any{"transition_root": admissionTestDigest("a")})},
	} {
		if err := CheckCandidateEnrollmentChange(bad.before, bad.after, bad.kind); err == nil {
			t.Fatal("invalid transition accepted", bad.kind)
		}
	}
	if err := CheckCandidateEnrollmentChange(retired, admissionTestNext(t, retired, nil), "retire"); err == nil {
		t.Fatal("retired root revived")
	}
}

func TestAdmissionCandidateRootCopiesAndV1Isolation(t *testing.T) {
	context := enrollmentTestContext(t)
	m := admissionTestRootObject(t)
	raw := providerPolicyTestBytes(t, m)
	root, err := ParseCandidateEnrollmentRoot(context, raw, 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	raw[0] = '!'
	copy := root.CanonicalBytes()
	copy[0] = '!'
	if bytes.Equal(root.CanonicalBytes(), copy) {
		t.Fatal("mutable root")
	}
	diagnostic, err := root.Diagnostic()
	if err != nil || diagnostic.Digest != enrollmentTestHash("sage-provider-enrollment-v2", root.CanonicalBytes()) {
		t.Fatal(err)
	}
	if _, err := ParseCandidateEnrollmentRoot(context, providerPolicyTestBytes(t, enrollmentTestRoot(t, true, "enrolling")), 64<<10); err == nil {
		t.Fatal("V1 relabeled V2")
	}
	id, _ := context.Identity()
	if got, err := candidateEnrollmentRoot(m, id.Namespace, id.Instance); err == nil || got != nil {
		t.Fatal("V1 parser accepts V2")
	}
	var empty *CandidateEnrollmentRoot
	if empty.CanonicalBytes() != nil {
		t.Fatal("nil bytes")
	}
	if _, err := empty.Diagnostic(); err == nil {
		t.Fatal("nil diagnostic")
	}
	if _, err := (&CandidateEnrollmentRoot{}).Diagnostic(); err == nil {
		t.Fatal("zero diagnostic")
	}
	if err := json.Unmarshal([]byte(`{}`), &CandidateEnrollmentRoot{}); err != ErrAdmissionCandidate {
		t.Fatal("JSON restored private data")
	}
}
