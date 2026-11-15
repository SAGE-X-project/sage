package hpke

import (
	"context"

	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

// SAGE 0.11.0 endpoints. The completion, record session and HTTP binding are
// the 0.10.0 constructions with the 0.11.0 labels and field values of
// profile011; the types are shared and keep their historical names.
//
// A 0.11.0 endpoint signs and accepts only 0.11.0 messages. It never falls
// back to 0.10.0, and a 0.10.0 endpoint never accepts 0.11.0 messages. The
// replay store may be shared between the two: a denial recorded by either
// only ever refuses more.

// NewCompletionEndpoint011 is NewCompletionEndpoint010 for SAGE 0.11.0.
func NewCompletionEndpoint011(did, kid string, signingSeed, kem []byte, g *registry010.Gate, c registry010.Clock, r ReplayStore010) (*CompletionEndpoint010, error) {
	return as011(NewCompletionEndpoint010(did, kid, signingSeed, kem, g, c, r))
}

// NewCustodyCompletionEndpoint011 is NewCustodyCompletionEndpoint010 for SAGE 0.11.0.
func NewCustodyCompletionEndpoint011(ctx context.Context, did, kid string, custody Ed25519Custody010, kem []byte, g *registry010.Gate, c registry010.Clock, r ReplayStore010) (*CompletionEndpoint010, error) {
	return as011(NewCustodyCompletionEndpoint010(ctx, did, kid, custody, kem, g, c, r))
}

// NewProtectedCompletionEndpoint011 is NewProtectedCompletionEndpoint010 for SAGE 0.11.0.
func NewProtectedCompletionEndpoint011(ctx context.Context, did, kid string, custody Ed25519Custody010, kem X25519Custody010, g *registry010.Gate, c registry010.Clock, r ReplayStore010) (*CompletionEndpoint010, error) {
	return as011(NewProtectedCompletionEndpoint010(ctx, did, kid, custody, kem, g, c, r))
}

// as011 switches a just-constructed, never-used endpoint to the 0.11.0 profile.
func as011(e *CompletionEndpoint010, err error) (*CompletionEndpoint010, error) {
	if err != nil {
		return nil, err
	}
	e.p = profile011
	return e, nil
}

// Version reports the SAGE version this endpoint speaks.
func (e *CompletionEndpoint010) Version() string {
	if e == nil || e.p == nil {
		return ""
	}
	return e.p.version
}
