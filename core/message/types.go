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


package message

import (
	"time"
)


type MessageControlHeader struct{
	// Sequence is an ever‑increasing packet counter
    Sequence uint64 	`json:"sequence"`
    // Nonce is a one‑time random value to prevent replay
    Nonce string 		`json:"nonce"`
    // Timestamp records when this packet was generated
    Timestamp time.Time `json:"timestamp"`
}

type BaseMessage struct {
	ContextID       string      `json:"-"`
}

type ControlHeader interface {
    GetNonce() string  
    GetTimestamp() time.Time
	GetSequence() uint64
}