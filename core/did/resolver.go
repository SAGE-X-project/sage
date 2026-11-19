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

package did

import (
	"context"
	"time"
)

// KeyType is the registry's storage code of a key (01-crypto.md §1). It is
// not an algorithm identifier.
type KeyType uint8

// Registry key types.
const (
	KeyTypeECDSA   KeyType = 0 // secp256k1
	KeyTypeEd25519 KeyType = 1
	KeyTypeX25519  KeyType = 2
)

// Key is one entry of a record's keys[] (§3). Its values are claimed by
// the registry and unchecked: eligibility is decided from them by the
// verifier (§3.1), never from Verified.
type Key struct {
	Type      KeyType
	KeyData   []byte // raw public key bytes as stored
	Signature []byte // the §4 proof of possession for types 0 and 1
	Verified  bool   // the registry's flag; not used for eligibility (E06-09)
	CreatedAt time.Time
}

// Record is the registry record of an agent (§3, AgentMetadataV4), as the
// registry returned it. Nothing in it is verified.
//
// DID is parsed from the record's stored did member, so DID.AsSent() is the
// stored string that the proof-of-possession challenge uses (§4); an
// adapter must not put the queried DID there. A stored did that Parse
// rejects fails resolution.
type Record struct {
	DID          DID
	Name         string
	Description  string
	Endpoint     string
	Keys         []Key  // current keys in registry order
	Capabilities []byte // the free-form object as raw JSON
	Owner        string // chain address
	IsActive     bool
	PublicKEMKey []byte // dedicated KEM field; the KEM key only under §3.1
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Observation says where and when a record was read.
type Observation struct {
	At     time.Time // when the registry was read, not when a cache served the record
	Block  uint64    // block the record was read at
	Source string    // registry and network, for diagnostics
}

// KeyResolver resolves a DID to its registry record. The host implements
// it (an adapter per chain). An implementation resolves each chain only on
// its one configured network and registry and never falls back to another
// (06 §2, E06-07); any error fails the verification that asked for it. A
// positive result older than 300 seconds is not relied on, and when the
// registry cannot be read the resolver returns an error rather than an
// older positive result (E06-12).
type KeyResolver interface {
	Resolve(ctx context.Context, d DID) (Record, Observation, error)
}
