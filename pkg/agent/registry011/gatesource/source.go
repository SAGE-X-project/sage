// Package gatesource feeds SAGE 0.11.0 chain Registry records of a local
// development chain into a registry010.Gate, so that the HPKE completion and
// session code can select and recheck keys of 0.11.0 agents on Anvil.
//
// It refuses qualifying network profiles. A qualifying deployment needs the
// registry011 replay Source (history replay, lineage and a qualified
// observation policy); this package does none of that.
//
// Each Read is a new observation; nothing is cached. What it establishes:
//
//   - the acquisition time is sampled on the Gate's clock before the read
//     starts, so a slow read cannot look fresh;
//   - the provider's finalized header is no older than Limits.MaxFinalizedAge
//     and never goes back to a lower block or another hash at the same height;
//   - at that block, the runtime code of every contract in the deployment
//     record has its recorded SHA-256;
//   - the record passes registry011.CheckRecord: canonical bytes, exact spec 09
//     members and bounds, and every key proof with the 0.11.0 challenge;
//   - the record names the requested DID.
//
// The record history is not replayed, and the provider is the local node the
// developer runs. Validated in the snapshot means exactly the list above.
package gatesource

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/sage-x-project/sage/pkg/agent/registry010"
	"github.com/sage-x-project/sage/pkg/agent/registry011"
)

var blockHashPattern = regexp.MustCompile(`^0x[0-9a-f]{64}$`)

// Profile is the SAGE version of the records this Source validates. Only
// 0.11.0 HPKE endpoints accept a Gate built on it.
const Profile = "0.11.0"

// Read is one pinned read: the finalized header and the record at that block.
// A Reader must have checked the deployed code at that block.
type Read struct {
	Header  registry011.FinalizedHeader
	Present bool
	Record  []byte
}

// Reader performs one new pinned read per call.
type Reader interface {
	Read(ctx context.Context, agentID string) (Read, error)
}

// Limits bound what one read may accept.
type Limits struct {
	// MaxFinalizedAge is the largest accepted difference, in seconds, between
	// the trusted clock and the finalized header's timestamp. An idle Anvil
	// mines no blocks, so a local chain needs a generous value.
	MaxFinalizedAge int64
}

// Source implements registry010.Source for one development chain Registry.
type Source struct {
	name, registryID, network string
	reader                    Reader
	clock                     registry010.Clock
	limits                    Limits

	mu   sync.Mutex
	last *registry011.FinalizedHeader
}

var _ registry010.Source = (*Source)(nil)

// NewDevelopment reads through rpc, which must belong to network, a
// development profile. name identifies this Source in the Gate configuration;
// clock must be the Gate's own clock.
func NewDevelopment(name string, rpc *registry011.ReadRPC, network registry011.NetworkProfile, clock registry010.Clock, limits Limits) (*Source, error) {
	if network.Qualifying() {
		return nil, errors.New("gatesource: qualifying profiles need the registry011 replay Source")
	}
	if rpc == nil {
		return nil, errors.New("gatesource: no RPC")
	}
	return newWithReader(name, network.RegistryID(), network.Name(), rpcReader{rpc: rpc, registry: network.RegistryAddress()}, clock, limits)
}

// newWithReader is NewDevelopment with an explicit reader, for tests. It is
// unexported so that no caller can bypass the refusal of qualifying profiles.
func newWithReader(name, registryID, network string, reader Reader, clock registry010.Clock, limits Limits) (*Source, error) {
	if name == "" || registryID == "" || network == "" || reader == nil || clock == nil || limits.MaxFinalizedAge <= 0 {
		return nil, errors.New("gatesource: incomplete configuration")
	}
	return &Source{name: name, registryID: registryID, network: network, reader: reader, clock: clock, limits: limits}, nil
}

// Config is the Gate configuration that matches this Source.
func (s *Source) Config() registry010.Config {
	return registry010.Config{Source: s.name, Registry: s.registryID, Network: s.network, Blockchain: true, Profile: Profile}
}

