// Package registry010 checks operation-scoped authoritative observations.
// Sources must validate complete records, proofs, source identity and finality.
// This package is not a network resolver, signature verifier or session manager.
package registry010

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

var ErrUnreachable = errors.New("record.unreachable")
var ErrStale = errors.New("record.stale")
var ErrRejected = errors.New("record.rejected")
var hex32 = regexp.MustCompile(`^[0-9a-f]{64}$`)
var namePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)
var agentPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// Stamp uses local monotonic milliseconds and trusted Unix seconds. A Clock
// error means the clock cannot currently be trusted; it must fail closed.
type Stamp struct {
	MonoMS int64
	Unix   int64
}
type Clock interface{ Now() (Stamp, error) }

// Config is local trusted deployment configuration, never received from a peer.
type Config struct {
	Source, Registry, Network string
	Blockchain                bool
}

// Key is the public-key projection of a fully validated registry entry.
// Material is canonical lowercase hex. Optional algorithms are not enabled here.
type Key struct {
	Name     string `json:"name"`
	Alg      string `json:"alg"`
	Material string `json:"material"`
	State    string `json:"state"`
	Expires  *int64 `json:"expires,omitempty"`
}

// Snapshot is supplied only by the configured trusted Source. Validated means
// complete record syntax, cryptography and proofs were checked by that source.
// AcquiredMS must use the same local monotonic clock and describe this read,
// not a peer timestamp. Digest commits the entire validated canonical record.
type Snapshot struct {
	Source        string `json:"source"`
	Registry      string `json:"registry"`
	Network       string `json:"network"`
	DID           string `json:"did"`
	Version       string `json:"version"`
	State         string `json:"state"`
	Digest        string `json:"digest"`
	Ready         bool   `json:"ready"`
	Validated     bool   `json:"validated"`
	Finalized     bool   `json:"finalized"`
	Conflicting   bool   `json:"conflicting"`
	AcquiredMS    int64  `json:"acquired_ms"`
	BlockHash     string `json:"block_hash"`
	KeysBlockHash string `json:"keys_block_hash"`
	Keys          []Key  `json:"keys"`
}

// Source must perform a new authoritative read for each call and establish
// readiness including network/finality policy. Errors include timeouts and absence.
// A remote JSON flag is not evidence of validation, readiness or finality.
type Source interface {
	Read(context.Context, string) (Snapshot, error)
}
type Scope struct {
	Registry string `json:"registry"`
	DID      string `json:"did"`
}

// Store must atomically enforce nondecreasing finalized versions, equal-version
// content equality and terminal deactivation, and durably commit before returning.
type Store interface {
	Advance(Scope, uint64, string, bool) error
}

// Gate serializes its operations; each method always obtains a new observation.
type Gate struct {
	mu     sync.Mutex
	cfg    Config
	source Source
	clock  Clock
	store  Store
	last   *Stamp
}

func NewGate(c Config, s Source, k Clock, p Store) (*Gate, error) {
	for _, v := range []string{c.Source, c.Registry, c.Network} {
		if v == "" || len(v) > 256 || strings.ContainsAny(v, "\x00\r\n") {
			return nil, ErrRejected
		}
	}
	if s == nil || k == nil || p == nil {
		return nil, ErrRejected
	}
	return &Gate{cfg: c, source: s, clock: k, store: p}, nil
}
func (g *Gate) sample() (Stamp, error) {
	t, e := g.clock.Now()
	if e != nil || t.MonoMS < 0 || t.Unix < 0 || t.Unix > 9007199254740991 {
		return Stamp{}, ErrUnreachable
	}
	if g.last != nil && (t.MonoMS < g.last.MonoMS || t.Unix < g.last.Unix) {
		return Stamp{}, ErrStale
	}
	g.last = &t
	return t, nil
}
func cloneKey(k Key) Key {
	if k.Expires != nil {
		x := *k.Expires
		k.Expires = &x
	}
	return k
}
func cloneSnapshot(s Snapshot) Snapshot {
	keys := make([]Key, len(s.Keys))
	for i, k := range s.Keys {
		keys[i] = cloneKey(k)
	}
	s.Keys = keys
	return s
}
func (g *Gate) validDID(did string) bool {
	prefix := "did:sage:" + g.cfg.Registry + ":"
	if len(did) > 256 || !strings.HasPrefix(did, prefix) {
		return false
	}
	agent := strings.TrimPrefix(did, prefix)
	return agentPattern.MatchString(agent) && agent != "." && agent != ".."
}
func fresh(start, now Stamp, acquired int64) bool {
	return acquired >= start.MonoMS && acquired <= now.MonoMS && now.MonoMS-acquired <= 5000
}
func (g *Gate) read(ctx context.Context, did string) (Snapshot, Stamp, error) {
	if !g.validDID(did) {
		return Snapshot{}, Stamp{}, ErrRejected
	}
	start, e := g.sample()
	if e != nil {
		return Snapshot{}, Stamp{}, e
	}
	s, e := g.source.Read(ctx, did)
	if e != nil || ctx.Err() != nil {
		return Snapshot{}, Stamp{}, ErrUnreachable
	}
	s = cloneSnapshot(s)
	now, e := g.sample()
	if e != nil {
		return Snapshot{}, Stamp{}, e
	}
	if !s.Ready || s.Source != g.cfg.Source || s.Registry != g.cfg.Registry || s.Network != g.cfg.Network {
		return Snapshot{}, Stamp{}, ErrUnreachable
	}
	if !s.Validated || s.Conflicting || s.DID != did || !s.Finalized || !fresh(start, now, s.AcquiredMS) {
		return Snapshot{}, Stamp{}, ErrStale
	}
	if g.cfg.Blockchain && (!hex32.MatchString(s.BlockHash) || s.KeysBlockHash != s.BlockHash) {
		return Snapshot{}, Stamp{}, ErrStale
	}
	v, e := strconv.ParseUint(s.Version, 10, 64)
	if e != nil || v == 0 || strconv.FormatUint(v, 10) != s.Version || !hex32.MatchString(s.Digest) {
		return Snapshot{}, Stamp{}, ErrRejected
	}
	if s.State != "created" && s.State != "active" && s.State != "deactivated" {
		return Snapshot{}, Stamp{}, ErrRejected
	}
	if len(s.Keys) < 1 || len(s.Keys) > 128 {
		return Snapshot{}, Stamp{}, ErrRejected
	}
	previous := ""
	materials := map[string]bool{}
	for _, k := range s.Keys {
		if !namePattern.MatchString(k.Name) || k.Name <= previous || materials[k.Material] || !hex32.MatchString(k.Material) || (k.Alg != "ed25519" && k.Alg != "x25519") || (k.State != "accepted" && k.State != "revoked") || (k.Expires != nil && (*k.Expires < 0 || *k.Expires > 9007199254740991)) {
			return Snapshot{}, Stamp{}, ErrRejected
		}
		previous = k.Name
		materials[k.Material] = true
	}
	if e = g.store.Advance(Scope{g.cfg.Registry, did}, v, s.Digest, s.State == "deactivated"); e != nil {
		return Snapshot{}, Stamp{}, e
	}
	// Durable storage may block. Recheck at the last gate, after it completes.
	now, e = g.sample()
	if e != nil {
		return Snapshot{}, Stamp{}, e
	}
	if ctx.Err() != nil {
		return Snapshot{}, Stamp{}, ErrUnreachable
	}
	if !fresh(start, now, s.AcquiredMS) {
		return Snapshot{}, Stamp{}, ErrStale
	}
	return s, now, nil
}

