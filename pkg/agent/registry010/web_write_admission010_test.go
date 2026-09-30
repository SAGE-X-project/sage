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
