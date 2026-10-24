package registry011

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/internal/rfc8785"
)

// Digests and small budgets below are inert fixture values. They do not name
// qualified artifacts, select operating budgets or enroll any real Source.
func providerPolicyTestObject() map[string]any {
	digest := "0x" + strings.Repeat("1", 64)
	work := func() map[string]any {
		out := map[string]any{}
		for _, key := range []string{"requests", "wire_bytes", "decoded_bytes", "blocks", "transactions", "frames", "logs", "proofs", "segments", "disk_bytes_written", "work_units", "deadline_ms"} {
			out[key] = "1000"
		}
		return out
	}
	return map[string]any{
		"format": "sage-provider-registry-observation-v1", "spec_revision": "a22a275b4d88f1d415b53cdffc5bed41231b77a0",
		"implementation_artifact": digest, "fork_artifact": digest,
		"instance":        map[string]any{"chain_id": "11155111", "genesis_hash": sepoliaGenesis, "registry_id": "eip155:11155111:" + SepoliaRegistryAddress, "deployment_hash": "0x54138f0dbccac32a1cd6fb0ed049fa013e78ca33910dd0471b09efea663d365e"},
		"provider":        map[string]any{"endpoint": SepoliaRPCURL, "transport_artifact": digest, "tls_roots_artifact": digest, "redirect_policy": "deny", "proxy_policy": "none"},
		"finality":        map[string]any{"selector": "finalized", "basis": "trusted-endpoint-finalized", "validation_artifact": digest},
		"trace_format":    map[string]any{"normalizer_artifact": digest, "acquisition_artifact": digest},
		"carrier_formats": []any{"protected-legacy", "access-list", "dynamic-fee"},
		"limits":          map[string]any{"operation": work(), "segment": work(), "bootstrap": work(), "live_disk_bytes": "1000", "retained_bytes": "1000", "lifetime_claims": "1000", "lifetime_identities": "1000"},
		"clock":           map[string]any{"id": "fixture-clock", "artifact": digest, "max_uncertainty_ms": "1000", "max_read_window_ms": "1000", "max_observation_age_ms": "1000", "max_finalized_lag_ms": "1000"},
		"retention":       map[string]any{"format": "sage-provider-retention-v1", "authority_artifact": digest, "store_artifact": digest, "acknowledgement_artifact": digest},
		"assumptions":     map[string]any{"provider_execution": "trusted-provider-execution", "provider_finality": "trusted-provider-finality", "historical_code": "qualified-fixed-code-closure", "agent_boundary": "trusted-agent-integration", "clock": "qualified-local-clock", "retaining_authority": "qualified-retaining-authority"},
		"claims":          map[string]any{"gas_decomposition": "not-verified", "independent_consensus": "not-verified", "compromised_trusted_boundary": "not-verified"},
	}
}

