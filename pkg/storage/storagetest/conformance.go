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

// Package storagetest holds the conformance suite every storage.Store
// implementation must pass. A backend (memory, PostgreSQL, ...) calls
// RunConformance from its own tests with a constructor for a fresh, empty
// store, so all backends are held to the same contract: duplicate keys are
// rejected, missing keys are errors, expired entries are invisible and
// reclaimable, nonces are single-use.
package storagetest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sage-x-project/sage/pkg/storage"
)

// RunConformance runs every conformance test against stores produced by
// newStore. newStore must return an empty store; it is called once per
// sub-test and the store is closed afterwards.
func RunConformance(t *testing.T, newStore func(t *testing.T) storage.Store) {
	t.Helper()
	run := func(name string, fn func(t *testing.T, s storage.Store)) {
		t.Run(name, func(t *testing.T) {
			s := newStore(t)
			t.Cleanup(func() { _ = s.Close() })
			require.NoError(t, s.Ping(context.Background()))
			fn(t, s)
		})
	}
	run("Nonce/single-use", testNonceSingleUse)
	run("Nonce/expiry", testNonceExpiry)
	run("Session/lifecycle", testSessionLifecycle)
	run("Session/expiry", testSessionExpiry)
	run("DID/lifecycle", testDIDLifecycle)
}

