package guard010

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"strings"
	"sync"

	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

// RegistryAuthority binds one principal and signing key to a trusted registry gate.
// Every callback performs a new authoritative read, including the final Now before
// dispatch. No earlier positive observation authorizes later work. The first key
// is pinned for this authority's lifetime; replacing it requires trusted setup.
// The host must supply a validating Source, durable Store and bounded callbacks,
// and must not reenter this authority from those callbacks. This is not a resolver.
type RegistryAuthority struct {
	mu            sync.Mutex
	gate          *registry010.Gate
	issuer, keyid string
	key           *registry010.Key
}

// NewRegistryAuthority configures a binding without making an observation.
// The gate and its dependencies remain trusted local configuration, never wire data.
func NewRegistryAuthority(gate *registry010.Gate, issuer, keyid string) (*RegistryAuthority, error) {
	parts := strings.Split(keyid, "#")
	if gate == nil || !did(issuer) || len(parts) != 2 || parts[0] != issuer || !keyName.MatchString(parts[1]) {
		return nil, ErrInvalid
	}
	return &RegistryAuthority{gate: gate, issuer: issuer, keyid: keyid}, nil
}

type registryObservation struct {
	key     ed25519.PublicKey
	stamp   registry010.Stamp
	expires *int64
}

func (a *RegistryAuthority) read(ctx context.Context) (ed25519.PublicKey, int64, error) {
	o, err := a.observe(ctx)
	if err != nil {
		return nil, 0, err
	}
	return o.key, o.stamp.Unix, nil
}
func (a *RegistryAuthority) observe(ctx context.Context) (*registryObservation, error) {
	if a == nil || ctx == nil || ctx.Err() != nil {
		return nil, ErrInvalid
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.gate == nil {
		return nil, ErrInvalid
	}
	p, stamp, err := a.gate.SelectWithTime(ctx, a.issuer, a.keyid, false)
	if err != nil || ctx.Err() != nil {
		return nil, ErrInvalid
	}
	key := p.Signing()
	if a.key != nil && (key.Name != a.key.Name || key.Alg != a.key.Alg || key.Material != a.key.Material ||
		(key.Expires == nil) != (a.key.Expires == nil) || (key.Expires != nil && *key.Expires != *a.key.Expires)) {
		return nil, ErrInvalid
	}
	raw, err := hex.DecodeString(key.Material)
	if err != nil || len(raw) != ed25519.PublicKeySize || !strongPoint(raw) {
		return nil, ErrInvalid
	}
	a.key = &key
	return &registryObservation{key: ed25519.PublicKey(raw), stamp: stamp, expires: key.Expires}, nil
}

// Now revalidates the exact pinned principal/key and returns that read's gate time.
func (a *RegistryAuthority) Now(ctx context.Context) (int64, error) {
	_, now, err := a.read(ctx)
	return now, err
}

// ActiveKey freshly resolves only the locally configured issuer and key ID.
func (a *RegistryAuthority) ActiveKey(ctx context.Context, issuer, keyid string) (ed25519.PublicKey, error) {
	if a == nil || issuer != a.issuer || keyid != a.keyid {
		return nil, ErrInvalid
	}
	key, _, err := a.read(ctx)
	return key, err
}
