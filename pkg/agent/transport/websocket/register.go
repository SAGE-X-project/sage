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

package websocket

import (
	"github.com/sage-x-project/sage/pkg/agent/transport"
)

// init registers the WebSocket transport factory with the default selector
func init() {
	// Register WebSocket factory
	transport.DefaultSelector.RegisterFactory(transport.TransportWebSocket, func(endpoint string) (transport.MessageTransport, error) {
		return NewWSTransport(endpoint), nil
	})

	// Register WebSocket Secure factory (same implementation)
	transport.DefaultSelector.RegisterFactory(transport.TransportWebSocketSecure, func(endpoint string) (transport.MessageTransport, error) {
		return NewWSTransport(endpoint), nil
	})
}
