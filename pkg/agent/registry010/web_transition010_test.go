package registry010

import (
	"encoding/json"
	"testing"
)

func cloneWebRecord010(t *testing.T, record map[string]any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	var clone map[string]any
	if err := json.Unmarshal(raw, &clone); err != nil {
		t.Fatal(err)
	}
	return clone
}

func TestWebRegistryCreationShape010(t *testing.T) {
	record := proofFixture010(t)
	record["state"] = "created"
	if err := CheckWebRegistryCreationShape010(proofBody010(t, record), proofDID010, 100); err != nil {
		t.Fatal(err)
	}
	record["keys"].([]any)[0].(map[string]any)["state"] = "revoked"
	if err := CheckWebRegistryCreationShape010(proofBody010(t, record), proofDID010, 100); err == nil {
		t.Fatal("creation accepted a KEM endorsed by a revoked signer")
	}
}

func TestWebRegistryTransitionShape010(t *testing.T) {
	before := proofFixture010(t)
	before["state"] = "created"
	after := cloneWebRecord010(t, before)
	after["state"] = "active"
	after["version"] = "2"
	check := func(name string, old, next map[string]any, operation string, valid bool) {
		t.Helper()
		err := CheckWebRegistryTransitionShape010(
			proofBody010(t, old), proofBody010(t, next), proofDID010, 100, 100, operation)
		if (err == nil) != valid {
			t.Errorf("%s: got %v, valid %v", name, err, valid)
		}
	}
	check("activate", before, after, "activate", true)
	wrongVersion := cloneWebRecord010(t, after)
	wrongVersion["version"] = "3"
	check("skip version", before, wrongVersion, "activate", false)
	wrongController := cloneWebRecord010(t, after)
	wrongController["controller"] = "other"
	check("controller transfer", before, wrongController, "activate", false)
	removed := cloneWebRecord010(t, after)
	removed["version"] = "3"
	removed["keys"] = removed["keys"].([]any)[:3]
	check("remove retained key", after, removed, "update-services", false)

	activeWithoutKEM := cloneWebRecord010(t, after)
	keys := activeWithoutKEM["keys"].([]any)
	activeWithoutKEM["keys"] = append(keys[:1:1], keys[2:]...)
	addedKEM := cloneWebRecord010(t, after)
	addedKEM["version"] = "3"
	check("add endorsed KEM", activeWithoutKEM, addedKEM, "add-key", true)
	revokedSignerBefore := cloneWebRecord010(t, activeWithoutKEM)
	revokedSignerBefore["keys"].([]any)[0].(map[string]any)["state"] = "revoked"
	revokedSignerAfter := cloneWebRecord010(t, addedKEM)
	revokedSignerAfter["keys"].([]any)[0].(map[string]any)["state"] = "revoked"
	check("new KEM with revoked signer", revokedSignerBefore, revokedSignerAfter, "add-key", false)

	deactivated := cloneWebRecord010(t, after)
	deactivated["version"] = "3"
	deactivated["state"] = "deactivated"
	check("deactivate", after, deactivated, "deactivate", true)
	check("reactivate", deactivated, after, "activate", false)
}
