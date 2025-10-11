// SAGE - Secure Agent Guarantee Engine
// Copyright (C) 2025 SAGE-X-project
//
// This file is part of SAGE.
//
// SAGE is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// SAGE is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with SAGE. If not, see <https://www.gnu.org/licenses/>.

package transport

import (
	"context"
	"fmt"
	"testing"
)

// MockTransport for testing
type mockTransport struct {
	transportType string
	endpoint      string
}

func (m *mockTransport) Send(ctx context.Context, msg *SecureMessage) (*Response, error) {
	return &Response{Success: true}, nil
}

func TestTransportSelector_SelectByURL(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		expectedType  TransportType
		shouldSucceed bool
	}{
		{
			name:          "HTTP URL",
			url:           "http://agent.example.com",
			expectedType:  TransportHTTP,
			shouldSucceed: true,
		},
		{
			name:          "HTTPS URL",
			url:           "https://agent.example.com",
			expectedType:  TransportHTTPS,
			shouldSucceed: true,
		},
		{
			name:          "gRPC URL",
			url:           "grpc://agent.example.com:50051",
			expectedType:  TransportGRPC,
			shouldSucceed: false, // Not registered by default
		},
		{
			name:          "WebSocket URL",
			url:           "ws://agent.example.com/ws",
			expectedType:  TransportWebSocket,
			shouldSucceed: false, // Not registered by default
		},
		{
			name:          "WebSocket Secure URL",
			url:           "wss://agent.example.com/ws",
			expectedType:  TransportWebSocketSecure,
			shouldSucceed: false, // Not registered by default
		},
		{
			name:          "Invalid URL scheme",
			url:           "ftp://agent.example.com",
			expectedType:  "",
			shouldSucceed: false,
		},
		{
			name:          "Malformed URL",
			url:           "not a url",
			expectedType:  "",
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := NewTransportSelector()

			// Register mock factories for testing
			selector.RegisterFactory(TransportHTTP, func(endpoint string) (MessageTransport, error) {
				return &mockTransport{transportType: "http", endpoint: endpoint}, nil
			})
			selector.RegisterFactory(TransportHTTPS, func(endpoint string) (MessageTransport, error) {
				return &mockTransport{transportType: "https", endpoint: endpoint}, nil
			})

			transport, err := selector.SelectByURL(tt.url)

			if tt.shouldSucceed {
				if err != nil {
					t.Errorf("Expected success, got error: %v", err)
				}
				if transport == nil {
					t.Errorf("Expected transport, got nil")
				}
				if mock, ok := transport.(*mockTransport); ok {
					if mock.endpoint != tt.url {
						t.Errorf("Expected endpoint %q, got %q", tt.url, mock.endpoint)
					}
				}
			} else {
				if err == nil {
					t.Errorf("Expected error, got success")
				}
			}
		})
	}
}

func TestTransportSelector_RegisterFactory(t *testing.T) {
	selector := NewTransportSelector()

	// Register custom transport
	selector.RegisterFactory("custom", func(endpoint string) (MessageTransport, error) {
		return &mockTransport{transportType: "custom", endpoint: endpoint}, nil
	})

	// Check if registered
	if !selector.IsRegistered("custom") {
		t.Errorf("Expected 'custom' to be registered")
	}

	// Try to create transport
	transport, err := selector.Select("custom", "custom://test")
	if err != nil {
		t.Fatalf("Failed to create custom transport: %v", err)
	}

	if mock, ok := transport.(*mockTransport); ok {
		if mock.transportType != "custom" {
			t.Errorf("Expected type 'custom', got '%s'", mock.transportType)
		}
	} else {
		t.Errorf("Expected mockTransport, got %T", transport)
	}
}

func TestTransportSelector_IsRegistered(t *testing.T) {
	selector := NewTransportSelector()

	// Register HTTP
	selector.RegisterFactory(TransportHTTP, func(endpoint string) (MessageTransport, error) {
		return &mockTransport{}, nil
	})

	tests := []struct {
		transportType TransportType
		expected      bool
	}{
		{TransportHTTP, true},
		{TransportGRPC, false},
		{TransportWebSocket, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.transportType), func(t *testing.T) {
			result := selector.IsRegistered(tt.transportType)
			if result != tt.expected {
				t.Errorf("Expected IsRegistered(%s)=%v, got %v", tt.transportType, tt.expected, result)
			}
		})
	}
}

func TestTransportSelector_AvailableTransports(t *testing.T) {
	selector := NewTransportSelector()

	// Initially empty
	available := selector.AvailableTransports()
	if len(available) != 0 {
		t.Errorf("Expected 0 available transports, got %d", len(available))
	}

	// Register some transports
	selector.RegisterFactory(TransportHTTP, func(endpoint string) (MessageTransport, error) {
		return &mockTransport{}, nil
	})
	selector.RegisterFactory(TransportGRPC, func(endpoint string) (MessageTransport, error) {
		return &mockTransport{}, nil
	})

	available = selector.AvailableTransports()
	if len(available) != 2 {
		t.Errorf("Expected 2 available transports, got %d", len(available))
	}

	// Check that both are present
	hasHTTP := false
	hasGRPC := false
	for _, t := range available {
		if t == TransportHTTP {
			hasHTTP = true
		}
		if t == TransportGRPC {
			hasGRPC = true
		}
	}

	if !hasHTTP || !hasGRPC {
		t.Errorf("Expected HTTP and gRPC transports in available list")
	}
}

func TestTransportSelector_FactoryError(t *testing.T) {
	selector := NewTransportSelector()

	// Register factory that returns error
	selector.RegisterFactory("error", func(endpoint string) (MessageTransport, error) {
		return nil, fmt.Errorf("factory error")
	})

	transport, err := selector.Select("error", "test")
	if err == nil {
		t.Errorf("Expected error from factory")
	}
	if transport != nil {
		t.Errorf("Expected nil transport on error")
	}
}

func TestTransportSelector_UnregisteredType(t *testing.T) {
	selector := NewTransportSelector()

	transport, err := selector.Select("unregistered", "test")
	if err == nil {
		t.Errorf("Expected error for unregistered transport type")
	}
	if transport != nil {
		t.Errorf("Expected nil transport for unregistered type")
	}
}
