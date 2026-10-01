package registry010

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
)

type webMemoryTransaction010 struct {
	mu        sync.Mutex
	state     WebRegistryWriteState010
	authority WebRegistryAdminAuthority010
}

func (s *webMemoryTransaction010) Update(ctx context.Context, _ string, _ int64, decide func(WebRegistryWriteSnapshot010) (WebRegistryWriteState010, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next, err := decide(WebRegistryWriteSnapshot010{
		State: webCopyWriteState010(s.state), Authority: s.authority,
	})
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		return ErrUnreachable
	}
	s.state = webCopyWriteState010(next)
	return nil
}

func TestWebRegistryTransactionalWrite010(t *testing.T) {
	base := proofFixture010(t)
	created := cloneWebRecord010(t, base)
	created["state"] = "created"
	active := cloneWebRecord010(t, created)
	active["state"] = "active"
	active["version"] = "2"
	terminal := cloneWebRecord010(t, active)
	terminal["state"] = "deactivated"
	terminal["version"] = "3"
	create, activate, deactivate := proofBody010(t, created), proofBody010(t, active), proofBody010(t, terminal)
	store := &webMemoryTransaction010{
		state:     WebRegistryWriteState010{Source: "trusted-web-origin"},
		authority: &webTestAuthority010{actor: "operator"},
	}
	apply := func(candidate []byte, version, operation string) error {
		return ApplyWebRegistryWrite010(context.Background(), store, "trusted-web-origin",
			proofDID010, candidate, 100, version, operation)
	}
	if err := apply(create, "", "create"); err != nil {
		t.Fatal(err)
	}
	if len(store.state.History) != 1 || store.state.Tombstoned {
		t.Fatal("creation did not commit record and history together")
	}
	before := webCopyWriteState010(store.state)
	if err := apply(activate, "2", "activate"); !errors.Is(err, ErrStale) {
		t.Fatalf("stale write: %v", err)
	}
	if !reflect.DeepEqual(store.state, before) {
		t.Fatal("rejected write changed state")
	}
	if err := apply(activate, "1", "activate"); err != nil {
		t.Fatal(err)
	}
	if len(store.state.History) != 2 || store.state.Tombstoned {
		t.Fatal("activation state incomplete")
	}
	if err := apply(deactivate, "2", "deactivate"); err != nil {
		t.Fatal(err)
	}
	if len(store.state.History) != 3 || !store.state.Tombstoned {
		t.Fatal("terminal tombstone and history not committed")
	}
	before = webCopyWriteState010(store.state)
	if err := apply(create, "", "create"); err == nil || !reflect.DeepEqual(store.state, before) {
		t.Fatal("terminal identifier reused or changed")
	}
}

func TestWebRegistryTransactionSourceAndHistory010(t *testing.T) {
	created := proofFixture010(t)
	created["state"] = "created"
	active := cloneWebRecord010(t, created)
	active["state"] = "active"
	active["version"] = "2"
	create, activate := proofBody010(t, created), proofBody010(t, active)
	store := &webMemoryTransaction010{
		state:     WebRegistryWriteState010{Source: "other-origin"},
		authority: &webTestAuthority010{actor: "operator"},
	}
	apply := func(version, operation string, candidate []byte) error {
		return ApplyWebRegistryWrite010(context.Background(), store, "trusted-web-origin",
			proofDID010, candidate, 100, version, operation)
	}
	if err := apply("", "create", create); !errors.Is(err, ErrUnreachable) || len(store.state.History) != 0 {
		t.Fatalf("wrong source changed state: %v", err)
	}
	store.state.Source = "trusted-web-origin"
	if err := apply("", "create", create); err != nil {
		t.Fatal(err)
	}
	store.state.History[0].Envelope = activate
	before := webCopyWriteState010(store.state)
	if err := apply("1", "activate", activate); !errors.Is(err, ErrInvalidRecord010) || !reflect.DeepEqual(store.state, before) {
		t.Fatalf("inconsistent history changed state: %v", err)
	}
	store.state.History[0].Envelope = create
	store.authority = &webTestAuthority010{actor: "assistant"}
	before = webCopyWriteState010(store.state)
	if err := apply("1", "activate", activate); !errors.Is(err, ErrRejected) || !reflect.DeepEqual(store.state, before) {
		t.Fatalf("untrusted authority changed state: %v", err)
	}
}

func TestWebRegistryTransactionSerializesWriters010(t *testing.T) {
	created := proofFixture010(t)
	created["state"] = "created"
	active := cloneWebRecord010(t, created)
	active["state"] = "active"
	active["version"] = "2"
	store := &webMemoryTransaction010{
		state:     WebRegistryWriteState010{Source: "trusted-web-origin"},
		authority: &webTestAuthority010{actor: "operator"},
	}
	if err := ApplyWebRegistryWrite010(context.Background(), store, "trusted-web-origin", proofDID010,
		proofBody010(t, created), 100, "", "create"); err != nil {
		t.Fatal(err)
	}
	candidate := proofBody010(t, active)
	results := make(chan error, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			results <- ApplyWebRegistryWrite010(context.Background(), store, "trusted-web-origin",
				proofDID010, candidate, 100, "1", "activate")
		}()
	}
	workers.Wait()
	close(results)
	accepted, stale := 0, 0
	for err := range results {
		if err == nil {
			accepted++
		} else if errors.Is(err, ErrStale) {
			stale++
		} else {
			t.Fatalf("unexpected concurrent result: %v", err)
		}
	}
	if accepted != 1 || stale != 1 || len(store.state.History) != 2 {
		t.Fatalf("non-atomic concurrent update: accepted=%d stale=%d history=%d",
			accepted, stale, len(store.state.History))
	}
}

func TestWebRegistryTransactionRecoversExpiredSigner010(t *testing.T) {
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
	create, before, after := proofBody010(t, created), proofBody010(t, active), proofBody010(t, repaired)
	store := &webMemoryTransaction010{
		state: WebRegistryWriteState010{
			Source: "trusted-web-origin", Envelope: before,
			History: []WebRegistryHistoryEntry010{
				{Envelope: create, At: 100, Operation: "create", Actor: "operator"},
				{Envelope: before, At: 100, Operation: "activate", Actor: "operator"},
			},
		},
		authority: &webTestAuthority010{actor: "operator"},
	}
	if err := ApplyWebRegistryWrite010(context.Background(), store, "trusted-web-origin",
		proofDID010, after, 102, "2", "add-key"); err != nil {
		t.Fatalf("expired signer recovery: %v", err)
	}
	if len(store.state.History) != 3 || store.state.Tombstoned {
		t.Fatal("recovery did not commit complete state")
	}
	if err := CheckWebRegistryProofs010(store.state.Envelope, proofDID010, 102); err != nil {
		t.Fatalf("repaired record unusable: %v", err)
	}
	store.state.Tombstoned = true
	old := webCopyWriteState010(store.state)
	if err := ApplyWebRegistryWrite010(context.Background(), store, "trusted-web-origin",
		proofDID010, after, 102, "3", "add-key"); !errors.Is(err, ErrInvalidRecord010) || !reflect.DeepEqual(store.state, old) {
		t.Fatalf("inconsistent tombstone changed state: %v", err)
	}
}
