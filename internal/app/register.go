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

// Package app is the composition root of the SAGE binaries. It performs the
// explicit wiring that the library packages no longer do in init():
// registering chain providers, DID clients and transports. Library consumers
// call the individual Register functions (or their own equivalents) instead.
package app

import (
	chaineth "github.com/sage-x-project/sage/pkg/agent/crypto/chain/ethereum"
	chainsol "github.com/sage-x-project/sage/pkg/agent/crypto/chain/solana"
	dideth "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
	didsol "github.com/sage-x-project/sage/pkg/agent/did/solana"
	transporthttp "github.com/sage-x-project/sage/pkg/agent/transport/http"
	transportws "github.com/sage-x-project/sage/pkg/agent/transport/websocket"
)

// RegisterDefaults registers every built-in chain provider, DID client and
// transport. It is idempotent and is called at the start of each binary's main.
func RegisterDefaults() {
	chaineth.Register()
	chainsol.Register()
	dideth.Register()
	didsol.Register()
	transporthttp.Register()
	transportws.Register()
}
