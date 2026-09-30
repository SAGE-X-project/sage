package registry010

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func webJournalEnvelopeAt010(t *testing.T, record map[string]any, at int64) []byte {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"record": record, "issued": at, "expires": at + 5})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestWebRegistryWriteJournalRestart010(t *testing.T) {
	path := filepath.Join(t.TempDir(), "writes.log")
	created := proofFixture010(t)
	created["state"] = "created"
	active := cloneWebRecord010(t, created)
	active["state"] = "active"
	active["version"] = "2"
	terminal := cloneWebRecord010(t, active)
	terminal["state"] = "deactivated"
	terminal["version"] = "3"
	authority := &webTestAuthority010{actor: "operator"}
	open := func(create bool) *WebRegistryWriteJournal010 {
		store, err := OpenWebRegistryWriteJournal010(path, proofDID010, "trusted-web-origin", authority, create)
		if err != nil {
			t.Fatal(err)
		}
		return store
	}
	apply := func(store *WebRegistryWriteJournal010, record map[string]any, now int64, version, operation string) error {
		return ApplyWebRegistryWrite010(context.Background(), store, "trusted-web-origin", proofDID010,
			webJournalEnvelopeAt010(t, record, now), now, version, operation)
	}
	store := open(true)
	if err := apply(store, created, 100, "", "create"); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenWebRegistryWriteJournal010(path, proofDID010, "trusted-web-origin", authority, false); err == nil {
		t.Fatal("second writer acquired the same journal")
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store = open(false)
	if len(store.state.History) != 1 {
		t.Fatal("creation lost after restart")
	}
	if err := apply(store, active, 200, "1", "activate"); err != nil {
		t.Fatalf("expired response was not refreshed: %v", err)
	}
	if err := apply(store, terminal, 300, "1", "deactivate"); !errors.Is(err, ErrStale) || len(store.state.History) != 2 {
		t.Fatalf("stale write changed journal: %v", err)
	}
	if err := apply(store, terminal, 300, "2", "deactivate"); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store = open(false)
	if len(store.state.History) != 3 || !store.state.Tombstoned {
		t.Fatal("terminal state lost after restart")
	}
	if err := apply(store, created, 400, "", "create"); err == nil || len(store.state.History) != 3 {
		t.Fatal("terminal identifier was reused")
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenWebRegistryWriteJournal010(path, proofDID010, "other-origin", authority, false); err == nil {
		t.Fatal("journal opened under a different source")
	}
}

func TestWebRegistryWriteJournalIncompleteState010(t *testing.T) {
	path := filepath.Join(t.TempDir(), "writes.log")
	authority := &webTestAuthority010{actor: "operator"}
	store, err := OpenWebRegistryWriteJournal010(path, proofDID010, "trusted-web-origin", authority, true)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = file.WriteString("{\"source\":"); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = OpenWebRegistryWriteJournal010(path, proofDID010, "trusted-web-origin", authority, false); err == nil {
		t.Fatal("incomplete journal was accepted")
	}
}

func TestWebRegistryWriteJournalQuarantine010(t *testing.T) {
	path := filepath.Join(t.TempDir(), "writes.log")
	authority := &webTestAuthority010{actor: "operator"}
	store, err := OpenWebRegistryWriteJournal010(path, proofDID010, "trusted-web-origin", authority, true)
	if err != nil {
		t.Fatal(err)
	}
	store.failed = true // deterministic I/O-failure state without corrupting the file
	if err = store.Close(); !errors.Is(err, ErrUnreachable) {
		t.Fatalf("failed store close: %v", err)
	}
	if _, err = OpenWebRegistryWriteJournal010(path, proofDID010, "trusted-web-origin", authority, false); err == nil {
		t.Fatal("quarantined journal lock was lost")
	}
}

func TestWebRegistryWriteJournalExpiredKeyRecovery010(t *testing.T) {
	path := filepath.Join(t.TempDir(), "writes.log")
	base := proofFixture010(t)
	created := cloneWebRecord010(t, base)
	created["state"] = "created"
	created["keys"] = created["keys"].([]any)[:2]
	created["keys"].([]any)[0].(map[string]any)["expires"] = float64(101)
	active := cloneWebRecord010(t, created)
	active["state"] = "active"
	active["version"] = "2"
	repaired := cloneWebRecord010(t, active)
	repaired["version"] = "3"
	repaired["keys"] = append(repaired["keys"].([]any), base["keys"].([]any)[2])
	authority := &webTestAuthority010{actor: "operator"}
	store, err := OpenWebRegistryWriteJournal010(path, proofDID010, "trusted-web-origin", authority, true)
	if err != nil {
		t.Fatal(err)
	}
	apply := func(record map[string]any, at int64, version, operation string) error {
		return ApplyWebRegistryWrite010(context.Background(), store, "trusted-web-origin", proofDID010,
			webJournalEnvelopeAt010(t, record, at), at, version, operation)
	}
	if err = apply(created, 100, "", "create"); err != nil {
		t.Fatal(err)
	}
	if err = apply(active, 100, "1", "activate"); err != nil {
		t.Fatal(err)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = OpenWebRegistryWriteJournal010(path, proofDID010, "trusted-web-origin", authority, false)
	if err != nil {
		t.Fatal(err)
	}
	if err = apply(repaired, 200, "2", "add-key"); err != nil {
		t.Fatalf("expired signer recovery after restart: %v", err)
	}
	if len(store.Inspect().History) != 3 {
		t.Fatal("recovery history not committed")
	}
	if err = CheckWebRegistryProofs010(store.Inspect().Envelope, proofDID010, 200); err != nil {
		t.Fatalf("repaired record unusable: %v", err)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
}