// Observe reads a record without granting authority. Inactive records remain readable.
// ErrStale requires a new read; stale data never authorizes a deferred operation.
func (g *Gate) Observe(ctx context.Context, did string) (Snapshot, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	s, _, e := g.read(ctx, did)
	return s, e
}

// Pinned owns exact selected keys; accessors return copies. It is not a portable
// authorization token. Every later decision must call CheckPinned for a fresh read.
type Pinned struct {
	did, registry string
	signing       Key
	kem           *Key
}

func (p *Pinned) DID() string  { return p.did }
func (p *Pinned) Signing() Key { return cloneKey(p.signing) }
func (p *Pinned) KEM() *Key {
	if p.kem == nil {
		return nil
	}
	k := cloneKey(*p.kem)
	return &k
}
func usable(k Key, now int64) bool {
	return k.State == "accepted" && (k.Expires == nil || now < *k.Expires)
}

// Select uses the exact signing key URL and, when requested, the first usable
// X25519 key in ASCII name order. No trial verification or fallback is performed.
func (g *Gate) Select(ctx context.Context, did, signingURL string, requireKEM bool) (*Pinned, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	s, now, e := g.read(ctx, did)
	if e != nil {
		return nil, e
	}
	if s.State != "active" {
		return nil, ErrRejected
	}
	p := &Pinned{did: did, registry: g.cfg.Registry}
	found := false
	for _, k := range s.Keys {
		if !usable(k, now.Unix) {
			continue
		}
		if k.Alg == "ed25519" && signingURL == did+"#"+k.Name {
			p.signing = cloneKey(k)
			found = true
		}
		if requireKEM && k.Alg == "x25519" && p.kem == nil {
			x := cloneKey(k)
			p.kem = &x
		}
	}
	if !found || (requireKEM && p.kem == nil) {
		return nil, ErrRejected
	}
	return p, nil
}
func sameKey(a, b Key) bool {
	return a.Name == b.Name && a.Alg == b.Alg && a.Material == b.Material && ((a.Expires == nil && b.Expires == nil) || (a.Expires != nil && b.Expires != nil && *a.Expires == *b.Expires))
}

// CheckPinned revalidates the original material without changing KEM/signing
// selection. A caller owning a session must close it on failure.
func (g *Gate) CheckPinned(ctx context.Context, p *Pinned) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if p == nil || p.registry != g.cfg.Registry {
		return ErrRejected
	}
	s, now, e := g.read(ctx, p.did)
	if e != nil {
		return e
	}
	if s.State != "active" {
		return ErrRejected
	}
	wanted := []Key{p.signing}
	if p.kem != nil {
		wanted = append(wanted, *p.kem)
	}
	for _, w := range wanted {
		found := false
		for _, k := range s.Keys {
			if sameKey(k, w) && usable(k, now.Unix) {
				found = true
				break
			}
		}
		if !found {
			return ErrRejected
		}
	}
	return nil
}
