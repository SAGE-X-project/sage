package registry

import (
	"context"
	"testing"
	"time"

	"github.com/sage-x-project/sage/did"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockClient is a mock implementation of the Client interface
type MockClient struct {
	mock.Mock
}

func (m *MockClient) IsAgentRegistered(ctx context.Context, agentDID string) (bool, error) {
	args := m.Called(ctx, agentDID)
	return args.Bool(0), args.Error(1)
}

func (m *MockClient) GetRegistrationStatus(ctx context.Context, agentDID string) (*did.RegistrationStatus, error) {
	args := m.Called(ctx, agentDID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*did.RegistrationStatus), args.Error(1)
}

func (m *MockClient) RegisterAgent(ctx context.Context, req *did.RegistrationRequest) (*did.RegistrationResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*did.RegistrationResult), args.Error(1)
}

func (m *MockClient) UpdateAgent(ctx context.Context, agentDID string, req *UpdateRequest) error {
	args := m.Called(ctx, agentDID, req)
	return args.Error(0)
}

func (m *MockClient) DeactivateAgent(ctx context.Context, agentDID string) error {
	args := m.Called(ctx, agentDID)
	return args.Error(0)
}

func (m *MockClient) GetAgentByDID(ctx context.Context, agentDID string) (*did.AgentMetadata, error) {
	args := m.Called(ctx, agentDID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*did.AgentMetadata), args.Error(1)
}

