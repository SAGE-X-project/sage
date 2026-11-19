//go:build registry_dev_live

package gatesource

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/registry010"
	"github.com/sage-x-project/sage/pkg/agent/registry011"
)

type monoClock struct{ start time.Time }

func (c monoClock) Now() (registry010.Stamp, error) {
	now := time.Now()
	return registry010.Stamp{MonoMS: now.Sub(c.start).Milliseconds(), Unix: now.Unix()}, nil
}

// TestGateOnLocalChain needs a chain Registry on Anvil with an agent left
// active by `registry-lifecycle-scenarios.mjs --keep-active`, finalized:
//
//	SAGE_DEV_RPC, SAGE_DEV_ROOT, SAGE_DEV_GENESIS, SAGE_DEV_DEPLOYMENT as for registry011,
//	SAGE_DEV_AGENT  the agent identifier used by the scenario.
func TestGateOnLocalChain(t *testing.T) {
	env := func(name string) string {
		v := os.Getenv(name)
		if v == "" {
			t.Fatalf("%s is required", name)
		}
		return v
	}
	root, err := os.ReadFile(env("SAGE_DEV_ROOT"))
	if err != nil {
		t.Fatal(err)
	}
	record, err := os.ReadFile(env("SAGE_DEV_DEPLOYMENT"))
	if err != nil {
		t.Fatal(err)
	}
	network, err := registry011.NewDevNetwork(registry011.DevNetworkConfig{Endpoint: env("SAGE_DEV_RPC"), RootsPEM: root,
		GenesisHash: env("SAGE_DEV_GENESIS"), DeploymentRecord: record})
	if err != nil {
		t.Fatal(err)
	}
	rpc, err := registry011.NewReadRPC(network, registry011.RPCTransportLimits{RequestTimeout: 20 * time.Second, MaxBytes: 4 << 20, MaxDepth: 128})
	if err != nil {
		t.Fatal(err)
	}
	defer rpc.Close()
	clock := monoClock{start: time.Now().Add(-time.Second)}
	// An idle Anvil mines no blocks, so its finalized block can be old.
	src, err := NewDevelopment("anvil-dev", rpc, network, clock, Limits{MaxFinalizedAge: 24 * 3600})
	if err != nil {
		t.Fatal(err)
	}
	gate, err := registry010.NewGate(src.Config(), src, clock, store{})
	if err != nil {
		t.Fatal(err)
	}
	did := "did:sage:" + network.RegistryID() + ":" + env("SAGE_DEV_AGENT")
	ctx := context.Background()

	pinned, err := gate.Select(ctx, did, did+"#sig-3", true)
	if err != nil {
		t.Fatalf("select sig-3 with a KEM key: %v", err)
	}
	t.Logf("selected signing %s, KEM %s", pinned.Signing().Name, pinned.KEM().Name)
	if err := gate.CheckPinned(ctx, pinned); err != nil {
		t.Fatalf("recheck: %v", err)
	}
	for _, leaked := range []string{"#sig-1", "#sig-2"} {
		if _, err := gate.Select(ctx, did, did+leaked, false); !errors.Is(err, registry010.ErrRejected) {
			t.Fatalf("revoked %s selected: %v", leaked, err)
		}
	}
}
