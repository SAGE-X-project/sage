package registry011

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func enrollmentTestHash(domain string, raw []byte) string {
	sum := sha256.Sum256(append([]byte(domain+"\x00"), raw...))
	return "0x" + hex.EncodeToString(sum[:])
}

func TestProviderEnrollmentFixedDigestVector(t *testing.T) {
	// Inert ASCII-only fixture independently encoded with sorted JSON members
	// and stdlib SHA-256. These digests name no selected operating artifacts.
	id := ProviderCandidateIdentity{
		Namespace: "0x" + strings.Repeat("2", 64), Instance: "0x" + strings.Repeat("4", 64),
		Policy: "0x" + strings.Repeat("5", 64), SourceID: "0x" + strings.Repeat("3", 64),
	}
	anchor := providerPolicyTestBytes(t, map[string]any{
		"format": "sage-provider-namespace-anchor-v1", "namespace": id.Namespace, "instance_key": id.Instance, "enrollment": nil,
	})
	const pin = "0xce8304287c363e7466e1f3cfd5994f1eba2a2fe2c630698f7b4e57b73da5d3b8"
	if providerCandidateHash("sage-provider-namespace-anchor-v1", anchor) != pin {
		t.Fatal("anchor vector")
	}
	root, entry, err := makeCandidateEnrollment(id, pin)
	if err != nil {
		t.Fatal(err)
	}
	if providerCandidateHash("sage-provider-enrollment-v1", root) != "0x6820fbb2f372c37ec7d9b39187ce22ae40bdf98671278c29d8889b1add07e491" ||
		providerCandidateHash("sage-provider-namespace-entry-v1", entry) != "0x638cfca49b64b9c0bd7f12be108038b201e369a2ba69b4a4d74b1cf71481fada" {
		t.Fatal("enrollment/entry vector")
	}
}

