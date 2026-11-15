package registry011

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"math"
	"net"
	"net/url"
	"regexp"
	"strconv"
)

// devChainID is the local Anvil/Hardhat chain ID. A development profile never
// reads any other chain.
const devChainID uint64 = 31337

const sepoliaChainID uint64 = 11155111

var (
	networkHashPattern    = regexp.MustCompile(`^0x[0-9a-f]{64}$`)
	networkAddressPattern = regexp.MustCompile(`^0x[0-9a-f]{40}$`)
	networkDigestPattern  = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// NetworkProfile identifies the one Registry deployment a reader may observe:
// chain, genesis, Registry instance and runtime identities, RPC endpoint and
// TLS trust, deployment block and fork rules. A profile is immutable after
// construction. Only SepoliaNetwork is qualifying; a development profile can
// exercise the same code paths but never makes a Source ready.
type NetworkProfile struct {
	name            string
	chainID         uint64
	genesisHash     string
	registryAddress string
	endpoint        string
	serverName      string
	roots           *x509.CertPool
	deploymentBlock uint64
	identity        []byte
	amsterdamTime   uint64
	qualifying      bool
}

// SepoliaNetwork returns the approved Sepolia deployment with the values this
// package has always pinned.
func SepoliaNetwork() NetworkProfile {
	return NetworkProfile{name: "sepolia", chainID: sepoliaChainID, genesisHash: sepoliaGenesis,
		registryAddress: SepoliaRegistryAddress, endpoint: SepoliaRPCURL, deploymentBlock: sepoliaDeploymentBlock,
		identity: bytes.Clone(nodeIdentityJSON), amsterdamTime: sepoliaAmsterdamTime, qualifying: true}
}

// DevNetworkConfig describes a local development chain. Endpoint must be an
// https URL on a loopback address, served with a certificate under RootsPEM.
// DeploymentRecord is the JSON written by sage-contracts
// scripts/deploy-chain-registry.mjs for that chain.
type DevNetworkConfig struct {
	Endpoint         string
	RootsPEM         []byte
	GenesisHash      string
	DeploymentRecord []byte
}

type devDeploymentRecord struct {
	ChainID      uint64             `json:"chain_id"`
	Registry     string             `json:"registry_address"`
	RegistryID   string             `json:"registry_id"`
	Deployer     string             `json:"deployer"`
	Contracts    []nodeCodeIdentity `json:"contracts"`
	Transactions []struct {
		Name        string `json:"name"`
		Hash        string `json:"hash"`
		Address     string `json:"contract_address"`
		BlockNumber uint64 `json:"block_number"`
		BlockHash   string `json:"block_hash"`
	} `json:"transactions"`
}

// NewDevNetwork builds a non-qualifying profile for a local chain. Blocks are
// checked with pre-Amsterdam rules because local nodes do not produce the
// complete Amsterdam header that the Sepolia profile requires.
func NewDevNetwork(config DevNetworkConfig) (NetworkProfile, error) {
	host, ok := devEndpointHost(config.Endpoint)
	if !ok || !networkHashPattern.MatchString(config.GenesisHash) {
		return NetworkProfile{}, ErrConfiguration
	}
	roots := x509.NewCertPool()
	if len(config.RootsPEM) == 0 || !roots.AppendCertsFromPEM(config.RootsPEM) {
		return NetworkProfile{}, ErrConfiguration
	}
	identity, registry, block, ok := devIdentity(config.DeploymentRecord)
	if !ok {
		return NetworkProfile{}, ErrConfiguration
	}
	return NetworkProfile{name: "dev", chainID: devChainID, genesisHash: config.GenesisHash, registryAddress: registry,
		endpoint: config.Endpoint, serverName: host, roots: roots, deploymentBlock: block, identity: identity,
		amsterdamTime: math.MaxUint64}, nil
}

func devEndpointHost(endpoint string) (string, bool) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", false
	}
	host := parsed.Hostname()
	if port, err := strconv.Atoi(parsed.Port()); err != nil || port < 1 || port > 65535 {
		return "", false
	}
	// Only IP literals: a name would depend on the hosts file or DNS.
	ip := net.ParseIP(host)
	return host, ip != nil && ip.IsLoopback()
}

// devIdentity converts a deployment record into the node identity shape that
// ObserveDeployment checks, and returns the Registry address and creation block.
func devIdentity(raw []byte) ([]byte, string, uint64, bool) {
	var record devDeploymentRecord
	// A development record may never claim the approved Sepolia instance.
	if json.Unmarshal(raw, &record) != nil || record.ChainID != devChainID || !networkAddressPattern.MatchString(record.Registry) ||
		record.Registry == SepoliaRegistryAddress ||
		record.RegistryID != "eip155:"+strconv.FormatUint(devChainID, 10)+":"+record.Registry ||
		!networkAddressPattern.MatchString(record.Deployer) || len(record.Contracts) != 13 || len(record.Transactions) == 0 {
		return nil, "", 0, false
	}
	for _, contract := range record.Contracts {
		if contract.Name == "" || !networkAddressPattern.MatchString(contract.Address) || !networkDigestPattern.MatchString(contract.SHA256) {
			return nil, "", 0, false
		}
	}
	creation := record.Transactions[len(record.Transactions)-1]
	if creation.Name != "ChainRegistry" || creation.Address != record.Registry || !networkHashPattern.MatchString(creation.Hash) ||
		!networkHashPattern.MatchString(creation.BlockHash) || creation.BlockNumber == 0 {
		return nil, "", 0, false
	}
	identity := map[string]any{"registry_address": record.Registry, "registry_id": record.RegistryID, "deployer": record.Deployer,
		"contracts": record.Contracts, "deployment": map[string]any{"name": creation.Name, "hash": creation.Hash,
			"contract_address": creation.Address, "block_number": creation.BlockNumber, "block_hash": creation.BlockHash}}
	encoded, err := json.Marshal(identity)
	if err != nil {
		return nil, "", 0, false
	}
	return encoded, record.Registry, creation.BlockNumber, true
}

func (n NetworkProfile) valid() bool {
	return n.name != "" && n.chainID != 0 && networkHashPattern.MatchString(n.genesisHash) &&
		networkAddressPattern.MatchString(n.registryAddress) && n.endpoint != "" && len(n.identity) != 0 && n.deploymentBlock != 0
}

// Name is "sepolia" or "dev".
func (n NetworkProfile) Name() string { return n.name }

// ChainID is the EIP-155 chain ID the profile reads.
func (n NetworkProfile) ChainID() uint64 { return n.chainID }

// RegistryAddress is the lowercase ChainRegistry address.
func (n NetworkProfile) RegistryAddress() string { return n.registryAddress }

// RegistryID is the value ChainRegistry.registryId() must return.
func (n NetworkProfile) RegistryID() string {
	return "eip155:" + strconv.FormatUint(n.chainID, 10) + ":" + n.registryAddress
}

// Qualifying reports whether results from this profile may count toward Source
// readiness. Only the Sepolia profile qualifies.
func (n NetworkProfile) Qualifying() bool { return n.qualifying && n.valid() }

func (n NetworkProfile) chainIDHex() string { return "0x" + strconv.FormatUint(n.chainID, 16) }

// tlsConfig keeps the historical Sepolia settings exactly: system roots and
// the URL host as server name.
func (n NetworkProfile) tlsConfig() *tls.Config {
	config := &tls.Config{MinVersion: tls.VersionTLS12}
	if n.roots != nil {
		config.RootCAs = n.roots.Clone()
		config.ServerName = n.serverName
	}
	return config
}
