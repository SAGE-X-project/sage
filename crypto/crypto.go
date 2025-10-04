// Copyright (C) 2025 sage-x-project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// SPDX-License-Identifier: LGPL-3.0-or-later


// Package crypto provides cryptographic operations for the SAGE system.
package crypto

// This file is intentionally minimal to avoid circular dependencies.
// The actual implementations are in the subpackages:
// - crypto/keys: Key pair generation and operations
// - crypto/storage: Key storage implementations
// - crypto/formats: Key import/export formats
// - crypto/chain: Blockchain integration
// - crypto/rotation: Key rotation support