func providerPolicyTestBytes(t *testing.T, object map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = rfc8785.Canonicalize(raw)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func providerPolicyTestMap(object map[string]any, path []string) map[string]any {
	for _, key := range path {
		object = object[key].(map[string]any)
	}
	return object
}

func providerPolicyTestRefuse(t *testing.T, object map[string]any) {
	t.Helper()
	if p, err := ParseProviderPolicyCandidate(providerPolicyTestBytes(t, object), 64<<10); p != nil || err != ErrProviderPolicyCandidate {
		t.Fatalf("accepted invalid candidate: %v %v", p, err)
	}
}

func TestProviderPolicyCandidateIdentityAndCopies(t *testing.T) {
	raw := providerPolicyTestBytes(t, providerPolicyTestObject())
	original := bytes.Clone(raw)
	p, err := ParseProviderPolicyCandidate(raw, len(raw))
	if err != nil {
		t.Fatal(err)
	}
	raw[0] = '!'
	copy := p.CanonicalBytes()
	if !bytes.Equal(copy, original) {
		t.Fatal("retained mutable input")
	}
	copy[0] = '!'
	if !bytes.Equal(p.CanonicalBytes(), original) {
		t.Fatal("returned mutable retained bytes")
	}
	ns, source := "0x"+strings.Repeat("2", 64), "0x"+strings.Repeat("3", 64)
	c, err := p.BindCandidate(ns, source, "0")
	if err != nil {
		t.Fatal(err)
	}
	identity, err := c.Identity()
	// Independently computed ASCII JCS/domain-separated SHA-256 fixture values.
	want := ProviderCandidateIdentity{
		Policy:    "0x68a7c1962d90dfe41bdd46c588eecf522757b33c64aec01c38f9b5210606c2ed",
		Instance:  "0x10ee8900f137f11de1fcce55e7ebebe8cbcb22c25d3cf57817d3097d601d6f26",
		Provider:  "0xe9465a15436c23779526f2ec5be676878d66b5762f2a89aa509c4c0cfbb3e7f3",
		Namespace: ns, SourceID: source, Generation: 0,
	}
	if err != nil || identity != want {
		t.Fatalf("identity mismatch: %+v %v", identity, err)
	}
	identity.Policy = "changed"
	again, err := c.Identity()
	if err != nil || again != want {
		t.Fatal("mutable identity escaped")
	}
	var pinned struct {
		Spec       string `json:"spec_commit"`
		Deployment struct {
			Hash string `json:"block_hash"`
		}
	}
	if json.Unmarshal(nodeIdentityJSON, &pinned) != nil || pinned.Spec != providerPolicyTestObject()["spec_revision"] || pinned.Deployment.Hash != providerPolicyTestObject()["instance"].(map[string]any)["deployment_hash"] {
		t.Fatal("candidate literals differ from existing public deployment identity")
	}
}

func TestProviderPolicyCandidateClosedObjects(t *testing.T) {
	var paths [][]string
	var walk func(map[string]any, []string)
	walk = func(object map[string]any, path []string) {
		paths = append(paths, append([]string(nil), path...))
		for key, value := range object {
			if child, ok := value.(map[string]any); ok {
				walk(child, append(append([]string(nil), path...), key))
			}
		}
	}
	walk(providerPolicyTestObject(), nil)
	for _, path := range paths {
		for _, mode := range []string{"extra", "null", "array"} {
			t.Run(strings.Join(path, "/")+"/"+mode, func(t *testing.T) {
				object := providerPolicyTestObject()
				if mode == "extra" {
					providerPolicyTestMap(object, path)["ready"] = true
				} else if len(path) == 0 {
					object = nil
				} else {
					parent := providerPolicyTestMap(object, path[:len(path)-1])
					if mode == "null" {
						parent[path[len(path)-1]] = nil
					} else {
						parent[path[len(path)-1]] = []any{}
					}
				}
				providerPolicyTestRefuse(t, object)
			})
		}
		for key, value := range providerPolicyTestMap(providerPolicyTestObject(), path) {
			for _, mode := range []string{"missing", "wrong-value"} {
				t.Run(strings.Join(path, "/")+"/"+key+"/"+mode, func(t *testing.T) {
					object := providerPolicyTestObject()
					child := providerPolicyTestMap(object, path)
					if mode == "missing" {
						delete(child, key)
					} else {
						child[key] = "invalid"
						if key == "id" {
							child[key] = "\n"
						}
						if value == "1000" {
							child[key] = "0"
						}
					}
					providerPolicyTestRefuse(t, object)
				})
			}
		}
	}
}

func TestProviderPolicyCandidateRawAndBudgetBounds(t *testing.T) {
	raw := providerPolicyTestBytes(t, providerPolicyTestObject())
	for _, maximum := range []int{-1, 0, 64<<10 + 1} {
		if p, err := ParseProviderPolicyCandidate(raw, maximum); p != nil || err != ErrConfiguration {
			t.Fatal("accepted invalid parsing ceiling")
		}
	}
	for name, input := range map[string][]byte{
		"empty": nil, "oversize": bytes.Repeat([]byte(" "), 64<<10+1), "whitespace": append([]byte(" "), raw...),
		"duplicate": []byte(`{"format":"x","format":"y"}`), "escaped-duplicate": []byte(`{"format":"x","\u0066ormat":"y"}`),
		"nested-duplicate": []byte(`{"instance":{"chain_id":"1","chain_id":"2"}}`),
		"unicode":          []byte(`{"format":"\ud800"}`), "utf8": {0xff}, "trailing": append(bytes.Clone(raw), []byte("null")...),
		"invalid": []byte(`{`), "scalar": []byte(`false`), "deep": []byte(strings.Repeat("[", 33) + "0" + strings.Repeat("]", 33)),
	} {
		t.Run(name, func(t *testing.T) {
			if p, err := ParseProviderPolicyCandidate(input, 64<<10); p != nil || err != ErrProviderPolicyCandidate {
				t.Fatal("accepted malformed raw candidate")
			}
		})
	}
	if p, err := ParseProviderPolicyCandidate(raw, len(raw)-1); p != nil || err != ErrProviderPolicyCandidate {
		t.Fatal("exceeded explicit size ceiling")
	}
	for _, value := range []any{nil, 1, "", "-1", "+1", "01", "1.0", "1e0", "18446744073709551616", strings.Repeat("1", 21)} {
		object := providerPolicyTestObject()
		object["limits"].(map[string]any)["live_disk_bytes"] = value
		providerPolicyTestRefuse(t, object)
	}
	for _, key := range []string{"requests", "wire_bytes", "decoded_bytes", "blocks", "transactions", "frames", "logs", "proofs", "segments", "disk_bytes_written", "work_units", "deadline_ms"} {
		for _, level := range []string{"segment", "bootstrap"} {
			object := providerPolicyTestObject()
			object["limits"].(map[string]any)[level].(map[string]any)[key] = "999"
			providerPolicyTestRefuse(t, object)
		}
	}
	for key, bound := range map[string]uint64{"requests": 65536, "wire_bytes": 256 << 20, "decoded_bytes": 256 << 20, "deadline_ms": 300000} {
		for _, extra := range []uint64{0, 1} {
			object := providerPolicyTestObject()
			for _, level := range []string{"operation", "segment", "bootstrap"} {
				object["limits"].(map[string]any)[level].(map[string]any)[key] = fmt.Sprint(bound + extra)
			}
			p, err := ParseProviderPolicyCandidate(providerPolicyTestBytes(t, object), 64<<10)
			if (extra == 0) != (err == nil && p != nil) {
				t.Fatalf("wrong inclusive ceiling %s/%d: %v", key, extra, err)
			}
		}
	}
	for _, id := range []any{nil, "", strings.Repeat("c", 257), "space clock", "del\x7f", "\u0080"} {
		object := providerPolicyTestObject()
		object["clock"].(map[string]any)["id"] = id
		providerPolicyTestRefuse(t, object)
	}
	for _, carriers := range []any{nil, "protected-legacy", []any{}, []any{"protected-legacy", "access-list"}, []any{"access-list", "protected-legacy", "dynamic-fee"}, []any{"protected-legacy", "access-list", "dynamic-fee", "blob"}} {
		object := providerPolicyTestObject()
		object["carrier_formats"] = carriers
		providerPolicyTestRefuse(t, object)
	}
}

func TestProviderPolicyCandidateCannotSelectOrEnroll(t *testing.T) {
	ns, source := "0x"+strings.Repeat("2", 64), "0x"+strings.Repeat("3", 64)
	p, err := ParseProviderPolicyCandidate(providerPolicyTestBytes(t, providerPolicyTestObject()), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	for _, generation := range []string{"0", "1", "18446744073709551615"} {
		c, err := p.BindCandidate(ns, source, generation)
		if err != nil {
			t.Fatal(err)
		}
		i, err := c.Identity()
		if err != nil || fmt.Sprint(i.Generation) != generation {
			t.Fatal("changed generation")
		}
	}
	for _, generation := range []string{"", "01", "-1", "+1", "18446744073709551616"} {
		if c, err := p.BindCandidate(ns, source, generation); c != nil || err != ErrProviderPolicyCandidate {
			t.Fatal("invalid generation")
		}
	}
	for _, pair := range [][2]string{{"", source}, {ns, ""}, {"0X" + strings.Repeat("2", 64), source}, {ns, "0x" + strings.Repeat("G", 64)}} {
		if c, err := p.BindCandidate(pair[0], pair[1], "0"); c != nil || err != ErrProviderPolicyCandidate {
			t.Fatal("invalid identity")
		}
	}
	for _, zero := range []*ProviderPolicyCandidate{nil, {}} {
		if zero.CanonicalBytes() != nil {
			t.Fatal("zero bytes")
		}
		if c, err := zero.BindCandidate(ns, source, "0"); c != nil || err != ErrProviderPolicyCandidate {
			t.Fatal("zero candidate")
		}
	}
	for _, zero := range []*ProviderCandidateContext{nil, {}} {
		if i, err := zero.Identity(); i != (ProviderCandidateIdentity{}) || err != ErrProviderPolicyCandidate {
			t.Fatal("zero context")
		}
	}
	var forged ProviderCandidateContext
	if json.Unmarshal([]byte(`{"bound":true,"Policy":"0x123","ready":true}`), &forged) != nil {
		t.Fatal("fixture JSON")
	}
	if _, err := forged.Identity(); err != ErrProviderPolicyCandidate {
		t.Fatal("JSON forged a context")
	}
	c, _ := p.BindCandidate(ns, source, "0")
	before, _ := c.Identity()
	object := providerPolicyTestObject()
	object["clock"].(map[string]any)["id"] = "different-fixture-clock"
	other, err := ParseProviderPolicyCandidate(providerPolicyTestBytes(t, object), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	d, _ := other.BindCandidate(source, ns, "1")
	after, _ := d.Identity()
	if before.Policy == after.Policy || before.Instance != after.Instance || before.Provider != after.Provider || before.Namespace == after.Namespace || before.SourceID == after.SourceID || before.Generation == after.Generation {
		t.Fatal("policy-independent instance identity changed")
	}
	// The candidate has no exported authority fields. Serialized metadata cannot
	// turn these private, non-authorizing types into a selected Source.
	for _, value := range []any{p, c} {
		raw, err := json.Marshal(value)
		if err != nil || string(raw) != "{}" {
			t.Fatal("candidate leaked mutable fields")
		}
	}
}

func TestProviderPolicyCandidateConcurrentReads(t *testing.T) {
	p, err := ParseProviderPolicyCandidate(providerPolicyTestBytes(t, providerPolicyTestObject()), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	ns := "0x" + strings.Repeat("2", 64)
	c, err := p.BindCandidate(ns, ns, "3")
	if err != nil {
		t.Fatal(err)
	}
	want, _ := c.Identity()
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			for range 8 {
				raw := p.CanonicalBytes()
				raw[0] = '!'
				got, e := c.Identity()
				if e != nil || got != want || p.CanonicalBytes()[0] != '{' {
					t.Error("concurrent immutable read changed")
				}
			}
		})
	}
	wg.Wait()
}
