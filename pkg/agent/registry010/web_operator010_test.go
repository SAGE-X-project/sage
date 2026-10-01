package registry010

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strconv"
	"sync"
	"testing"
)

func TestWebRegistryOperatorJournal010(t *testing.T) {
	path := filepath.Join(t.TempDir(), "operator.log")
	created := proofFixture010(t)
	created["state"] = "created"
	active := cloneWebRecord010(t, created)
	active["state"] = "active"
	active["version"] = "3"
	authority := &webTestAuthority010{actor: "operator"}
	store, err := OpenWebRegistryWriteJournal010(path, proofDID010, "trusted-web-origin", authority, true)
	if err != nil {
		t.Fatal(err)
	}
	write := func(candidate map[string]any, now int64, version, operation string) error {
		return ApplyWebRegistryWrite010(context.Background(), store, "trusted-web-origin", proofDID010,
			webJournalEnvelopeAt010(t, candidate, now), now, version, operation)
	}
	command := func(now int64, version, operation, target, scope string) error {
		return ApplyWebRegistryOperatorCommand010(context.Background(), store, "trusted-web-origin", proofDID010,
			now, version, operation, target, scope)
	}
	if err := write(created, 100, "", "create"); err != nil {
		t.Fatal(err)
	}
	if err := command(101, "1", "authorize-operator", "assistant", "activate"); err != nil {
		t.Fatal(err)
	}
	granted := store.Inspect()
	if len(granted.Grants) != 1 || len(granted.History) != 2 || granted.History[1].Target != "assistant" ||
		granted.History[1].Scope != "activate" || granted.History[1].Operation != "authorize-operator" {
		t.Fatal("grant and management history were not committed together")
	}
	if err := command(102, "2", "authorize-operator", "assistant", "activate"); !errors.Is(err, ErrRejected) ||
		!reflect.DeepEqual(granted, store.Inspect()) {
		t.Fatalf("duplicate grant changed state: %v", err)
	}
	authority.actor = "assistant"
	if err := write(active, 102, "2", "activate"); err != nil {
		t.Fatalf("scoped operator activation: %v", err)
	}
	retired := store.Inspect()
	if len(retired.Grants) != 0 || len(retired.History) != 3 || retired.History[2].Actor != "assistant" {
		t.Fatal("activation did not retire its grant atomically")
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = OpenWebRegistryWriteJournal010(path, proofDID010, "trusted-web-origin", authority, false)
	if err != nil {
		t.Fatalf("restart lost grant history: %v", err)
	}
	if !reflect.DeepEqual(retired, store.Inspect()) {
		t.Fatal("restart changed committed state")
	}
	if err := command(103, "3", "authorize-operator", "assistant", "update-services"); !errors.Is(err, ErrRejected) {
		t.Fatalf("operator delegated further: %v", err)
	}
	authority.actor = "operator"
	if err := command(103, "2", "authorize-operator", "assistant", "update-services"); !errors.Is(err, ErrStale) {
		t.Fatalf("stale grant: %v", err)
	}
	if err := command(103, "3", "authorize-operator", "assistant", "update-services"); err != nil {
		t.Fatal(err)
	}
	if err := command(104, "4", "revoke-operator", "assistant", "update-services"); err != nil {
		t.Fatal(err)
	}
	if err := command(105, "5", "revoke-operator", "assistant", "update-services"); !errors.Is(err, ErrRejected) {
		t.Fatalf("absent grant revoke: %v", err)
	}
	if len(store.Inspect().Grants) != 0 || len(store.Inspect().History) != 5 {
		t.Fatal("revoke did not preserve exact history")
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestWebRegistryOperatorConcurrentVersion010(t *testing.T) {
	created := proofFixture010(t)
	created["state"] = "created"
	store := &webMemoryTransaction010{
		state: WebRegistryWriteState010{Source: "trusted-web-origin"}, authority: &webTestAuthority010{actor: "operator"},
	}
	if err := ApplyWebRegistryWrite010(context.Background(), store, "trusted-web-origin", proofDID010,
		proofBody010(t, created), 100, "", "create"); err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	var workers sync.WaitGroup
	for _, target := range []string{"assistant-a", "assistant-b"} {
		workers.Add(1)
		go func(target string) {
			defer workers.Done()
			results <- ApplyWebRegistryOperatorCommand010(context.Background(), store, "trusted-web-origin", proofDID010,
				101, "1", "authorize-operator", target, "activate")
		}(target)
	}
	workers.Wait()
	close(results)
	accepted, stale := 0, 0
	for err := range results {
		switch {
		case err == nil:
			accepted++
		case errors.Is(err, ErrStale):
			stale++
		default:
			t.Fatalf("unexpected concurrent result: %v", err)
		}
	}
	if accepted != 1 || stale != 1 || len(store.state.Grants) != 1 || len(store.state.History) != 2 {
		t.Fatalf("commands both committed: accepted=%d stale=%d", accepted, stale)
	}
}

func TestWebRegistryOperatorGrantBoundary010(t *testing.T) {
	before := webTransitionRecord010{Controller: "controller", State: "created"}
	after := before
	entry := WebRegistryHistoryEntry010{Operation: "authorize-operator", Actor: "controller", Target: "assistant", Scope: "activate"}
	grants, err := webNextGrants010(nil, before, after, entry)
	if err != nil || len(grants) != 1 {
		t.Fatalf("valid grant: %v", err)
	}
	for _, change := range []WebRegistryHistoryEntry010{
		{Operation: "authorize-operator", Actor: "assistant", Target: "other", Scope: "activate"},
		{Operation: "authorize-operator", Actor: "controller", Target: "controller", Scope: "activate"},
		{Operation: "authorize-operator", Actor: "controller", Target: "assistant", Scope: "add-key"},
		{Operation: "revoke-operator", Actor: "controller", Target: "missing", Scope: "activate"},
	} {
		if _, err := webNextGrants010(grants, before, after, change); err == nil {
			t.Fatalf("invalid command accepted: %+v", change)
		}
	}
	active := before
	active.State = "active"
	retired, err := webNextGrants010(grants, before, active, WebRegistryHistoryEntry010{Operation: "activate", Actor: "assistant"})
	if err != nil || len(retired) != 0 {
		t.Fatalf("state transition retained invalid grant: %v", err)
	}
	activeGrant := []WebRegistryOperatorGrant010{{Operator: "assistant", Scope: "deactivate"}}
	terminal := active
	terminal.State = "deactivated"
	retired, err = webNextGrants010(activeGrant, active, terminal,
		WebRegistryHistoryEntry010{Operation: "deactivate", Actor: "assistant"})
	if err != nil || len(retired) != 0 {
		t.Fatalf("deactivation retained authority: %v", err)
	}
	full := make([]WebRegistryOperatorGrant010, 128)
	for i := range full {
		full[i] = WebRegistryOperatorGrant010{Operator: "actor-" + strconv.Itoa(i+1000), Scope: "activate"}
	}
	if _, err := webNextGrants010(full, before, after, WebRegistryHistoryEntry010{
		Operation: "authorize-operator", Actor: "controller", Target: "overflow", Scope: "activate",
	}); err == nil {
		t.Fatal("129th active grant accepted")
	}
}

func TestWebRegistryOperatorRejectsUncommittedGrant010(t *testing.T) {
	created := proofFixture010(t)
	created["state"] = "created"
	store := &webMemoryTransaction010{
		state: WebRegistryWriteState010{Source: "trusted-web-origin"}, authority: &webTestAuthority010{actor: "operator"},
	}
	if err := ApplyWebRegistryWrite010(context.Background(), store, "trusted-web-origin", proofDID010,
		proofBody010(t, created), 100, "", "create"); err != nil {
		t.Fatal(err)
	}
	store.state.Grants = []WebRegistryOperatorGrant010{{Operator: "assistant", Scope: "activate"}}
	before := webCopyWriteState010(store.state)
	if err := ApplyWebRegistryOperatorCommand010(context.Background(), store, "trusted-web-origin", proofDID010,
		101, "1", "authorize-operator", "other", "activate"); !errors.Is(err, ErrInvalidRecord010) ||
		!reflect.DeepEqual(before, store.state) {
		t.Fatalf("uncommitted positive grant was trusted: %v", err)
	}
}
