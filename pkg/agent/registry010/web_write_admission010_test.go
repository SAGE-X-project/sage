package registry010

import (
	"context"
	"errors"
	"testing"
)

type webTestAuthority010 struct {
	actor  string
	scopes map[string]bool
	fail   bool
	calls  int
}

func (a *webTestAuthority010) AuthenticatedActor(context.Context) (string, error) {
	if a.fail {
		return "", errors.New("credential unavailable")
	}
	return a.actor, nil
}

func (a *webTestAuthority010) Delegated(_ context.Context, controller, actor, did, operation, version string) (bool, error) {
	a.calls++
	return controller == "operator" && actor == "assistant" &&
		did == proofDID010 && version == "1" && a.scopes[operation], nil
}

func TestWebRegistryWriteAdmission010(t *testing.T) {
	created := proofFixture010(t)
	created["state"] = "created"
	active := cloneWebRecord010(t, created)
	active["state"] = "active"
	active["version"] = "2"
	before, after := proofBody010(t, created), proofBody010(t, active)
	ctx := context.Background()
	authority := &webTestAuthority010{actor: "operator"}
	if err := CheckWebRegistryCreationAdmission010(ctx, authority, before, proofDID010, 100); err != nil {
		t.Fatal(err)
	}
	if err := CheckWebRegistryMutationAdmission010(ctx, authority, before, after, proofDID010, 100, "1", "activate"); err != nil {
		t.Fatal(err)
	}
	if authority.calls != 0 {
		t.Fatal("controller required delegated scope lookup")
	}
	if err := CheckWebRegistryMutationAdmission010(ctx, authority, before, after, proofDID010, 100, "2", "activate"); !errors.Is(err, ErrStale) {
		t.Fatalf("wrong expected version: %v", err)
	}
	authority.actor = "assistant"
	authority.scopes = map[string]bool{"activate": true}
	if err := CheckWebRegistryCreationAdmission010(ctx, authority, before, proofDID010, 100); !errors.Is(err, ErrRejected) {
		t.Fatalf("non-controller creation: %v", err)
	}
	if err := CheckWebRegistryMutationAdmission010(ctx, authority, before, after, proofDID010, 100, "1", "activate"); err != nil {
		t.Fatal(err)
	}
	if authority.calls != 1 {
		t.Fatal("delegated scope was not checked")
	}
	authority.scopes = nil
	if err := CheckWebRegistryMutationAdmission010(ctx, authority, before, after, proofDID010, 100, "1", "activate"); !errors.Is(err, ErrRejected) {
		t.Fatalf("missing scope: %v", err)
	}
	authority.scopes = map[string]bool{"activate": true}
	wrong := cloneWebRecord010(t, active)
	wrong["version"] = "3"
	if err := CheckWebRegistryMutationAdmission010(ctx, authority, before, proofBody010(t, wrong), proofDID010, 100, "1", "activate"); err == nil {
		t.Fatal("invalid mutation admitted")
	}
	authority.fail = true
	if err := CheckWebRegistryMutationAdmission010(ctx, authority, before, after, proofDID010, 100, "1", "activate"); !errors.Is(err, ErrRejected) {
		t.Fatalf("unavailable credentials: %v", err)
	}
}

func TestWebRegistryExpiredSigningKeyRecovery010(t *testing.T) {
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
	before, after := proofBody010(t, active), proofBody010(t, repaired)
	if err := CheckWebRegistryCreationShape010(proofBody010(t, created), proofDID010, 100); err != nil {
		t.Fatal(err)
	}
	if err := CheckWebRegistryTransitionShape010(proofBody010(t, created), before, proofDID010, 100, 100, "activate"); err != nil {
		t.Fatal(err)
	}
	if err := CheckWebRegistryProofs010(before, proofDID010, 102); !errors.Is(err, ErrInvalidRecord010) {
		t.Fatalf("expired active read: %v", err)
	}
	if err := CheckWebRegistryTransitionShape010(before, after, proofDID010, 100, 102, "add-key"); err != nil {
		t.Fatalf("historically valid transition: %v", err)
	}
	if err := CheckWebRegistryTransitionShape010(before, after, proofDID010, 102, 102, "add-key"); !errors.Is(err, ErrInvalidRecord010) {
		t.Fatalf("untrusted expired history: %v", err)
	}
	authority := &webTestAuthority010{actor: "operator"}
	if err := CheckWebRegistryMutationAdmission010(context.Background(), authority, before, after,
		proofDID010, 102, "2", "add-key"); err != nil {
		t.Fatalf("controller recovery: %v", err)
	}
	if err := CheckWebRegistryProofs010(after, proofDID010, 102); err != nil {
		t.Fatalf("repaired active read: %v", err)
	}
	broken := cloneWebRecord010(t, active)
	broken["keys"].([]any)[0].(map[string]any)["proof"].(map[string]any)["value"] = "invalid"
	if err := CheckWebRegistryMutationAdmission010(context.Background(), authority,
		proofBody010(t, broken), after, proofDID010, 102, "2", "add-key"); !errors.Is(err, ErrInvalidRecord010) {
		t.Fatalf("invalid prior proof: %v", err)
	}
	revoked := cloneWebRecord010(t, active)
	revoked["keys"].([]any)[0].(map[string]any)["state"] = "revoked"
	if err := CheckWebRegistryMutationAdmission010(context.Background(), authority,
		proofBody010(t, revoked), after, proofDID010, 102, "2", "add-key"); !errors.Is(err, ErrInvalidRecord010) {
		t.Fatalf("active prior record without accepted signer: %v", err)
	}
}
