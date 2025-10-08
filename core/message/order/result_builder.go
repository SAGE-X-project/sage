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


package order

import "github.com/sage-x-project/sage/core/message"

// Result holds the outcome of ordering logic for a message.
type Result struct {
	IsProcessed   bool                     // true if message was processed immediately
	IsDuplicate   bool                     // true if message ID was already seen
	IsWaiting     bool                     // true if message was buffered for later
	ReadyMessages []message.ControlHeader  // any previously buffered messages now ready
}

// ResultBuilder builds an Result via chained setters.
type ResultBuilder struct {
	res *Result
}

// NewResultBuilder creates a new builder with default (zero) values.
func NewResultBuilder() *ResultBuilder {
	return &ResultBuilder{
		res: &Result{
			IsProcessed:   false,
			IsDuplicate:   false,
			IsWaiting:     false,
			ReadyMessages: []message.ControlHeader{},
		},
	}
}

// WithProcessed sets the IsProcessed flag.
func (b *ResultBuilder) WithProcessed(p bool) *ResultBuilder {
	b.res.IsProcessed = p
	return b
}

// WithDuplicate sets the IsDuplicate flag.
func (b *ResultBuilder) WithDuplicate(d bool) *ResultBuilder {
	b.res.IsDuplicate = d
	return b
}

// WithWaiting sets the IsWaiting flag.
func (b *ResultBuilder) WithWaiting(w bool) *ResultBuilder {
	b.res.IsWaiting = w
	return b
}

// WithReadyMessages sets the ReadyMessages slice.
func (b *ResultBuilder) WithReadyMessages(msgs []message.ControlHeader) *ResultBuilder {
	b.res.ReadyMessages = msgs
	return b
}

// Build finalizes and returns the Result.
func (b *ResultBuilder) Build() *Result {
	return b.res
}
