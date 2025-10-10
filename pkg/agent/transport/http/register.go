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

package http

import (
	"github.com/sage-x-project/sage/pkg/agent/transport"
)

// init registers the HTTP transport factory with the default selector
func init() {
	// Register HTTP factory
	transport.DefaultSelector.RegisterFactory(transport.TransportHTTP, func(endpoint string) (transport.MessageTransport, error) {
		return NewHTTPTransport(endpoint), nil
	})

	// Register HTTPS factory (same implementation)
	transport.DefaultSelector.RegisterFactory(transport.TransportHTTPS, func(endpoint string) (transport.MessageTransport, error) {
		return NewHTTPTransport(endpoint), nil
	})
}