func enrollmentTestContext(t *testing.T) *ProviderCandidateContext {
	t.Helper()
	p, err := ParseProviderPolicyCandidate(providerPolicyTestBytes(t, providerPolicyTestObject()), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	c, err := p.BindCandidate("0x"+strings.Repeat("2", 64), "0x"+strings.Repeat("3", 64), "0")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func enrollmentTestRoot(t *testing.T, initial bool, status string) map[string]any {
	t.Helper()
	id, err := enrollmentTestContext(t).Identity()
	if err != nil {
		t.Fatal(err)
	}
	root := map[string]any{
		"format": "sage-provider-enrollment-v1", "namespace": id.Namespace, "instance_key": id.Instance,
		"policy": id.Policy, "source_id": id.SourceID, "generation": "0", "sequence": "0", "previous_root": nil,
		"status": status, "checkpoint": nil, "budget_root": nil, "transition_root": nil, "fence": "1",
	}
	if !initial {
		root["generation"], root["sequence"] = "1", "1"
		for _, key := range []string{"previous_root", "checkpoint", "budget_root", "transition_root"} {
			root[key] = "0x" + strings.Repeat("4", 64)
		}
	}
	return root
}

// Test-only provision of an owned fixture, not a production authority factory.
// Pins here are known fixture data, not a real selected trust/operating policy.
func enrollmentTestNamespace(t *testing.T, root any) CandidateNamespaceConfig {
	t.Helper()
	id, err := enrollmentTestContext(t).Identity()
	if err != nil {
		t.Fatal(err)
	}
	anchor := providerPolicyTestBytes(t, map[string]any{"format": "sage-provider-namespace-anchor-v1", "namespace": id.Namespace, "instance_key": id.Instance, "enrollment": root})
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ANCHOR"), anchor, 0600); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(filepath.Join(dir, "ANCHOR"), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	f, err = os.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	pin := enrollmentTestHash("sage-provider-namespace-anchor-v1", anchor)
	return CandidateNamespaceConfig{dir, id.Namespace, id.Instance, pin, pin, 16 << 10}
}

func enrollmentTestOpen(t *testing.T, config CandidateNamespaceConfig) *CandidateProviderNamespace {
	t.Helper()
	n, err := OpenCandidateProviderNamespace(config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := n.Close(); err != nil {
			t.Error(err)
		}
	})
	return n
}

func TestProviderEnrollmentClosedRoots(t *testing.T) {
	id, _ := enrollmentTestContext(t).Identity()
	for _, status := range []string{"enrolling", "active", "quarantined", "fenced", "retired"} {
		t.Run(status, func(t *testing.T) {
			root := enrollmentTestRoot(t, false, status)
			if _, err := candidateEnrollmentRoot(root, id.Namespace, id.Instance); err != nil {
				t.Fatal(err)
			}
			for key := range root {
				for _, mode := range []string{"missing", "wrong"} {
					t.Run(key+"/"+mode, func(t *testing.T) {
						changed := enrollmentTestRoot(t, false, status)
						if mode == "missing" {
							delete(changed, key)
						} else {
							changed[key] = "invalid"
						}
						if raw, err := candidateEnrollmentRoot(changed, id.Namespace, id.Instance); err != ErrProviderRetention || raw != nil {
							t.Fatal("accepted invalid root")
						}
					})
				}
			}
		})
	}
	for _, mode := range []string{"null", "array", "extra", "initial-generation", "initial-status", "initial-predecessor", "initial-checkpoint", "initial-budget", "initial-transition", "zero-fence", "overflow", "numeric", "null-noninitial"} {
		t.Run(mode, func(t *testing.T) {
			var value any = enrollmentTestRoot(t, true, "enrolling")
			switch mode {
			case "null":
				value = nil
			case "array":
				value = []any{}
			case "extra":
				value.(map[string]any)["ready"] = true
			case "initial-generation":
				value.(map[string]any)["generation"] = "1"
			case "initial-status":
				value.(map[string]any)["status"] = "active"
			case "initial-predecessor":
				value.(map[string]any)["previous_root"] = "0x" + strings.Repeat("4", 64)
			case "initial-checkpoint":
				value.(map[string]any)["checkpoint"] = "0x" + strings.Repeat("4", 64)
			case "initial-budget":
				value.(map[string]any)["budget_root"] = "0x" + strings.Repeat("4", 64)
			case "initial-transition":
				value.(map[string]any)["transition_root"] = "0x" + strings.Repeat("4", 64)
			case "zero-fence":
				value.(map[string]any)["fence"] = "0"
			case "overflow":
				value.(map[string]any)["sequence"] = "18446744073709551616"
			case "numeric":
				value.(map[string]any)["generation"] = 0
			case "null-noninitial":
				value = enrollmentTestRoot(t, false, "active")
				value.(map[string]any)["checkpoint"] = nil
			}
			if raw, err := candidateEnrollmentRoot(value, id.Namespace, id.Instance); raw != nil || err != ErrProviderRetention {
				t.Fatal("invalid supported root")
			}
		})
	}
	for _, pair := range [][2]string{{"", id.Instance}, {id.Namespace, ""}} {
		if raw, err := candidateEnrollmentRoot(enrollmentTestRoot(t, true, "enrolling"), pair[0], pair[1]); raw != nil || err != ErrProviderRetention {
			t.Fatal("invalid root context")
		}
	}
	for _, value := range []any{math.NaN(), json.Number("1e400")} {
		if raw, err := canonicalCandidateEnrollment(value); raw != nil || err != ErrProviderRetention {
			t.Fatal("invalid canonical value")
		}
	}
}

func TestProviderEnrollmentAnchorAndEntrySchemas(t *testing.T) {
	config := enrollmentTestNamespace(t, nil)
	anchor, err := os.ReadFile(filepath.Join(config.Directory, "ANCHOR"))
	if err != nil {
		t.Fatal(err)
	}
	id, _ := enrollmentTestContext(t).Identity()
	if root, err := candidateNamespaceAnchor(anchor, id.Namespace, id.Instance, config.AnchorDigest); root != nil || err != nil {
		t.Fatal("declared initial state")
	}
	root, entry, err := makeCandidateEnrollment(id, config.AnchorDigest)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := candidateEnrollmentEntry(entry, id.Namespace, id.Instance, config.AnchorDigest); err != nil || !bytes.Equal(got, root) {
		t.Fatal("exact entry")
	}
	for _, target := range []string{"anchor", "entry"} {
		for _, mode := range []string{"extra", "duplicate", "whitespace", "trailing", "bad-pin", "foreign-namespace", "foreign-instance"} {
			t.Run(target+"/"+mode, func(t *testing.T) {
				raw := bytes.Clone(anchor)
				if target == "entry" {
					raw = bytes.Clone(entry)
				}
				ns, instance, pin := id.Namespace, id.Instance, config.AnchorDigest
				switch mode {
				case "extra":
					var object map[string]any
					if json.Unmarshal(raw, &object) != nil {
						t.Fatal("fixture")
					}
					object["ready"] = true
					raw = providerPolicyTestBytes(t, object)
				case "duplicate":
					raw = []byte(`{"format":"a","format":"b"}`)
				case "whitespace":
					raw = append([]byte(" "), raw...)
				case "trailing":
					raw = append(raw, 'x')
				case "bad-pin":
					pin = "0x" + strings.Repeat("8", 64)
				case "foreign-namespace":
					ns = "0x" + strings.Repeat("8", 64)
				case "foreign-instance":
					instance = "0x" + strings.Repeat("8", 64)
				}
				var out []byte
				var err error
				if target == "anchor" {
					out, err = candidateNamespaceAnchor(raw, ns, instance, pin)
				} else {
					out, err = candidateEnrollmentEntry(raw, ns, instance, pin)
				}
				if out != nil || err != ErrProviderRetention {
					t.Fatal("invalid closed object")
				}
			})
		}
	}
	for _, key := range []string{"format", "namespace", "instance_key", "index", "previous_tip", "enrollment"} {
		t.Run(key, func(t *testing.T) {
			var m map[string]any
			if json.Unmarshal(entry, &m) != nil {
				t.Fatal("fixture")
			}
			delete(m, key)
			if out, err := candidateEnrollmentEntry(providerPolicyTestBytes(t, m), id.Namespace, id.Instance, config.AnchorDigest); out != nil || err != ErrProviderRetention {
				t.Fatal("missing field")
			}
		})
	}
	for _, field := range []string{"generation", "sequence", "fence"} {
		var m map[string]any
		if json.Unmarshal(entry, &m) != nil {
			t.Fatal("fixture")
		}
		nested := m["enrollment"].(map[string]any)
		if field == "fence" {
			nested[field] = "2"
		} else {
			nested["sequence"] = "1"
			nested[field] = "1"
			for _, k := range []string{"previous_root", "checkpoint", "budget_root", "transition_root"} {
				nested[k] = "0x" + strings.Repeat("4", 64)
			}
		}
		if out, err := candidateEnrollmentEntry(providerPolicyTestBytes(t, m), id.Namespace, id.Instance, config.AnchorDigest); out != nil || err != ErrProviderRetention {
			t.Fatal("noninitial entry")
		}
	}
	for _, change := range []string{"generation", "anchor", "policy", "namespace", "instance", "source"} {
		bad := id
		pin := config.AnchorDigest
		switch change {
		case "generation":
			bad.Generation = 1
		case "anchor":
			pin = ""
		case "policy":
			bad.Policy = ""
		case "namespace":
			bad.Namespace = ""
		case "instance":
			bad.Instance = ""
		case "source":
			bad.SourceID = ""
		}
		if a, b, err := makeCandidateEnrollment(bad, pin); a != nil || b != nil || err != ErrProviderRetention {
			t.Fatal("invalid enrollment context")
		}
	}
}
