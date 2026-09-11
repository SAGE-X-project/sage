package did

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
)

// fakeChainClient satisfies Registry and Resolver with canned answers.
type fakeChainClient struct{ meta *AgentMetadata }

func (f *fakeChainClient) Register(context.Context, *RegistrationRequest) (*RegistrationResult, error) {
	return &RegistrationResult{TransactionHash: "0xfake"}, nil
}
func (f *fakeChainClient) Update(context.Context, AgentDID, map[string]interface{}, sagecrypto.KeyPair) error {
	return nil
}
func (f *fakeChainClient) Deactivate(context.Context, AgentDID, sagecrypto.KeyPair) error { return nil }
func (f *fakeChainClient) GetRegistrationStatus(context.Context, string) (*RegistrationResult, error) {
	return nil, nil
}
func (f *fakeChainClient) Resolve(context.Context, AgentDID) (*AgentMetadata, error) {
	return f.meta, nil
}
func (f *fakeChainClient) ResolvePublicKey(context.Context, AgentDID) (interface{}, error) {
	return f.meta.PublicKey, nil
}
func (f *fakeChainClient) ResolveKEMKey(context.Context, AgentDID) (interface{}, error) {
	return nil, nil
}
func (f *fakeChainClient) VerifyMetadata(context.Context, AgentDID, *AgentMetadata) (*VerificationResult, error) {
	return &VerificationResult{Valid: true}, nil
}
func (f *fakeChainClient) ListAgentsByOwner(context.Context, string) ([]*AgentMetadata, error) {
	return []*AgentMetadata{f.meta}, nil
}
func (f *fakeChainClient) Search(context.Context, SearchCriteria) ([]*AgentMetadata, error) {
	return []*AgentMetadata{f.meta}, nil
}

// Configure must install the chain client registered by the chain package so
// that resolution works without a manual SetClient call.
func TestManagerConfigure_InstallsRegisteredClient(t *testing.T) {
	prev := GetEthereumV4ClientCreator()
	t.Cleanup(func() { RegisterEthereumV4ClientCreator(prev) })

	meta := &AgentMetadata{DID: "did:sage:ethereum:0xabc", Name: "agent", IsActive: true}
	RegisterEthereumV4ClientCreator(func(cfg *RegistryConfig) (interface{}, error) {
		return &fakeChainClient{meta: meta}, nil
	})

	m := NewManager()
	require.NoError(t, m.Configure(ChainEthereum, &RegistryConfig{
		Chain: ChainEthereum, ContractAddress: "0x1", RPCEndpoint: "http://localhost:8545",
	}))
	require.True(t, m.HasClient(ChainEthereum))

	got, err := m.ResolveAgent(context.Background(), meta.DID)
	require.NoError(t, err)
	require.Equal(t, meta.Name, got.Name)
}

func TestManagerConfigure_WithoutCreatorLeavesChainUnwired(t *testing.T) {
	prev := GetEthereumV4ClientCreator()
	t.Cleanup(func() { RegisterEthereumV4ClientCreator(prev) })
	RegisterEthereumV4ClientCreator(nil)

	m := NewManager()
	require.NoError(t, m.Configure(ChainEthereum, &RegistryConfig{
		Chain: ChainEthereum, ContractAddress: "0x1", RPCEndpoint: "http://localhost:8545",
	}))
	require.False(t, m.HasClient(ChainEthereum))

	_, err := m.ResolveAgent(context.Background(), "did:sage:ethereum:0xabc")
	require.Error(t, err)
	require.Contains(t, err.Error(), "pkg/agent/did/ethereum")
}
