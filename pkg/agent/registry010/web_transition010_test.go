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

func TestWebRegistryHistoryContinuity010(t *testing.T) {
	created := proofFixture010(t)
	created["state"] = "created"
	active := cloneWebRecord010(t, created)
	active["state"] = "active"
	active["version"] = "2"
	updated := cloneWebRecord010(t, active)
	updated["version"] = "3"
	updated["services"] = []any{map[string]any{
		"name": "api", "type": "Agent", "uri": "https://agents.example.com/api",
	}}
	entry := func(record map[string]any, at int64, operation string) WebRegistryHistoryEntry010 {
		return WebRegistryHistoryEntry010{Envelope: proofBody010(t, record), At: at, Operation: operation}
	}
	history := []WebRegistryHistoryEntry010{
		entry(created, 100, "create"), entry(active, 101, "activate"),
		entry(updated, 102, "update-services"),
	}
	current := proofBody010(t, updated)
	check := func(name string, entries []WebRegistryHistoryEntry010, live []byte, valid bool) {
		t.Helper()
		err := CheckWebRegistryHistoryContinuity010(entries, live, proofDID010, 103)
		if (err == nil) != valid {
			t.Errorf("%s: got %v, valid %v", name, err, valid)
		}
	}
	check("complete asserted chain", history, current, true)
	check("empty", nil, current, false)
	check("missing creation", history[1:], current, false)
	check("skipped version", []WebRegistryHistoryEntry010{history[0], history[2]}, current, false)
	check("stale current", history, proofBody010(t, active), false)
	regressed := append([]WebRegistryHistoryEntry010(nil), history...)
	regressed[2].At = 99
	check("time regression", regressed, current, false)
	wrongOperation := append([]WebRegistryHistoryEntry010(nil), history...)
	wrongOperation[2].Operation = "add-key"
	check("wrong operation", wrongOperation, current, false)
	terminal := cloneWebRecord010(t, updated)
	terminal["version"] = "4"
	terminal["state"] = "deactivated"
	postTerminal := cloneWebRecord010(t, terminal)
	postTerminal["version"] = "5"
	check("post-terminal mutation", append(append([]WebRegistryHistoryEntry010(nil), history...),
		entry(terminal, 103, "deactivate"), entry(postTerminal, 103, "update-services")),
		proofBody010(t, postTerminal), false)
}
