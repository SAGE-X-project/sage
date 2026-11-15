package registry011

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"testing"
	"time"
)

func testRootPEM(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "sage dev root"},
		NotBefore: time.Unix(0, 0), NotAfter: time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC), IsCA: true, BasicConstraintsValid: true,
		KeyUsage: x509.KeyUsageCertSign}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// testDeploymentRecord mirrors the record written by sage-contracts
// scripts/deploy-chain-registry.mjs for a fresh Anvil chain.
func testDeploymentRecord(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	names := []string{"RegistryRenderer", "RegistryEd25519", "RegistryRules", "RegistryCallValidator",
		"RegistryStoreFactory", "RegistryClaimProcessorFactory", "RegistryKeyServiceProcessorFactory",
		"RegistryLifecycleProcessorFactory", "ChainRegistry", "RegistryStore", "RegistryClaimProcessor",
		"RegistryKeyServiceProcessor", "RegistryLifecycleProcessor"}
	contracts := make([]any, 0, len(names))
	for i, name := range names {
		contracts = append(contracts, map[string]any{"name": name, "address": fmt.Sprintf("0x%040x", i+1),
			"code_sha256": strings.Repeat("ab", 32)})
	}
	registry := "0x2279b7a0a67db372996a5fab50d91eaa73d2ebe6"
	record := map[string]any{
		"chain_id": 31337, "registry_address": registry, "registry_id": "eip155:31337:" + registry,
		"deployer": "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266", "contracts": contracts,
		"transactions": []any{map[string]any{"name": "RegistryRenderer", "nonce": 0, "hash": "0x" + strings.Repeat("11", 32),
			"contract_address": "0x5fbdb2315678afecb367f032d93f642f64180aa3", "block_number": 1, "block_hash": "0x" + strings.Repeat("21", 32)},
			map[string]any{"name": "ChainRegistry", "nonce": 8, "hash": "0x" + strings.Repeat("19", 32),
				"contract_address": registry, "block_number": 9, "block_hash": "0x" + strings.Repeat("29", 32)}},
	}
	if mutate != nil {
		mutate(record)
	}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func testDevConfig(t *testing.T) DevNetworkConfig {
	return DevNetworkConfig{Endpoint: "https://127.0.0.1:8546", RootsPEM: testRootPEM(t),
		GenesisHash: "0x" + strings.Repeat("0c", 32), DeploymentRecord: testDeploymentRecord(t, nil)}
}

func TestSepoliaNetworkKeepsPinnedValues(t *testing.T) {
	n := SepoliaNetwork()
	if !n.valid() || !n.Qualifying() || n.Name() != "sepolia" {
		t.Fatal("Sepolia profile must be valid and qualifying")
	}
	if n.ChainID() != 11155111 || n.chainIDHex() != "0xaa36a7" || n.genesisHash != sepoliaGenesis ||
		n.RegistryAddress() != SepoliaRegistryAddress || n.RegistryID() != "eip155:11155111:"+SepoliaRegistryAddress ||
		n.endpoint != SepoliaRPCURL || n.deploymentBlock != sepoliaDeploymentBlock || n.amsterdamTime != sepoliaAmsterdamTime {
		t.Fatal("Sepolia profile changed a pinned value")
	}
	if n.roots != nil || n.serverName != "" {
		t.Fatal("Sepolia must keep the system roots and URL host")
	}
	if string(n.identity) != string(nodeIdentityJSON) {
		t.Fatal("Sepolia identity must be the embedded node identity")
	}
	if config := n.tlsConfig(); config.RootCAs != nil || config.ServerName != "" || config.MinVersion == 0 {
		t.Fatal("Sepolia TLS settings changed")
	}
}

func TestDevNetworkBuildsNonQualifyingProfile(t *testing.T) {
	n, err := NewDevNetwork(testDevConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	if !n.valid() || n.Qualifying() || n.Name() != "dev" || n.ChainID() != 31337 || n.chainIDHex() != "0x7a69" {
		t.Fatal("unexpected dev profile identity")
	}
	if n.RegistryAddress() != "0x2279b7a0a67db372996a5fab50d91eaa73d2ebe6" ||
		n.RegistryID() != "eip155:31337:0x2279b7a0a67db372996a5fab50d91eaa73d2ebe6" || n.deploymentBlock != 9 {
		t.Fatal("dev profile did not take the deployment record values")
	}
	if n.amsterdamTime != math.MaxUint64 || n.serverName != "127.0.0.1" || n.roots == nil {
		t.Fatal("dev profile must use pre-Amsterdam rules and its own TLS root")
	}
	identity, _, err := nodeIdentityManifestFor(n, nodeABIJSON)
	if err != nil || identity.Registry != n.RegistryAddress() || identity.Deployment.BlockNumber != 9 ||
		identity.Deployment.Hash != "0x"+strings.Repeat("19", 32) || len(identity.Contracts) != 13 {
		t.Fatalf("dev identity manifest rejected: %v", err)
	}
}

func TestDevNetworkRefusesUnsafeConfiguration(t *testing.T) {
	cases := map[string]func(*DevNetworkConfig){
		"remote host":     func(c *DevNetworkConfig) { c.Endpoint = "https://rpc.example.org:8546" },
		"private address": func(c *DevNetworkConfig) { c.Endpoint = "https://10.0.0.1:8546" },
		"unspecified":     func(c *DevNetworkConfig) { c.Endpoint = "https://0.0.0.0:8546" },
		"host name":       func(c *DevNetworkConfig) { c.Endpoint = "https://localhost:8546" },
		"no port":         func(c *DevNetworkConfig) { c.Endpoint = "https://127.0.0.1" },
		"query":           func(c *DevNetworkConfig) { c.Endpoint = "https://127.0.0.1:8546?x=1" },
		"fragment":        func(c *DevNetworkConfig) { c.Endpoint = "https://127.0.0.1:8546#x" },
		"zoned ipv6":      func(c *DevNetworkConfig) { c.Endpoint = "https://[fe80::1%25lo0]:8546" },
		"sepolia registry": func(c *DevNetworkConfig) {
			c.DeploymentRecord = testDeploymentRecord(t, func(r map[string]any) {
				r["registry_address"] = SepoliaRegistryAddress
				r["registry_id"] = "eip155:31337:" + SepoliaRegistryAddress
				r["transactions"].([]any)[1].(map[string]any)["contract_address"] = SepoliaRegistryAddress
			})
		},
		"plain http":    func(c *DevNetworkConfig) { c.Endpoint = "http://127.0.0.1:8546" },
		"path":          func(c *DevNetworkConfig) { c.Endpoint = "https://127.0.0.1:8546/rpc" },
		"user info":     func(c *DevNetworkConfig) { c.Endpoint = "https://u@127.0.0.1:8546" },
		"no roots":      func(c *DevNetworkConfig) { c.RootsPEM = nil },
		"bad roots":     func(c *DevNetworkConfig) { c.RootsPEM = []byte("not a certificate") },
		"short genesis": func(c *DevNetworkConfig) { c.GenesisHash = "0x01" },
		"upper genesis": func(c *DevNetworkConfig) { c.GenesisHash = "0x" + strings.Repeat("0C", 32) },
		"other chain": func(c *DevNetworkConfig) {
			c.DeploymentRecord = testDeploymentRecord(t, func(r map[string]any) { r["chain_id"] = 11155111 })
		},
		"wrong registry id": func(c *DevNetworkConfig) {
			c.DeploymentRecord = testDeploymentRecord(t, func(r map[string]any) { r["registry_id"] = "eip155:1:x" })
		},
		"missing contract": func(c *DevNetworkConfig) {
			c.DeploymentRecord = testDeploymentRecord(t, func(r map[string]any) { r["contracts"] = r["contracts"].([]any)[:12] })
		},
		"no creation": func(c *DevNetworkConfig) {
			c.DeploymentRecord = testDeploymentRecord(t, func(r map[string]any) { r["transactions"] = r["transactions"].([]any)[:1] })
		},
		"malformed record": func(c *DevNetworkConfig) { c.DeploymentRecord = []byte("{") },
		"mixed case address": func(c *DevNetworkConfig) {
			c.DeploymentRecord = testDeploymentRecord(t, func(r map[string]any) { r["registry_address"] = "0x2279B7A0a67db372996a5fab50d91eaa73d2ebe6" })
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			config := testDevConfig(t)
			mutate(&config)
			if _, err := NewDevNetwork(config); !errors.Is(err, ErrConfiguration) {
				t.Fatalf("expected ErrConfiguration, got %v", err)
			}
		})
	}
}

func TestDevNetworkAcceptsIPv6Loopback(t *testing.T) {
	config := testDevConfig(t)
	config.Endpoint = "https://[::1]:8546"
	n, err := NewDevNetwork(config)
	if err != nil || n.serverName != "::1" {
		t.Fatalf("IPv6 loopback refused: %v", err)
	}
}

func TestDevNetworkCannotCollectReplayBatch(t *testing.T) {
	n, err := NewDevNetwork(testDevConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewReadRPC(n, RPCTransportLimits{RequestTimeout: time.Second, MaxBytes: 1024, MaxDepth: 8})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, err := c.CollectReplayBatch(context.Background(), JournalConfig{}, JournalCheckpoint{}, SourceReadLimits{}, func() ClockSample { return ClockSample{} }); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("dev batch must refuse before any read, got %v", err)
	}
}

func TestDevNetworkDetachesCallerInput(t *testing.T) {
	config := testDevConfig(t)
	n, err := NewDevNetwork(config)
	if err != nil {
		t.Fatal(err)
	}
	before := string(n.identity)
	for i := range config.DeploymentRecord {
		config.DeploymentRecord[i] = 0
	}
	if string(n.identity) != before {
		t.Fatal("profile must not alias the caller's record")
	}
}

func TestZeroNetworkProfileIsInvalid(t *testing.T) {
	var n NetworkProfile
	if n.valid() || n.Qualifying() {
		t.Fatal("zero profile must refuse")
	}
	if _, err := NewReadRPC(n, RPCTransportLimits{RequestTimeout: time.Second, MaxBytes: 1024, MaxDepth: 8}); !errors.Is(err, ErrConfiguration) {
		t.Fatal("ReadRPC must refuse a zero profile")
	}
}