// Read implements registry010.Source.
func (s *Source) Read(ctx context.Context, did string) (registry010.Snapshot, error) {
	agentID, ok := strings.CutPrefix(did, "did:sage:"+s.registryID+":")
	if !ok || agentID == "" || strings.ContainsAny(agentID, ":#/") {
		return registry010.Snapshot{}, fmt.Errorf("gatesource: %q is not an agent of this Registry", did)
	}
	// Sampled before the read: the Gate's 5-second window then covers the
	// whole read, so a slow or stalled provider cannot make old data fresh.
	acquired, err := s.clock.Now()
	if err != nil {
		return registry010.Snapshot{}, fmt.Errorf("gatesource: clock: %w", err)
	}
	read, err := s.reader.Read(ctx, agentID)
	if err != nil {
		return registry010.Snapshot{}, fmt.Errorf("gatesource: read: %w", err)
	}
	if !read.Present {
		return registry010.Snapshot{}, errors.New("gatesource: no record")
	}
	completed, err := s.clock.Now()
	if err != nil {
		return registry010.Snapshot{}, fmt.Errorf("gatesource: clock: %w", err)
	}
	if err := s.checkHeader(read.Header, completed.Unix); err != nil {
		return registry010.Snapshot{}, err
	}
	if err := registry011.CheckRecord(s.registryID, agentID, read.Record); err != nil {
		return registry010.Snapshot{}, fmt.Errorf("gatesource: %w", err)
	}
	snapshot, err := project(read.Record, did)
	if err != nil {
		return registry010.Snapshot{}, err
	}
	hash := strings.TrimPrefix(read.Header.Hash, "0x")
	snapshot.Source, snapshot.Registry, snapshot.Network = s.name, s.registryID, s.network
	// Keys and record come from the same call at the same block, so the
	// Gate's "keys from the record's block" condition holds by construction.
	snapshot.BlockHash, snapshot.KeysBlockHash = hash, hash
	snapshot.AcquiredMS = acquired.MonoMS
	snapshot.Ready, snapshot.Finalized, snapshot.Validated = true, true, true
	// This Source validated a 0.11.0 record with 0.11.0 proofs (CheckRecord).
	snapshot.Profile = Profile
	return snapshot, nil
}

// checkHeader refuses malformed, stale or rolled-back finalized headers.
func (s *Source) checkHeader(h registry011.FinalizedHeader, now int64) error {
	if !blockHashPattern.MatchString(h.Hash) || h.Hash == "0x"+strings.Repeat("0", 64) {
		return errors.New("gatesource: malformed block hash")
	}
	// Compare in uint64: the header timestamp is untrusted input of that width.
	if now < 0 || h.Timestamp > uint64(now) || uint64(now)-h.Timestamp > uint64(s.limits.MaxFinalizedAge) {
		return fmt.Errorf("gatesource: finalized block %d at %d is outside %ds of now %d", h.Number, h.Timestamp, s.limits.MaxFinalizedAge, now)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.last != nil && (h.Number < s.last.Number || h.Number == s.last.Number && h.Hash != s.last.Hash) {
		return fmt.Errorf("gatesource: finalized block went back from %d to %d", s.last.Number, h.Number)
	}
	last := h
	s.last = &last
	return nil
}

type record struct {
	ID      string `json:"id"`
	Keys    []key  `json:"keys"`
	State   string `json:"state"`
	Version string `json:"version"`
}

type key struct {
	Name    string `json:"name"`
	Alg     string `json:"alg"`
	Key     string `json:"key"`
	State   string `json:"state"`
	Expires *int64 `json:"expires,omitempty"`
}

// project converts a record that already passed registry011.CheckRecord.
func project(raw []byte, did string) (registry010.Snapshot, error) {
	var rec record
	if err := json.Unmarshal(raw, &rec); err != nil || rec.ID != did {
		return registry010.Snapshot{}, errors.New("gatesource: record does not name the requested DID")
	}
	keys := make([]registry010.Key, 0, len(rec.Keys))
	for _, k := range rec.Keys {
		material, err := base64.RawURLEncoding.Strict().DecodeString(k.Key)
		if err != nil {
			return registry010.Snapshot{}, fmt.Errorf("gatesource: key %s encoding", k.Name)
		}
		keys = append(keys, registry010.Key{Name: k.Name, Alg: k.Alg, Material: hex.EncodeToString(material), State: k.State, Expires: k.Expires})
	}
	digest := sha256.Sum256(raw)
	return registry010.Snapshot{DID: rec.ID, Version: rec.Version, State: rec.State, Digest: hex.EncodeToString(digest[:]), Keys: keys}, nil
}

type rpcReader struct {
	rpc      *registry011.ReadRPC
	registry string
}

func (r rpcReader) Read(ctx context.Context, agentID string) (Read, error) {
	reads, err := r.rpc.PinAgentReads(ctx, r.registry, agentID)
	if err != nil {
		return Read{}, err
	}
	header, err := reads.Header()
	if err != nil {
		return Read{}, err
	}
	if err := reads.CheckCode(ctx); err != nil {
		return Read{}, fmt.Errorf("deployed code: %w", err)
	}
	present, raw, err := reads.RecordBytes(ctx)
	if err != nil {
		return Read{}, err
	}
	return Read{Header: header, Present: present, Record: raw}, nil
}
