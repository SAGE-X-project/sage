// Package gatesource feeds SAGE 0.11.0 chain Registry records into a
// registry010.Gate, so that the HPKE completion and session code can select
// and recheck keys of 0.11.0 agents.
//
// Each Read pins the provider's finalized block again and reads the record at
// that block; nothing is cached. What Validated means here is limited and
// stated exactly:
//
//   - the Registry contract checked key proofs, schema and lifecycle when the
//     record was written;
//   - this package checks, on every read, that the bytes are canonical JCS,
//     that the record and its keys have exactly the expected members, and that
//     the record names the requested DID.
//
// The record history is not replayed. That is enough for the development
// profile; a qualifying deployment needs the full history replay of the
// registry011 provider before it can call a record validated.
package gatesource

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/sage-x-project/sage/pkg/agent/internal/rfc8785"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
	"github.com/sage-x-project/sage/pkg/agent/registry011"
)

// ErrRecord means the record bytes are not a well-formed 0.11.0 record of the
// requested DID. The Gate turns every Source error into record.unreachable.
var ErrRecord = errors.New("gatesource: malformed record")

// Read is one pinned read: the finalized header and the record at that block.
type Read struct {
	Header  registry011.FinalizedHeader
	Present bool
	Record  []byte
}

// Reader performs one new pinned read per call.
type Reader interface {
	Read(ctx context.Context, agentID string) (Read, error)
}

// Source implements registry010.Source for one 0.11.0 chain Registry.
type Source struct {
	name       string
	registryID string
	network    string
	reader     Reader
	clock      registry010.Clock
}

var _ registry010.Source = (*Source)(nil)

// New reads through rpc, which must belong to network. name identifies this
// Source in the Gate configuration; clock must be the Gate's own clock.
func New(name string, rpc *registry011.ReadRPC, network registry011.NetworkProfile, clock registry010.Clock) (*Source, error) {
	if rpc == nil {
		return nil, errors.New("gatesource: no RPC")
	}
	return NewWithReader(name, network.RegistryID(), network.Name(), rpcReader{rpc: rpc, registry: network.RegistryAddress()}, clock)
}

// NewWithReader is New with an explicit reader, for tests and other transports.
func NewWithReader(name, registryID, network string, reader Reader, clock registry010.Clock) (*Source, error) {
	if name == "" || registryID == "" || network == "" || reader == nil || clock == nil {
		return nil, errors.New("gatesource: incomplete configuration")
	}
	return &Source{name: name, registryID: registryID, network: network, reader: reader, clock: clock}, nil
}

// Config is the Gate configuration that matches this Source.
func (s *Source) Config() registry010.Config {
	return registry010.Config{Source: s.name, Registry: s.registryID, Network: s.network, Blockchain: true}
}

// Read implements registry010.Source.
func (s *Source) Read(ctx context.Context, did string) (registry010.Snapshot, error) {
	agentID, ok := strings.CutPrefix(did, "did:sage:"+s.registryID+":")
	if !ok || agentID == "" || strings.ContainsAny(agentID, ":#/") {
		return registry010.Snapshot{}, fmt.Errorf("gatesource: %q is not an agent of this Registry", did)
	}
	read, err := s.reader.Read(ctx, agentID)
	if err != nil {
		return registry010.Snapshot{}, fmt.Errorf("gatesource: read: %w", err)
	}
	if !read.Present {
		return registry010.Snapshot{}, errors.New("gatesource: no record")
	}
	// The acquisition time is taken after the read completes, on the Gate's
	// clock, so the Gate's freshness window covers this read and nothing older.
	acquired, err := s.clock.Now()
	if err != nil {
		return registry010.Snapshot{}, fmt.Errorf("gatesource: clock: %w", err)
	}
	snapshot, err := project(read.Record, did)
	if err != nil {
		return registry010.Snapshot{}, err
	}
	snapshot.Source, snapshot.Registry, snapshot.Network = s.name, s.registryID, s.network
	snapshot.BlockHash, snapshot.KeysBlockHash = blockHash(read.Header.Hash), blockHash(read.Header.Hash)
	snapshot.AcquiredMS = acquired.MonoMS
	snapshot.Ready, snapshot.Finalized = true, true
	return snapshot, nil
}

type record struct {
	ID         string            `json:"id"`
	Controller string            `json:"controller"`
	Keys       []key             `json:"keys"`
	Services   []json.RawMessage `json:"services"`
	State      string            `json:"state"`
	Version    string            `json:"version"`
}

type key struct {
	Name    string          `json:"name"`
	Alg     string          `json:"alg"`
	Key     string          `json:"key"`
	State   string          `json:"state"`
	Expires *int64          `json:"expires,omitempty"`
	Proof   json.RawMessage `json:"proof"`
}

// project checks the record bytes and converts them to a Gate snapshot. The
// Gate itself checks names, order, algorithms, states, versions and material.
func project(raw []byte, did string) (registry010.Snapshot, error) {
	canonical, err := rfc8785.Canonicalize(raw)
	if err != nil || !bytes.Equal(canonical, raw) {
		return registry010.Snapshot{}, fmt.Errorf("%w: not canonical JCS", ErrRecord)
	}
	var rec record
	if err := strictDecode(raw, &rec, "id", "controller", "keys", "services", "state", "version"); err != nil {
		return registry010.Snapshot{}, err
	}
	if rec.ID != did {
		return registry010.Snapshot{}, fmt.Errorf("%w: record is for another DID", ErrRecord)
	}
	var rawKeys []json.RawMessage
	_ = json.Unmarshal(memberOf(raw, "keys"), &rawKeys)
	keys := make([]registry010.Key, 0, len(rec.Keys))
	for i, k := range rec.Keys {
		members := []string{"name", "alg", "key", "state", "proof"}
		if k.Expires != nil {
			members = append(members, "expires")
		}
		if err := strictDecode(rawKeys[i], new(key), members...); err != nil {
			return registry010.Snapshot{}, err
		}
		material, err := base64.RawURLEncoding.Strict().DecodeString(k.Key)
		if err != nil {
			return registry010.Snapshot{}, fmt.Errorf("%w: key %s encoding", ErrRecord, k.Name)
		}
		keys = append(keys, registry010.Key{Name: k.Name, Alg: k.Alg, Material: hex.EncodeToString(material), State: k.State, Expires: k.Expires})
	}
	digest := sha256.Sum256(raw)
	return registry010.Snapshot{DID: rec.ID, Version: rec.Version, State: rec.State, Digest: hex.EncodeToString(digest[:]),
		Validated: true, Keys: keys}, nil
}

// strictDecode decodes into v and requires exactly the listed members.
func strictDecode(raw []byte, v any, members ...string) error {
	var shape map[string]json.RawMessage
	if err := json.Unmarshal(raw, &shape); err != nil || len(shape) != len(members) {
		return fmt.Errorf("%w: unexpected members", ErrRecord)
	}
	for _, m := range members {
		if _, ok := shape[m]; !ok {
			return fmt.Errorf("%w: missing %s", ErrRecord, m)
		}
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return fmt.Errorf("%w: %v", ErrRecord, err)
	}
	return nil
}

func memberOf(raw []byte, name string) json.RawMessage {
	var shape map[string]json.RawMessage
	_ = json.Unmarshal(raw, &shape)
	return shape[name]
}

// blockHash converts 0x-prefixed lowercase hex to the Gate's bare form.
func blockHash(h string) string { return strings.TrimPrefix(h, "0x") }

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
	present, raw, err := reads.RecordBytes(ctx)
	if err != nil {
		return Read{}, err
	}
	return Read{Header: header, Present: present, Record: raw}, nil
}