func (m *MockClient) GetAgentsByOwner(ctx context.Context, owner string) ([]string, error) {
	args := m.Called(ctx, owner)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func TestClientInterface(t *testing.T) {
	ctx := context.Background()
	mockClient := new(MockClient)

	t.Run("IsAgentRegistered", func(t *testing.T) {
		// Test registered agent
		mockClient.On("IsAgentRegistered", ctx, "did:sage:eth:001").Return(true, nil).Once()
		registered, err := mockClient.IsAgentRegistered(ctx, "did:sage:eth:001")
		assert.NoError(t, err)
		assert.True(t, registered)

		// Test unregistered agent
		mockClient.On("IsAgentRegistered", ctx, "did:sage:eth:002").Return(false, nil).Once()
		registered, err = mockClient.IsAgentRegistered(ctx, "did:sage:eth:002")
		assert.NoError(t, err)
		assert.False(t, registered)

		mockClient.AssertExpectations(t)
	})

	t.Run("GetRegistrationStatus", func(t *testing.T) {
		// Test registered and active agent
		status := &did.RegistrationStatus{
			IsRegistered: true,
			IsActive:     true,
			RegisteredAt: time.Now(),
			AgentID:      "agent123",
		}
		mockClient.On("GetRegistrationStatus", ctx, "did:sage:eth:003").Return(status, nil).Once()
		
		result, err := mockClient.GetRegistrationStatus(ctx, "did:sage:eth:003")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.IsRegistered)
		assert.True(t, result.IsActive)
		assert.Equal(t, "agent123", result.AgentID)

		// Test unregistered agent
		unregisteredStatus := &did.RegistrationStatus{
			IsRegistered: false,
			IsActive:     false,
		}
		mockClient.On("GetRegistrationStatus", ctx, "did:sage:eth:004").Return(unregisteredStatus, nil).Once()
		
		result, err = mockClient.GetRegistrationStatus(ctx, "did:sage:eth:004")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsRegistered)
		assert.False(t, result.IsActive)

		mockClient.AssertExpectations(t)
	})

	t.Run("RegisterAgent", func(t *testing.T) {
		req := &did.RegistrationRequest{
			DID:         "did:sage:eth:005",
			Name:        "Test Agent",
			Description: "Test Description",
			Endpoint:    "https://test.example.com",
			Capabilities: map[string]interface{}{
				"models": []string{"gpt-4"},
			},
		}

		expectedResult := &did.RegistrationResult{
			TransactionHash: "0xabc123",
			BlockNumber:     12345,
			Timestamp:       time.Now(),
			GasUsed:         21000,
		}

		mockClient.On("RegisterAgent", ctx, req).Return(expectedResult, nil).Once()

		result, err := mockClient.RegisterAgent(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "0xabc123", result.TransactionHash)
		assert.Equal(t, uint64(12345), result.BlockNumber)

		mockClient.AssertExpectations(t)
	})

	t.Run("UpdateAgent", func(t *testing.T) {
		updateReq := &UpdateRequest{
			Name:        "Updated Name",
			Description: "Updated Description",
			Endpoint:    "https://updated.example.com",
			Capabilities: map[string]interface{}{
				"models": []string{"gpt-4", "claude"},
			},
		}

		mockClient.On("UpdateAgent", ctx, "did:sage:eth:006", updateReq).Return(nil).Once()

		err := mockClient.UpdateAgent(ctx, "did:sage:eth:006", updateReq)
		assert.NoError(t, err)

		mockClient.AssertExpectations(t)
	})

	t.Run("DeactivateAgent", func(t *testing.T) {
		mockClient.On("DeactivateAgent", ctx, "did:sage:eth:007").Return(nil).Once()

		err := mockClient.DeactivateAgent(ctx, "did:sage:eth:007")
		assert.NoError(t, err)

		mockClient.AssertExpectations(t)
	})

	t.Run("GetAgentByDID", func(t *testing.T) {
		expectedMetadata := &did.AgentMetadata{
			DID:         "did:sage:eth:008",
			Name:        "Test Agent",
			Description: "Test Description",
			Endpoint:    "https://test.example.com",
			PublicKey:   []byte{1, 2, 3, 4},
			Capabilities: map[string]interface{}{
				"models": []string{"gpt-4"},
			},
			Owner:     "0x123",
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockClient.On("GetAgentByDID", ctx, "did:sage:eth:008").Return(expectedMetadata, nil).Once()

		metadata, err := mockClient.GetAgentByDID(ctx, "did:sage:eth:008")
		assert.NoError(t, err)
		assert.NotNil(t, metadata)
		assert.Equal(t, did.AgentDID("did:sage:eth:008"), metadata.DID)
		assert.Equal(t, "Test Agent", metadata.Name)
		assert.True(t, metadata.IsActive)

		mockClient.AssertExpectations(t)
	})

	t.Run("GetAgentsByOwner", func(t *testing.T) {
		expectedDIDs := []string{
			"did:sage:eth:009",
			"did:sage:eth:010",
			"did:sage:eth:011",
		}

		mockClient.On("GetAgentsByOwner", ctx, "0x456").Return(expectedDIDs, nil).Once()

		dids, err := mockClient.GetAgentsByOwner(ctx, "0x456")
		assert.NoError(t, err)
		assert.Len(t, dids, 3)
		assert.Equal(t, expectedDIDs, dids)

		mockClient.AssertExpectations(t)
	})
}

func TestClientConfig(t *testing.T) {
	config := &ClientConfig{
		RPC:      "https://eth-mainnet.example.com",
		Contract: "0x1234567890123456789012345678901234567890",
		ChainID:  1,
	}

	assert.Equal(t, "https://eth-mainnet.example.com", config.RPC)
	assert.Equal(t, "0x1234567890123456789012345678901234567890", config.Contract)
	assert.Equal(t, uint64(1), config.ChainID)
}

func TestUpdateRequest(t *testing.T) {
	req := &UpdateRequest{
		Name:        "Updated Agent",
		Description: "Updated Description",
		Endpoint:    "https://updated.example.com",
		Capabilities: map[string]interface{}{
			"models":    []string{"gpt-4", "claude-3"},
			"languages": []string{"en", "es", "fr"},
			"skills":    []string{"coding", "analysis"},
		},
	}

	assert.Equal(t, "Updated Agent", req.Name)
	assert.Equal(t, "Updated Description", req.Description)
	assert.Equal(t, "https://updated.example.com", req.Endpoint)
	assert.NotNil(t, req.Capabilities)
	assert.Len(t, req.Capabilities["models"].([]string), 2)
}