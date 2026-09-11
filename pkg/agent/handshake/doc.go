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

// Package handshake implements the legacy four-phase (Invitation, Request,
// Response, Complete) session handshake.
//
// Deprecated: the package is retained for existing integrations only and is
// scheduled for removal (docs/refactoring/BACKLOG.md, D-06). Its server does
// not validate the Nonce and Timestamp fields of incoming messages, and no
// SAGE binary or example uses it. New code should establish sessions with
// the 1-RTT HPKE handshake in package hpke, which binds the session to both
// DIDs, checks freshness and replay, and signs the server response.
package handshake
