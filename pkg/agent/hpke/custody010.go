package hpke

import (
	"context"
	"crypto/ed25519"
	"reflect"

	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

// Ed25519Custody010 is protected custody for one fixed registered Ed25519
// signing key. The endpoint never receives private bytes from it. PublicKey and
// Sign must be bounded, cancellation-aware, concurrency-safe and immutable for
// the endpoint's lifetime, and Sign must sign the supplied bytes exactly. The
// endpoint verifies every returned signature against the public key read at
// construction and fails closed. Custody must stay outside model/plugin reach.
type Ed25519Custody010 interface {
	PublicKey(context.Context) (ed25519.PublicKey, error)
	Sign(context.Context, []byte) ([]byte, error)
}

// NewCustodyCompletionEndpoint010 is NewCompletionEndpoint010 with the signing
// key held by external custody. The optional X25519 KEM private key is still a
// local copy. Exact correspondence with current registered public bytes is
// checked on use, as with the seed constructor.
func NewCustodyCompletionEndpoint010(ctx context.Context, did, kid string, custody Ed25519Custody010, kem []byte, g *registry010.Gate, c registry010.Clock, r ReplayStore010) (endpoint *CompletionEndpoint010, err error) {
	defer func() {
		if recover() != nil {
			endpoint, err = nil, errCompletion010
		}
	}()
	if ctx == nil || ctx.Err() != nil || absentCustody010(custody) || !did010(did) || !key010(kid, did) || (len(kem) != 0 && len(kem) != 32) || g == nil || c == nil || r == nil {
		return nil, errCompletion010
	}
	public, x := custody.PublicKey(ctx)
	if x != nil || len(public) != ed25519.PublicKeySize || ctx.Err() != nil {
		return nil, errCompletion010
	}
	return &CompletionEndpoint010{registry: g, clock: c, replay: r, did: did, kid: kid, public: append(ed25519.PublicKey(nil), public...), custody: custody, kem: append([]byte(nil), kem...)}, nil
}

// NewProtectedCompletionEndpoint010 keeps both the Ed25519 signing key and the
// optional X25519 KEM key in external custody. kem is nil for an endpoint that
// only initiates. Both public keys are read once at construction; the KEM
// public key must match the current registered KEM key on every response, and
// custody performs only the X25519 operation of HPKE decapsulation.
func NewProtectedCompletionEndpoint010(ctx context.Context, did, kid string, custody Ed25519Custody010, kem X25519Custody010, g *registry010.Gate, c registry010.Clock, r ReplayStore010) (endpoint *CompletionEndpoint010, err error) {
	defer func() {
		if recover() != nil {
			endpoint, err = nil, errCompletion010
		}
	}()
	e, err := NewCustodyCompletionEndpoint010(ctx, did, kid, custody, nil, g, c, r)
	if err != nil {
		return nil, err
	}
	if kem == nil || reflect.ValueOf(kem).Kind() == reflect.Pointer && reflect.ValueOf(kem).IsNil() {
		return e, nil
	}
	public, x := kem.PublicKey(ctx)
	if x != nil || len(public) != 32 || ctx.Err() != nil {
		return nil, errCompletion010
	}
	e.kemCustody, e.kemPublic = kem, append([]byte(nil), public...)
	return e, nil
}

// kemReady reports whether exactly one KEM source is retained for responding.
func (e *CompletionEndpoint010) kemReady() bool {
	if e.kemCustody != nil {
		return len(e.kem) == 0 && len(e.kemPublic) == 32
	}
	return len(e.kem) == 32
}

func absentCustody010(v Ed25519Custody010) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	}
	return false
}

// signable reports whether exactly one signing source is retained. Callers hold
// e.mu or own the endpoint during construction.
func (e *CompletionEndpoint010) signable() bool {
	if len(e.public) != ed25519.PublicKeySize {
		return false
	}
	return (len(e.signing) == ed25519.PrivateKeySize) != (e.custody != nil)
}

// sign signs message with the retained key. A custody result is copied and must
// verify under the constructed public key; otherwise no signature is returned.
func (e *CompletionEndpoint010) sign(ctx context.Context, message []byte) (signature []byte, err error) {
	defer func() {
		if recover() != nil {
			signature, err = nil, errCompletion010
		}
	}()
	if !e.signable() || e.retired.Load() {
		return nil, errCompletion010
	}
	if e.custody == nil {
		return ed25519.Sign(e.signing, message), nil
	}
	if ctx == nil || ctx.Err() != nil {
		return nil, errCompletion010
	}
	owned := append([]byte(nil), message...)
	raw, x := e.custody.Sign(ctx, append([]byte(nil), owned...))
	if x != nil || ctx.Err() != nil {
		return nil, errCompletion010
	}
	signature = append([]byte(nil), raw...)
	if len(signature) != ed25519.SignatureSize || !ed25519.Verify(e.public, owned, signature) {
		return nil, errCompletion010
	}
	return signature, nil
}
