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
	"fmt"
	"net/url"
	"strings"
)

// TransportType identifies the transport protocol to use
type TransportType string

const (
	// TransportHTTP uses HTTP/REST protocol
	TransportHTTP TransportType = "http"

	// TransportHTTPS uses HTTP/REST with TLS
	TransportHTTPS TransportType = "https"

	// TransportGRPC uses gRPC protocol (A2A)
	TransportGRPC TransportType = "grpc"

	// TransportWebSocket uses WebSocket protocol
	TransportWebSocket TransportType = "ws"

	// TransportWebSocketSecure uses WebSocket with TLS
	TransportWebSocketSecure TransportType = "wss"
)

// TransportFactory creates a MessageTransport instance
type TransportFactory func(endpoint string) (MessageTransport, error)

// TransportSelector manages transport selection and creation
type TransportSelector struct {
	factories map[TransportType]TransportFactory
}

// NewTransportSelector creates a new transport selector with default factories
func NewTransportSelector() *TransportSelector {
	s := &TransportSelector{
		factories: make(map[TransportType]TransportFactory),
	}

	// Register default factories
	s.RegisterHTTPFactory()

	return s
}

// RegisterFactory registers a transport factory for a specific type
func (s *TransportSelector) RegisterFactory(transportType TransportType, factory TransportFactory) {
	s.factories[transportType] = factory
}

// RegisterHTTPFactory registers the HTTP/HTTPS transport factory
func (s *TransportSelector) RegisterHTTPFactory() {
	// HTTP factory will be registered when http package is imported
	// This is a placeholder for dependency injection
}

// SelectByURL creates a transport based on URL scheme
//
// Supported URL schemes:
//   - http://agent.example.com -> HTTP transport
//   - https://agent.example.com -> HTTPS transport
//   - grpc://agent.example.com:50051 -> gRPC transport (requires a2a tag)
//   - ws://agent.example.com/ws -> WebSocket transport
//   - wss://agent.example.com/ws -> WebSocket with TLS
//
// Example:
//
//	selector := transport.NewTransportSelector()
//	transport, err := selector.SelectByURL("https://agent.example.com")
func (s *TransportSelector) SelectByURL(endpoint string) (MessageTransport, error) {
	// Parse URL
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %q: %w", endpoint, err)
	}

	// Determine transport type from scheme
	var transportType TransportType
	switch strings.ToLower(parsedURL.Scheme) {
	case "http":
		transportType = TransportHTTP
	case "https":
		transportType = TransportHTTPS
	case "grpc":
		transportType = TransportGRPC
	case "ws":
		transportType = TransportWebSocket
	case "wss":
		transportType = TransportWebSocketSecure
	default:
		return nil, fmt.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
	}

	return s.Select(transportType, endpoint)
}

// Select creates a transport of the specified type
func (s *TransportSelector) Select(transportType TransportType, endpoint string) (MessageTransport, error) {
	factory, ok := s.factories[transportType]
	if !ok {
		return nil, fmt.Errorf("transport type %q not registered (missing import or build tag?)", transportType)
	}

	transport, err := factory(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s transport: %w", transportType, err)
	}

	return transport, nil
}

// IsRegistered checks if a transport type is registered
func (s *TransportSelector) IsRegistered(transportType TransportType) bool {
	_, ok := s.factories[transportType]
	return ok
}

// AvailableTransports returns a list of registered transport types
func (s *TransportSelector) AvailableTransports() []TransportType {
	types := make([]TransportType, 0, len(s.factories))
	for t := range s.factories {
		types = append(types, t)
	}
	return types
}

// DefaultSelector is the global transport selector with all registered transports
var DefaultSelector = NewTransportSelector()

// SelectByURL is a convenience function using the default selector
//
// Example:
//
//	transport, err := transport.SelectByURL("https://agent.example.com")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	client := handshake.NewClient(transport, keyPair)
func SelectByURL(endpoint string) (MessageTransport, error) {
	return DefaultSelector.SelectByURL(endpoint)
}

// Select is a convenience function using the default selector
func Select(transportType TransportType, endpoint string) (MessageTransport, error) {
	return DefaultSelector.Select(transportType, endpoint)
}
