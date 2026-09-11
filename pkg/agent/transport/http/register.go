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
// Register adds this transport's factories to transport.DefaultSelector so
// transport.SelectByURL can build it from an endpoint. Call it from the
// composition root (internal/app.RegisterDefaults does).
func Register() {
	for _, kind := range []transport.TransportType{transport.TransportHTTP, transport.TransportHTTPS} {
		transport.DefaultSelector.RegisterFactory(kind, func(endpoint string) (transport.MessageTransport, error) {
			return NewHTTPTransport(endpoint), nil
		})
	}
}
