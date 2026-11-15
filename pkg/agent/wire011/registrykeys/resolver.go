// Package registrykeys resolves wire011 signing keys from a SAGE 0.11.0 chain
// Registry. Every resolution reads the record again at the provider's
// finalized block; there is no positive cache, so a revocation takes effect
// as soon as it is final.
package registrykeys

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/sage-x-project/sage/pkg/agent/registry011"
	"github.com/sage-x-project/sage/pkg/agent/wire011"
)

// RecordSource returns the canonical record of one agent identifier.
type RecordSource interface {
	Record(ctx context.Context, agentID string) (present bool, canonical []byte, err error)
}

// Resolver implements wire011.KeyResolver for agents of one Registry.
type Resolver struct {
	Source     RecordSource
	RegistryID string // e.g. eip155:31337:0x...; DIDs of other Registries are refused
	Clock      wire011.Clock
}

var _ wire011.KeyResolver = (*Resolver)(nil)

// NewRegistryResolver reads records through rpc, which must belong to network.
func NewRegistryResolver(rpc *registry011.ReadRPC, network registry011.NetworkProfile, clock wire011.Clock) *Resolver {
	return &Resolver{Source: rpcSource{rpc: rpc, registry: network.RegistryAddress()}, RegistryID: network.RegistryID(), Clock: clock}
}

type rpcSource struct {
	rpc      *registry011.ReadRPC
	registry string
}

func (s rpcSource) Record(ctx context.Context, agentID string) (bool, []byte, error) {
	if s.rpc == nil {
		return false, nil, errors.New("registrykeys: no RPC")
	}
	reads, err := s.rpc.PinAgentReads(ctx, s.registry, agentID)
	if err != nil {
		return false, nil, err
	}
	return reads.RecordBytes(ctx)
}

type record struct {
	ID    string `json:"id"`
	State string `json:"state"`
	Keys  []struct {
		Name    string `json:"name"`
		Alg     string `json:"alg"`
		Key     string `json:"key"`
		State   string `json:"state"`
		Expires *int64 `json:"expires"`
	} `json:"keys"`
}

// ResolveSigningKey returns the key only if the record is active and the named
// key is an accepted, unexpired Ed25519 key. X25519 keys never sign.
func (r *Resolver) ResolveSigningKey(ctx context.Context, kid string) (ed25519.PublicKey, error) {
	if r == nil || r.Source == nil || r.Clock == nil || r.RegistryID == "" {
		return nil, errors.New("registrykeys: resolver not configured")
	}
	did, name, ok := strings.Cut(kid, "#")
	prefix := "did:sage:" + r.RegistryID + ":"
	agentID, local := strings.CutPrefix(did, prefix)
	if !ok || !local || agentID == "" || name == "" || strings.ContainsAny(agentID, ":#/") {
		return nil, fmt.Errorf("registrykeys: %q is not a key of this Registry", kid)
	}
	present, canonical, err := r.Source.Record(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("registrykeys: read record: %w", err)
	}
	if !present {
		return nil, errors.New("registrykeys: no record")
	}
	now, err := r.Clock.Now()
	if err != nil {
		return nil, fmt.Errorf("registrykeys: clock: %w", err)
	}
	return SelectSigningKey(canonical, did, name, now.Unix())
}

// SelectSigningKey picks the named signing key from a canonical record.
func SelectSigningKey(canonical []byte, did, name string, now int64) (ed25519.PublicKey, error) {
	var rec record
	if err := json.Unmarshal(canonical, &rec); err != nil {
		return nil, fmt.Errorf("registrykeys: record: %w", err)
	}
	if rec.ID != did {
		return nil, errors.New("registrykeys: record is for another DID")
	}
	if rec.State != "active" {
		return nil, fmt.Errorf("registrykeys: record is %s", rec.State)
	}
	for _, k := range rec.Keys {
		if k.Name != name {
			continue
		}
		switch {
		case k.Alg != "ed25519":
			return nil, fmt.Errorf("registrykeys: key %s is %s, not a signing key", name, k.Alg)
		case k.State != "accepted":
			return nil, fmt.Errorf("registrykeys: key %s is %s", name, k.State)
		case k.Expires != nil && now >= *k.Expires:
			return nil, fmt.Errorf("registrykeys: key %s expired", name)
		}
		raw, err := base64.RawURLEncoding.Strict().DecodeString(k.Key)
		if err != nil || len(raw) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("registrykeys: key %s encoding", name)
		}
		return ed25519.PublicKey(raw), nil
	}
	return nil, fmt.Errorf("registrykeys: no key %s", name)
}
