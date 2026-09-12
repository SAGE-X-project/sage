package ethereum

import (
	chaineth "github.com/sage-x-project/sage/pkg/agent/crypto/chain/ethereum"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// EthereumClient is the previous Ethereum DID client. It is now a thin
// wrapper over AgentCardClient, so reads and writes both target the
// AgentCardRegistry contract; the earlier implementation packed
// SageRegistryV2 function signatures and could not write at all.
//
// Deprecated: use AgentCardClient (NewAgentCardClient). The type stays for
// callers that name it; every method is promoted from AgentCardClient.
type EthereumClient struct {
	*AgentCardClient
}

// Register installs the Ethereum DID client into did.Manager.Configure and
// registers the Ethereum chain provider. Call it from the composition root
// (internal/app.RegisterDefaults does) before configuring did.ChainEthereum.
func Register() {
	chaineth.Register()
	did.RegisterClientCreator(did.ChainEthereum, func(config *did.RegistryConfig) (did.ChainClient, error) {
		return NewAgentCardClient(config)
	})
}

// NewEthereumClient creates an AgentCardClient wrapped in the previous type.
//
// Deprecated: use NewAgentCardClient.
func NewEthereumClient(config *did.RegistryConfig) (*EthereumClient, error) {
	client, err := NewAgentCardClient(config)
	if err != nil {
		return nil, err
	}
	return &EthereumClient{AgentCardClient: client}, nil
}
