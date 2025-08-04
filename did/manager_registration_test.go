package did

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDIDRegistry for testing registration status
type MockDIDRegistry struct {
	mock.Mock
}

func (m *MockDIDRegistry) Register(ctx context.Context, req *RegistrationRequest) (*RegistrationResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RegistrationResult), args.Error(1)
}

func (m *MockDIDRegistry) Resolve(ctx context.Context, did AgentDID) (*AgentMetadata, error) {
	args := m.Called(ctx, did)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*AgentMetadata), args.Error(1)
}

func (m *MockDIDRegistry) Update(ctx context.Context, did AgentDID, updates map[string]interface{}, keyPair interface{}) error {
	args := m.Called(ctx, did, updates, keyPair)
	return args.Error(0)
}

func (m *MockDIDRegistry) Deactivate(ctx context.Context, did AgentDID, keyPair interface{}) error {
	args := m.Called(ctx, did, keyPair)
	return args.Error(0)
}

// Additional methods to implement Resolver interface
func (m *MockDIDRegistry) ResolvePublicKey(ctx context.Context, did AgentDID) (interface{}, error) {
	args := m.Called(ctx, did)
	return args.Get(0), args.Error(1)
}

func (m *MockDIDRegistry) VerifyMetadata(ctx context.Context, did AgentDID, metadata *AgentMetadata) (*VerificationResult, error) {
	args := m.Called(ctx, did, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*VerificationResult), args.Error(1)
}

func (m *MockDIDRegistry) ListAgentsByOwner(ctx context.Context, ownerAddress string) ([]*AgentMetadata, error) {
	args := m.Called(ctx, ownerAddress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*AgentMetadata), args.Error(1)
}

func (m *MockDIDRegistry) Search(ctx context.Context, criteria SearchCriteria) ([]*AgentMetadata, error) {
	args := m.Called(ctx, criteria)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*AgentMetadata), args.Error(1)
}

func (m *MockDIDRegistry) GetRegistrationStatus(ctx context.Context, txHash string) (*RegistrationResult, error) {
	args := m.Called(ctx, txHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RegistrationResult), args.Error(1)
}

func TestManager_IsAgentRegistered(t *testing.T) {
	ctx := context.Background()
	manager := NewManager()
	mockRegistry := new(MockDIDRegistry)

	// Configure manager with mock resolver (since IsAgentRegistered uses resolver)  
	manager.resolver.resolvers[ChainEthereum] = mockRegistry

	t.Run("agent is registered", func(t *testing.T) {
		did := AgentDID("did:sage:ethereum:agent001")
		metadata := &AgentMetadata{
			DID:      did,
			Name:     "Test Agent",
			IsActive: true,
		}

		mockRegistry.On("Resolve", ctx, did).Return(metadata, nil).Once()

		registered, err := manager.IsAgentRegistered(ctx, did)
		assert.NoError(t, err)
		assert.True(t, registered)

		mockRegistry.AssertExpectations(t)
	})

	t.Run("agent not registered", func(t *testing.T) {
		did := AgentDID("did:sage:ethereum:agent002")

		mockRegistry.On("Resolve", ctx, did).Return(nil, ErrDIDNotFound).Once()

		registered, err := manager.IsAgentRegistered(ctx, did)
		assert.NoError(t, err)
		assert.False(t, registered)

		mockRegistry.AssertExpectations(t)
	})

	t.Run("invalid DID format", func(t *testing.T) {
		did := AgentDID("invalid-did")

		registered, err := manager.IsAgentRegistered(ctx, did)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid DID format")
		assert.False(t, registered)
	})

	t.Run("chain not configured", func(t *testing.T) {
		did := AgentDID("did:sage:solana:agent003")

		registered, err := manager.IsAgentRegistered(ctx, did)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "chain solana not configured")
		assert.False(t, registered)
	})

	t.Run("resolve error", func(t *testing.T) {
		did := AgentDID("did:sage:ethereum:agent004")

		mockRegistry.On("Resolve", ctx, did).Return(nil, assert.AnError).Once()

		registered, err := manager.IsAgentRegistered(ctx, did)
		assert.Error(t, err)
		assert.False(t, registered)

		mockRegistry.AssertExpectations(t)
	})
}

