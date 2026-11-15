package registrykeys

import (
	"context"
	"errors"
	"testing"
)

func TestClassifier(t *testing.T) {
	ctx := context.Background()
	present := &Classifier{Source: &source{present: true}, RegistryID: registryID}
	if sage, err := present.RequiresSAGE(ctx, agentDID, ""); err != nil || !sage {
		t.Fatalf("registered agent not a SAGE peer: %v", err)
	}
	absent := &Classifier{Source: &source{}, RegistryID: registryID}
	if sage, err := absent.RequiresSAGE(ctx, agentDID, ""); err != nil || sage {
		t.Fatal("unregistered agent treated as SAGE peer")
	}
	if sage, err := present.RequiresSAGE(ctx, "did:web:example.invalid", ""); err != nil || sage {
		t.Fatal("foreign DID treated as SAGE peer")
	}
	failing := &Classifier{Source: &source{err: errors.New("rpc down")}, RegistryID: registryID}
	if _, err := failing.RequiresSAGE(ctx, agentDID, ""); err == nil {
		t.Fatal("source failure hidden")
	}
	var unconfigured *Classifier
	if _, err := unconfigured.RequiresSAGE(ctx, agentDID, ""); err == nil {
		t.Fatal("nil classifier answered")
	}
}
