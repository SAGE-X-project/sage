//go:build registry_dev_live

package registrykeys

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/registry011"
	"github.com/sage-x-project/sage/pkg/agent/wire011"
)

type wallClock struct{}

func (wallClock) Now() (time.Time, error) { return time.Now(), nil }

type serverKeys struct {
	resolver *Resolver
	server   map[string]ed25519.PublicKey
}

func (k serverKeys) ResolveSigningKey(ctx context.Context, kid string) (ed25519.PublicKey, error) {
	if key, ok := k.server[kid]; ok {
		return key, nil
	}
	return nil, errors.New("unknown server key")
}

// ownedSeed mirrors sage-contracts owned-fixtures.mjs secret(index): the
// disclosed public recipe the lifecycle scenario uses for its agent keys.
func ownedSeed(index string) []byte {
	sum := sha256.Sum256([]byte("SAGE owned constructor unit public key recipe " + index + " false"))
	return sum[:]
}

// TestRegistryKeysOnLocalChain needs a chain Registry on Anvil with an agent
// left active by `registry-lifecycle-scenarios.mjs --keep-active`, finalized:
//
//	SAGE_DEV_RPC, SAGE_DEV_ROOT, SAGE_DEV_GENESIS, SAGE_DEV_DEPLOYMENT as for registry011,
//	SAGE_DEV_AGENT  the agent identifier used by the scenario.
func TestRegistryKeysOnLocalChain(t *testing.T) {
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
	resolver := NewRegistryResolver(rpc, network, wallClock{})
	did := "did:sage:" + network.RegistryID() + ":" + env("SAGE_DEV_AGENT")
	ctx := context.Background()

	agentKey := ed25519.NewKeyFromSeed(ownedSeed("103")) // sig-3, the replacement after the leak
	got, err := resolver.ResolveSigningKey(ctx, did+"#sig-3")
	if err != nil || !bytes.Equal(got, agentKey.Public().(ed25519.PublicKey)) {
		t.Fatalf("sig-3 from the Registry: %v", err)
	}
	for kid, reason := range map[string]string{did + "#sig-1": "revoked", did + "#sig-2": "revoked", did + "#kem-2": "not a signing key"} {
		if _, err := resolver.ResolveSigningKey(ctx, kid); err == nil || !strings.Contains(err.Error(), reason) {
			t.Fatalf("%s: got %v, want %q", kid, err, reason)
		}
	}

	// A message signed with the Registry key verifies; one signed with the
	// leaked, revoked sig-1 does not.
	serverPub, serverPriv, _ := ed25519.GenerateKey(nil)
	serverDID := "did:sage:" + network.RegistryID() + ":resource-server"
	receiver := &wire011.Receiver{DID: serverDID, Endpoint: "https://resource.example.invalid/pay", Keys: resolver,
		Clock: wallClock{}, Replay: &wire011.MemoryReplay{Capacity: 64, Clock: wallClock{}, Started: time.Now().Add(-wire011.Quarantine)}}
	agent := &wire011.Signer{DID: did, KID: did + "#sig-3", Key: agentKey, Clock: wallClock{}, TTL: 60}
	req, sent, err := agent.SignRequest(serverDID, receiver.Endpoint, []byte(`{"tool":"pay"}`), wire011.RequestOptions{})
	if err != nil {
		t.Fatal(err)
	}
	verified, err := receiver.VerifyRequest(ctx, *req)
	if err != nil {
		t.Fatal("request signed with the Registry key refused")
	}
	server := &wire011.Signer{DID: serverDID, KID: serverDID + "#sig-1", Key: serverPriv, Clock: wallClock{}, TTL: 60}
	resp, err := server.SignResponse(verified, 200, true, []byte(`{"ok":true}`), "")
	if err != nil {
		t.Fatal(err)
	}
	responses := &wire011.ResponseVerifier{Keys: serverKeys{server: map[string]ed25519.PublicKey{server.KID: serverPub}},
		Clock: wallClock{}, Replay: &wire011.MemoryReplay{Capacity: 64, Clock: wallClock{}, Started: time.Now().Add(-wire011.Quarantine)}}
	if _, err := responses.VerifyResponse(ctx, sent, *resp); err != nil {
		t.Fatal("response refused")
	}
	leaked := &wire011.Signer{DID: did, KID: did + "#sig-1", Key: ed25519.NewKeyFromSeed(ownedSeed("101")), Clock: wallClock{}, TTL: 60}
	stolen, _, err := leaked.SignRequest(serverDID, receiver.Endpoint, []byte(`{"tool":"pay"}`), wire011.RequestOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := receiver.VerifyRequest(ctx, *stolen); err == nil {
		t.Fatal("request signed with the revoked leaked key accepted")
	}
}