func testNonceSingleUse(t *testing.T, s storage.Store) {
	ctx := context.Background()
	ns := s.NonceStore()
	exp := time.Now().Add(time.Hour)

	used, err := ns.IsUsed(ctx, "n1")
	require.NoError(t, err)
	assert.False(t, used, "unknown nonce is not used")

	require.NoError(t, ns.CheckAndStore(ctx, "n1", "sess-1", exp))
	require.Error(t, ns.CheckAndStore(ctx, "n1", "sess-1", exp), "a nonce is accepted exactly once")
	require.Error(t, ns.CheckAndStore(ctx, "n1", "sess-2", exp), "regardless of the session that presents it")

	used, err = ns.IsUsed(ctx, "n1")
	require.NoError(t, err)
	assert.True(t, used)

	require.NoError(t, ns.CheckAndStore(ctx, "n2", "sess-1", exp))
	count, err := ns.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func testNonceExpiry(t *testing.T, s storage.Store) {
	ctx := context.Background()
	ns := s.NonceStore()

	require.NoError(t, ns.CheckAndStore(ctx, "old", "sess", time.Now().Add(-time.Minute)))
	require.NoError(t, ns.CheckAndStore(ctx, "fresh", "sess", time.Now().Add(time.Hour)))

	used, err := ns.IsUsed(ctx, "old")
	require.NoError(t, err)
	assert.False(t, used, "an expired nonce no longer counts as used")

	deleted, err := ns.DeleteExpired(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	count, err := ns.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Once reclaimed, the value may be presented again.
	require.NoError(t, ns.CheckAndStore(ctx, "old", "sess", time.Now().Add(time.Hour)))
}

func testSessionLifecycle(t *testing.T, s storage.Store) {
	ctx := context.Background()
	ss := s.SessionStore()
	now := time.Now()
	sess := &storage.Session{
		ID: "s1", ClientDID: "did:sage:ethereum:0xclient", ServerDID: "did:sage:ethereum:0xserver",
		SessionKey: []byte("key-material"), CreatedAt: now, ExpiresAt: now.Add(time.Hour), LastActivity: now,
		Metadata: map[string]interface{}{"role": "initiator"},
	}

	_, err := ss.Get(ctx, "s1")
	require.Error(t, err, "missing session is an error")

	require.NoError(t, ss.Create(ctx, sess))
	require.Error(t, ss.Create(ctx, sess), "duplicate session id is rejected")

	// The store must not alias the caller's buffers.
	sess.SessionKey[0] ^= 0xff
	got, err := ss.Get(ctx, "s1")
	require.NoError(t, err)
	assert.Equal(t, []byte("key-material"), got.SessionKey)
	assert.Equal(t, "did:sage:ethereum:0xclient", got.ClientDID)
	assert.Equal(t, "initiator", got.Metadata["role"])

	got.LastActivity = now.Add(time.Minute)
	require.NoError(t, ss.Update(ctx, got))
	again, err := ss.Get(ctx, "s1")
	require.NoError(t, err)
	assert.WithinDuration(t, now.Add(time.Minute), again.LastActivity, time.Second)

	require.Error(t, ss.Update(ctx, &storage.Session{ID: "missing", ExpiresAt: now.Add(time.Hour)}), "updating a missing session is an error")

	require.NoError(t, ss.Delete(ctx, "s1"))
	require.Error(t, ss.Delete(ctx, "s1"), "deleting twice is an error")
	_, err = ss.Get(ctx, "s1")
	require.Error(t, err)
}

func testSessionExpiry(t *testing.T, s storage.Store) {
	ctx := context.Background()
	ss := s.SessionStore()
	now := time.Now()
	require.NoError(t, ss.Create(ctx, &storage.Session{ID: "expired", CreatedAt: now.Add(-2 * time.Hour), ExpiresAt: now.Add(-time.Hour)}))
	require.NoError(t, ss.Create(ctx, &storage.Session{ID: "live", CreatedAt: now, ExpiresAt: now.Add(time.Hour)}))

	_, err := ss.Get(ctx, "expired")
	require.Error(t, err, "an expired session is not returned")
	_, err = ss.Get(ctx, "live")
	require.NoError(t, err)

	deleted, err := ss.DeleteExpired(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)
	_, err = ss.Get(ctx, "live")
	require.NoError(t, err)
}

func testDIDLifecycle(t *testing.T, s storage.Store) {
	ctx := context.Background()
	ds := s.DIDStore()
	now := time.Now()
	owner := "0x1111111111111111111111111111111111111111"
	d1 := &storage.DID{DID: "did:sage:ethereum:0xa", PublicKey: []byte{1, 2, 3}, OwnerAddress: owner, KeyType: "secp256k1", CreatedAt: now, UpdatedAt: now}
	d2 := &storage.DID{DID: "did:sage:ethereum:0xb", PublicKey: []byte{4, 5, 6}, OwnerAddress: owner, KeyType: "ed25519", CreatedAt: now, UpdatedAt: now}
	other := &storage.DID{DID: "did:sage:ethereum:0xc", PublicKey: []byte{7}, OwnerAddress: "0x2222222222222222222222222222222222222222", KeyType: "ed25519", CreatedAt: now, UpdatedAt: now}

	_, err := ds.Get(ctx, d1.DID)
	require.Error(t, err, "missing DID is an error")

	require.NoError(t, ds.Create(ctx, d1))
	require.Error(t, ds.Create(ctx, d1), "duplicate DID is rejected")
	require.NoError(t, ds.Create(ctx, d2))
	require.NoError(t, ds.Create(ctx, other))

	got, err := ds.Get(ctx, d1.DID)
	require.NoError(t, err)
	assert.Equal(t, []byte{1, 2, 3}, got.PublicKey)
	assert.Equal(t, owner, got.OwnerAddress)

	got.Revoked = true
	require.NoError(t, ds.Update(ctx, got))
	again, err := ds.Get(ctx, d1.DID)
	require.NoError(t, err)
	assert.True(t, again.Revoked)
	require.Error(t, ds.Update(ctx, &storage.DID{DID: "did:sage:ethereum:0xmissing"}), "updating a missing DID is an error")

	list, err := ds.ListByOwner(ctx, owner)
	require.NoError(t, err)
	require.Len(t, list, 2)
	ids := []string{list[0].DID, list[1].DID}
	assert.ElementsMatch(t, []string{d1.DID, d2.DID}, ids)

	require.NoError(t, ds.Delete(ctx, d2.DID))
	require.Error(t, ds.Delete(ctx, d2.DID), "deleting twice is an error")
	list, err = ds.ListByOwner(ctx, owner)
	require.NoError(t, err)
	require.Len(t, list, 1)
}
