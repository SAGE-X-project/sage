//go:build registry_dev_live

package registry011

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestDevNetworkObservesLocalDeployment reads a chain Registry deployed on a
// local node through the same ReadRPC, pin and deployment-observation path
// the Sepolia profile uses. It needs:
//
//	SAGE_DEV_RPC         https URL of a loopback TLS endpoint in front of the node
//	SAGE_DEV_ROOT        PEM root certificate for that endpoint
//	SAGE_DEV_GENESIS     genesis block hash of the node
//	SAGE_DEV_DEPLOYMENT  record written by sage-contracts deploy-chain-registry.mjs
func TestDevNetworkObservesLocalDeployment(t *testing.T) {
	read := func(name string) string {
		value := os.Getenv(name)
		if value == "" {
			t.Fatalf("%s is required", name)
		}
		return value
	}
	root, err := os.ReadFile(read("SAGE_DEV_ROOT"))
	if err != nil {
		t.Fatal(err)
	}
	record, err := os.ReadFile(read("SAGE_DEV_DEPLOYMENT"))
	if err != nil {
		t.Fatal(err)
	}
	network, err := NewDevNetwork(DevNetworkConfig{Endpoint: read("SAGE_DEV_RPC"), RootsPEM: root,
		GenesisHash: read("SAGE_DEV_GENESIS"), DeploymentRecord: record})
	if err != nil {
		t.Fatal("profile", err)
	}
	if network.Qualifying() {
		t.Fatal("a development profile must never qualify")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	client, err := NewReadRPC(network, RPCTransportLimits{20 * time.Second, 4 << 20, 128})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	reads, err := client.PinAgentReads(ctx, network.RegistryAddress(), "sage-deployment-check")
	if err != nil {
		t.Fatal("pin", err)
	}
	limits := nodeTestLimits()
	limits.Replay.MaxTransactions = 1024
	limits.Replay.MaxFrames = 256
	limits.Replay.MaxLogs = 4096
	limits.Replay.MaxEvidenceBytes = 8 << 20
	got, err := reads.ObserveDeployment(ctx, limits)
	if err != nil {
		t.Fatal("deployment evidence", err)
	}
	if len(got.CodeHashes) != 13 || got.Deployment.Number != network.deploymentBlock {
		t.Fatalf("observed %d runtime identities at block %d", len(got.CodeHashes), got.Deployment.Number)
	}
	if _, err := reads.ReadReplayBlock(ctx, network.deploymentBlock, limits); err != nil {
		t.Fatal("replay block", err)
	}
	t.Logf("observed %s at block %d from %s with %d exchanges", network.RegistryID(), got.Deployment.Number,
		got.ClientVersion, len(got.Exchanges))
}