func TestManager_GetRegistrationStatus(t *testing.T) {
	ctx := context.Background()
	manager := NewManager()
	mockRegistry := new(MockDIDRegistry)

	// Configure manager with mock resolver (since IsAgentRegistered uses resolver)  
	manager.resolver.resolvers[ChainEthereum] = mockRegistry

	t.Run("registered and active agent", func(t *testing.T) {
		did := AgentDID("did:sage:ethereum:agent005")
		now := time.Now()
		metadata := &AgentMetadata{
			DID:       did,
			Name:      "Active Agent",
			IsActive:  true,
			CreatedAt: now,
		}

		mockRegistry.On("Resolve", ctx, did).Return(metadata, nil).Once()

		status, err := manager.GetRegistrationStatus(ctx, did)
		require.NoError(t, err)
		require.NotNil(t, status)
		assert.True(t, status.IsRegistered)
		assert.True(t, status.IsActive)
		assert.Equal(t, now, status.RegisteredAt)
		assert.Equal(t, string(did), status.AgentID)

		mockRegistry.AssertExpectations(t)
	})

	t.Run("registered but inactive agent", func(t *testing.T) {
		did := AgentDID("did:sage:ethereum:agent006")
		now := time.Now()
		metadata := &AgentMetadata{
			DID:       did,
			Name:      "Inactive Agent",
			IsActive:  false,
			CreatedAt: now,
		}

		mockRegistry.On("Resolve", ctx, did).Return(metadata, nil).Once()

		status, err := manager.GetRegistrationStatus(ctx, did)
		require.NoError(t, err)
		require.NotNil(t, status)
		assert.True(t, status.IsRegistered)
		assert.False(t, status.IsActive)
		assert.Equal(t, now, status.RegisteredAt)

		mockRegistry.AssertExpectations(t)
	})

	t.Run("unregistered agent", func(t *testing.T) {
		did := AgentDID("did:sage:ethereum:agent007")

		mockRegistry.On("Resolve", ctx, did).Return(nil, ErrDIDNotFound).Once()

		status, err := manager.GetRegistrationStatus(ctx, did)
		require.NoError(t, err)
		require.NotNil(t, status)
		assert.False(t, status.IsRegistered)
		assert.False(t, status.IsActive)
		assert.Zero(t, status.RegisteredAt)
		assert.Empty(t, status.AgentID)

		mockRegistry.AssertExpectations(t)
	})

	t.Run("invalid DID format", func(t *testing.T) {
		did := AgentDID("not-a-valid-did")

		status, err := manager.GetRegistrationStatus(ctx, did)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid DID format")
		assert.Nil(t, status)
	})

	t.Run("chain not configured", func(t *testing.T) {
		did := AgentDID("did:sage:polygon:agent008")

		status, err := manager.GetRegistrationStatus(ctx, did)
		assert.Error(t, err)
		// ParseDID fails first with unknown chain error
		assert.Contains(t, err.Error(), "unknown chain")
		assert.Nil(t, status)
	})

	t.Run("resolve error", func(t *testing.T) {
		did := AgentDID("did:sage:ethereum:agent009")

		mockRegistry.On("Resolve", ctx, did).Return(nil, assert.AnError).Once()

		status, err := manager.GetRegistrationStatus(ctx, did)
		assert.Error(t, err)
		assert.Nil(t, status)

		mockRegistry.AssertExpectations(t)
	})
}

func TestManager_ConcurrentRegistrationChecks(t *testing.T) {
	ctx := context.Background()
	manager := NewManager()
	mockRegistry := new(MockDIDRegistry)

	// Configure manager with mock resolver (since IsAgentRegistered uses resolver)  
	manager.resolver.resolvers[ChainEthereum] = mockRegistry

	// Set up expectations for concurrent calls
	dids := []AgentDID{
		"did:sage:ethereum:concurrent001",
		"did:sage:ethereum:concurrent002",
		"did:sage:ethereum:concurrent003",
	}

	for i, did := range dids {
		metadata := &AgentMetadata{
			DID:      did,
			Name:     "Concurrent Agent",
			IsActive: i%2 == 0, // Alternate active/inactive
		}
		mockRegistry.On("Resolve", ctx, did).Return(metadata, nil).Maybe()
	}

	// Run concurrent checks
	results := make(chan bool, len(dids))
	errors := make(chan error, len(dids))

	for _, did := range dids {
		go func(d AgentDID) {
			registered, err := manager.IsAgentRegistered(ctx, d)
			if err != nil {
				errors <- err
			} else {
				results <- registered
			}
		}(did)
	}

	// Collect results
	for i := 0; i < len(dids); i++ {
		select {
		case registered := <-results:
			assert.True(t, registered)
		case err := <-errors:
			t.Errorf("Unexpected error: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for results")
		}
	}

	mockRegistry.AssertExpectations(t)
}

func TestRegistrationStatusEdgeCases(t *testing.T) {
	ctx := context.Background()
	manager := NewManager()

	t.Run("empty DID", func(t *testing.T) {
		registered, err := manager.IsAgentRegistered(ctx, "")
		assert.Error(t, err)
		assert.False(t, registered)

		status, err := manager.GetRegistrationStatus(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, status)
	})

	t.Run("malformed DID", func(t *testing.T) {
		malformedDIDs := []AgentDID{
			"did:",
			"did:sage",
			"did:sage:",
			"did:sage::",
			"did:sage:ethereum:",
			":sage:ethereum:agent",
		}

		for _, did := range malformedDIDs {
			registered, err := manager.IsAgentRegistered(ctx, did)
			assert.Error(t, err, "DID: %s", did)
			assert.False(t, registered)

			status, err := manager.GetRegistrationStatus(ctx, did)
			assert.Error(t, err, "DID: %s", did)
			assert.Nil(t, status)
		}
	})
}